#!/bin/bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

go test -c -o "$tmpdir/semverflags.test" . >/dev/null

out="$($tmpdir/semverflags.test \
  -test.run '^$' \
  -test.bench '^BenchmarkRegistryResolveColdLatest$' \
  -test.benchmem \
  -test.benchtime=200ms \
  -test.count=5)"

printf '%s\n' "$out" >&2

BENCH_OUTPUT="$out" python3 - <<'PY'
import os, re, statistics, sys
text = os.environ["BENCH_OUTPUT"]
values = {}
alloc_b = {}
alloc_count = {}
pattern = re.compile(r"BenchmarkRegistryResolveColdLatest/features=(\d+)-\d+\s+\d+\s+([0-9.]+)\s+ns/op\s+([0-9.]+)\s+B/op\s+([0-9.]+)\s+allocs/op")
for line in text.splitlines():
    m = pattern.search(line)
    if not m:
        continue
    n = int(m.group(1))
    values.setdefault(n, []).append(float(m.group(2)))
    alloc_b.setdefault(n, []).append(float(m.group(3)))
    alloc_count.setdefault(n, []).append(float(m.group(4)))

required = [100, 1000, 10000]
missing = [n for n in required if n not in values]
if missing:
    print(f"missing benchmark results for {missing}", file=sys.stderr)
    sys.exit(1)

for n in required:
    print(f"METRIC resolve_{n}_ns={statistics.median(values[n]):.0f}")
print(f"METRIC alloc_10000_b={statistics.median(alloc_b[10000]):.0f}")
print(f"METRIC alloc_10000_count={statistics.median(alloc_count[10000]):.0f}")
PY
