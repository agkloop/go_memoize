package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	memoize "github.com/agkloop/go_memoize"
)

type singleEntry[K comparable, V any] struct {
	key   K
	value memoize.Stored[V]
	cost  int64
}

// SingleStore stores one key/value pair with atomic read access.
// It is intended for one logical cached value where LRU bookkeeping is unnecessary.
type SingleStore[K comparable, V any] struct {
	mu    sync.Mutex
	entry atomic.Pointer[singleEntry[K, V]]
}

func NewSingle[K comparable, V any]() *SingleStore[K, V] {
	return &SingleStore[K, V]{}
}

func (s *SingleStore[K, V]) Get(_ context.Context, key K) (memoize.Stored[V], bool, error) {
	return s.load(key)
}

func (s *SingleStore[K, V]) Peek(_ context.Context, key K) (memoize.Stored[V], bool, error) {
	return s.load(key)
}

func (s *SingleStore[K, V]) PeekFreshValue(_ context.Context, key K, now time.Time) (V, bool, error) {
	e := s.entry.Load()
	if e == nil || e.key != key {
		var zero V
		return zero, false, nil
	}
	entry := &e.value
	if entry.NoExpire || now.Before(entry.FreshUntil) || now.Equal(entry.FreshUntil) {
		return entry.Value, true, nil
	}
	var zero V
	return zero, false, nil
}

func (s *SingleStore[K, V]) load(key K) (memoize.Stored[V], bool, error) {
	e := s.entry.Load()
	if e == nil || e.key != key {
		var zero memoize.Stored[V]
		return zero, false, nil
	}
	return e.value, true, nil
}

func (s *SingleStore[K, V]) Set(_ context.Context, key K, value memoize.Stored[V]) error {
	s.mu.Lock()
	s.entry.Store(&singleEntry[K, V]{key: key, value: value, cost: singleEntryBytes(key, value)})
	s.mu.Unlock()
	return nil
}

func (s *SingleStore[K, V]) Delete(_ context.Context, key K) error {
	s.mu.Lock()
	if e := s.entry.Load(); e != nil && e.key == key {
		s.entry.Store(nil)
	}
	s.mu.Unlock()
	return nil
}

func (s *SingleStore[K, V]) Clear(context.Context) error {
	s.mu.Lock()
	s.entry.Store(nil)
	s.mu.Unlock()
	return nil
}

func (s *SingleStore[K, V]) Len() int {
	if s.entry.Load() == nil {
		return 0
	}
	return 1
}

func (s *SingleStore[K, V]) UsedBytes() int64 {
	e := s.entry.Load()
	if e == nil {
		return 0
	}
	return e.cost
}

func (s *SingleStore[K, V]) DeleteByTag(_ context.Context, tag string) error {
	s.mu.Lock()
	if e := s.entry.Load(); e != nil && hasTag(e.value.Tags, tag) {
		s.entry.Store(nil)
	}
	s.mu.Unlock()
	return nil
}

func singleEntryBytes[K comparable, V any](key K, value memoize.Stored[V]) int64 {
	return int64(unsafe.Sizeof(key)) + int64(unsafe.Sizeof(value))
}
