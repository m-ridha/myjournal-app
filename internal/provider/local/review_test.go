package local

import (
	"context"
	"errors"
	"github/mridha/myjournal-app/internal/journal"
	"os"
	"path/filepath"
	"testing"
)

func TestCanceledIndexes(t *testing.T) {
	s := NewStore(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.ListYears(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListYears: %v", err)
	}
	if _, err := s.ListYear(ctx, 2026); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListYear: %v", err)
	}
}

func TestCanceledMutationDoesNotWaitForLock(t *testing.T) {
	unlock, err := lockMutation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := lockMutation(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("lockMutation: %v", err)
	}
}

func TestLegacyEarlyYearDirectoryRemainsUsable(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "entries", "123")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0123-01-01.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewStore(root)
	d := journal.Date{Year: 123, Month: 1, Day: 1}
	e, err := s.Get(context.Background(), d)
	if err != nil || e.Text != "old" {
		t.Fatalf("Get: %v, %v", e, err)
	}
	items, err := s.ListYear(context.Background(), 123)
	if err != nil || len(items) != 1 {
		t.Fatalf("ListYear: %v, %v", items, err)
	}
	if _, err := s.Put(context.Background(), d, "new", e.Version); err != nil {
		t.Fatal(err)
	}
	if err := s.Trash(context.Background(), d); err != nil {
		t.Fatal(err)
	}
}
