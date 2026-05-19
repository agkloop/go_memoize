// v2/background/background_test.go
package background_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/background"
)

func TestKeep_InitialLoad(t *testing.T) {
	calls := 0
	fn := func(ctx context.Context) (int, error) {
		calls++
		return 42, nil
	}
	val, err := background.Keep(context.Background(), fn, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := val.Get(); got != 42 {
		t.Fatalf("want 42, got %d", got)
	}
	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}
}

func TestValueGetZeroValueReturnsZero(t *testing.T) {
	var val background.Value[int]
	if got := val.Get(); got != 0 {
		t.Fatalf("want zero value, got %d", got)
	}
}

func TestKeep_NonPositiveIntervalDoesNotPanic(t *testing.T) {
	val, err := background.Keep(context.Background(), func(context.Context) (int, error) {
		return 42, nil
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := val.Get(); got != 42 {
		t.Fatalf("want initial value 42, got %d", got)
	}
	time.Sleep(10 * time.Millisecond)
}

func TestMirror_NonPositiveIntervalDoesNotPanic(t *testing.T) {
	store := &fakeStore[int]{entry: memoize.Stored[int]{Value: 42, NoExpire: true}, ok: true}
	val, err := background.Mirror[int](context.Background(), "k", store, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := val.Get(); got != 42 {
		t.Fatalf("want initial value 42, got %d", got)
	}
	time.Sleep(10 * time.Millisecond)
}

func TestKeep_InitialLoadError(t *testing.T) {
	fn := func(ctx context.Context) (int, error) {
		return 0, errors.New("boom")
	}
	_, err := background.Keep(context.Background(), fn, time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestKeep_Refreshes(t *testing.T) {
	var counter atomic.Int32
	fn := func(ctx context.Context) (int32, error) {
		return counter.Add(1), nil
	}
	val, err := background.Keep(context.Background(), fn, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(70 * time.Millisecond)
	if got := val.Get(); got < 3 {
		t.Fatalf("want >= 3 refreshes, got %d", got)
	}
}

func TestKeep_KeepsStaleOnError(t *testing.T) {
	var fail atomic.Bool
	fn := func(ctx context.Context) (int, error) {
		if fail.Load() {
			return 0, errors.New("transient")
		}
		return 99, nil
	}
	val, err := background.Keep(context.Background(), fn, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	fail.Store(true)
	time.Sleep(50 * time.Millisecond)
	if got := val.Get(); got != 99 {
		t.Fatalf("want stale 99, got %d", got)
	}
}

func TestKeep_OnError_Called(t *testing.T) {
	var fail atomic.Bool
	var errCount atomic.Int32
	fn := func(ctx context.Context) (int, error) {
		if fail.Load() {
			return 0, errors.New("transient")
		}
		return 1, nil
	}
	_, err := background.Keep(context.Background(), fn, 20*time.Millisecond,
		background.OnError[int](func(e error) { errCount.Add(1) }),
	)
	if err != nil {
		t.Fatal(err)
	}
	fail.Store(true)
	time.Sleep(60 * time.Millisecond)
	if errCount.Load() == 0 {
		t.Fatal("OnError was never called")
	}
}

func TestKeep_StopsOnCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	fn := func(ctx context.Context) (int, error) {
		calls.Add(1)
		return 1, nil
	}
	_, err := background.Keep(ctx, fn, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	time.Sleep(50 * time.Millisecond)
	snapshot := calls.Load()
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != snapshot {
		t.Fatal("goroutine did not stop after ctx cancel")
	}
}

// fakeStore is an in-memory Store[V] for tests.
type fakeStore[V any] struct {
	mu    sync.Mutex
	entry memoize.Stored[V]
	ok    bool
	err   error
}

func (f *fakeStore[V]) Get(_ context.Context, _ string) (memoize.Stored[V], bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.entry, f.ok, f.err
}
func (f *fakeStore[V]) Set(_ context.Context, _ string, v memoize.Stored[V]) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entry = v
	f.ok = true
	return nil
}
func (f *fakeStore[V]) Delete(_ context.Context, _ string) error { return nil }
func (f *fakeStore[V]) Clear(_ context.Context) error            { return nil }

func TestMirror_InitialRead(t *testing.T) {
	store := &fakeStore[string]{
		entry: memoize.Stored[string]{Value: "hello", NoExpire: true},
		ok:    true,
	}
	val, err := background.Mirror[string](context.Background(), "k", store, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got := val.Get(); got != "hello" {
		t.Fatalf("want hello, got %s", got)
	}
}

func TestMirror_InitialMissErrors(t *testing.T) {
	store := &fakeStore[string]{ok: false}
	_, err := background.Mirror[string](context.Background(), "k", store, time.Hour)
	if err == nil {
		t.Fatal("expected error on miss")
	}
}

func TestMirror_Refreshes(t *testing.T) {
	var mu sync.Mutex
	n := 0
	store := &fakeStore[int]{entry: memoize.Stored[int]{Value: 1, NoExpire: true}, ok: true}

	val, err := background.Mirror[int](context.Background(), "k", store, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		ticker := time.NewTicker(15 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			n++
			v := n
			mu.Unlock()
			_ = store.Set(context.Background(), "k", memoize.Stored[int]{Value: v, NoExpire: true})
		}
	}()

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	want := n
	mu.Unlock()
	if got := val.Get(); got < 2 {
		t.Fatalf("want refreshed value >= 2, got %d (store at %d)", got, want)
	}
}

func TestMirror_KeepsStaleOnStoreMiss(t *testing.T) {
	store := &fakeStore[string]{
		entry: memoize.Stored[string]{Value: "stale", NoExpire: true},
		ok:    true,
	}
	val, err := background.Mirror[string](context.Background(), "k", store, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	// make store return miss
	store.mu.Lock()
	store.ok = false
	store.mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	if got := val.Get(); got != "stale" {
		t.Fatalf("want stale, got %s", got)
	}
}

func TestMirror_StopsOnCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &fakeStore[int]{entry: memoize.Stored[int]{Value: 1, NoExpire: true}, ok: true}
	var calls atomic.Int32
	_, err := background.Mirror[int](ctx, "k", store, 10*time.Millisecond,
		background.OnRefresh[int](func(int) { calls.Add(1) }),
	)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	time.Sleep(50 * time.Millisecond)
	snapshot := calls.Load()
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != snapshot {
		t.Fatal("goroutine did not stop")
	}
}

func TestKeep_WriteThrough(t *testing.T) {
	store := &fakeStore[int]{}
	calls := 0
	fn := func(ctx context.Context) (int, error) {
		calls++
		return calls * 10, nil
	}

	_, err := background.Keep(context.Background(), fn, 20*time.Millisecond,
		background.WriteThrough[int]("mykey", store),
	)
	if err != nil {
		t.Fatal(err)
	}

	// initial write happened synchronously
	store.mu.Lock()
	got := store.entry.Value
	store.mu.Unlock()
	if got != 10 {
		t.Fatalf("want 10 after initial write, got %d", got)
	}

	// wait for a refresh cycle
	time.Sleep(40 * time.Millisecond)
	store.mu.Lock()
	got = store.entry.Value
	store.mu.Unlock()
	if got < 20 {
		t.Fatalf("want >= 20 after refresh, got %d", got)
	}
}
