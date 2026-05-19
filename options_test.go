package memoize_test

import (
	"errors"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/memory"
)

func TestNewRequiresExpirationPolicy(t *testing.T) {
	_, err := memoize.New[string, string]()
	if !errors.Is(err, memoize.ErrMissingExpirationPolicy) {
		t.Fatalf("expected ErrMissingExpirationPolicy, got %v", err)
	}
}

func TestNewRejectsNonPositiveTTL(t *testing.T) {
	_, err := memoize.New[string, string](memoize.Opts().WithTTL(0))
	if !errors.Is(err, memoize.ErrInvalidTTL) {
		t.Fatalf("expected ErrInvalidTTL, got %v", err)
	}
}

func TestNewRejectsStaleTTLWithoutTTL(t *testing.T) {
	_, err := memoize.New[string, string](memoize.Opts().NoExpiration().WithStaleTTL(time.Second))
	if !errors.Is(err, memoize.ErrInvalidStaleTTL) {
		t.Fatalf("expected ErrInvalidStaleTTL, got %v", err)
	}
}

func TestNewRejectsWrongStoreType(t *testing.T) {
	_, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[int, string](16)).WithTTL(time.Minute))
	if !errors.Is(err, memoize.ErrInvalidStore) {
		t.Fatalf("expected ErrInvalidStore, got %v", err)
	}
}
