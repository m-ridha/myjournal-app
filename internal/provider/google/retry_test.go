package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/api/googleapi"
)

func TestReadRetryClassification(t *testing.T) {
	transportFailure := errors.New("transport failure")
	for _, tc := range []struct {
		name     string
		err      error
		attempts int
	}{
		{"rate limited", &googleapi.Error{Code: http.StatusTooManyRequests}, 3},
		{"unavailable", &googleapi.Error{Code: http.StatusServiceUnavailable}, 3},
		{"internal error", &googleapi.Error{Code: http.StatusInternalServerError}, 3},
		{"wrapped transient", fmt.Errorf("read: %w", &googleapi.Error{Code: http.StatusBadGateway}), 3},
		{"quota rate limit", &googleapi.Error{Code: http.StatusForbidden, Errors: []googleapi.ErrorItem{{Reason: "rateLimitExceeded"}}}, 3},
		{"user rate limit", &googleapi.Error{Code: http.StatusForbidden, Errors: []googleapi.ErrorItem{{Reason: "userRateLimitExceeded"}}}, 3},
		{"permission denied", &googleapi.Error{Code: http.StatusForbidden, Errors: []googleapi.ErrorItem{{Reason: "insufficientPermissions"}}}, 1},
		{"unauthorized", &googleapi.Error{Code: http.StatusUnauthorized}, 1},
		{"missing", &googleapi.Error{Code: http.StatusNotFound}, 1},
		{"precondition failed", &googleapi.Error{Code: http.StatusPreconditionFailed}, 1},
		{"transport failure", transportFailure, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				got, err := retryRead(context.Background(), func() (string, error) {
					calls++
					return "unverified result", tc.err
				})
				if calls != tc.attempts || !errors.Is(err, tc.err) || got != "" {
					t.Errorf("retryRead(%s) = (%q, %v), calls %d; want empty result, original error, calls %d", tc.name, got, err, calls, tc.attempts)
				}
			})
		})
	}
}

func TestReadRetryRecoversWithinBound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		got, err := retryRead(context.Background(), func() (string, error) {
			calls++
			if calls < 3 {
				return "", &googleapi.Error{Code: http.StatusTooManyRequests}
			}
			return "verified result", nil
		})
		if err != nil || got != "verified result" || calls != 3 {
			t.Errorf("retryRead(transient then success) = (%q, %v), calls %d; want verified result, nil, 3 calls", got, err, calls)
		}
	})
}

func TestReadRetryCancellationDuringBackoff(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%t", deadline), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				want := context.Canceled
				if deadline {
					cancel()
					ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
					want = context.DeadlineExceeded
				}
				defer cancel()
				calls := 0
				done := make(chan error, 1)
				go func() {
					_, err := retryRead(ctx, func() (string, error) {
						calls++
						return "", &googleapi.Error{Code: http.StatusServiceUnavailable}
					})
					done <- err
				}()
				synctest.Wait() // The first attempt is blocked on its backoff timer.
				if !deadline {
					cancel()
				}
				if err := <-done; !errors.Is(err, want) || calls != 1 {
					t.Errorf("retryRead interrupted backoff = %v, calls %d; want %v, 1 call", err, calls, want)
				}
			})
		})
	}
}
