package storetest

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/civil"

	"myjournal-app/internal/core/journal"
)

// Factory returns a fresh, empty Store for one contract test.
type Factory func(t *testing.T) journal.Store

// Run executes the shared Store behavior contract.
func Run(t *testing.T, newStore Factory) {
	t.Helper()
	ctx := context.Background()
	date := civil.Date{Year: 2026, Month: 9, Day: 21}

	t.Run("missing entry returns ErrNotFound", func(t *testing.T) {
		store := newStore(t)
		_, err := store.Get(ctx, date)
		if !errors.Is(err, journal.ErrNotFound) {
			t.Fatalf("Get() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("put then get preserves text and version", func(t *testing.T) {
		store := newStore(t)
		want := "hello journal"
		entry, err := store.Put(ctx, date, want, "")
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}
		if entry.Version == "" {
			t.Fatal("Put() returned an empty version")
		}
		got, err := store.Get(ctx, date)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Text != want || got.Version != entry.Version || !got.Exists {
			t.Fatalf("Get() = %#v, want text %q and version %q", got, want, entry.Version)
		}
	})

	t.Run("stale version returns ErrConflict", func(t *testing.T) {
		store := newStore(t)
		entry, err := store.Put(ctx, date, "first", "")
		if err != nil {
			t.Fatalf("initial Put() error = %v", err)
		}
		if _, err := store.Put(ctx, date, "second", entry.Version+"-stale"); !errors.Is(err, journal.ErrConflict) {
			t.Fatalf("stale Put() error = %v, want ErrConflict", err)
		}
	})

	t.Run("list years and entries", func(t *testing.T) {
		store := newStore(t)
		for _, item := range []struct {
			date civil.Date
			text string
		}{
			{date: date, text: "today"},
			{date: civil.Date{Year: 2025, Month: 12, Day: 31}, text: "last year"},
		} {
			if _, err := store.Put(ctx, item.date, item.text, ""); err != nil {
				t.Fatalf("Put(%v) error = %v", item.date, err)
			}
		}
		years, err := store.ListYears(ctx)
		if err != nil {
			t.Fatalf("ListYears() error = %v", err)
		}
		if len(years) != 2 || years[0] != 2026 || years[1] != 2025 {
			t.Fatalf("ListYears() = %v, want [2026 2025]", years)
		}
		items, err := store.ListYear(ctx, 2026)
		if err != nil || len(items) != 1 || items[0].Date != date {
			t.Fatalf("ListYear() = %#v, error %v", items, err)
		}
	})

	t.Run("previews honor the size cap", func(t *testing.T) {
		store := newStore(t)
		if _, err := store.Put(ctx, date, "123456789", ""); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
		previews, err := store.Previews(ctx, []civil.Date{date}, 4)
		if err != nil {
			t.Fatalf("Previews() error = %v", err)
		}
		if got := previews[date]; got != "1234" {
			t.Fatalf("preview = %q, want %q", got, "1234")
		}
	})

	t.Run("Unicode previews trim before truncation", func(t *testing.T) {
		store := newStore(t)
		if _, err := store.Put(ctx, date, "\n\u2003   中文🙂tail\nignored", ""); err != nil {
			t.Fatal(err)
		}
		for _, budget := range []int{-1, 0, 1, 7, 10} {
			previews, err := store.Previews(ctx, []civil.Date{date}, budget)
			if err != nil {
				t.Fatal(err)
			}
			want := journal.PreviewPrefix("中文🙂tail", budget)
			if previews[date] != want {
				t.Fatalf("budget=%d got=%q want=%q", budget, previews[date], want)
			}
		}
	})
	t.Run("invalid UTF-8 preserves entry", func(t *testing.T) {
		store := newStore(t)
		original, err := store.Put(ctx, date, "valid", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Put(ctx, date, string([]byte{0xff}), original.Version); !errors.Is(err, journal.ErrInvalidText) {
			t.Fatalf("error=%v", err)
		}
		current, err := store.Get(ctx, date)
		if err != nil || current.Text != "valid" || current.Version != original.Version {
			t.Fatalf("current=%+v err=%v", current, err)
		}
	})

	t.Run("trash removes the entry", func(t *testing.T) {
		store := newStore(t)
		if _, err := store.Put(ctx, date, "trash me", ""); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
		if err := store.Trash(ctx, date); err != nil {
			t.Fatalf("Trash() error = %v", err)
		}
		if _, err := store.Get(ctx, date); !errors.Is(err, journal.ErrNotFound) {
			t.Fatalf("Get() after Trash() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("put after deletion with a base version conflicts", func(t *testing.T) {
		store := newStore(t)
		entry, err := store.Put(ctx, date, "original", "")
		if err != nil {
			t.Fatalf("initial Put() error = %v", err)
		}
		if err := store.Trash(ctx, date); err != nil {
			t.Fatalf("Trash() error = %v", err)
		}
		if _, err := store.Put(ctx, date, "restored", entry.Version); !errors.Is(err, journal.ErrConflict) {
			t.Fatalf("Put() after deletion error = %v, want ErrConflict", err)
		}
	})
}
