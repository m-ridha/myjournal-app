package cachestore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/civil"
	"golang.org/x/sync/errgroup"

	"github/mridha/myjournal-app/internal/core/journal"
)

// concurrent preview reads reduce cold-page waves while keeping I/O bounded.
const previewWorkers = 10

type cacheKey struct {
	account string
	kind    string
	year    int
	date    civil.Date
	maxSize int
}

// Cache coordinates disposable snapshots and overlapping reads across Store wrappers.
// Generation guards coordinate only this process, even with a remote ByteCache.
type Cache struct {
	gate           chan struct{}
	backend        ByteCache
	ttl            time.Duration
	now            func() time.Time
	generation     uint64
	disabled       atomic.Bool
	reads          readGroup
	previewBatches readGroup
	readEpochs     [64]atomic.Uint64
}

// Store decorates a journal.Store with account-scoped cached indexes and previews.
type Store struct {
	cache   *Cache
	account string
	base    journal.Store
}

var (
	_ journal.Store          = (*Store)(nil)
	_ journal.OperationStore = (*Store)(nil)
)

// New constructs a shared decorator coordinator. Nil storage or nonpositive TTL
// disables snapshots; bounded in-flight read sharing remains enabled.
func New(backend ByteCache, ttl time.Duration) *Cache {
	return &Cache{backend: backend, ttl: ttl, now: time.Now, gate: make(chan struct{}, 1)}
}

// Wrap shares snapshots and overlapping reads by provider/account. The caller
// must validate the connection and include the provider in the account namespace.
func (c *Cache) Wrap(account string, base journal.Store) journal.Store {
	return &Store{cache: c, account: account, base: base}
}

func (s *Store) ListYears(ctx context.Context) ([]int, error) {
	key := cacheKey{account: s.account, kind: "years"}
	items, err := cachedRead(ctx, s, key, func(ctx context.Context) ([]int, error) { return s.base.ListYears(ctx) })
	return append([]int(nil), items...), err
}

func (s *Store) ListYear(ctx context.Context, year int) ([]journal.EntryMeta, error) {
	key := cacheKey{account: s.account, kind: "year", year: year}
	items, err := cachedRead(ctx, s, key, func(ctx context.Context) ([]journal.EntryMeta, error) { return s.base.ListYear(ctx, year) })
	return append([]journal.EntryMeta(nil), items...), err
}

func (s *Store) Get(ctx context.Context, d civil.Date) (journal.Entry, error) {
	return sharedRead(ctx, s, cacheKey{account: s.account, kind: "entry", date: d}, func(ctx context.Context) (journal.Entry, error) { return s.base.Get(ctx, d) })
}

func (s *Store) Previews(ctx context.Context, dates []civil.Date, maxBytes int) (map[civil.Date]string, error) {
	if s.account == "" || maxBytes <= 0 {
		return s.base.Previews(ctx, dates, maxBytes)
	}
	ordered := append([]civil.Date(nil), dates...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Before(ordered[j]) })
	ordered = slices.Compact(ordered)
	digest := sha256.Sum256(fmt.Append(nil, ordered))
	key := fmt.Sprintf("%s:%x:%d:%d", namespace(s.account), digest, maxBytes, s.cache.readEpoch(s.account).Load())
	// One batch keeps identical requests in one operation view; individual flights
	// inside it also share dates with different overlapping batches.
	value, err := s.cache.previewBatches.do(ctx, key, func(ctx context.Context) (any, error) { return s.previews(ctx, ordered, maxBytes) })
	if err != nil {
		return nil, err
	}
	return maps.Clone(value.(map[civil.Date]string)), nil
}

func (s *Store) previews(ctx context.Context, dates []civil.Date, maxBytes int) (map[civil.Date]string, error) {
	if s.account == "" || maxBytes <= 0 {
		return s.base.Previews(ctx, dates, maxBytes)
	}
	out := make(map[civil.Date]string, len(dates))
	var mu sync.Mutex
	group, ctx := errgroup.WithContext(ctx)
	// Keep provider work bounded while sharing individual dates across batches.
	group.SetLimit(previewWorkers)
	seen := make(map[civil.Date]bool, len(dates))
	for _, date := range dates {
		if seen[date] {
			continue
		}
		seen[date] = true
		group.Go(func() error {
			key := cacheKey{account: s.account, kind: "preview", date: date, maxSize: maxBytes}
			value, err := cachedRead(ctx, s, key, func(ctx context.Context) (previewResult, error) {
				previews, err := s.base.Previews(ctx, []civil.Date{date}, maxBytes)
				text, exists := previews[date]
				return previewResult{text: text, exists: exists}, err
			})
			if err != nil {
				return err
			}
			if value.exists {
				mu.Lock()
				out[date] = value.text
				mu.Unlock()
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

type previewResult struct {
	text   string
	exists bool
}

// cachedRead fills snapshots inside the flight, before its result is published.
func cachedRead[T any](ctx context.Context, s *Store, key cacheKey, load func(context.Context) (T, error)) (T, error) {
	value, _, ok := s.cache.get(ctx, key)
	if ok {
		if key.kind == "preview" {
			return any(previewResult{text: value.(string), exists: true}).(T), nil
		}
		return value.(T), nil
	}
	return sharedRead(ctx, s, key, func(ctx context.Context) (T, error) {
		// A prior flight may have filled the cache between the miss and admission.
		value, generation, ok := s.cache.get(ctx, key)
		if ok {
			if key.kind == "preview" {
				return any(previewResult{text: value.(string), exists: true}).(T), nil
			}
			return value.(T), nil
		}
		items, err := load(ctx)
		if err == nil {
			if key.kind == "preview" {
				preview := any(items).(previewResult)
				if preview.exists {
					s.cache.set(ctx, key, preview.text, generation)
				}
			} else {
				s.cache.set(ctx, key, items, generation)
			}
		}
		return items, err
	})
}

func sharedRead[T any](ctx context.Context, s *Store, key cacheKey, load func(context.Context) (T, error)) (T, error) {
	if s.account == "" {
		return load(ctx)
	}
	flightKey := fmt.Sprintf("%s:%s:%d", key.kind, key.string(), s.cache.readEpoch(s.account).Load())
	value, err := s.cache.reads.do(ctx, flightKey, func(ctx context.Context) (any, error) { return load(ctx) })
	if err != nil {
		var zero T
		return zero, err
	}
	return value.(T), nil
}

func (c *Cache) readEpoch(account string) *atomic.Uint64 {
	hash := sha256.Sum256([]byte(account))
	return &c.readEpochs[int(hash[0])%len(c.readEpochs)]
}

func (s *Store) Put(ctx context.Context, d civil.Date, text string, base journal.Version) (journal.Entry, error) {
	epoch := s.cache.readEpoch(s.account)
	epoch.Add(1)
	defer epoch.Add(1)
	entry, err := s.base.Put(ctx, d, text, base)
	// Failed uploads can be ambiguous; invalidate affected data without retrying.
	s.cache.invalidateDate(ctx, s.account, d, err != nil || base == "")
	if err != nil {
		return journal.Entry{}, err
	}
	return entry, nil
}

func (s *Store) Trash(ctx context.Context, d civil.Date) error {
	epoch := s.cache.readEpoch(s.account)
	epoch.Add(1)
	defer epoch.Add(1)
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
