package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myjournal-app/internal/core/journal"
)

func TestOperationalErrorsAreUnavailable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(root, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	date := journal.Date{Year: 2026, Month: time.October, Day: 11}
	for name, call := range map[string]func() error{
		"years": func() error { _, err := store.ListYears(context.Background()); return err },
		"year":  func() error { _, err := store.ListYear(context.Background(), 2026); return err },
		"get":   func() error { _, err := store.Get(context.Background(), date); return err },
		"put":   func() error { _, err := store.Put(context.Background(), date, "synthetic", ""); return err },
		"trash": func() error { return store.Trash(context.Background(), date) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, journal.ErrUnavailable) {
				t.Fatalf("got %v, want unavailable", err)
			}
		})
	}
}

func TestMetadataFailureDoesNotReturnPartialList(t *testing.T) {
	for _, failure := range []error{errors.New("synthetic metadata failure"), os.ErrNotExist} {
		store := NewStore(t.TempDir())
		ctx := context.Background()
		for _, day := range []int{10, 11} {
			_, err := store.Put(ctx, journal.Date{Year: 2026, Month: 10, Day: day}, "synthetic", "")
			if err != nil {
				t.Fatal(err)
			}
		}
		calls := 0
		store.entryInfo = func(entry os.DirEntry) (os.FileInfo, error) {
			calls++
			if calls == 2 {
				return nil, failure
			}
			return entry.Info()
		}
		items, err := store.ListYear(ctx, 2026)
		if !errors.Is(err, journal.ErrUnavailable) || items != nil {
			t.Fatal("metadata failure returned partial success")
		}
	}
}

func TestFilesystemErrorClassificationPreservesIdentity(t *testing.T) {
	failure := errors.New("synthetic filesystem failure")
	for _, err := range []error{
		&os.PathError{Op: "open", Path: "synthetic", Err: failure},
		&os.LinkError{Op: "rename", Old: "synthetic-old", New: "synthetic-new", Err: failure},
	} {
		got := storageError(err)
		if !errors.Is(got, journal.ErrUnavailable) || !errors.Is(got, failure) {
			t.Fatalf("filesystem error identity/classification lost: %T", err)
		}
	}
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, journal.ErrNotFound, journal.ErrConflict, journal.ErrInvalidText} {
		if got := storageError(err); got != err {
			t.Fatalf("existing error identity changed: %v", err)
		}
	}
}
