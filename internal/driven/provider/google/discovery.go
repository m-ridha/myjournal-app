package google

import (
	"context"
	"sync"

	"google.golang.org/api/drive/v3"

	"github/mridha/myjournal-app/internal/core/journal"
)

const (
	maxDiscoveryQueries = 16
	maxDiscoveryFiles   = 1000
)

type operationDiscovery struct {
	mu      sync.Mutex
	queries map[string][]*drive.File
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
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if files, ok := d.queries[query]; ok {
		return files, nil
	}
	files, err := s.listFresh(ctx, query)
	if err == nil && len(d.queries) < maxDiscoveryQueries && len(files) <= maxDiscoveryFiles {
		d.queries[query] = files
	}
	return files, err
}

func (s *Store) clearDiscovery() {
	if s.discovery == nil {
		return
	}
	s.discovery.mu.Lock()
	clear(s.discovery.queries)
	s.discovery.mu.Unlock()
}

var _ journal.OperationStore = (*Store)(nil)
