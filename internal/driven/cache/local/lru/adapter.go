// Package lru preserves the former map/list cache storage as an optional adapter.
// Production dependency injection continues to select BigCache.
package lru

import (
	"container/list"
	"context"
	"strings"
	"time"

	"github/mridha/myjournal-app/internal/driven/cache/journal"
)

const defaultCapacity = 2048

type item struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// Adapter is an entry-count-bounded LRU with lazy expiry. It owns no goroutines
// and requires no shutdown cleanup. The capacity is not a byte-memory limit.
type Adapter struct {
	gate     chan struct{}
	ttl      time.Duration
	capacity int
	now      func() time.Time
	items    map[string]*list.Element
	lru      list.List
}

var _ cachestore.ByteCache = (*Adapter)(nil)

// New constructs an LRU with the previous 2048-entry default capacity.
func New(ttl time.Duration) *Adapter { return NewWithCapacity(ttl, defaultCapacity) }

// NewWithCapacity constructs an LRU with an explicit entry bound. Nonpositive
// capacity selects the default; nonpositive TTL disables storage.
func NewWithCapacity(ttl time.Duration, capacity int) *Adapter {
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	return &Adapter{gate: make(chan struct{}, 1), ttl: ttl, capacity: capacity, now: time.Now, items: make(map[string]*list.Element)}
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
	if a.ttl <= 0 {
		return nil
	}
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
