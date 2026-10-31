package cachestore_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"myjournal-app/internal/core/journal"
	"myjournal-app/internal/core/journal/storetest"
	cachestore "myjournal-app/internal/driven/cache/cachestore"
	cacheAdapter "myjournal-app/internal/driven/cache/local/bigcache"
	"myjournal-app/internal/driven/cache/local/lru"
	"myjournal-app/internal/driven/provider/local"
)

func TestCachedLocalStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		storage, err := cacheAdapter.New(time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := storage.Close(); err != nil {
				t.Error(err)
			}
		})
		return cachestore.New(storage, time.Minute).Wrap("local/dev", local.NewStore(t.TempDir()))
	})
}

func TestLRUCachedLocalStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		storage, err := lru.New(time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := storage.Close(); err != nil {
				t.Error(err)
			}
		})
		return cachestore.New(storage, time.Minute).Wrap("local/dev", local.NewStore(t.TempDir()))
	})
}

// Capacity pressure must affect hit rates, never returned content.
type countedStore struct {
	journal.Store
	gets int
}

var _ journal.Store = (*countedStore)(nil)

func (s *countedStore) Get(ctx context.Context, d journal.Date) (journal.Entry, error) {
	s.gets++
	return s.Store.Get(ctx, d)
}
func TestSharedContentBigCacheCapacity(t *testing.T) {
	storage, err := cacheAdapter.New(2 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	base := &countedStore{Store: local.NewStore(t.TempDir())}
	large := journal.Date{Year: 2020, Month: 1, Day: 1}
	small := journal.Date{Year: 2020, Month: 1, Day: 2}
	nearLimit := journal.Date{Year: 2020, Month: 1, Day: 3}
	text := strings.Repeat("a", 1<<20)
	if _, err := base.Put(ctx, large, text, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := base.Put(ctx, small, "small", ""); err != nil {
		t.Fatal(err)
	}
	nearLimitText := strings.Repeat("b", (1<<20)-(16<<10))
	if _, err := base.Put(ctx, nearLimit, nearLimitText, ""); err != nil {
		t.Fatal(err)
	}
	store := cachestore.New(storage, 2*time.Minute).Wrap("local:dev", base)
	for range 2 {
		previews, err := store.Previews(ctx, []journal.Date{large}, 4)
		if err != nil || previews[large] != "aaaa" {
			t.Fatal("capacity changed preview")
		}
	}
	// The serialized 1 MiB entry exceeds the 1 MiB per-shard queue ceiling.
	entry, err := store.Get(ctx, large)
	if err != nil || entry.Text != text || base.gets != 3 {
		t.Fatal("oversized cache item failed to bypass safely")
	}
	// The encoded value passes the adapter's size guard but BigCache's queue
	// can still reject it while growing to its shard limit.
	for range 2 {
		entry, err := store.Get(ctx, nearLimit)
		if err != nil || entry.Text != nearLimitText {
			t.Fatalf("near-limit read: %v", err)
		}
	}
	if base.gets != 5 {
		t.Fatal("near-limit value did not exercise queue capacity")
	}
	if _, err := store.Previews(ctx, []journal.Date{small}, 4); err != nil {
		t.Fatal(err)
	}
	if entry, err := store.Get(ctx, small); err != nil || entry.Text != "small" || base.gets != 6 {
		t.Fatal("capacity bypass disabled ordinary sharing")
	}
}
