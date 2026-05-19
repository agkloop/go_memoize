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
	refreshTimeout   time.Duration
	flightMu         sync.Mutex
	flights          map[K]*flight[V]
}

func New[K comparable, V any](opts ...Options) (*Cache[K, V], error) {
	c := &Cache[K, V]{
		metrics:        noopMetrics{},
		clock:          NewTickerClock(time.Millisecond),
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
	return c, nil
}

// Stop releases background resources held by the cache.
// If the cache uses a TickerClock (the default), Stop shuts down its
// background goroutine. Safe to call multiple times.
func (c *Cache[K, V]) Stop() {
	if tc, ok := c.clock.(*TickerClock); ok {
		tc.Stop()
	}
}
