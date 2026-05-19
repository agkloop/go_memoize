package memoize

import "context"

func MemoizeCtx[V any](computeFn func(context.Context) V, opts Options) (func(context.Context) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) V {
		value, err := cache.GetOrCompute(ctx, 0, func(context.Context) (V, error) {
			return computeFn(ctx), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtx1[K comparable, V any](computeFn func(context.Context, K) V, opts Options) (func(context.Context, K) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, k K) V {
		value, err := cache.GetOrCompute(ctx, hash1(k), func(context.Context) (V, error) {
			return computeFn(ctx, k), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtx2[K1, K2 comparable, V any](computeFn func(context.Context, K1, K2) V, opts Options) (func(context.Context, K1, K2) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2) V {
		value, err := cache.GetOrCompute(ctx, hash2(key1, key2), func(context.Context) (V, error) {
			return computeFn(ctx, key1, key2), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtx3[K1, K2, K3 comparable, V any](computeFn func(context.Context, K1, K2, K3) V, opts Options) (func(context.Context, K1, K2, K3) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3) V {
		value, err := cache.GetOrCompute(ctx, hash3(key1, key2, key3), func(context.Context) (V, error) {
			return computeFn(ctx, key1, key2, key3), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtx4[K1, K2, K3, K4 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4) V, opts Options) (func(context.Context, K1, K2, K3, K4) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4) V {
		value, err := cache.GetOrCompute(ctx, hash4(key1, key2, key3, key4), func(context.Context) (V, error) {
			return computeFn(ctx, key1, key2, key3, key4), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtx5[K1, K2, K3, K4, K5 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5) V, opts Options) (func(context.Context, K1, K2, K3, K4, K5) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) V {
		value, err := cache.GetOrCompute(ctx, hash5(key1, key2, key3, key4, key5), func(context.Context) (V, error) {
			return computeFn(ctx, key1, key2, key3, key4, key5), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtx6[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6) V, opts Options) (func(context.Context, K1, K2, K3, K4, K5, K6) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) V {
		value, err := cache.GetOrCompute(ctx, hash6(key1, key2, key3, key4, key5, key6), func(context.Context) (V, error) {
			return computeFn(ctx, key1, key2, key3, key4, key5, key6), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtx7[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6, K7) V, opts Options) (func(context.Context, K1, K2, K3, K4, K5, K6, K7) V, error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) V {
		value, err := cache.GetOrCompute(ctx, hash7(key1, key2, key3, key4, key5, key6, key7), func(context.Context) (V, error) {
			return computeFn(ctx, key1, key2, key3, key4, key5, key6, key7), nil
		})
		if err != nil {
			var zero V
			return zero
		}
		return value
	}, nil
}

func MemoizeCtxE[V any](computeFn func(context.Context) (V, error), opts Options) (func(context.Context) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) (V, error) {
		return getSetDirect(ctx, cache, 0, func() (V, error) { return computeFn(ctx) })
	}, nil
}

func MemoizeCtx1E[K comparable, V any](computeFn func(context.Context, K) (V, error), opts Options) (func(context.Context, K) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, k K) (V, error) {
		return getSetDirect(ctx, cache, hash1(k), func() (V, error) { return computeFn(ctx, k) })
	}, nil
}

func MemoizeCtx2E[K1, K2 comparable, V any](computeFn func(context.Context, K1, K2) (V, error), opts Options) (func(context.Context, K1, K2) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2) (V, error) {
		return getSetDirect(ctx, cache, hash2(key1, key2), func() (V, error) { return computeFn(ctx, key1, key2) })
	}, nil
}

func MemoizeCtx3E[K1, K2, K3 comparable, V any](computeFn func(context.Context, K1, K2, K3) (V, error), opts Options) (func(context.Context, K1, K2, K3) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3) (V, error) {
		return getSetDirect(ctx, cache, hash3(key1, key2, key3), func() (V, error) { return computeFn(ctx, key1, key2, key3) })
	}, nil
}

func MemoizeCtx4E[K1, K2, K3, K4 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4) (V, error), opts Options) (func(context.Context, K1, K2, K3, K4) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4) (V, error) {
		return getSetDirect(ctx, cache, hash4(key1, key2, key3, key4), func() (V, error) { return computeFn(ctx, key1, key2, key3, key4) })
	}, nil
}

func MemoizeCtx5E[K1, K2, K3, K4, K5 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5) (V, error), opts Options) (func(context.Context, K1, K2, K3, K4, K5) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5) (V, error) {
		return getSetDirect(ctx, cache, hash5(key1, key2, key3, key4, key5), func() (V, error) { return computeFn(ctx, key1, key2, key3, key4, key5) })
	}, nil
}

func MemoizeCtx6E[K1, K2, K3, K4, K5, K6 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6) (V, error), opts Options) (func(context.Context, K1, K2, K3, K4, K5, K6) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6) (V, error) {
		return getSetDirect(ctx, cache, hash6(key1, key2, key3, key4, key5, key6), func() (V, error) { return computeFn(ctx, key1, key2, key3, key4, key5, key6) })
	}, nil
}

func MemoizeCtx7E[K1, K2, K3, K4, K5, K6, K7 comparable, V any](computeFn func(context.Context, K1, K2, K3, K4, K5, K6, K7) (V, error), opts Options) (func(context.Context, K1, K2, K3, K4, K5, K6, K7) (V, error), error) {
	cache, err := newDirectCache[V](opts)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, key1 K1, key2 K2, key3 K3, key4 K4, key5 K5, key6 K6, key7 K7) (V, error) {
		return getSetDirect(ctx, cache, hash7(key1, key2, key3, key4, key5, key6, key7), func() (V, error) { return computeFn(ctx, key1, key2, key3, key4, key5, key6, key7) })
	}, nil
}
