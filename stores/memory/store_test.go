package memory

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
)

func TestStoreGetSetDeleteClear(t *testing.T) {
	ctx := context.Background()
	store := New[string, string](16)
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	entry := memoize.Stored[string]{Value: "value", CreatedAt: now, FreshUntil: now.Add(time.Minute)}

	if _, ok, err := store.Get(ctx, "missing"); err != nil || ok {
		t.Fatalf("missing key returned ok=%v err=%v", ok, err)
	}
	if err := store.Set(ctx, "key", entry); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	got, ok, err := store.Get(ctx, "key")
	if err != nil || !ok || got.Value != "value" {
		t.Fatalf("get returned value=%q ok=%v err=%v", got.Value, ok, err)
	}
	if err := store.Delete(ctx, "key"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, ok, err := store.Get(ctx, "key"); err != nil || ok {
		t.Fatalf("deleted key returned ok=%v err=%v", ok, err)
	}
	if err := store.Set(ctx, "key", entry); err != nil {
		t.Fatalf("set before clear failed: %v", err)
	}
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if _, ok, err := store.Get(ctx, "key"); err != nil || ok {
		t.Fatalf("cleared key returned ok=%v err=%v", ok, err)
	}
}

func TestLRUEvictsLeastRecentlyUsed(t *testing.T) {
	ctx := context.Background()
	s := New[string, string](2)

	stored := func(v string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true}
	}

	_ = s.Set(ctx, "a", stored("A"))
	_ = s.Set(ctx, "b", stored("B"))
	// Access "a" so "b" becomes LRU
	_, _, _ = s.Get(ctx, "a")
	// Insert "c" — should evict "b"
	_ = s.Set(ctx, "c", stored("C"))

	if _, ok, _ := s.Get(ctx, "b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
	if _, ok, _ := s.Get(ctx, "a"); !ok {
		t.Fatal("expected 'a' to survive")
	}
	if _, ok, _ := s.Get(ctx, "c"); !ok {
		t.Fatal("expected 'c' to be present")
	}
}

func TestPeekDoesNotAffectRecency(t *testing.T) {
	ctx := context.Background()
	s := New[string, string](2)
	stored := func(v string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true}
	}

	_ = s.Set(ctx, "a", stored("A"))
	_ = s.Set(ctx, "b", stored("B"))
	got, ok, err := s.Peek(ctx, "a")
	if err != nil || !ok || got.Value != "A" {
		t.Fatalf("peek returned value=%q ok=%v err=%v", got.Value, ok, err)
	}
	_ = s.Set(ctx, "c", stored("C"))

	if _, ok, _ := s.Get(ctx, "a"); ok {
		t.Fatal("peek should not refresh recency; expected a to be evicted")
	}
	if _, ok, _ := s.Get(ctx, "b"); !ok {
		t.Fatal("expected b to survive")
	}
	if _, ok, _ := s.Get(ctx, "c"); !ok {
		t.Fatal("expected c to be present")
	}
}

func TestGetRecencySampling(t *testing.T) {
	ctx := context.Background()
	stored := func(v string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true}
	}

	skipped := New[string, string](2, WithGetRecencySample[string, string](3))
	_ = skipped.Set(ctx, "a", stored("A"))
	_ = skipped.Set(ctx, "b", stored("B"))
	_, _, _ = skipped.Get(ctx, "a")
	_ = skipped.Set(ctx, "c", stored("C"))
	if _, ok, _ := skipped.Get(ctx, "a"); ok {
		t.Fatal("first sampled get should not refresh recency; expected a to be evicted")
	}

	refreshed := New[string, string](2, WithGetRecencySample[string, string](3))
	_ = refreshed.Set(ctx, "a", stored("A"))
	_ = refreshed.Set(ctx, "b", stored("B"))
	for i := 0; i < 3; i++ {
		_, _, _ = refreshed.Get(ctx, "a")
	}
	_ = refreshed.Set(ctx, "c", stored("C"))
	if _, ok, _ := refreshed.Get(ctx, "a"); !ok {
		t.Fatal("third sampled get should refresh recency; expected a to survive")
	}
	if _, ok, _ := refreshed.Get(ctx, "b"); ok {
		t.Fatal("expected b to be evicted after sampled refresh of a")
	}
}

func TestCapacityFirstConstructorEvictsLeastRecentlyUsed(t *testing.T) {
	ctx := context.Background()
	s := New[string, string](2)
	stored := func(v string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true}
	}

	if err := s.Set(ctx, "a", stored("A")); err != nil {
		t.Fatalf("set a failed: %v", err)
	}
	if err := s.Set(ctx, "b", stored("B")); err != nil {
		t.Fatalf("set b failed: %v", err)
	}
	if got := s.Len(); got != 2 {
		t.Fatalf("expected len 2 before eviction, got %d", got)
	}
	if _, ok, _ := s.Get(ctx, "a"); !ok {
		t.Fatal("expected a before eviction")
	}
	if err := s.Set(ctx, "c", stored("C")); err != nil {
		t.Fatalf("set c failed: %v", err)
	}

	if got := s.Len(); got != 2 {
		t.Fatalf("expected len 2 after eviction, got %d", got)
	}
	if _, ok, _ := s.Get(ctx, "b"); ok {
		t.Fatal("expected b to be evicted")
	}
	if _, ok, _ := s.Get(ctx, "a"); !ok {
		t.Fatal("expected a to survive as most recently used")
	}
	if _, ok, _ := s.Get(ctx, "c"); !ok {
		t.Fatal("expected c to be present")
	}
}

func TestEntriesSurviveUpdateDeleteAndCompaction(t *testing.T) {
	ctx := context.Background()
	s := New[string, string](4)
	stored := func(v string, tags ...string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true, Tags: tags}
	}

	for _, key := range []string{"k3", "k7", "k12", "k16"} {
		if err := s.Set(ctx, key, stored(key, "group")); err != nil {
			t.Fatalf("set %s failed: %v", key, err)
		}
	}
	if err := s.Set(ctx, "k7", stored("B2", "group")); err != nil {
		t.Fatalf("update k7 failed: %v", err)
	}
	if got := s.Len(); got != 4 {
		t.Fatalf("update should not change len, got %d", got)
	}
	if err := s.Delete(ctx, "k7"); err != nil {
		t.Fatalf("delete k7 failed: %v", err)
	}
	if err := s.Set(ctx, "k23", stored("e", "other")); err != nil {
		t.Fatalf("set k23 failed: %v", err)
	}

	if _, ok, _ := s.Get(ctx, "k7"); ok {
		t.Fatal("deleted key k7 should be absent")
	}
	for _, key := range []string{"k3", "k12", "k16", "k23"} {
		if _, ok, _ := s.Get(ctx, key); !ok {
			t.Fatalf("expected key %s to survive collision-chain compaction", key)
		}
	}
	if err := s.DeleteByTag(ctx, "group"); err != nil {
		t.Fatalf("delete by tag failed: %v", err)
	}
	for _, key := range []string{"k3", "k12", "k16"} {
		if _, ok, _ := s.Get(ctx, key); ok {
			t.Fatalf("expected tagged key %s to be removed", key)
		}
	}
	if _, ok, _ := s.Get(ctx, "k23"); !ok {
		t.Fatal("expected differently-tagged key k23 to survive")
	}
}

func TestNoEvictionWhenUnbounded(t *testing.T) {
	ctx := context.Background()
	s := New[string, string](1000)

	stored := func(v string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true}
	}
	for i := 0; i < 1000; i++ {
		_ = s.Set(ctx, strconv.Itoa(i), stored(strconv.Itoa(i)))
	}
	if _, ok, _ := s.Get(ctx, "0"); !ok {
		t.Fatal("entry 0 should still exist in unbounded store")
	}
}

func TestByteCapacityEvicts(t *testing.T) {
	ctx := context.Background()
	// unsafe.Sizeof(memoize.Stored[string]{}) = 136 bytes on 64-bit (string header 16 +
	// time.Time 24 + bool + padding + additional fields).
	// key "key1" = 4 bytes → total per entry = 140 bytes.
	// Limit = 200 bytes → fits 1 entry cleanly; 2nd entry (total 280) triggers eviction.
	s := New[string, string](16, WithMaxBytes[string, string](200))

	stored := func(v string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true}
	}
	_ = s.Set(ctx, "key1", stored("0123456789")) // 140 bytes
	_ = s.Set(ctx, "key2", stored("abcdefghij")) // 140 bytes → total 280, evicts key1
	_ = s.Set(ctx, "key3", stored("XXXXXXXXXX")) // 140 bytes → evicts key2

	if _, ok, _ := s.Get(ctx, "key1"); ok {
		t.Fatal("key1 should have been evicted")
	}
	if _, ok, _ := s.Get(ctx, "key2"); ok {
		t.Fatal("key2 should have been evicted")
	}
	if _, ok, _ := s.Get(ctx, "key3"); !ok {
		t.Fatal("key3 should be present")
	}
}

func TestUsedBytesAccountingAfterDeleteAndClear(t *testing.T) {
	ctx := context.Background()
	s := New[string, string](16)

	stored := func(v string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true}
	}

	_ = s.Set(ctx, "k1", stored("v1"))
	_ = s.Set(ctx, "k2", stored("v2"))
	before := s.UsedBytes()
	if before == 0 {
		t.Fatal("usedBytes should be nonzero after two sets")
	}

	_ = s.Delete(ctx, "k1")
	after := s.UsedBytes()
	if after >= before {
		t.Fatalf("usedBytes should decrease after delete: before=%d after=%d", before, after)
	}

	_ = s.Clear(ctx)
	if s.UsedBytes() != 0 {
		t.Fatalf("usedBytes should be 0 after clear, got %d", s.UsedBytes())
	}
}

func TestDeleteByTag(t *testing.T) {
	ctx := context.Background()
	s := New[string, string](16)

	stored := func(v string, tags ...string) memoize.Stored[string] {
		return memoize.Stored[string]{Value: v, NoExpire: true, Tags: tags}
	}

	_ = s.Set(ctx, "a", stored("A", "group1"))
	_ = s.Set(ctx, "b", stored("B", "group1", "group2"))
	_ = s.Set(ctx, "c", stored("C", "group2"))
	_ = s.Set(ctx, "d", stored("D")) // no tags

	_ = s.DeleteByTag(ctx, "group1")

	if _, ok, _ := s.Get(ctx, "a"); ok {
		t.Fatal("'a' should have been invalidated by tag group1")
	}
	if _, ok, _ := s.Get(ctx, "b"); ok {
		t.Fatal("'b' should have been invalidated by tag group1")
	}
	if _, ok, _ := s.Get(ctx, "c"); !ok {
		t.Fatal("'c' should survive (only has group2)")
	}
	if _, ok, _ := s.Get(ctx, "d"); !ok {
		t.Fatal("'d' should survive (no tags)")
	}
}

func TestConcurrentGetSet(t *testing.T) {
	s := New[string, string](10)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := strconv.Itoa(i % 15)
			_ = s.Set(context.Background(), key, memoize.Stored[string]{Value: "v", NoExpire: true})
			_, _, _ = s.Get(context.Background(), key)
		}(i)
	}
	wg.Wait()
}
