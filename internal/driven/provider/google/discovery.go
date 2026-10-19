package google

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
	"google.golang.org/api/drive/v3"

	"myjournal-app/internal/core/journal"
)

const (
	maxDiscoveryQueries = 16
	maxDiscoveryFiles   = 1000
)

type operationDiscovery struct {
	mu         sync.Mutex
	queries    map[string][]*drive.File
	loads      singleflight.Group
	generation uint64
}

// ForOperation owns bounded metadata reuse for one service call, never a request
// after it. Content downloads and pre-upload metadata checks are always fresh.
func (s *Store) ForOperation() journal.Store {
	return &Store{service: s.service, lock: s.lock, discovery: &operationDiscovery{queries: make(map[string][]*drive.File)}}
}

func (s *Store) list(ctx context.Context, query string) ([]*drive.File, error) {
	if s.discovery == nil {
		return s.listFresh(ctx, query)
	}
	d := s.discovery
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Reuse is operation-local. Do not hold the metadata mutex across I/O:
	// the generic decorator can request different preview dates concurrently.
	d.mu.Lock()
	generation := d.generation
	d.mu.Unlock()
	value, err, _ := d.loads.Do(fmt.Sprintf("%d:%s", generation, query), func() (any, error) {
		d.mu.Lock()
		files, ok := d.queries[query]
		d.mu.Unlock()
		if ok {
			return files, nil
		}
		files, err := s.listFresh(ctx, query)
		if err == nil && len(files) <= maxDiscoveryFiles {
			d.mu.Lock()
			if d.generation == generation && len(d.queries) < maxDiscoveryQueries {
				d.queries[query] = files
			}
			d.mu.Unlock()
		}
		return files, err
	})
	if err != nil {
		return nil, err
	}
	return value.([]*drive.File), nil
}

func (s *Store) clearDiscovery() {
	if s.discovery == nil {
		return
	}
	s.discovery.mu.Lock()
	s.discovery.generation++
	clear(s.discovery.queries)
	s.discovery.mu.Unlock()
}
