---
name: go-performance-optimization
description: Use when optimizing Go hot paths, reducing allocations, investigating slow benchmarks, reading pprof output, checking escape analysis, or applying goperf.dev/go-optimization-guide patterns to Go code.
license: Apache-2.0
compatibility: Requires Go tooling, benchmarks, pprof, and git.
---

# Go Performance Optimization

## Overview

Optimize from measurements, not folklore. Apply `goperf.dev` patterns only when they fit the observed bottleneck and keep the code simpler or measurably faster.

## Workflow

1. Establish a focused benchmark with `go test -bench ... -benchmem -count=1`.
2. Profile before changing code when the cause is not obvious: add `-cpuprofile` and `-memprofile`, then inspect with `go tool pprof -top`.
3. Use escape analysis for allocation questions: `go test -gcflags=-m=2 ./path`.
4. Rank hypotheses by expected impact and risk.
5. Change one variable at a time.
6. Re-run the same benchmark and compare `ns/op`, `B/op`, and `allocs/op`.
7. Run the package tests, then the repository verification command.

## Applicable Patterns

| Symptom | Prefer | Avoid |
|---|---|---|
| Hot-path allocations | Remove closure/interface/string formatting escapes | `sync.Pool` by default |
| Growing slices/maps | Preallocate with known or bounded capacity | Letting hot buffers resize repeatedly |
| Interface boxing in loops | Generics or concrete types | `any`, `interface{}`, `fmt` conversions on hot paths |
| Large dense structs | Field alignment and locality checks | Reordering public structs without compatibility review |
| Lock contention | Sharding, read-mostly stores, atomics for simple counters | Atomics for multi-step invariants |
| Context overhead | Keep context at API boundaries and miss/IO paths | Creating timeout contexts on hot hits |
| GC pressure | Fewer heap objects and shorter object lifetimes | Pooling tiny or long-lived objects |

## Decision Rules

- Keep benchmarks representative: one-hot-key, distributed-key, hit, miss, stale, and parallel workloads have different bottlenecks.
- Do not optimize exact LRU by replacing its mutex with atomics; linked-list/index/map updates are one invariant.
- Use `sync.Pool` only for reusable temporary objects that allocate heavily, such as buffers or encoders.
- Remove `fmt.Sprint`, `fmt.Sprintf`, and `any` conversions from hot paths when a typed alternative exists.
- Keep error handling and singleflight off hot-hit paths when behavior allows a separate fast path.
- Prefer shallow, explicit APIs over hidden conversions.

## Verification

Run the benchmark that motivated the change and record before/after numbers:

```sh
go test ./benchmarks/ -bench='BenchmarkName$' -benchmem -benchtime=1s -count=1 -run '^$'
```

Run repository checks from the root:

```sh
git diff --check
go test ./... -count=1
go test ./... -race -count=1
```

## Common Mistakes

- Claiming performance improved without fresh benchmark output.
- Optimizing a benchmark-only workload while slowing the production path.
- Adding `sync.Pool` when the real allocation is closure escape or interface boxing.
- Using `fmt` for generic keys in cache/store hot paths.
- Treating sharding as a fix for one hot key; one key still maps to one shard.
- Ignoring `B/op` and `allocs/op` because `ns/op` moved slightly.
