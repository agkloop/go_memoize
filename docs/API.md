# API Reference

This reference covers the public API for `github.com/agkloop/go_memoize` and its subpackages. The root module is the only public module; examples should use the root import paths shown below.

## Import Paths

Use the root package for direct memoizers, explicit caches, options, root interfaces, errors, metrics events, and serializers interfaces:

```go
import memoize "github.com/agkloop/go_memoize"
```

Subpackages provide store implementations, serializers, background values, loaders, and optional adapters:

```go
import (
    "github.com/agkloop/go_memoize/background"
    "github.com/agkloop/go_memoize/loader"
    "github.com/agkloop/go_memoize/metrics"
    "github.com/agkloop/go_memoize/serializers"
    "github.com/agkloop/go_memoize/stores/chain"
    "github.com/agkloop/go_memoize/stores/local"
    "github.com/agkloop/go_memoize/stores/memory"
)
```

The Redis adapter is its own module under `adapters/redis`:

```go
import redisstore "github.com/agkloop/go_memoize/adapters/redis"
```

## Direct Memoizers

Direct memoizers wrap functions whose cache keys can be derived from comparable arguments. They return `(func, error)` because they build an internal `memoize.Cache[uint64,V]` and validate the same root `memoize.Opts()` options as explicit caches.

| Function family | Function shape | Notes |
|---|---|---|
| `Memoize` | `func() V` | No-argument value memoization. |
| `Memoize1` ... `Memoize7` | `func(A...) V` | One to seven comparable args; arguments are hashed into a `uint64` key. |
| `MemoizeE` | `func() (V, error)` | Caches successful values only; errors are returned and not stored. |
| `Memoize1E` ... `Memoize7E` | `func(A...) (V, error)` | One to seven comparable args; successful values only. |
| `MemoizeCtx` | `func(context.Context) V` | Context is passed through but is not part of the cache key. |
| `MemoizeCtx1` ... `MemoizeCtx7` | `func(context.Context, A...) V` | Context is not the key; comparable args are hashed. |
| `MemoizeCtxE` | `func(context.Context) (V, error)` | Context-aware, successful values only. |
| `MemoizeCtx1E` ... `MemoizeCtx7E` | `func(context.Context, A...) (V, error)` | Context-aware, one to seven comparable args; successful values only. |

TTL example:

```go
cached, err := memoize.Memoize2(func(tenantID string, userID int64) Profile {
    return loadProfile(tenantID, userID)
}, memoize.Opts().WithTTL(time.Minute))
if err != nil {
    return err
}

profile := cached("acme", 42)
```

Context/error example:

```go
cached, err := memoize.MemoizeCtx1E(func(ctx context.Context, id int64) (User, error) {
    return repo.LoadUser(ctx, id)
}, memoize.Opts().WithTTL(30*time.Second))
if err != nil {
    return err
}

user, err := cached(ctx, 42)
```

Stale-on-error example:

```go
cached, err := memoize.MemoizeCtx1E(loadProfile,
    memoize.Opts().
        WithTTL(time.Minute).
        WithStaleTTL(5*time.Minute).
        KeepStaleOnError(),
)
if err != nil {
    return err
}
```

Custom direct store example:

```go
var store memoize.Store[uint64, Profile] = memory.New[uint64, Profile](10_000)

cached, err := memoize.MemoizeCtx1E(loadProfile,
    memoize.Opts().WithStore(store).WithTTL(time.Minute),
)
```

Use direct memoizers for simple function caching. Use `memoize.New[K,V]` when callers need explicit keys, direct cache operations, tag invalidation, or a store key type other than `uint64`.

## Explicit Cache Engine

`memoize.New[K,V](opts ...memoize.Options) (*memoize.Cache[K,V], error)` creates an explicit cache. Most application caches use `K=string`; caches that mirror direct memoizer keying can use `K=uint64`.

```go
cache, err := memoize.New[string, User](
    memoize.Opts().WithStore(memory.New[string, User](10_000)).WithTTL(time.Minute),
)
if err != nil {
    return err
}
defer cache.Stop()
```

`Cache[K,V]` methods:

| Method | Meaning |
|---|---|
| `Get(ctx, key)` | Reads a fresh value. Missing, expired, or stale entries return a miss. |
| `Set(ctx, key, value)` | Stores a value using the cache expiration policy. |
| `Delete(ctx, key)` | Deletes one key. |
| `Clear(ctx)` | Clears the backing store. |
| `GetOrCompute(ctx, key, fn)` | Returns a fresh cached value or computes, stores, and returns it. Concurrent misses for the same key are coalesced. |
| `Stop()` | Releases cache-owned ticker-clock resources. It does not stop clocks supplied with `WithClock`. Safe to call more than once. |

The cache engine owns freshness decisions. Stores persist `memoize.Stored[V]` envelopes and return entries even when they might be stale or expired; `Cache[K,V]` decides whether to serve, refresh, or miss.

`GetOrCompute` example:

```go
user, err := cache.GetOrCompute(ctx, "user:42", func(ctx context.Context) (User, error) {
    return repo.LoadUser(ctx, 42)
})
```

Stale-while-revalidate with stale-on-error:

```go
cache, err := memoize.New[string, User](
    memoize.Opts().
        WithStore(memory.New[string, User](10_000)).
        WithTTL(time.Minute).
        WithStaleTTL(5*time.Minute).
        KeepStaleOnError(),
)
```

## Options

Build options with the non-generic root builder `memoize.Opts()`.

| Option | Applies to | Meaning | Validation |
|---|---|---|---|
| `WithStore(store)` | Direct memoizers and explicit caches | Sets the backing `memoize.Store[K,V]`. Direct memoizers require `Store[uint64,V]`; explicit caches require a store matching `K,V`. | Store must implement the exact typed `Store[K,V]`; wrong or nil store returns `ErrInvalidStore`. Explicit caches also need a store unless `Bypass()` is set. |
| `WithTTL(ttl)` | Direct memoizers and explicit caches | Fresh duration for stored values. | Must be greater than zero when set, or `New` returns `ErrInvalidTTL`. |
| `WithStaleTTL(ttl)` | Direct memoizers and explicit caches | Additional stale-serving window after freshness expires. Enables stale-while-revalidate behavior. | Must not be negative and requires a positive TTL, or `New` returns `ErrInvalidStaleTTL`. |
| `KeepStaleOnError()` | Direct memoizers and explicit caches | If recompute fails while a stale entry exists, return the stale value instead of the recompute error. | Meaningful only with `WithTTL` plus `WithStaleTTL`; no separate validation. |
| `NoExpiration()` | Direct memoizers and explicit caches | Values remain fresh until overwritten, deleted, or cleared. | Satisfies the required expiration-policy validation. |
| `Bypass()` | Direct memoizers and explicit caches | Always computes and never stores. Useful for feature flags, tests, or temporarily disabling caching. | Satisfies expiration-policy validation and does not require a store. |
| `WithMetrics(metrics)` | Direct memoizers and explicit caches | Records cache events through `RecordMetric(MetricEvent)`. Nil is ignored. | No error; nil leaves metrics disabled. |
| `WithClock(clock)` | Direct memoizers and explicit caches | Injects a caller-owned clock, mainly for tests, shared clocks, or custom timing. The cache does not stop it. | Nil is ignored. |
| `WithTickerClock(interval)` | Direct memoizers and explicit caches | Creates a cache-owned ticker-backed clock at the given interval. `Cache.Stop` releases it. | Non-positive intervals are ignored. |
| `WithRefreshTimeout(timeout)` | Direct memoizers and explicit caches | Timeout used for background stale refresh work. | Non-positive values are ignored; default remains in effect. |

Every cache needs exactly one expiration strategy in practice: `WithTTL`, `NoExpiration`, or `Bypass`. `WithStaleTTL` extends a TTL policy; it is not a standalone expiration policy.

## Defaults

| Setting | Explicit `memoize.New[K,V]` | Direct `Memoize*` |
|---|---|---|
| Store | No default store. Operations that need storage return `ErrMissingStore` unless `Bypass()` is set. | Injects an internal unbounded `Store[uint64,V]` when `WithStore` is omitted. |
| Expiration policy | No default expiration policy. `New` returns `ErrMissingExpirationPolicy` unless `WithTTL`, `NoExpiration`, or `Bypass` is configured. | Same validation; direct memoizers still require `WithTTL`, `NoExpiration`, or `Bypass`. |
| Metrics | Disabled by default; an internal noop recorder is used. | Same. |
| Clock | `NewTickerClock(time.Millisecond)`. | Same. |
| Refresh timeout | `30 * time.Second`. | Same. |
| Concurrent miss coalescing | Enabled by an internal per-key flight map. | Same. |

Call `cache.Stop()` for explicit caches when you own the cache lifetime. It stops the default clock and clocks created by `WithTickerClock`; the caller remains responsible for clocks injected through `WithClock`. Direct memoizers own their internal cache; use an explicit cache if shutdown control is required.

## Errors

| Error | Meaning |
|---|---|
| `ErrMissingExpirationPolicy` | `WithTTL`, `NoExpiration`, and `Bypass` were all omitted. |
| `ErrInvalidTTL` | `WithTTL` was set to zero or a negative duration. |
| `ErrInvalidStaleTTL` | `WithStaleTTL` was negative or was used without a positive TTL. |
| `ErrMissingStore` | A cache operation required storage, but no store was configured. |
| `ErrInvalidStore` | `WithStore` received a nil store or a store whose key/value types do not match the cache. |

## Stores

`memoize.Store[K,V]` is the common storage interface used by memory, chain, local, Redis, and custom stores:

```go
type Store[K comparable, V any] interface {
    Get(ctx context.Context, key K) (memoize.Stored[V], bool, error)
    Set(ctx context.Context, key K, value memoize.Stored[V]) error
    Delete(ctx context.Context, key K) error
    Clear(ctx context.Context) error
}
```

`memoize.Stored[V]` is the envelope stores must persist and return:

| Field | Meaning |
|---|---|
| `Value` | Cached value. |
| `CreatedAt` | Write timestamp. |
| `FreshUntil` | Freshness deadline. |
| `StaleUntil` | Stale-serving deadline. |
| `NoExpire` | Entry is always fresh when true. |
| `Version` | Optional application version marker. |
| `Tags` | Optional invalidation tags. |

`memoize.TaggedStore[K,V]` extends `Store[K,V]` with `DeleteByTag(ctx, tag)`. Stores that support tags should remove entries whose `Stored[V].Tags` contains the tag.

Custom stores should not filter stale or expired entries inside `Get`. Return the raw stored envelope and let `Cache[K,V]` decide freshness.

Custom SQL store shape:

```go
type SQLStore[K comparable, V any] struct {
    db          *sql.DB
    encodeKey   func(K) string
    encodeEntry func(memoize.Stored[V]) ([]byte, error)
    decodeEntry func([]byte) (memoize.Stored[V], error)
}

func (s *SQLStore[K,V]) Get(ctx context.Context, key K) (memoize.Stored[V], bool, error) {
    row := s.db.QueryRowContext(ctx, `select entry from cache_entries where key = ?`, s.encodeKey(key))
    var data []byte
    if err := row.Scan(&data); errors.Is(err, sql.ErrNoRows) {
        var zero memoize.Stored[V]
        return zero, false, nil
    } else if err != nil {
        var zero memoize.Stored[V]
        return zero, false, err
    }
    entry, err := s.decodeEntry(data)
    return entry, err == nil, err
}

func (s *SQLStore[K,V]) Set(ctx context.Context, key K, entry memoize.Stored[V]) error {
    data, err := s.encodeEntry(entry)
    if err != nil {
        return err
    }
    _, err = s.db.ExecContext(ctx,
        `insert into cache_entries(key, entry) values(?, ?)
         on conflict(key) do update set entry = excluded.entry`,
        s.encodeKey(key), data,
    )
    return err
}

func (s *SQLStore[K,V]) Delete(ctx context.Context, key K) error {
    _, err := s.db.ExecContext(ctx, `delete from cache_entries where key = ?`, s.encodeKey(key))
    return err
}

func (s *SQLStore[K,V]) Clear(ctx context.Context) error {
    _, err := s.db.ExecContext(ctx, `delete from cache_entries`)
    return err
}
```

Object stores such as S3 can also implement `Store[K,V]`. Use the object key as the encoded cache key and the object body as the encoded `Stored[V]` envelope. S3 lifecycle policies can clean up old objects, but freshness still belongs to the cache engine through `FreshUntil`, `StaleUntil`, and `NoExpire`. `Clear` should delete only objects under the store prefix. For hot paths, place object storage behind a memory L1 with `stores/chain`.

## Memory Stores

Import:

```go
import "github.com/agkloop/go_memoize/stores/memory"
```

`memory.New[K,V](capacity, opts...)` creates an exact-LRU in-memory store for many bounded keys:

```go
store := memory.New[string, User](10_000)
```

`memory.NewSharded[K,V](capacity, opts...)` creates a sharded in-memory store for high concurrency across many different keys. Sharding improves distributed-key contention, not one-hot-key contention. Use it with supported primitive key types such as strings and integers; unsupported key types panic when the store chooses a shard:

```go
store := memory.NewSharded[string, User](100_000, memory.WithShards[string, User](32))
```

`memory.NewSingle[K,V]()` stores one logical value or one hot key with an atomic read path:

```go
store := memory.NewSingle[string, Config]()
```

Memory options:

| Option | Meaning |
|---|---|
| `memory.WithMaxBytes(n)` | Shallow byte budget; evicts LRU entries when exceeded. |
| `memory.WithGetRecencySample(n)` | Refreshes LRU recency every `n` direct-store or cache-engine fresh hits. `n <= 1` keeps exact LRU on every hit. |
| `memory.WithShards(n)` | Shard count for `NewSharded`; must be a positive power of two. |

Memory stores support `Get`, `Peek`, `Set`, `Delete`, `Clear`, `DeleteByTag`, `Len`, and `UsedBytes`. The cache engine updates recency for fresh hits and uses `Peek` for policy inspection without making stale or expired entries recent; user code should usually call `Cache` methods instead.

## Chain Store

Import:

```go
import "github.com/agkloop/go_memoize/stores/chain"
```

`chain.New[K,V](tiers ...memoize.Store[K,V])` creates ordered cache tiers. `Get` checks L1, then lower tiers, and backfills higher-priority tiers after a lower-tier hit. `Set`, `Delete`, and `Clear` propagate to every tier.

```go
l1 := memory.New[string, User](10_000)
l2 := local.New[User]("/var/cache/myapp/users")
store := chain.New[string, User](l1, l2)

cache, err := memoize.New[string, User](
    memoize.Opts().WithStore(store).WithTTL(time.Minute),
)
```

## Local Store

Import:

```go
import "github.com/agkloop/go_memoize/stores/local"
```

`local.New[V](dir)` creates a file-backed `memoize.Store[string,V]`. It maps string keys to SHA-256 filenames, writes atomically, and stores Gob-encoded `Stored[V]` entries. Values must be Gob-encodable.

```go
store := local.New[Report]("/var/cache/myapp/reports")
```

The local store returns the stored envelope as written; the cache engine decides freshness and staleness.

## Serializers

External stores that need byte encoding use `memoize.Serializer[V]`:

```go
type Serializer[V any] interface {
    Marshal(V) ([]byte, error)
    Unmarshal([]byte) (V, error)
}
```

Built-in serializer types:

| Type | Meaning |
|---|---|
| `serializers.JSON[V]` | JSON marshal/unmarshal. |
| `serializers.Gob[V]` | Gob marshal/unmarshal. |
| `serializers.Func[V]` | Custom marshal/unmarshal functions. |

Examples:

```go
jsonSerializer := serializers.JSON[User]{}
gobSerializer := serializers.Gob[User]{}

protoSerializer := serializers.Func[User]{
    MarshalFunc: func(user User) ([]byte, error) {
        return proto.Marshal(user.ToProto())
    },
    UnmarshalFunc: func(data []byte) (User, error) {
        var msg userpb.User
        if err := proto.Unmarshal(data, &msg); err != nil {
            return User{}, err
        }
        return UserFromProto(&msg), nil
    },
}
```

Serializers encode the value payload for stores such as Redis. Stores that persist the full envelope must also preserve `Stored[V]` metadata.

## Metrics

The root metrics interface is one method:

```go
type Metrics interface {
    RecordMetric(memoize.MetricEvent)
}
```

`memoize.MetricEvent` has `Kind memoize.MetricEventKind`, `Key string`, `Duration time.Duration`, and `Err error`. `MetricEventKind` values are `MetricHit`, `MetricMiss`, `MetricStaleHit`, `MetricRefreshStart`, `MetricRefreshSuccess`, `MetricRefreshError`, `MetricSet`, and `MetricDelete`.

Attach metrics with `WithMetrics`:

```go
m := metrics.NewInMemoryMetrics()

cache, err := memoize.New[string, User](
    memoize.Opts().WithStore(memory.New[string, User](1024)).WithTTL(time.Minute).WithMetrics(m),
)
```

`metrics.InMemoryMetrics` records events in process. `Stats()` returns `map[string]CacheStats` keyed by `MetricEvent.Key`, including hits, misses, stale hits, sets, deletes, refresh counts, hit rate, and refresh latency percentiles. `Reset()` clears counters.

`MetricEvent.Key` is usually the cache entry key. Treat it as high cardinality in production metrics exporters; aggregate, sample, hash, or bucket keys before turning them into labels.

Custom metrics implementation:

```go
type PromMetrics struct{}

func (PromMetrics) RecordMetric(event memoize.MetricEvent) {
    switch event.Kind {
    case memoize.MetricHit:
        cacheHits.Inc()
    case memoize.MetricRefreshSuccess:
        refreshLatency.Observe(event.Duration.Seconds())
    case memoize.MetricRefreshError:
        refreshErrors.Inc()
    }
}
```

## Background Values

Import:

```go
import "github.com/agkloop/go_memoize/background"
```

Use `background` for one value refreshed on a schedule and served by local atomic reads.

| API | Meaning |
|---|---|
| `background.Keep(ctx, fn, interval, opts...)` | Loads once, refreshes with `fn` on the interval, and returns `*background.Value[V]`. The initial load must succeed. |
| `background.MustKeep(ctx, fn, interval, opts...)` | Same as `Keep`, but panics on initial load failure. |
| `background.Mirror(ctx, key, store, interval, opts...)` | Reads one `Store[string,V]` key immediately, copies `entry.Value` into local process memory, and refreshes that local mirror on the interval. |
| `background.MustMirror(ctx, key, store, interval, opts...)` | Same as `Mirror`, but panics on initial read failure. Use it in startup paths when the remote snapshot must exist before serving. |
| `Value.Get()` | Returns the latest local value atomically. It does not call the remote store, does not block, and does not return an error. |

Options:

| Option | Meaning |
|---|---|
| `background.WriteThrough(key, store)` | Writes every successful `Keep` refresh to a shared `Store[string,V]`, useful for publishing snapshots to Redis or another shared store. |
| `background.OnError(fn)` | Observes refresh errors after the initial load; the previous value is kept. |
| `background.OnRefresh(fn)` | Observes each successful refresh value. |

High-level patterns:

```go
value, err := background.Keep(ctx, loadConfig, time.Minute)
cfg := value.Get()

publisher, err := background.Keep(ctx, loadConfig, time.Minute,
    background.WriteThrough[Config]("config:current", sharedStore),
)

mirror, err := background.Mirror(ctx, "config:current", sharedStore, time.Second)
```

`Mirror` and `MustMirror` are reader-side helpers. The first read must find the remote key. After that, request handlers should call `Value.Get()`; each call is an atomic local memory read of the last successfully mirrored value. The background goroutine is the only part that polls the shared store. Treat returned values as shared memory: keep them immutable, or copy maps and slices before mutating.

## Loader

Import:

```go
import "github.com/agkloop/go_memoize/loader"
```

`loader.New(fn, interval, opts...)` creates a fixed-interval background loader. `Value(ctx)` blocks until the first successful load or context cancellation, then returns instantly after readiness. Call `Stop()` to shut down the goroutine.

```go
l := loader.New(loadConfig, time.Minute, loader.WithOnError[Config](logError))
defer l.Stop()

cfg, err := l.Value(ctx)
```

The loader is readiness-oriented. `background.Keep` is producer-side snapshot refresh. `background.Mirror` is reader-side snapshot mirroring from an existing store key.

## Redis Adapter

The Redis adapter lives in the separate `adapters/redis` module:

```sh
go get github.com/agkloop/go_memoize/adapters/redis
```

```go
import redisstore "github.com/agkloop/go_memoize/adapters/redis"
```

Create a Redis store with a Redis universal client, a serializer, and optional prefix/key encoder:

```go
redisStore, err := redisstore.New[string, User](
    redisstore.WithClient[string, User](client),
    redisstore.WithPrefix[string, User]("users"),
    redisstore.WithSerializer[string, User](serializers.JSON[User]{}),
)

cache, err := memoize.New[string, User](
    memoize.Opts().WithStore(redisStore).WithTTL(time.Minute),
)
```

Adapter options:

| Option | Meaning |
|---|---|
| `WithClient` | Required Redis universal client. |
| `WithPrefix` | Optional key prefix. |
| `WithSerializer` | Required custom serializer for values, such as `serializers.JSON[V]`, `serializers.Gob[V]`, or `serializers.Func[V]`. |
| `WithKeyEncoder` | Optional typed-key to Redis-key encoder. Defaults to strings, decimal integer formatting, or `fmt.Sprint`. |

Use `K=uint64` for direct memoizer stores:

```go
redisStore, err := redisstore.New[uint64, Profile](
    redisstore.WithClient[uint64, Profile](client),
    redisstore.WithPrefix[uint64, Profile]("profiles"),
    redisstore.WithSerializer[uint64, Profile](serializers.JSON[Profile]{}),
)

cached, err := memoize.MemoizeCtx1E(loadProfile,
    memoize.Opts().WithStore(redisStore).WithTTL(time.Minute),
)
```

Use a custom key encoder when Redis keys need stable application formatting instead of default typed-key formatting:

```go
redisStore, err := redisstore.New[UserKey, User](
    redisstore.WithClient[UserKey, User](client),
    redisstore.WithPrefix[UserKey, User]("users"),
    redisstore.WithSerializer[UserKey, User](serializers.JSON[User]{}),
    redisstore.WithKeyEncoder[UserKey, User](func(key UserKey) string {
        return key.TenantID + ":" + strconv.FormatInt(key.UserID, 10)
    }),
)
```

Redis storage TTL is backend cleanup based on the later of `FreshUntil` and `StaleUntil`. It is not the public freshness policy; `Cache[K,V]` still decides whether a returned entry is fresh, stale, or expired.
