package local_test

import (
	"context"
	"os"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/local"
)

func stored(v string, ttl time.Duration) memoize.Stored[string] {
	now := time.Now()
	s := memoize.Stored[string]{Value: v, CreatedAt: now}
	if ttl > 0 {
		s.FreshUntil = now.Add(ttl)
	} else {
		s.NoExpire = true
	}
	return s
}

func TestLocalFileStore_SetGet(t *testing.T) {
	dir := t.TempDir()
	s := local.New[string](dir)
	ctx := context.Background()

	_ = s.Set(ctx, "hello", stored("world", time.Hour))
	got, ok, err := s.Get(ctx, "hello")
	if err != nil || !ok || got.Value != "world" {
		t.Fatalf("got=%v ok=%v err=%v", got, ok, err)
	}
}

func TestLocalFileStore_Miss(t *testing.T) {
	dir := t.TempDir()
	s := local.New[string](dir)
	ctx := context.Background()

	_, ok, err := s.Get(ctx, "nonexistent")
	if err != nil || ok {
		t.Fatalf("expected miss: ok=%v err=%v", ok, err)
	}
}

func TestLocalFileStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s := local.New[string](dir)
	ctx := context.Background()

	_ = s.Set(ctx, "k", stored("v", time.Hour))
	_ = s.Delete(ctx, "k")
	_, ok, _ := s.Get(ctx, "k")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestLocalFileStore_Clear(t *testing.T) {
	dir := t.TempDir()
	s := local.New[string](dir)
	ctx := context.Background()

	_ = s.Set(ctx, "a", stored("1", time.Hour))
	_ = s.Set(ctx, "b", stored("2", time.Hour))
	_ = s.Clear(ctx)

	_, okA, _ := s.Get(ctx, "a")
	_, okB, _ := s.Get(ctx, "b")
	if okA || okB {
		t.Fatal("expected store to be empty after Clear")
	}
	// Verify no files remain
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("expected 0 files after Clear, got %d", len(entries))
	}
}

func TestLocalFileStoreReturnsExpiredEntriesForCacheEngine(t *testing.T) {
	dir := t.TempDir()
	s := local.New[string](dir)
	ctx := context.Background()

	past := time.Now().Add(-time.Hour)
	expired := memoize.Stored[string]{Value: "old", CreatedAt: past, FreshUntil: past}
	if err := s.Set(ctx, "expired", expired); err != nil {
		t.Fatalf("set expired entry: %v", err)
	}

	got, ok, err := s.Get(ctx, "expired")
	if err != nil {
		t.Fatalf("get expired entry: %v", err)
	}
	if !ok {
		t.Fatal("expected expired entry to be returned for cache engine")
	}
	if got.Value != expired.Value || !got.FreshUntil.Equal(expired.FreshUntil) {
		t.Fatalf("got=%+v want=%+v", got, expired)
	}
}

func TestLocalFileStoreReturnsStaleEntriesForCacheEngine(t *testing.T) {
	dir := t.TempDir()
	s := local.New[string](dir)
	ctx := context.Background()
	now := time.Now()
	entry := memoize.Stored[string]{
		Value:      "stale",
		CreatedAt:  now.Add(-time.Hour),
		FreshUntil: now.Add(-time.Minute),
		StaleUntil: now.Add(time.Hour),
	}

	if err := s.Set(ctx, "stale", entry); err != nil {
		t.Fatalf("set stale entry: %v", err)
	}
	got, ok, err := s.Get(ctx, "stale")
	if err != nil {
		t.Fatalf("get stale entry: %v", err)
	}
	if !ok {
		t.Fatal("expected stale entry to be returned for cache engine")
	}
	if got.Value != entry.Value || !got.FreshUntil.Equal(entry.FreshUntil) || !got.StaleUntil.Equal(entry.StaleUntil) {
		t.Fatalf("got=%+v want=%+v", got, entry)
	}
}

func TestLocalFileStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Write with one store instance
	s1 := local.New[string](dir)
	_ = s1.Set(ctx, "persistent", stored("stays", time.Hour))

	// Read with a new instance pointing at the same dir
	s2 := local.New[string](dir)
	got, ok, err := s2.Get(ctx, "persistent")
	if err != nil || !ok || got.Value != "stays" {
		t.Fatalf("expected persisted value: got=%v ok=%v err=%v", got, ok, err)
	}
}
