# Direct Stale Profile Cache

This example wraps a Beeceptor-shaped profile repository with direct function memoization, stale-while-revalidate, and stale-on-error behavior.

The service keeps the memoize call front and center:

```go
cached, err := memoize.MemoizeCtx1E(
	repo.LoadProfile,
	memoize.Opts().
		WithTTL(time.Minute).
		WithStaleTTL(5*time.Minute).
		KeepStaleOnError(),
)
```

Because this is a direct memoizer, `go_memoize` hashes the `int64` profile ID into an internal `uint64` key and uses the default internal `Store[uint64, Profile]`. Use `WithStore` only when you need to supply a custom direct store tier.

`WithTTL(time.Minute)` marks a profile fresh for one minute. `WithStaleTTL(5*time.Minute)` keeps the old value available while a refresh is attempted. `KeepStaleOnError()` lets the service continue returning the stale profile if the upstream Beeceptor-style `/users` request fails during the stale window.

Run the example test with:

```sh
go test ./examples/direct_stale_profile_cache -count=1
```
