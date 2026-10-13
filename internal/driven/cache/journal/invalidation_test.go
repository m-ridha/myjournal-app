package cachestore

import (
	"context"
	"sync"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/core/journal"
)

func TestSaveRetainsUnrelatedPreviewsButRefreshesSavedDate(t *testing.T) {
	ctx := context.Background()
	base := newFakeStore()
	a := journal.Date{Year: 2020, Month: 1, Day: 1}
	b := journal.Date{Year: 2020, Month: 1, Day: 2}
	base.years = []int{2020}
	base.entries[2020] = []journal.EntryMeta{{Date: a}, {Date: b}}
	base.contents[a] = journal.Entry{Date: a, Text: "old", Version: "old", Exists: true}
	base.contents[b] = journal.Entry{Date: b, Text: "unrelated", Version: "b", Exists: true}
	cache := New(newTestStorage(), time.Minute)
	store := cache.Wrap("account", base)
	_, _ = store.ListYears(ctx)
	_, _ = store.ListYear(ctx, 2020)
	_, _ = store.Previews(ctx, []journal.Date{a, b}, 44)
	// Simulate an external edit observed by an unchanged save acknowledgement.
	base.contents[a] = journal.Entry{Date: a, Text: "new", Version: "v1", Exists: true}
	if _, err := store.Put(ctx, a, "new", "v1"); err != nil {
		t.Fatal(err)
	}
	before := base.calls
	_, _ = store.ListYears(ctx)
	_, _ = store.ListYear(ctx, 2020)
	previews, err := store.Previews(ctx, []journal.Date{b}, 44)
	if err != nil || previews[b] != "unrelated" || base.calls != before {
		t.Fatal("unrelated cached data was discarded")
	}
	previews, err = store.Previews(ctx, []journal.Date{a}, 44)
	if err != nil || previews[a] != "new" || base.calls != before+1 {
		t.Fatal("saved-date preview was not refreshed")
	}
}

func TestRetainedCacheTTLDoesNotExtendOnSave(t *testing.T) {
	base := newFakeStore()
	cache := New(newTestStorage(), time.Minute)
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }
	ctx := context.Background()
	store := cache.Wrap("account", base)
	a := journal.Date{Year: 2020, Month: 1, Day: 1}
	b := journal.Date{Year: 2020, Month: 1, Day: 2}
	base.contents[b] = journal.Entry{Date: b, Text: "before"}
	_, _ = store.Previews(ctx, []journal.Date{b}, 44)
	now = now.Add(50 * time.Second)
	_, _ = store.Put(ctx, a, "new", "")
	base.contents[b] = journal.Entry{Date: b, Text: "after external edit"}
	now = now.Add(11 * time.Second)
	got, err := store.Previews(ctx, []journal.Date{b}, 44)
	if err != nil || got[b] != "after external edit" {
		t.Fatalf("TTL extended: %v %v", got, err)
	}
}

type delayedPreviewStore struct {
	*fakeStore
	once             sync.Once
	started, release chan struct{}
}

func (s *delayedPreviewStore) Get(ctx context.Context, date journal.Date) (journal.Entry, error) {
	out, err := s.fakeStore.Get(ctx, date)
	s.once.Do(func() { close(s.started); <-s.release })
	return out, err
}
func TestSaveRejectsRacingOldPreviewFill(t *testing.T) {
	base := &delayedPreviewStore{fakeStore: newFakeStore(), started: make(chan struct{}), release: make(chan struct{})}
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	base.contents[d] = journal.Entry{Date: d, Text: "old"}
	store := New(newTestStorage(), time.Minute).Wrap("account", base)
	ctx := context.Background()
	done := make(chan error, 1)
	go func() { _, err := store.Previews(ctx, []journal.Date{d}, 44); done <- err }()
	<-base.started
	if _, err := store.Put(ctx, d, "new", "old"); err != nil {
		t.Fatal(err)
	}
	close(base.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got, err := store.Previews(ctx, []journal.Date{d}, 44)
	if err != nil || got[d] != "new" {
		t.Fatalf("stale fill survived: %v %v", got, err)
	}
}

func TestCreateAndTrashRefreshOnlyAffectedIndexes(t *testing.T) {
	ctx := context.Background()
	base := newFakeStore()
	cache := New(newTestStorage(), time.Minute)
	store := cache.Wrap("account", base)
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	base.years = []int{2019}
	base.entries[2019] = []journal.EntryMeta{{Date: journal.Date{Year: 2019, Month: 1, Day: 1}}}
	_, _ = store.ListYears(ctx)
	_, _ = store.ListYear(ctx, 2019)
	_, _ = store.ListYear(ctx, 2020)
	if _, err := store.Put(ctx, d, "created", ""); err != nil {
		t.Fatal(err)
	}
	base.years = []int{2020, 2019}
	base.entries[2020] = []journal.EntryMeta{{Date: d}}
	before := base.calls
	years, err := store.ListYears(ctx)
	if err != nil || len(years) != 2 {
		t.Fatalf("create years: %v %v", years, err)
	}
	entries, err := store.ListYear(ctx, 2020)
	if err != nil || len(entries) != 1 {
		t.Fatalf("create index: %v %v", entries, err)
	}
	_, _ = store.ListYear(ctx, 2019)
	if base.calls != before+2 {
		t.Fatal("unrelated year index evicted")
	}
	if err := store.Trash(ctx, d); err != nil {
		t.Fatal(err)
	}
	base.entries[2020] = nil
	entries, err = store.ListYear(ctx, 2020)
	if err != nil || len(entries) != 0 {
		t.Fatalf("trash index: %v %v", entries, err)
	}
}

type ambiguousWriteStore struct{ *fakeStore }

func (s *ambiguousWriteStore) Put(ctx context.Context, d journal.Date, text string, base journal.Version) (journal.Entry, error) {
	_, _ = s.fakeStore.Put(ctx, d, text, base)
	return journal.Entry{}, journal.ErrUnavailable
}
func TestAmbiguousSaveDiscardsAffectedPreviewWithoutRetry(t *testing.T) {
	base := &ambiguousWriteStore{newFakeStore()}
	ctx := context.Background()
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	base.contents[d] = journal.Entry{Date: d, Text: "old"}
	store := New(newTestStorage(), time.Minute).Wrap("account", base)
	_, _ = store.Previews(ctx, []journal.Date{d}, 44)
	before := base.calls
	if _, err := store.Put(ctx, d, "new", "old"); err != journal.ErrUnavailable {
		t.Fatal(err)
	}
	if base.calls != before+1 {
		t.Fatal("ambiguous write retried")
	}
	got, err := store.Previews(ctx, []journal.Date{d}, 44)
	if err != nil || got[d] != "new" {
		t.Fatalf("ambiguous outcome left stale cache: %v %v", got, err)
	}
}

func TestOffPageUpdateRetainsKnownYearListWithoutYearIndex(t *testing.T) {
	ctx := context.Background()
	base := newFakeStore()
	base.years = []int{2026, 2020}
	store := New(newTestStorage(), time.Minute).Wrap("account", base)
	_, _ = store.ListYears(ctx)
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	if _, err := store.Put(ctx, d, "new", "existing"); err != nil {
		t.Fatal(err)
	}
	before := base.calls
	_, _ = store.ListYears(ctx)
	if base.calls != before {
		t.Fatal("off-page update discarded known year list")
	}
}

func TestUpdateRefreshesYearListMissingExternallyAddedYear(t *testing.T) {
	ctx := context.Background()
	base := newFakeStore()
	base.years = []int{2026}
	store := New(newTestStorage(), time.Minute).Wrap("account", base)
	_, _ = store.ListYears(ctx)
	d := journal.Date{Year: 2020, Month: 1, Day: 1}
	base.entries[2020] = []journal.EntryMeta{{Date: d}}
	_, _ = store.ListYear(ctx, 2020)
	base.years = []int{2026, 2020}
	if _, err := store.Put(ctx, d, "new", "existing"); err != nil {
		t.Fatal(err)
	}
	years, err := store.ListYears(ctx)
	if err != nil || len(years) != 2 {
		t.Fatalf("externally added year stayed hidden: %v %v", years, err)
	}
}
