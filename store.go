package memoize

import "context"

type Store[K comparable, V any] interface {
	Get(ctx context.Context, key K) (Stored[V], bool, error)
	Set(ctx context.Context, key K, value Stored[V]) error
	Delete(ctx context.Context, key K) error
	Clear(ctx context.Context) error
}
