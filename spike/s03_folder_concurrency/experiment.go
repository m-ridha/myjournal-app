package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

const folderType = "application/vnd.google-apps.folder"
const workers = 5

type folderAPI interface {
	list(context.Context, string, string) ([]*drive.File, error)
	create(context.Context, string, string) (*drive.File, error)
}

type driveFolders struct {
	ds    *drive.Service
	runID string
}

func queryValue(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "'", "\\'")
}

func (d driveFolders) list(ctx context.Context, role, parent string) ([]*drive.File, error) {
	q := "mimeType = '" + folderType + "' and trashed = false and appProperties has { key='s3Run' and value='" + queryValue(d.runID) + "' } and appProperties has { key='role' and value='" + queryValue(role) + "' }"
	if parent != "" {
		q += " and '" + queryValue(parent) + "' in parents"
	}
	var folders []*drive.File
	page := ""
	for {
		result, err := d.ds.Files.List().Q(q).Spaces("drive").PageSize(100).PageToken(page).Fields("nextPageToken,files(id,createdTime)").Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		folders = append(folders, result.Files...)
		page = result.NextPageToken
		if page == "" {
			return folders, nil
		}
	}
}

func (d driveFolders) create(ctx context.Context, role, parent string) (*drive.File, error) {
	name := role
	if role == "root" {
		name = "myjournal-s3-" + d.runID
	}
	if role == "year" {
		name = "2026"
	}
	f := &drive.File{Name: name, MimeType: folderType, AppProperties: map[string]string{"s3Run": d.runID, "role": role}}
	if role == "year" {
		f.AppProperties["year"] = "2026"
	}
	if parent != "" {
		f.Parents = []string{parent}
	}
	return d.ds.Files.Create(f).Fields("id,createdTime").Context(ctx).Do()
}

// Stable oldest-wins selection preserves duplicates for inspection; it never
// merges, trashes, or assumes that data in a losing folder is disposable.
func oldest(folders []*drive.File) *drive.File {
	if len(folders) == 0 {
		return nil
	}
	sort.Slice(folders, func(i, j int) bool {
		if folders[i].CreatedTime == folders[j].CreatedTime {
			return folders[i].Id < folders[j].Id
		}
		return folders[i].CreatedTime < folders[j].CreatedTime
	})
	return folders[0]
}

func ensurePath(ctx context.Context, api folderAPI, lock *sync.Mutex) ([]string, error) {
	if lock != nil {
		lock.Lock()
		defer lock.Unlock()
	}
	parent := ""
	var ids []string
	for _, role := range []string{"root", "entries", "year"} {
		found, err := api.list(ctx, role, parent)
		if err != nil {
			return nil, err
		}
		folder := oldest(found)
		if folder == nil {
			folder, err = api.create(ctx, role, parent)
			if err != nil {
				return nil, err
			}
		}
		parent = folder.Id
		ids = append(ids, parent)
	}
	return ids, nil
}

func concurrentPaths(ctx context.Context, api folderAPI, lock *sync.Mutex) ([][]string, error) {
	paths := make([][]string, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range paths {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; paths[i], errs[i] = ensurePath(ctx, api, lock) }()
	}
	close(start)
	wg.Wait()
	return paths, errors.Join(errs...)
}

func samePaths(paths [][]string) bool {
	if len(paths) == 0 {
		return false
	}
	first := strings.Join(paths[0], "/")
	for _, path := range paths {
		if len(path) != 3 || strings.Join(path, "/") != first {
			return false
		}
	}
	return true
}

func runExperiment(ctx context.Context, ds *drive.Service, w io.Writer) error {
	id, err := newRunID()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "run_id: %s\nconcurrent_workers: %d\n", id, workers)
	for _, stage := range []string{"naive", "locked"} {
		api := driveFolders{ds: ds, runID: id + "-" + stage}
		var lock *sync.Mutex
		if stage == "locked" {
			lock = new(sync.Mutex)
		}
		paths, err := concurrentPaths(ctx, api, lock)
		if err != nil {
			return safeError(w, stage, err)
		}
		oneEach := true
		for _, role := range []string{"root", "entries", "year"} {
			folders, err := api.list(ctx, role, "")
			if err != nil {
				return safeError(w, stage+"_count", err)
			}
			_, _ = fmt.Fprintf(w, "%s_%s_count: %d\n", stage, role, len(folders))
			oneEach = oneEach && len(folders) == 1
		}
		_, _ = fmt.Fprintf(w, "%s_workers_same_path: %t\n", stage, samePaths(paths))
		if stage == "locked" {
			_, _ = fmt.Fprintf(w, "locked_one_folder_per_role: %t\n", oneEach)
			if !oneEach || !samePaths(paths) {
				return safeError(w, "locked_validation", errors.New("count/path mismatch"))
			}
		}
	}
	_, _ = fmt.Fprintln(w, "S3_SINGLE_PROCESS_CONCURRENCY: passed")
	return nil
}

func safeError(w io.Writer, stage string, err error) error {
	_, _ = fmt.Fprintf(w, "failed_stage: %s\n", stage)
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		_, _ = fmt.Fprintf(w, "google_http_status: %d\n", apiErr.Code)
	}
	return fmt.Errorf("%s failed", stage)
}
