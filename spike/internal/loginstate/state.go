// Package loginstate bounds pending OAuth transactions in historical spikes.
package loginstate

import (
	"sync"
	"time"
)

const (
	lifetime = 10 * time.Minute
	capacity = 128
)

type pending struct {
	verifier string
	expires  time.Time
}

// Store has a usable zero value and consumes each transaction at most once.
type Store struct {
	mu    sync.Mutex
	items map[string]pending
}

// Add drops expired transactions and rejects new ones at capacity.
func (s *Store) Add(state, verifier string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, item := range s.items {
		if !now.Before(item.expires) {
			delete(s.items, key)
		}
	}
	if len(s.items) >= capacity {
		return false
	}
	if s.items == nil {
		s.items = make(map[string]pending)
	}
	s.items[state] = pending{verifier: verifier, expires: now.Add(lifetime)}
	return true
}

// Take consumes a transaction, including expired transactions.
func (s *Store) Take(state string, now time.Time) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[state]
	delete(s.items, state)
	if !ok || !now.Before(item.expires) {
		return "", false
	}
	return item.verifier, true
}
