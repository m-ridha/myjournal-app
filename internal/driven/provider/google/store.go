package google

import (
	"context"
	"crypto/md5" // Drive's content checksum is an opaque change token, not a security primitive.
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
)

const folderMIME = "application/vnd.google-apps.folder"
const appTag = "myjournal-v1"
const fileFields = "id,name,mimeType,parents,appProperties,createdTime,modifiedTime,md5Checksum,trashed"
const (
	maxContentBytes  = 1 << 20
	previewReadBytes = 64 << 10
	previewWorkers   = 4
	readAttempts     = 3
)

const (
	propertyApp  = "app"
	propertyRole = "role"
	propertyRoot = "root"
	propertyYear = "year"
	roleRoot     = "root"
	roleEntries  = "entries"
	roleYear     = "year"
	roleEntry    = "entry"
	textMIME     = "text/plain"
)

// Store uses private app properties for folders; names are presentation only.
// Mutation locks coordinate this process, not external Drive clients or replicas.
type Store struct {
	service   *drive.Service
	lock      func(context.Context) (func(), error)
	discovery *operationDiscovery
}

func (p *Provider) Store(ctx context.Context, c provider.Connection) (journal.Store, error) {
	stored, err := decodeCredentials(c)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	client := oauth2.NewClient(ctx, tokenSource{p: p, ctx: ctx, refresh: stored.Refresh})
	client.Timeout = requestTimeout
	client.Transport = timingTransport{next: client.Transport}
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, journal.ErrUnavailable
	}
	// ponytail: fixed lock stripes bound memory; unrelated accounts can briefly serialize.
	hash := sha256.Sum256([]byte(c.AccountID))
	return &Store{service: service, lock: func(ctx context.Context) (func(), error) {
		return acquire(ctx, p.mutations[int(hash[0])%len(p.mutations)])
	}}, nil
}

func acquire(ctx context.Context, lock chan struct{}) (func(), error) {
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

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

func checksum(data []byte) string { sum := md5.Sum(data); return hex.EncodeToString(sum[:]) }

func (s *Store) Get(ctx context.Context, date journal.Date) (journal.Entry, error) {
	file, err := s.entryFile(ctx, date)
	if err != nil {
		return journal.Entry{}, err
	}
	if file.Md5Checksum == "" {
		return journal.Entry{}, journal.ErrUnavailable
	}
	response, err := retryRead(ctx, func() (*http.Response, error) { return s.service.Files.Get(file.Id).Context(ctx).Download() })
	if err != nil {
		return journal.Entry{}, mapError(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxContentBytes+1))
	if err != nil {
		return journal.Entry{}, mapError(err)
	}
	if len(body) > maxContentBytes {
		return journal.Entry{}, journal.ErrUnavailable
	}
	if !utf8.Valid(body) {
		return journal.Entry{}, journal.ErrInvalidText
	}
	if checksum(body) != file.Md5Checksum {
		return journal.Entry{}, journal.ErrUnavailable
	}
	return journal.Entry{Date: date, Text: string(body), Version: journal.Version(file.Md5Checksum), Exists: true}, nil
}

func (s *Store) Put(ctx context.Context, date journal.Date, text string, base journal.Version) (journal.Entry, error) {
	defer s.clearDiscovery()
	if !date.IsValid() || date.Year < 1 || date.Year > 9999 {
		return journal.Entry{}, journal.ErrInvalidDate
	}
	if !utf8.ValidString(text) {
		return journal.Entry{}, journal.ErrInvalidText
	}
	if len(text) > maxContentBytes {
		return journal.Entry{}, journal.ErrUnavailable
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return journal.Entry{}, err
	}
	defer unlock()
	file, err := s.entryFile(ctx, date)
	if err != nil && !errors.Is(err, journal.ErrNotFound) {
		return journal.Entry{}, err
	}
	missing := errors.Is(err, journal.ErrNotFound)
	if missing && base != "" {
		return journal.Entry{}, journal.ErrConflict
	}
	if !missing && file.Md5Checksum == "" {
		return journal.Entry{}, journal.ErrUnavailable
	}
	if !missing && (base == "" || journal.Version(file.Md5Checksum) != base) {
		return journal.Entry{}, journal.ErrConflict
	}
	if !missing {
		// Re-read just before upload. This reduces but does not eliminate the external
		// writer race: Drive checksum comparison and upload are separate operations.
		file, err = s.metadata(ctx, file.Id)
		if errors.Is(err, journal.ErrNotFound) {
			return journal.Entry{}, journal.ErrConflict
		}
		if err != nil {
			return journal.Entry{}, err
		}
		if file.Md5Checksum == "" {
			return journal.Entry{}, journal.ErrUnavailable
		}
		if journal.Version(file.Md5Checksum) != base {
			return journal.Entry{}, journal.ErrConflict
		}
		if checksum([]byte(text)) == file.Md5Checksum {
			return journal.Entry{Date: date, Text: text, Version: base, Exists: true}, nil
		}
		// Entries are capped at 1 MiB; stream one request instead of allocating the SDK
		// default 16 MiB chunk. Ambiguous uploads remain non-retryable.
		call := s.service.Files.Update(file.Id, &drive.File{}).Media(strings.NewReader(text), googleapi.ContentType("text/plain; charset=utf-8"), googleapi.ChunkSize(0)).Fields(fileFields).Context(ctx)
		if etag := file.Header.Get("ETag"); etag != "" {
			call.Header().Set("If-Match", etag)
		}
		file, err = call.Do()
	} else {
		folder, folderErr := s.year(ctx, date.Year, true)
		if folderErr != nil {
			return journal.Entry{}, folderErr
		}
		file, err = s.service.Files.Create(&drive.File{Name: date.String() + ".txt", MimeType: textMIME, Parents: []string{folder.Id}, AppProperties: map[string]string{propertyApp: appTag, propertyRole: roleEntry}}).Media(strings.NewReader(text), googleapi.ContentType("text/plain; charset=utf-8"), googleapi.ChunkSize(0)).Fields(fileFields).Context(ctx).Do()
	}
	if err != nil {
		return journal.Entry{}, mapError(err)
	}
	if file.Md5Checksum == "" || file.Md5Checksum != checksum([]byte(text)) {
		return journal.Entry{}, journal.ErrUnavailable
	}
	return journal.Entry{Date: date, Text: text, Version: journal.Version(file.Md5Checksum), Exists: true}, nil
}

func (s *Store) Trash(ctx context.Context, date journal.Date) error {
	defer s.clearDiscovery()
	unlock, err := s.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	file, err := s.entryFile(ctx, date)
	if err != nil {
		return err
	}
	_, err = s.service.Files.Update(file.Id, &drive.File{Trashed: true}).Fields("id,trashed").Context(ctx).Do()
	return mapError(err)
}

func (s *Store) Previews(ctx context.Context, dates []journal.Date, maxBytes int) (map[journal.Date]string, error) {
	out := make(map[journal.Date]string)
	if len(dates) == 0 {
		return out, nil
	}
	files, err := s.previewFiles(ctx, dates)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan journal.Date)
	var group sync.WaitGroup
	var first error
	for range previewWorkers {
		group.Go(func() {
			for date := range jobs {
				file := files[date]
				if file == nil {
					continue
				}
				text, err := s.preview(ctx, file, maxBytes)
				if errors.Is(err, journal.ErrNotFound) {
					continue
				}
				mu.Lock()
				if err != nil {
					if first == nil {
						first = err
						cancel()
					}
				} else {
					out[date] = text
				}
				mu.Unlock()
			}
		})
	}
	for _, date := range dates {
		select {
		case jobs <- date:
		case <-ctx.Done():
		}
	}
	close(jobs)
	group.Wait()
	if first != nil {
		return nil, first
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Resolve folders once per batch; keep discovery request-scoped to respect external moves.
func (s *Store) previewFiles(ctx context.Context, dates []journal.Date) (map[journal.Date]*drive.File, error) {
	out := make(map[journal.Date]*drive.File, len(dates))
	root, err := s.root(ctx, false)
	if errors.Is(err, journal.ErrNotFound) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	years := make(map[int]bool)
	wanted := make(map[journal.Date]bool, len(dates))
	for _, d := range dates {
		years[d.Year] = true
		wanted[d] = true
	}
	for year := range years {
		folder, err := s.folder(ctx, roleYear, root.Id, strconv.Itoa(year), "", "", false)
		if errors.Is(err, journal.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		files, err := s.files(ctx, folder.Id)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			date, err := journal.ParseEntryDate(fmt.Sprintf("entries/%04d/%s", year, file.Name))
			if err == nil && wanted[date] && out[date] == nil {
				out[date] = file
			}
		}
	}
	return out, nil
}

func (s *Store) preview(ctx context.Context, file *drive.File, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		return "", nil
	}
	response, err := retryRead(ctx, func() (*http.Response, error) { return s.service.Files.Get(file.Id).Context(ctx).Download() })
	if err != nil {
		return "", mapError(err)
	}
	defer func() { _ = response.Body.Close() }()
	// Leading whitespace may be arbitrarily long; inspect a bounded 64 KiB prefix.
	// Never download the full entry merely to render a page preview.
	budget := previewReadBytes
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(budget)))
	if err != nil {
		return "", mapError(err)
	}
	if !utf8.Valid(data) {
		if len(data) != budget {
			return "", journal.ErrInvalidText
		}
		valid := false
		for trim := 1; trim <= 3 && trim <= len(data); trim++ {
			if utf8.Valid(data[:len(data)-trim]) && !utf8.FullRune(data[len(data)-trim:]) {
				data = data[:len(data)-trim]
				valid = true
				break
			}
		}
		if !valid {
			return "", journal.ErrInvalidText
		}
	}
	return journal.PreviewPrefix(string(data), maxBytes), nil
}

func retryRead[T any](ctx context.Context, call func() (T, error)) (T, error) {
	var zero T
	for attempt := 0; attempt < readAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		result, err := call()
		if err == nil {
			return result, nil
		}
		var apiErr *googleapi.Error
		retry := errors.As(err, &apiErr) && (apiErr.Code == http.StatusTooManyRequests || apiErr.Code >= http.StatusInternalServerError || (apiErr.Code == http.StatusForbidden && rateLimited(apiErr)))
		if !retry || attempt == readAttempts-1 {
			return zero, err
		}
		delay := time.Duration(100*(1<<attempt)+rand.IntN(100)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, journal.ErrUnavailable
}

func rateLimited(err *googleapi.Error) bool {
	for _, item := range err.Errors {
		if item.Reason == "rateLimitExceeded" || item.Reason == "userRateLimitExceeded" {
			return true
		}
	}
	return false
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, journal.ErrAuthRequired) || errors.Is(err, journal.ErrUnavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var e *googleapi.Error
	if errors.As(err, &e) {
		switch e.Code {
		case http.StatusUnauthorized:
			return journal.ErrAuthRequired
		case http.StatusNotFound:
			return journal.ErrNotFound
		case http.StatusConflict, http.StatusPreconditionFailed:
			return journal.ErrConflict
		case http.StatusForbidden:
			for _, item := range e.Errors {
				if item.Reason == "insufficientPermissions" {
					return journal.ErrAuthRequired
				}
			}
		}
	}
	return journal.ErrUnavailable
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

var _ journal.Store = (*Store)(nil)
