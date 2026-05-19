package metrics_test

import (
	"errors"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/metrics"
)

func TestInMemoryMetrics_HitMissCount(t *testing.T) {
	m := metrics.NewInMemoryMetrics()

	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricHit, Key: "users"})
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricHit, Key: "users"})
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricMiss, Key: "users"})

	s := m.Stats()
	uc, ok := s["users"]
	if !ok {
		t.Fatal("expected stats for 'users'")
	}
	if uc.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", uc.Hits)
	}
	if uc.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", uc.Misses)
	}
	if uc.HitRatePercent < 66.0 || uc.HitRatePercent > 67.0 {
		t.Errorf("expected hit rate ~66.7%%, got %.2f", uc.HitRatePercent)
	}
}

func TestInMemoryMetrics_RefreshCounts(t *testing.T) {
	m := metrics.NewInMemoryMetrics()

	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricRefreshStart, Key: "feed"})
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricRefreshSuccess, Key: "feed", Duration: 10 * time.Millisecond})
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricRefreshStart, Key: "feed"})
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricRefreshError, Key: "feed", Err: errors.New("timeout")})

	s := m.Stats()
	fc := s["feed"]
	if fc.RefreshSuccess != 1 {
		t.Errorf("expected 1 success, got %d", fc.RefreshSuccess)
	}
	if fc.RefreshErrors != 1 {
		t.Errorf("expected 1 error, got %d", fc.RefreshErrors)
	}
}

func TestInMemoryMetrics_MultipleCaches(t *testing.T) {
	m := metrics.NewInMemoryMetrics()
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricHit, Key: "a"})
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricHit, Key: "a"})
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricMiss, Key: "b"})

	s := m.Stats()
	if _, ok := s["a"]; !ok {
		t.Fatal("expected stats for 'a'")
	}
	if _, ok := s["b"]; !ok {
		t.Fatal("expected stats for 'b'")
	}
}

func TestInMemoryMetrics_Reset(t *testing.T) {
	m := metrics.NewInMemoryMetrics()
	m.RecordMetric(memoize.MetricEvent{Kind: memoize.MetricHit, Key: "x"})
	m.Reset()
	s := m.Stats()
	if len(s) != 0 {
		t.Fatalf("expected empty stats after Reset, got %v", s)
	}
}
