---
name: go-memoize-package
description: Work on the go_memoize Go package. Use when changing direct memoization, the root cache engine, stores, background refresh, loader snapshots, metrics, benchmarks, profiling docs, package docs, package README, or package examples.
license: Apache-2.0
compatibility: Requires Go and git. Run verification from the repository root.
---

# go_memoize Package

## Purpose

Use this skill to make correct package-level decisions in `go_memoize`. The root package is the only public module and contains direct function memoization plus the cache engine with stores, stale refresh, metrics, background values, and loaders.

## First Decision

Choose the API and store before editing:

| User Need | Use |
|---|---|
| Simple function memoization with comparable args | Root package `Memoize`, `Memoize1` ... `Memoize7`, or `E`/`Ctx` variants |
| Explicit cache key, TTL, stale refresh, metrics | `memoize.New[K,V]` and `memoize.Cache[K,V]` |
| One logical value or one hot key | `memory.NewSingle[K,V]()` or `background.Keep` |
| Many bounded in-memory keys | `memory.New[K,V](capacity)` |
| Many concurrent supported primitive keys | `memory.NewSharded[K,V](capacity)` |
| Two-tier cache | `chain.New[K,V](l1, l2)` |
| File-backed local cache | `local.New[V](dir)` |
| One writer publishes a full snapshot to many readers | Writer: `background.Keep` + `background.WriteThrough`; readers: `background.Mirror` or `background.MustMirror` |
| Readiness-gated periodic local value | `loader.New` |

## Public API Rules

- Root public module only: `github.com/agkloop/go_memoize`. Do not add public import examples using legacy module paths or helper packages.
- Direct memoizers use non-generic `memoize.Opts()` and return `(func, error)`.
- Direct memoizers hash comparable function arguments to `uint64`; custom direct stores must be `memoize.Store[uint64,V]`.
- Explicit caches use typed keys through `memoize.New[K,V]`; production examples usually use `K=string` for business keys.
- Error-returning direct memoizers use the root cache engine and support stale-on-error through `WithStaleTTL(...).KeepStaleOnError()`.
- `memoize.New[K,V]` has no default store and no default expiration policy. Choose `WithStore` plus `WithTTL`, `NoExpiration`, or `Bypass`.
- Direct memoizers create an internal unbounded `Store[uint64,V]` when `WithStore` is omitted, but still require `WithTTL`, `NoExpiration`, or `Bypass`.

## Cache Invariants

- `memory.New[K,V](capacity)` is exact LRU by default and stores key type `K` directly.
- `memory.NewSharded[K,V](capacity)` improves distributed-key concurrency for supported primitive key types. It does not reduce contention for one hot key; one key maps to one shard.
- `memory.NewSingle[K,V]()` is read-mostly and avoids LRU/hash overhead for one logical value.
- Stores persist raw `memoize.Stored[V]` envelopes. The cache engine owns fresh, stale, and expired decisions.
- Built-in stores may expose private fast paths used by the cache engine. Do not document those as public extension points.
- `WithGetRecencySample(n)` makes direct `Store.Get` recency approximate when `n > 1`.
- Metrics use one public method: `RecordMetric(memoize.MetricEvent)`.
- `background.Keep` and `loader.New` share internal periodic refresh-loop infrastructure. Cache stale refresh uses cache flight machinery, not the shared refresh loop.
- Source compatibility may change for performance or clarity; users can pin module versions.
- Keep core code standard-library-only unless the user explicitly approves a dependency.

## Background And Loader Semantics

- `background.Keep` is producer-side: it calls `fn` immediately, stores the value in local process memory, then refreshes on the interval.
- `background.WriteThrough(key, store)` publishes every successful `Keep` refresh to a shared `memoize.Store[string,V]` as a no-expiration snapshot.
- `background.Mirror` is reader-side: it reads one shared `Store[string,V]` key immediately, copies `entry.Value` into local process memory, then polls the store on the interval.
- `background.MustMirror` is `Mirror` for startup paths; it panics when the initial remote read fails.
- `background.Value.Get()` is an atomic local memory read. It does not call Redis, SQL, MySQL, S3, or any remote dependency; it does not block and does not return an error.
- Values returned by `Value.Get()` are shared memory. Keep them immutable, or copy maps, slices, and pointer-heavy fields before mutation.
- `loader.New` is readiness-oriented: callers use `Value(ctx)` to block until the first successful load, then read the latest loaded value after readiness.

## Docs Sync Rule

- When changing public docs, examples, README, API behavior, store behavior, background/loader semantics, metrics, benchmarks, or production recommendations, update this skill in the same change if future agents need to know the new rule.
- Run `scripts/check-docs-skill-sync.sh` before finishing docs-heavy work. The repo also includes `.githooks/pre-commit` for teams that opt in with `git config core.hooksPath .githooks`.

## Verification

Run checks from the repository root:

```sh
go test ./... -count=1
go test ./... -race -count=1
```

Benchmark cache/store work with an explicit command and record the result:

```sh
go test ./benchmarks/ -bench=. -benchmem -benchtime=1s -count=1
```

## Gotchas

- Do not optimize single-hot-key workloads by adding more LRU tuning; use `memory.NewSingle`.
- Do not claim sharding fixes single-key contention; one key maps to one shard.
- Do not document legacy module paths or helper-package imports in public examples.
- Do not update profiling docs without the exact benchmark command and observed numbers.
- Do not add behavior changes without tests first.
