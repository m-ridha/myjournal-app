package bigcache

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	cachestore "github/m-ridha/myjournal-app/internal/driven/cache/journal"
)

func TestBlockedStorageReadCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		storage, err := New(time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = storage.Close() }()
		for range 64 {
			storage.gate <- struct{}{}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { _, err := storage.Get(ctx, "synthetic"); done <- err }()
			synctest.Wait()
			<-storage.gate
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) && !errors.Is(err, cachestore.ErrMiss) {
				t.Fatal(err)
			}
		}
	})
}

func TestBlockedStorageReadCanceledBeforeRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		storage, err := New(time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = storage.Close() }()
		storage.gate <- struct{}{}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { _, err := storage.Get(ctx, "synthetic"); done <- err }()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		<-storage.gate
	})
}

func TestClosedAdapterRejectsReads(t *testing.T) {
	storage, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Get(context.Background(), "synthetic"); !errors.Is(err, errClosed) {
		t.Fatal("closed storage accepted read")
	}
}

// Cancellation follows an observed deletion, not an arbitrary number of Err calls.
// This models a request canceled during the bounded prefix scan deterministically.
type cancelAfterDeletion struct {
	context.Context
	cancel  context.CancelFunc
	storage *Adapter
	initial int
}

func (c cancelAfterDeletion) Err() error {
	if c.storage.cache.Len() < c.initial {
		c.cancel()
	}
	return c.Context.Err()
}
func TestPrefixDeletionCancellationAndRetry(t *testing.T) {
	storage, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = storage.Close() }()
	for _, key := range []string{"account/a", "account/b", "account/c", "other/d"} {
		if err := storage.Set(context.Background(), key, []byte("synthetic")); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := cancelAfterDeletion{Context: ctx, cancel: cancel, storage: storage, initial: 4}
	if err := storage.DeletePrefix(observed, "account/"); !errors.Is(err, context.Canceled) {
		t.Fatal("prefix deletion ignored cancellation")
	}
	if err := storage.DeletePrefix(context.Background(), "account/"); err != nil {
		t.Fatal(err)
	}
	if value, err := storage.Get(context.Background(), "other/d"); err != nil || string(value) != "synthetic" {
		t.Fatal("prefix scan deleted unrelated account")
	}
	for _, key := range []string{"account/a", "account/b", "account/c"} {
		if _, err := storage.Get(context.Background(), key); !errors.Is(err, cachestore.ErrMiss) {
			t.Fatal("retry retained matching key")
		}
	}
}
