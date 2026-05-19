package chain_test

import (
	"context"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/chain"
	"github.com/agkloop/go_memoize/stores/memory"
)

func stored(v string, ttl time.Duration) memoize.Stored[string] {
	now := time.Now()
	return memoize.Stored[string]{
		Value:      v,
		CreatedAt:  now,
		FreshUntil: now.Add(ttl),
	}
}

func TestChainGet_L1Hit(t *testing.T) {
	ctx := context.Background()
	l1 := memory.New[string, string](16)
	l2 := memory.New[string, string](16)
	c := chain.New[string, string](l1, l2)

	_ = l1.Set(ctx, "k", stored("from-l1", time.Hour))
	got, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || got.Value != "from-l1" {
		t.Fatalf("expected L1 hit: got=%v ok=%v err=%v", got, ok, err)
	}
}

func TestChainGet_L2HitBackfillsL1(t *testing.T) {
	ctx := context.Background()
	l1 := memory.New[string, string](16)
	l2 := memory.New[string, string](16)
	c := chain.New[string, string](l1, l2)

	_ = l2.Set(ctx, "k", stored("from-l2", time.Hour))
	got, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || got.Value != "from-l2" {
		t.Fatalf("expected L2 hit: got=%v ok=%v err=%v", got, ok, err)
	}
	// L1 should now have it
	l1got, l1ok, _ := l1.Get(ctx, "k")
	if !l1ok || l1got.Value != "from-l2" {
		t.Fatal("expected L1 to be backfilled from L2")
	}
}

func TestChainGet_Miss(t *testing.T) {
	ctx := context.Background()
	c := chain.New[string, string](memory.New[string, string](16), memory.New[string, string](16))
	_, ok, err := c.Get(ctx, "missing")
	if err != nil || ok {
		t.Fatalf("expected miss: ok=%v err=%v", ok, err)
	}
}

func TestChainSet_WritesAllTiers(t *testing.T) {
	ctx := context.Background()
	l1 := memory.New[string, string](16)
	l2 := memory.New[string, string](16)
	c := chain.New[string, string](l1, l2)

	_ = c.Set(ctx, "k", stored("v", time.Hour))
	if _, ok, _ := l1.Get(ctx, "k"); !ok {
		t.Fatal("L1 should have the key after Set")
	}
	if _, ok, _ := l2.Get(ctx, "k"); !ok {
		t.Fatal("L2 should have the key after Set")
	}
}

func TestChainDelete_AllTiers(t *testing.T) {
	ctx := context.Background()
	l1 := memory.New[string, string](16)
	l2 := memory.New[string, string](16)
	c := chain.New[string, string](l1, l2)

	_ = c.Set(ctx, "k", stored("v", time.Hour))
	_ = c.Delete(ctx, "k")

	if _, ok, _ := l1.Get(ctx, "k"); ok {
		t.Fatal("L1 should not have key after Delete")
	}
	if _, ok, _ := l2.Get(ctx, "k"); ok {
		t.Fatal("L2 should not have key after Delete")
	}
}

func TestChainClear_AllTiers(t *testing.T) {
	ctx := context.Background()
	l1 := memory.New[string, string](16)
	l2 := memory.New[string, string](16)
	c := chain.New[string, string](l1, l2)

	_ = c.Set(ctx, "k", stored("v", time.Hour))
	_ = c.Clear(ctx)

	if _, ok, _ := l1.Get(ctx, "k"); ok {
		t.Fatal("L1 should be empty after Clear")
	}
	if _, ok, _ := l2.Get(ctx, "k"); ok {
		t.Fatal("L2 should be empty after Clear")
	}
}
