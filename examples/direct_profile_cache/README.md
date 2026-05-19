# Direct Profile Cache

Working direct memoizer example for profiles derived from Beeceptor's sample user API shape: `GET https://fake-json-api.mock.beeceptor.com/users`.

- Uses `MemoizeCtx1E` for `LoadProfile(ctx, profileID int64)`.
- Omits `WithStore`, so direct memoization uses the default internal `Store[uint64, Profile]`.
- Still requires `WithTTL`; direct memoizers do not silently choose an expiration policy.
- Error-returning memoizers cache successful results only.
- Use this style for simple in-process function memoization when you do not need explicit cache keys or manual invalidation.
- Read `profile_service.go` first for the memoize package usage.
- Beeceptor HTTP and JSON parsing live in `beeceptor_repository.go`.

Test it directly:

```sh
go test ./examples/direct_profile_cache -count=1
```
