package cachestore

import (
	"context"
	"errors"
)

// ErrMiss identifies an absent cache key.
var ErrMiss = errors.New("cache miss")

// ErrCapacity means a value cannot fit; callers may skip caching it.
var ErrCapacity = errors.New("cache capacity exceeded")

// ByteCache stores disposable byte snapshots. Set must copy retained inputs and Get
// must return caller-owned bytes. Missing Delete is successful. DeletePrefix
// removes all matching current keys; it is not a distributed transaction. Methods
// honor cancellation, including lock waits, and return explicit failures.
type ByteCache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte) error
	Delete(context.Context, string) error
	DeletePrefix(context.Context, string) error
}
