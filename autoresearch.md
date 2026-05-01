# Autoresearch: Optimize Resolve cold latest benchmark

## Objective
Optimize `Registry.Resolve`/internal `resolve` performance for the current representative workload: many discrete feature registrations across a wide version range, mostly querying a very new/latest version that supports most features, while some old features are removed via `RegisterRange`.

The benchmark intentionally calls unexported `resolve` so each iteration measures cold computation rather than exported `Resolve` cache hits.

## Metrics
- **Primary**: `resolve_10000_ns` (ns/op, lower is better) — median benchmark time for 10,000 registered features.
- **Secondary**: `resolve_1000_ns`, `resolve_100_ns`, `alloc_10000_b`, `alloc_10000_count` — watch for regressions and allocation tradeoffs.

## How to Run
`./autoresearch.sh` — builds the package test binary, runs `BenchmarkRegistryResolveColdLatest` multiple times, and emits `METRIC name=value` lines.

## Files in Scope
- `registry.go` — registry storage, freeze/cache/resolve implementation, future index structures.
- `featureset.go` — FeatureSet representation; may change to shared immutable backing if API behavior stays compatible.
- `registry_benchmark_test.go` — representative benchmark. Do not weaken or special-case it.
- `registry_test.go`, `example_test.go` — tests may be extended for correctness coverage.
- `autoresearch.md`, `autoresearch.sh`, `autoresearch.checks.sh`, `autoresearch.ideas.md` — experiment harness and notes.

## Off Limits
- Do not remove `RegisterRange` semantics or change public API behavior.
- Do not special-case benchmark feature names/counts/versions.
- Do not bypass semver correctness for public inputs.
- Do not rely on exported `Resolve` cache hits for the primary metric.

## Constraints
- `go test ./...` must pass.
- No new third-party dependencies unless clearly justified.
- Preserve `[since, until)` semantics, including removals at the exact `until` version.
- Keep code maintainable; primary metric is king, but avoid benchmark-only hacks.

## What's Been Tried
- Baseline before optimization: current implementation parses version then linearly scans all registered entries, constructing a fresh `map[F]struct{}` for every cold resolve.
- Planned first direction: build frozen version breakpoint/index data in `Freeze()`, add a latest-version fast path, and return shared immutable FeatureSet backing where possible.
