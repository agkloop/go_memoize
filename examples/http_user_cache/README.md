# HTTP User Cache

Working explicit cache example for users loaded from Beeceptor's sample user API shape: `GET https://fake-json-api.mock.beeceptor.com/users`.

- Uses user IDs as explicit cache keys.
- Uses bounded `memory.New[int64, User]` as LRU storage.
- Configures `WithTTL`, `WithStaleTTL`, `KeepStaleOnError`, `WithRefreshTimeout`, and optional metrics.
- Keeps the memoize package usage in `user_service.go`.
- Keeps Beeceptor HTTP and JSON parsing in `beeceptor_repository.go`.
- Exposes `Close()` so services can call `cache.Stop()` during shutdown.

Test it directly:

```sh
go test ./examples/http_user_cache -count=1
```
