package cachestore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/core/journal"
)

// Used only behind Cache's coordinator; production storage lives in adapters.
type testStorage struct {
	items                     map[string][]byte
	getErr, setErr, deleteErr error
}

func newTestStorage() *testStorage { return &testStorage{items: make(map[string][]byte)} }
func (b *testStorage) Get(ctx context.Context, key string) ([]byte, error) {
	if b.getErr != nil {
		return nil, b.getErr
	}
	v, ok := b.items[key]
	if !ok {
		return nil, ErrMiss
	}
	return append([]byte(nil), v...), nil
}
func (b *testStorage) Set(ctx context.Context, key string, value []byte) error {
	if b.setErr != nil {
		return b.setErr
	}
	if len(b.items) >= 2048 {
		for key := range b.items {
			delete(b.items, key)
			break
		}
	}
	b.items[key] = append([]byte(nil), value...)
	return nil
}
func (b *testStorage) Delete(ctx context.Context, key string) error {
	if b.deleteErr != nil {
		return b.deleteErr
	}
	delete(b.items, key)
	return nil
}
func (b *testStorage) DeletePrefix(ctx context.Context, prefix string) error {
	if b.deleteErr != nil {
		return b.deleteErr
	}
	for key := range b.items {
		if strings.HasPrefix(key, prefix) {
			delete(b.items, key)
		}
	}
	return nil
}

func TestCacheStorageFailuresFallBack(t *testing.T) {
	for _, failure := range []string{"get", "set", "decode", "delete", "capacity"} {
		t.Run(failure, func(t *testing.T) {
			storage := newTestStorage()
			c := New(storage, time.Minute)
			base := newFakeStore()
			base.years = []int{2026}
			store := c.Wrap("account", base)
			ctx := context.Background()
			if _, err := store.ListYears(ctx); err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "get":
				storage.getErr = errors.New("unavailable")
			case "set":
				storage.setErr = errors.New("unavailable")
				storage.items = make(map[string][]byte)
			case "decode":
				for k := range storage.items {
					storage.items[k] = []byte("bad")
				}
			case "delete":
				storage.deleteErr = errors.New("partial deletion")
				if _, err := store.Put(ctx, journal.Date{Year: 2026, Month: 1, Day: 1}, "new", ""); err != nil {
					t.Fatal(err)
				}
			case "capacity":
				storage.setErr = ErrCapacity
				storage.items = make(map[string][]byte)
			}
			base.years = []int{2025}
			got, err := store.ListYears(ctx)
			if err != nil || len(got) != 1 || got[0] != 2025 {
				t.Fatalf("fallback %v %v", got, err)
			}
			storage.getErr = nil
			storage.setErr = nil
			storage.deleteErr = nil
			base.years = []int{2024}
			got, err = store.ListYears(ctx)
			if err != nil || len(got) != 1 || got[0] != 2024 {
				t.Fatalf("stale reuse %v %v", got, err)
			}
			if failure == "capacity" {
				before := base.calls
				_, _ = store.ListYears(ctx)
				if base.calls != before {
					t.Fatal("capacity rejection permanently disabled cache")
				}
			}
		})
	}
}

func TestCacheInvalidatesEveryPreviewBudget(t *testing.T) {
	ctx := context.Background()
	base := newFakeStore()
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	base.contents[d] = journal.Entry{Text: "old text"}
	store := New(newTestStorage(), time.Minute).Wrap("account", base)
	for _, budget := range []int{4, 8} {
		_, _ = store.Previews(ctx, []journal.Date{d}, budget)
	}
	if _, err := store.Put(ctx, d, "new text", "old"); err != nil {
		t.Fatal(err)
	}
	for _, budget := range []int{4, 8} {
		got, err := store.Previews(ctx, []journal.Date{d}, budget)
		if err != nil || !strings.HasPrefix(got[d], "new") {
			t.Fatalf("budget %d: %v %v", budget, got, err)
		}
	}
}

func TestCacheCopiesIndexResults(t *testing.T) {
	base := newFakeStore()
	base.years = []int{2026}
	store := New(newTestStorage(), time.Minute).Wrap("a/:preview/", base)
	got, err := store.ListYears(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got[0] = 1900
	got, err = store.ListYears(context.Background())
	if err != nil || got[0] != 2026 {
		t.Fatalf("mutated snapshot: %v %v", got, err)
	}
}

func TestCacheNamespaceIsolation(t *testing.T) {
	c := New(newTestStorage(), time.Minute)
	for _, account := range []string{"a", "a/year/2026", "a/preview/2020-01-01/4", "a:", "a/"} {
		base := newFakeStore()
		base.years = []int{len(account)}
		got, err := c.Wrap(account, base).ListYears(context.Background())
		if err != nil || got[0] != len(account) || base.calls != 1 {
			t.Fatalf("account alias %q: %v %v", account, got, err)
		}
	}
}

func TestDisabledCacheAndFreshGetBypass(t *testing.T) {
	for _, mode := range []string{"nil", "ttl", "account", "get"} {
		t.Run(mode, func(t *testing.T) {
			storage := newTestStorage()
			var port ByteCache = storage
			ttl := time.Minute
			account := "a"
			switch mode {
			case "nil":
				port = nil
			case "ttl":
				ttl = 0
			case "account":
				account = ""
			}
			store := New(port, ttl).Wrap(account, newFakeStore())
			ctx := context.Background()
			if mode == "get" {
				for range 2 {
					_, _ = store.(journal.FreshEntryStore).GetFresh(ctx, journal.Date{Year: 2020, Month: 1, Day: 1})
				}
			} else {
				for range 2 {
					_, _ = store.ListYears(ctx)
				}
			}
			if len(storage.items) != 0 {
				t.Fatal("bypass populated cache")
			}
			if base := store.(*Store).base.(*fakeStore); base.calls != 2 {
				t.Fatalf("backing calls: %d", base.calls)
			}
		})
	}
}

type operationFake struct {
	*fakeStore
	scopes int
}

func (s *operationFake) ForOperation() journal.Store { s.scopes++; return s.fakeStore }
func TestOperationViewRetainsSharedCache(t *testing.T) {
	base := &operationFake{fakeStore: newFakeStore()}
	base.years = []int{2026}
	store := New(newTestStorage(), time.Minute).Wrap("a", base)
	_, _ = store.ListYears(context.Background())
	scoped := store.(journal.OperationStore).ForOperation()
	got, err := scoped.ListYears(context.Background())
	if err != nil || got[0] != 2026 || base.calls != 1 || base.scopes != 1 {
		t.Fatalf("scope lost cache: %v %v calls=%d scopes=%d", got, err, base.calls, base.scopes)
	}
}

func TestCancelledInvalidationDisablesReuse(t *testing.T) {
	storage := newTestStorage()
	c := New(storage, time.Minute)
	base := newFakeStore()
	base.years = []int{2026}
	store := c.Wrap("a", base)
	_, _ = store.ListYears(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Simulate cancellation after the backing mutation completed, before invalidation.
	c.invalidateDate(ctx, "a", journal.Date{Year: 2026, Month: 1, Day: 1}, true)
	base.years = []int{2025}
	got, err := store.ListYears(context.Background())
	if err != nil || got[0] != 2025 {
		t.Fatalf("cancelled invalidation reused old data: %v %v", got, err)
	}
}

func TestSnapshotKindAndExactExpiry(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	data, err := encodeSnapshot("entry", entryResult{Missing: true}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeSnapshot(data, "years", now); err == nil {
		t.Fatal("accepted wrong snapshot kind")
	}
	value, err := decodeSnapshot(data, "entry", now.Add(time.Minute))
	if err != nil || value != nil {
		t.Fatalf("exact expiry: %v %v", value, err)
	}
}

func TestCoordinatorLockWaitIsCancelable(t *testing.T) {
	c := New(newTestStorage(), time.Minute)
	c.gate <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() { _, _, _ = c.get(ctx, cacheKey{account: "a", kind: "years"}); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		c.unlock()
		t.Fatal("coordinator lock wait ignored cancellation")
	}
	c.unlock()
}
