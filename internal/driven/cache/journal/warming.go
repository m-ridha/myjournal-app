package cachestore

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github/mridha/myjournal-app/internal/core/journal"
)

const (
	warmWorkers = 4
	// Includes active and queued dates; admission never waits on optional work.
	maxWarmDates = 64
	warmTimeout  = 10 * time.Second
)

type warmTask struct {
	key      string
	date     journal.Date
	account  string
	factory  func(context.Context) (journal.Store, error)
	revision uint64
}

type cacheWarmer struct {
	cache    *Cache
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	tasks    map[string]*warmTask
	jobs     chan *warmTask
	workers  sync.WaitGroup
	stopOnce sync.Once
}

// NewWithWarming enables best-effort read-through after successful mutations.
// The owner must call stop before closing the byte backend. Only wrappers created
// with WrapWithFactory warm; factories must construct clients with the supplied context.
func NewWithWarming(backend ByteCache, ttl time.Duration) (*Cache, func()) {
	cache := New(backend, ttl)
	ctx, cancel := context.WithCancel(context.Background())
	w := &cacheWarmer{cache: cache, ctx: ctx, cancel: cancel, tasks: make(map[string]*warmTask), jobs: make(chan *warmTask, maxWarmDates)}
	cache.warmer = w
	for range warmWorkers {
		w.workers.Go(w.run)
	}
	return cache, w.stop
}

// WrapWithFactory supplies fresh request-independent adapters for cache warming.
// The base adapter remains dedicated to the caller's current request.
func (c *Cache) WrapWithFactory(account string, base journal.Store, factory func(context.Context) (journal.Store, error)) journal.Store {
	return &Store{cache: c, account: account, base: base, factory: factory}
}

func (s *Store) warm(date journal.Date) {
	w := s.cache.warmer
	if w == nil || s.factory == nil || !s.cache.active(s.account) {
		return
	}
	key := (cacheKey{account: s.account, kind: "entry", date: date}).string()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx.Err() != nil {
		return
	}
	if task := w.tasks[key]; task != nil {
		// A later mutation needs a new attempt if an older read is still running.
		task.factory = s.factory
		task.revision++
		return
	}
	if len(w.tasks) >= maxWarmDates {
		return
	}
	task := &warmTask{key: key, date: date, account: s.account, factory: s.factory}
	w.tasks[key] = task
	w.jobs <- task // Capacity covers every admitted date, including active ones.
}

func (w *cacheWarmer) run() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case task := <-w.jobs:
			for w.ctx.Err() == nil {
				w.mu.Lock()
				factory, revision := task.factory, task.revision
				w.mu.Unlock()
				ctx, cancel := context.WithTimeout(w.ctx, warmTimeout)
				base, err := factory(ctx)
				if err == nil && base != nil {
					// A new operation must not reuse the mutation's discovery or OAuth context.
					if scoped, ok := base.(journal.OperationStore); ok {
						base = scoped.ForOperation()
					}
					_, err = w.cache.Wrap(task.account, base).Get(ctx, task.date)
				}
				cancel()
				if err != nil && !errors.Is(err, journal.ErrNotFound) {
					slog.Debug("journal cache warming incomplete")
				}
				w.mu.Lock()
				again := task.revision != revision && w.ctx.Err() == nil
				if !again {
					delete(w.tasks, task.key)
				}
				w.mu.Unlock()
				if !again {
					break
				}
			}
		}
	}
}

func (w *cacheWarmer) stop() {
	w.stopOnce.Do(func() {
		w.cancel()
		w.workers.Wait()
		w.mu.Lock()
		defer w.mu.Unlock()
		clear(w.tasks)
		for {
			select {
			case <-w.jobs:
			default:
				return
			}
		}
	})
}
