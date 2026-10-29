package google

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
	cachestore "myjournal-app/internal/driven/cache/cachestore"
)

func collapseProvider(t *testing.T, block func(http.ResponseWriter, *http.Request) bool) (*Provider, *fakeDrive, *benchmarkRequests) {
	t.Helper()
	fake := &fakeDrive{files: map[string]*drive.File{}, bodies: map[string][]byte{}, pageSize: 1000}
	counts := &benchmarkRequests{counts: make(map[string]int)}
	p := New(Config{ClientID: "synthetic-client", ClientSecret: "synthetic-secret"})
	handler := counts.handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if block != nil && block(w, r) {
			return
		}
		fake.ServeHTTP(w, r)
	}))
	p.client = &http.Client{Timeout: requestTimeout, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "oauth2.googleapis.com" && r.Header.Get("Authorization") != "Bearer synthetic-access" {
			t.Error("missing synthetic access token")
		}
		w := httptest.NewRecorder()
		if r.URL.Host == "oauth2.googleapis.com" {
			if block == nil || !block(w, r) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"access_token":"synthetic-access","token_type":"Bearer","expires_in":3600}`)
			}
		} else {
			handler.ServeHTTP(w, r)
		}
		if err := r.Context().Err(); err != nil {
			return nil, err
		}
		return w.Result(), nil
	})}
	p.cacheToken("synthetic-refresh", &oauth2.Token{AccessToken: "synthetic-access", Expiry: time.Now().Add(time.Hour)})
	return p, fake, counts
}

func collapseConnection() provider.Connection {
	return provider.Connection{ProviderID: "google", AccountID: "synthetic-account", Credentials: `{"v":1,"s":"synthetic-account","r":"synthetic-refresh"}`}
}

func collapseStore(t *testing.T, p *Provider, ctx context.Context) *Store {
	t.Helper()
	store, err := p.Store(ctx, collapseConnection())
	if err != nil {
		t.Fatal(err)
	}
	return store.(*Store)
}

func TestProviderReadLeaderCancellationUsesNewOAuthContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var blocked atomic.Bool
		p, _, counts := collapseProvider(t, func(_ http.ResponseWriter, r *http.Request) bool {
			if r.URL.Query().Get("alt") == "media" && blocked.CompareAndSwap(false, true) {
				select {
				case <-release:
				case <-r.Context().Done():
					return true
				}
			}
			return false
		})
		date := journal.Date{Year: 2020, Month: 1, Day: 1}
		if _, err := collapseStore(t, p, context.Background()).Put(context.Background(), date, "synthetic", ""); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cache := cachestore.New(nil, 0)
		first := cache.Wrap("google:synthetic-account", collapseStore(t, p, ctx))
		second := cache.Wrap("google:synthetic-account", collapseStore(t, p, context.Background()))
		leader, waiter := make(chan error, 1), make(chan error, 1)
		go func() { _, err := first.Get(ctx, date); leader <- err }()
		synctest.Wait()
		go func() {
			entry, err := second.Get(context.Background(), date)
			if err == nil && entry.Text != "synthetic" {
				err = errors.New("wrong recovered entry")
			}
			waiter <- err
		}()
		synctest.Wait()
		cancel()
		if err := <-leader; !errors.Is(err, context.Canceled) {
			t.Fatalf("leader=%v", err)
		}
		if err := <-waiter; err != nil {
			t.Fatalf("independent OAuth waiter=%v", err)
		}
		close(release)
		if counts.counts["download"] != 2 {
			t.Fatalf("cancelled leader/retry downloads=%d, want 2", counts.counts["download"])
		}
	})
}

func TestConcurrentBrowsingRequestCounts(t *testing.T) {
	for _, scenario := range []string{"entries", "calendar", "mixed", "editor"} {
		for _, collapse := range []bool{false, true} {
			t.Run(scenario+"/"+map[bool]string{false: "baseline", true: "collapsed"}[collapse], func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					p, _, counts := collapseProvider(t, func(_ http.ResponseWriter, r *http.Request) bool {
						if r.Method == http.MethodGet {
							time.Sleep(time.Millisecond)
						}
						return false
					})
					store := collapseStore(t, p, context.Background())
					for day := 1; day <= 20; day++ {
						if _, err := store.Put(context.Background(), journal.Date{Year: 2020, Month: 1, Day: day}, "synthetic", ""); err != nil {
							t.Fatal(err)
						}
					}
					counts.counts = make(map[string]int)
					cache := cachestore.New(nil, 0)
					done := make(chan error, 8)
					for i := range 8 {
						view := collapseStore(t, p, context.Background())
						var wrapped journal.Store = view
						if collapse {
							wrapped = cache.Wrap("google:synthetic-account", view)
						}
						service := journal.NewService(wrapped, benchmarkClock{})
						go func() {
							var err error
							if scenario == "editor" {
								_, err = service.Open(context.Background(), journal.Date{Year: 2020, Month: 1, Day: 1})
							} else if scenario == "calendar" || (scenario == "mixed" && i%2 == 0) {
								_, err = service.Month(context.Background(), 2020, time.January)
							} else {
								var page journal.ListPage
								page, err = service.ListEntries(context.Background(), 1, 20)
								if err == nil && len(page.Items) != 20 {
									err = errors.New("wrong entry page size")
								}
							}
							done <- err
						}()
					}
					for range 8 {
						if err := <-done; err != nil {
							t.Fatal(err)
						}
					}
					lists, downloads := counts.counts["list"], counts.counts["download"]
					t.Logf("eight-request %s burst: lists=%d downloads=%d", scenario, lists, downloads)
					if collapse {
						maxLists := 8
						if scenario == "calendar" || scenario == "editor" {
							maxLists = 3
						}
						if lists > maxLists {
							t.Fatalf("metadata lists=%d, want <=%d", lists, maxLists)
						}
						want := 20
						if scenario == "calendar" {
							want = 0
						}
						if scenario == "editor" {
							want = 1
						}
						if downloads != want {
							t.Fatalf("downloads=%d want %d", downloads, want)
						}
					}
				})
			})
		}
	}
}
func TestMutationSeparatesNewReadsFromOldFlight(t *testing.T) {
	for _, operation := range []string{"save", "trash", "failed-save"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				var blocked atomic.Bool
				p, _, _ := collapseProvider(t, func(w http.ResponseWriter, r *http.Request) bool {
					if r.URL.Query().Get("alt") == "media" && blocked.CompareAndSwap(false, true) {
						select {
						case <-release:
						case <-r.Context().Done():
							return true
						}
						_, _ = io.WriteString(w, "old")
						return true
					}
					return false
				})
				date := journal.Date{Year: 2020, Month: 1, Day: 1}
				cache := cachestore.New(nil, 0)
				store := cache.Wrap("google:synthetic-account", collapseStore(t, p, context.Background()))
				created, err := store.Put(context.Background(), date, "old", "")
				if err != nil {
					t.Fatal(err)
				}
				original := make(chan error, 1)
				go func() {
					_, err := cache.Wrap("google:synthetic-account", collapseStore(t, p, context.Background())).Get(context.Background(), date)
					original <- err
				}()
				synctest.Wait()
				switch operation {
				case "save":
					_, err = store.Put(context.Background(), date, "new", created.Version)
				case "trash":
					err = store.Trash(context.Background(), date)
				case "failed-save":
					_, err = store.Put(context.Background(), date, "new", "stale")
				}
				if operation == "failed-save" {
					if !errors.Is(err, journal.ErrConflict) {
						t.Fatal(err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				entry, err := cache.Wrap("google:synthetic-account", collapseStore(t, p, context.Background())).Get(context.Background(), date)
				if operation == "trash" {
					if !errors.Is(err, journal.ErrNotFound) {
						t.Fatalf("post-trash read=%v", err)
					}
				} else {
					want := "new"
					if operation == "failed-save" {
						want = "old"
					}
					if err != nil || entry.Text != want {
						t.Fatalf("post-mutation read=%q %v", entry.Text, err)
					}
				}
				close(release)
				if err := <-original; err != nil {
					t.Fatalf("original in-flight read=%v", err)
				}
			})
		})
	}
}
func TestOverlappingPreviewBatchesShareDownloads(t *testing.T) {
	for _, collapse := range []bool{false, true} {
		t.Run(map[bool]string{false: "baseline", true: "collapsed"}[collapse], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				p, _, counts := collapseProvider(t, func(_ http.ResponseWriter, r *http.Request) bool {
					if r.URL.Query().Get("alt") == "media" {
						select {
						case <-release:
						case <-r.Context().Done():
							return true
						}
					}
					return false
				})
				dates := []journal.Date{{Year: 2020, Month: 1, Day: 1}, {Year: 2020, Month: 1, Day: 2}, {Year: 2020, Month: 1, Day: 3}}
				for _, date := range dates {
					if _, err := collapseStore(t, p, context.Background()).Put(context.Background(), date, "synthetic-"+date.String(), ""); err != nil {
						t.Fatal(err)
					}
				}
				counts.counts = make(map[string]int)
				cache := cachestore.New(nil, 0)
				done := make(chan error, 2)
				for _, batch := range [][]journal.Date{dates[:2], dates[1:]} {
					store := collapseStore(t, p, context.Background()).ForOperation()
					if collapse {
						store = cache.Wrap("google:synthetic-account", store)
					}
					go func() {
						result, err := store.Previews(context.Background(), batch, 44)
						if err == nil {
							for _, date := range batch {
								if result[date] != "synthetic-"+date.String() {
									err = errors.New("wrong batch preview")
								}
							}
						}
						done <- err
					}()
				}
				synctest.Wait()
				close(release)
				for range 2 {
					if err := <-done; err != nil {
						t.Fatal(err)
					}
				}
				want := 4
				if collapse {
					want = 3
				}
				if got := counts.counts["download"]; got != want {
					t.Fatalf("overlapping preview downloads=%d, want %d", got, want)
				}
				t.Logf("two overlapping preview batches: downloads=%d", counts.counts["download"])
			})
		})
	}
}

func TestDiscoveryClearSeparatesActiveLoads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var blocked atomic.Bool
		p, _, _ := collapseProvider(t, func(w http.ResponseWriter, r *http.Request) bool {
			if r.Method == http.MethodGet && blocked.CompareAndSwap(false, true) {
				<-release
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"files":[{"id":"old"}]}`)
				return true
			}
			return false
		})
		scoped := collapseStore(t, p, context.Background()).ForOperation().(*Store)
		original := make(chan error, 1)
		go func() {
			files, err := scoped.list(context.Background(), "trashed = false")
			if err == nil && (len(files) != 1 || files[0].Id != "old") {
				err = errors.New("old snapshot changed")
			}
			original <- err
		}()
		synctest.Wait()
		scoped.clearDiscovery()
		files, err := scoped.list(context.Background(), "trashed = false")
		if err != nil || len(files) != 0 {
			t.Fatalf("new load joined old generation: %v %v", files, err)
		}
		close(release)
		if err := <-original; err != nil {
			t.Fatal(err)
		}
		files, err = scoped.list(context.Background(), "trashed = false")
		if err != nil || len(files) != 0 {
			t.Fatalf("old load filled cleared discovery: %v %v", files, err)
		}
	})
}
