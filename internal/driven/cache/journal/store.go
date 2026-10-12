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
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}
