package main

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"google.golang.org/api/drive/v3"
)

type fakeFolders struct {
	mu        sync.Mutex
	folders   map[string][]*drive.File
	next      int
	rootReads int
	barrier   chan struct{}
}

func (f *fakeFolders) list(ctx context.Context, role, parent string) ([]*drive.File, error) {
	f.mu.Lock()
	found := append([]*drive.File(nil), f.folders[role+"/"+parent]...)
	barrier := f.barrier
	if role == "root" && barrier != nil {
		f.rootReads++
		if f.rootReads == workers {
			close(barrier)
		}
	}
	f.mu.Unlock()
	if role == "root" && barrier != nil {
		select {
		case <-barrier:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return found, nil
}

func (f *fakeFolders) create(_ context.Context, role, parent string) (*drive.File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	folder := &drive.File{Id: fmt.Sprint(f.next), CreatedTime: "2026-10-08T00:00:00Z"}
	f.folders[role+"/"+parent] = append(f.folders[role+"/"+parent], folder)
	return folder, nil
}

func TestConcurrentNaiveAndLockedPaths(t *testing.T) {
	naive := &fakeFolders{folders: make(map[string][]*drive.File), barrier: make(chan struct{})}
	paths, err := concurrentPaths(context.Background(), naive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(naive.folders["root/"]) != workers || samePaths(paths) {
		t.Fatal("naive race must produce distinct roots")
	}
	locked := &fakeFolders{folders: make(map[string][]*drive.File)}
	paths, err = concurrentPaths(context.Background(), locked, new(sync.Mutex))
	if err != nil {
		t.Fatal(err)
	}
	if locked.next != 3 || !samePaths(paths) {
		t.Fatal("lock must serialize creation of one shared path")
	}
	// A fresh store view with no lock still reuses existing folders.
	if _, err := ensurePath(context.Background(), locked, nil); err != nil || locked.next != 3 {
		t.Fatal("existing path must be reused")
	}
}

func TestOldestSelectionPreservesDuplicates(t *testing.T) {
	items := []*drive.File{{Id: "new", CreatedTime: "2026-10-09T00:00:00Z"}, {Id: "b", CreatedTime: "2026-10-08T00:00:00Z"}, {Id: "a", CreatedTime: "2026-10-08T00:00:00Z"}}
	if oldest(items).Id != "a" || len(items) != 3 {
		t.Fatal("must deterministically select oldest without deleting duplicates")
	}
	if oldest(nil) != nil || samePaths([][]string{{"incomplete"}}) {
		t.Fatal("empty/partial paths must not pass")
	}
}
