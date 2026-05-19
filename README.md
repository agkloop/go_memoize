# go_memoize

![Workflow Status](https://github.com/agkloop/go_memoize/actions/workflows/ci.yml/badge.svg)

`go_memoize` is a Go memoization and caching package with one public root module for direct function memoization, explicit cache-engine workflows, in-memory stores, background snapshots, loaders, metrics, and adapters.

## Install

```sh
go get github.com/agkloop/go_memoize
```

## Choose The Right API

| Need | Use | Why |
|---|---|---|
| Memoize a function by comparable args | `memoize.Memoize1(fn, memoize.Opts().WithTTL(ttl))` through `Memoize7` | Small direct API with cache-engine options. |
| Memoize a function that can fail | `memoize.Memoize1E` or `memoize.MemoizeCtx1E` | Errors are returned and successful values are cached. |
| Cache explicit business keys | `memoize.New[K,V]` + `Cache.GetOrCompute` | Full control over key type, store, TTL, stale behavior, metrics, and shutdown. |
| Many bounded in-memory keys | `memory.New[K,V](capacity)` | Exact LRU by default. |
| One logical value or one hot key | `memory.NewSingle[K,V]()` or `background.Keep` | Avoids unnecessary LRU/hash overhead. |
| Many hot keys concurrently | `memory.NewSharded[K,V](capacity)` | Distributes different supported primitive keys across shards. |
| Cross-process or cross-host cache | Redis adapter or a custom shared store | In-process memory stores are not shared between processes. |
| One scheduled snapshot with instant reads | `background.Keep` or `background.Mirror` | Refreshes independently of request traffic. |

## Quick Examples

Direct TTL memoizer:

```go
cached, err := memoize.Memoize1(loadUser, memoize.Opts().WithTTL(time.Minute))
if err != nil {
    return err
}
user := cached(42)
```

Direct stale-on-error memoizer:

```go
cached, err := memoize.MemoizeCtx1E(repo.LoadProfile,
    memoize.Opts().
        WithTTL(time.Minute).
        WithStaleTTL(5*time.Minute).
        KeepStaleOnError(),
)
if err != nil {
    return err
}
profile, err := cached(ctx, 42)
```

Explicit cache with business keys:

```go
cache, err := memoize.New[string, User](
    memoize.Opts().
        WithStore(memory.New[string, User](10_000)).
        WithTTL(time.Minute),
)
if err != nil {
    return err
}
defer cache.Stop()

user, err := cache.GetOrCompute(ctx, "user:42", func(ctx context.Context) (User, error) {
    return repo.LoadUser(ctx, 42)
})
```

## Defaults

`memoize.New[K,V]` intentionally does not choose a store or expiration policy for you:

| Setting | Default |
|---|---|
| Store | None. Use `Opts().WithStore(store)` unless using `Bypass()`. |
| Expiration policy | None. Choose `WithTTL`, `NoExpiration`, or `Bypass`. |
| Metrics | Disabled with a noop metrics implementation. Enable with `WithMetrics`. |
| Clock | Ticker-backed clock with a 1ms tick. Call `cache.Stop()` to release it. |
| Refresh timeout | 30 seconds for background stale refresh. Override with `WithRefreshTimeout`. |
| Same-key miss coalescing | Enabled internally for concurrent `GetOrCompute` misses on the same key. |

Direct `Memoize*` functions use the same options. If `WithStore` is omitted, they create an internal unbounded `Store[uint64,V]` for hashed argument keys. Direct memoizers still require `WithTTL`, `NoExpiration`, or `Bypass`; they do not silently choose an expiration policy.

## Architecture And Performance

go_memoize uses one root cache engine for both direct memoizers and explicit caches.

- Direct memoizers hash comparable arguments to `uint64` and use the same cache engine as `memoize.New`.
- Stores persist raw `memoize.Stored[V]` envelopes; the cache engine owns fresh, stale, and expired decisions.
- `GetOrCompute` coalesces same-key concurrent misses through an internal singleflight map.
- Memory stores provide exact LRU behavior, optional byte limits, and private fast paths used by the cache engine.
- Cache stale refresh uses cache-engine flight machinery; `background.Keep` and `loader.New` share periodic refresh-loop infrastructure internally.
- Metrics use one event method, `RecordMetric(MetricEvent)`, to keep hot-path instrumentation small and backend-neutral.

Latest benchmark snapshot from this checkout used:

```sh
go test ./benchmarks/ -bench=. -benchmem -benchtime=1s -count=1
```

Top-line results on this machine: `BenchmarkMemoryHotHit` was `29.80 ns/op` with `0 B/op` and `0 allocs/op`; `BenchmarkSingleHotHit` was `14.74 ns/op`; `BenchmarkGetOrComputeStampede` was `1747 ns/op` with `0 allocs/op`; stale stampede was `399.8 ns/op` with `0 allocs/op`. Sharding did not improve a single hot key (`BenchmarkMemoryHotHitParallel` `150.9 ns/op`, sharded `150.0 ns/op`). See `docs/PERFORMANCE.md` for full output and interpretation.

## Production Recommendations

- Use direct `Memoize*` functions for simple in-process function memoization.
- Use `Cache.GetOrCompute` when keys are business identifiers, when you need custom stores, or when stale refresh matters.
- Use `background.Keep` with a shared store for a single-writer snapshot refresher, and `background.Mirror` in reader processes for local atomic reads.
- Custom stores implement `memoize.Store[K,V]` and must store raw `memoize.Stored[V]` envelopes; the cache engine owns freshness decisions.
- S3 and other object stores can be custom durable L2 stores behind `chain.New`, but should not be the first cache tier for low-latency hot paths.
- Use `memory.NewSingle` or `background.Keep` for one logical value such as config, feature flags, or exchange rates.
- Use `memory.NewSharded` only when many different supported primitive keys are hot concurrently; one key still maps to one shard.
- Use the Redis adapter or another shared store when multiple processes or hosts need the same backing cache.
- Always `defer cache.Stop()` for explicit caches using the default ticker clock.
- Run `go test ./... -count=1` and `go test ./... -race -count=1` before release.

## Documentation

- `docs/GETTING_STARTED.md` - install and first working examples.
- `docs/CONCEPTS.md` - direct vs explicit cache, keys, stores, TTL/stale, and defaults.
- `docs/RECIPES.md` - copy-paste usage patterns.
- `docs/PERFORMANCE.md` - internal architecture, performance optimizations, and benchmark results.
- `docs/API.md` - full public API reference.
- `docs/PRODUCTION.md` - production store selection, observability, and failure behavior.
- `examples/direct_stale_profile_cache` - direct memoizer stale-on-error example.
- `examples/http_user_cache` - explicit cache with business keys.
- `adapters/redis/examples/hybrid_profile_cache` - direct memoizer with memory L1 and Redis L2.
