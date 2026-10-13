package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/api/drive/v3"

	"github/m-ridha/myjournal-app/internal/core/journal"
	"github/m-ridha/myjournal-app/internal/core/journal/storetest"
	cachestore "github/m-ridha/myjournal-app/internal/driven/cache/cachestore"
	"github/m-ridha/myjournal-app/internal/driven/cache/local/lru"
)

func TestOperationGetsReuseYearListing(t *testing.T) {
	s, f := testStore(t)
	f.pageSize = 1000
	ctx := context.Background()
	for day := 1; day <= 3; day++ {
		if _, err := s.Put(ctx, journal.Date{Year: 2020, Month: 1, Day: day}, "synthetic", ""); err != nil {
			t.Fatal(err)
		}
	}
	scoped := s.ForOperation().(*Store)
	if _, err := scoped.ListYear(ctx, 2020); err != nil {
		t.Fatal(err)
	}
	before := f.lists
	for day := 1; day <= 3; day++ {
		if _, err := scoped.Get(ctx, journal.Date{Year: 2020, Month: 1, Day: day}); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.lists - before; got != 0 {
		t.Fatalf("Get added %d lists after ListYear; want zero", got)
	}
	// A fresh operation without ListYear still shares one complete listing.
	scoped = s.ForOperation().(*Store)
	before = f.lists
	for day := 1; day <= 3; day++ {
		if _, err := scoped.Get(ctx, journal.Date{Year: 2020, Month: 1, Day: day}); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.lists - before; got != 3 {
		t.Fatalf("Get-only discovery=%d want root/year/files=3", got)
	}
}

func TestPreviewsValidateFullContent(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       error
	}{
		{"large valid", strings.Repeat(" ", 65535) + "🙂end", nil},
		{"invalid after prefix", strings.Repeat("a", 65536) + "\xff", journal.ErrInvalidText},
		{"at limit", strings.Repeat("a", maxContentBytes), nil},
		{"over limit", strings.Repeat("a", maxContentBytes+1), journal.ErrUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := testStore(t)
			ctx := context.Background()
			d := journal.Date{Year: 2020, Month: 1, Day: 1}
			if _, err := s.Put(ctx, d, "seed", ""); err != nil {
				t.Fatal(err)
			}
			file, err := s.entryFile(ctx, d)
			if err != nil {
				t.Fatal(err)
			}
			f.mu.Lock()
			f.bodies[file.Id] = []byte(tc.text)
			f.files[file.Id].Md5Checksum = checksum([]byte(tc.text))
			f.mu.Unlock()
			previews, err := s.Previews(ctx, []journal.Date{d}, 20)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
			if tc.want == nil && previews[d] != journal.PreviewPrefix(tc.text, 20) {
				t.Fatalf("preview=%q", previews[d])
			}
			if tc.want != nil && previews != nil {
				t.Fatal("partial previews returned on validation error")
			}
		})
	}
}

func TestReadDiscoveryPaginationDuplicatesAndBounds(t *testing.T) {
	for _, mode := range []string{"pagination", "oversized", "full query cache"} {
		t.Run(mode, func(t *testing.T) {
			s, f := testStore(t)
			ctx := context.Background()
			d := journal.Date{Year: 2020, Month: 1, Day: 1}
			if _, err := s.Put(ctx, d, "first", ""); err != nil {
				t.Fatal(err)
			}
			original, err := s.entryFile(ctx, d)
			if err != nil {
				t.Fatal(err)
			}
			duplicate := *original
			duplicate.Id = "duplicate"
			duplicate.CreatedTime = "2099-01-01T00:00:00Z"
			duplicate.Md5Checksum = checksum([]byte("duplicate"))
			f.files[duplicate.Id] = &duplicate
			f.bodies[duplicate.Id] = []byte("duplicate")
			if mode == "oversized" {
				f.pageSize = 1000
				for i := 0; i < maxDiscoveryFiles; i++ {
					id := fmt.Sprintf("unrelated-%d", i)
					f.files[id] = &drive.File{Id: id, Name: id, MimeType: "text/plain", Parents: original.Parents}
				}
			} else {
				f.pageSize = 1
			}
			scoped := s.ForOperation().(*Store)
			if mode == "full query cache" {
				for i := 0; i < maxDiscoveryQueries; i++ {
					if _, err := scoped.list(ctx, fmt.Sprintf("synthetic-%d", i)); err != nil {
						t.Fatal(err)
					}
				}
			}
			before := f.lists
			for range 2 {
				entry, err := scoped.Get(ctx, d)
				if err != nil || entry.Text != "first" {
					t.Fatalf("duplicate selection=%+v err=%v", entry, err)
				}
			}
			t.Logf("%s: two full reads metadata requests=%d retained queries=%d", mode, f.lists-before, len(scoped.discovery.queries))
			if len(scoped.discovery.queries) > maxDiscoveryQueries {
				t.Fatal("query bound exceeded")
			}
			for _, files := range scoped.discovery.queries {
				if len(files) > maxDiscoveryFiles {
					t.Fatal("oversized files retained")
				}
			}
			if mode == "pagination" && f.lists-before != 4 {
				t.Fatalf("root/year/two file pages reused: %d", f.lists-before)
			}
			// Reusing read discovery must never hide a changed version from Put.
			f.files[original.Id].Md5Checksum = checksum([]byte("external"))
			f.bodies[original.Id] = []byte("external")
			if _, err := scoped.Put(ctx, d, "overwrite", journal.Version(checksum([]byte("first")))); !errors.Is(err, journal.ErrConflict) {
				t.Fatalf("mutation borrowed stale read version: %v", err)
			}
			f.files[original.Id].Trashed = true
			f.files[duplicate.Id].Trashed = true
			if _, err := s.ForOperation().Get(ctx, d); !errors.Is(err, journal.ErrNotFound) {
				t.Fatalf("new operation missed trash: %v", err)
			}
		})
	}
}

func TestPreviewChecksumValidation(t *testing.T) {
	for _, version := range []string{"", "wrong"} {
		t.Run(version, func(t *testing.T) {
			s, f := testStore(t)
			ctx := context.Background()
			d := journal.Date{Year: 2020, Month: 1, Day: 1}
			if _, err := s.Put(ctx, d, "synthetic", ""); err != nil {
				t.Fatal(err)
			}
			file, err := s.entryFile(ctx, d)
			if err != nil {
				t.Fatal(err)
			}
			f.files[file.Id].Md5Checksum = version
			cache := cachestore.New(newLRUStorage(t, 2*time.Minute), 2*time.Minute)
			previews, err := cache.Wrap("google:synthetic", s.ForOperation()).Previews(ctx, []journal.Date{d}, 10)
			if !errors.Is(err, journal.ErrUnavailable) || previews != nil {
				t.Fatalf("unchecked version accepted: %v", err)
			}
			f.files[file.Id].Md5Checksum = checksum([]byte("synthetic"))
			entry, err := cache.Wrap("google:synthetic", s.ForOperation()).Get(ctx, d)
			if err != nil || entry.Text != "synthetic" {
				t.Fatalf("failed validation was cached: %v", err)
			}
		})
	}
}

func TestSharedContentBrowsingRequestCounts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var fake *fakeDrive
		var bytes, active, peak atomic.Int64
		p, f, counts := collapseProvider(t, func(w http.ResponseWriter, r *http.Request) bool {
			if r.Method != http.MethodGet {
				return false
			}
			if r.URL.Query().Get("alt") == "media" {
				n := active.Add(1)
				defer active.Add(-1)
				for old := peak.Load(); n > old; old = peak.Load() {
					if peak.CompareAndSwap(old, n) {
						break
					}
				}
				time.Sleep(time.Millisecond)
				id := strings.TrimPrefix(r.URL.Path, "/drive/v3/files/")
				fake.mu.Lock()
				size := len(fake.bodies[id])
				fake.mu.Unlock()
				bytes.Add(int64(size))
				fake.ServeHTTP(w, r)
				return true
			}
			time.Sleep(time.Millisecond)
			return false
		})
		fake = f
		ctx := context.Background()
		raw := collapseStore(t, p, ctx)
		text := strings.Repeat("synthetic entry\n", 5000)
		dates := []journal.Date{{Year: 2020, Month: 1, Day: 1}}
		for day := 1; day <= 7; day++ {
			dates = append(dates, journal.Date{Year: 2021, Month: 1, Day: day})
		}
		for _, d := range dates {
			if _, err := raw.Put(ctx, d, text, ""); err != nil {
				t.Fatal(err)
			}
		}
		newCache := func() *cachestore.Cache { return cachestore.New(newLRUStorage(t, 2*time.Minute), 2*time.Minute) }
		service := func(cache *cachestore.Cache) *journal.Service {
			return journal.NewService(cache.Wrap("google:synthetic-account", collapseStore(t, p, ctx)), benchmarkClock{})
		}
		reset := func() { counts.counts = make(map[string]int); bytes.Store(0); peak.Store(0) }
		check := func(label string, lists, downloads int, start time.Time) {
			t.Helper()
			t.Logf("%s lists=%d downloads=%d content bytes=%d elapsed=%v peak=%d", label, counts.counts["list"], counts.counts["download"], bytes.Load(), time.Since(start), peak.Load())
			if counts.counts["list"] != lists || counts.counts["download"] != downloads || bytes.Load() != int64(downloads*len(text)) {
				t.Fatalf("%s unexpected work", label)
			}
		}
		cache := newCache()
		reset()
		start := time.Now()
		if page, err := service(cache).ListEntries(ctx, 1, 10); err != nil || len(page.Items) != 8 {
			t.Fatalf("page=%+v err=%v", page, err)
		}
		check("cold list", 4, 8, start)
		reset()
		start = time.Now()
		for _, d := range dates {
			entry, err := service(cache).Open(ctx, d)
			if err != nil || entry.Text != text || entry.Version != journal.Version(checksum([]byte(text))) {
				t.Fatal("list did not warm complete editor entry")
			}
		}
		check("list -> all details", 0, 0, start)
		reset()
		start = time.Now()
		if _, err := service(cache).OpenFresh(ctx, dates[0]); err != nil {
			t.Fatal(err)
		}
		check("fresh detail bypass", 3, 1, start)
		cache = newCache()
		reset()
		start = time.Now()
		entry, err := service(cache).Open(ctx, dates[0])
		if err != nil {
			t.Fatal(err)
		}
		check("cold detail", 3, 1, start)
		reset()
		start = time.Now()
		if _, err := service(cache).ListEntries(ctx, 1, 10); err != nil {
			t.Fatal(err)
		}
		check("detail -> list", 4, 7, start)
		// Warm index snapshots but deliberately cold content; no per-date filename queries.
		cache = newCache()
		wrapped := cache.Wrap("google:synthetic-account", raw).(journal.OperationStore).ForOperation()
		if _, err := wrapped.ListYears(ctx); err != nil {
			t.Fatal(err)
		}
		for _, year := range []int{2020, 2021} {
			if _, err := wrapped.ListYear(ctx, year); err != nil {
				t.Fatal(err)
			}
		}
		reset()
		start = time.Now()
		if _, err := service(cache).ListEntries(ctx, 1, 10); err != nil {
			t.Fatal(err)
		}
		check("warm indexes / cold content", 5, 8, start)
		// Updating and trashing must invalidate shared content used by both consumers.
		if _, err := service(cache).Save(ctx, dates[0], text, entry.Version); err != nil {
			t.Fatal(err)
		}
		reset()
		start = time.Now()
		if _, err := service(cache).ListEntries(ctx, 1, 10); err != nil {
			t.Fatal(err)
		}
		check("after save", 3, 1, start)
		if err := service(cache).Delete(ctx, dates[0]); err != nil {
			t.Fatal(err)
		}
		if page, err := service(cache).ListEntries(ctx, 1, 10); err != nil || len(page.Items) != 7 {
			t.Fatal("trash retained list membership")
		}
		missing, err := service(cache).Open(ctx, dates[0])
		if !errors.Is(err, journal.ErrNotFound) || missing.Exists {
			t.Fatal("trash retained shared content")
		}
	})
}

func TestCachedGoogleStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		raw, _ := testStore(t)
		return cachestore.New(newLRUStorage(t, 2*time.Minute), 2*time.Minute).Wrap("google:synthetic-account", raw)
	})
}

func newLRUStorage(t *testing.T, ttl time.Duration) *lru.Adapter {
	t.Helper()
	storage, err := lru.New(ttl)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Error(err)
		}
	})
	return storage
}
