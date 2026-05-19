package benchmarks

import (
	"context"
	"time"

	memoize "github.com/agkloop/go_memoize"
	internalhash "github.com/agkloop/go_memoize/internal/hash"
	"github.com/agkloop/go_memoize/stores/memory"
)

var benchLRUClock = memoize.ClockFunc(func() time.Time { return time.Unix(1, 0) })

func newLRUCache[V any](ttl time.Duration, capacity int) *memoize.Cache[uint64, V] {
	cache, err := memoize.New[uint64, V](
		memoize.Opts().WithStore(memory.New[uint64, V](capacity)).WithTTL(ttl).WithClock(benchLRUClock),
	)
	if err != nil {
		panic(err)
	}
	return cache
}

func lruMemoize[V any](computeFn func() V, ttl time.Duration, capacity int) func() V {
	ctx := context.Background()
	cache := newLRUCache[V](ttl, capacity)
	return func() V {
		if value, ok, err := cache.Get(ctx, 0); err != nil {
			panic(err)
		} else if ok {
			return value
		}
		value, err := cache.GetOrCompute(ctx, 0, func(context.Context) (V, error) {
			return computeFn(), nil
		})
		if err != nil {
			panic(err)
		}
		return value
	}
}

func lruMemoize1[K comparable, V any](computeFn func(K) V, ttl time.Duration, capacity int) func(K) V {
	ctx := context.Background()
	cache := newLRUCache[V](ttl, capacity)
	return func(key K) V {
		cacheKey := benchHash1(key)
		if value, ok, err := cache.Get(ctx, cacheKey); err != nil {
			panic(err)
		} else if ok {
			return value
		}
		value, err := cache.GetOrCompute(ctx, cacheKey, func(context.Context) (V, error) {
			return computeFn(key), nil
		})
		if err != nil {
			panic(err)
		}
		return value
	}
}

func lruMemoize2[K1, K2 comparable, V any](computeFn func(K1, K2) V, ttl time.Duration, capacity int) func(K1, K2) V {
	ctx := context.Background()
	cache := newLRUCache[V](ttl, capacity)
	return func(key1 K1, key2 K2) V {
		cacheKey := benchHash2(key1, key2)
		if value, ok, err := cache.Get(ctx, cacheKey); err != nil {
			panic(err)
		} else if ok {
			return value
		}
		value, err := cache.GetOrCompute(ctx, cacheKey, func(context.Context) (V, error) {
			return computeFn(key1, key2), nil
		})
		if err != nil {
			panic(err)
		}
		return value
	}
}

func lruMemoize3[K1, K2, K3 comparable, V any](computeFn func(K1, K2, K3) V, ttl time.Duration, capacity int) func(K1, K2, K3) V {
	ctx := context.Background()
	cache := newLRUCache[V](ttl, capacity)
	return func(key1 K1, key2 K2, key3 K3) V {
		cacheKey := benchHash3(key1, key2, key3)
		if value, ok, err := cache.Get(ctx, cacheKey); err != nil {
			panic(err)
		} else if ok {
			return value
		}
		value, err := cache.GetOrCompute(ctx, cacheKey, func(context.Context) (V, error) {
			return computeFn(key1, key2, key3), nil
		})
		if err != nil {
			panic(err)
		}
		return value
	}
}

func lruMemoize4[K1, K2, K3, K4 comparable, V any](computeFn func(K1, K2, K3, K4) V, ttl time.Duration, capacity int) func(K1, K2, K3, K4) V {
	ctx := context.Background()
	cache := newLRUCache[V](ttl, capacity)
	return func(key1 K1, key2 K2, key3 K3, key4 K4) V {
		cacheKey := benchHash4(key1, key2, key3, key4)
		if value, ok, err := cache.Get(ctx, cacheKey); err != nil {
			panic(err)
		} else if ok {
			return value
		}
		value, err := cache.GetOrCompute(ctx, cacheKey, func(context.Context) (V, error) {
			return computeFn(key1, key2, key3, key4), nil
		})
		if err != nil {
			panic(err)
		}
		return value
	}
}

func benchHash1[A comparable](key A) uint64 {
	return internalhash.Comparable(internalhash.Offset64, key)
}

func benchHash2[A, B comparable](key1 A, key2 B) uint64 {
	return internalhash.Comparable(internalhash.Comparable(internalhash.Offset64, key1), key2)
}

func benchHash3[A, B, C comparable](key1 A, key2 B, key3 C) uint64 {
	return internalhash.Comparable(internalhash.Comparable(internalhash.Comparable(internalhash.Offset64, key1), key2), key3)
}

func benchHash4[A, B, C, D comparable](key1 A, key2 B, key3 C, key4 D) uint64 {
	return internalhash.Comparable(internalhash.Comparable(internalhash.Comparable(internalhash.Comparable(internalhash.Offset64, key1), key2), key3), key4)
}
