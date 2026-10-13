package journal

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/civil"

	"github/m-ridha/myjournal-app/internal/testutil"
)

func TestUserDateBoundaries(t *testing.T) {
	cases := []struct{ zone, instant, today string }{
		{"Asia/Jakarta", "2026-10-07T19:00:00Z", "2026-10-08"},
		{"America/Los_Angeles", "2026-10-08T01:00:00Z", "2026-10-07"},
		{"Pacific/Kiritimati", "2026-10-07T12:00:00Z", "2026-10-08"},
		{"Pacific/Honolulu", "2026-10-08T01:00:00Z", "2026-10-07"},
		{"America/New_York", "2026-03-08T06:59:00Z", "2026-03-08"},
		{"America/New_York", "2026-03-08T07:01:00Z", "2026-03-08"},
		{"America/New_York", "2026-11-01T05:30:00Z", "2026-11-01"},
		{"America/New_York", "2026-11-01T06:30:00Z", "2026-11-01"},
	}
	for _, tc := range cases {
		t.Run(tc.zone+tc.instant, func(t *testing.T) {
			zone, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			now, err := time.Parse(time.RFC3339, tc.instant)
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithTimeZone(context.Background(), zone)
			store := newInMemStore()
			service := NewService(store, testutil.StubClock{Time: now})
			today, err := civil.ParseDate(tc.today)
			if err != nil {
				t.Fatal(err)
			}
			if got := service.Today(TimeZone(ctx)); got != today {
				t.Fatalf("today=%v want %v", got, today)
			}
			if _, err := service.Save(ctx, today, "text", ""); err != nil {
				t.Fatal(err)
			}
			future := today.AddDays(1)
			store.entries[future] = Entry{Date: future, Text: "untouched"}
			if _, err := service.Save(ctx, future, "changed", ""); !errors.Is(err, ErrFutureDate) {
				t.Fatalf("Save future: %v", err)
			}
			if _, err := service.Open(ctx, future); !errors.Is(err, ErrFutureDate) {
				t.Fatalf("Open future: %v", err)
			}
			if err := service.Delete(ctx, future); !errors.Is(err, ErrFutureDate) {
				t.Fatalf("Delete future: %v", err)
			}
			if store.getCalls != 0 || store.entries[future].Text != "untouched" {
				t.Fatal("future operations touched storage")
			}
			month, err := service.Month(ctx, today.Year, today.Month)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, week := range month.Weeks {
				for _, day := range week {
					if day.Date == today {
						found = true
						if !day.IsToday || day.IsFuture {
							t.Fatalf("today flags: %#v", day)
						}
					}
					if day.Date == future && !day.IsFuture {
						t.Fatalf("future flags: %#v", day)
					}
				}
			}
			if !found {
				t.Fatal("today absent from month")
			}
		})
	}
}

func TestSelectedDateSurvivesMidnight(t *testing.T) {
	zone, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithTimeZone(context.Background(), zone)
	clock := &testutil.StubClock{Time: time.Date(2026, 10, 8, 23, 59, 0, 0, zone)}
	store := newInMemStore()
	service := NewService(store, clock)
	selected := service.Today(zone)
	clock.Time = clock.Time.Add(2 * time.Minute)
	entry, err := service.Save(ctx, selected, "late save", "")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Date != selected || len(store.entries) != 1 {
		t.Fatal("save moved selected date")
	}
}

func TestMissingTimeZoneUsesUTC(t *testing.T) {
	service := NewService(newInMemStore(), testutil.StubClock{Time: time.Date(2026, 10, 8, 1, 0, 0, 0, time.FixedZone("host", 7*3600))})
	for _, ctx := range []context.Context{context.Background(), WithTimeZone(context.Background(), nil)} {
		if got := service.Today(TimeZone(ctx)).String(); got != "2026-10-07" {
			t.Fatalf("fallback today=%s", got)
		}
	}
}
