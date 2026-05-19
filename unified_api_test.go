package memoize_test

import (
	"context"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/memory"
)

func TestUnifiedRootMemoize1UsesDirectOptions(t *testing.T) {
	calls := 0
	cached, err := memoize.Memoize1(func(id int) string {
		calls++
		return "user"
	}, memoize.Opts().WithTTL(time.Minute))
	if err != nil {
		t.Fatalf("Memoize1 returned error: %v", err)
	}

	if got := cached(42); got != "user" {
		t.Fatalf("first call = %q", got)
	}
	if got := cached(42); got != "user" {
		t.Fatalf("second call = %q", got)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestUnifiedRootMemoizeCtx1EUsesDirectOptions(t *testing.T) {
	calls := 0
	cached, err := memoize.MemoizeCtx1E(func(ctx context.Context, id int) (string, error) {
		calls++
		return "user", ctx.Err()
	}, memoize.Opts().WithTTL(time.Minute))
	if err != nil {
		t.Fatalf("MemoizeCtx1E returned error: %v", err)
	}

	if got, err := cached(context.Background(), 42); err != nil || got != "user" {
		t.Fatalf("first call = %q, %v", got, err)
	}
	if got, err := cached(context.Background(), 42); err != nil || got != "user" {
		t.Fatalf("second call = %q, %v", got, err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestUnifiedRootCacheEngineUsesRootStores(t *testing.T) {
	cache, err := memoize.New[string, string](
		memoize.Opts().
			WithStore(memory.New[string, string](16)).
			WithTTL(time.Minute),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer cache.Stop()

	calls := 0
	compute := func(context.Context) (string, error) {
		calls++
		return "value", nil
	}
	if got, err := cache.GetOrCompute(context.Background(), "key", compute); err != nil || got != "value" {
		t.Fatalf("first GetOrCompute = %q, %v", got, err)
	}
	if got, err := cache.GetOrCompute(context.Background(), "key", compute); err != nil || got != "value" {
		t.Fatalf("second GetOrCompute = %q, %v", got, err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestUnifiedRootMemoize1UsesNonGenericOpts(t *testing.T) {
	cached, err := memoize.Memoize1(func(id int) string {
		return "user"
	}, memoize.Opts().WithTTL(time.Minute))
	if err != nil {
		t.Fatalf("Memoize1 returned error: %v", err)
	}
	if got := cached(42); got != "user" {
		t.Fatalf("cached value = %q", got)
	}
}
