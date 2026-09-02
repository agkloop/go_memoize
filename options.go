package memoize

import (
	"sync"
	"time"
)

type Cache[K comparable, V any] struct {
	store            Store[K, V]
	peeker           peekingStore[K, V]
	ttlSet           bool
	ttl              time.Duration
	staleTTL         time.Duration
	noExpiration     bool
	bypass           bool
	keepStaleOnError bool
	metrics          Metrics
	metricsEnabled   bool
	clock            Clock
	clockOwned       bool
	tickerInterval   time.Duration
	refreshTimeout   time.Duration
	flightMu         sync.Mutex
	flights          map[K]*flight[V]
}

func New[K comparable, V any](opts ...Options) (*Cache[K, V], error) {
	c := &Cache[K, V]{
		metrics:        noopMetrics{},
		refreshTimeout: 30 * time.Second,
		flights:        make(map[K]*flight[V]),
	}
	for _, opt := range opts {
		if err := applyOptions(c, opt); err != nil {
			return nil, err
		}
	}
	if c.ttlSet && c.ttl <= 0 {
		return nil, ErrInvalidTTL
	}
	if c.staleTTL < 0 {
		return nil, ErrInvalidStaleTTL
	}
	if c.ttl == 0 && !c.noExpiration && !c.bypass {
		return nil, ErrMissingExpirationPolicy
	}
	if c.ttl == 0 && c.staleTTL > 0 {
		return nil, ErrInvalidStaleTTL
	}
	if c.store == nil && !c.bypass {
		return nil, ErrMissingStore
	}
	if c.tickerInterval > 0 {
		c.clock = NewTickerClock(c.tickerInterval)
		c.clockOwned = true
	} else if c.clock == nil {
		c.clock = NewTickerClock(time.Millisecond)
		c.clockOwned = true
	}
	return c, nil
}

// Stop releases background resources held by the cache.
// If the cache owns a TickerClock (the default or WithTickerClock), Stop shuts
// down its background goroutine. Clocks supplied with WithClock remain owned
// by the caller. Safe to call multiple times.
func (c *Cache[K, V]) Stop() {
	if tc, ok := c.clock.(*TickerClock); ok && c.clockOwned {
		tc.Stop()
	}
}
