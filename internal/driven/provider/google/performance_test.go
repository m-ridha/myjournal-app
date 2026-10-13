package google

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/api/drive/v3"

	"github/m-ridha/myjournal-app/internal/core/journal"
	cachestore "github/m-ridha/myjournal-app/internal/driven/cache/cachestore"
)

func TestSmallUploadDoesNotAllocateDefaultChunk(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	entry, err := s.Put(ctx, date, "old", "")
	if err != nil {
		t.Fatal(err)
	}
	// A wide ceiling catches the 16 MiB SDK buffer, not small allocator variation.
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	text := strings.Repeat("synthetic 🙂\n", 100)
	saved, err := s.Put(ctx, date, text, entry.Version)
	runtime.ReadMemStats(&after)
	if err != nil || saved.Text != text {
		t.Fatalf("save: %v", err)
	}
	if used := after.TotalAlloc - before.TotalAlloc; used > 8<<20 {
		t.Fatalf("small upload allocated %d bytes; default SDK chunk is unnecessary", used)
	}
	loaded, err := s.Get(ctx, date)
	if err != nil || loaded.Text != text || loaded.Version != saved.Version {
		t.Fatalf("roundtrip: %v", err)
	}
}

func TestPageReusesDiscoveryOnlyWithinOperation(t *testing.T) {
	s, f := testStore(t)
	f.pageSize = 1000
	ctx := context.Background()
	for day := 1; day <= 20; day++ {
		if _, err := s.Put(ctx, journal.Date{Year: 2020, Month: 1, Day: day}, "synthetic", ""); err != nil {
			t.Fatal(err)
		}
	}
	service := journal.NewService(cachestore.New(nil, 0).Wrap("test", s), benchmarkClock{})
	before := f.lists
	page, err := service.ListEntries(ctx, 1, 20)
	if err != nil || len(page.Items) != 20 {
		t.Fatalf("page: %v", err)
	}
	if got := f.lists - before; got != 3 {
		t.Fatalf("page discovery lists=%d want 3", got)
	}
	// A second operation must rediscover the externally changed folder contents.
	f.mu.Lock()
	for _, file := range f.files {
		if file.Name == "2020-01-20.txt" {
			file.Trashed = true
		}
	}
	f.mu.Unlock()
	before = f.lists
	page, err = service.ListEntries(ctx, 1, 20)
	if err != nil || len(page.Items) != 19 || f.lists-before != 3 {
		t.Fatalf("scope leaked across operations: count=%d lists=%d err=%v", len(page.Items), f.lists-before, err)
	}
}

func TestDiscoveryBoundsAndMutationReset(t *testing.T) {
	s, _ := testStore(t)
	scoped := s.ForOperation().(*Store)
	ctx := context.Background()
	for i := 0; i < maxDiscoveryQueries+5; i++ {
		if _, err := scoped.list(ctx, fmt.Sprintf("query-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if len(scoped.discovery.queries) != maxDiscoveryQueries {
		t.Fatal("unbounded query reuse")
	}
	if _, err := scoped.Put(ctx, journal.Date{Year: 2020, Month: 1, Day: 1}, "created", ""); err != nil {
		t.Fatal(err)
	}
	if len(scoped.discovery.queries) != 0 {
		t.Fatal("mutation left stale discovery")
	}
}

func TestDiscoveryDoesNotRetainOversizedLists(t *testing.T) {
	s, f := testStore(t)
	f.pageSize = 1000
	for i := 0; i < maxDiscoveryFiles+1; i++ {
		id := fmt.Sprintf("file-%d", i)
		f.files[id] = &drive.File{Id: id}
	}
	scoped := s.ForOperation().(*Store)
	files, err := scoped.list(context.Background(), "all")
	if err != nil || len(files) != maxDiscoveryFiles+1 {
		t.Fatalf("list: %d %v", len(files), err)
	}
	if len(scoped.discovery.queries) != 0 {
		t.Fatal("oversized list retained")
	}
}

func TestStreamingUploadAtContentLimit(t *testing.T) {
	s, f := testStore(t)
	ctx := context.Background()
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	text := strings.Repeat("a", maxContentBytes)
	created, err := s.Put(ctx, d, text, "")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.Put(ctx, d, strings.Repeat("b", maxContentBytes), created.Version)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Get(ctx, d)
	if err != nil || loaded.Text != updated.Text || loaded.Version != updated.Version {
		t.Fatalf("limit roundtrip: %v", err)
	}
	before := f.writes
	if _, err := s.Put(ctx, d, text+"x", updated.Version); err != journal.ErrUnavailable {
		t.Fatal("oversized upload accepted")
	}
	if f.writes != before {
		t.Fatal("oversized text reached Drive")
	}
}

func TestColdAndPostSaveEntriesRequestCounts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, _, counts := collapseProvider(t, func(_ http.ResponseWriter, r *http.Request) bool {
			if r.Method == http.MethodGet {
				time.Sleep(time.Millisecond)
			}
			return false
		})
		store := collapseStore(t, p, context.Background())
		var saved journal.Entry
		for day := 1; day <= 7; day++ {
			entry, err := store.Put(context.Background(), journal.Date{Year: 2021, Month: 1, Day: day}, "synthetic", "")
			if err != nil {
				t.Fatal(err)
			}
			if day == 7 {
				saved = entry
			}
		}
		if _, err := store.Put(context.Background(), journal.Date{Year: 2020, Month: 1, Day: 1}, "synthetic", ""); err != nil {
			t.Fatal(err)
		}
		service := journal.NewService(cachestore.New(newLRUStorage(t, time.Minute), time.Minute).Wrap("synthetic", store), benchmarkClock{})
		check := func(label string, wantLists, wantDownloads int) {
			counts.counts = make(map[string]int)
			start := time.Now()
			page, err := service.ListEntries(context.Background(), 1, 20)
			if err != nil || len(page.Items) != 8 {
				t.Fatalf("%s page: count=%d err=%v", label, len(page.Items), err)
			}
			lists, downloads := counts.counts["list"], counts.counts["download"]
			t.Logf("%s eight-entry/two-year page: lists=%d downloads=%d simulated elapsed=%v", label, lists, downloads, time.Since(start))
			if lists != wantLists || downloads != wantDownloads {
				t.Fatalf("%s lists/downloads=%d/%d, want %d/%d", label, lists, downloads, wantLists, wantDownloads)
			}
		}
		check("cold", 4, 8)
		check("warm", 0, 0)
		if _, err := service.Save(context.Background(), saved.Date, "changed synthetic", saved.Version); err != nil {
			t.Fatal(err)
		}
		check("after-update", 3, 1)
	})
}
