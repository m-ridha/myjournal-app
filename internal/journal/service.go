package journal

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/civil"
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
	return civil.DateOf(s.clock.Now().In(tz))
}

func normalizeText(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func (s *Service) Save(ctx context.Context, d Date, text string, base Version) (Entry, error) {
	if d.After(civil.DateOf(s.clock.Now())) {
		return Entry{}, ErrFutureDate
	}
	clean := normalizeText(text)
	return s.store.Put(ctx, d, clean, base)
}

func (s *Service) Open(ctx context.Context, d Date) (Entry, error) {
	return s.store.Get(ctx, d)
}

func (s *Service) Page(ctx context.Context, page, size int) (ListPage, error) {
	page = max(1, page)
	size = max(1, size)

	years, err := s.store.ListYears(ctx)
	if err != nil {
		return ListPage{}, err
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	offset := (page - 1) * size
	limit := offset + size + 1
	dates := make([]Date, 0, limit)
	for _, year := range years {
		meta, err := s.store.ListYear(ctx, year)
		if err != nil {
			return ListPage{}, err
		}
		sort.Slice(meta, func(i, j int) bool { return meta[i].Date.After(meta[j].Date) })
		for _, item := range meta {
			dates = append(dates, item.Date)
			if len(dates) == limit {
				break
			}
		}
		if len(dates) == limit {
			break
		}
	}

	if offset >= len(dates) {
		return ListPage{Page: page, PageSize: size}, nil
	}
	end := min(offset+size, len(dates))
	pageDates := dates[offset:end]
	previews, err := s.store.Previews(ctx, pageDates, 40)
	if err != nil {
		return ListPage{}, err
	}
	items := make([]ListItem, 0, len(pageDates))
	for _, date := range pageDates {
		items = append(items, ListItem{Date: date, Preview: PreviewText(previews[date], 40)})
	}
	return ListPage{
		Items:    items,
		Page:     page,
		PageSize: size,
		HasNext:  len(dates) > offset+size,
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
	return line[:max] + "..."
}

func (s *Service) Month(ctx context.Context, year int, month time.Month) (MonthView, error) {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)
	start := first.AddDate(0, 0, -((int(first.Weekday()) + 6) % 7))
	today := civil.DateOf(s.clock.Now())
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
	return s.store.Trash(ctx, d)
}
