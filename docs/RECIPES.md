# Recipes

Copy-paste starting points for common `github.com/agkloop/go_memoize` setups. Direct memoizers hash comparable arguments to `uint64`; explicit caches usually use business keys such as strings.

## Direct TTL Memoizer

Use a direct memoizer when the function arguments are the cache key.

```go
package profiles

import (
	"time"

	memoize "github.com/agkloop/go_memoize"
)

func NewCachedProfileLoader(loadProfile func(int64) Profile) (func(int64) Profile, error) {
	return memoize.Memoize1(loadProfile, memoize.Opts().WithTTL(time.Minute))
}

profile := cached(42)
```

## Direct Context/Error Memoizer

Use the `Ctx` and `E` variants for context-aware functions that can fail. Errors are not cached.

```go
cached, err := memoize.MemoizeCtx1E(loadProfile, memoize.Opts().WithTTL(30*time.Second))
if err != nil {
	return err
}

profile, err := cached(ctx, 42)
if err != nil {
	return err
}
```

## Direct Stale-On-Error Memoizer

Use stale-on-error when a stale value is better than failing the request. This uses the default internal direct memoizer store keyed by hashed args as `uint64`.

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

profile, err := cached(ctx, 42)
if err != nil {
	return err
}
```

## Direct Memoizer With Custom Store

Direct memoizers accept a `memoize.Store[uint64,V]` because direct keys are hashed arguments.

```go
store := memory.New[uint64, Profile](10_000)

cached, err := memoize.MemoizeCtx1E(loadProfile,
	memoize.Opts().WithStore(store).WithTTL(time.Minute),
)
if err != nil {
	return err
}

profile, err := cached(ctx, 42)
```

The store must implement the root interface:

```go
type Store[K comparable, V any] interface {
	Get(ctx context.Context, key K) (memoize.Stored[V], bool, error)
	Set(ctx context.Context, key K, value memoize.Stored[V]) error
	Delete(ctx context.Context, key K) error
	Clear(ctx context.Context) error
}
```

## Explicit Cache With String Keys

Use an explicit cache when the key is already part of your domain.

```go
store := memory.New[string, Profile](10_000)

cache, err := memoize.New[string, Profile](
	memoize.Opts().WithStore(store).WithTTL(time.Minute),
)
if err != nil {
	return err
}
defer cache.Stop()

profile, err := cache.GetOrCompute(ctx, "profile:42", func(ctx context.Context) (Profile, error) {
	return repo.LoadProfile(ctx, 42)
})
```

## Explicit Stale-While-Revalidate Cache

`WithStaleTTL` returns stale values while one goroutine refreshes the key in the background.

```go
cache, err := memoize.New[string, Profile](
	memoize.Opts().
		WithStore(memory.New[string, Profile](10_000)).
		WithTTL(time.Minute).
		WithStaleTTL(5*time.Minute).
		KeepStaleOnError(),
)
if err != nil {
	return err
}
defer cache.Stop()

profile, err := cache.GetOrCompute(ctx, "profile:42", func(ctx context.Context) (Profile, error) {
	return repo.LoadProfile(ctx, 42)
})
```

## Two-Tier Memory And Local Cache

Chain a fast per-process memory L1 over a local file L2 for warm restarts on one machine.

```go
l1 := memory.New[string, Profile](10_000)
l2 := local.New[Profile]("/var/cache/myapp/profiles")
store := chain.New[string, Profile](l1, l2)

cache, err := memoize.New[string, Profile](
	memoize.Opts().WithStore(store).WithTTL(5*time.Minute),
)
if err != nil {
	return err
}
defer cache.Stop()
```

## Two-Tier Memory And Redis Cache

Chain a per-process memory L1 over Redis when multiple processes need a shared L2.

```go
redisStore, err := redisstore.New[string, Profile](
	redisstore.WithClient[string, Profile](redisClient),
	redisstore.WithPrefix[string, Profile]("profiles"),
	redisstore.WithSerializer[string, Profile](serializers.JSON[Profile]{}),
)
if err != nil {
	return err
}

store := chain.New[string, Profile](
	memory.New[string, Profile](10_000),
	redisStore,
)

cache, err := memoize.New[string, Profile](
	memoize.Opts().WithStore(store).WithTTL(time.Minute).WithStaleTTL(5*time.Minute),
)
if err != nil {
	return err
}
defer cache.Stop()
```

## Single Writer, Many Reader Category Snapshot

Kubernetes pattern: one cache-refresher pod reads MySQL and many API pods read a shared store. `memory.New`, `memory.NewSharded`, and `memory.NewSingle` are per-process only; they do not share values between pods. The shared store must be Redis, SQL, S3/object storage, or another distributed store.

Writer pod:

```go
categories, err := background.Keep(ctx,
	func(ctx context.Context) (CategorySnapshot, error) {
		return mysqlRepo.LoadAllCategories(ctx)
	},
	30*time.Second,
	background.WriteThrough("categories:all", redisStore),
	background.OnError(func(err error) {
		logger.Error("category refresh failed", "err", err)
	}),
)
if err != nil {
	return err
}

_ = categories.Get()
```

API reader pods:

```go
categories := background.MustMirror(ctx,
	"categories:all",
	redisStore,
	5*time.Second,
	background.OnError(func(err error) {
		logger.Error("category mirror failed", "err", err)
	}),
)

func handleCategories(w http.ResponseWriter, r *http.Request) {
	snapshot := categories.Get()
	writeJSON(w, snapshot)
}
```

`background.MustMirror` performs the first remote read during startup and panics if the shared key is missing. Once startup succeeds, every API pod holds its own local in-memory copy. `categories.Get()` is an atomic local memory read of that copy; it does not call Redis, does not hit MySQL, does not block, and does not return an error. The mirror goroutine is the only code that polls the shared store every `5*time.Second`.

Treat mirrored values as immutable shared memory. If `CategorySnapshot` contains maps, slices, or pointers and a handler needs to mutate them, copy before mutating.

`WriteThrough` stores `memoize.Stored[V]{NoExpire: true}`, so the last good snapshot remains until overwritten or deleted. Use this for category trees, config snapshots, and other read-mostly data where last known good is acceptable.

Stricter freshness variant: when the shared store must carry `WithTTL` or `WithStaleTTL` metadata instead of `NoExpire`, write through an explicit cache in the writer and read through the same cache policy in every API replica.

```go
writerCache, err := memoize.New[string, CategorySnapshot](
	memoize.Opts().WithStore(redisStore).WithTTL(time.Minute).WithStaleTTL(10*time.Minute),
)
if err != nil {
	return err
}
defer writerCache.Stop()

_, err = background.Keep(ctx,
	func(ctx context.Context) (CategorySnapshot, error) {
		snapshot, err := mysqlRepo.LoadAllCategories(ctx)
		if err != nil {
			return CategorySnapshot{}, err
		}
		return snapshot, writerCache.Set(ctx, "categories:all", snapshot)
	},
	30*time.Second,
	background.OnError(func(err error) {
		logger.Error("category refresh failed", "err", err)
	}),
)
```

```go
readerCache, err := memoize.New[string, CategorySnapshot](
	memoize.Opts().WithStore(redisStore).WithTTL(time.Minute).WithStaleTTL(10*time.Minute),
)
if err != nil {
	return err
}
defer readerCache.Stop()

func handleCategories(w http.ResponseWriter, r *http.Request) {
	snapshot, ok, err := readerCache.Get(r.Context(), "categories:all")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "categories unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, snapshot)
}
```

## One-Value Config Snapshot With background.Keep

Use `background.Keep` for one logical value that should refresh outside request paths.

```go
config, err := background.Keep(ctx,
	func(ctx context.Context) (AppConfig, error) {
		return configAPI.Load(ctx)
	},
	time.Minute,
	background.OnError(func(err error) {
		logger.Warn("config refresh failed", "err", err)
	}),
)
if err != nil {
	return err
}

func handler(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	_ = cfg
}
```

## Readiness-Gated Loader

Use `loader.Loader` when startup readiness must wait for the first successful load.

```go
categories := loader.New(
	func(ctx context.Context) (CategorySnapshot, error) {
		return repo.LoadAllCategories(ctx)
	},
	30*time.Second,
	loader.WithOnError(func(err error) {
		logger.Error("category load failed", "err", err)
	}),
)
defer categories.Stop()

func readiness(w http.ResponseWriter, r *http.Request) {
	if _, err := categories.Value(r.Context()); err != nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

## No Expiration Cache

Use `NoExpiration` for values invalidated manually.

```go
cache, err := memoize.New[string, FeatureFlags](
	memoize.Opts().WithStore(memory.New[string, FeatureFlags](100)).NoExpiration(),
)
if err != nil {
	return err
}
defer cache.Stop()

if err := cache.Set(ctx, "tenant:acme", flags); err != nil {
	return err
}

flags, ok, err := cache.Get(ctx, "tenant:acme")
```

## Bypass Mode

Use `Bypass` to disable storage without changing call sites.

```go
cache, err := memoize.New[string, Profile](memoize.Opts().Bypass())
if err != nil {
	return err
}
defer cache.Stop()

profile, err := cache.GetOrCompute(ctx, "profile:42", func(ctx context.Context) (Profile, error) {
	return repo.LoadProfile(ctx, 42)
})
```

## Custom Store

Implement `memoize.Store[K,V]` when you need a storage backend not provided by the package.

```go
type Store[K comparable, V any] interface {
	Get(ctx context.Context, key K) (memoize.Stored[V], bool, error)
	Set(ctx context.Context, key K, value memoize.Stored[V]) error
	Delete(ctx context.Context, key K) error
	Clear(ctx context.Context) error
}
```

Minimal wrapper shape:

```go
type SQLStore[V any] struct {
	db *sql.DB
}

func (s *SQLStore[V]) Get(ctx context.Context, key string) (memoize.Stored[V], bool, error) {
	// Load and decode the full memoize.Stored[V] envelope.
	return memoize.Stored[V]{}, false, nil
}

func (s *SQLStore[V]) Set(ctx context.Context, key string, value memoize.Stored[V]) error {
	// Encode and persist value, including FreshUntil, StaleUntil, and NoExpire.
	return nil
}

func (s *SQLStore[V]) Delete(ctx context.Context, key string) error { return nil }
func (s *SQLStore[V]) Clear(ctx context.Context) error { return nil }
```

## S3/Object Store As Durable L2

Use an object store as a durable L2 behind memory when latency is acceptable and persistence matters more than hot-path speed.

```go
type ObjectStore[V any] struct {
	bucket string
	codec  memoize.Serializer[memoize.Stored[V]]
}

func (s *ObjectStore[V]) Get(ctx context.Context, key string) (memoize.Stored[V], bool, error) {
	data, err := getObject(ctx, s.bucket, key)
	if errors.Is(err, ErrNotFound) {
		return memoize.Stored[V]{}, false, nil
	}
	if err != nil {
		return memoize.Stored[V]{}, false, err
	}
	entry, err := s.codec.Unmarshal(data)
	return entry, err == nil, err
}

func (s *ObjectStore[V]) Set(ctx context.Context, key string, value memoize.Stored[V]) error {
	data, err := s.codec.Marshal(value)
	if err != nil {
		return err
	}
	return putObject(ctx, s.bucket, key, data)
}

func (s *ObjectStore[V]) Delete(ctx context.Context, key string) error { return deleteObject(ctx, s.bucket, key) }
func (s *ObjectStore[V]) Clear(ctx context.Context) error { return nil }

store := chain.New[string, Profile](
	memory.New[string, Profile](10_000),
	&ObjectStore[Profile]{bucket: "profile-cache", codec: serializers.JSON[memoize.Stored[Profile]]{}},
)
```

## Custom Serializer

Use `serializers.Func[V]` to adapt an existing codec.

```go
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

serializer := serializers.Func[User]{
	MarshalFunc: func(user User) ([]byte, error) {
		return json.Marshal(user)
	},
	UnmarshalFunc: func(data []byte) (User, error) {
		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			return User{}, err
		}
		return user, nil
	},
}

redisStore, err := redisstore.New[string, User](
	redisstore.WithClient[string, User](redisClient),
	redisstore.WithSerializer[string, User](serializer),
)
```

## Metrics Exporter

Implement `memoize.Metrics` to export cache events to your metrics system.

```go
type PromMetrics struct {
	hits   *prometheus.CounterVec
	misses *prometheus.CounterVec
}

func (m *PromMetrics) RecordMetric(event memoize.MetricEvent) {
	switch event.Kind {
	case memoize.MetricHit:
		m.hits.WithLabelValues(event.Key).Inc()
	case memoize.MetricMiss:
		m.misses.WithLabelValues(event.Key).Inc()
	case memoize.MetricRefreshError:
		logger.Error("cache refresh failed", "key", event.Key, "err", event.Err)
	}
}

cache, err := memoize.New[string, Profile](
	memoize.Opts().
		WithStore(memory.New[string, Profile](10_000)).
		WithTTL(time.Minute).
		WithMetrics(promMetrics),
)
```
