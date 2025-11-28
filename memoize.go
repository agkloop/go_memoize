package go_memoize

import (
	"time"
)

// Memoize returns a memoized version of the compute function with a specified TTL.
// V is the type of the value returned by the compute function.
func Memoize[V any](computeFn func() V, ttl time.Duration) func() V {
	cache := NewCacheSized[uint64, V](1, int64(ttl.Seconds()))
	return func() V {
		return cache.GetOrCompute(0, func() V {
			return computeFn()
		})
	}
}

// Memoize1 returns a memoized version of the compute function with a single key and a specified TTL.
// K is the type of the key, and V is the type of the value returned by the compute function.
func Memoize1[K comparable, V any](computeFn func(K) V, ttl time.Duration) func(K) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(k K) V {
		return cache.GetOrCompute(hash1(k), func() V {
			return computeFn(k)
		})
	}
}

// Memoize2 returns a memoized version of the compute function with two keys and a specified TTL.
// K1 and K2 are the types of the keys, and V is the type of the value returned by the compute function.
func Memoize2[K1, K2 comparable, V any](computeFn func(K1, K2) V, ttl time.Duration) func(K1, K2) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2) V {
		return cache.GetOrCompute(hash2(key1, key2), func() V {
			return computeFn(key1, key2)
		})
	}
}

// Memoize3 returns a memoized version of the compute function with three keys and a specified TTL.
// K1, K2, and K3 are the types of the keys, and V is the type of the value returned by the compute function.
func Memoize3[K1, K2, K3 comparable, V any](computeFn func(K1, K2, K3) V, ttl time.Duration) func(K1, K2, K3) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3) V {
		return cache.GetOrCompute(hash3(key1, key2, key3), func() V {
			return computeFn(key1, key2, key3)
		})
	}
}

// Memoize4 returns a memoized version of the compute function with four keys and a specified TTL.
// K1, K2, K3, and K4 are the types of the keys, and V is the type of the value returned by the compute function.
func Memoize4[K1, K2, K3, K4 comparable, V any](computeFn func(K1, K2, K3, K4) V, ttl time.Duration) func(K1, K2, K3, K4) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4) V {
		return cache.GetOrCompute(hash4(key1, key2, key3, key4), func() V {
			return computeFn(key1, key2, key3, key4)
		})
	}
}

// Memoize5 returns a memoized version of the compute function with five keys and a specified TTL.
// K1, K2, K3, K4, and K5 are the types of the keys, and V is the type of the value returned by the compute function.
func Memoize5[K1, K2, K3, K4, K5 comparable, V any](computeFn func(K1, K2, K3, K4, K5) V, ttl time.Duration) func(K1, K2, K3, K4, K5) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) V {
		return cache.GetOrCompute(hash5(key1, key2, key3, key4, key5), func() V {
			return computeFn(key1, key2, key3, key4, key5)
		})
	}
}

// Memoize6 returns a memoized version of the compute function with six keys and a specified TTL.
// K1, K2, K3, K4, K5, and K6 are the types of the keys, and V is the type of the value returned by the compute function.
func Memoize6[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6) V, ttl time.Duration) func(K1, K2, K3, K4, K5, K6) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) V {
		return cache.GetOrCompute(hash6(key1, key2, key3, key4, key5, key6), func() V {
			return computeFn(key1, key2, key3, key4, key5, key6)
		})
	}
}

// Memoize7 returns a memoized version of the compute function with seven keys and a specified TTL.
// K1, K2, K3, K4, K5, K6, and K7 are the types of the keys, and V is the type of the value returned by the compute function.
func Memoize7[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6, K7) V, ttl time.Duration) func(K1, K2, K3, K4, K5, K6, K7) V {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) V {
		return cache.GetOrCompute(hash7(key1, key2, key3, key4, key5, key6, key7), func() V {
			return computeFn(key1, key2, key3, key4, key5, key6, key7)
		})
	}
}

// --- New variants that return an error and avoid caching when computeFn returns a non-nil error ---

// MemoizeE memoizes a function that returns (V, error). Errors are not cached.
func MemoizeE[V any](computeFn func() (V, error), ttl time.Duration) func() (V, error) {
	cache := NewCacheSized[uint64, V](1, int64(ttl.Seconds()))
	return func() (V, error) {
		// try cached
		if v, ok := cache.Get(0); ok {
			return v, nil
		}
		// compute
		v, err := computeFn()
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(0, v)
		return v, nil
	}
}

// Memoize1E memoizes a function with 1 arg that returns (V, error). Errors are not cached.
func Memoize1E[K comparable, V any](computeFn func(K) (V, error), ttl time.Duration) func(K) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(k K) (V, error) {
		key := hash1(k)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(k)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// Memoize2E memoizes a function with 2 args that returns (V, error). Errors are not cached.
func Memoize2E[K1, K2 comparable, V any](computeFn func(K1, K2) (V, error), ttl time.Duration) func(K1, K2) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2) (V, error) {
		key := hash2(key1, key2)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(key1, key2)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// Memoize3E memoizes a function with 3 args that returns (V, error). Errors are not cached.
func Memoize3E[K1, K2, K3 comparable, V any](computeFn func(K1, K2, K3) (V, error), ttl time.Duration) func(K1, K2, K3) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3) (V, error) {
		key := hash3(key1, key2, key3)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(key1, key2, key3)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// Memoize4E memoizes a function with 4 args that returns (V, error). Errors are not cached.
func Memoize4E[K1, K2, K3, K4 comparable, V any](computeFn func(K1, K2, K3, K4) (V, error), ttl time.Duration) func(K1, K2, K3, K4) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4) (V, error) {
		key := hash4(key1, key2, key3, key4)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(key1, key2, key3, key4)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// Memoize5E memoizes a function with 5 args that returns (V, error). Errors are not cached.
func Memoize5E[K1, K2, K3, K4, K5 comparable, V any](computeFn func(K1, K2, K3, K4, K5) (V, error), ttl time.Duration) func(K1, K2, K3, K4, K5) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) (V, error) {
		key := hash5(key1, key2, key3, key4, key5)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(key1, key2, key3, key4, key5)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// Memoize6E memoizes a function with 6 args that returns (V, error). Errors are not cached.
func Memoize6E[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6) (V, error), ttl time.Duration) func(K1, K2, K3, K4, K5, K6) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) (V, error) {
		key := hash6(key1, key2, key3, key4, key5, key6)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(key1, key2, key3, key4, key5, key6)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}

// Memoize7E memoizes a function with 7 args that returns (V, error). Errors are not cached.
func Memoize7E[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(K1, K2, K3, K4, K5, K6, K7) (V, error), ttl time.Duration) func(K1, K2, K3, K4, K5, K6, K7) (V, error) {
	cache := NewCache[uint64, V](int64(ttl.Seconds()))
	return func(key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) (V, error) {
		key := hash7(key1, key2, key3, key4, key5, key6, key7)
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		v, err := computeFn(key1, key2, key3, key4, key5, key6, key7)
		if err != nil {
			return zeroValue[V](), err
		}
		cache.Set(key, v)
		return v, nil
	}
}
