// Package bigcache implements cachestore.ByteCache using BigCache.
package bigcache

import (
	"context"
	"errors"
	"strings"
	"time"

	library "github.com/allegro/bigcache"

	cachestore "github/m-ridha/myjournal-app/internal/driven/cache/cachestore"
)

const (
	shardCount    = 16
	queueLimitMiB = 16
	// Reserve space for BigCache's entry header and queue framing.
	entryOverhead = 64
	maxShardBytes = queueLimitMiB * 1024 * 1024 / shardCount
	maxKeyBytes   = 65535 // BigCache v1 stores key length as uint16.
)

var errClosed = errors.New("cache is closed")

// Adapter owns bounded BigCache byte queues. The queue limit excludes Go maps
// and runtime overhead; it is not a process memory ceiling.
type Adapter struct {
	cache  *library.BigCache
	gate   chan struct{}
	closed bool
}

var _ cachestore.ByteCache = (*Adapter)(nil)

// New constructs an adapter with modest initial allocation and a byte-queue cap.
func New(ttl time.Duration) (*Adapter, error) {
	if ttl <= 0 {
		return nil, errors.New("cache TTL must be positive")
	}
	c, err := library.NewBigCache(library.Config{
		Shards: shardCount, LifeWindow: ttl, CleanWindow: time.Second,
		MaxEntriesInWindow: 2048, MaxEntrySize: 512, HardMaxCacheSize: queueLimitMiB,
		Verbose: false,
	})
	if err != nil {
		return nil, err
	}
	return &Adapter{cache: c, gate: make(chan struct{}, 1)}, nil
}

func (a *Adapter) lock(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case a.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		a.unlock()
		return err
	}
	if a.closed {
		a.unlock()
		return errClosed
	}
	return nil
}
func (a *Adapter) unlock() { <-a.gate }

// Get returns a caller-owned value or cachestore.ErrMiss.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	if err := a.lock(ctx); err != nil {
		return nil, err
	}
	defer a.unlock()
	value, err := a.cache.Get(key)
	if errors.Is(err, library.ErrEntryNotFound) {
		return nil, cachestore.ErrMiss
	}
	return value, err
}

// Set stores a copy; oversized entries are rejected without evicting other data.
func (a *Adapter) Set(ctx context.Context, key string, value []byte) error {
	if err := a.lock(ctx); err != nil {
		return err
	}
	defer a.unlock()
	if len(key) > maxKeyBytes || len(key) > maxShardBytes-entryOverhead || len(value) > maxShardBytes-entryOverhead-len(key) {
		return cachestore.ErrCapacity
	}
	return a.cache.Set(key, value)
}

// Delete removes a key, accepting absence.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	if err := a.lock(ctx); err != nil {
		return err
	}
	defer a.unlock()
	return a.delete(key)
}
func (a *Adapter) delete(key string) error {
	err := a.cache.Delete(key)
	if errors.Is(err, library.ErrEntryNotFound) {
		return nil
	}
	return err
}

// DeletePrefix removes all matching keys while excluding concurrent writes.
func (a *Adapter) DeletePrefix(ctx context.Context, prefix string) error {
	if err := a.lock(ctx); err != nil {
		return err
	}
	defer a.unlock()
	iterator := a.cache.Iterator()
	for iterator.SetNext() {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry, err := iterator.Value()
		// BigCache's cleaner may remove expired entries during iteration.
		if errors.Is(err, library.ErrCannotRetrieveEntry) {
			continue
		}
		if err != nil {
			return err
		}
		if strings.HasPrefix(entry.Key(), prefix) {
			if err := a.delete(entry.Key()); err != nil {
				return err
			}
		}
	}
	return ctx.Err()
}

// Close stops cleanup exactly once; later operations fail explicitly.
func (a *Adapter) Close() error {
	a.gate <- struct{}{}
	defer a.unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	return a.cache.Close()
}
