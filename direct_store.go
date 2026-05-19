package memoize

import (
	"context"
	"sync"
	"time"
)

type directStore[V any] struct {
	mu      sync.RWMutex
	entries map[uint64]Stored[V]
}

func newDirectStore[V any]() *directStore[V] {
	return &directStore[V]{entries: make(map[uint64]Stored[V])}
}

func (s *directStore[V]) Get(ctx context.Context, key uint64) (Stored[V], bool, error) {
	s.mu.RLock()
	value, ok := s.entries[key]
	s.mu.RUnlock()
	return value, ok, nil
}

func (s *directStore[V]) PeekFreshValue(ctx context.Context, key uint64, now time.Time) (V, bool, error) {
	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok || entry.state(now) != entryFresh {
		var zero V
		return zero, false, nil
	}
	return entry.Value, true, nil
}

func (s *directStore[V]) Set(ctx context.Context, key uint64, value Stored[V]) error {
	s.mu.Lock()
	s.entries[key] = value
	s.mu.Unlock()
	return nil
}

func (s *directStore[V]) Delete(ctx context.Context, key uint64) error {
	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
	return nil
}

func (s *directStore[V]) Clear(ctx context.Context) error {
	s.mu.Lock()
	s.entries = make(map[uint64]Stored[V])
	s.mu.Unlock()
	return nil
}
