package loader

import (
	"context"
	"sync"
	"time"

	"github.com/agkloop/go_memoize/internal/refreshloop"
)

type options[V any] struct {
	onError func(error)
}

// Option configures a Loader.
type Option[V any] func(*options[V])

// WithOnError sets a callback invoked whenever the load function returns an error.
// If not set, errors are silently ignored (stale value is kept).
func WithOnError[V any](fn func(error)) Option[V] {
	return func(o *options[V]) { o.onError = fn }
}

// Loader runs a load function on a fixed interval and caches the latest result.
// Value() always returns instantly (after the first successful load).
type Loader[V any] struct {
	fn       func(context.Context) (V, error)
	interval time.Duration
	opts     options[V]

	mu      sync.RWMutex
	value   V
	err     error
	hasVal  bool
	ready   chan struct{} // closed on first successful load
	stop    chan struct{}
	stopped chan struct{}
}

// New creates and starts a Loader. The load function is called immediately,
// then on every interval tick. Stop must be called to release resources.
func New[V any](fn func(context.Context) (V, error), interval time.Duration, opts ...Option[V]) *Loader[V] {
	o := options[V]{}
	for _, opt := range opts {
		opt(&o)
	}
	l := &Loader[V]{
		fn:       fn,
		interval: interval,
		opts:     o,
		ready:    make(chan struct{}),
		stop:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
	go l.run()
	return l
}

func (l *Loader[V]) run() {
	defer close(l.stopped)
	l.load()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		select {
		case <-l.stop:
			cancel()
		case <-done:
		}
	}()
	defer func() {
		close(done)
		cancel()
	}()

	refreshloop.Run(ctx, l.interval, l.fn, refreshloop.Hooks[V]{
		OnValue: l.store,
		OnError: l.storeError,
	})
}

func (l *Loader[V]) load() {
	v, err := l.fn(context.Background())
	if err != nil {
		l.storeError(err)
		return
	}
	l.store(v)
}

func (l *Loader[V]) store(v V) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.value = v
	l.err = nil
	if !l.hasVal {
		l.hasVal = true
		close(l.ready)
	}
}

func (l *Loader[V]) storeError(err error) {
	l.mu.Lock()
	if !l.hasVal {
		l.err = err
	}
	l.mu.Unlock()
	if l.opts.onError != nil {
		l.opts.onError(err)
	}
}

// Value returns the latest successfully loaded value.
// Blocks until the first successful load or ctx is cancelled.
// After the first success, always returns instantly.
func (l *Loader[V]) Value(ctx context.Context) (V, error) {
	select {
	case <-l.ready:
		l.mu.RLock()
		v, err := l.value, l.err
		l.mu.RUnlock()
		return v, err
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	}
}

// Stop halts the background refresh goroutine and waits for it to exit.
func (l *Loader[V]) Stop() {
	close(l.stop)
	<-l.stopped
}
