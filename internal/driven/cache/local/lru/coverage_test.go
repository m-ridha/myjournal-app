package lru

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	cachestore "myjournal-app/internal/driven/cache/cachestore"
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
