# Migration Guide

The package now exposes one public root module: `github.com/agkloop/go_memoize`.

Users can pin module versions with Go modules, so the repository no longer carries a parallel module path or a helper package for memoization wrappers.

## Import Paths

Use the root module for cache-engine imports:

```go
import memoize "github.com/agkloop/go_memoize"
```

Use root subpackages directly:

```go
import "github.com/agkloop/go_memoize/stores/memory"
```

## Direct Function Memoization

For simple function memoization, use root direct memoizers with `Opts()`:

```go
cached, err := memoize.Memoize1(func(id int) User {
    return loadUser(id)
}, memoize.Opts().WithTTL(time.Minute))
if err != nil {
    return err
}
```

For functions that can fail, use `E` variants:

```go
cached, err := memoize.MemoizeCtx1E(func(ctx context.Context, id int) (User, error) {
    return loadUser(ctx, id)
}, memoize.Opts().WithTTL(time.Minute))
if err != nil {
    return err
}
```

## Helper Package Removal

The old helper-wrapper style should be replaced. If you previously used a helper function with an explicit cache and key function, choose one of these replacements.

### Replacement 1: Direct Memoization

Use this when comparable arguments are enough and you do not need a custom store or explicit business key.

```go
getUser, err := memoize.MemoizeCtx1E(func(ctx context.Context, id int) (User, error) {
    return loadUser(ctx, id)
}, memoize.Opts().WithTTL(time.Minute))
if err != nil {
    return err
}
```

### Replacement 2: Explicit Cache Key

Use this when the key matters, the cache store matters, or you need metrics/stale refresh.

```go
user, err := cache.GetOrCompute(ctx, fmt.Sprintf("user:%d", id), func(ctx context.Context) (User, error) {
    return loadUser(ctx, id)
})
```

## Store Migration

| Package | Import |
|---|---|
| Memory store | `github.com/agkloop/go_memoize/stores/memory` |
| Chain store | `github.com/agkloop/go_memoize/stores/chain` |
| Local store | `github.com/agkloop/go_memoize/stores/local` |
| Background values | `github.com/agkloop/go_memoize/background` |
| Metrics Adapter | `github.com/agkloop/go_memoize/metrics` |
| Serializers | `github.com/agkloop/go_memoize/serializers` |
| Loader | `github.com/agkloop/go_memoize/loader` |

## Verification

After migration, run:

```sh
go test ./... -count=1
go test ./... -race -count=1
```

If you use the Redis adapter, run its module tests too:

```sh
cd adapters/redis
go test ./... -count=1
go test ./... -race -count=1
```
