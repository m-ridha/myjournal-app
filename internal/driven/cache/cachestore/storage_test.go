package cachestore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"myjournal-app/internal/core/journal"
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

type cancelledMutationStore struct {
	*fakeStore
	cancel context.CancelFunc
}

func (s *cancelledMutationStore) Put(ctx context.Context, d journal.Date, text string, version journal.Version) (journal.Entry, error) {
	_, _ = s.fakeStore.Put(ctx, d, text, version)
	s.cancel()
	return journal.Entry{}, ctx.Err()
}

func (s *cancelledMutationStore) Trash(ctx context.Context, d journal.Date) error {
	_ = s.fakeStore.Trash(ctx, d)
	s.cancel()
	return ctx.Err()
}

func TestCancelledInvalidationPreservesOtherAccounts(t *testing.T) {
	for _, mutation := range []string{"put", "trash"} {
		t.Run(mutation, func(t *testing.T) {
			testCancelledMutation(t, mutation)
		})
	}
}

func testCancelledMutation(t *testing.T, mutation string) {
	t.Helper()
	storage := newTestStorage()
	c := New(storage, time.Minute)
	base := newFakeStore()
	base.years = []int{2026}
	d := journal.Date{Year: 2026, Month: 1, Day: 1}
	base.contents[d] = journal.Entry{Date: d, Text: "old", Exists: true}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := c.Wrap("a", &cancelledMutationStore{fakeStore: base, cancel: cancel})
	otherBase := newFakeStore()
	otherBase.years = []int{2026}
	other := c.Wrap("b", otherBase)
	if c.quarantineStripe("a") == c.quarantineStripe("b") {
		t.Fatal("test accounts share a quarantine stripe")
	}
	_, _ = store.ListYears(context.Background())
	_, _ = store.Get(context.Background(), d)
	_, _ = other.ListYears(context.Background())
	var err error
	if mutation == "put" {
		_, err = store.Put(ctx, d, "new", "old")
	} else {
		err = store.Trash(ctx, d)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("mutation error: %v", err)
	}
	base.years = []int{2025}
	got, err := store.ListYears(context.Background())
	if err != nil || got[0] != 2025 {
		t.Fatalf("cancelled invalidation reused old data: %v %v", got, err)
	}
	_, _ = other.ListYears(context.Background())
	_, _ = store.ListYears(context.Background())
	entry, entryErr := store.Get(context.Background(), d)
	if mutation == "put" && (entryErr != nil || entry.Text != "new") {
		t.Fatalf("put left stale entry: %v %v", entry, entryErr)
	}
	if mutation == "trash" && !errors.Is(entryErr, journal.ErrNotFound) {
		t.Fatalf("trash left stale entry: %v %v", entry, entryErr)
	}
	if otherBase.calls != 1 || base.calls != 5 {
		t.Fatalf("cache did not remain available: affected=%d other=%d", base.calls, otherBase.calls)
	}
}

func TestInvalidationFailureQuarantinesAndRecovers(t *testing.T) {
	ctx := context.Background()
	storage := newTestStorage()
	c := New(storage, time.Minute)
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	base, otherBase := newFakeStore(), newFakeStore()
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	base.contents[d] = journal.Entry{Date: d, Text: "old"}
	otherBase.years = []int{2020}
	store, other := c.Wrap("a", base), c.Wrap("b", otherBase)
	if c.quarantineStripe("a") == c.quarantineStripe("b") {
		t.Fatal("test accounts share a quarantine stripe")
	}
	_, _ = store.Get(ctx, d)
	_, _ = other.ListYears(ctx)
	storage.deleteErr = errors.New("delete failed")
	if _, err := store.Put(ctx, d, "new", "old"); err != nil {
		t.Fatal(err)
	}
	storage.deleteErr = nil
	for range 2 {
		entry, err := store.Get(ctx, d)
		if err != nil || entry.Text != "new" {
			t.Fatalf("quarantine returned stale entry: %v %v", entry, err)
		}
	}
	_, _ = other.ListYears(ctx)
	if otherBase.calls != 1 {
		t.Fatal("unrelated account lost its snapshot")
	}
	now = now.Add(time.Minute)
	entry, err := store.Get(ctx, d)
	if err != nil || entry.Text != "new" {
		t.Fatalf("recovery returned stale entry: %v %v", entry, err)
	}
	before := base.calls
	_, _ = store.Get(ctx, d)
	if base.calls != before {
		t.Fatal("cache did not recover after snapshot expiry")
	}
}

func TestInvalidationLockTimeoutRejectsOldFill(t *testing.T) {
	ctx := context.Background()
	storage := newTestStorage()
	c := New(storage, time.Minute)
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	key := cacheKey{account: "a", kind: "years"}
	_, generation, _ := c.get(ctx, key)
	c.gate <- struct{}{}
	c.invalidateDate(ctx, "a", journal.Date{Year: 2020, Month: 1, Day: 1}, true)
	c.unlock()
	if c.active("a") {
		t.Fatal("failed lock wait did not quarantine account")
	}
	now = now.Add(time.Minute)
	c.set(ctx, key, []int{2020}, generation)
	if _, _, ok := c.get(ctx, key); ok {
		t.Fatal("pre-invalidation load filled after quarantine")
	}
}

func TestCorruptSnapshotCanRecover(t *testing.T) {
	ctx := context.Background()
	storage := newTestStorage()
	c := New(storage, time.Minute)
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	base := newFakeStore()
	base.years = []int{2026}
	store := c.Wrap("a", base)
	storage.items[(cacheKey{account: "a", kind: "years"}).string()] = []byte("corrupt")
	_, _ = store.ListYears(ctx)
	now = now.Add(time.Minute)
	_, _ = store.ListYears(ctx)
	_, _ = store.ListYears(ctx)
	if base.calls != 2 {
		t.Fatalf("corrupt snapshot prevented recovery: %d backing reads", base.calls)
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
