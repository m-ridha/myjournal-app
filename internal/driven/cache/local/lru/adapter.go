// Package lru implements cachestore.ByteCache using entry-count-bounded LRU storage.
package lru

import (
	"container/list"
	"context"
	"errors"
	"strings"
	"time"

	cachestore "github/m-ridha/myjournal-app/internal/driven/cache/cachestore"
)

const defaultCapacity = 2048

var errClosed = errors.New("cache is closed")

type item struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// Adapter is an entry-count-bounded LRU with lazy expiry. It owns no goroutines
// and supports the same explicit Close lifecycle as BigCache.
// The capacity is not a byte-memory limit.
type Adapter struct {
	gate     chan struct{}
	closed   bool
	ttl      time.Duration
	capacity int
	now      func() time.Time
	items    map[string]*list.Element
	lru      list.List
}

var _ cachestore.ByteCache = (*Adapter)(nil)

// New constructs an LRU with a 2048-entry default capacity and positive TTL.
func New(ttl time.Duration) (*Adapter, error) { return NewWithCapacity(ttl, defaultCapacity) }

// NewWithCapacity constructs an LRU with an explicit entry bound. Nonpositive
// capacity selects the default; TTL must be positive.
func NewWithCapacity(ttl time.Duration, capacity int) (*Adapter, error) {
	if ttl <= 0 {
		return nil, errors.New("cache TTL must be positive")
	}
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	return &Adapter{gate: make(chan struct{}, 1), ttl: ttl, capacity: capacity, now: time.Now, items: make(map[string]*list.Element)}, nil
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

// Get returns a copy and promotes an unexpired entry to most recently used.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	if err := a.lock(ctx); err != nil {
		return nil, err
	}
	defer a.unlock()
	element := a.items[key]
	if element == nil {
		return nil, cachestore.ErrMiss
	}
	stored := element.Value.(*item)
	if !a.now().Before(stored.expiresAt) {
		a.remove(element)
		return nil, cachestore.ErrMiss
	}
	a.lru.MoveToFront(element)
	return append([]byte(nil), stored.value...), nil
}

// Set stores a copy and evicts least recently used entries above the entry bound.
func (a *Adapter) Set(ctx context.Context, key string, value []byte) error {
	if err := a.lock(ctx); err != nil {
		return err
	}
	defer a.unlock()
	if element := a.items[key]; element != nil {
		stored := element.Value.(*item)
		stored.value = append([]byte(nil), value...)
		stored.expiresAt = a.now().Add(a.ttl)
		a.lru.MoveToFront(element)
		return nil
	}
	stored := &item{key: key, value: append([]byte(nil), value...), expiresAt: a.now().Add(a.ttl)}
	a.items[key] = a.lru.PushFront(stored)
	for len(a.items) > a.capacity {
		a.remove(a.lru.Back())
	}
	return nil
}

// Delete removes an entry, accepting absence.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	if err := a.lock(ctx); err != nil {
		return err
	}
	defer a.unlock()
	if element := a.items[key]; element != nil {
		a.remove(element)
	}
	return nil
}

// DeletePrefix removes matching current keys. Cancellation can leave a partial
// deletion, which the cache coordinator handles by disabling further reuse.
func (a *Adapter) DeletePrefix(ctx context.Context, prefix string) error {
	if err := a.lock(ctx); err != nil {
		return err
	}
	defer a.unlock()
	for key, element := range a.items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if strings.HasPrefix(key, prefix) {
			a.remove(element)
		}
	}
	return ctx.Err()
}

func (a *Adapter) remove(element *list.Element) {
	delete(a.items, element.Value.(*item).key)
	a.lru.Remove(element)
}

// Close releases retained bytes exactly once; later operations fail explicitly.
func (a *Adapter) Close() error {
	a.gate <- struct{}{}
	defer a.unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	clear(a.items)
	a.lru.Init()
	return nil
}
