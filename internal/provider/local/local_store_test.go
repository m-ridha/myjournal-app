package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/testutil"

	"cloud.google.com/go/civil"
)

func TestStoreSaveAndGet(t *testing.T) {
	root := t.TempDir()
	store, err := New(testutil.LocalBaseURL, root).Store(context.Background(), providerConnection())
	if err != nil {
		t.Fatalf("Store() unexpected error: %v", err)
	}

	date := civil.Date{Year: 2026, Month: 9, Day: 21}
	entry, err := store.Put(context.Background(), date, "hello world", "")
	if err != nil {
		t.Fatalf("Put() unexpected error: %v", err)
	}
	if got, want := entry.Text, "hello world"; got != want {
		t.Fatalf("Put() text = %q, want %q", got, want)
	}

	got, err := store.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if got.Text != "hello world" {
		t.Fatalf("Get() text = %q, want %q", got.Text, "hello world")
	}

	path := filepath.Join(root, "entries", "2026", "2026-09-21.txt")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() unexpected error: %v", err)
	}
	if got, want := string(content), "hello world"; got != want {
		t.Fatalf("stored file content = %q, want %q", got, want)
	}
}

func TestStoreListYearsAndYear(t *testing.T) {
	root := t.TempDir()
	store, err := New(testutil.LocalBaseURL, root).Store(context.Background(), providerConnection())
	if err != nil {
		t.Fatalf("Store() unexpected error: %v", err)
	}

	date := civil.Date{Year: 2026, Month: 9, Day: 21}
	if _, err := store.Put(context.Background(), date, "hello", ""); err != nil {
		t.Fatalf("Put() unexpected error: %v", err)
	}

	years, err := store.ListYears(context.Background())
	if err != nil {
		t.Fatalf("ListYears() unexpected error: %v", err)
	}
	if len(years) != 1 || years[0] != 2026 {
		t.Fatalf("ListYears() = %v, want [2026]", years)
	}

	items, err := store.ListYear(context.Background(), 2026)
	if err != nil {
		t.Fatalf("ListYear() unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].Date != date {
		t.Fatalf("ListYear() = %#v, want single entry for 2026-09-21", items)
	}

	if time.Since(items[0].ModifiedTime) > time.Minute {
		t.Fatalf("ListYear() ModifiedTime looks unset or stale: %v", items[0].ModifiedTime)
	}
}

func providerConnection() provider.Connection {
	return provider.Connection{
		ProviderID:   "local",
		AccountID:    "dev",
		AccountEmail: "dev@localhost",
		Credentials:  "local-dev-user",
	}
}
