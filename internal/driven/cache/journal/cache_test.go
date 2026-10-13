package cachestore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/civil"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

type fakeStore struct {
	mu       sync.Mutex
	calls    int
	years    []int
	entries  map[int][]journal.EntryMeta
	contents map[civil.Date]journal.Entry
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		entries:  make(map[int][]journal.EntryMeta),
		contents: make(map[civil.Date]journal.Entry),
	}
}

func (s *fakeStore) ListYears(ctx context.Context) ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return append([]int(nil), s.years...), nil
}

func (s *fakeStore) ListYear(ctx context.Context, year int) ([]journal.EntryMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	out := make([]journal.EntryMeta, len(s.entries[year]))
	copy(out, s.entries[year])
	return out, nil
}

func (s *fakeStore) Get(ctx context.Context, d civil.Date) (journal.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if e, ok := s.contents[d]; ok {
		return e, nil
	}
	return journal.Entry{}, journal.ErrNotFound
}

func (s *fakeStore) Previews(ctx context.Context, dates []civil.Date, maxBytes int) (map[civil.Date]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	out := make(map[civil.Date]string, len(dates))
	for _, d := range dates {
		if e, ok := s.contents[d]; ok {
			preview := journal.PreviewPrefix(e.Text, maxBytes)
			out[d] = preview
		}
	}
	return out, nil
}

func (s *fakeStore) Put(ctx context.Context, d civil.Date, text string, base journal.Version) (journal.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.contents[d] = journal.Entry{Date: d, Text: text, Version: journal.Version("v1"), Exists: true}
	return s.contents[d], nil
}

func (s *fakeStore) Trash(ctx context.Context, d civil.Date) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if _, ok := s.contents[d]; !ok {
		return journal.ErrNotFound
	}
	delete(s.contents, d)
	return nil
}

func TestCacheWrapsListYear(t *testing.T) {
	base := newFakeStore()
	base.years = []int{2026, 2025}
	base.entries[2026] = []journal.EntryMeta{{Date: civil.Date{Year: 2026, Month: 9, Day: 20}, ModifiedTime: time.Now()}}
	base.contents[civil.Date{Year: 2026, Month: 9, Day: 20}] = journal.Entry{Date: civil.Date{Year: 2026, Month: 9, Day: 20}, Text: "hello", Version: journal.Version("v1"), Exists: true}

	store := New(newTestStorage(), 5*time.Minute).Wrap("user-1", base)

	first, err := store.ListYear(context.Background(), 2026)
	if err != nil {
		t.Fatalf("ListYear() first call error: %v", err)
	}
	if len(first) != 1 || first[0].Date.Year != 2026 {
		t.Fatalf("ListYear() first call = %#v, want a single 2026 record", first)
	}

	second, err := store.ListYear(context.Background(), 2026)
	if err != nil {
		t.Fatalf("ListYear() second call error: %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("ListYear() second call len = %d, want 1", len(second))
	}
	if base.calls != 1 {
		t.Fatalf("underlying ListYear calls = %d, want 1 (a cached hit should not re-read from base)", base.calls)
	}
}

func TestCacheRetainsExistingMembershipOnPut(t *testing.T) {
	base := newFakeStore()
	base.years = []int{2026}
	base.entries[2026] = []journal.EntryMeta{{Date: civil.Date{Year: 2026, Month: 9, Day: 20}, ModifiedTime: time.Now()}}
	base.contents[civil.Date{Year: 2026, Month: 9, Day: 20}] = journal.Entry{Date: civil.Date{Year: 2026, Month: 9, Day: 20}, Text: "old", Version: journal.Version("old"), Exists: true}

	store := New(newTestStorage(), 5*time.Minute).Wrap("user-1", base)
	if _, err := store.ListYear(context.Background(), 2026); err != nil {
		t.Fatalf("ListYear() initial call error: %v", err)
	}

	if _, err := store.Put(context.Background(), civil.Date{Year: 2026, Month: 9, Day: 20}, "new", "old"); err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	if _, err := store.ListYear(context.Background(), 2026); err != nil {
		t.Fatalf("ListYear() after Put error: %v", err)
	}
	if base.calls != 2 {
		t.Fatalf("expected existing membership to remain cached after Put, got calls=%d", base.calls)
	}
}

func TestCachePreviewsReturnValue(t *testing.T) {
	base := newFakeStore()
	base.contents[civil.Date{Year: 2026, Month: 9, Day: 20}] = journal.Entry{Date: civil.Date{Year: 2026, Month: 9, Day: 20}, Text: "preview text", Version: journal.Version("v1"), Exists: true}
	store := New(newTestStorage(), 5*time.Minute).Wrap("user-1", base)

	first, err := store.Previews(context.Background(), []civil.Date{{Year: 2026, Month: 9, Day: 20}}, 4)
	if err != nil {
		t.Fatalf("Previews() error: %v", err)
	}
	if first[civil.Date{Year: 2026, Month: 9, Day: 20}] != "prev" {
		t.Fatalf("Previews() = %q, want %q", first[civil.Date{Year: 2026, Month: 9, Day: 20}], "prev")
	}

	second, err := store.Previews(context.Background(), []civil.Date{{Year: 2026, Month: 9, Day: 20}}, 4)
	if err != nil {
		t.Fatalf("Previews() second call error: %v", err)
	}
	if second[civil.Date{Year: 2026, Month: 9, Day: 20}] != "prev" {
		t.Fatalf("Previews() second call = %q, want %q", second[civil.Date{Year: 2026, Month: 9, Day: 20}], "prev")
	}
	if base.calls != 1 {
		t.Fatalf("base Previews calls = %d, want 1 (cached hits should not re-read from base)", base.calls)
	}
}

func TestCacheExpiresEntries(t *testing.T) {
	base := newFakeStore()
	cache := New(newTestStorage(), time.Minute)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }
	store := cache.Wrap("user-1", base)

	for range 2 {
		if _, err := store.ListYear(context.Background(), 2026); err != nil {
			t.Fatalf("ListYear() error = %v", err)
		}
	}
	if base.calls != 1 {
		t.Fatalf("underlying calls before expiry = %d, want 1", base.calls)
	}

	now = now.Add(time.Minute)
	if _, err := store.ListYear(context.Background(), 2026); err != nil {
		t.Fatalf("ListYear() after expiry error = %v", err)
	}
	if base.calls != 2 {
		t.Fatalf("underlying calls after expiry = %d, want 2", base.calls)
	}
}

func TestCacheIsSharedPerAccountAndIsolatedAcrossAccounts(t *testing.T) {
	cache := New(newTestStorage(), time.Minute)
	firstBase := newFakeStore()
	firstBase.years = []int{2026}
	first := cache.Wrap("user-1", firstBase)

	if _, err := first.ListYears(context.Background()); err != nil {
		t.Fatalf("first ListYears() error = %v", err)
	}
	secondBase := newFakeStore()
	secondBase.years = []int{2025}
	second := cache.Wrap("user-2", secondBase)
	got, err := second.ListYears(context.Background())
	if err != nil {
		t.Fatalf("second ListYears() error = %v", err)
	}
	if len(got) != 1 || got[0] != 2025 {
		t.Fatalf("second account years = %v, want [2025]", got)
	}

	sharedBase := newFakeStore()
	shared := cache.Wrap("user-1", sharedBase)
	got, err = shared.ListYears(context.Background())
	if err != nil {
		t.Fatalf("shared ListYears() error = %v", err)
	}
	if len(got) != 1 || got[0] != 2026 || sharedBase.calls != 0 {
		t.Fatalf("same-account cached years = %v, backing calls = %d; want [2026], 0", got, sharedBase.calls)
	}
}

func TestPreviewBudgetsShareEntryContent(t *testing.T) {
	base := newFakeStore()
	date := civil.Date{Year: 2026, Month: 9, Day: 20}
	base.contents[date] = journal.Entry{Date: date, Text: "preview text", Version: "v1", Exists: true}
	store := New(newTestStorage(), time.Minute).Wrap("user-1", base)

	short, err := store.Previews(context.Background(), []civil.Date{date}, 4)
	if err != nil {
		t.Fatalf("short Previews() error = %v", err)
	}
	long, err := store.Previews(context.Background(), []civil.Date{date}, 8)
	if err != nil {
		t.Fatalf("long Previews() error = %v", err)
	}
	if short[date] != "prev" || long[date] != "preview " {
		t.Fatalf("previews = %q and %q, want %q and %q", short[date], long[date], "prev", "preview ")
	}
	if base.calls != 1 {
		t.Fatalf("underlying Get calls = %d, want 1 across different preview budgets", base.calls)
	}
}

func TestCacheWrapReturnsErrNotFoundForMissingValue(t *testing.T) {
	base := newFakeStore()
	store := New(newTestStorage(), 5*time.Minute).Wrap("user-1", base)
	_, err := store.Get(context.Background(), civil.Date{Year: 2026, Month: 9, Day: 20})
	if !errors.Is(err, journal.ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestInvalidationEpochRejectsOldFillAfterAccountChurn(t *testing.T) {
	c := New(newTestStorage(), time.Minute)
	key := cacheKey{account: "original", kind: "years"}
	_, epoch, _ := c.get(context.Background(), key)
	for range 1000 {
		c.invalidateDate(context.Background(), "another", civil.Date{Year: 2026, Month: 1, Day: 1}, true)
	}
	c.set(context.Background(), key, []int{2026}, epoch)
	if _, _, ok := c.get(context.Background(), key); ok {
		t.Fatal("old in-flight fill survived epoch change")
	}
}

func TestEntryCacheTTLInvalidationAndFreshReads(t *testing.T) {
	ctx := context.Background()
	base := newFakeStore()
	date := civil.Date{Year: 2020, Month: 1, Day: 1}
	other := civil.Date{Year: 2020, Month: 1, Day: 2}
	base.contents[date] = journal.Entry{Date: date, Text: "first", Version: "v1", Exists: true}
	base.contents[other] = journal.Entry{Date: other, Text: "other", Version: "v1", Exists: true}
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	cache := New(newTestStorage(), 2*time.Minute)
	cache.now = func() time.Time { return now }
	store := cache.Wrap("google:account", base)
	read := func(d civil.Date) journal.Entry {
		t.Helper()
		e, err := store.Get(ctx, d)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	read(date)
	read(other)
	base.contents[date] = journal.Entry{Date: date, Text: "external", Version: "v2", Exists: true}
	if got := read(date); got.Text != "first" || got.Version != "v1" || base.calls != 2 {
		t.Fatalf("cached=%+v calls=%d", got, base.calls)
	}
	fresh, err := store.(journal.FreshEntryStore).GetFresh(ctx, date)
	if err != nil || fresh.Text != "external" || base.calls != 3 {
		t.Fatalf("fresh=%+v err=%v calls=%d", fresh, err, base.calls)
	}
	// A fresh read does not renew or replace the ordinary snapshot.
	now = now.Add(2 * time.Minute)
	if got := read(date); got.Text != "external" || base.calls != 4 {
		t.Fatalf("expired=%+v calls=%d", got, base.calls)
	}
	read(other)
	if _, err := store.Put(ctx, date, "saved", "v2"); err != nil {
		t.Fatal(err)
	}
	calls := base.calls
	if got := read(date); got.Text != "saved" || base.calls != calls+1 {
		t.Fatalf("post-save=%+v calls=%d", got, base.calls)
	}
	read(other)
	if base.calls != calls+1 {
		t.Fatal("save discarded unrelated content")
	}
	if err := store.Trash(ctx, date); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, date); !errors.Is(err, journal.ErrNotFound) {
		t.Fatalf("post-trash=%v", err)
	}
	if _, err := store.Put(ctx, date, "recreated", ""); err != nil {
		t.Fatal(err)
	}
	if got := read(date); got.Text != "recreated" {
		t.Fatal("create did not invalidate cached absence")
	}
	isolated := cache.Wrap("google:different-account", base)
	calls = base.calls
	if _, err := isolated.Get(ctx, date); err != nil || base.calls != calls+1 {
		t.Fatal("account isolation failed")
	}
}

func TestMissingEntriesCacheExpiryAndFreshBypass(t *testing.T) {
	ctx := context.Background()
	date := civil.Date{Year: 2020, Month: 1, Day: 1}
	base := newFakeStore()
	cache := New(newTestStorage(), 2*time.Minute)
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }
	store := cache.Wrap("account", base)
	for range 2 {
		if _, err := store.Get(ctx, date); !errors.Is(err, journal.ErrNotFound) {
			t.Fatalf("missing=%v", err)
		}
	}
	if base.calls != 1 {
		t.Fatalf("missing reads=%d", base.calls)
	}
	base.contents[date] = journal.Entry{Date: date, Text: "external create", Version: "v1", Exists: true}
	if _, err := store.Get(ctx, date); !errors.Is(err, journal.ErrNotFound) {
		t.Fatal("ordinary read did not retain absence")
	}
	fresh, err := store.(journal.FreshEntryStore).GetFresh(ctx, date)
	if err != nil || fresh.Text != "external create" {
		t.Fatal("fresh read reused cached absence")
	}
	now = now.Add(2 * time.Minute)
	entry, err := store.Get(ctx, date)
	if err != nil || entry.Text != "external create" || base.calls != 3 {
		t.Fatalf("expiry=%+v err=%v calls=%d", entry, err, base.calls)
	}
	if err := store.Trash(ctx, date); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, date); !errors.Is(err, journal.ErrNotFound) {
		t.Fatal("trash did not invalidate positive snapshot")
	}
	calls := base.calls
	if _, err := store.Get(ctx, date); !errors.Is(err, journal.ErrNotFound) || base.calls != calls {
		t.Fatal("post-trash absence not cached")
	}
}

func TestEntryFailuresAreNotCached(t *testing.T) {
	for _, failure := range []error{journal.ErrUnavailable, journal.ErrAuthRequired, context.Canceled} {
		t.Run(failure.Error(), func(t *testing.T) {
			var calls int
			base := slowReadStore{get: func(context.Context, civil.Date) (journal.Entry, error) { calls++; return journal.Entry{}, failure }}
			store := New(newTestStorage(), 2*time.Minute).Wrap("account", base)
			for range 2 {
				if _, err := store.Get(context.Background(), civil.Date{Year: 2020, Month: 1, Day: 1}); !errors.Is(err, failure) {
					t.Fatal(err)
				}
			}
			if calls != 2 {
				t.Fatal("operational failure cached")
			}
		})
	}
}
