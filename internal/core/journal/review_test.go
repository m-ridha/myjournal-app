package journal

import (
	"context"
	"errors"
	"testing"
	"time"

	"myjournal-app/internal/testutil"
)

func TestServiceRejectsInvalidDates(t *testing.T) {
	s := NewService(newInMemStore(), testutil.StubClock{Time: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)})
	for _, d := range []Date{{}, {Year: 2020, Month: 13, Day: 1}, {Year: 2020, Month: 2, Day: 30}} {
		if _, err := s.Save(context.Background(), d, "test", ""); !errors.Is(err, ErrInvalidDate) {
			t.Errorf("Save(%v): %v", d, err)
		}
		if _, err := s.Open(context.Background(), d); !errors.Is(err, ErrInvalidDate) {
			t.Errorf("Open(%v): %v", d, err)
		}
		if err := s.Delete(context.Background(), d); !errors.Is(err, ErrInvalidDate) {
			t.Errorf("Delete(%v): %v", d, err)
		}
	}
	for _, tc := range []struct {
		year  int
		month time.Month
	}{{0, 1}, {10000, 1}, {2026, 0}, {2026, 13}} {
		if _, err := s.Month(context.Background(), tc.year, tc.month); !errors.Is(err, ErrInvalidDate) {
			t.Errorf("Month(%v): %v", tc, err)
		}
	}
}

func TestEntryPathEarlyYearRoundTrip(t *testing.T) {
	for _, year := range []int{1, 123, 999, 1000, 9999} {
		d := Date{Year: year, Month: 1, Day: 1}
		got, err := ParseEntryDate(EntryPath(d))
		if err != nil || got != d {
			t.Errorf("year %d: got %v, %v", year, got, err)
		}
	}
}
