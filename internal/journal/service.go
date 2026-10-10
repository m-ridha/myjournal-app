package journal

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"cloud.google.com/go/civil"
)

const (
	previewBytes = 40
	MaxPage      = 10000
	MaxPageSize  = 100
)

// Service contains the journal domain rules.
type Service struct {
	store Store
	clock Clock
}

func NewService(store Store, clock Clock) *Service {
	return &Service{store: store, clock: clock}
}

func (s *Service) Today(tz *time.Location) Date {
	if tz == nil {
		tz = time.UTC
	}
	return civil.DateOf(s.clock.Now().In(tz))
}

func normalizeText(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func (s *Service) Save(ctx context.Context, d Date, text string, base Version) (Entry, error) {
	if !d.IsValid() || d.Year < 1 || d.Year > 9999 {
		return Entry{}, ErrInvalidDate
	}
	if d.After(s.Today(TimeZone(ctx))) {
		return Entry{}, ErrFutureDate
	}
	if !utf8.ValidString(text) {
		return Entry{}, ErrInvalidText
	}
	clean := normalizeText(text)
	return s.store.Put(ctx, d, clean, base)
}

func (s *Service) Open(ctx context.Context, d Date) (Entry, error) {
	if !d.IsValid() || d.Year < 1 || d.Year > 9999 {
		return Entry{}, ErrInvalidDate
	}
	if d.After(s.Today(TimeZone(ctx))) {
		return Entry{}, ErrFutureDate
	}
	return s.store.Get(ctx, d)
}

func (s *Service) Page(ctx context.Context, page, size int) (ListPage, error) {
	if page < 1 || page > MaxPage || size < 1 || size > MaxPageSize {
		return ListPage{}, ErrInvalidPagination
	}
	years, err := s.store.ListYears(ctx)
	if err != nil {
		return ListPage{}, err
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	offset := (page - 1) * size
	// Retain only the requested page plus one item to detect a next page.
	pageDates := make([]Date, 0, size+1)
	for _, year := range years {
		meta, err := s.store.ListYear(ctx, year)
		if err != nil {
			return ListPage{}, err
		}
		sort.Slice(meta, func(i, j int) bool { return meta[i].Date.After(meta[j].Date) })
		for _, item := range meta {
			if offset > 0 {
				offset--
				continue
			}
			pageDates = append(pageDates, item.Date)
			if len(pageDates) == size+1 {
				break
			}
		}
		if len(pageDates) == size+1 {
			break
		}
	}
	hasNext := len(pageDates) > size
	if hasNext {
		pageDates = pageDates[:size]
	}
	if len(pageDates) == 0 {
		return ListPage{Page: page, PageSize: size}, nil
	}
	previews, err := s.store.Previews(ctx, pageDates, previewBytes+utf8.UTFMax)
	if err != nil {
		return ListPage{}, err
	}
	items := make([]ListItem, 0, len(pageDates))
	for _, date := range pageDates {
		items = append(items, ListItem{Date: date, Preview: PreviewText(previews[date], previewBytes)})
	}
	return ListPage{
		Items:    items,
		Page:     page,
		PageSize: size,
		HasNext:  hasNext,
	}, nil
}

func PreviewText(text string, max int) string {
	trimmed := strings.TrimLeftFunc(text, unicode.IsSpace)
	if trimmed == "" {
		return ""
	}
	line := strings.SplitN(trimmed, "\n", 2)[0]
	line = strings.SplitN(line, "\r", 2)[0]
	if max <= 0 || len(line) <= max {
		return line
	}
	return PreviewPrefix(line, max) + "..."
}

// PreviewPrefix returns the first meaningful line within a UTF-8 byte budget.
// A nonpositive budget produces an empty preview; it never splits a code point.
func PreviewPrefix(text string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	line := PreviewText(text, 0)
	if len(line) <= maxBytes {
		return line
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(line[end]) {
		end--
	}
	return line[:end]
}

func (s *Service) Month(ctx context.Context, year int, month time.Month) (MonthView, error) {
	if year < 1 || year > 9999 || month < time.January || month > time.December {
		return MonthView{}, ErrInvalidDate
	}
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)
	start := first.AddDate(0, 0, -((int(first.Weekday()) + 6) % 7))
	today := s.Today(TimeZone(ctx))
	meta, err := s.store.ListYear(ctx, year)
	if err != nil {
		return MonthView{}, err
	}
	entries := make(map[Date]struct{}, len(meta))
	for _, item := range meta {
		entries[item.Date] = struct{}{}
	}
	weeks := make([][]Day, 0, 6)
	for len(weeks) < 6 {
		week := make([]Day, 0, 7)
		for range 7 {
			date := civil.DateOf(start)
			_, hasEntry := entries[date]
			inMonth := date.Month == month
			week = append(week, Day{
				Date: date, InMonth: inMonth, HasEntry: inMonth && hasEntry,
				IsToday: date == today, IsFuture: date.After(today),
			})
			start = start.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
		if start.After(last) && start.Weekday() == time.Monday {
			break
		}
	}
	return MonthView{Year: year, Month: month, Weeks: weeks}, nil
}

func (s *Service) Delete(ctx context.Context, d Date) error {
	if !d.IsValid() || d.Year < 1 || d.Year > 9999 {
		return ErrInvalidDate
	}
	if d.After(s.Today(TimeZone(ctx))) {
		return ErrFutureDate
	}
	return s.store.Trash(ctx, d)
}
