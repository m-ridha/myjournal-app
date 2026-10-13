package cachestore

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github/mridha/myjournal-app/internal/core/journal"
)

func TestMutationWarmsValidatedContentAndAbsence(t *testing.T) {
	for _, trash := range []bool{false, true} {
		name := "save"
		if trash {
			name = "trash"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
				defer stop()
				base := newFakeStore()
				date := journal.Date{Year: 2020, Month: 1, Day: 1}
				base.contents[date] = journal.Entry{Date: date, Text: "old", Version: "v1", Exists: true}
				request, cancel := context.WithCancel(context.Background())
				release := make(chan struct{})
				var factories atomic.Int32
				store := cache.WrapWithFactory("google:a", base, func(ctx context.Context) (journal.Store, error) {
					factories.Add(1)
					select {
					case <-release:
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					if ctx.Err() != nil {
						return nil, ctx.Err()
					}
					return base, nil
				})
				store = store.(journal.OperationStore).ForOperation()
				if trash {
					if err := store.Trash(request, date); err != nil {
						t.Fatal(err)
					}
				} else {
					if _, err := store.Put(request, date, "saved", "v1"); err != nil {
						t.Fatal(err)
					}
					// Acknowledgements must never seed the cache. Only the subsequent read does.
					base.mu.Lock()
					base.contents[date] = journal.Entry{Date: date, Text: "observed", Version: "v2", Exists: true}
					base.mu.Unlock()
				}
				cancel()
				synctest.Wait()
				if factories.Load() != 1 {
					t.Fatal("warming did not start independently")
				}
				close(release)
				synctest.Wait()
				base.mu.Lock()
				before := base.calls
				base.mu.Unlock()
				entry, err := store.Get(context.Background(), date)
				if trash {
					if !errors.Is(err, journal.ErrNotFound) {
						t.Fatal(err)
					}
				} else if err != nil || entry.Text != "observed" || entry.Version != "v2" {
					t.Fatalf("entry=%+v err=%v", entry, err)
				}
				base.mu.Lock()
				defer base.mu.Unlock()
				if base.calls != before {
					t.Fatal("next read missed warmed snapshot")
				}
			})
		})
	}
}

func TestWarmingCoalescesLaterMutationWithoutStaleFill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
		defer stop()
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		base := newFakeStore()
		release := make(chan struct{})
		var calls atomic.Int32
		loader := func(context.Context) (journal.Store, error) {
			return slowReadStore{Store: base, get: func(ctx context.Context, d journal.Date) (journal.Entry, error) {
				if calls.Add(1) == 1 {
					select {
					case <-release:
					case <-ctx.Done():
						return journal.Entry{}, ctx.Err()
					}
					return journal.Entry{Date: d, Text: "old", Version: "v1", Exists: true}, nil
				}
				return base.Get(ctx, d)
			}}, nil
		}
		store := cache.WrapWithFactory("google:a", base, loader)
		if _, err := store.Put(context.Background(), date, "old", ""); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if _, err := store.Put(context.Background(), date, "latest", "v1"); err != nil {
			t.Fatal(err)
		}
		close(release)
		synctest.Wait()
		entry, err := store.Get(context.Background(), date)
		if err != nil || entry.Text != "latest" || calls.Load() != 2 {
			t.Fatalf("entry=%+v calls=%d err=%v", entry, calls.Load(), err)
		}
	})
}

func TestWarmingFailureAndShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
		defer stop()
		base := newFakeStore()
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		var started, ended atomic.Int32
		store := cache.WrapWithFactory("google:a", base, func(ctx context.Context) (journal.Store, error) {
			started.Add(1)
			<-ctx.Done()
			ended.Add(1)
			return nil, ctx.Err()
		})
		if _, err := store.Put(context.Background(), date, "saved", ""); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		stop()
		if started.Load() != 1 || ended.Load() != 1 {
			t.Fatal("shutdown did not cancel and join warming")
		}
		if _, err := store.Put(context.Background(), date, "again", "v1"); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if started.Load() != 1 {
			t.Fatal("warming restarted after shutdown")
		}
		entry, err := store.Get(context.Background(), date)
		if err != nil || entry.Text != "again" {
			t.Fatal("warming failure changed saved content")
		}
	})
}

func TestWarmingAdmissionAndTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
		defer stop()
		base := newFakeStore()
		var active, peak, ended atomic.Int32
		store := cache.WrapWithFactory("google:a", base, func(ctx context.Context) (journal.Store, error) {
			n := active.Add(1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			<-ctx.Done()
			active.Add(-1)
			ended.Add(1)
			return nil, ctx.Err()
		})
		for i := 0; i < maxWarmDates+10; i++ {
			d := journal.Date{Year: 2000 + i, Month: 1, Day: 1}
			if _, err := store.Put(context.Background(), d, "saved", ""); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		cache.warmer.mu.Lock()
		pending := len(cache.warmer.tasks)
		cache.warmer.mu.Unlock()
		if pending != 64 || peak.Load() != 4 {
			t.Fatalf("pending=%d peak=%d", pending, peak.Load())
		}
		time.Sleep(10*time.Second + time.Millisecond)
		if ended.Load() < 4 || peak.Load() > 4 {
			t.Fatal("work did not time out within concurrency limit")
		}
		stop()
		if active.Load() != 0 || len(cache.warmer.tasks) != 0 || len(cache.warmer.jobs) != 0 {
			t.Fatal("shutdown retained active/queued work")
		}
	})
}

func TestFailedMutationDoesNotWarm(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
		defer stop()
		var factories atomic.Int32
		store := cache.WrapWithFactory("google:a", &ambiguousWriteStore{newFakeStore()}, func(context.Context) (journal.Store, error) {
			factories.Add(1)
			return nil, journal.ErrUnavailable
		})
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		if _, err := store.Put(context.Background(), date, "saved", ""); !errors.Is(err, journal.ErrUnavailable) {
			t.Fatal(err)
		}
		if err := store.Trash(context.Background(), journal.Date{Year: 2020, Month: 1, Day: 2}); !errors.Is(err, journal.ErrNotFound) {
			t.Fatal(err)
		}
		synctest.Wait()
		if factories.Load() != 0 {
			t.Fatal("failed mutation scheduled warming")
		}
	})
}

func TestForegroundReadSharesWarmingAndFreshReadStaysSeparate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
		defer stop()
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		base := newFakeStore()
		release := make(chan struct{})
		var reads atomic.Int32
		backing := slowReadStore{Store: base, get: func(ctx context.Context, d journal.Date) (journal.Entry, error) {
			if reads.Add(1) == 1 {
				select {
				case <-release:
				case <-ctx.Done():
					return journal.Entry{}, ctx.Err()
				}
			}
			return base.Get(ctx, d)
		}}
		store := cache.WrapWithFactory("google:a", backing, func(context.Context) (journal.Store, error) { return backing, nil })
		if _, err := store.Put(context.Background(), date, "saved", ""); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		done := make(chan error, 1)
		go func() { _, err := store.Get(context.Background(), date); done <- err }()
		synctest.Wait()
		if reads.Load() != 1 {
			t.Fatal("foreground read did not share warming")
		}
		if _, err := store.(journal.FreshEntryStore).GetFresh(context.Background(), date); err != nil {
			t.Fatal(err)
		}
		if reads.Load() != 2 {
			t.Fatal("fresh read joined warming")
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
	})
}

func TestMembershipMutationWarmsIndexes(t *testing.T) {
	for _, operation := range []string{"create", "delete", "update"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
				defer stop()
				base := newFakeStore()
				date := journal.Date{Year: 2020, Month: 1, Day: 1}
				base.years = []int{2020}
				base.entries[2020] = []journal.EntryMeta{{Date: date}}
				base.contents[date] = journal.Entry{Date: date, Text: "old", Version: "v1", Exists: true}
				store := cache.WrapWithFactory("google:a", base, func(context.Context) (journal.Store, error) { return base, nil })
				ctx := context.Background()
				if operation == "delete" {
					if err := store.Trash(ctx, date); err != nil {
						t.Fatal(err)
					}
				} else {
					version := journal.Version("")
					if operation == "update" {
						version = "v1"
					}
					if _, err := store.Put(ctx, date, "saved", version); err != nil {
						t.Fatal(err)
					}
				}
				synctest.Wait()
				base.mu.Lock()
				before := base.calls
				base.mu.Unlock()
				if _, err := store.ListYears(ctx); err != nil {
					t.Fatal(err)
				}
				if _, err := store.ListYear(ctx, 2020); err != nil {
					t.Fatal(err)
				}
				base.mu.Lock()
				defer base.mu.Unlock()
				want := before
				if operation == "update" {
					want += 2
				}
				if base.calls != want {
					t.Fatalf("backing calls = %d, want %d", base.calls, want)
				}
			})
		})
	}
}

type warmingIndexStore struct {
	journal.Store
	started chan struct{}
	release chan struct{}
	reads   atomic.Int32
}

func (s *warmingIndexStore) ListYears(ctx context.Context) ([]int, error) {
	if s.reads.Add(1) == 1 {
		close(s.started)
	}
	select {
	case <-s.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return s.Store.ListYears(ctx)
}

func TestForegroundIndexReadSharesWarming(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache, stop := NewWithWarming(newTestStorage(), 2*time.Minute)
		defer stop()
		backing := &warmingIndexStore{Store: newFakeStore(), started: make(chan struct{}), release: make(chan struct{})}
		store := cache.WrapWithFactory("google:a", backing, func(context.Context) (journal.Store, error) { return backing, nil })
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		if _, err := store.Put(context.Background(), date, "saved", ""); err != nil {
			t.Fatal(err)
		}
		<-backing.started
		done := make(chan error, 1)
		go func() { _, err := store.ListYears(context.Background()); done <- err }()
		synctest.Wait()
		if backing.reads.Load() != 1 {
			t.Fatal("foreground index read did not share warming")
		}
		close(backing.release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
	})
}
