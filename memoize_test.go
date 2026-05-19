package memoize

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func mustMemoized[F any](fn F, err error) F {
	if err != nil {
		panic(err)
	}
	return fn
}

func TestMemoize2WithOptionsDefaultStoreCaches(t *testing.T) {
	calls := 0
	cached, err := Memoize2(func(org string, id int) string {
		calls++
		return org + ":" + strconv.Itoa(id)
	}, Opts().WithTTL(time.Minute))
	if err != nil {
		t.Fatalf("Memoize2 returned error: %v", err)
	}
	if got := cached("acme", 42); got != "acme:42" {
		t.Fatalf("first call = %q", got)
	}
	if got := cached("acme", 42); got != "acme:42" {
		t.Fatalf("second call = %q", got)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestMemoize2WithOptionsInvalidTTLReturnsError(t *testing.T) {
	_, err := Memoize2(func(org string, id int) string { return org }, Opts().WithTTL(0))
	if !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("err = %v, want ErrInvalidTTL", err)
	}
}

func TestDirectStoreStoresRawEntries(t *testing.T) {
	store := newDirectStore[string]()
	now := time.Now()
	stale := Stored[string]{Value: "cached", CreatedAt: now.Add(-2 * time.Minute), FreshUntil: now.Add(-time.Minute)}
	if err := store.Set(context.Background(), 7, stale); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	got, ok, err := store.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !ok || got.Value != "cached" || !got.FreshUntil.Equal(stale.FreshUntil) {
		t.Fatalf("got = %#v, ok = %v", got, ok)
	}
}

func TestMemoizeCtx1EKeepStaleOnErrorReturnsStaleValue(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	calls := 0
	cached, err := MemoizeCtx1E(func(context.Context, int) (string, error) {
		calls++
		if calls > 1 {
			return "", errors.New("upstream unavailable")
		}
		return "fresh", nil
	}, Opts().
		WithTTL(time.Second).
		WithStaleTTL(time.Minute).
		KeepStaleOnError().
		WithClock(ClockFunc(func() time.Time { return now })))
	if err != nil {
		t.Fatalf("MemoizeCtx1E returned error: %v", err)
	}

	first, err := cached(context.Background(), 42)
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	if first != "fresh" {
		t.Fatalf("first call = %q, want fresh", first)
	}

	now = now.Add(2 * time.Minute)
	second, err := cached(context.Background(), 42)
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	if second != "fresh" {
		t.Fatalf("second call = %q, want stale fresh", second)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestMemoizeWithTTL(t *testing.T) {
	count := 0
	computeFn := func() int {
		count++
		return 1
	}
	memoizedFn := mustMemoized(Memoize(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn()
	memoizedFn()
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn()
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}

}

func TestMemoize1WithTTL(t *testing.T) {
	count := 0
	computeFn := func(key int) int {
		count++
		return key * 2
	}
	memoizedFn := mustMemoized(Memoize1(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn(21)
	memoizedFn(21)
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn(21)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize2WithTTL(t *testing.T) {
	count := 0
	computeFn := func(key1, key2 int) int {
		count++
		return key1 + key2
	}
	memoizedFn := mustMemoized(Memoize2(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn(20, 22)
	memoizedFn(20, 22)
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn(20, 22)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize3WithTTL(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3 int) int {
		count++
		return key1 + key2 + key3
	}
	memoizedFn := mustMemoized(Memoize3(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn(10, 20, 12)
	memoizedFn(10, 20, 12)
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn(10, 20, 12)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize4WithTTL(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4 int) int {
		count++
		return key1 + key2 + key3 + key4
	}
	memoizedFn := mustMemoized(Memoize4(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn(10, 10, 10, 12)
	memoizedFn(10, 10, 10, 12)
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn(10, 10, 10, 12)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}
func TestMemoizeWithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func() int {
		count++
		return 1
	}
	memoizedFn := mustMemoized(Memoize(computeFn, Opts().NoExpiration()))
	memoizedFn()
	memoizedFn()
	if count != 1 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize1WithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func(key int) int {
		count++
		return key * 2
	}
	memoizedFn := mustMemoized(Memoize1(computeFn, Opts().NoExpiration()))
	memoizedFn(21)
	memoizedFn(21)
	if count != 1 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize2WithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func(key1, key2 int) int {
		count++
		return key1 + key2
	}
	memoizedFn := mustMemoized(Memoize2(computeFn, Opts().NoExpiration()))
	memoizedFn(20, 22)
	memoizedFn(20, 22)
	if count != 1 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize3WithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3 int) int {
		count++
		return key1 + key2 + key3
	}
	memoizedFn := mustMemoized(Memoize3(computeFn, Opts().NoExpiration()))
	memoizedFn(10, 20, 12)
	memoizedFn(10, 20, 12)
	memoizedFn(10, 20, 12)
	memoizedFn(10, 20, 12)
	if count != 1 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize4WithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4 int) int {
		count++
		return key1 + key2 + key3 + key4
	}
	memoizedFn := mustMemoized(Memoize4(computeFn, Opts().NoExpiration()))
	memoizedFn(10, 10, 10, 12)
	memoizedFn(10, 10, 10, 12)
	memoizedFn(10, 11, 10, 12)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize5WithTTL(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4, key5 int) int {
		count++
		return key1 + key2 + key3 + key4 + key5
	}
	memoizedFn := mustMemoized(Memoize5(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn(1, 2, 3, 4, 5)
	memoizedFn(1, 2, 3, 4, 5)
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn(1, 2, 3, 4, 5)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize6WithTTL(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4, key5, key6 int) int {
		count++
		return key1 + key2 + key3 + key4 + key5 + key6
	}
	memoizedFn := mustMemoized(Memoize6(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn(1, 2, 3, 4, 5, 6)
	memoizedFn(1, 2, 3, 4, 5, 6)
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn(1, 2, 3, 4, 5, 6)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize7WithTTL(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4, key5, key6, key7 int) int {
		count++
		return key1 + key2 + key3 + key4 + key5 + key6 + key7
	}
	memoizedFn := mustMemoized(Memoize7(computeFn, Opts().WithTTL(1*time.Second)))
	memoizedFn(1, 2, 3, 4, 5, 6, 7)
	memoizedFn(1, 2, 3, 4, 5, 6, 7)
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}

	time.Sleep(2 * time.Second)
	memoizedFn(1, 2, 3, 4, 5, 6, 7)
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize5WithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4, key5 int) int {
		count++
		return key1 + key2 + key3 + key4 + key5
	}
	memoizedFn := mustMemoized(Memoize5(computeFn, Opts().NoExpiration()))
	memoizedFn(1, 2, 3, 4, 5)
	memoizedFn(1, 2, 3, 4, 5)
	if count != 1 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize6WithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4, key5, key6 int) int {
		count++
		return key1 + key2 + key3 + key4 + key5 + key6
	}
	memoizedFn := mustMemoized(Memoize6(computeFn, Opts().NoExpiration()))
	memoizedFn(1, 2, 3, 4, 5, 6)
	memoizedFn(1, 2, 3, 4, 5, 6)
	if count != 1 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoize7WithTTL_NoExpiry(t *testing.T) {
	count := 0
	computeFn := func(key1, key2, key3, key4, key5, key6, key7 int) int {
		count++
		return key1 + key2 + key3 + key4 + key5 + key6 + key7
	}
	memoizedFn := mustMemoized(Memoize7(computeFn, Opts().NoExpiration()))
	memoizedFn(1, 2, 3, 4, 5, 6, 7)
	memoizedFn(1, 2, 3, 4, 5, 6, 7)
	memoizedFn(1, 2, 3, 4, 5, 44, 77)
	memoizedFn(1, 2, 3, 4, 5, 233, 1000)
	if count != 3 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoizeWithTTL_ConcurrentAccess(t *testing.T) {
	var count int32
	computeFn := func() int {
		atomic.AddInt32(&count, 1)
		return 1
	}
	memoizedFn := mustMemoized(Memoize(computeFn, Opts().WithTTL(10*time.Second)))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			memoizedFn()
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestMemoize1WithTTL_ConcurrentAccess(t *testing.T) {
	var count int32
	computeFn := func(key int) int {
		atomic.AddInt32(&count, 1)
		return key * 2
	}
	memoizedFn := mustMemoized(Memoize1(computeFn, Opts().WithTTL(1*time.Minute)))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			memoizedFn(21)
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestMemoize2WithTTL_ConcurrentAccess(t *testing.T) {
	var count int32
	computeFn := func(key1, key2 int) int {
		atomic.AddInt32(&count, 1)
		return key1 + key2
	}
	memoizedFn := mustMemoized(Memoize2(computeFn, Opts().WithTTL(1*time.Minute)))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			memoizedFn(20, 22)
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestMemoize3WithTTL_ConcurrentAccess(t *testing.T) {
	var count int32
	computeFn := func(key1, key2, key3 int) int {
		atomic.AddInt32(&count, 1)
		return key1 + key2 + key3
	}
	memoizedFn := mustMemoized(Memoize3(computeFn, Opts().WithTTL(1*time.Minute)))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			memoizedFn(10, 20, 12)
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestMemoize4WithTTL_ConcurrentAccess(t *testing.T) {
	var count int32
	computeFn := func(key1, key2, key3, key4 int) int {
		atomic.AddInt32(&count, 1)
		return key1 + key2 + key3 + key4
	}
	memoizedFn := mustMemoized(Memoize4(computeFn, Opts().WithTTL(1*time.Minute)))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			memoizedFn(10, 10, 10, 12)
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestMemoize5WithTTL_ConcurrentAccess(t *testing.T) {
	var count int32
	computeFn := func(key1, key2, key3, key4, key5 int) int {
		atomic.AddInt32(&count, 1)
		return key1 + key2 + key3 + key4 + key5
	}
	memoizedFn := mustMemoized(Memoize5(computeFn, Opts().WithTTL(1*time.Minute)))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			memoizedFn(1, 2, 3, 4, 5)
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestMemoize6WithTTL_ConcurrentAccess(t *testing.T) {
	var count int32
	computeFn := func(key1, key2, key3, key4, key5, key6 int) int {
		atomic.AddInt32(&count, 1)
		return key1 + key2 + key3 + key4 + key5 + key6
	}
	memoizedFn := mustMemoized(Memoize6(computeFn, Opts().WithTTL(1*time.Minute)))
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			memoizedFn(1, 2, 3, 4, 5, 6)
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}
