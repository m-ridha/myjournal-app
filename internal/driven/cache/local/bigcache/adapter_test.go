package bigcache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	cachestore "myjournal-app/internal/driven/cache/cachestore"
)

func TestAdapterStorage(t *testing.T) {
	a, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	if _, err := a.Get(ctx, "missing"); !errors.Is(err, cachestore.ErrMiss) {
		t.Fatalf("missing: %v", err)
	}
	if err := a.Delete(ctx, "missing"); err != nil {
		t.Fatal(err)
	}
	input := []byte("value")
	if err := a.Set(ctx, "a/date/4", input); err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'
	got, err := a.Get(ctx, "a/date/4")
	if err != nil || string(got) != "value" {
		t.Fatalf("stored: %q %v", got, err)
	}
	got[0] = 'X'
	got, err = a.Get(ctx, "a/date/4")
	if err != nil || string(got) != "value" {
		t.Fatalf("copied: %q %v", got, err)
	}
	for _, key := range []string{"a/date/8", "a/other/4", "ab/date/4"} {
		if err := a.Set(ctx, key, []byte("keep")); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.DeletePrefix(ctx, "a/date/"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"a/date/4", "a/date/8"} {
		if _, err := a.Get(ctx, key); !errors.Is(err, cachestore.ErrMiss) {
			t.Fatalf("not deleted %s: %v", key, err)
		}
	}
	for _, key := range []string{"a/other/4", "ab/date/4"} {
		if _, err := a.Get(ctx, key); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Set(ctx, "too-large", []byte(strings.Repeat("x", 2<<20))); !errors.Is(err, cachestore.ErrCapacity) {
		t.Fatalf("capacity: %v", err)
	}
	if _, err := a.Get(ctx, "a/other/4"); err != nil {
		t.Fatalf("oversized write evicted unrelated data: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterCancellation(t *testing.T) {
	a, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, op := range []func() error{
		func() error { _, err := a.Get(ctx, "k"); return err },
		func() error { return a.Set(ctx, "k", nil) },
		func() error { return a.Delete(ctx, "k") },
		func() error { return a.DeletePrefix(ctx, "") },
	} {
		if err := op(); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	}
}

func TestAdapterClassifiesQueueCapacityFailure(t *testing.T) {
	a, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	key := "near-shard-limit"
	value := make([]byte, maxShardBytes-entryOverhead-len(key))
	// The adapter's size check permits this value, but BigCache's initial
	// queue cannot grow by this amount under its hard maximum.
	if err := a.Set(context.Background(), key, value); !errors.Is(err, cachestore.ErrCapacity) {
		t.Fatalf("queue capacity: %v", err)
	}
	if err := a.Set(context.Background(), "small", []byte("ok")); err != nil {
		t.Fatalf("subsequent write: %v", err)
	}
}

func TestAdapterRejectsDisabledTTL(t *testing.T) {
	if _, err := New(0); err == nil {
		t.Fatal("accepted disabled TTL")
	}
}

func TestAdapterEvictsWithinQueueLimit(t *testing.T) {
	a, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	value := make([]byte, 256<<10)
	// More than the configured queue budget must be evicted, not retained forever.
	for i := range 160 {
		if err := a.Set(ctx, fmt.Sprintf("entry-%d", i), value); err != nil {
			t.Fatal(err)
		}
	}
	misses := 0
	for i := range 160 {
		_, err := a.Get(ctx, fmt.Sprintf("entry-%d", i))
		if errors.Is(err, cachestore.ErrMiss) {
			misses++
		} else if err != nil {
			t.Fatal(err)
		}
	}
	if misses == 0 {
		t.Fatal("cache retained more than the queue budget")
	}
	if a.cache.Capacity() > 16<<20 {
		t.Fatalf("byte queue capacity exceeds 16 MiB: %d", a.cache.Capacity())
	}
}

func TestAdapterLockWaitIsCancelable(t *testing.T) {
	a, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	a.gate <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := a.Get(ctx, "key"); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		a.unlock()
		t.Fatal("lock wait ignored cancellation")
	}
	a.unlock()
}
