package go_memoize

import (
	"context"
	"time"
)

// MemoizeCtx returns a memoized version of the compute function with a specified TTL.
func MemoizeCtx[V any](computeFn func(context.Context) V, ttl time.Duration) func(context.Context) V {
	cache := NewCacheSized[uint64, V](1, int64(ttl.Seconds()))
	return func(ctx context.Context) V {
		return cache.GetOrCompute(0, func() V {
			return computeFn(ctx)
		})
	}
}

// MemoizeCtx1 returns a memoized version of the compute function with a single key and a specified TTL.
func MemoizeCtx1[K comparable, V any](computeFn func(context.Context, K) V, ttl time.Duration) func(context.Context, K) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, k K) V {
		return cache.GetOrCompute(hash1(k), func() V {
			return computeFn(ctx, k)
		})
	}
}

// MemoizeCtx2 returns a memoized version of the compute function with two keys and a specified TTL.
func MemoizeCtx2[K1, K2 comparable, V any](computeFn func(context.Context, K1, K2) V, ttl time.Duration) func(context.Context, K1, K2) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2) V {
		return cache.GetOrCompute(hash2(key1, key2), func() V {
			return computeFn(ctx, key1, key2)
		})
	}
}

// MemoizeCtx3 returns a memoized version of the compute function with three keys and a specified TTL.
func MemoizeCtx3[K1, K2, K3 comparable, V any](computeFn func(context.Context, K1, K2, K3) V, ttl time.Duration) func(context.Context, K1, K2, K3) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3) V {
		return cache.GetOrCompute(hash3(key1, key2, key3), func() V {
			return computeFn(ctx, key1, key2, key3)
		})
	}
}

// MemoizeCtx4 returns a memoized version of the compute function with four keys and a specified TTL.
func MemoizeCtx4[K1, K2, K3, K4 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4) V, ttl time.Duration) func(context.Context, K1, K2, K3, K4) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4) V {
		return cache.GetOrCompute(hash4(key1, key2, key3, key4), func() V {
			return computeFn(ctx, key1, key2, key3, key4)
		})
	}
}

// MemoizeCtx5 returns a memoized version of the compute function with five keys and a specified TTL.
func MemoizeCtx5[K1, K2, K3, K4, K5 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5) V, ttl time.Duration) func(context.Context, K1, K2, K3, K4, K5) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) V {
		return cache.GetOrCompute(hash5(key1, key2, key3, key4, key5), func() V {
			return computeFn(ctx, key1, key2, key3, key4, key5)
		})
	}
}

// MemoizeCtx6 returns a memoized version of the compute function with six keys and a specified TTL.
func MemoizeCtx6[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6) V, ttl time.Duration) func(context.Context, K1, K2, K3, K4, K5, K6) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) V {
		return cache.GetOrCompute(hash6(key1, key2, key3, key4, key5, key6), func() V {
			return computeFn(ctx, key1, key2, key3, key4, key5, key6)
		})
	}
}

// MemoizeCtx7 returns a memoized version of the compute function with seven keys and a specified TTL.
func MemoizeCtx7[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6, K7) V, ttl time.Duration) func(context.Context, K1, K2, K3, K4, K5, K6, K7) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) V {
		return cache.GetOrCompute(hash7(key1, key2, key3, key4, key5, key6, key7), func() V {
			return computeFn(ctx, key1, key2, key3, key4, key5, key6, key7)
		})
	}
}

// --- New context-aware variants returning (V, error) that avoid caching errors ---

// MemoizeCtxE memoizes a context-aware function returning (V, error). Errors are not cached.
func MemoizeCtxE[V any](computeFn func(context.Context) (V, error), ttl time.Duration) func(context.Context) (V, error) {
	cache := NewCacheSized[uint64, V](1, int64(ttl.Seconds()))
	return func(ctx context.Context) (V, error) {
		if v, ok := cache.Get(0); ok {
			return v, nil
		}
		v, err := computeFn(ctx)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(0, v)
		return v, nil
	}
}

// MemoizeCtx1E memoizes a context-aware function with 1 arg returning (V, error). Errors are not cached.
func MemoizeCtx1E[K comparable, V any](computeFn func(context.Context, K) (V, error), ttl time.Duration) func(context.Context, K) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, k K) (V, error) {
		key := hash1(k)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(ctx, k)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// MemoizeCtx2E memoizes a context-aware function with 2 args returning (V, error). Errors are not cached.
func MemoizeCtx2E[K1, K2 comparable, V any](computeFn func(context.Context, K1, K2) (V, error), ttl time.Duration) func(context.Context, K1, K2) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2) (V, error) {
		key := hash2(key1, key2)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(ctx, key1, key2)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// MemoizeCtx3E memoizes a context-aware function with 3 args returning (V, error). Errors are not cached.
func MemoizeCtx3E[K1, K2, K3 comparable, V any](computeFn func(context.Context, K1, K2, K3) (V, error), ttl time.Duration) func(context.Context, K1, K2, K3) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3) (V, error) {
		key := hash3(key1, key2, key3)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(ctx, key1, key2, key3)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// MemoizeCtx4E memoizes a context-aware function with 4 args returning (V, error). Errors are not cached.
func MemoizeCtx4E[K1, K2, K3, K4 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4) (V, error), ttl time.Duration) func(context.Context, K1, K2, K3, K4) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4) (V, error) {
		key := hash4(key1, key2, key3, key4)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(ctx, key1, key2, key3, key4)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// MemoizeCtx5E memoizes a context-aware function with 5 args returning (V, error). Errors are not cached.
func MemoizeCtx5E[K1, K2, K3, K4, K5 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5) (V, error), ttl time.Duration) func(context.Context, K1, K2, K3, K4, K5) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) (V, error) {
		key := hash5(key1, key2, key3, key4, key5)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(ctx, key1, key2, key3, key4, key5)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// MemoizeCtx6E memoizes a context-aware function with 6 args returning (V, error). Errors are not cached.
func MemoizeCtx6E[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6) (V, error), ttl time.Duration) func(context.Context, K1, K2, K3, K4, K5, K6) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) (V, error) {
		key := hash6(key1, key2, key3, key4, key5, key6)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(ctx, key1, key2, key3, key4, key5, key6)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// MemoizeCtx7E memoizes a context-aware function with 7 args returning (V, error). Errors are not cached.
func MemoizeCtx7E[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6, K7) (V, error), ttl time.Duration) func(context.Context, K1, K2, K3, K4, K5, K6, K7) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) (V, error) {
		key := hash7(key1, key2, key3, key4, key5, key6, key7)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(ctx, key1, key2, key3, key4, key5, key6, key7)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}
