package memory

import (
	"context"
	"strconv"
	"sync"
	"testing"

	memoize "github.com/agkloop/go_memoize"
)

func TestShardedStoreBasicOps(t *testing.T) {
	ctx := context.Background()
	s := NewSharded[string, string](16, WithShards[string, string](16))

	stored := memoize.Stored[string]{Value: "hello", NoExpire: true}
	_ = s.Set(ctx, "key", stored)
	got, ok, err := s.Get(ctx, "key")
	if err != nil || !ok || got.Value != "hello" {
		t.Fatalf("got=%v ok=%v err=%v", got, ok, err)
	}
	peeked, ok, err := s.Peek(ctx, "key")
	if err != nil || !ok || peeked.Value != "hello" {
		t.Fatalf("peeked=%v ok=%v err=%v", peeked, ok, err)
	}

	_ = s.Delete(ctx, "key")
	if _, ok, _ := s.Get(ctx, "key"); ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestShardedStoreConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	s := NewSharded[string, int](1024, WithShards[string, int](8))

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := strconv.Itoa(i)
			_ = s.Set(ctx, key, memoize.Stored[int]{Value: i, NoExpire: true})
			_, _, _ = s.Get(ctx, key)
		}(i)
	}
	wg.Wait()
}

func TestShardedStoreClear(t *testing.T) {
	ctx := context.Background()
	s := NewSharded[string, string](32, WithShards[string, string](4))
	for i := 0; i < 20; i++ {
		_ = s.Set(ctx, strconv.Itoa(i), memoize.Stored[string]{Value: "v", NoExpire: true})
	}
	_ = s.Clear(ctx)
	if s.Len() != 0 {
		t.Fatalf("expected 0 after clear, got %d", s.Len())
	}
}

func TestShardedStorePanicsOnBadN(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-power-of-two n")
		}
	}()
	NewSharded[string, string](16, WithShards[string, string](3)) // not a power of two
}

func TestShardedStorePanicsOnUnsupportedKeyType(t *testing.T) {
	type unsupported struct{ value string }

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unsupported sharded key type")
		}
	}()

	s := NewSharded[unsupported, string](16, WithShards[unsupported, string](4))
	_ = s.Set(context.Background(), unsupported{value: "key"}, memoize.Stored[string]{Value: "value", NoExpire: true})
}
