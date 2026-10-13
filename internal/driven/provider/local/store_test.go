package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/civil"

	"github/m-ridha/myjournal-app/internal/config"
	"github/m-ridha/myjournal-app/internal/core/journal"
	"github/m-ridha/myjournal-app/internal/core/journal/storetest"
	"github/m-ridha/myjournal-app/internal/testutil"
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
	service := journal.NewService(store, testutil.StubClock{Time: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)})

	_, err := service.Save(context.Background(), civil.Date{Year: 2026, Month: 9, Day: 22}, "future", "")
	if err != journal.ErrFutureDate {
		t.Fatalf("Save() error = %v, want %v", err, journal.ErrFutureDate)
	}
}

func TestLocalStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		return NewStore(t.TempDir())
	})
}

func TestValidateLoopbackURL(t *testing.T) {
	for _, value := range []string{"http://localhost:3030", "https://localhost", "http://127.0.0.1:3030", "http://[::1]:3030"} {
		t.Run(value, func(t *testing.T) {
			if err := config.ValidateLoopbackURL(value); err != nil {
				t.Fatalf("config.ValidateLoopbackURL(%q) error = %v", value, err)
			}
		})
	}
	for _, value := range []string{"http://localhost.evil.com", "https://example.com", "localhost:3030", "http://localhost/path"} {
		t.Run(value, func(t *testing.T) {
			if err := config.ValidateLoopbackURL(value); err == nil {
				t.Fatalf("config.ValidateLoopbackURL(%q) error = nil, want rejection", value)
			}
		})
	}
}
