# Config Snapshot

Working one-value snapshot example for configuration derived from Beeceptor's sample company API shape: `GET https://fake-json-api.mock.beeceptor.com/companies`.

- Uses `background.Keep`, not an LRU cache, because every request reads the same logical value.
- Blocks startup until the first successful load.
- Keeps serving the last successful value when later refreshes fail.
- Copies maps on load and read so callers do not mutate shared memory returned by `background.Value.Get()`.
- Keeps the `background.Keep` package usage in `config_service.go`.
- Keeps Beeceptor HTTP and JSON parsing in `beeceptor_source.go`.
- Exposes `Close()` to cancel the background refresh loop during shutdown.

Test it directly:

```sh
go test ./examples/config_snapshot -count=1
```
