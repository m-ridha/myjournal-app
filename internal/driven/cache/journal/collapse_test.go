package cachestore

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"cloud.google.com/go/civil"

	"github/mridha/myjournal-app/internal/core/journal"
)

type slowReadStore struct {
	journal.Store
	get     func(context.Context, civil.Date) (journal.Entry, error)
	preview func(context.Context, []civil.Date, int) (map[civil.Date]string, error)
}

func (s slowReadStore) Get(ctx context.Context, d civil.Date) (journal.Entry, error) {
	return s.get(ctx, d)
}
func (s slowReadStore) Previews(ctx context.Context, d []civil.Date, n int) (map[civil.Date]string, error) {
	return s.preview(ctx, d, n)
}

func TestOverlappingGetsAreSharedButNeverCached(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		release := make(chan struct{})
		base := slowReadStore{get: func(ctx context.Context, d civil.Date) (journal.Entry, error) {
			calls.Add(1)
			select {
			case <-release:
			case <-ctx.Done():
				return journal.Entry{}, ctx.Err()
			}
			return journal.Entry{Date: d, Text: "synthetic"}, nil
		}}
		cache := New(nil, 0)
		done := make(chan error, 8)
		for range 8 {
			go func() {
				_, err := cache.Wrap("account", base).Get(context.Background(), civil.Date{Year: 2020, Month: 1, Day: 1})
				done <- err
			}()
		}
		synctest.Wait()
		if calls.Load() != 1 {
			t.Fatalf("calls=%d, want 1", calls.Load())
		}
		close(release)
		for range 8 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
		_, err := cache.Wrap("account", base).Get(context.Background(), civil.Date{Year: 2020, Month: 1, Day: 1})
		if err != nil || calls.Load() != 2 {
			t.Fatalf("sequential read: calls=%d err=%v", calls.Load(), err)
		}
	})
}
func TestReadCancellationUsesSurvivingCaller(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := New(nil, 0)
		owner, cancel := context.WithCancel(context.Background())
		first := make(chan error, 1)
		go func() {
			_, err := cache.Wrap("account", slowReadStore{get: func(ctx context.Context, _ civil.Date) (journal.Entry, error) {
				<-ctx.Done()
				return journal.Entry{}, ctx.Err()
			}}).Get(owner, civil.Date{})
			first <- err
		}()
		synctest.Wait()
		second := make(chan error, 1)
		var calls atomic.Int32
		go func() {
			entry, err := cache.Wrap("account", slowReadStore{get: func(context.Context, civil.Date) (journal.Entry, error) {
				calls.Add(1)
				return journal.Entry{Text: "own store"}, nil
			}}).Get(context.Background(), civil.Date{})
			if err == nil && entry.Text != "own store" {
				err = errors.New("wrong result")
			}
			second <- err
		}()
		synctest.Wait()
		cancel()
		if !errors.Is(<-first, context.Canceled) {
			t.Fatal("owner must cancel")
		}
		if err := <-second; err != nil {
			t.Fatal(err)
		}
		if calls.Load() != 1 {
			t.Fatal("survivor must retry its own Store")
		}
	})
}
func TestReadScopesAndMutationGeneration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := New(nil, 0)
		release := make(chan struct{})
		done := make(chan error, 4)
		var calls atomic.Int32
		base := slowReadStore{Store: newFakeStore(), get: func(ctx context.Context, _ civil.Date) (journal.Entry, error) {
			calls.Add(1)
			select {
			case <-release:
			case <-ctx.Done():
				return journal.Entry{}, ctx.Err()
			}
			return journal.Entry{}, nil
		}}
		start := func(account string) {
			go func() {
				_, err := cache.Wrap(account, base).Get(context.Background(), civil.Date{})
				done <- err
			}()
		}
		start("a")
		start("b")
		start("a")
		synctest.Wait()
		if calls.Load() != 2 {
			t.Fatalf("isolated calls=%d", calls.Load())
		}
		// Even a failed mutation must separate future reads from older flights.
		_ = cache.Wrap("a", base).Trash(context.Background(), civil.Date{Year: 2020, Month: 1, Day: 1})
		start("a")
		synctest.Wait()
		if calls.Load() != 3 {
			t.Fatalf("post-mutation calls=%d", calls.Load())
		}
		close(release)
		for range 4 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
	})
}
func TestOverlappingPreviewDatesAndBudgets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		release := make(chan struct{})
		cache := New(nil, 0)
		base := slowReadStore{preview: func(ctx context.Context, dates []civil.Date, n int) (map[civil.Date]string, error) {
			calls.Add(1)
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			out := map[civil.Date]string{}
			for _, d := range dates {
				out[d] = journal.PreviewPrefix("synthetic", n)
			}
			return out, nil
		}}
		d1 := civil.Date{Year: 2020, Month: 1, Day: 1}
		d2 := civil.Date{Year: 2020, Month: 1, Day: 2}
		d3 := civil.Date{Year: 2020, Month: 1, Day: 3}
		done := make(chan error, 3)
		for _, request := range []struct {
			dates []civil.Date
			n     int
		}{{[]civil.Date{d1, d2}, 4}, {[]civil.Date{d2, d3}, 4}, {[]civil.Date{d2}, 8}} {
			go func() {
				out, err := cache.Wrap("a", base).Previews(context.Background(), request.dates, request.n)
				if err == nil && len(out) != len(request.dates) {
					err = errors.New("missing previews")
				}
				done <- err
			}()
		}
		synctest.Wait()
		if calls.Load() != 4 {
			t.Fatalf("calls=%d want 4 (3 dates plus separate budget)", calls.Load())
		}
		close(release)
		for range 3 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
	})
}
func TestReadWaiterCancellationAndBounds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var group readGroup
		release := make(chan struct{})
		done := make(chan error, 320)
		fn := func(ctx context.Context) (any, error) {
			select {
			case <-release:
				return "ok", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		for range 320 {
			go func() { _, err := group.do(context.Background(), "same", fn); done <- err }()
		}
		synctest.Wait()
		if _, err := group.do(context.Background(), "same", fn); !errors.Is(err, journal.ErrUnavailable) {
			t.Fatalf("admission err=%v", err)
		}
		close(release)
		for range 320 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
		if _, err := group.do(context.Background(), "same", fn); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		release = make(chan struct{})
		owner := make(chan error, 1)
		go func() { _, err := group.do(context.Background(), "waiter", fn); owner <- err }()
		synctest.Wait()
		waiter := make(chan error, 1)
		go func() { _, err := group.do(ctx, "waiter", fn); waiter <- err }()
		synctest.Wait()
		cancel()
		if !errors.Is(<-waiter, context.Canceled) {
			t.Fatal("waiter must cancel independently")
		}
		close(release)
		if err := <-owner; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
	})
}
func TestReadDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var group readGroup
		start := time.Now()
		_, err := group.do(context.Background(), "timeout", func(ctx context.Context) (any, error) { <-ctx.Done(); return nil, ctx.Err() })
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) != 30*time.Second {
			t.Fatalf("deadline: %v %v", err, time.Since(start))
		}
	})
}

func TestDistinctReadBoundAndErrorCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var group readGroup
		release := make(chan struct{})
		done := make(chan error, 64)
		for i := range 64 {
			go func() {
				_, err := group.do(context.Background(), fmt.Sprint(i), func(context.Context) (any, error) { <-release; return nil, journal.ErrAuthRequired })
				done <- err
			}()
		}
		synctest.Wait()
		if _, err := group.do(context.Background(), "overflow", func(context.Context) (any, error) { t.Error("overflow executed"); return nil, nil }); !errors.Is(err, journal.ErrUnavailable) {
			t.Fatalf("capacity err=%v", err)
		}
		close(release)
		for range 64 {
			if !errors.Is(<-done, journal.ErrAuthRequired) {
				t.Fatal("error identity lost")
			}
		}
		value, err := group.do(context.Background(), "0", func(context.Context) (any, error) { return "retried", nil })
		if err != nil || value != "retried" {
			t.Fatalf("completed error retained: %v %v", value, err)
		}
	})
}

func TestDevicesShareSuccessAndRetryAuthenticationIndependently(t *testing.T) {
	for _, scenario := range []string{"success", "expired-leader", "both-expired"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cache := New(nil, 0)
				release := make(chan struct{})
				first := make(chan error, 1)
				second := make(chan error, 1)
				go func() {
					_, err := cache.Wrap("google:account", slowReadStore{get: func(context.Context, civil.Date) (journal.Entry, error) {
						<-release
						if scenario != "success" {
							return journal.Entry{}, journal.ErrAuthRequired
						}
						return journal.Entry{Text: "shared"}, nil
					}}).Get(context.Background(), civil.Date{})
					first <- err
				}()
				synctest.Wait()
				var ownCalls atomic.Int32
				go func() {
					entry, err := cache.Wrap("google:account", slowReadStore{get: func(context.Context, civil.Date) (journal.Entry, error) {
						ownCalls.Add(1)
						if scenario == "both-expired" {
							return journal.Entry{}, journal.ErrAuthRequired
						}
						return journal.Entry{Text: "own"}, nil
					}}).Get(context.Background(), civil.Date{})
					want := "own"
					if scenario == "success" {
						want = "shared"
					}
					if err == nil && entry.Text != want {
						err = errors.New("wrong device result")
					}
					second <- err
				}()
				synctest.Wait()
				close(release)
				leaderErr, followerErr := <-first, <-second
				if scenario == "success" {
					if leaderErr != nil || followerErr != nil || ownCalls.Load() != 0 {
						t.Fatalf("success not shared: %v %v calls=%d", leaderErr, followerErr, ownCalls.Load())
					}
				} else {
					if !errors.Is(leaderErr, journal.ErrAuthRequired) || ownCalls.Load() != 1 {
						t.Fatalf("auth retry: %v calls=%d", leaderErr, ownCalls.Load())
					}
					if scenario == "both-expired" {
						if !errors.Is(followerErr, journal.ErrAuthRequired) {
							t.Fatal(followerErr)
						}
					} else if followerErr != nil {
						t.Fatal(followerErr)
					}
				}
			})
		})
	}
}

func TestColdPreviewConcurrencyAndLatency(t *testing.T) {
	for _, count := range []int{8, 20} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var active, peak atomic.Int32
				base := slowReadStore{preview: func(ctx context.Context, dates []civil.Date, n int) (map[civil.Date]string, error) {
					current := active.Add(1)
					defer active.Add(-1)
					for previous := peak.Load(); current > previous; previous = peak.Load() {
						if peak.CompareAndSwap(previous, current) {
							break
						}
					}
					select {
					case <-time.After(100 * time.Millisecond):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					return map[civil.Date]string{dates[0]: "synthetic"}, nil
				}}
				dates := make([]civil.Date, count)
				for i := range dates {
					dates[i] = civil.Date{Year: 2020, Month: 1, Day: i + 1}
				}
				start := time.Now()
				result, err := New(nil, 0).Wrap("synthetic", base).Previews(context.Background(), dates, 44)
				elapsed := time.Since(start)
				want := 100 * time.Millisecond
				if count == 20 {
					want = 300 * time.Millisecond
				}
				t.Logf("cold %d previews: elapsed=%v peak downloads=%d", count, elapsed, peak.Load())
				if err != nil || len(result) != count || peak.Load() != 8 || elapsed != want {
					t.Fatalf("previews=%d err=%v peak=%d elapsed=%v", len(result), err, peak.Load(), elapsed)
				}
			})
		})
	}
}
