# Agent Skills

Use `.agents/skills/go-memoize-package/SKILL.md` when working on this repository.
Use `.agents/skills/go-performance-optimization/SKILL.md` when optimizing Go hot paths, benchmarks, allocation profiles, pprof output, or escape analysis.

Key reminders:

- Root package is the only public module and contains both direct memoization and the cache engine.
- Do not reintroduce `/v2` or `helpers` imports; users can pin module versions.
- Run verification from the repository root: `go test ./... -count=1` and `go test ./... -race -count=1`.
- Pick the memory store by workload: `memory.New` for many-key LRU, `memory.NewSharded` for distributed-key concurrency, `memory.NewSingle` for one logical value.
