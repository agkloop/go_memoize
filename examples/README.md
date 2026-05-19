# Production Examples

These examples are working, tested packages that show production-shaped service types, constructor validation, cache shutdown, context handling, and store selection. They are not toy `main` programs.

The examples use Beeceptor's public sample API shapes for realistic data:

- Users: `https://fake-json-api.mock.beeceptor.com/users`
- Companies: `https://fake-json-api.mock.beeceptor.com/companies`

| Directory | Use Case |
|---|---|
| `http_user_cache` | Explicit string-key HTTP/service cache with bounded memory, stale refresh, metrics, and shutdown. |
| `direct_profile_cache` | Direct function memoization with the default direct store for a context-aware repository method. |
| `direct_stale_profile_cache` | Direct function memoization with stale-while-revalidate and stale-on-error for a context-aware repository method. |
| `config_snapshot` | One scheduled configuration snapshot with local atomic reads and refresh error hooks. |

Redis examples live under `adapters/redis/examples` because the Redis adapter is a separate Go module.

Run each root example on its own:

```sh
go test ./examples/http_user_cache -count=1
go test ./examples/direct_profile_cache -count=1
go test ./examples/direct_stale_profile_cache -count=1
go test ./examples/config_snapshot -count=1
```

The tests use local `httptest` servers with Beeceptor-compatible payloads so they verify caching behavior without depending on network availability.
