package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
	"myjournal-app/internal/driven/cache/cachestore"
)

// authorizationProvider uses real OAuth, Drive and cache adapters with synthetic
// provider responses, including the period before remote revocation propagates.
func authorizationProvider(t *testing.T, block func(http.ResponseWriter, *http.Request) bool) (*Provider, *int) {
	t.Helper()
	p, _, _ := collapseProvider(t, block)
	transport := p.client.Transport
	revokeStatus := http.StatusOK
	p.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "oauth2.googleapis.com" {
			return transport.RoundTrip(r)
		}
		response := &http.Response{Header: http.Header{"Content-Type": {"application/json"}}, StatusCode: http.StatusOK}
		if r.URL.Path == "/revoke" {
			response.StatusCode = revokeStatus
			response.Body = io.NopCloser(strings.NewReader(`{}`))
			return response, nil
		}
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		if r.Form.Get("refresh_token") == "revoked-refresh" {
			response.StatusCode = http.StatusBadRequest
			response.Body = io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))
			return response, nil
		}
		return transport.RoundTrip(r)
	})
	return p, &revokeStatus
}

func authorizedConnection(t *testing.T, refresh string, issuedAt int64) provider.Connection {
	t.Helper()
	current := collapseConnection()
	encoded, err := json.Marshal(credentials{Version: credentialsVersion, Subject: current.AccountID, Refresh: refresh, IssuedAt: issuedAt})
	require.NoError(t, err)
	current.Credentials = string(encoded)
	return current
}

func wrappedConnection(t *testing.T, p *Provider, cache *cachestore.Cache, current provider.Connection) journal.Store {
	t.Helper()
	raw, err := p.Store(context.Background(), current)
	require.NoError(t, err)
	return cache.Wrap("google:"+current.AccountID, raw).(journal.OperationStore).ForOperation()
}

func TestCachedReadsRejectRevokedGoogleConnection(t *testing.T) {
	ctx := context.Background()
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	p, _ := authorizationProvider(t, nil)
	cache := cachestore.New(newLRUStorage(t, time.Minute), time.Minute)
	valid := wrappedConnection(t, p, cache, collapseConnection())
	_, err := valid.Put(ctx, date, "private entry", "")
	require.NoError(t, err)
	_, err = valid.Get(ctx, date)
	require.NoError(t, err)
	_, err = valid.ListYears(ctx)
	require.NoError(t, err)
	_, err = valid.ListYear(ctx, 2020)
	require.NoError(t, err)
	// A valid cached access token must not bypass a revoked refresh grant.
	p.cacheToken("revoked-refresh", &oauth2.Token{AccessToken: "synthetic-access", Expiry: time.Now().Add(time.Hour)})
	for _, operation := range []string{"entry", "fresh entry", "years", "year", "previews"} {
		t.Run(operation, func(t *testing.T) {
			denied := wrappedConnection(t, p, cache, authorizedConnection(t, "revoked-refresh", 0))
			checkDeniedRead(t, ctx, denied, date, operation)
		})
	}
}

func checkDeniedRead(t *testing.T, ctx context.Context, store journal.Store, date journal.Date, operation string) {
	t.Helper()
	switch operation {
	case "entry", "fresh entry":
		var entry journal.Entry
		var err error
		if operation == "fresh entry" {
			entry, err = store.(journal.FreshEntryStore).GetFresh(ctx, date)
		} else {
			entry, err = store.Get(ctx, date)
		}
		require.ErrorIs(t, err, journal.ErrAuthRequired)
		assert.Empty(t, entry)
	case "years":
		years, err := store.ListYears(ctx)
		require.ErrorIs(t, err, journal.ErrAuthRequired)
		assert.Empty(t, years)
	case "year":
		entries, err := store.ListYear(ctx, date.Year)
		require.ErrorIs(t, err, journal.ErrAuthRequired)
		assert.Empty(t, entries)
	case "previews":
		previews, err := store.Previews(ctx, []journal.Date{date}, 20)
		require.ErrorIs(t, err, journal.ErrAuthRequired)
		assert.Empty(t, previews)
	default:
		t.Fatalf("unknown read operation: %s", operation)
	}
}

func TestDisconnectRejectsRetainedCookiesAfterReconnect(t *testing.T) {
	ctx := context.Background()
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	p, revokeStatus := authorizationProvider(t, nil)
	cache := cachestore.New(newLRUStorage(t, time.Minute), time.Minute)
	original := collapseConnection() // Legacy cookies have no credential issuance field.
	retained := wrappedConnection(t, p, cache, original)
	_, err := retained.Put(ctx, date, "private before disconnect", "")
	require.NoError(t, err)
	before, err := retained.Get(ctx, date)
	require.NoError(t, err)
	*revokeStatus = http.StatusServiceUnavailable
	require.ErrorIs(t, p.Revoke(ctx, original), journal.ErrUnavailable)
	_, err = retained.Get(ctx, date)
	require.NoError(t, err, "failed disconnect must preserve authorization")
	*revokeStatus = http.StatusOK
	require.NoError(t, p.Revoke(ctx, original))
	for _, operation := range []string{"entry", "fresh entry", "years", "year", "previews"} {
		t.Run(operation, func(t *testing.T) {
			checkDeniedRead(t, ctx, retained, date, operation)
			checkDeniedRead(t, ctx, wrappedConnection(t, p, cache, original), date, operation)
		})
	}
	reconnected := authorizedConnection(t, "new-refresh", time.Now().UnixNano())
	current := wrappedConnection(t, p, cache, reconnected)
	_, err = current.Put(ctx, date, "private after reconnect", before.Version)
	require.NoError(t, err)
	after, err := current.Get(ctx, date)
	require.NoError(t, err)
	assert.Equal(t, "private after reconnect", after.Text)
	// Token eviction and fresh snapshots must never reactivate an old cookie.
	for i := range maxCachedTokens * 2 {
		p.cacheToken(fmt.Sprint(i), &oauth2.Token{AccessToken: "synthetic-access", Expiry: time.Now().Add(time.Hour)})
	}
	checkDeniedRead(t, ctx, wrappedConnection(t, p, cache, original), date, "entry")
	require.ErrorIs(t, p.Revoke(ctx, original), journal.ErrAuthRequired)
	// Issued credentials are rejected too; this is not limited to legacy cookies.
	require.NoError(t, p.Revoke(ctx, reconnected))
	checkDeniedRead(t, ctx, wrappedConnection(t, p, cache, reconnected), date, "entry")
}

func TestSharedReadChecksEachGoogleRecipient(t *testing.T) {
	for _, operation := range []string{"entry", "fresh entry", "previews"} {
		for _, disconnect := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/disconnect=%t", operation, disconnect), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					release := make(chan struct{})
					p, _ := authorizationProvider(t, func(_ http.ResponseWriter, r *http.Request) bool {
						if r.URL.Query().Get("alt") == "media" {
							<-release
						}
						return false
					})
					ctx := context.Background()
					date := journal.Date{Year: 2020, Month: 1, Day: 1}
					_, err := collapseStore(t, p, ctx).Put(ctx, date, "private entry", "")
					require.NoError(t, err)
					cache := cachestore.New(nil, 0)
					leader := wrappedConnection(t, p, cache, collapseConnection())
					follower := wrappedConnection(t, p, cache, collapseConnection())
					badFollower := wrappedConnection(t, p, cache, authorizedConnection(t, "revoked-refresh", 0))
					read := func(store journal.Store) error {
						switch operation {
						case "entry":
							_, err := store.Get(ctx, date)
							return err
						case "fresh entry":
							_, err := store.(journal.FreshEntryStore).GetFresh(ctx, date)
							return err
						default:
							_, err := store.Previews(ctx, []journal.Date{date}, 20)
							return err
						}
					}
					first, second := make(chan error, 1), make(chan error, 1)
					go func() { first <- read(leader) }()
					synctest.Wait()
					require.ErrorIs(t, read(badFollower), journal.ErrAuthRequired, "revoked follower must not inherit leader success")
					go func() { second <- read(follower) }()
					synctest.Wait()
					// Both requests were authorized before joining the same successful load.
					if disconnect {
						require.NoError(t, p.Revoke(ctx, collapseConnection()))
					}
					close(release)
					firstErr, secondErr := <-first, <-second
					if disconnect {
						require.ErrorIs(t, firstErr, journal.ErrAuthRequired)
						require.ErrorIs(t, secondErr, journal.ErrAuthRequired)
					} else {
						assert.NoError(t, firstErr)
						assert.NoError(t, secondErr)
					}
				})
			})
		}
	}
}

func TestGoogleReadAuthorizationCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, _ := authorizationProvider(t, nil)
		ctx := context.Background()
		store := collapseStore(t, p, ctx)
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		require.ErrorIs(t, store.Authorize(canceled), context.Canceled)

		unlock, err := p.lock(ctx, "token:synthetic-refresh")
		require.NoError(t, err)
		defer unlock()
		waiting, stop := context.WithCancel(ctx)
		defer stop()
		done := make(chan error, 1)
		go func() { done <- store.Authorize(waiting) }()
		synctest.Wait()
		stop()
		require.ErrorIs(t, <-done, context.Canceled)
	})
}
