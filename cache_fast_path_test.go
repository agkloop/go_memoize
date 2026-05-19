package memoize

import (
	"context"
	"testing"
	"time"
)

type fastPathClock struct{ now time.Time }

func (c *fastPathClock) Now() time.Time { return c.now }

const fastPathTestTimeout = time.Second
const fastPathNoResultWindow = 100 * time.Millisecond

func waitForFastPathSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(fastPathTestTimeout):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitForFastPathResult(t *testing.T, ch <-chan string, name string) string {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(fastPathTestTimeout):
		t.Fatalf("timed out waiting for %s", name)
	}
	return ""
}

func assertNoFastPathResult(t *testing.T, result <-chan string, computeCalled <-chan struct{}) {
	t.Helper()
	select {
	case got := <-result:
		t.Fatalf("GetOrCompute returned %q before active flight completed", got)
	case <-computeCalled:
		t.Fatal("compute ran while an active flight exists")
	case <-time.After(fastPathNoResultWindow):
	}
}

type recordingFreshStore struct {
	entry       Stored[string]
	ok          bool
	freshCalls  int
	peekCalls   int
	getCalls    int
	freshSeen   chan struct{}
	freshReturn chan struct{}
}

func (s *recordingFreshStore) PeekFreshValue(context.Context, string, time.Time) (string, bool, error) {
	s.freshCalls++
	if s.freshSeen != nil {
		close(s.freshSeen)
	}
	if s.freshReturn != nil {
		<-s.freshReturn
	}
	return s.entry.Value, s.ok, nil
}

func (s *recordingFreshStore) Get(context.Context, string) (Stored[string], bool, error) {
	s.getCalls++
	return s.entry, s.ok, nil
}

func (s *recordingFreshStore) Peek(context.Context, string) (Stored[string], bool, error) {
	s.peekCalls++
	return s.entry, s.ok, nil
}

func (s *recordingFreshStore) Set(_ context.Context, _ string, value Stored[string]) error {
	s.entry = value
	s.ok = true
	return nil
}

func (s *recordingFreshStore) Delete(context.Context, string) error {
	s.ok = false
	return nil
}

func (s *recordingFreshStore) Clear(context.Context) error {
	s.ok = false
	return nil
}

func TestGetOrComputeUsesFreshValuePathForFreshHit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	store := &recordingFreshStore{
		entry:       Stored[string]{Value: "value", CreatedAt: now, FreshUntil: now.Add(time.Minute)},
		ok:          true,
		freshSeen:   make(chan struct{}),
		freshReturn: make(chan struct{}),
	}
	cache, err := New[string, string](Opts().WithStore(store).WithTTL(time.Minute).WithClock(&fastPathClock{now: now}))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}

	result := make(chan string, 1)
	go func() {
		got, err := cache.GetOrCompute(ctx, "key", func(context.Context) (string, error) {
			t.Error("compute should not run for fresh hit")
			return "", nil
		})
		if err != nil {
			t.Errorf("GetOrCompute returned error: %v", err)
		}
		result <- got
	}()
	waitForFastPathSignal(t, store.freshSeen, "fresh-value seam")
	if _, ok, err := cache.waitForFlight("key"); err != nil || ok {
		t.Fatalf("fresh-value hit should not start a flight, ok=%v err=%v", ok, err)
	}
	close(store.freshReturn)

	got := waitForFastPathResult(t, result, "fresh-value result")
	if got != "value" {
		t.Fatalf("got %q, want value", got)
	}
	if store.freshCalls != 1 || store.peekCalls != 0 || store.getCalls != 0 {
		t.Fatalf("expected one fresh-value call only, got fresh=%d peek=%d get=%d", store.freshCalls, store.peekCalls, store.getCalls)
	}
}

func TestGetOrComputeWaitsForActiveFlightAfterFreshMiss(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	store := &recordingFreshStore{freshSeen: make(chan struct{}), freshReturn: make(chan struct{})}
	cache, err := New[string, string](Opts().WithStore(store).WithTTL(time.Minute).WithClock(&fastPathClock{now: now}))
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}

	f := &flight[string]{}
	f.wg.Add(1)
	cache.flightMu.Lock()
	cache.flights["key"] = f
	cache.flightMu.Unlock()
	defer func() {
		cache.flightMu.Lock()
		delete(cache.flights, "key")
		cache.flightMu.Unlock()
	}()

	result := make(chan string, 1)
	computeCalled := make(chan struct{})
	go func() {
		got, err := cache.GetOrCompute(ctx, "key", func(context.Context) (string, error) {
			close(computeCalled)
			t.Error("compute should not run while an active flight exists")
			return "", nil
		})
		if err != nil {
			t.Errorf("GetOrCompute returned error: %v", err)
		}
		result <- got
	}()
	waitForFastPathSignal(t, store.freshSeen, "fresh-value seam")
	close(store.freshReturn)
	assertNoFastPathResult(t, result, computeCalled)
	f.value = "computed"
	f.wg.Done()

	if got := waitForFastPathResult(t, result, "active-flight result"); got != "computed" {
		t.Fatalf("got %q, want computed", got)
	}
	if store.freshCalls != 1 || store.peekCalls != 0 || store.getCalls != 0 {
		t.Fatalf("expected fresh miss then active-flight wait without peek/get, got fresh=%d peek=%d get=%d", store.freshCalls, store.peekCalls, store.getCalls)
	}
}
