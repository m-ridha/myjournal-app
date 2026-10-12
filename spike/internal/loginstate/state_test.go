package loginstate

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExpiryAndConsumption(t *testing.T) {
	t.Parallel()
	var store Store
	now := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	if !store.Add("state", "verifier", now) {
		t.Fatal("Add on empty Store = false, want true")
	}
	if got, ok := store.Take("state", now.Add(time.Minute)); !ok || got != "verifier" {
		t.Fatalf("Take active state = %q, %t, want verifier, true", got, ok)
	}
	if _, ok := store.Take("state", now); ok {
		t.Fatal("Take consumed state = true, want false")
	}
	store.Add("expired", "verifier", now)
	if _, ok := store.Take("expired", now.Add(10*time.Minute)); ok {
		t.Fatal("Take at expiry = true, want false")
	}
}

func TestConcurrentCapacityAndExpiryReclaimsSpace(t *testing.T) {
	t.Parallel()
	var store Store
	now := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	var accepted atomic.Int32
	var workers sync.WaitGroup
	for i := range 256 {
		workers.Go(func() {
			if store.Add(fmt.Sprint(i), "verifier", now) {
				accepted.Add(1)
			}
		})
	}
	workers.Wait()
	if got := accepted.Load(); got != 128 {
		t.Fatalf("concurrent Add accepted = %d, want 128", got)
	}
	if store.Add("overflow", "verifier", now.Add(time.Minute)) {
		t.Fatal("Add at capacity = true, want false")
	}
	if !store.Add("replacement", "verifier", now.Add(10*time.Minute)) {
		t.Fatal("Add after expiry = false, want true")
	}
	if got := len(store.items); got != 1 {
		t.Errorf("retained transactions after expiry = %d, want 1", got)
	}
}
