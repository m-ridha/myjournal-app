package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
)

func TestDevelopmentProviderFailsClosedAndDisconnectPreservesFiles(t *testing.T) {
	if _, err := New("https://example.invalid", t.TempDir()).Begin(context.Background(), provider.LoginRequest{}); err == nil {
		t.Fatal("network origin accepted")
	}
	p := New("http://localhost:3030", t.TempDir())
	if p.ID() != "local" || p.Name() == "" || p.RequiresLoginFlow() {
		t.Fatal("development provider contract changed")
	}
	current, err := p.Complete(context.Background(), provider.LoginCallback{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Revoke(context.Background(), current); err != nil {
		t.Fatal(err)
	}
}
func TestLocalDiscoveryRejectsNonDirectoriesAndOperationalFailures(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	if err := os.WriteFile(filepath.Join(root, "entries"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListYears(context.Background()); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal(err)
	}
	if _, err := store.ListYear(context.Background(), 2020); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal(err)
	}
	if err := store.Trash(context.Background(), journal.Date{Year: 2020, Month: 1, Day: 1}); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "entries")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "entries", "2020"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"entries/plain.txt", "entries/2020/bad.txt", "entries/2020/2020-02-30.txt"} {
		if err := os.WriteFile(filepath.Join(root, path), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if years, err := store.ListYears(context.Background()); err != nil || len(years) != 1 || years[0] != 2020 {
		t.Fatalf("years=%v err=%v", years, err)
	}
	if entries, err := store.ListYear(context.Background(), 2020); err != nil || len(entries) != 0 {
		t.Fatalf("malformed entries were listed: %v %v", entries, err)
	}
}
func TestLocalTrashFailurePreservesEntry(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	if _, err := store.Put(context.Background(), date, "synthetic", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".trash"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Trash(context.Background(), date); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal(err)
	}
	if entry, err := store.Get(context.Background(), date); err != nil || entry.Text != "synthetic" {
		t.Fatal("failed trash lost saved text")
	}
}
func TestLocalMutationWaitCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		previous := mutationLock
		mutationLock = make(chan struct{}, 1)
		defer func() { mutationLock = previous }()
		store := NewStore(t.TempDir())
		mutationLock <- struct{}{}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, err := store.Put(ctx, journal.Date{Year: 2020, Month: 1, Day: 1}, "synthetic", "")
			done <- err
		}()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		<-mutationLock
	})
}
func TestAtomicWriteMissingDirectoryAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "entry.txt")
	if err := writeAtomically(context.Background(), path, []byte("synthetic"), os.Rename); err == nil {
		t.Fatal("missing directory accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	path = filepath.Join(root, "entry.txt")
	if err := writeAtomically(ctx, path, []byte("synthetic"), os.Rename); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatal("canceled write left staging files")
	}
}

func TestLocalIndexSortAndCancellationDoNotReturnPartialResults(t *testing.T) {
	store := NewStore(t.TempDir())
	ctx := context.Background()
	for _, day := range []int{1, 2} {
		if _, err := store.Put(ctx, journal.Date{Year: 2020, Month: 1, Day: day}, "synthetic", ""); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := store.ListYear(ctx, 2020)
	if err != nil || len(entries) != 2 || entries[0].Date.Day != 2 {
		t.Fatal("index is not newest first")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	store.entryInfo = func(entry os.DirEntry) (os.FileInfo, error) { cancel(); return entry.Info() }
	if entries, err := store.ListYear(ctx, 2020); !errors.Is(err, context.Canceled) || len(entries) != 0 {
		t.Fatal("canceled index returned partial success")
	}
}
func TestTrashRejectsDirectoryAndCleansReservation(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	path := filepath.Join(root, "entries", "2020", date.String()+".txt")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.Trash(context.Background(), date); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal("directory unexpectedly trashed")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatal("failed trash lost original path")
	}
	if entries, err := os.ReadDir(filepath.Join(root, ".trash")); err != nil || len(entries) != 0 {
		t.Fatal("failed trash retained reservation")
	}
}
func TestTrashUnwritableDirectoryPreservesEntry(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses filesystem permissions")
	}
	root := t.TempDir()
	store := NewStore(root)
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	if _, err := store.Put(context.Background(), date, "synthetic", ""); err != nil {
		t.Fatal(err)
	}
	trash := filepath.Join(root, ".trash")
	if err := os.Mkdir(trash, 0500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(trash, 0700) }()
	if err := store.Trash(context.Background(), date); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal("unwritable trash directory accepted")
	}
	if entry, err := store.Get(context.Background(), date); err != nil || entry.Text != "synthetic" {
		t.Fatal("failed trash lost saved content")
	}
}

func TestTrashCancellationAtMutationLockHandoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		previous := mutationLock
		mutationLock = make(chan struct{}, 1)
		defer func() { mutationLock = previous }()
		store := NewStore(t.TempDir())
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		for range 64 {
			entry, err := store.Get(context.Background(), date)
			version := entry.Version
			if err != nil && !errors.Is(err, journal.ErrNotFound) {
				t.Fatal(err)
			}
			if _, err := store.Put(context.Background(), date, "synthetic", version); err != nil {
				t.Fatal(err)
			}
			mutationLock <- struct{}{}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- store.Trash(ctx, date) }()
			synctest.Wait()
			<-mutationLock
			cancel()
			err = <-done
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if errors.Is(err, context.Canceled) {
				if got, err := store.Get(context.Background(), date); err != nil || got.Text != "synthetic" {
					t.Fatal("canceled trash lost content")
				}
			}
		}
	})
}

// Observe the owned reservation to model cancellation after it has been created.
// This avoids sleeps and assumptions about how many times Err is checked.
type cancelAfterTrashReservation struct {
	context.Context
	cancel    context.CancelFunc
	directory string
}

func (c cancelAfterTrashReservation) Err() error {
	if entries, err := os.ReadDir(c.directory); err == nil && len(entries) > 0 {
		c.cancel()
	}
	return c.Context.Err()
}
func TestTrashCancellationCleansReservationAndPreservesText(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	if _, err := store.Put(context.Background(), date, "synthetic", ""); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	trash := filepath.Join(root, ".trash")
	observed := cancelAfterTrashReservation{Context: ctx, cancel: cancel, directory: trash}
	if err := store.Trash(observed, date); !errors.Is(err, context.Canceled) {
		t.Fatal("trash ignored cancellation after reservation")
	}
	if entry, err := store.Get(context.Background(), date); err != nil || entry.Text != "synthetic" {
		t.Fatal("canceled trash lost entry")
	}
	if entries, err := os.ReadDir(trash); err != nil || len(entries) != 0 {
		t.Fatal("canceled trash left reserved file")
	}
}
