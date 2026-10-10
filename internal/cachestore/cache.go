package cachestore

import (
	"container/list"
	"context"
	"slices"
	"sync"
	"time"

	"github/mridha/myjournal-app/internal/journal"

	"cloud.google.com/go/civil"
)

const defaultMaxEntries = 2048

type cacheKey struct {
	account string
	kind    string
	year    int
	date    civil.Date
	maxSize int
}

type cacheItem struct {
	key       cacheKey
	value     any
	expiresAt time.Time
}

// Cache is a bounded, process-wide cache shared by per-account Store wrappers.
type Cache struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	now        func() time.Time
	items      map[cacheKey]*list.Element
	lru        list.List
	generation uint64
}

// Store decorates a journal.Store with account-scoped cached indexes and previews.
type Store struct {
	cache   *Cache
	account string
	base    journal.Store
}

// New constructs a cache with the default entry bound.
func New(ttl time.Duration) *Cache { return NewWithCapacity(ttl, defaultMaxEntries) }

// NewWithCapacity constructs a cache with an explicit entry bound.
func NewWithCapacity(ttl time.Duration, capacity int) *Cache {
	if capacity <= 0 {
		capacity = defaultMaxEntries
	}

	return &Cache{
		ttl:        ttl,
		maxEntries: capacity,
		now:        time.Now,
		items:      make(map[cacheKey]*list.Element),
	}
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
	value, generation, ok := s.cache.get(key)
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
	s.cache.set(key, append([]int(nil), items...), generation)
	return append([]int(nil), items...), nil
}

func (s *Store) ListYear(ctx context.Context, year int) ([]journal.EntryMeta, error) {
	key := cacheKey{account: s.account, kind: "year", year: year}
	value, generation, ok := s.cache.get(key)
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
	s.cache.set(key, append([]journal.EntryMeta(nil), items...), generation)
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
		if value, generation, ok := s.cache.get(key); ok {
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
		s.cache.set(key, v, generations[d])
	}
	return out, nil
}

func (s *Store) Put(ctx context.Context, d civil.Date, text string, base journal.Version) (journal.Entry, error) {
	entry, err := s.base.Put(ctx, d, text, base)
	// Failed uploads can be ambiguous; invalidate affected data without retrying.
	s.cache.invalidateDate(s.account, d, err != nil || base == "")
	if err != nil {
		return journal.Entry{}, err
	}
	return entry, nil
}

func (s *Store) Trash(ctx context.Context, d civil.Date) error {
	err := s.base.Trash(ctx, d)
	s.cache.invalidateDate(s.account, d, true)
	return err
}

func (c *Cache) get(key cacheKey) (any, uint64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	generation := c.generation
	if c.ttl <= 0 || key.account == "" {
		return nil, generation, false
	}
	element, ok := c.items[key]
	if !ok {
		return nil, generation, false
	}
	item := element.Value.(*cacheItem)
	if !c.now().Before(item.expiresAt) {
		c.lru.Remove(element)
		delete(c.items, key)
		return nil, generation, false
	}
	c.lru.MoveToFront(element)
	return item.value, generation, true
}

func (c *Cache) set(key cacheKey, value any, generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ttl <= 0 || key.account == "" || c.generation != generation {
		return
	}
	if element, ok := c.items[key]; ok {
		item := element.Value.(*cacheItem)
		item.value = value
		item.expiresAt = c.now().Add(c.ttl)
		c.lru.MoveToFront(element)
		return
	}
	item := &cacheItem{key: key, value: value, expiresAt: c.now().Add(c.ttl)}
	c.items[key] = c.lru.PushFront(item)
	for len(c.items) > c.maxEntries {
		oldest := c.lru.Back()
		delete(c.items, oldest.Value.(*cacheItem).key)
		c.lru.Remove(oldest)
	}
}

// invalidateDate retains unrelated snapshots with their original expiry. It never
// publishes acknowledgement text, which could arrive out of order across writers.
func (c *Cache) invalidateDate(account string, date civil.Date, membershipMayChange bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Keep the bounded epoch: even targeted invalidation rejects older in-flight fills.
	c.generation++
	retainYearIndex, retainYears := false, false
	if !membershipMayChange {
		if element := c.items[cacheKey{account: account, kind: "years"}]; element != nil {
			item := element.Value.(*cacheItem)
			retainYears = c.now().Before(item.expiresAt) && slices.Contains(item.value.([]int), date.Year)
		}
		key := cacheKey{account: account, kind: "year", year: date.Year}
		if element := c.items[key]; element != nil {
			item := element.Value.(*cacheItem)
			if c.now().Before(item.expiresAt) {
				for _, meta := range item.value.([]journal.EntryMeta) {
					if meta.Date == date {
						retainYearIndex = true
						break
					}
				}
			}
		}
	}
	for key, element := range c.items {
		if key.account != account {
			continue
		}
		affected := key.kind == "preview" && key.date == date
		if (key.kind == "years" && !retainYears) || (key.kind == "year" && key.year == date.Year && !retainYearIndex) {
			affected = true
		}
		if affected {
			c.lru.Remove(element)
			delete(c.items, key)
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

var _ journal.OperationStore = (*Store)(nil)
