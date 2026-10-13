package journal

import (
	"context"
	"errors"
	"testing"
	"time"
)

type coverageClock struct{ time.Time }

func (c coverageClock) Now() time.Time { return c.Time }

func TestDefaultTodayAndInvalidEntryOperations(t *testing.T) {
	store := newInMemStore()
	service := NewService(store, coverageClock{time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)})
	if got := service.Today(nil); got != (Date{Year: 2026, Month: 10, Day: 13}) {
		t.Fatal(got)
	}
	future := Date{Year: 2026, Month: 10, Day: 14}
	if _, err := service.Open(context.Background(), future); !errors.Is(err, ErrFutureDate) {
		t.Fatal(err)
	}
	if err := service.Delete(context.Background(), future); !errors.Is(err, ErrFutureDate) {
		t.Fatal(err)
	}
	if store.getCalls != 0 {
		t.Fatal("invalid date reached storage")
	}
}
func TestParseEntryPathRejectsInvalidDateAndYear(t *testing.T) {
	for _, path := range []string{"bad", "entries/2020/2021-01-01.txt", "entries/2020/2020-02-30.txt", "entries/0000/0000-01-01.txt", "entries/2020/2020-13-01.txt"} {
		if _, err := ParseEntryDate(path); err == nil {
			t.Fatalf("invalid path accepted: %s", path)
		}
	}
	got, err := ParseEntryDate("./entries/2020/2020-02-29.txt")
	if err != nil || got != (Date{Year: 2020, Month: 2, Day: 29}) {
		t.Fatalf("leap date: %v %v", got, err)
	}
}
