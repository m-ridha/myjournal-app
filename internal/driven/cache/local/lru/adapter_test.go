package lru

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/driven/cache/journal"
)

func TestStorageOwnershipAndPrefixDeletion(t *testing.T) {
	a := newTestAdapter(t, time.Minute, 0)
	ctx := context.Background()
	if _, err := a.Get(ctx, "absent"); !errors.Is(err, cachestore.ErrMiss) {
		t.Fatalf("miss: %v", err)
	}
	if err := a.Delete(ctx, "absent"); err != nil {
		t.Fatal(err)
	}
	input := []byte("value")
	if err := a.Set(ctx, "a/date/4", input); err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'
	got, err := a.Get(ctx, "a/date/4")
	if err != nil || string(got) != "value" {
		t.Fatalf("input alias: %q %v", got, err)
	}
	got[0] = 'X'
	got, err = a.Get(ctx, "a/date/4")
	if err != nil || string(got) != "value" {
		t.Fatalf("output alias: %q %v", got, err)
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
	if err := a.Delete(ctx, "a/other/4"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get(ctx, "a/other/4"); !errors.Is(err, cachestore.ErrMiss) {
		t.Fatal("Delete retained key")
	}
}

func TestLRUEvictionAndUpdates(t *testing.T) {
	a := newTestAdapter(t, time.Minute, 2)
	ctx := context.Background()
	for _, key := range []string{"a", "b"} {
		if err := a.Set(ctx, key, []byte(key)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.Get(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if err := a.Set(ctx, "c", []byte("c")); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get(ctx, "b"); !errors.Is(err, cachestore.ErrMiss) {
		t.Fatal("did not evict least recently read entry")
	}
	if err := a.Set(ctx, "a", []byte("updated")); err != nil {
		t.Fatal(err)
	}
	if err := a.Set(ctx, "d", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get(ctx, "c"); !errors.Is(err, cachestore.ErrMiss) {
		t.Fatal("update did not promote entry")
	}
	got, err := a.Get(ctx, "a")
	if err != nil || string(got) != "updated" {
		t.Fatalf("update: %q %v", got, err)
	}
	if len(a.items) != 2 {
		t.Fatalf("capacity: %d", len(a.items))
	}
}

func TestExpiryAndReplacementTTL(t *testing.T) {
	a := newTestAdapter(t, time.Minute, 0)
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }
	ctx := context.Background()
	if err := a.Set(ctx, "a", []byte("old")); err != nil {
		t.Fatal(err)
	}
	now = now.Add(50 * time.Second)
	if _, err := a.Get(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(10 * time.Second)
	if _, err := a.Get(ctx, "a"); !errors.Is(err, cachestore.ErrMiss) {
		t.Fatal("read extended expiry or exact expiry was accepted")
	}
	if err := a.Set(ctx, "a", []byte("new")); err != nil {
		t.Fatal(err)
	}
	now = now.Add(50 * time.Second)
	if err := a.Set(ctx, "a", []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	now = now.Add(11 * time.Second)
	got, err := a.Get(ctx, "a")
	if err != nil || string(got) != "replacement" {
		t.Fatalf("replacement TTL: %q %v", got, err)
	}
}

func TestDefaultCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		a := newTestAdapter(t, time.Minute, capacity)
		if a.capacity != 2048 {
			t.Fatalf("default capacity: %d", a.capacity)
		}
	}
}

func TestCancellation(t *testing.T) {
	a := newTestAdapter(t, time.Minute, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, op := range []func() error{
		func() error { _, err := a.Get(ctx, "k"); return err }, func() error { return a.Set(ctx, "k", nil) },
		func() error { return a.Delete(ctx, "k") }, func() error { return a.DeletePrefix(ctx, "") },
	} {
		if err := op(); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	}
}

func TestLockWaitCancellation(t *testing.T) {
	a := newTestAdapter(t, time.Minute, 0)
	a.gate <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := a.Get(ctx, "k"); done <- err }()
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

func TestConcurrentStorage(t *testing.T) {
	a := newTestAdapter(t, time.Minute, 512)
	done := make(chan error, 8)
	for worker := range 8 {
		go func() {
			ctx := context.Background()
			for iteration := range 20 {
				key := fmt.Sprintf("worker-%d/%d", worker, iteration)
				if err := a.Set(ctx, key, []byte(key)); err != nil {
					done <- err
					return
				}
				got, err := a.Get(ctx, key)
				if err != nil || string(got) != key {
					done <- fmt.Errorf("Get(%s) = %q, %v", key, got, err)
					return
				}
				if err := a.DeletePrefix(ctx, fmt.Sprintf("unused-%d/", worker)); err != nil {
					done <- err
					return
				}
				if err := a.Delete(ctx, key); err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}()
	}
	for range 8 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

func TestAdapterCloseAndConstructorCompatibility(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		if a, err := New(ttl); err == nil || a != nil {
			t.Fatalf("invalid TTL accepted: %v", ttl)
		}
		if a, err := NewWithCapacity(ttl, 2); err == nil || a != nil {
			t.Fatalf("capacity constructor accepted invalid TTL: %v", ttl)
		}
	}
	a, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := a.Set(ctx, "key", []byte("value")); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if len(a.items) != 0 || a.lru.Len() != 0 {
		t.Fatal("Close retained stored values")
	}
	for _, op := range []func() error{
		func() error { _, err := a.Get(ctx, "key"); return err },
		func() error { return a.Set(ctx, "key", []byte("new")) },
		func() error { return a.Delete(ctx, "key") },
		func() error { return a.DeletePrefix(ctx, "") },
	} {
		if err := op(); !errors.Is(err, errClosed) {
			t.Fatalf("operation after Close: %v", err)
		}
	}
}

func newTestAdapter(t *testing.T, ttl time.Duration, capacity int) *Adapter {
	t.Helper()
	a, err := NewWithCapacity(ttl, capacity)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	return a
}

func TestConcurrentCloseAndOperations(t *testing.T) {
	a := newTestAdapter(t, time.Minute, 0)
	start := make(chan struct{})
	done := make(chan error, 16)
	for worker := range 8 {
		go func() {
			<-start
			ctx := context.Background()
			key := fmt.Sprintf("worker-%d", worker)
			for range 20 {
				if err := a.Set(ctx, key, []byte("value")); err != nil && !errors.Is(err, errClosed) {
					done <- err
					return
				}
				if _, err := a.Get(ctx, key); err != nil && !errors.Is(err, errClosed) {
					done <- err
					return
				}
				if err := a.Delete(ctx, key); err != nil && !errors.Is(err, errClosed) {
					done <- err
					return
				}
			}
			done <- nil
		}()
		go func() { <-start; done <- a.Close() }()
	}
	close(start)
	for range 16 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
