package memoize

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type flight[V any] struct {
	wg         sync.WaitGroup
	value      V
	err        error
	panicValue any
	panicked   bool
}

func (f *flight[V]) wait() (V, error) {
	f.wg.Wait()
	if f.panicked {
		panic(f.panicValue)
	}
	return f.value, f.err
}

// peekingStore lets GetOrCompute inspect stored entry state without applying
// Store.Get side effects such as recency updates before it chooses a policy.
type peekingStore[K comparable, V any] interface {
	Peek(ctx context.Context, key K) (Stored[V], bool, error)
}

// freshValueStore is the first cache policy check: stores that can prove a
// value is fresh return it directly before entry loading or flight handling.
type freshValueStore[K comparable, V any] interface {
	PeekFreshValue(ctx context.Context, key K, now time.Time) (V, bool, error)
}

func metricKey[K comparable](key K) string {
	if s, ok := any(key).(string); ok {
		return s
	}
	return fmt.Sprint(key)
}

func (c *Cache[K, V]) emitHit(key K) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricHit, Key: metricKey(key)})
}

func (c *Cache[K, V]) emitMiss(key K) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricMiss, Key: metricKey(key)})
}

func (c *Cache[K, V]) emitStaleHit(key K) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricStaleHit, Key: metricKey(key)})
}

func (c *Cache[K, V]) emitRefreshStart(key K) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricRefreshStart, Key: metricKey(key)})
}

func (c *Cache[K, V]) emitRefreshSuccess(key K, duration time.Duration) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricRefreshSuccess, Key: metricKey(key), Duration: duration})
}

func (c *Cache[K, V]) emitRefreshError(key K, err error) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricRefreshError, Key: metricKey(key), Err: err})
}

func (c *Cache[K, V]) emitSet(key K) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricSet, Key: metricKey(key)})
}

func (c *Cache[K, V]) emitDelete(key K) {
	if !c.metricsEnabled {
		return
	}
	c.metrics.RecordMetric(MetricEvent{Kind: MetricDelete, Key: metricKey(key)})
}

func (c *Cache[K, V]) getEntry(ctx context.Context, key K) (Stored[V], bool, error) {
	if c.peeker != nil {
		return c.peeker.Peek(ctx, key)
	}
	return c.store.Get(ctx, key)
}

func (c *Cache[K, V]) getFreshValue(ctx context.Context, key K, now time.Time) (V, bool, bool, error) {
	if store, ok := c.store.(freshValueStore[K, V]); ok {
		value, fresh, err := store.PeekFreshValue(ctx, key, now)
		return value, fresh, true, err
	}
	var zero V
	return zero, false, false, nil
}

func (c *Cache[K, V]) waitForFlight(key K) (V, bool, error) {
	c.flightMu.Lock()
	existing := c.flights[key]
	c.flightMu.Unlock()
	if existing == nil {
		var zero V
		return zero, false, nil
	}
	value, err := existing.wait()
	return value, true, err
}

func (c *Cache[K, V]) startFlight(key K) (*flight[V], bool) {
	c.flightMu.Lock()
	if existing := c.flights[key]; existing != nil {
		c.flightMu.Unlock()
		return existing, false
	}
	f := &flight[V]{}
	f.wg.Add(1)
	c.flights[key] = f
	c.flightMu.Unlock()
	return f, true
}

func (c *Cache[K, V]) finishFlight(key K, f *flight[V], value V, err error) {
	f.value = value
	f.err = err

	c.flightMu.Lock()
	delete(c.flights, key)
	c.flightMu.Unlock()
	f.wg.Done()
}

func (c *Cache[K, V]) finishPanickedFlight(key K, f *flight[V], panicValue any) {
	f.panicValue = panicValue
	f.panicked = true

	c.flightMu.Lock()
	delete(c.flights, key)
	c.flightMu.Unlock()
	f.wg.Done()
}

func (c *Cache[K, V]) do(key K, fn func() (V, error)) (value V, err error) {
	f, leader := c.startFlight(key)
	if !leader {
		return f.wait()
	}

	defer func() {
		if panicValue := recover(); panicValue != nil {
			c.finishPanickedFlight(key, f, panicValue)
			panic(panicValue)
		}
	}()

	value, err = fn()
	c.finishFlight(key, f, value, err)
	return value, err
}

func (c *Cache[K, V]) refresh(key K, compute func(context.Context) (V, error)) {
	f, leader := c.startFlight(key)
	if !leader {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.refreshTimeout)
		defer cancel()
		started := c.clock.Now()
		c.emitRefreshStart(key)
		value, err := compute(ctx)
		if err != nil {
			c.emitRefreshError(key, err)
			c.finishFlight(key, f, value, err)
			return
		}
		if err := c.Set(ctx, key, value); err != nil {
			c.emitRefreshError(key, err)
			c.finishFlight(key, f, value, err)
			return
		}
		c.emitRefreshSuccess(key, c.clock.Now().Sub(started))
		c.finishFlight(key, f, value, nil)
	}()
}

func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	var zero V
	if c.bypass {
		return zero, false, nil
	}
	if c.store == nil {
		return zero, false, ErrMissingStore
	}
	now := c.clock.Now()
	if value, ok, _, err := c.getFreshValue(ctx, key, now); err != nil || ok {
		if ok {
			c.emitHit(key)
		}
		return value, ok, err
	}
	entry, ok, err := c.getEntry(ctx, key)
	if err != nil || !ok {
		return zero, false, err
	}
	if entry.state(now) != entryFresh {
		return zero, false, nil
	}
	c.emitHit(key)
	return entry.Value, true, nil
}

func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) error {
	if c.bypass {
		return nil
	}
	if c.store == nil {
		return ErrMissingStore
	}
	now := c.clock.Now()
	entry := Stored[V]{Value: value, CreatedAt: now, NoExpire: c.noExpiration}
	if !c.noExpiration {
		entry.FreshUntil = now.Add(c.ttl)
		if c.staleTTL > 0 {
			entry.StaleUntil = entry.FreshUntil.Add(c.staleTTL)
		}
	}
	if err := c.store.Set(ctx, key, entry); err != nil {
		return err
	}
	c.emitSet(key)
	return nil
}

func (c *Cache[K, V]) Delete(ctx context.Context, key K) error {
	if c.store == nil {
		return ErrMissingStore
	}
	if err := c.store.Delete(ctx, key); err != nil {
		return err
	}
	c.emitDelete(key)
	return nil
}

func (c *Cache[K, V]) Clear(ctx context.Context) error {
	if c.store == nil {
		return ErrMissingStore
	}
	return c.store.Clear(ctx)
}

// GetOrCompute follows the cache policy order: fresh-value fast path,
// active-flight wait for non-stale caches, stored entry state handling,
// stale-while-revalidate, miss computation, then configured stale fallback on
// compute error.
func (c *Cache[K, V]) GetOrCompute(ctx context.Context, key K, compute func(context.Context) (V, error)) (V, error) {
	if c.bypass {
		c.emitMiss(key)
		return compute(ctx)
	}
	if c.store == nil {
		var zero V
		return zero, ErrMissingStore
	}
	now := c.clock.Now()
	value, ok, supportsFreshValue, err := c.getFreshValue(ctx, key, now)
	if err != nil || ok {
		if ok {
			c.emitHit(key)
		}
		return value, err
	}
	if supportsFreshValue && c.staleTTL == 0 && !c.keepStaleOnError {
		if value, ok, err := c.waitForFlight(key); ok || err != nil {
			return value, err
		}
	}
	entry, ok, err := c.getEntry(ctx, key)
	if err != nil {
		var zero V
		return zero, err
	}
	if ok {
		switch entry.state(now) {
		case entryFresh:
			c.emitHit(key)
			return entry.Value, nil
		case entryStale:
			c.emitStaleHit(key)
			c.refresh(key, compute)
			return entry.Value, nil
		}
	}
	c.emitMiss(key)
	value, err = c.do(key, func() (V, error) {
		now := c.clock.Now()
		if value, ok, _, err := c.getFreshValue(ctx, key, now); err != nil || ok {
			return value, err
		}
		entry, ok, err := c.getEntry(ctx, key)
		if err != nil {
			var zero V
			return zero, err
		}
		if ok && entry.state(now) == entryFresh {
			return entry.Value, nil
		}
		computed, computeErr := compute(ctx)
		if computeErr != nil {
			return computed, computeErr
		}
		return computed, c.Set(ctx, key, computed)
	})
	if err != nil && ok && c.keepStaleOnError {
		c.emitRefreshError(key, err)
		return entry.Value, nil
	}
	return value, err
}
