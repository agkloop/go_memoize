package chain

import (
	"context"

	memoize "github.com/agkloop/go_memoize"
)

// ChainStore is an ordered sequence of Store[K, V] tiers.
// On Get: checks tiers in order; on a hit in tier i, backfills tiers 0..i-1.
// On Set/Delete/Clear: propagates to all tiers.
type ChainStore[K comparable, V any] struct {
	tiers []memoize.Store[K, V]
}

// New creates a ChainStore from the given tiers (L1 first, L2 second, ...).
// At least two tiers are required.
func New[K comparable, V any](tiers ...memoize.Store[K, V]) *ChainStore[K, V] {
	if len(tiers) < 2 {
		panic("chain.New: at least two tiers required")
	}
	return &ChainStore[K, V]{tiers: tiers}
}

func (c *ChainStore[K, V]) Get(ctx context.Context, key K) (memoize.Stored[V], bool, error) {
	for i, tier := range c.tiers {
		val, ok, err := tier.Get(ctx, key)
		if err != nil {
			return val, false, err
		}
		if ok {
			// Backfill all higher-priority tiers (indices 0..i-1)
			for j := 0; j < i; j++ {
				_ = c.tiers[j].Set(ctx, key, val)
			}
			return val, true, nil
		}
	}
	var zero memoize.Stored[V]
	return zero, false, nil
}

func (c *ChainStore[K, V]) Set(ctx context.Context, key K, value memoize.Stored[V]) error {
	for _, tier := range c.tiers {
		if err := tier.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainStore[K, V]) Delete(ctx context.Context, key K) error {
	for _, tier := range c.tiers {
		if err := tier.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainStore[K, V]) Clear(ctx context.Context) error {
	for _, tier := range c.tiers {
		if err := tier.Clear(ctx); err != nil {
			return err
		}
	}
	return nil
}
