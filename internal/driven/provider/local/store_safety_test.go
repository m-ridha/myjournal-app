package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"cloud.google.com/go/civil"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

func TestConcurrentStoresRejectStaleSave(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	date := civil.Date{Year: 2026, Month: 10, Day: 9}
	initial, err := NewStore(root).Put(ctx, date, "original", "")
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 12)
	var workers sync.WaitGroup
	for range 12 {
		workers.Go(func() {
			store := NewStore(root)
			<-start
			_, err := store.Put(ctx, date, "replacement", initial.Version)
			results <- err
		})
	}
	close(start)
	workers.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, journal.ErrConflict):
			conflicts++
		default:
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 11 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestAtomicReplacementPreservesOpenReaders(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	ctx := context.Background()
	date := civil.Date{Year: 2026, Month: 10, Day: 9}
	original, err := store.Put(ctx, date, "original", "")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(filepath.Join(root, journal.EntryPath(date)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	if _, err := store.Put(ctx, date, "replacement", original.Version); err != nil {
		t.Fatal(err)
	}
	old, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "original" {
		t.Fatalf("open reader saw %q", old)
	}
	current, err := store.Get(ctx, date)
	if err != nil || current.Text != "replacement" {
		t.Fatalf("current=%+v error=%v", current, err)
	}
}

func TestAtomicReplacementFailureKeepsOriginal(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "entry.txt")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("rename failed")
	err := writeAtomically(context.Background(), path, []byte("replacement"), func(string, string) error { return failure })
	if !errors.Is(err, failure) {
		t.Fatalf("error=%v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "original" {
		t.Fatalf("data=%q error=%v", data, err)
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("staged files remain: %v error=%v", files, err)
	}
}

func TestRepeatedTrashPreservesEveryCopy(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	ctx := context.Background()
	date := civil.Date{Year: 2026, Month: 10, Day: 9}
	trashDir := filepath.Join(root, ".trash")
	if err := os.Mkdir(trashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(trashDir, date.String()+".txt"), []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"first", "second"} {
		if _, err := store.Put(ctx, date, text, ""); err != nil {
			t.Fatal(err)
		}
		if err := store.Trash(ctx, date); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Trash(ctx, date); !errors.Is(err, journal.ErrNotFound) {
		t.Fatalf("repeat trash error=%v", err)
	}
	files, err := os.ReadDir(trashDir)
	if err != nil {
		t.Fatal(err)
	}
	contents := make(map[string]bool)
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(trashDir, file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		contents[string(data)] = true
	}
	if len(files) != 3 || !contents["legacy"] || !contents["first"] || !contents["second"] {
		t.Fatalf("trash copies=%v", contents)
	}
}

func TestCanceledMutationDoesNotChangeEntry(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	date := civil.Date{Year: 2026, Month: 10, Day: 9}
	initial, err := store.Put(context.Background(), date, "original", "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Put(ctx, date, "replacement", initial.Version); !errors.Is(err, context.Canceled) {
		t.Fatalf("put error=%v", err)
	}
	if err := store.Trash(ctx, date); !errors.Is(err, context.Canceled) {
		t.Fatalf("trash error=%v", err)
	}
	current, err := store.Get(context.Background(), date)
	if err != nil || current.Text != "original" {
		t.Fatalf("current=%+v error=%v", current, err)
	}
}

func TestMalformedFileIsNotSilentlyReplacedOrSkipped(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	date := civil.Date{Year: 2026, Month: 10, Day: 9}
	saved, err := store.Put(context.Background(), date, "valid", "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, journal.EntryPath(date))
	if err := os.WriteFile(path, []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), date); !errors.Is(err, journal.ErrInvalidText) {
		t.Fatalf("get error=%v", err)
	}
	if _, err := store.Previews(context.Background(), []civil.Date{date}, 40); !errors.Is(err, journal.ErrInvalidText) {
		t.Fatalf("preview error=%v", err)
	}
	if _, err := store.Put(context.Background(), date, "replacement", saved.Version); !errors.Is(err, journal.ErrInvalidText) {
		t.Fatalf("put error=%v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) != 1 || data[0] != 0xff {
		t.Fatal("malformed file changed")
	}
}
