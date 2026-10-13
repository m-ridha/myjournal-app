package journal

import (
	"context"
	"sort"
	"testing"
	"time"

	"cloud.google.com/go/civil"

	"github/m-ridha/myjournal-app/internal/testutil"
)

type inMemStore struct {
	entries          map[Date]Entry
	listYearCalls    int
	getCalls         int
	previewCalls     int
	lastPreviewDates []Date
}

func newInMemStore() *inMemStore {
	return &inMemStore{entries: map[Date]Entry{}}
}

func (s *inMemStore) ListYears(ctx context.Context) ([]int, error) {
	yearsSet := make(map[int]struct{})
	for date := range s.entries {
		yearsSet[date.Year] = struct{}{}
	}
	years := make([]int, 0, len(yearsSet))
	for year := range yearsSet {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years, nil
}
func (s *inMemStore) ListYear(_ context.Context, year int) ([]EntryMeta, error) {
	s.listYearCalls++
	items := make([]EntryMeta, 0)
	for date := range s.entries {
		if date.Year == year {
			items = append(items, EntryMeta{Date: date})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Date.After(items[j].Date) })
	return items, nil
}
func (s *inMemStore) Get(ctx context.Context, d Date) (Entry, error) {
	s.getCalls++
	if e, ok := s.entries[d]; ok {
		return e, nil
	}
	return Entry{}, ErrNotFound
}
func (s *inMemStore) Previews(_ context.Context, dates []Date, maxBytes int) (map[Date]string, error) {
	s.previewCalls++
	s.lastPreviewDates = append([]Date(nil), dates...)
	out := make(map[Date]string, len(dates))
	for _, date := range dates {
		entry, ok := s.entries[date]
		if !ok {
			continue
		}
		out[date] = PreviewPrefix(entry.Text, maxBytes)
	}
	return out, nil
}
func (s *inMemStore) Put(ctx context.Context, d Date, text string, base Version) (Entry, error) {
	s.entries[d] = Entry{Date: d, Text: text, Version: "v1", Exists: true}
	return s.entries[d], nil
}
func (s *inMemStore) Trash(ctx context.Context, d Date) error { delete(s.entries, d); return nil }

func TestEntryPath(t *testing.T) {
	date := civil.Date{Year: 2026, Month: 9, Day: 21}
	got := EntryPath(date)
	want := "entries/2026/2026-09-21.txt"
	if got != want {
		t.Fatalf("EntryPath() = %q, want %q", got, want)
	}

	parsed, err := ParseEntryDate("entries/2026/2026-09-21.txt")
	if err != nil {
		t.Fatalf("ParseEntryDate() unexpected error: %v", err)
	}
	if parsed != date {
		t.Fatalf("ParseEntryDate() = %v, want %v", parsed, date)
	}
}

func TestServiceSaveRejectsFutureDate(t *testing.T) {
	store := newInMemStore()
	clk := testutil.StubClock{Time: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}
	svc := NewService(store, clk)

	future := civil.Date{Year: 2026, Month: 9, Day: 22}
	_, err := svc.Save(context.Background(), future, "hello", "")
	if err != ErrFutureDate {
		t.Fatalf("Save() error = %v, want %v", err, ErrFutureDate)
	}
}

func TestServiceSaveNormalizesText(t *testing.T) {
	store := newInMemStore()
	clk := testutil.StubClock{Time: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}
	svc := NewService(store, clk)

	date := civil.Date{Year: 2026, Month: 9, Day: 21}
	entry, err := svc.Save(context.Background(), date, "line1\r\nline2\r\n", "")
	if err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}
	if entry.Text != "line1\nline2\n" {
		t.Fatalf("Save() text = %q, want %q", entry.Text, "line1\nline2\n")
	}
}

func TestServiceListEntriesLoadsOnlyRequestedPreviews(t *testing.T) {
	store := newInMemStore()
	store.entries[Date{Year: 2026, Month: 9, Day: 21}] = Entry{Date: Date{Year: 2026, Month: 9, Day: 21}, Text: "newest", Exists: true}
	store.entries[Date{Year: 2026, Month: 9, Day: 20}] = Entry{Date: Date{Year: 2026, Month: 9, Day: 20}, Text: "next", Exists: true}
	store.entries[Date{Year: 2025, Month: 12, Day: 31}] = Entry{Date: Date{Year: 2025, Month: 12, Day: 31}, Text: "older", Exists: true}
	service := NewService(store, testutil.StubClock{Time: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)})

	page, err := service.ListEntries(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("ListEntries() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Date != (Date{Year: 2026, Month: 9, Day: 21}) || page.Items[0].Preview != "newest" || !page.HasNext {
		t.Fatalf("ListEntries() = %#v, want newest item with HasNext", page)
	}
	if store.previewCalls != 1 || len(store.lastPreviewDates) != 1 {
		t.Fatalf("preview calls/dates = %d/%d, want 1/1", store.previewCalls, len(store.lastPreviewDates))
	}
	if store.listYearCalls != 1 {
		t.Fatalf("ListYear() calls = %d, want 1 after finding page and next item", store.listYearCalls)
	}
}

func TestServiceMonthUsesYearIndexInsteadOfPerDayReads(t *testing.T) {
	store := newInMemStore()
	entryDate := Date{Year: 2026, Month: 9, Day: 21}
	store.entries[entryDate] = Entry{Date: entryDate, Text: "today", Exists: true}
	service := NewService(store, testutil.StubClock{Time: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)})

	month, err := service.Month(context.Background(), 2026, time.September)
	if err != nil {
		t.Fatalf("Month() error = %v", err)
	}
	if store.listYearCalls != 1 || store.getCalls != 0 {
		t.Fatalf("ListYear()/Get() calls = %d/%d, want 1/0", store.listYearCalls, store.getCalls)
	}
	for _, week := range month.Weeks {
		for _, day := range week {
			if day.Date == entryDate && (!day.HasEntry || !day.IsToday) {
				t.Fatalf("entry day = %#v, want HasEntry and IsToday", day)
			}
		}
	}
}

func TestPaginationRejectsUnsafeBounds(t *testing.T) {
	for _, pair := range [][2]int{{0, 20}, {-1, 20}, {MaxPage + 1, 20}, {int(^uint(0) >> 1), 100}, {1, 0}, {1, -1}, {1, MaxPageSize + 1}} {
		store := newInMemStore()
		_, err := NewService(store, testutil.StubClock{}).ListEntries(context.Background(), pair[0], pair[1])
		if err != ErrInvalidPagination {
			t.Fatalf("Page(%d,%d): %v", pair[0], pair[1], err)
		}
		if store.listYearCalls != 0 || store.previewCalls != 0 {
			t.Fatal("invalid request read storage")
		}
	}
}

func TestPaginationAcrossYearsAndEmptyPage(t *testing.T) {
	store := newInMemStore()
	for _, value := range []string{"2026-01-01", "2025-12-31", "2025-12-30", "2024-12-31"} {
		date, err := civil.ParseDate(value)
		if err != nil {
			t.Fatal(err)
		}
		store.entries[date] = Entry{Date: date, Text: value}
	}
	service := NewService(store, testutil.StubClock{})
	page, err := service.ListEntries(context.Background(), 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].Date.String() != "2025-12-30" || page.Items[1].Date.String() != "2024-12-31" || page.HasNext {
		t.Fatalf("page=%#v", page)
	}
	calls := store.previewCalls
	empty, err := service.ListEntries(context.Background(), MaxPage, MaxPageSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Items) != 0 || empty.HasNext || store.previewCalls != calls {
		t.Fatalf("empty=%#v", empty)
	}
}

func TestListEntriesTwentyStopsBeforeOlderYear(t *testing.T) {
	store := newInMemStore()
	for day := 1; day <= 21; day++ {
		date := Date{Year: 2026, Month: 1, Day: day}
		store.entries[date] = Entry{Date: date, Text: "synthetic", Exists: true}
	}
	older := Date{Year: 2025, Month: 12, Day: 31}
	store.entries[older] = Entry{Date: older, Text: "older", Exists: true}
	result, err := NewService(store, testutil.StubClock{}).ListEntries(context.Background(), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 20 || !result.HasNext || result.Items[0].Date.Day != 21 || result.Items[19].Date.Day != 2 {
		t.Fatalf("latest twenty: %+v", result)
	}
	if store.listYearCalls != 1 || len(store.lastPreviewDates) != 20 || store.getCalls != 0 {
		t.Fatalf("years=%d previews=%d downloads=%d", store.listYearCalls, len(store.lastPreviewDates), store.getCalls)
	}
}

type freshOpenStore struct {
	*inMemStore
	freshCalls int
}

var _ FreshEntryStore = (*freshOpenStore)(nil)

func (s *freshOpenStore) GetFresh(_ context.Context, d Date) (Entry, error) {
	s.freshCalls++
	return Entry{Date: d, Text: "fresh", Exists: true}, nil
}
func TestOpenFreshSelectsPortAndValidatesDate(t *testing.T) {
	ctx := context.Background()
	date := Date{Year: 2020, Month: 1, Day: 1}
	clock := testutil.StubClock{Time: time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)}
	store := &freshOpenStore{inMemStore: newInMemStore()}
	service := NewService(store, clock)
	entry, err := service.OpenFresh(ctx, date)
	if err != nil || entry.Text != "fresh" || store.freshCalls != 1 || store.getCalls != 0 {
		t.Fatalf("entry=%+v err=%v fresh=%d gets=%d", entry, err, store.freshCalls, store.getCalls)
	}
	if _, err := service.OpenFresh(ctx, Date{Year: 2020, Month: 1, Day: 2}); err == nil || store.freshCalls != 1 {
		t.Fatal("future date reached storage")
	}
	base := newInMemStore()
	base.entries[date] = Entry{Date: date, Text: "fallback"}
	entry, err = NewService(base, clock).OpenFresh(ctx, date)
	if err != nil || entry.Text != "fallback" || base.getCalls != 1 {
		t.Fatal("uncached provider fallback failed")
	}
}
