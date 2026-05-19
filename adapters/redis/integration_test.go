package redisstore

import (
	"context"
	"os"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/serializers"
	"github.com/redis/go-redis/v9"
)

func TestRedisStoreIntegration(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR is not set")
	}
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	store, err := New[string, string](WithClient[string, string](client), WithPrefix[string, string]("go_memoize_test"), WithSerializer[string, string](serializers.JSON[string]{}))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	entry := memoize.Stored[string]{Value: "redis", CreatedAt: time.Now(), FreshUntil: time.Now().Add(time.Minute)}
	if err := store.Set(ctx, "key", entry); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	got, ok, err := store.Get(ctx, "key")
	if err != nil || !ok || got.Value != "redis" {
		t.Fatalf("get value=%q ok=%v err=%v", got.Value, ok, err)
	}
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
}
