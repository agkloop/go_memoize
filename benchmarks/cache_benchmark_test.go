package benchmarks

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/memory"
)

func BenchmarkMemoryHotHit(b *testing.B) {
	ctx := context.Background()
	cache, err := memoize.New[string, string](memoize.Opts().WithStore(memory.New[string, string](1024)).WithTTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	if err := cache.Set(ctx, "key", "value"); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = cache.Get(ctx, "key")
	}
}

func BenchmarkMemoryColdMiss(b *testing.B) {
	ctx := context.Background()
	cache, err := memoize.New[string, int](memoize.Opts().WithStore(memory.New[string, int](1024)).WithTTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	idx := 0
	b.ReportAllocs()
	for b.Loop() {
		key := string(rune('a'+(idx%26))) + string(rune('a'+((idx/26)%26)))
		_, _ = cache.GetOrCompute(ctx, key, func(context.Context) (int, error) { return idx, nil })
		idx++
	}
}

func BenchmarkLRUHotHit(b *testing.B) {
	ctx := context.Background()
	store := memory.New[string, string](1000)
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	compute := func(context.Context) (string, error) { return "v", nil }
	_, _ = c.GetOrCompute(ctx, "hot", compute)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = c.GetOrCompute(ctx, "hot", compute)
	}
}

func BenchmarkSingleHotHit(b *testing.B) {
	ctx := context.Background()
	store := memory.NewSingle[string, string]()
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	compute := func(context.Context) (string, error) { return "v", nil }
	_, _ = c.GetOrCompute(ctx, "hot", compute)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = c.GetOrCompute(ctx, "hot", compute)
	}
}

func BenchmarkShardedHotHit(b *testing.B) {
	ctx := context.Background()
	store := memory.NewSharded[string, string](100, memory.WithShards[string, string](16))
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	compute := func(context.Context) (string, error) { return "v", nil }
	_, _ = c.GetOrCompute(ctx, "hot", compute)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = c.GetOrCompute(ctx, "hot", compute)
	}
}

func BenchmarkParallelHotHit(b *testing.B) {
	ctx := context.Background()
	store := memory.New[string, string](1000)
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	_ = c.Set(ctx, "hot", "value")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = c.Get(ctx, "hot")
		}
	})
}

func BenchmarkParallelSingleHotHit(b *testing.B) {
	ctx := context.Background()
	store := memory.NewSingle[string, string]()
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	_ = c.Set(ctx, "hot", "value")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = c.Get(ctx, "hot")
		}
	})
}

func BenchmarkParallelShardedHotHit(b *testing.B) {
	ctx := context.Background()
	store := memory.NewSharded[string, string](100, memory.WithShards[string, string](32))
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	_ = c.Set(ctx, "hot", "value")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = c.Get(ctx, "hot")
		}
	})
}

func BenchmarkMixedWorkload(b *testing.B) {
	ctx := context.Background()
	const keyspace = 10_000
	store := memory.New[string, int](keyspace)
	c, _ := memoize.New[string, int](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	keys := make([]string, keyspace)
	for i := range keyspace {
		keys[i] = fmt.Sprintf("key-%d", i)
		_ = c.Set(ctx, keys[i], i)
	}
	var workerStart atomic.Int64
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		n := int(workerStart.Add(1))
		for pb.Next() {
			key := keys[n%keyspace]
			if n%5 == 0 { // 20% writes
				_ = c.Set(ctx, key, n)
			} else { // 80% reads
				_, _, _ = c.Get(ctx, key)
			}
			n++
		}
	})
}

func BenchmarkEvictionPressure(b *testing.B) {
	ctx := context.Background()
	const keyspace = 10_000
	keys := make([]string, keyspace)
	for i := range keyspace {
		keys[i] = fmt.Sprintf("k%d", i)
	}
	// Tiny store forces eviction on almost every Set
	store := memory.New[string, string](100)
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Hour),
	)
	defer c.Stop()
	var idx atomic.Int64
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		const batchSize = 256
		n := idx.Add(batchSize) - batchSize
		end := n + batchSize
		for pb.Next() {
			if n == end {
				n = idx.Add(batchSize) - batchSize
				end = n + batchSize
			}
			key := keys[int(n%keyspace)]
			_ = c.Set(ctx, key, key)
			n++
		}
	})
}

func BenchmarkGetOrComputeStampede(b *testing.B) {
	ctx := context.Background()
	store := memory.New[string, string](1)
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Nanosecond).WithClock(memoize.ClockFunc(time.Now)),
	)
	defer c.Stop()
	compute := func(context.Context) (string, error) {
		time.Sleep(10 * time.Microsecond)
		return "value", nil
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.GetOrCompute(ctx, "shared", compute)
		}
	})
}

func BenchmarkGetOrComputeStaleStampede(b *testing.B) {
	ctx := context.Background()
	store := memory.New[string, string](1)
	c, _ := memoize.New[string, string](
		memoize.Opts().WithStore(store).WithTTL(time.Nanosecond).WithStaleTTL(time.Hour).WithClock(memoize.ClockFunc(time.Now)),
	)
	defer c.Stop()
	if err := c.Set(ctx, "shared", "stale"); err != nil {
		b.Fatal(err)
	}
	time.Sleep(time.Microsecond)
	compute := func(context.Context) (string, error) {
		time.Sleep(10 * time.Microsecond)
		return "fresh", nil
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.GetOrCompute(ctx, "shared", compute)
		}
	})
}
