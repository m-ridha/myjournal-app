package google

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github/m-ridha/myjournal-app/internal/core/journal"
	cachestore "github/m-ridha/myjournal-app/internal/driven/cache/journal"
)

func TestWarmingUsesIndependentGoogleClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var refreshes atomic.Int32
		p, _, counts := collapseProvider(t, func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Host != "oauth2.googleapis.com" {
				return false
			}
			refreshes.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"synthetic-access","token_type":"Bearer","expires_in":3600}`))
			return true
		})
		snapshot := func() map[string]int {
			counts.mu.Lock()
			defer counts.mu.Unlock()
			return maps.Clone(counts.counts)
		}
		cache, stop := cachestore.NewWithWarming(newLRUStorage(t, 2*time.Minute), 2*time.Minute)
		defer stop()
		ctx, cancel := context.WithCancel(context.Background())
		release := make(chan struct{})
		store := cache.WrapWithFactory("google:synthetic-account", collapseStore(t, p, ctx), func(work context.Context) (journal.Store, error) {
			select {
			case <-release:
			case <-work.Done():
				return nil, work.Err()
			}
			return p.Store(work, collapseConnection())
		})
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		if _, err := store.Put(ctx, date, "synthetic", ""); err != nil {
			t.Fatal(err)
		}
		saved := snapshot()
		cancel()
		p.dropToken("synthetic-refresh")
		close(release)
		synctest.Wait()
		if refreshes.Load() != 1 {
			t.Fatal("warming did not refresh using its own OAuth context")
		}
		before := snapshot()
		t.Logf("PUT warming: lists=%d downloads=%d", before["list"]-saved["list"], before["download"]-saved["download"])
		if before["list"]-saved["list"] != 3 || before["download"]-saved["download"] != 1 {
			t.Fatal("unexpected PUT warming work")
		}
		entry, err := store.Get(context.Background(), date)
		if err != nil || entry.Text != "synthetic" {
			t.Fatalf("entry=%+v err=%v", entry, err)
		}
		if _, err := store.ListYears(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ListYear(context.Background(), 2020); err != nil {
			t.Fatal(err)
		}
		if !maps.Equal(snapshot(), before) {
			t.Fatal("detail or indexes did not hit warmed snapshots")
		}
		fresh := cache.WrapWithFactory("google:synthetic-account", collapseStore(t, p, context.Background()), func(work context.Context) (journal.Store, error) { return p.Store(work, collapseConnection()) })
		if err := fresh.Trash(context.Background(), date); err != nil {
			t.Fatal(err)
		}
		trashed := snapshot()
		synctest.Wait()
		before = snapshot()
		t.Logf("DELETE warming: lists=%d downloads=%d", before["list"]-trashed["list"], before["download"]-trashed["download"])
		if before["list"]-trashed["list"] != 3 || before["download"]-trashed["download"] != 0 {
			t.Fatal("unexpected DELETE warming work")
		}
		if _, err := fresh.Get(context.Background(), date); !errors.Is(err, journal.ErrNotFound) {
			t.Fatal(err)
		}
		if _, err := fresh.ListYears(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := fresh.ListYear(context.Background(), 2020); err != nil {
			t.Fatal(err)
		}
		if !maps.Equal(snapshot(), before) {
			t.Fatal("blank date or indexes did not hit warmed snapshots")
		}
	})
}
