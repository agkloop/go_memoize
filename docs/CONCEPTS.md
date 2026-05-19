# Concepts

## Two APIs, One Cache Engine

`go_memoize` has two public APIs backed by the same cache engine:

- Direct memoizers wrap functions such as `Memoize1`, `MemoizeCtx1E`, and other arity/context/error variants.
- Explicit caches use `memoize.New[K,V]` and `Cache.GetOrCompute` with caller-provided keys.

Use direct memoization when the function arguments are the natural cache key.
Use an explicit cache when you want business keys, bounded stores, lifecycle control, or a cache object shared across call sites.

## Keys

Direct memoizers hash comparable arguments to `uint64` keys.
Because of that, direct custom stores must be `Store[uint64,V]`.

Explicit caches let you choose the key type with `memoize.New[K,V]`.
Explicit cache examples usually use `K=string` because business keys such as `"user:42"` are stable, readable, and easy to share across systems.

## Stores

Stores persist raw `memoize.Stored[V]` entries.
The store is responsible for saving and returning entries, while the cache engine decides whether each stored value is fresh, stale, or expired.

For direct memoizers, provide custom stores as `Store[uint64,V]`.
For explicit caches, match the store key type to the cache key type, such as `memory.New[string, User](10_000)` with `memoize.New[string, User]`.

## Fresh, Stale, Expired

`WithTTL` sets how long a value is fresh.
Fresh values are returned directly without recomputing.

`WithStaleTTL` sets an additional stale window after freshness ends.
During the stale window, the cache may return the stale value while recomputing according to the cache engine behavior.

`KeepStaleOnError` allows a stale value to be returned when recomputation fails.
Without stale-on-error, a failed recomputation returns the error when no fresh value can be used.

After the stale window ends, the value is expired.
Expired values are not usable unless `KeepStaleOnError` is configured to keep serving stale data after recomputation errors.

## Defaults

`memoize.New[K,V]` has no default store and no default expiration policy.
Explicit caches must choose a store with `WithStore` and choose an expiration mode with `WithTTL`, `NoExpiration`, or `Bypass`.

Direct memoizers create an internal unbounded `Store[uint64,V]` when `WithStore` is omitted.
They still require an expiration mode: `WithTTL`, `NoExpiration`, or `Bypass`.

## Errors

Error-returning memoizers cache successful results only.
If the wrapped function returns an error, that failed result is not written as a fresh cached value.

For explicit caches, `GetOrCompute` follows the same principle: successful recomputations are stored, while errors are returned to the caller unless a usable stale value is returned by stale-on-error behavior.

## Shutdown

Explicit caches should call `Stop` when the cache is no longer needed.
This gives the cache engine a lifecycle hook for background work and store cleanup.

Direct memoizers do not expose a shutdown method.
If shutdown control is required, use explicit cache construction with `memoize.New[K,V]` and wrap calls around `Cache.GetOrCompute` yourself.
