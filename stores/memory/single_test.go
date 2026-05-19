package memory

import (
	"context"
	"strconv"
	"sync"
	"testing"

	memoize "github.com/agkloop/go_memoize"
)

func TestSingleStoreSetGetPeekDeleteClear(t *testing.T) {
	ctx := context.Background()
	s := NewSingle[string, string]()
	entry := memoize.Stored[string]{Value: "value", NoExpire: true}

	if got := s.Len(); got != 0 {
		t.Fatalf("expected empty store len 0, got %d", got)
	}
	if _, ok, err := s.Get(ctx, "key"); err != nil || ok {
		t.Fatalf("missing key returned ok=%v err=%v", ok, err)
	}
	if err := s.Set(ctx, "key", entry); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	got, ok, err := s.Get(ctx, "key")
	if err != nil || !ok || got.Value != "value" {
		t.Fatalf("get returned value=%q ok=%v err=%v", got.Value, ok, err)
	}
	peeked, ok, err := s.Peek(ctx, "key")
	if err != nil || !ok || peeked.Value != "value" {
		t.Fatalf("peek returned value=%q ok=%v err=%v", peeked.Value, ok, err)
	}
	if got := s.Len(); got != 1 {
		t.Fatalf("expected len 1, got %d", got)
	}

	if err := s.Delete(ctx, "other"); err != nil {
		t.Fatalf("delete other failed: %v", err)
	}
	if _, ok, _ := s.Get(ctx, "key"); !ok {
		t.Fatal("delete of different key should not remove stored value")
	}
	if err := s.Delete(ctx, "key"); err != nil {
		t.Fatalf("delete key failed: %v", err)
	}
	if _, ok, _ := s.Get(ctx, "key"); ok {
		t.Fatal("deleted key should be absent")
	}

	_ = s.Set(ctx, "key", entry)
	if err := s.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if got := s.Len(); got != 0 {
		t.Fatalf("expected len 0 after clear, got %d", got)
	}
}

func TestSingleStoreReplaceRemovesPreviousKey(t *testing.T) {
	ctx := context.Background()
	s := NewSingle[string, string]()

	_ = s.Set(ctx, "a", memoize.Stored[string]{Value: "A", NoExpire: true})
	_ = s.Set(ctx, "b", memoize.Stored[string]{Value: "B", NoExpire: true})

	if _, ok, _ := s.Get(ctx, "a"); ok {
		t.Fatal("single store should only retain the latest key")
	}
	got, ok, err := s.Get(ctx, "b")
	if err != nil || !ok || got.Value != "B" {
		t.Fatalf("expected latest key b, got value=%q ok=%v err=%v", got.Value, ok, err)
	}
}

func TestSingleStoreDeleteByTag(t *testing.T) {
	ctx := context.Background()
	s := NewSingle[string, string]()

	_ = s.Set(ctx, "key", memoize.Stored[string]{Value: "value", NoExpire: true, Tags: []string{"group"}})
	if err := s.DeleteByTag(ctx, "other"); err != nil {
		t.Fatalf("delete by other tag failed: %v", err)
	}
	if _, ok, _ := s.Get(ctx, "key"); !ok {
		t.Fatal("different tag should not delete entry")
	}
	if err := s.DeleteByTag(ctx, "group"); err != nil {
		t.Fatalf("delete by tag failed: %v", err)
	}
	if _, ok, _ := s.Get(ctx, "key"); ok {
		t.Fatal("matching tag should delete entry")
	}
}

func TestSingleStoreConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	s := NewSingle[string, string]()
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := strconv.Itoa(i % 4)
			_ = s.Set(ctx, key, memoize.Stored[string]{Value: key, NoExpire: true})
			_, _, _ = s.Get(ctx, key)
			_, _, _ = s.Peek(ctx, key)
			if i%17 == 0 {
				_ = s.Delete(ctx, key)
			}
		}(i)
	}
	wg.Wait()
}
