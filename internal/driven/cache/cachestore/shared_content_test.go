package cachestore

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

func TestBidirectionalEntryPreviewCache(t *testing.T) {
	for _, listFirst := range []bool{false, true} {
		name := "detail first"
		if listFirst {
			name = "list first"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			date := journal.Date{Year: 2020, Month: 1, Day: 1}
			base := newFakeStore()
			text := "\n\u2003 meaningful 🙂 first line\nfull second line"
			base.contents[date] = journal.Entry{Date: date, Text: text, Version: "v1", Exists: true}
			cache := New(newTestStorage(), 2*time.Minute)
			first, second := cache.Wrap("google:account", base), cache.Wrap("google:account", base)
			if listFirst {
				if _, err := first.Previews(ctx, []journal.Date{date}, 10); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := first.Get(ctx, date); err != nil {
					t.Fatal(err)
				}
			}
			entry, err := second.Get(ctx, date)
			if err != nil || entry.Text != text || entry.Version != "v1" {
				t.Fatalf("entry=%+v err=%v", entry, err)
			}
			for _, budget := range []int{-1, 0, 1, 10, 100} {
				previews, err := second.Previews(ctx, []journal.Date{date, date}, budget)
				if err != nil || len(previews) != 1 || previews[date] != journal.PreviewPrefix(text, budget) {
					t.Fatalf("budget=%d previews=%v err=%v", budget, previews, err)
				}
			}
			if base.calls != 1 {
				t.Fatalf("backing reads=%d want 1 across both directions/budgets", base.calls)
			}
		})
	}
}

func TestListDetailAndFreshFlights(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		var calls atomic.Int32
		release := make(chan struct{})
		base := slowReadStore{get: func(ctx context.Context, d journal.Date) (journal.Entry, error) {
			n := calls.Add(1)
			if n == 1 {
				select {
				case <-release:
				case <-ctx.Done():
					return journal.Entry{}, ctx.Err()
				}
			}
			return journal.Entry{Date: d, Text: "synthetic", Version: "v1", Exists: true}, nil
		}}
		cache := New(newTestStorage(), 2*time.Minute)
		store := cache.Wrap("google:a", base)
		done := make(chan error, 2)
		go func() { _, err := store.Previews(ctx, []journal.Date{date}, 4); done <- err }()
		synctest.Wait()
		go func() { _, err := store.Get(ctx, date); done <- err }()
		synctest.Wait()
		if calls.Load() != 1 {
			t.Fatalf("ordinary calls=%d", calls.Load())
		}
		if _, err := store.(journal.FreshEntryStore).GetFresh(ctx, date); err != nil {
			t.Fatal(err)
		}
		if calls.Load() != 2 {
			t.Fatal("fresh read joined ordinary flight")
		}
		close(release)
		for range 2 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestMissingPreviewSharesAbsenceAndMutationInvalidation(t *testing.T) {
	ctx := context.Background()
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	base := newFakeStore()
	store := New(newTestStorage(), 2*time.Minute).Wrap("google:a", base)
	previews, err := store.Previews(ctx, []journal.Date{date}, 4)
	if err != nil || len(previews) != 0 {
		t.Fatalf("previews=%v err=%v", previews, err)
	}
	if _, err := store.Get(ctx, date); !errors.Is(err, journal.ErrNotFound) || base.calls != 1 {
		t.Fatal("absence was not shared")
	}
	if _, err := store.Put(ctx, date, "created", ""); err != nil {
		t.Fatal(err)
	}
	previews, err = store.Previews(ctx, []journal.Date{date}, 4)
	if err != nil || previews[date] != "crea" {
		t.Fatalf("post-create=%v err=%v", previews, err)
	}
	if _, err := store.Put(ctx, date, "updated", "v1"); err != nil {
		t.Fatal(err)
	}
	previews, err = store.Previews(ctx, []journal.Date{date}, 4)
	if err != nil || previews[date] != "upda" {
		t.Fatal("update left stale preview")
	}
	if err := store.Trash(ctx, date); err != nil {
		t.Fatal(err)
	}
	previews, err = store.Previews(ctx, []journal.Date{date}, 4)
	if err != nil || len(previews) != 0 {
		t.Fatal("trash left stale preview")
	}
}
