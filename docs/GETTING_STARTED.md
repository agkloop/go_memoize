# Getting Started

## Install

Install the root module:

```sh
go get github.com/agkloop/go_memoize
```

Import the root package and any stores you need:

```go
import (
	"context"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/memory"
)
```

## Direct Memoization

Use direct memoization when you want to wrap a function with comparable arguments.
The memoizer hashes the arguments and manages cache lookups for you.

```go
type User struct {
	ID   int64
	Name string
}

func loadUser(id int64) User {
	return User{ID: id, Name: "Ada"}
}

cachedLoadUser, err := memoize.Memoize1(loadUser, memoize.Opts().WithTTL(time.Minute))
if err != nil {
	return err
}

user := cachedLoadUser(42)
```

## Direct Memoization With Errors And Context

Use `Ctx` and `E` variants when the source function accepts a `context.Context` or returns an error.
Only successful results are cached.

```go
func loadUser(ctx context.Context, id int64) (User, error) {
	return User{ID: id, Name: "Ada"}, nil
}

cachedLoadUser, err := memoize.MemoizeCtx1E(loadUser, memoize.Opts().WithTTL(30*time.Second))
if err != nil {
	return err
}

user, err := cachedLoadUser(ctx, 42)
if err != nil {
	return err
}
```

## Direct Stale-On-Error Memoization

Use stale-on-error when a previously cached value is better than failing the request.
`WithTTL` controls the fresh window, `WithStaleTTL` controls how long stale values remain usable, and `KeepStaleOnError` returns stale data if recomputation fails.

```go
func loadUser(ctx context.Context, id int64) (User, error) {
	return User{ID: id, Name: "Ada"}, nil
}

cachedLoadUser, err := memoize.MemoizeCtx1E(
	loadUser,
	memoize.Opts().
		WithTTL(time.Minute).
		WithStaleTTL(5*time.Minute).
		KeepStaleOnError(),
)
if err != nil {
	return err
}

user, err := cachedLoadUser(ctx, 42)
if err != nil {
	return err
}
```

## Explicit Cache With Business Keys

Use an explicit cache when you already have stable business keys, need a bounded store, or want direct lifecycle control.

```go
type User struct {
	ID   string
	Name string
}

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
	return User{ID: "42", Name: "Ada"}, nil
})
if err != nil {
	return err
}
```

## Next Steps

Read `docs/CONCEPTS.md` for the model behind keys, stores, freshness, errors, and shutdown.
Read `docs/API.md` for API details and `docs/PRODUCTION.md` for production guidance.
