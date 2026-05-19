package memoize_test

import (
	"context"
	"sync"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/memory"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

type recordingMetrics struct {
	mu     sync.Mutex
	events []memoize.MetricEvent
}

func (m *recordingMetrics) RecordMetric(event memoize.MetricEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *recordingMetrics) count(kind memoize.MetricEventKind) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, event := range m.events {
		if event.Kind == kind {
			count++
		}
	}
	return count
}

func (m *recordingMetrics) contains(kind memoize.MetricEventKind, key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, event := range m.events {
		if event.Kind == kind && event.Key == key {
			return true
		}
	}
	return false
}

func (m *recordingMetrics) waitFor(kind memoize.MetricEventKind, key string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if m.contains(kind, key) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

type recordingPeekStore[V any] struct {
	entry     memoize.Stored[V]
	ok        bool
	getCalls  int
	peekCalls int
}

func (s *recordingPeekStore[V]) Get(context.Context, string) (memoize.Stored[V], bool, error) {
	s.getCalls++
	return s.entry, s.ok, nil
}

func (s *recordingPeekStore[V]) Peek(context.Context, string) (memoize.Stored[V], bool, error) {
	s.peekCalls++
	return s.entry, s.ok, nil
}

func (s *recordingPeekStore[V]) Set(_ context.Context, _ string, value memoize.Stored[V]) error {
	s.entry = value
	s.ok = true
	return nil
}

func (s *recordingPeekStore[V]) Delete(context.Context, string) error {
	s.ok = false
	return nil
}

func (s *recordingPeekStore[V]) Clear(context.Context) error {
	s.ok = false
	return nil
}

func TestGetOrComputeCachesFreshValue(t *testing.T) {
	ctx := context.Background()
	clock := &testClock{now: time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)}
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[string, string](1024)).WithTTL(time.Minute).WithClock(clock))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	calls := 0
	compute := func(context.Context) (string, error) {
		calls++
		return "value", nil
	}

	first, err := cache.GetOrCompute(ctx, "key", compute)
	if err != nil || first != "value" {
		t.Fatalf("first call returned %q err=%v", first, err)
	}
	second, err := cache.GetOrCompute(ctx, "key", compute)
	if err != nil || second != "value" {
		t.Fatalf("second call returned %q err=%v", second, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 compute call, got %d", calls)
	}
}

func TestGetOrComputeUsesPeekForFreshHit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	store := &recordingPeekStore[string]{
		entry: memoize.Stored[string]{Value: "value", CreatedAt: now, FreshUntil: now.Add(time.Minute)},
		ok:    true,
	}
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(store).WithTTL(time.Minute).WithClock(&testClock{now: now}))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}

	got, err := cache.GetOrCompute(ctx, "key", func(context.Context) (string, error) {
		t.Fatal("compute should not run for fresh hit")
		return "", nil
	})
	if err != nil || got != "value" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if store.peekCalls != 1 || store.getCalls != 0 {
		t.Fatalf("expected one peek and no get, got peek=%d get=%d", store.peekCalls, store.getCalls)
	}
}

func TestBypassComputesWithoutStoring(t *testing.T) {
	ctx := context.Background()
	cache, err := memoize.New[string, int](memoize.Opts().Bypass())
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	calls := 0
	compute := func(context.Context) (int, error) {
		calls++
		return calls, nil
	}
	first, err := cache.GetOrCompute(ctx, "key", compute)
	if err != nil || first != 1 {
		t.Fatalf("first call returned %d err=%v", first, err)
	}
	second, err := cache.GetOrCompute(ctx, "key", compute)
	if err != nil || second != 2 {
		t.Fatalf("second call returned %d err=%v", second, err)
	}
}

func TestDeleteAndClear(t *testing.T) {
	ctx := context.Background()
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[string, string](1024)).NoExpiration())
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	if err := cache.Set(ctx, "a", "one"); err != nil {
		t.Fatalf("set a failed: %v", err)
	}
	if err := cache.Set(ctx, "b", "two"); err != nil {
		t.Fatalf("set b failed: %v", err)
	}
	if err := cache.Delete(ctx, "a"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, ok, err := cache.Get(ctx, "a"); err != nil || ok {
		t.Fatalf("deleted key returned ok=%v err=%v", ok, err)
	}
	if err := cache.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if _, ok, err := cache.Get(ctx, "b"); err != nil || ok {
		t.Fatalf("cleared key returned ok=%v err=%v", ok, err)
	}
}

func TestStaleHitReturnsStaleAndRefreshes(t *testing.T) {
	ctx := context.Background()
	clock := &testClock{now: time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)}
	metrics := &recordingMetrics{}
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[string, string](1024)).WithTTL(time.Second).WithStaleTTL(time.Minute).WithClock(clock).WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	if err := cache.Set(ctx, "key", "old"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	clock.now = clock.now.Add(2 * time.Second)
	refreshDone := make(chan struct{})
	got, err := cache.GetOrCompute(ctx, "key", func(context.Context) (string, error) {
		defer close(refreshDone)
		return "new", nil
	})
	if err != nil || got != "old" {
		t.Fatalf("stale call returned %q err=%v", got, err)
	}
	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("refresh did not complete")
	}
	if !metrics.waitFor(memoize.MetricRefreshSuccess, "key", time.Second) {
		t.Fatalf("refresh success metric was not recorded: %#v", metrics)
	}
	deadline := time.After(time.Second)
	for {
		got, ok, err := cache.Get(ctx, "key")
		if err != nil {
			t.Fatalf("refreshed value error: %v", err)
		}
		if ok && got == "new" {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("refreshed value=%q ok=%v, want new", got, ok)
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if metrics.count(memoize.MetricStaleHit) != 1 || metrics.count(memoize.MetricRefreshStart) != 1 || metrics.count(memoize.MetricRefreshSuccess) != 1 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}

func TestKeepStaleOnErrorAfterStaleWindow(t *testing.T) {
	ctx := context.Background()
	clock := &testClock{now: time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)}
	metrics := &recordingMetrics{}
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[string, string](1024)).WithTTL(time.Second).WithStaleTTL(time.Second).KeepStaleOnError().WithClock(clock).WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	if err := cache.Set(ctx, "key", "old"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	clock.now = clock.now.Add(3 * time.Second)
	got, err := cache.GetOrCompute(ctx, "key", func(context.Context) (string, error) {
		return "", context.Canceled
	})
	if err != nil || got != "old" {
		t.Fatalf("expected stale fallback, got %q err=%v", got, err)
	}
	if metrics.count(memoize.MetricRefreshError) != 1 {
		t.Fatalf("expected refresh error metric, got %#v", metrics)
	}
}

func TestWithMetricsRecordsTypedEvents(t *testing.T) {
	ctx := context.Background()
	clock := &testClock{now: time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)}
	metrics := &recordingMetrics{}
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[string, string](1024)).WithTTL(time.Second).WithStaleTTL(time.Minute).WithClock(clock).WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}

	if _, err := cache.GetOrCompute(ctx, "typed", func(context.Context) (string, error) { return "fresh", nil }); err != nil {
		t.Fatalf("miss compute failed: %v", err)
	}
	if _, err := cache.GetOrCompute(ctx, "typed", func(context.Context) (string, error) {
		t.Fatal("compute should not run for fresh hit")
		return "", nil
	}); err != nil {
		t.Fatalf("hit get failed: %v", err)
	}
	clock.now = clock.now.Add(2 * time.Second)
	refreshDone := make(chan struct{})
	if _, err := cache.GetOrCompute(ctx, "typed", func(context.Context) (string, error) {
		defer close(refreshDone)
		return "refreshed", nil
	}); err != nil {
		t.Fatalf("stale hit failed: %v", err)
	}
	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("refresh did not complete")
	}
	if !metrics.waitFor(memoize.MetricRefreshSuccess, "typed", time.Second) {
		t.Fatalf("refresh success metric was not recorded: %#v", metrics)
	}
	if err := cache.Delete(ctx, "typed"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if err := cache.Set(ctx, "error", "old"); err != nil {
		t.Fatalf("set error key failed: %v", err)
	}
	clock.now = clock.now.Add(2 * time.Second)
	refreshErrorDone := make(chan struct{})
	if _, err := cache.GetOrCompute(ctx, "error", func(context.Context) (string, error) {
		defer close(refreshErrorDone)
		return "", context.Canceled
	}); err != nil {
		t.Fatalf("stale error refresh returned err=%v", err)
	}
	select {
	case <-refreshErrorDone:
	case <-time.After(time.Second):
		t.Fatal("error refresh did not complete")
	}
	if !metrics.waitFor(memoize.MetricRefreshError, "error", time.Second) {
		t.Fatalf("refresh error metric was not recorded: %#v", metrics)
	}

	for _, want := range []memoize.MetricEventKind{
		memoize.MetricMiss,
		memoize.MetricSet,
		memoize.MetricHit,
		memoize.MetricStaleHit,
		memoize.MetricRefreshStart,
		memoize.MetricRefreshSuccess,
		memoize.MetricDelete,
		memoize.MetricRefreshError,
	} {
		if metrics.count(want) == 0 {
			t.Fatalf("missing metric kind %v in %#v", want, metrics.events)
		}
	}
	if !metrics.contains(memoize.MetricMiss, "typed") || !metrics.contains(memoize.MetricDelete, "typed") || !metrics.contains(memoize.MetricRefreshError, "error") {
		t.Fatalf("events recorded wrong keys: %#v", metrics.events)
	}
}

func TestSynchronousComputeRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[string, string](1024)).WithTTL(time.Minute))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	_, err = cache.GetOrCompute(ctx, "key", func(ctx context.Context) (string, error) {
		return "", ctx.Err()
	})
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestConcurrentMissComputesOnce(t *testing.T) {
	ctx := context.Background()
	cache, err := memoize.New[string, int](memoize.Opts().WithStore(memory.New[string, int](1024)).WithTTL(time.Minute))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	start := make(chan struct{})
	done := make(chan int, 8)
	calls := 0
	compute := func(context.Context) (int, error) {
		calls++
		<-start
		return 42, nil
	}
	for i := 0; i < 8; i++ {
		go func() {
			value, err := cache.GetOrCompute(ctx, "key", compute)
			if err != nil {
				done <- -1
				return
			}
			done <- value
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(start)
	for i := 0; i < 8; i++ {
		if value := <-done; value != 42 {
			t.Fatalf("goroutine returned %d", value)
		}
	}
	if calls != 1 {
		t.Fatalf("expected 1 compute call, got %d", calls)
	}
}
