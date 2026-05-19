package memoize

import "context"

// Memoize returns a memoized version of the compute function.
func Memoize[V any](computeFn func() V, opts Options) (func() V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func() V {
		value, err := cache.GetOrCompute(context.Background(), 0, func(context.Context) (V, error) {
			return computeFn(), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// Memoize1 returns a memoized version of the compute function with a single key.
func Memoize1[K comparable, V any](computeFn func(K) V, opts Options) (func(K) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(k K) V {
		value, err := cache.GetOrCompute(context.Background(), hash1(k), func(context.Context) (V, error) {
			return computeFn(k), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// Memoize2 returns a memoized version of the compute function with two keys.
func Memoize2[K1, K2 comparable, V any](computeFn func(K1, K2) V, opts Options) (func(K1, K2) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2) V {
		value, err := cache.GetOrCompute(context.Background(), hash2(key1, key2), func(context.Context) (V, error) {
			return computeFn(key1, key2), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// Memoize3 returns a memoized version of the compute function with three keys.
func Memoize3[K1, K2, K3 comparable, V any](computeFn func(K1, K2, K3) V, opts Options) (func(K1, K2, K3) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3) V {
		value, err := cache.GetOrCompute(context.Background(), hash3(key1, key2, key3), func(context.Context) (V, error) {
			return computeFn(key1, key2, key3), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// Memoize4 returns a memoized version of the compute function with four keys.
func Memoize4[K1, K2, K3, K4 comparable, V any](computeFn func(K1, K2, K3, K4) V, opts Options) (func(K1, K2, K3, K4) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4) V {
		value, err := cache.GetOrCompute(context.Background(), hash4(key1, key2, key3, key4), func(context.Context) (V, error) {
			return computeFn(key1, key2, key3, key4), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// Memoize5 returns a memoized version of the compute function with five keys.
func Memoize5[K1, K2, K3, K4, K5 comparable, V any](computeFn func(K1, K2, K3, K4, K5) V, opts Options) (func(K1, K2, K3, K4, K5) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) V {
		value, err := cache.GetOrCompute(context.Background(), hash5(key1, key2, key3, key4, key5), func(context.Context) (V, error) {
			return computeFn(key1, key2, key3, key4, key5), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// Memoize6 returns a memoized version of the compute function with six keys.
func Memoize6[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6) V, opts Options) (func(K1, K2, K3, K4, K5, K6) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) V {
		value, err := cache.GetOrCompute(context.Background(), hash6(key1, key2, key3, key4, key5, key6), func(context.Context) (V, error) {
			return computeFn(key1, key2, key3, key4, key5, key6), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// Memoize7 returns a memoized version of the compute function with seven keys.
func Memoize7[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6, K7) V, opts Options) (func(K1, K2, K3, K4, K5, K6, K7) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) V {
		value, err := cache.GetOrCompute(context.Background(), hash7(key1, key2, key3, key4, key5, key6, key7), func(context.Context) (V, error) {
			return computeFn(key1, key2, key3, key4, key5, key6, key7), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

// MemoizeE memoizes a function that returns (V, error). Errors are not cached.
func MemoizeE[V any](computeFn func() (V, error), opts Options) (func() (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func() (V, error) {
		return getSetDirect(context.Background(), cache, 0, func() (V, error) { return computeFn() })
	}, nil
}

func Memoize1E[K comparable, V any](computeFn func(K) (V, error), opts Options) (func(K) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(k K) (V, error) {
		return getSetDirect(context.Background(), cache, hash1(k), func() (V, error) { return computeFn(k) })
	}, nil
}

func Memoize2E[K1, K2 comparable, V any](computeFn func(K1, K2) (V, error), opts Options) (func(K1, K2) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2) (V, error) {
		return getSetDirect(context.Background(), cache, hash2(key1, key2), func() (V, error) { return computeFn(key1, key2) })
	}, nil
}

func Memoize3E[K1, K2, K3 comparable, V any](computeFn func(K1, K2, K3) (V, error), opts Options) (func(K1, K2, K3) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3) (V, error) {
		return getSetDirect(context.Background(), cache, hash3(key1, key2, key3), func() (V, error) { return computeFn(key1, key2, key3) })
	}, nil
}

func Memoize4E[K1, K2, K3, K4 comparable, V any](computeFn func(K1, K2, K3, K4) (V, error), opts Options) (func(K1, K2, K3, K4) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4) (V, error) {
		return getSetDirect(context.Background(), cache, hash4(key1, key2, key3, key4), func() (V, error) { return computeFn(key1, key2, key3, key4) })
	}, nil
}

func Memoize5E[K1, K2, K3, K4, K5 comparable, V any](computeFn func(K1, K2, K3, K4, K5) (V, error), opts Options) (func(K1, K2, K3, K4, K5) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) (V, error) {
		return getSetDirect(context.Background(), cache, hash5(key1, key2, key3, key4, key5), func() (V, error) { return computeFn(key1, key2, key3, key4, key5) })
	}, nil
}

func Memoize6E[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6) (V, error), opts Options) (func(K1, K2, K3, K4, K5, K6) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) (V, error) {
		return getSetDirect(context.Background(), cache, hash6(key1, key2, key3, key4, key5, key6), func() (V, error) { return computeFn(key1, key2, key3, key4, key5, key6) })
	}, nil
}

func Memoize7E[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6, K7) (V, error), opts Options) (func(K1, K2, K3, K4, K5, K6, K7) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) (V, error) {
		return getSetDirect(context.Background(), cache, hash7(key1, key2, key3, key4, key5, key6, key7), func() (V, error) { return computeFn(key1, key2, key3, key4, key5, key6, key7) })
	}, nil
}

func getSetDirect[V any](ctx context.Context, cache *Cache[uint64, V], key uint64, computeFn func() (V, error)) (V, error) {
	return cache.GetOrCompute(ctx, key, func(context.Context) (V, error) {
		return computeFn()
	})
}
