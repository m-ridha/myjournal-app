package cachestore_test

import (
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/cachestore"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/journal/storetest"
	"github/mridha/myjournal-app/internal/provider/local"
)

func TestCachedLocalStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store {
		return cachestore.New(time.Minute).Wrap("local/dev", local.NewStore(t.TempDir()))
	})
}
