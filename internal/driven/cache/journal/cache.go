package cachestore

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"time"

	"cloud.google.com/go/civil"

	"github/mridha/myjournal-app/internal/core/journal"
)

type cacheKey struct {
	account string
	kind    string
	year    int
	date    civil.Date
	maxSize int
}

// Cache coordinates disposable snapshots shared by account-scoped Store wrappers.
// Generation guards coordinate only this process, even with a remote ByteCache.
type Cache struct {
	gate       chan struct{}
	backend    ByteCache
	ttl        time.Duration
	now        func() time.Time
	generation uint64
	disabled   atomic.Bool
}

// Store decorates a journal.Store with account-scoped cached indexes and previews.
type Store struct {
	cache   *Cache
	account string
	base    journal.Store
}

var _ journal.OperationStore = (*Store)(nil)

// New constructs a shared decorator coordinator. Nil storage or nonpositive TTL
// disables caching without affecting the backing journal store.
func New(backend ByteCache, ttl time.Duration) *Cache {
	return &Cache{backend: backend, ttl: ttl, now: time.Now, gate: make(chan struct{}, 1)}
}

func (c *Cache) Wrap(account string, base journal.Store) journal.Store {
	return &Store{
		cache:   c,
		account: account,
		base:    base,
	}
}

func (s *Store) ListYears(ctx context.Context) ([]int, error) {
	key := cacheKey{account: s.account, kind: "years"}
	value, generation, ok := s.cache.get(ctx, key)
	if ok {
		return append([]int(nil), value.([]int)...), nil
	}
	if s.cache.ttl <= 0 || s.account == "" {
		return s.base.ListYears(ctx)
	}
	items, err := s.base.ListYears(ctx)
	if err != nil {
		return nil, err
	}
	s.cache.set(ctx, key, append([]int(nil), items...), generation)
	return append([]int(nil), items...), nil
}

func (s *Store) ListYear(ctx context.Context, year int) ([]journal.EntryMeta, error) {
	key := cacheKey{account: s.account, kind: "year", year: year}
	value, generation, ok := s.cache.get(ctx, key)
	if ok {
		return append([]journal.EntryMeta(nil), value.([]journal.EntryMeta)...), nil
	}
	if s.cache.ttl <= 0 || s.account == "" {
		return s.base.ListYear(ctx, year)
	}
	items, err := s.base.ListYear(ctx, year)
	if err != nil {
		return nil, err
	}
	s.cache.set(ctx, key, append([]journal.EntryMeta(nil), items...), generation)
	return append([]journal.EntryMeta(nil), items...), nil
}

func (s *Store) Get(ctx context.Context, d civil.Date) (journal.Entry, error) {
	return s.base.Get(ctx, d)
}

func (s *Store) Previews(ctx context.Context, dates []civil.Date, maxBytes int) (map[civil.Date]string, error) {
	out := make(map[civil.Date]string, len(dates))
	misses := make([]civil.Date, 0, len(dates))
	generations := make(map[civil.Date]uint64, len(dates))

	for _, d := range dates {
		key := cacheKey{account: s.account, kind: "preview", date: d, maxSize: maxBytes}
		if value, generation, ok := s.cache.get(ctx, key); ok {
			out[d] = value.(string)
		} else {
			misses = append(misses, d)
			generations[d] = generation
		}
	}

	if len(misses) == 0 {
		return out, nil
	}

	baseOut, err := s.base.Previews(ctx, misses, maxBytes)
	if err != nil {
		return nil, err
	}

	for d, v := range baseOut {
		out[d] = v
		key := cacheKey{account: s.account, kind: "preview", date: d, maxSize: maxBytes}
		s.cache.set(ctx, key, v, generations[d])
	}
	return out, nil
}

func (s *Store) Put(ctx context.Context, d civil.Date, text string, base journal.Version) (journal.Entry, error) {
	entry, err := s.base.Put(ctx, d, text, base)
	// Failed uploads can be ambiguous; invalidate affected data without retrying.
	s.cache.invalidateDate(ctx, s.account, d, err != nil || base == "")
	if err != nil {
		return journal.Entry{}, err
	}
	return entry, nil
}

func (s *Store) Trash(ctx context.Context, d civil.Date) error {
	err := s.base.Trash(ctx, d)
	s.cache.invalidateDate(ctx, s.account, d, true)
	return err
}

func (c *Cache) lock(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return false
	}
	if ctx.Err() != nil {
		c.unlock()
		return false
	}
	return true
}
func (c *Cache) unlock() { <-c.gate }

func (c *Cache) active(account string) bool {
	return c.backend != nil && c.ttl > 0 && account != "" && !c.disabled.Load()
}

func (c *Cache) get(ctx context.Context, key cacheKey) (any, uint64, bool) {
	if !c.lock(ctx) {
		return nil, 0, false
	}
	defer c.unlock()
	generation := c.generation
	value, ok := c.read(ctx, key)
	return value, generation, ok
}

// read requires the coordinator lock; it never refreshes a snapshot's expiry.
func (c *Cache) read(ctx context.Context, key cacheKey) (any, bool) {
	if !c.active(key.account) {
		return nil, false
	}
	data, err := c.backend.Get(ctx, key.string())
	if errors.Is(err, ErrMiss) {
		return nil, false
	}
	if err != nil {
		c.disabled.Store(true)
		return nil, false
	}
	value, err := decodeSnapshot(data, key.kind, c.now())
	if err != nil {
		c.disabled.Store(true)
		return nil, false
	}
	return value, value != nil
}

func (c *Cache) set(ctx context.Context, key cacheKey, value any, generation uint64) {
	if !c.lock(ctx) {
		return
	}
	defer c.unlock()
	if !c.active(key.account) || c.generation != generation {
		return
	}
	data, err := encodeSnapshot(key.kind, value, c.now().Add(c.ttl))
	if err == nil {
		err = c.backend.Set(ctx, key.string(), data)
	}
	if err != nil && !errors.Is(err, ErrCapacity) {
		c.disabled.Store(true)
	}
}

// invalidateDate preserves unrelated snapshots without renewing their expiry.
// Failed deletion disables reuse, but never changes a backing mutation's result.
func (c *Cache) invalidateDate(ctx context.Context, account string, date civil.Date, membershipMayChange bool) {
	if !c.lock(ctx) {
		c.disabled.Store(true)
		return
	}
	defer c.unlock()
	c.generation++
	if !c.active(account) {
		return
	}
	yearsKey := cacheKey{account: account, kind: "years"}
	yearKey := cacheKey{account: account, kind: "year", year: date.Year}
	retainYears, retainYear := false, false
	if !membershipMayChange {
		if value, ok := c.read(ctx, yearsKey); ok {
			retainYears = slices.Contains(value.([]int), date.Year)
		}
		if value, ok := c.read(ctx, yearKey); ok {
			for _, meta := range value.([]journal.EntryMeta) {
				if meta.Date == date {
					retainYear = true
					break
				}
			}
		}
	}
	if !c.active(account) {
		return
	}
	if err := c.backend.DeletePrefix(ctx, previewPrefix(account, date)); err != nil {
		c.disabled.Store(true)
		return
	}
	for _, item := range []struct {
		key    cacheKey
		retain bool
	}{{yearsKey, retainYears}, {yearKey, retainYear}} {
		if !item.retain {
			if err := c.backend.Delete(ctx, item.key.string()); err != nil {
				c.disabled.Store(true)
				return
			}
		}
	}
}

// ForOperation forwards scoped reuse without changing shared cache ownership.
func (s *Store) ForOperation() journal.Store {
	base := s.base
	if scoped, ok := base.(journal.OperationStore); ok {
		base = scoped.ForOperation()
	}
	return s.cache.Wrap(s.account, base)
}
