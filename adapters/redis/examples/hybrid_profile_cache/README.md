# Hybrid Profile Cache

Working direct memoizer example using a two-tier store in the Redis adapter module. Profiles are derived from Beeceptor's sample user API shape: `GET https://fake-json-api.mock.beeceptor.com/users`.

- Uses `MemoizeCtx1E` for repository calls.
- Uses `Store[uint64, Profile]` because direct memoizers hash comparable arguments to `uint64` keys.
- Uses `memory.New[uint64, Profile]` as L1 and Redis as L2 through `chain.New`.
- Relies on the Redis default key encoder: direct hash keys are formatted as decimal strings after the prefix, for example `profiles:123456789`.
- Configures TTL, stale TTL, and `KeepStaleOnError` for production outage tolerance.
- Read `profile_service.go` first for memoize, chain, memory, and Redis adapter usage.
- Beeceptor HTTP and JSON parsing live in `beeceptor_repository.go`.

Test it directly from the adapter module:

```sh
cd adapters/redis
go test ./examples/hybrid_profile_cache -count=1
```

The unit test uses a memory-backed L1/L2 chain to verify the same direct memoizer and chain-store caching behavior without requiring a Redis server. The `NewProfileService` constructor shows the production Redis wiring.
