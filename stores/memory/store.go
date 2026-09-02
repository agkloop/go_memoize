package memory

import (
	"context"
	"math"
	"sync"
	"time"
	"unsafe"

	memoize "github.com/agkloop/go_memoize"
)

const emptyIndex = uint32(math.MaxUint32)

type element[K comparable, V any] struct {
	key   K
	value memoize.Stored[V]
	cost  int64

	next uint32
	prev uint32
}

type Store[K comparable, V any] struct {
	mu        sync.Mutex
	index     map[K]uint32
	elements  []element[K, V]
	head      uint32
	len       uint32
	cap       uint32
	usedBytes int64
	getHits   uint64
	opts      options
}

func New[K comparable, V any](capacity int, opts ...Option[K, V]) *Store[K, V] {
	if capacity <= 0 {
		panic("memory.New: capacity must be positive")
	}
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.getRecencySample == 0 {
		o.getRecencySample = 1
	}
	cap32 := uint32(capacity)
	return &Store[K, V]{
		index:    make(map[K]uint32, capacity),
		elements: make([]element[K, V], cap32),
		head:     emptyIndex,
		cap:      cap32,
		opts:     o,
	}
}

func (s *Store[K, V]) Get(_ context.Context, key K) (memoize.Stored[V], bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos, ok := s.index[key]
	if !ok {
		var zero memoize.Stored[V]
		return zero, false, nil
	}
	if s.refreshOnGet() {
		s.moveToFront(pos)
	}
	return s.elements[pos].value, true, nil
}

func (s *Store[K, V]) Peek(_ context.Context, key K) (memoize.Stored[V], bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos, ok := s.index[key]
	if !ok {
		var zero memoize.Stored[V]
		return zero, false, nil
	}
	return s.elements[pos].value, true, nil
}

func (s *Store[K, V]) PeekFreshValue(_ context.Context, key K, now time.Time) (V, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos, ok := s.index[key]
	if !ok {
		var zero V
		return zero, false, nil
	}
	entry := &s.elements[pos].value
	if entry.NoExpire || now.Before(entry.FreshUntil) || now.Equal(entry.FreshUntil) {
		if s.refreshOnGet() {
			s.moveToFront(pos)
		}
		return entry.Value, true, nil
	}
	var zero V
	return zero, false, nil
}

func (s *Store[K, V]) refreshOnGet() bool {
	if s.opts.getRecencySample <= 1 {
		return true
	}
	s.getHits++
	return s.getHits%uint64(s.opts.getRecencySample) == 0
}

func (s *Store[K, V]) Set(_ context.Context, key K, value memoize.Stored[V]) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cost := entryBytes(key, value)
	if pos, ok := s.index[key]; ok {
		s.usedBytes += cost - s.elements[pos].cost
		s.elements[pos].value = value
		s.elements[pos].cost = cost
		s.moveToFront(pos)
		s.enforceBytes()
		return nil
	}

	for s.len == s.cap || (s.opts.maxBytes > 0 && s.usedBytes+cost > s.opts.maxBytes && s.len > 0) {
		s.removeAt(s.elements[s.head].next)
	}

	pos := s.len
	s.len++
	s.elements[pos] = element[K, V]{key: key, value: value, cost: cost, next: emptyIndex, prev: emptyIndex}
	s.index[key] = pos
	s.linkFront(pos)
	s.usedBytes += cost
	return nil
}

func (s *Store[K, V]) enforceBytes() {
	for s.opts.maxBytes > 0 && s.usedBytes > s.opts.maxBytes && s.len > 1 {
		s.removeAt(s.elements[s.head].next)
	}
}

// entryBytes returns a shallow estimate of the memory cost of one cache entry.
// Heap allocations inside K or V (e.g. string content, slice backing arrays) are not counted.
func entryBytes[K comparable, V any](key K, value memoize.Stored[V]) int64 {
	return int64(unsafe.Sizeof(key)) + int64(unsafe.Sizeof(value))
}

func (s *Store[K, V]) Delete(_ context.Context, key K) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pos, ok := s.index[key]; ok {
		s.removeAt(pos)
	}
	return nil
}

func (s *Store[K, V]) Clear(_ context.Context) error {
	s.mu.Lock()
	for i := uint32(0); i < s.len; i++ {
		s.elements[i] = element[K, V]{}
	}
	s.index = make(map[K]uint32, s.cap)
	s.head = emptyIndex
	s.len = 0
	s.usedBytes = 0
	s.mu.Unlock()
	return nil
}

// Len returns the number of items currently in the store.
func (s *Store[K, V]) Len() int {
	s.mu.Lock()
	n := s.len
	s.mu.Unlock()
	return int(n)
}

// UsedBytes returns the current estimated byte usage of the store.
func (s *Store[K, V]) UsedBytes() int64 {
	s.mu.Lock()
	n := s.usedBytes
	s.mu.Unlock()
	return n
}

// DeleteByTag removes all entries that have tag in their Tags slice.
func (s *Store[K, V]) DeleteByTag(_ context.Context, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for pos := uint32(0); pos < s.len; {
		if hasTag(s.elements[pos].value.Tags, tag) {
			s.removeAt(pos)
			continue
		}
		pos++
	}
	return nil
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (s *Store[K, V]) linkFront(pos uint32) {
	if s.head == emptyIndex {
		s.head = pos
		s.elements[pos].next = pos
		s.elements[pos].prev = pos
		return
	}
	tail := s.elements[s.head].next
	s.elements[pos].next = tail
	s.elements[pos].prev = s.head
	s.elements[tail].prev = pos
	s.elements[s.head].next = pos
	s.head = pos
}

func (s *Store[K, V]) unlinkLRU(pos uint32) {
	if s.elements[pos].next == pos {
		s.head = emptyIndex
		return
	}
	next := s.elements[pos].next
	prev := s.elements[pos].prev
	s.elements[next].prev = prev
	s.elements[prev].next = next
	if s.head == pos {
		s.head = prev
	}
}

func (s *Store[K, V]) moveToFront(pos uint32) {
	if s.head == pos || s.head == emptyIndex {
		return
	}
	s.unlinkLRU(pos)
	s.linkFront(pos)
}

func (s *Store[K, V]) removeAt(pos uint32) {
	removedKey := s.elements[pos].key
	s.usedBytes -= s.elements[pos].cost
	s.unlinkLRU(pos)
	delete(s.index, removedKey)

	last := s.len - 1
	s.len--
	if pos != last {
		s.elements[pos] = s.elements[last]
		s.index[s.elements[pos].key] = pos
		s.repointMoved(last, pos)
	}
	s.elements[last] = element[K, V]{}
}

func (s *Store[K, V]) repointMoved(oldPos, newPos uint32) {
	moved := &s.elements[newPos]
	if moved.next == oldPos {
		moved.next = newPos
	} else {
		s.elements[moved.next].prev = newPos
	}
	if moved.prev == oldPos {
		moved.prev = newPos
	} else {
		s.elements[moved.prev].next = newPos
	}
	if s.head == oldPos {
		s.head = newPos
	}
}
