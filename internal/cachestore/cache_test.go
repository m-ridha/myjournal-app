package cachestore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/journal"

	"cloud.google.com/go/civil"
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

	store := New(5*time.Minute).Wrap("user-1", base)

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

	store := New(5*time.Minute).Wrap("user-1", base)
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
	store := New(5*time.Minute).Wrap("user-1", base)

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
	cache := New(time.Minute)
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

func TestCacheEnforcesEntryBound(t *testing.T) {
	base := newFakeStore()
	cache := NewWithCapacity(time.Minute, 1)
	store := cache.Wrap("user-1", base)

	if _, err := store.ListYears(context.Background()); err != nil {
		t.Fatalf("ListYears() error = %v", err)
	}
	if _, err := store.ListYear(context.Background(), 2026); err != nil {
		t.Fatalf("ListYear() error = %v", err)
	}
	if got := len(cache.items); got != 1 {
		t.Fatalf("cached entries = %d, want capacity 1", got)
	}
	if _, err := store.ListYears(context.Background()); err != nil {
		t.Fatalf("ListYears() after eviction error = %v", err)
	}
	if base.calls != 3 {
		t.Fatalf("underlying calls after eviction = %d, want 3", base.calls)
	}
}

func TestCacheIsSharedPerAccountAndIsolatedAcrossAccounts(t *testing.T) {
	cache := New(time.Minute)
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

func TestPreviewCacheIncludesSizeLimit(t *testing.T) {
	base := newFakeStore()
	date := civil.Date{Year: 2026, Month: 9, Day: 20}
	base.contents[date] = journal.Entry{Date: date, Text: "preview text", Version: "v1", Exists: true}
	store := New(time.Minute).Wrap("user-1", base)

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
	if base.calls != 2 {
		t.Fatalf("underlying Previews calls = %d, want 2 for different size limits", base.calls)
	}
}

func TestCacheWrapReturnsErrNotFoundForMissingValue(t *testing.T) {
	base := newFakeStore()
	store := New(5*time.Minute).Wrap("user-1", base)
	_, err := store.Get(context.Background(), civil.Date{Year: 2026, Month: 9, Day: 20})
	if !errors.Is(err, journal.ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestInvalidationEpochRejectsOldFillAfterAccountChurn(t *testing.T) {
	c := New(time.Minute)
	key := cacheKey{account: "original", kind: "years"}
	_, epoch, _ := c.get(key)
	for range 1000 {
		c.invalidateDate("another", civil.Date{Year: 2026, Month: 1, Day: 1}, true)
	}
	c.set(key, []int{2026}, epoch)
	if _, _, ok := c.get(key); ok {
		t.Fatal("old in-flight fill survived epoch change")
	}
}
