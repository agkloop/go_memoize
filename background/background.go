// v2/background/background.go
package background

import (
	"context"
	"errors"
	"fmt"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/internal/refreshloop"
)

var errMirrorRefreshMiss = errors.New("background.Mirror: refresh missed remote key")

// Keep starts a background goroutine that calls fn every interval.
// Blocks until the first successful call. Returns error if the first call fails.
// The goroutine stops when ctx is cancelled.
func Keep[V any](
	ctx context.Context,
	fn func(context.Context) (V, error),
	interval time.Duration,
	opts ...Option[V],
) (*Value[V], error) {
	o := applyOptions(opts)

	v, err := fn(ctx)
	if err != nil {
		return nil, err
	}
	val := &Value[V]{}
	val.store(v)
	o.onRefresh(v)
	if o.writeThrough != nil {
		_ = o.writeThrough.store.Set(ctx, o.writeThrough.key, memoize.Stored[V]{Value: v, NoExpire: true})
	}

	go refreshloop.Run(ctx, interval, fn, refreshloop.Hooks[V]{
		OnValue: func(refreshed V) {
			val.store(refreshed)
			o.onRefresh(refreshed)
			if o.writeThrough != nil {
				_ = o.writeThrough.store.Set(ctx, o.writeThrough.key,
					memoize.Stored[V]{Value: refreshed, NoExpire: true})
			}
		},
		OnError: o.onError,
	})

	return val, nil
}

// MustKeep is Keep but panics on initial load error. For use in main().
func MustKeep[V any](
	ctx context.Context,
	fn func(context.Context) (V, error),
	interval time.Duration,
	opts ...Option[V],
) *Value[V] {
	val, err := Keep(ctx, fn, interval, opts...)
	if err != nil {
		panic("background.MustKeep: initial load failed: " + err.Error())
	}
	return val
}

// Mirror starts a background goroutine that reads key from remote every interval,
// storing the result in a local atomic mirror.
// Blocks until the first successful read. Returns error if key is missing.
// The goroutine stops when ctx is cancelled.
func Mirror[V any](
	ctx context.Context,
	key string,
	remote memoize.Store[string, V],
	interval time.Duration,
	opts ...Option[V],
) (*Value[V], error) {
	o := applyOptions(opts)

	entry, ok, err := remote.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("background.Mirror: key %q not found in remote store", key)
	}

	val := &Value[V]{}
	val.store(entry.Value)
	o.onRefresh(entry.Value)

	go refreshloop.Run(ctx, interval, func(ctx context.Context) (V, error) {
		e, ok, err := remote.Get(ctx, key)
		if err != nil {
			var zero V
			return zero, err
		}
		if !ok {
			var zero V
			return zero, errMirrorRefreshMiss
		}
		return e.Value, nil
	}, refreshloop.Hooks[V]{
		OnValue: func(refreshed V) {
			val.store(refreshed)
			o.onRefresh(refreshed)
		},
		OnError: func(err error) {
			if !errors.Is(err, errMirrorRefreshMiss) {
				o.onError(err)
			}
		},
	})

	return val, nil
}

// MustMirror is Mirror but panics on initial load error. For use in main().
func MustMirror[V any](
	ctx context.Context,
	key string,
	remote memoize.Store[string, V],
	interval time.Duration,
	opts ...Option[V],
) *Value[V] {
	val, err := Mirror(ctx, key, remote, interval, opts...)
	if err != nil {
		panic("background.MustMirror: initial read failed: " + err.Error())
	}
	return val
}
