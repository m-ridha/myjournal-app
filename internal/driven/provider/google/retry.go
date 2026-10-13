package google

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"time"

	"google.golang.org/api/googleapi"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

func retryRead[T any](ctx context.Context, call func() (T, error)) (T, error) {
	var zero T
	for attempt := 0; attempt < readAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		result, err := call()
		if err == nil {
			return result, nil
		}
		var apiErr *googleapi.Error
		retry := errors.As(err, &apiErr) && (apiErr.Code == http.StatusTooManyRequests || apiErr.Code >= http.StatusInternalServerError || (apiErr.Code == http.StatusForbidden && rateLimited(apiErr)))
		if !retry || attempt == readAttempts-1 {
			return zero, err
		}
		delay := time.Duration(100*(1<<attempt)+rand.IntN(100)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, journal.ErrUnavailable
}

func rateLimited(err *googleapi.Error) bool {
	for _, item := range err.Errors {
		if item.Reason == "rateLimitExceeded" || item.Reason == "userRateLimitExceeded" {
			return true
		}
	}
	return false
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, journal.ErrAuthRequired) || errors.Is(err, journal.ErrUnavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var e *googleapi.Error
	if errors.As(err, &e) {
		switch e.Code {
		case http.StatusUnauthorized:
			return journal.ErrAuthRequired
		case http.StatusNotFound:
			return journal.ErrNotFound
		case http.StatusConflict, http.StatusPreconditionFailed:
			return journal.ErrConflict
		case http.StatusForbidden:
			for _, item := range e.Errors {
				if item.Reason == "insufficientPermissions" {
					return journal.ErrAuthRequired
				}
			}
		}
	}
	return journal.ErrUnavailable
}
