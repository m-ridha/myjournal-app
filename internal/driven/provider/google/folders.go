package google

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
)

func queryValue(value string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "'", "\\'") + "'"
}

func property(key, value string) string {
	return "appProperties has { key = " + queryValue(key) + " and value = " + queryValue(value) + " }"
}

func older(a, b *drive.File) bool {
	left, leftErr := time.Parse(time.RFC3339Nano, a.CreatedTime)
	right, rightErr := time.Parse(time.RFC3339Nano, b.CreatedTime)
	if leftErr == nil && rightErr == nil && !left.Equal(right) {
		return left.Before(right)
	}
	return a.Id < b.Id
}

func (s *Store) listFresh(ctx context.Context, q string) ([]*drive.File, error) {
	var result []*drive.File
	token := ""
	seen := map[string]bool{}
	for {
		page, err := retryRead(ctx, func() (*drive.FileList, error) {
			return s.service.Files.List().Q(q).Spaces("drive").PageSize(1000).PageToken(token).Fields(googleapi.Field("nextPageToken,incompleteSearch,files(" + fileFields + ")")).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapError(err)
		}
		if page.IncompleteSearch {
			return nil, journal.ErrUnavailable
		}
		result = append(result, page.Files...)
		if page.NextPageToken == "" {
			break
		}
		if seen[page.NextPageToken] {
			return nil, journal.ErrUnavailable
		}
		seen[page.NextPageToken] = true
		token = page.NextPageToken
	}
	sort.Slice(result, func(i, j int) bool { return older(result[i], result[j]) })
	return result, nil
}

func (s *Store) folders(ctx context.Context, role, root, year string) ([]*drive.File, error) {
	q := "trashed = false and mimeType = " + queryValue(folderMIME) + " and " + property(propertyApp, appTag) + " and " + property(propertyRole, role)
	if root != "" {
		q += " and " + property(propertyRoot, root)
	}
	if year != "" {
		// ListYears already fetched all tagged year folders for this root.
		// Reuse that complete operation-local result instead of another round trip.
		if role == roleYear && s.discovery != nil {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			s.discovery.mu.Lock()
			files, ok := s.discovery.queries[q]
			var matching []*drive.File
			if ok {
				for _, file := range files {
					if file.AppProperties[propertyYear] == year {
						matching = append(matching, file)
					}
				}
			}
			s.discovery.mu.Unlock()
			if ok {
				return matching, nil
			}
		}
		q += " and " + property(propertyYear, year)
	}
	return s.list(ctx, q)
}

func (s *Store) folder(ctx context.Context, role, root, year, parent, name string, create bool) (*drive.File, error) {
	folders, err := s.folders(ctx, role, root, year)
	if err != nil {
		return nil, err
	}
	if len(folders) > 0 {
		return folders[0], nil
	}
	if !create {
		return nil, journal.ErrNotFound
	}
	props := map[string]string{propertyApp: appTag, propertyRole: role}
	if root != "" {
		props[propertyRoot] = root
	}
	if year != "" {
		props[propertyYear] = year
	}
	metadata := &drive.File{Name: name, MimeType: folderMIME, AppProperties: props}
	if parent != "" {
		metadata.Parents = []string{parent}
	}
	// Never blindly retry an ambiguous create. A subsequent user retry re-queries tags.
	file, err := s.service.Files.Create(metadata).Fields(fileFields).Context(ctx).Do()
	s.clearDiscovery()
	if err != nil {
		return nil, mapError(err)
	}
	return file, nil
}

func (s *Store) root(ctx context.Context, create bool) (*drive.File, error) {
	return s.folder(ctx, roleRoot, "", "", "", "myjournal", create)
}

func (s *Store) year(ctx context.Context, year int, create bool) (*drive.File, error) {
	root, err := s.root(ctx, create)
	if err != nil {
		return nil, err
	}
	parent := ""
	if create {
		entries, err := s.folder(ctx, roleEntries, root.Id, "", root.Id, "entries", true)
		if err != nil {
			return nil, err
		}
		parent = entries.Id
	}
	return s.folder(ctx, roleYear, root.Id, strconv.Itoa(year), parent, fmt.Sprintf("%04d", year), create)
}

func (s *Store) ListYears(ctx context.Context) ([]int, error) {
	root, err := s.root(ctx, false)
	if errors.Is(err, journal.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	folders, err := s.folders(ctx, roleYear, root.Id, "")
	if err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	var years []int
	for _, f := range folders {
		year, err := strconv.Atoi(f.AppProperties[propertyYear])
		if err == nil && year >= 1 && year <= 9999 && !seen[year] {
			years = append(years, year)
			seen[year] = true
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years, nil
}

func (s *Store) files(ctx context.Context, folderID string) ([]*drive.File, error) {
	return s.list(ctx, "trashed = false and mimeType = 'text/plain' and "+queryValue(folderID)+" in parents")
}

func (s *Store) ListYear(ctx context.Context, year int) ([]journal.EntryMeta, error) {
	folder, err := s.year(ctx, year, false)
	if errors.Is(err, journal.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	files, err := s.files(ctx, folder.Id)
	if err != nil {
		return nil, err
	}
	seen := map[journal.Date]bool{}
	var result []journal.EntryMeta
	for _, f := range files {
		date, err := journal.ParseEntryDate(fmt.Sprintf("entries/%04d/%s", year, f.Name))
		if err != nil || seen[date] {
			continue
		}
		seen[date] = true
		modified, _ := time.Parse(time.RFC3339Nano, f.ModifiedTime)
		result = append(result, journal.EntryMeta{Date: date, ModifiedTime: modified})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date.After(result[j].Date) })
	return result, nil
}

func (s *Store) entryFile(ctx context.Context, date journal.Date) (*drive.File, error) {
	folder, err := s.year(ctx, date.Year, false)
	if err != nil {
		return nil, err
	}
	files, err := s.list(ctx, "trashed = false and mimeType = 'text/plain' and "+queryValue(folder.Id)+" in parents and name = "+queryValue(date.String()+".txt"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, journal.ErrNotFound
	}
	return files[0], nil
}

// readEntryFile reuses a complete year listing only within this read operation.
// Mutation lookup remains separate and never borrows this broad listing.
func (s *Store) readEntryFile(ctx context.Context, date journal.Date) (*drive.File, error) {
	if s.discovery == nil {
		return s.entryFile(ctx, date)
	}
	folder, err := s.year(ctx, date.Year, false)
	if err != nil {
		return nil, err
	}
	files, err := s.files(ctx, folder.Id)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.Name == date.String()+".txt" {
			return file, nil
		}
	}
	return nil, journal.ErrNotFound
}

func (s *Store) metadata(ctx context.Context, id string) (*drive.File, error) {
	file, err := retryRead(ctx, func() (*drive.File, error) { return s.service.Files.Get(id).Fields(fileFields).Context(ctx).Do() })
	if err != nil {
		return nil, mapError(err)
	}
	if file.Trashed {
		return nil, journal.ErrNotFound
	}
	return file, nil
}

func (p *Provider) FolderURL(ctx context.Context, c provider.Connection) (string, error) {
	store, err := p.Store(ctx, c)
	if err != nil {
		return "", err
	}
	root, err := store.(*Store).root(ctx, false)
	if errors.Is(err, journal.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return "https://drive.google.com/drive/folders/" + url.PathEscape(root.Id), nil
}
