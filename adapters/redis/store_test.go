package redisstore

import (
	"errors"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/serializers"
)

func TestNewRequiresClientAndSerializer(t *testing.T) {
	if _, err := New[string, string](); !errors.Is(err, ErrMissingClient) {
		t.Fatalf("expected ErrMissingClient, got %v", err)
	}
	if _, err := New[string, string](WithClient[string, string](nil)); !errors.Is(err, ErrMissingClient) {
		t.Fatalf("expected ErrMissingClient for nil client, got %v", err)
	}
}

func TestStoreKeyUsesPrefixAndDefaultEncoder(t *testing.T) {
	store := &Store[uint64, string]{prefix: "profiles:"}
	if got := store.key(123456789); got != "profiles:123456789" {
		t.Fatalf("key = %q, want profiles:123456789", got)
	}
}

func TestStoreKeyUsesCustomEncoder(t *testing.T) {
	store := &Store[int, string]{prefix: "users", keyEncoder: func(key int) string { return "id-" + defaultKeyEncoder(key) }}
	if got := store.key(42); got != "users:id-42" {
		t.Fatalf("key = %q, want users:id-42", got)
	}
}

func TestStorageTTLUsesStaleUntil(t *testing.T) {
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	entry := memoize.Stored[string]{FreshUntil: now.Add(time.Minute), StaleUntil: now.Add(3 * time.Minute)}
	if ttl := storageTTL(entry, now); ttl != 3*time.Minute {
		t.Fatalf("expected 3m ttl, got %s", ttl)
	}
	_ = serializers.JSON[string]{}
}
