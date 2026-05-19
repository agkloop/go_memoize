// v2/background/options.go
package background

import (
	memoize "github.com/agkloop/go_memoize"
)

type writeThroughCfg[V any] struct {
	key   string
	store memoize.Store[string, V]
}

type options[V any] struct {
	onError      func(error)
	onRefresh    func(V)
	writeThrough *writeThroughCfg[V]
}

// Option configures Keep or Mirror.
type Option[V any] func(*options[V])

// WriteThrough writes each successfully refreshed value to store under key.
// Used by writer processes to publish to a shared store (e.g. Redis).
func WriteThrough[V any](key string, store memoize.Store[string, V]) Option[V] {
	return func(o *options[V]) {
		o.writeThrough = &writeThroughCfg[V]{key: key, store: store}
	}
}

// OnError is called when a refresh fails after the initial load.
// The stale value is kept automatically.
func OnError[V any](fn func(error)) Option[V] {
	return func(o *options[V]) { o.onError = fn }
}

// OnRefresh is called after each successful refresh with the new value.
func OnRefresh[V any](fn func(V)) Option[V] {
	return func(o *options[V]) { o.onRefresh = fn }
}

func applyOptions[V any](opts []Option[V]) options[V] {
	o := options[V]{
		onError:   func(error) {},
		onRefresh: func(V) {},
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
