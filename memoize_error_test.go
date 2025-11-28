package go_memoize

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestMemoize1E_DoesNotCacheError(t *testing.T) {
	calls := 0
	fn := func(k int) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("transient failure")
		}
		return "ok", nil
	}
	m := Memoize1E(fn, time.Minute)

	// First call should return error and not be cached
	if v, err := m(1); err == nil {
		t.Fatalf("expected error on first call, got value=%q", v)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call after first invocation, got %d", calls)
	}

	// Second call should succeed and be cached
	v, err := m(1)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if v != "ok" {
		t.Fatalf("unexpected value on second call: %q", v)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls after second invocation, got %d", calls)
	}

	// Third call should be served from cache (no additional calls)
	v2, err2 := m(1)
	if err2 != nil || v2 != "ok" {
		t.Fatalf("unexpected result on third call: value=%q err=%v", v2, err2)
	}
	if calls != 2 {
		t.Fatalf("expected no additional compute calls for cached value, got %d", calls)
	}
}

func TestMemoizeE_DoesNotCacheError(t *testing.T) {
	calls := 0
	fn := func() (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("transient failure")
		}
		return "ok", nil
	}
	m := MemoizeE(fn, time.Minute)

	// First call fails and should not be cached
	if v, err := m(); err == nil {
		t.Fatalf("expected error on first call, got value=%q", v)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call after first invocation, got %d", calls)
	}

	// Second call succeeds and should be cached
	v, err := m()
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if v != "ok" {
		t.Fatalf("unexpected value on second call: %q", v)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls after second invocation, got %d", calls)
	}

	// Third call should be served from cache
	v2, err2 := m()
	if err2 != nil || v2 != "ok" {
		t.Fatalf("unexpected result on third call: value=%q err=%v", v2, err2)
	}
	if calls != 2 {
		t.Fatalf("expected no additional compute calls for cached value, got %d", calls)
	}
}

func TestMemoize2E_DoesNotCacheError(t *testing.T) {
	calls := 0
	fn := func(a int, b string) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("transient failure")
		}
		return fmt.Sprintf("%d-%s", a, b), nil
	}
	m := Memoize2E(fn, time.Minute)

	// First call fails and should not be cached
	if v, err := m(5, "x"); err == nil {
		t.Fatalf("expected error on first call, got value=%q", v)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call after first invocation, got %d", calls)
	}

	// Second call succeeds and should be cached
	v, err := m(5, "x")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if v != "5-x" {
		t.Fatalf("unexpected value on second call: %q", v)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls after second invocation, got %d", calls)
	}

	// Third call served from cache
	v2, err2 := m(5, "x")
	if err2 != nil || v2 != "5-x" {
		t.Fatalf("unexpected result on third call: value=%q err=%v", v2, err2)
	}
	if calls != 2 {
		t.Fatalf("expected no additional compute calls for cached value, got %d", calls)
	}
}

func TestMemoizeCtx1E_DoesNotCacheError(t *testing.T) {
	calls := 0
	fn := func(ctx context.Context, k int) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("transient failure")
		}
		return "ok", nil
	}
	m := MemoizeCtx1E(fn, time.Minute)

	// First call should return error and not be cached
	if v, err := m(context.Background(), 42); err == nil {
		t.Fatalf("expected error on first call, got value=%q", v)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call after first invocation, got %d", calls)
	}

	// Second call should succeed and be cached
	v, err := m(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if v != "ok" {
		t.Fatalf("unexpected value on second call: %q", v)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls after second invocation, got %d", calls)
	}

	// Third call should be served from cache (no additional calls)
	v2, err2 := m(context.Background(), 42)
	if err2 != nil || v2 != "ok" {
		t.Fatalf("unexpected result on third call: value=%q err=%v", v2, err2)
	}
	if calls != 2 {
		t.Fatalf("expected no additional compute calls for cached value, got %d", calls)
	}
}
