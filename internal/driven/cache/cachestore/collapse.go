package cachestore

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

const (
	maxReadFlights    = 64
	maxReadCallers    = 320
	readFlightTimeout = 30 * time.Second
)

// singleflight owns duplicate suppression and result publication. This wrapper
// bounds both distinct keys and registrations (including cancelled waiters).
type readGroup struct {
	mu      sync.Mutex
	group   singleflight.Group
	keys    map[string]int
	callers int
}

func (g *readGroup) do(ctx context.Context, key string, load func(context.Context) (any, error)) (any, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		g.mu.Lock()
		if g.callers >= maxReadCallers || (g.keys[key] == 0 && len(g.keys) >= maxReadFlights) {
			g.mu.Unlock()
			return nil, journal.ErrUnavailable
		}
		if g.keys == nil {
			g.keys = make(map[string]int)
		}
		g.keys[key]++
		g.callers++
		leader := false
		result := g.group.DoChan(key, func() (any, error) {
			leader = true
			return runRead(ctx, load)
		})
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			// DoChan retains its result channel until completion. Hold admission until
			// it is drained, so cancellation cannot create unbounded registrations.
			go func() { <-result; g.release(key) }()
			return nil, ctx.Err()
		case out := <-result:
			if !leader && errors.Is(out.Err, journal.ErrAuthRequired) && ctx.Err() == nil {
				// Each device verifies with its own authorization once. Hold its
				// admission slot through the retry and do not rejoin bad grants.
				defer g.release(key)
				return runRead(ctx, load)
			}
			g.release(key)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// A surviving request retries using its own Store/OAuth context.
			if !leader && (errors.Is(out.Err, context.Canceled) || errors.Is(out.Err, context.DeadlineExceeded)) {
				continue
			}
			return out.Val, out.Err
		}
	}
}
func (g *readGroup) release(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.callers--
	g.keys[key]--
	if g.keys[key] == 0 {
		delete(g.keys, key)
	}
}

func runRead(ctx context.Context, load func(context.Context) (any, error)) (any, error) {
	work, cancel := context.WithTimeout(ctx, readFlightTimeout)
	defer cancel()
	value, err := load(work)
	if work.Err() != nil {
		return nil, work.Err()
	}
	return value, err
}
