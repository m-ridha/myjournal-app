package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/journal/storetest"

	"cloud.google.com/go/civil"
)

func TestLocalStorePutAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	date := civil.Date{Year: 2026, Month: 9, Day: 21}

	entry, err := store.Put(context.Background(), date, "hello world", "")
	if err != nil {
		t.Fatalf("Put() unexpected error: %v", err)
	}
	if entry.Text != "hello world" {
		t.Fatalf("Put() text = %q, want %q", entry.Text, "hello world")
	}

	got, err := store.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if got.Text != "hello world" {
		t.Fatalf("Get() text = %q, want %q", got.Text, "hello world")
	}

	if _, err := os.Stat(filepath.Join(dir, journal.EntryPath(date))); err != nil {
		t.Fatalf("expected file on disk: %v", err)
	}
}

func TestLocalStoreRejectsFutureDateViaService(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	service := journal.NewService(store, fixedClock{now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)})

	_, err := service.Save(context.Background(), civil.Date{Year: 2026, Month: 9, Day: 22}, "future", "")
	if err != journal.ErrFutureDate {
		t.Fatalf("Save() error = %v, want %v", err, journal.ErrFutureDate)
	}
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func TestLocalStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		return NewStore(t.TempDir())
	})
}
