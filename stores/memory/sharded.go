package memory

import (
	"context"
	"runtime"
	"time"

	memoize "github.com/agkloop/go_memoize"
	internalhash "github.com/agkloop/go_memoize/internal/hash"
)

// ShardedStore distributes keys across independent Store[K, V] instances
// using FNV-1a hashing to reduce mutex contention for distributed-key workloads.
type ShardedStore[K comparable, V any] struct {
	shards []Store[K, V]
	mask   uint64
}

// NewSharded creates a sharded store with total item capacity split across shards.
func NewSharded[K comparable, V any](capacity int, opts ...Option[K, V]) *ShardedStore[K, V] {
	if capacity <= 0 {
		panic("memory.NewSharded: capacity must be positive")
	}
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	shardCount := o.shards
	if shardCount == 0 {
		shardCount = nextPowerOfTwoInt(runtime.GOMAXPROCS(0) * 16)
	}
	if shardCount <= 0 || shardCount&(shardCount-1) != 0 {
		panic("memory.NewSharded: shard count must be a positive power of two")
	}
	for shardCount > capacity {
		shardCount >>= 1
	}
	if shardCount == 0 {
		shardCount = 1
	}

	shards := make([]Store[K, V], shardCount)
	for i := range shards {
		shardCap := capacity / shardCount
		if i < capacity%shardCount {
			shardCap++
		}
		shardOpts := o
		shardOpts.shards = 0
		if o.maxBytes > 0 {
			shardBytes := o.maxBytes / int64(shardCount)
			if i < int(o.maxBytes%int64(shardCount)) {
				shardBytes++
			}
			shardOpts.maxBytes = shardBytes
		}
		shards[i] = *New[K, V](shardCap, func(opts *options) { *opts = shardOpts })
	}
	return &ShardedStore[K, V]{shards: shards, mask: uint64(shardCount - 1)}
}

func (s *ShardedStore[K, V]) shard(key K) *Store[K, V] {
	h := shardHash(key)
	return &s.shards[(h>>16)&s.mask]
}

func (s *ShardedStore[K, V]) Get(ctx context.Context, key K) (memoize.Stored[V], bool, error) {
	return s.shard(key).Get(ctx, key)
}

func (s *ShardedStore[K, V]) Peek(ctx context.Context, key K) (memoize.Stored[V], bool, error) {
	return s.shard(key).Peek(ctx, key)
}

func (s *ShardedStore[K, V]) PeekFreshValue(ctx context.Context, key K, now time.Time) (V, bool, error) {
	return s.shard(key).PeekFreshValue(ctx, key, now)
}

func (s *ShardedStore[K, V]) Set(ctx context.Context, key K, value memoize.Stored[V]) error {
	return s.shard(key).Set(ctx, key, value)
}

func (s *ShardedStore[K, V]) Delete(ctx context.Context, key K) error {
	return s.shard(key).Delete(ctx, key)
}

func (s *ShardedStore[K, V]) Clear(ctx context.Context) error {
	for i := range s.shards {
		if err := s.shards[i].Clear(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Len returns total number of entries across all shards.
func (s *ShardedStore[K, V]) Len() int {
	total := 0
	for i := range s.shards {
		total += s.shards[i].Len()
	}
	return total
}

func (s *ShardedStore[K, V]) UsedBytes() int64 {
	var total int64
	for i := range s.shards {
		total += s.shards[i].UsedBytes()
	}
	return total
}

func (s *ShardedStore[K, V]) DeleteByTag(ctx context.Context, tag string) error {
	for i := range s.shards {
		if err := s.shards[i].DeleteByTag(ctx, tag); err != nil {
			return err
		}
	}
	return nil
}

func shardHash[K comparable](key K) uint64 {
	switch v := any(key).(type) {
	case string:
		return internalhash.String(internalhash.Offset64, v)
	case int:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case int8:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case int16:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case int32:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case int64:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case uint:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case uint8:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case uint16:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case uint32:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case uint64:
		return internalhash.Uint(internalhash.Offset64, v)
	case uintptr:
		return internalhash.Uint(internalhash.Offset64, uint64(v))
	case bool:
		return internalhash.Bool(internalhash.Offset64, v)
	default:
		panic("memory.NewSharded: unsupported key type")
	}
}

func nextPowerOfTwoInt(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	if unsafeIntSize == 64 {
		n |= n >> 32
	}
	return n + 1
}

const unsafeIntSize = 32 << (^uint(0) >> 63)
