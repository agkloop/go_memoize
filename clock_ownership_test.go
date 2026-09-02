package memoize

import (
	"context"
	"testing"
	"time"
)

type clockTestStore struct{}

func (*clockTestStore) Get(context.Context, string) (Stored[string], bool, error) {
	return Stored[string]{}, false, nil
}

func (*clockTestStore) Set(context.Context, string, Stored[string]) error { return nil }
func (*clockTestStore) Delete(context.Context, string) error              { return nil }
func (*clockTestStore) Clear(context.Context) error                       { return nil }

func TestCacheStopDoesNotStopInjectedClock(t *testing.T) {
	clock := NewTickerClock(time.Hour)
	t.Cleanup(clock.Stop)

	cache, err := New[string, string](
		Opts().
			WithStore(&clockTestStore{}).
			WithTTL(time.Minute).
			WithClock(clock),
	)
	if err != nil {
		t.Fatalf("new cache failed: %v", err)
	}
	cache.Stop()

	select {
	case <-clock.stop:
		t.Fatal("cache stopped a caller-owned clock")
	default:
	}

	if err := cache.Set(t.Context(), "key", "value"); err != nil {
		t.Fatalf("cache should remain usable with the injected clock: %v", err)
	}
}

func TestCacheStopStopsOwnedClock(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "default",
			opts: Opts().WithStore(&clockTestStore{}).NoExpiration(),
		},
		{
			name: "configured ticker",
			opts: Opts().WithStore(&clockTestStore{}).NoExpiration().WithTickerClock(time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := New[string, string](tt.opts)
			if err != nil {
				t.Fatalf("new cache failed: %v", err)
			}
			clock, ok := cache.clock.(*TickerClock)
			if !ok || !cache.clockOwned {
				t.Fatalf("cache clock = %T owned=%v, want owned TickerClock", cache.clock, cache.clockOwned)
			}

			cache.Stop()
			select {
			case <-clock.stop:
			default:
				t.Fatal("cache did not stop its owned clock")
			}
		})
	}
}

func TestClockOptionOrder(t *testing.T) {
	shared := NewTickerClock(time.Hour)
	t.Cleanup(shared.Stop)

	borrowed, err := New[string, string](
		Opts().
			WithStore(&clockTestStore{}).
			NoExpiration().
			WithTickerClock(time.Hour).
			WithClock(shared),
	)
	if err != nil {
		t.Fatalf("new cache with final injected clock failed: %v", err)
	}
	if borrowed.clock != shared || borrowed.clockOwned {
		t.Fatalf("final WithClock selected clock=%T owned=%v, want borrowed shared clock", borrowed.clock, borrowed.clockOwned)
	}
	borrowed.Stop()

	owned, err := New[string, string](
		Opts().
			WithStore(&clockTestStore{}).
			NoExpiration().
			WithClock(shared).
			WithTickerClock(time.Hour),
	)
	if err != nil {
		t.Fatalf("new cache with final ticker clock failed: %v", err)
	}
	if owned.clock == shared || !owned.clockOwned {
		t.Fatalf("final WithTickerClock selected clock=%T owned=%v, want new owned clock", owned.clock, owned.clockOwned)
	}
	owned.Stop()

	select {
	case <-shared.stop:
		t.Fatal("cache stopped the caller-owned shared clock")
	default:
	}
}
