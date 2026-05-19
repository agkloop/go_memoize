# Package Context

This package has one public Module, `github.com/agkloop/go_memoize`. It contains two primary user-facing Modules: direct memoization for functions and an explicit cache engine for keyed caching.

## Domain Vocabulary

Direct memoization is the Shallow Interface for common function caching. Users pass a function plus a TTL, and the Implementation derives keys from comparable arguments. Use it when Locality matters more than policy control: in-process values, no custom Store Adapter, no stale-while-revalidate, and no explicit metrics wiring.

The cache engine is the Deep Module. Its public Interface is small: `New`, `Cache.Get`, `Cache.Set`, `Cache.Delete`, `Cache.Clear`, `Cache.GetOrCompute`, and `Cache.Stop`. Its Implementation owns TTL policy, expiration ownership, stale-while-revalidate, flight coalescing, fallback on refresh error, and metrics emission. This Depth gives users Leverage because Store Adapters can stay simple while cache behavior remains consistent.

A Store Adapter implements the `Store[K,V]` Interface: `Get`, `Set`, `Delete`, and `Clear`. Stores persist and return raw `Stored[V]` entries. They do not decide whether an entry is fresh, stale, or expired for public cache reads. That expiration ownership belongs to the cache engine. Store-specific cleanup TTLs, such as Redis key expiry, are backend cleanup, not the public freshness policy.

`Stored[V]` is the cache entry envelope. It carries `Value`, `CreatedAt`, `FreshUntil`, `StaleUntil`, `NoExpire`, `Version`, and `Tags`. The cache engine interprets the envelope; stores preserve it.

Stale-while-revalidate means a cache hit can return an entry after `FreshUntil` but before `StaleUntil`, then refresh in the background. If configured with stale-on-error behavior, the cache can keep serving the stale value when recompute fails.

A background value is one locally served value refreshed on a schedule through `background.Keep` or mirrored from a store through `background.Mirror`. It is best for config snapshots, feature flags, rates, and other one-value workloads.

A loader is a periodic refresh Module exposed by `loader.New`. It retries until the first successful load, then `Value(ctx)` returns the latest successful value. Loader and background share an internal periodic refresh Implementation, but expose different public Interfaces for different use cases.

Metrics use one event Interface: `Metrics.RecordMetric(MetricEvent)`. `MetricEventKind` identifies hits, misses, stale hits, refresh start/success/error, set, and delete. `Duration` is meaningful for refresh success latency. `Err` is meaningful for refresh errors.

Private cache/store fast paths are Seams inside the Implementation. They let optimized stores provide fresh-value or peek behavior without making the public Store Interface larger. These Seams are tested through observable behavior and deletion tests: removing a private fast path test should reveal whether the Seam still protects policy Locality and avoids accidental coupling.

Test surface should follow Module boundaries. Public behavior tests cover direct memoization, cache policy, and Store Adapter contracts. Private fast-path tests should stay narrow and verify observable effects, not leak private names into user documentation.
