package metrics

import (
	"sort"
	"sync"
	"time"

	memoize "github.com/agkloop/go_memoize"
)

// CacheStats holds aggregated metrics for a single metric key.
type CacheStats struct {
	Hits           int64
	Misses         int64
	StaleHits      int64
	Sets           int64
	Deletes        int64
	RefreshSuccess int64
	RefreshErrors  int64
	HitRatePercent float64
	// Latency percentiles (milliseconds), populated if latency samples exist.
	LatencyP50Ms float64
	LatencyP95Ms float64
	LatencyP99Ms float64
}

type cacheEntry struct {
	mu             sync.Mutex
	hits           int64
	misses         int64
	staleHits      int64
	sets           int64
	deletes        int64
	refreshSuccess int64
	refreshErrors  int64
	// latency samples in milliseconds, capped at 1024 (ring buffer).
	latencies [1024]float64
	latLen    int
	latHead   int
}

func (e *cacheEntry) addLatency(d time.Duration) {
	e.mu.Lock()
	e.latencies[e.latHead] = float64(d.Milliseconds())
	e.latHead = (e.latHead + 1) % len(e.latencies)
	if e.latLen < len(e.latencies) {
		e.latLen++
	}
	e.mu.Unlock()
}

func (e *cacheEntry) percentiles() (p50, p95, p99 float64) {
	e.mu.Lock()
	if e.latLen == 0 {
		e.mu.Unlock()
		return 0, 0, 0
	}
	samples := make([]float64, e.latLen)
	copy(samples, e.latencies[:e.latLen])
	e.mu.Unlock()
	sort.Float64s(samples)
	idx := func(p float64) float64 {
		i := int(p * float64(len(samples)-1))
		return samples[i]
	}
	return idx(0.50), idx(0.95), idx(0.99)
}

// InMemoryMetrics is a thread-safe implementation that accumulates
// hit/miss/refresh counts and latency percentiles in memory.
type InMemoryMetrics struct {
	mu     sync.RWMutex
	caches map[string]*cacheEntry
}

// NewInMemoryMetrics returns a ready-to-use InMemoryMetrics.
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{caches: make(map[string]*cacheEntry)}
}

func (m *InMemoryMetrics) entry(name string) *cacheEntry {
	m.mu.RLock()
	e := m.caches[name]
	m.mu.RUnlock()
	if e != nil {
		return e
	}
	m.mu.Lock()
	if m.caches[name] == nil {
		m.caches[name] = &cacheEntry{}
	}
	e = m.caches[name]
	m.mu.Unlock()
	return e
}

func (m *InMemoryMetrics) RecordMetric(event memoize.MetricEvent) {
	if event.Kind == memoize.MetricRefreshStart {
		return
	}
	e := m.entry(event.Key)
	e.mu.Lock()
	switch event.Kind {
	case memoize.MetricHit:
		e.hits++
	case memoize.MetricMiss:
		e.misses++
	case memoize.MetricStaleHit:
		e.staleHits++
	case memoize.MetricSet:
		e.sets++
	case memoize.MetricDelete:
		e.deletes++
	case memoize.MetricRefreshSuccess:
		e.refreshSuccess++
	case memoize.MetricRefreshError:
		e.refreshErrors++
	}
	e.mu.Unlock()
	if event.Kind == memoize.MetricRefreshSuccess {
		e.addLatency(event.Duration)
	}
}

// Stats returns a snapshot of all metrics keyed by MetricEvent.Key.
func (m *InMemoryMetrics) Stats() map[string]CacheStats {
	m.mu.RLock()
	names := make([]string, 0, len(m.caches))
	for n := range m.caches {
		names = append(names, n)
	}
	m.mu.RUnlock()

	out := make(map[string]CacheStats, len(names))
	for _, name := range names {
		e := m.entry(name)
		e.mu.Lock()
		cs := CacheStats{
			Hits:           e.hits,
			Misses:         e.misses,
			StaleHits:      e.staleHits,
			Sets:           e.sets,
			Deletes:        e.deletes,
			RefreshSuccess: e.refreshSuccess,
			RefreshErrors:  e.refreshErrors,
		}
		e.mu.Unlock()
		total := cs.Hits + cs.Misses
		if total > 0 {
			cs.HitRatePercent = float64(cs.Hits) / float64(total) * 100
		}
		cs.LatencyP50Ms, cs.LatencyP95Ms, cs.LatencyP99Ms = e.percentiles()
		out[name] = cs
	}
	return out
}

// Reset clears all accumulated metrics.
func (m *InMemoryMetrics) Reset() {
	m.mu.Lock()
	m.caches = make(map[string]*cacheEntry)
	m.mu.Unlock()
}
