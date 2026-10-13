package cachestore

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

func TestWaitingCacheLockCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := New(newTestStorage(), time.Minute)
		for range 64 {
			cache.gate <- struct{}{}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan bool, 1)
			go func() { done <- cache.lock(ctx) }()
			synctest.Wait()
			cancel()
			<-cache.gate
			if <-done {
				cache.unlock()
				t.Fatal("canceled waiter acquired cache lock")
			}
		}
	})
}
func TestSnapshotEncodingFailureAndUnknownKind(t *testing.T) {
	if _, err := encodeSnapshot("entry", make(chan int), time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("unserializable cache value accepted")
	}
	if got := (cacheKey{account: "synthetic", kind: "unknown"}).string(); got != namespace("synthetic")+"unknown" {
		t.Fatal("unknown key changed account namespace")
	}
	store := New(newTestStorage(), time.Minute).Wrap("", newFakeStore())
	if _, err := store.Previews(context.Background(), nil, 40); err != nil {
		t.Fatal(err)
	}
	store = New(newTestStorage(), time.Minute).Wrap("synthetic", newFakeStore())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Previews(ctx, []journal.Date{{Year: 2020, Month: 1, Day: 1}}, 40); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestCacheLockCancellationAtHandoffReleasesAdmission(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := New(newTestStorage(), time.Minute)
		for range 64 {
			cache.gate <- struct{}{}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan bool, 1)
			go func() { done <- cache.lock(ctx) }()
			synctest.Wait()
			<-cache.gate
			cancel()
			if <-done {
				cache.unlock()
			} // Acquisition may linearize before cancellation.
			if !cache.lock(context.Background()) {
				t.Fatal("canceled handoff leaked admission")
			}
			cache.unlock()
		}
	})
}
