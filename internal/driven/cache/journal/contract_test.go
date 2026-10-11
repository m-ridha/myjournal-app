package cachestore_test

import (
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/core/journal/storetest"
	"github/mridha/myjournal-app/internal/driven/cache/journal"
	cacheAdapter "github/mridha/myjournal-app/internal/driven/cache/local/bigcache"
	"github/mridha/myjournal-app/internal/driven/cache/local/lru"
	"github/mridha/myjournal-app/internal/driven/provider/local"
)

func TestCachedLocalStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		backend, err := cacheAdapter.New(time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := backend.Close(); err != nil {
				t.Error(err)
			}
		})
		return cachestore.New(backend, time.Minute).Wrap("local/dev", local.NewStore(t.TempDir()))
	})
}

func TestLRUCachedLocalStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		return cachestore.New(lru.New(time.Minute), time.Minute).Wrap("local/dev", local.NewStore(t.TempDir()))
	})
}
