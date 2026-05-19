package benchmarks

import (
	"testing"
	"time"
)

func TestLRUMemoizeCachesZeroArgValue(t *testing.T) {
	calls := 0
	memoized := lruMemoize(func() int {
		calls++
		return calls
	}, time.Minute, 1)

	if got := memoized(); got != 1 {
		t.Fatalf("first call = %d, want 1", got)
	}
	if got := memoized(); got != 1 {
		t.Fatalf("second call = %d, want cached 1", got)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestLRUMemoize1CachesPerKey(t *testing.T) {
	calls := 0
	memoized := lruMemoize1(func(key string) int {
		calls++
		return len(key) + calls
	}, time.Minute, 2)

	firstA := memoized("a")
	firstB := memoized("bb")
	secondA := memoized("a")

	if firstA != secondA {
		t.Fatalf("second call for same key = %d, want cached %d", secondA, firstA)
	}
	if firstB == firstA {
		t.Fatalf("different keys produced same value %d", firstB)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
