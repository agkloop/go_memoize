package loader_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agkloop/go_memoize/loader"
)

func TestLoaderValue(t *testing.T) {
	calls := atomic.Int32{}
	l := loader.New[string](
		func(_ context.Context) (string, error) {
			calls.Add(1)
			return "hello", nil
		},
		50*time.Millisecond,
	)
	defer l.Stop()

	v, err := l.Value(context.Background())
	if err != nil || v != "hello" {
		t.Fatalf("expected 'hello', got %q, err=%v", v, err)
	}
}

func TestLoaderRefreshes(t *testing.T) {
	counter := atomic.Int32{}
	l := loader.New[int](
		func(_ context.Context) (int, error) {
			return int(counter.Add(1)), nil
		},
		30*time.Millisecond,
	)
	defer l.Stop()

	// Wait long enough for at least 3 ticks
	time.Sleep(120 * time.Millisecond)
	v, _ := l.Value(context.Background())
	if v < 2 {
		t.Fatalf("expected at least 2 refreshes, got %d", v)
	}
}

func TestLoaderKeepsStaleOnError(t *testing.T) {
	first := atomic.Bool{}
	first.Store(true)

	l := loader.New[string](
		func(_ context.Context) (string, error) {
			if first.CompareAndSwap(true, false) {
				return "good", nil
			}
			return "", errors.New("transient")
		},
		30*time.Millisecond,
	)
	defer l.Stop()

	// First call: get the good value
	v, err := l.Value(context.Background())
	if err != nil || v != "good" {
		t.Fatalf("expected 'good': v=%q err=%v", v, err)
	}

	// Wait for at least one failing refresh
	time.Sleep(70 * time.Millisecond)

	// Stale value must still be returned
	v, err = l.Value(context.Background())
	if err != nil || v != "good" {
		t.Fatalf("expected stale 'good' after error: v=%q err=%v", v, err)
	}
}

func TestLoaderKeepsStaleOnRefreshErrorAndCallsOnError(t *testing.T) {
	first := atomic.Bool{}
	first.Store(true)
	errored := make(chan error, 1)

	l := loader.New[string](
		func(_ context.Context) (string, error) {
			if first.CompareAndSwap(true, false) {
				return "good", nil
			}
			return "", errors.New("transient")
		},
		20*time.Millisecond,
		loader.WithOnError[string](func(err error) {
			select {
			case errored <- err:
			default:
			}
		}),
	)
	defer l.Stop()

	v, err := l.Value(context.Background())
	if err != nil || v != "good" {
		t.Fatalf("expected 'good': v=%q err=%v", v, err)
	}

	select {
	case err := <-errored:
		if err == nil || err.Error() != "transient" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("OnError was not called for refresh error")
	}

	v, err = l.Value(context.Background())
	if err != nil || v != "good" {
		t.Fatalf("expected stale 'good' after error: v=%q err=%v", v, err)
	}
}

func TestLoaderStop(t *testing.T) {
	calls := atomic.Int32{}
	l := loader.New[int](
		func(_ context.Context) (int, error) {
			calls.Add(1)
			return 1, nil
		},
		20*time.Millisecond,
	)
	l.Stop()
	before := calls.Load()
	time.Sleep(60 * time.Millisecond)
	after := calls.Load()
	if after > before+1 { // allow one in-flight call
		t.Fatalf("loader continued after Stop: before=%d after=%d", before, after)
	}
}

func TestLoaderOnError(t *testing.T) {
	errored := make(chan error, 1)
	l := loader.New[string](
		func(_ context.Context) (string, error) {
			return "", errors.New("boom")
		},
		20*time.Millisecond,
		loader.WithOnError[string](func(err error) {
			select {
			case errored <- err:
			default:
			}
		}),
	)
	defer l.Stop()

	// Expect an error to be reported
	select {
	case err := <-errored:
		if err == nil || err.Error() != "boom" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("OnError was not called within timeout")
	}
}
