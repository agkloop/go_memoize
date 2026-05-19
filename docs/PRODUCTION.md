# Production Guide

This guide describes production choices for the root module `github.com/agkloop/go_memoize`. Public examples should import the root package and root subpackages only.

## Production Defaults And Required Choices

`go_memoize` keeps defaults minimal so production code makes cache topology and freshness explicit.

| Area | Behavior | Production choice |
|---|---|---|
| Explicit `memoize.New[K,V]` store | No default store. | Pass `memoize.Opts().WithStore(...)`, or intentionally use `Bypass()`. |
| Explicit `memoize.New[K,V]` freshness | No default expiration policy. | Choose `WithTTL`, `NoExpiration`, or `Bypass`; missing policy returns `ErrMissingExpirationPolicy`. |
| Direct memoizer store | Internal unbounded `Store[uint64,V]` when `WithStore` is omitted. | Use only when unbounded in-process key growth is acceptable. Pass `WithStore(memory.New[uint64,V](capacity))` or `WithStore(chain.New[uint64,V](...))` for bounded or tiered direct memoizers. |
| Direct memoizer freshness | No default expiration policy. | Choose `WithTTL`, `NoExpiration`, or `Bypass`. |
| Metrics | Disabled/noop. | Pass `WithMetrics` where hit rate, stale behavior, refresh errors, or writes need observability. |
| Refresh timeout | 30 seconds. | Use `WithRefreshTimeout` when stale refresh must respect a stricter dependency SLO. |
| Clock lifecycle | Default ticker-backed clock. | Call `cache.Stop()` for explicit caches during shutdown. |

## Which API Should I Use?

Use direct memoizers when the function arguments are the cache key and an in-process function wrapper is enough:

```go
loadProfile, err := memoize.MemoizeCtx1E(
    repo.LoadProfile,
    memoize.Opts().WithTTL(time.Minute),
)
```

Use explicit caches when the service owns domain cache keys, needs `Get`, `Set`, `Delete`, multi-tier stores, shared stores, or request-path `GetOrCompute`:

```go
cache, err := memoize.New[string, Profile](
    memoize.Opts().
        WithStore(memory.New[string, Profile](50_000)).
        WithTTL(time.Minute),
)
```

Use `background.Keep`, `background.Mirror`, or `loader.New` when there is one logical snapshot value and request handlers should not be responsible for refreshing it.

## Store Selection

| Store or helper | Best fit | Production notes |
|---|---|---|
| `memory.New[K,V](capacity)` | Many bounded in-memory keys. | Default in-process LRU choice. Use `K=string` for explicit business keys and `K=uint64` for direct memoizer backing stores. |
| `memory.NewSharded[K,V](capacity)` | Many distributed hot keys under concurrent access. | Improves distributed-key concurrency; one hot key still maps to one shard. |
| `memory.NewSingle[K,V]()` | One logical key inside the cache engine. | Avoids LRU/hash overhead for one-value workloads. |
| `chain.New[K,V](tiers...)` | Multi-tier caches. | Put the fastest tier first; lower-tier hits backfill earlier tiers. |
| `local.New[V](dir)` | Local restart persistence. | File-backed `Store[string,V]`; values must be Gob-encodable. Not for cross-host sharing. |
| Redis adapter | Shared multi-process or multi-host cache. | Separate module under `adapters/redis`; use prefixes and serializers deliberately. |
| Custom SQL store | Durable shared cache with queryable backend. | Implement `memoize.Store[K,V]` and store the full `memoize.Stored[V]` envelope. |
| S3/object store | Durable object-backed L2 or snapshot storage. | Usually too slow for hot L1; use behind memory with `chain.New`. |
| `background.Keep` | One periodically refreshed in-process snapshot. | Blocks until first load succeeds, then refreshes in the background and keeps last good value on refresh errors. |
| `background.Mirror` | Local in-process mirror of one shared `Store[string,V]` key. | Initial remote read must succeed; request handlers call `Value.Get()` for atomic local reads, while the mirror goroutine polls the shared store on the interval. |
| `loader.New` | Readiness-gated periodic value. | Use when callers should block until first successful load through `Value(ctx)`. |

## Freshness And Stale Behavior

Use `WithTTL` for the fresh window. Add `WithStaleTTL` when stale reads are acceptable and recompute should happen asynchronously after the fresh window.

```go
cache, err := memoize.New[string, Product](
    memoize.Opts().
        WithStore(memory.New[string, Product](100_000)).
        WithTTL(15*time.Second).
        WithStaleTTL(5*time.Minute).
        WithRefreshTimeout(3*time.Second).
        KeepStaleOnError(),
)
```

Fresh hits return immediately. Stale hits return the old value immediately and start a refresh. Expired misses block on recompute. `KeepStaleOnError` keeps the last stored value serving through refresh failures where the cache entry still exists.

Use `NoExpiration` only for values whose lifetime is controlled by explicit writes, deletes, process lifetime, or background replacement.

## Request-Path Caching

Request handlers should pass request contexts to `GetOrCompute`; the compute function receives the same context.

```go
type UserRepo interface {
    LoadUser(context.Context, int64) (User, error)
}

func NewUserCache() (*memoize.Cache[string, User], error) {
    return memoize.New[string, User](
        memoize.Opts().
            WithStore(memory.New[string, User](50_000,
                memory.WithMaxBytes[string, User](128<<20),
            )).
            WithTTL(30*time.Second).
            WithStaleTTL(2*time.Minute).
            KeepStaleOnError(),
    )
}

func GetUser(ctx context.Context, cache *memoize.Cache[string, User], repo UserRepo, id int64) (User, error) {
    key := fmt.Sprintf("user:%d", id)
    return cache.GetOrCompute(ctx, key, func(ctx context.Context) (User, error) {
        return repo.LoadUser(ctx, id)
    })
}
```

Use explicit string keys that include tenant, account, locale, or authorization scope when those dimensions affect the result.

## Direct Memoizer Production Patterns

Direct memoizers hash arguments to `uint64` keys. They are convenient for repository or client methods where the function signature already defines the cache key.

```go
m := metrics.NewInMemoryMetrics()

loadProfile, err := memoize.MemoizeCtx1E(
    repo.LoadProfile,
    memoize.Opts().
        WithTTL(time.Minute).
        WithStaleTTL(5*time.Minute).
        KeepStaleOnError().
        WithMetrics(m),
)
```

The default internal unbounded direct store is appropriate only when unbounded in-process key growth is acceptable. For production services with unknown cardinality, pass a bounded direct store:

```go
loadProfile, err := memoize.MemoizeCtx1E(
    repo.LoadProfile,
    memoize.Opts().
        WithStore(memory.New[uint64, Profile](50_000)).
        WithTTL(time.Minute).
        WithStaleTTL(5*time.Minute).
        KeepStaleOnError(),
)
```

For tiered direct memoizers, every tier must use `uint64` keys:

```go
store := chain.New[uint64, Profile](
    memory.New[uint64, Profile](10_000),
    directL2Store,
)

loadProfile, err := memoize.MemoizeCtx1E(
    repo.LoadProfile,
    memoize.Opts().WithStore(store).WithTTL(time.Minute),
)
```

Prefer explicit `memoize.New[string,V]` when cache keys need to be stable across languages, visible to operators, shared across services, or manually invalidated.

## Multi-Tier Caching

Use `chain.New` to combine fast in-process reads with a slower durable or shared tier.

```go
l1 := memory.New[string, Report](10_000)
l2 := local.New[Report]("/var/cache/myapp/reports")

cache, err := memoize.New[string, Report](
    memoize.Opts().
        WithStore(chain.New[string, Report](l1, l2)).
        WithTTL(10*time.Minute),
)
```

Put the fastest tier first. Use local files for restart persistence on one host, Redis for cross-process sharing, and object stores as durable lower tiers when their latency is acceptable.

## Redis

The Redis adapter is a separate module under `adapters/redis`. Test it from that module when adapter code or examples change.

```go
client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

redisStore, err := redisstore.New[string, User](
    redisstore.WithClient[string, User](client),
    redisstore.WithPrefix[string, User]("users"),
    redisstore.WithSerializer[string, User](serializers.JSON[User]{}),
)
if err != nil {
    return err
}

cache, err := memoize.New[string, User](
    memoize.Opts().WithStore(redisStore).WithTTL(time.Minute),
)
```

Use a prefix per service or domain to avoid key collisions. Choose JSON for debuggability, Gob for Go-only payloads, or `serializers.Func` for custom formats. Redis backend TTL is cleanup through the stale deadline; public freshness still comes from the `memoize.Stored[V]` envelope interpreted by the cache engine.

For direct memoizer stores backed by Redis, use `redisstore.New[uint64,V]` because direct memoizers store hashed argument keys.

## Single Writer, Many Reader Distributed Snapshots

For full snapshots such as e-commerce category trees, avoid making every API replica load the same snapshot from MySQL and avoid making every request hit Redis. Run one cache-refresher writer and many API readers that mirror the shared value into local atomic memory.

```text
cache-refresher (1 replica)
    MySQL -> background.Keep -> background.WriteThrough -> Redis/shared store

api-service (many replicas)
    Redis/shared store -> background.Mirror -> local atomic value -> HTTP responses
```

The refresher owns the database load and publishes each successful refresh:

```go
categories, err := background.Keep(ctx, loadCategoriesFromMySQL, time.Minute,
    background.WriteThrough[CategorySnapshot]("categories:v1", redisStore),
    background.OnError[CategorySnapshot](func(err error) {
        log.Printf("category refresh failed: %v", err)
    }),
)
```

Each API pod mirrors Redis into a local atomic value and serves requests from memory:

```go
categories, err := background.Mirror(ctx, "categories:v1", redisStore, 5*time.Second,
    background.OnError[CategorySnapshot](func(err error) {
        log.Printf("category mirror failed: %v", err)
    }),
)

func handler(w http.ResponseWriter, r *http.Request) {
    snapshot := categories.Get()
    _ = snapshot
}
```

Use `background.Keep` plus `WriteThrough` plus `Mirror` for full snapshots where the last known good value should remain until it is replaced. The stored value is written with no-expiration metadata and replacement is controlled by the refresher loop.

`background.MustMirror` loads the shared store value into each API pod's process memory during startup. After startup, `categories.Get()` is only an atomic local memory read of the last successful mirror refresh; it does not call Redis, MySQL, or any other remote dependency on the request path, does not block, and does not return an error. Keep mirrored snapshots immutable or copy mutable fields before editing them.

Use `memoize.Cache.Set` plus `Cache.Get` for normal cache entries that should carry `WithTTL` and `WithStaleTTL` metadata. This is better for independently keyed data where freshness windows and cache misses matter per key.

The refresher deployment must enforce one writer with Kubernetes replicas, leader election, or job semantics. `go_memoize` does not provide distributed leader election.

## Custom Stores Including S3

Implement a custom store when built-in memory, chain, local, and Redis stores do not match your persistence or topology needs. A custom store implements `memoize.Store[K,V]` and stores raw `memoize.Stored[V]` envelopes:

```go
type Store[K comparable, V any] interface {
    Get(context.Context, K) (memoize.Stored[V], bool, error)
    Set(context.Context, K, memoize.Stored[V]) error
    Delete(context.Context, K) error
    Clear(context.Context) error
}
```

Rules:

- Return stale entries as stored; the cache engine decides whether they are fresh, stale, or expired.
- Use `K=string` for business keys and `K=uint64` for direct memoizer backing stores.
- Keep backend cleanup TTL separate from public cache freshness.
- Make `Clear` safe for the store scope; use prefixes or namespaces for shared backends.

SQL stores should encode the full `Stored[V]` envelope, not only the value, so freshness metadata survives process restarts and cross-process reads.

S3 or any object store can be a custom store. Store the encoded `memoize.Stored[V]` envelope as the object body and map the typed cache key to an object key. S3 is usually a durable L2, not a hot L1, because latency and request cost are higher than memory or Redis:

```go
store := chain.New[string, User](
    memory.New[string, User](10_000),
    s3UserStore, // memoize.Store[string, User]
)

cache, err := memoize.New[string, User](
    memoize.Opts().
        WithStore(store).
        WithTTL(time.Minute).
        WithStaleTTL(10*time.Minute),
)
```

For direct memoizers, an object-store tier must be `memoize.Store[uint64,V]`. Encode the `uint64` hash as a decimal string or another stable object key format. Use object lifecycle policies only for backend cleanup; do not use them as the cache freshness policy.

## Serializers

External stores that encode values use `memoize.Serializer[V]`:

```go
type Serializer[V any] interface {
    Marshal(V) ([]byte, error)
    Unmarshal([]byte) (V, error)
}
```

Use `serializers.JSON[V]` for debuggable payloads, `serializers.Gob[V]` for Go-only payloads, or `serializers.Func[V]` for custom formats such as protobuf, msgpack, compression, or encryption. Serializer implementations should be deterministic enough for operational debugging and should treat decode failures as data corruption or cache misses according to the adapter semantics.

## Metrics And Cardinality

```go
type Metrics struct{}

func (Metrics) RecordMetric(event memoize.MetricEvent) {
    switch event.Kind {
    case memoize.MetricHit:
        recordCounter("cache_hit")
    case memoize.MetricMiss:
        recordCounter("cache_miss")
    case memoize.MetricStaleHit:
        recordCounter("cache_stale_hit")
    case memoize.MetricRefreshSuccess:
        recordDuration("cache_refresh", event.Duration)
    case memoize.MetricRefreshError:
        recordError("cache_refresh", event.Err)
    }
}
```

`MetricEvent.Key` is usually the exact cache entry key. Do not export it as a production metric label unless the keyspace is intentionally bounded; user IDs, URLs, query strings, tenant IDs, and direct memoizer hash keys can create high-cardinality metrics. Prefer labels for cache name, operation, result, store tier, and service.

Track hit rate, stale hits, refresh errors, refresh latency, set/delete rates, and backend errors. `metrics.NewInMemoryMetrics()` is useful for tests and local diagnostics, not as a production metrics backend.

## Shutdown

- Call `cache.Stop()` for explicit caches using the default ticker clock.
- Cancel the context passed to `background.Keep` or `background.Mirror` to stop refresh loops.
- Call `loader.Stop()` to stop loader goroutines.
- Pass request contexts into `GetOrCompute`; compute functions receive the same context.
- Treat values returned by `background.Value.Get()` as shared memory. Use immutable structs, copied maps, or read-only conventions.

## Testing And Release Checklist

- Run root tests: `go test ./... -count=1`.
- Run root race tests: `go test ./... -race -count=1`.
- If Redis adapter code or examples changed, test from `adapters/redis`: `go test ./... -count=1` and `go test ./... -race -count=1`.
- Run example tests when examples or docs examples change.
- Confirm public docs and examples use only root-module import paths.
- Confirm explicit `memoize.New[K,V]` examples choose both a store and an expiration policy.
- Confirm direct memoizer examples choose `WithTTL`, `NoExpiration`, or `Bypass`, and use bounded or tiered `Store[uint64,V]` when unbounded in-process key growth is not acceptable.
