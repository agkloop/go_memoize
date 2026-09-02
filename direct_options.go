package memoize

import (
	"reflect"
	"time"
)

type Options struct {
	store            any
	hasStore         bool
	ttlSet           bool
	ttl              time.Duration
	staleTTL         time.Duration
	noExpiration     bool
	bypass           bool
	keepStaleOnError bool
	metrics          Metrics
	metricsEnabled   bool
	clock            Clock
	refreshTimeout   time.Duration
	tickerInterval   time.Duration
}

func Opts() Options { return Options{} }

func (o Options) WithStore(store any) Options {
	o.hasStore = true
	o.store = store
	return o
}

func (o Options) WithTTL(ttl time.Duration) Options {
	o.ttlSet = true
	o.ttl = ttl
	return o
}

func (o Options) WithStaleTTL(ttl time.Duration) Options {
	o.staleTTL = ttl
	return o
}

func (o Options) KeepStaleOnError() Options {
	o.keepStaleOnError = true
	return o
}

func (o Options) WithMetrics(metrics Metrics) Options {
	if metrics != nil {
		o.metrics = metrics
		o.metricsEnabled = true
	}
	return o
}

func (o Options) WithClock(clock Clock) Options {
	if clock != nil {
		o.clock = clock
		o.tickerInterval = 0
	}
	return o
}

func (o Options) WithRefreshTimeout(timeout time.Duration) Options {
	if timeout > 0 {
		o.refreshTimeout = timeout
	}
	return o
}

func (o Options) WithTickerClock(interval time.Duration) Options {
	if interval > 0 {
		o.clock = nil
		o.tickerInterval = interval
	}
	return o
}

func (o Options) NoExpiration() Options {
	o.noExpiration = true
	return o
}

func (o Options) Bypass() Options {
	o.bypass = true
	return o
}

func applyOptions[K comparable, V any](c *Cache[K, V], opts Options) error {
	if opts.hasStore {
		store, ok := opts.store.(Store[K, V])
		if !ok || isNilStore(store) {
			return ErrInvalidStore
		}
		c.store = store
		c.peeker, _ = store.(peekingStore[K, V])
	}
	if opts.ttlSet {
		c.ttlSet = true
		c.ttl = opts.ttl
	}
	if opts.staleTTL != 0 {
		c.staleTTL = opts.staleTTL
	}
	if opts.noExpiration {
		c.noExpiration = true
	}
	if opts.bypass {
		c.bypass = true
	}
	if opts.keepStaleOnError {
		c.keepStaleOnError = true
	}
	if opts.metricsEnabled {
		c.metrics = opts.metrics
		c.metricsEnabled = true
	}
	if opts.clock != nil {
		c.clock = opts.clock
		c.clockOwned = false
		c.tickerInterval = 0
	}
	if opts.refreshTimeout > 0 {
		c.refreshTimeout = opts.refreshTimeout
	}
	if opts.tickerInterval > 0 {
		c.clock = nil
		c.clockOwned = false
		c.tickerInterval = opts.tickerInterval
	}
	return nil
}

func isNilStore[K comparable, V any](store Store[K, V]) bool {
	if store == nil {
		return true
	}
	v := reflect.ValueOf(store)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func newDirectCache[V any](opts Options) (*Cache[uint64, V], error) {
	if !opts.hasStore {
		opts = opts.WithStore(newDirectStore[V]())
	}
	return New[uint64, V](opts)
}
