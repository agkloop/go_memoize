# Performance And Architecture

This document explains the performance-sensitive architecture inside `go_memoize` and records the latest local benchmark run. Treat the benchmark numbers as a point-in-time reference, not a portable guarantee.

## Internal Architecture

Direct memoizers and explicit caches share the root cache engine. The public `Memoize`, `Memoize1` through `Memoize7`, and context/error variants build a `Cache[uint64, V]` internally, while explicit cache users build typed caches with `memoize.New[K, V]`.

Direct memoizers hash comparable arguments to a `uint64` cache key. Explicit caches use caller key types such as `string`, `int64`, or other comparable key types selected by the application. Stores persist raw `memoize.Stored[V]` entries; the cache engine owns freshness, staleness, expiration, and stale-on-error policy rather than delegating those decisions to the store.

By default, direct memoizers use an internal unbounded map-backed store. Use `memoize.Opts().WithStore(...)` when direct memoization needs a bounded memory store, sharded store, chain store, or adapter-backed store.

## Cache Engine Hot Path

`Cache.GetOrCompute` is the central hot path. It first checks whether a built-in store can return a fresh value directly, then handles same-key miss coalescing, stored-entry state, stale-while-revalidate, miss computation, and optional stale fallback on compute error.

The cache engine coalesces concurrent same-key misses internally. One caller becomes the leader and computes the value; followers wait for the active flight and receive the same result. This avoids stampedes for cold keys and for refresh paths that converge on the same key.

Explicit caches own a default ticker clock and should be shut down with `Stop` when the cache lifetime ends. `WithTickerClock` also creates a cache-owned clock. `WithClock` injects a caller-owned clock instead, allowing several caches to share one clock without an individual cache stopping it. Clock construction happens only after cache options validate, so failed construction and injected clocks do not leave a hidden default ticker behind.

## Direct Memoizer Keying

Direct memoizers convert function arguments into a `uint64` key before calling the shared cache engine. Zero-argument memoizers use key `0`; one-argument and multi-argument memoizers hash comparable arguments into the direct cache key space.

This design keeps the public direct memoization API simple while allowing the implementation to reuse the same `Cache.GetOrCompute` policy as explicit caches. When a custom store is supplied to a direct memoizer, it must be a `memoize.Store[uint64, V]` because the direct memoizer has already converted the original arguments to a `uint64` key.

## Store Fast Paths

The store fast paths are implementation notes, not public extension points. Built-in stores may implement private interfaces used by the cache engine to avoid avoidable allocation and work on fresh hits.

For example, the memory stores can prove that an entry is fresh and return only the value needed by the caller. That lets fresh hits avoid loading full entry metadata through the generic store interface and skip extra policy work when the entry is already usable.

External stores only need to implement the public `memoize.Store[K, V]` interface. Private fast paths may change as the implementation changes.

## Memory Store Design

`memory.New` keeps an exact LRU with a fixed item capacity. Direct `Store.Get` calls and fresh cache-engine hits both refresh recency; `WithGetRecencySample(n)` can make those updates approximate to reduce contention. Optional byte limits use a shallow entry-size estimate; heap allocations inside values such as string contents or slice backing arrays are not counted.

`memory.NewSingle` stores one logical value and avoids LRU and hash overhead on the hot read path. Use it for one cached snapshot, one global configuration value, or one hot key. It is not a general replacement for many-key caches.

`memory.NewSharded` splits total capacity across independent LRU shards. It improves distributed-key concurrency by reducing mutex contention across many keys, but it does not improve one-hot-key contention because one key maps to one shard.

## Stale Refresh Loop

Stale refresh returns the stale value immediately and starts a background refresh for that key. The same internal flight mechanism prevents duplicate refreshes for the same key while one refresh is already running.

`background.Keep` and `loader.New` share refresh-loop infrastructure internally. Cache stale refresh uses the cache engine's flight machinery instead because it refreshes one key at a time after stale hits, while `background` and `loader` run periodic whole-value refresh loops.

## Metrics Event Model

Metrics use a single event method: `RecordMetric(memoize.MetricEvent)`. Events carry a kind, string key, optional duration, and optional error. The cache engine emits events for hits, misses, stale hits, refresh start/success/error, set, and delete.

The single event method keeps instrumentation cheap for the cache engine and lets metrics implementations decide whether to aggregate, sample, export, or ignore events.

## Benchmark Methodology

Command:

```sh
go test ./benchmarks/ -bench=. -benchmem -benchtime=1s -count=1
```

Go version:

```text
go version go1.25.5 darwin/arm64
```

Machine caveat: these results were recorded on `darwin/arm64`, CPU `Apple M2 Pro`. Benchmark numbers depend on Go version, CPU, OS scheduler, power state, background load, and benchmark shape.

Full benchmark output:

```text
go version go1.25.5 darwin/arm64
goos: darwin
goarch: arm64
pkg: github.com/agkloop/go_memoize/benchmarks
cpu: Apple M2 Pro
BenchmarkDo0Mem-10                       	23160691	        52.38 ns/op	      24 B/op	       1 allocs/op
BenchmarkDo0LRU-10                       	47512914	        23.81 ns/op	       0 B/op	       0 allocs/op
BenchmarkDo1Mem-10                       	20862187	        59.20 ns/op	      48 B/op	       1 allocs/op
BenchmarkDo1LRU-10                       	30650035	        39.37 ns/op	       0 B/op	       0 allocs/op
BenchmarkDo2Mem-10                       	17754127	        67.13 ns/op	      64 B/op	       1 allocs/op
BenchmarkDo2LRU-10                       	28463869	        45.57 ns/op	       0 B/op	       0 allocs/op
BenchmarkDo3Mem-10                       	16592020	        73.02 ns/op	      80 B/op	       1 allocs/op
BenchmarkDo3LRU-10                       	26605867	        44.54 ns/op	       0 B/op	       0 allocs/op
BenchmarkDo4Mem-10                       	16346425	        72.76 ns/op	      80 B/op	       1 allocs/op
BenchmarkDo4LRU-10                       	24477202	        46.47 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemoryHotHit-10                 	40423036	        29.80 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemoryColdMiss-10               	17744544	        67.53 ns/op	      18 B/op	       2 allocs/op
BenchmarkLRUHotHit-10                    	41497600	        28.59 ns/op	       0 B/op	       0 allocs/op
BenchmarkSingleHotHit-10                 	80741475	        14.74 ns/op	       0 B/op	       0 allocs/op
BenchmarkShardedHotHit-10                	44206764	        28.13 ns/op	       0 B/op	       0 allocs/op
BenchmarkParallelHotHit-10               	 7884925	       150.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkParallelSingleHotHit-10         	493886186	         2.275 ns/op	       0 B/op	       0 allocs/op
BenchmarkParallelShardedHotHit-10        	 8078821	       150.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkMixedWorkload-10                	 6738607	       186.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkEvictionPressure-10             	 3514544	       331.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetOrComputeStampede-10         	  702915	      1747 ns/op	       4 B/op	       0 allocs/op
BenchmarkGetOrComputeStaleStampede-10    	 3021088	       399.8 ns/op	       2 B/op	       0 allocs/op
PASS
ok  	github.com/agkloop/go_memoize/benchmarks	29.210s
```

## Latest Benchmark Results

The latest local run shows allocation-free fresh hits for the built-in bounded, single-value, and sharded memory stores. Representative fresh-hit results were `BenchmarkMemoryHotHit` at `29.80 ns/op`, `BenchmarkLRUHotHit` at `28.59 ns/op`, `BenchmarkSingleHotHit` at `14.74 ns/op`, and `BenchmarkShardedHotHit` at `28.13 ns/op`, all with `0 B/op` and `0 allocs/op`.

The single-value store is the fastest result for the one-logical-value workload: `BenchmarkParallelSingleHotHit` measured `2.275 ns/op`, compared with `150.9 ns/op` for the regular memory store and `150.0 ns/op` for the sharded store under the one-hot-key parallel benchmark. This matches the design expectation: sharding helps distributed keys, not one hot key.

Direct memoization benchmarks show the bounded LRU-backed variants avoiding allocations on repeated hits, while the default direct map-backed benchmark path records one allocation in these microbenchmarks.

## Reading The Numbers

Read these numbers as workload-specific signals. They compare implementation choices on this machine under the benchmark shapes in `./benchmarks`, not universal latency guarantees.

The relative shape matters more than any one number: `memory.NewSingle` is best for one logical value, `memory.NewSharded` is for distributed-key concurrency, and `memory.New` is the default exact-LRU choice for many bounded keys. If your workload has larger values, remote stores, serialization, expensive compute functions, different contention patterns, or different TTL behavior, your bottleneck may move.

## When To Benchmark Your Workload

Benchmark your workload when changing key shape, store type, TTL/stale policy, concurrency level, value size, serialization, or compute cost. Also benchmark when choosing between `memory.New`, `memory.NewSingle`, and `memory.NewSharded`; their tradeoffs depend on whether your application has many keys, one logical value, distributed contention, or a single hot key.

Use the package benchmarks as a starting point, then add a benchmark that resembles your production access pattern before optimizing further.
