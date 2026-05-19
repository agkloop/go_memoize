#!/usr/bin/env bash
set -euo pipefail

# Run all v2 benchmarks with profiling enabled.
# Profiles land in v2/benchmarks/cpu.prof and v2/benchmarks/mem.prof.
# Usage: ./scripts/profile.sh [bench-regex]

BENCH="${1:-.}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BENCH_DIR="$SCRIPT_DIR/../v2/benchmarks"
PPROF_PID=""
PPROF_MEM_PID=""

cleanup() {
  if [[ -n "$PPROF_PID" ]]; then
    kill "$PPROF_PID" 2>/dev/null || true
  fi
  if [[ -n "$PPROF_MEM_PID" ]]; then
    kill "$PPROF_MEM_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

echo "==> Running benchmarks matching: $BENCH"
cd "$BENCH_DIR"
BENCH_PROFILE=1 go test . \
  -bench="$BENCH" \
  -benchmem \
  -benchtime=10s \
  -count=1 \
  -timeout=300s

echo ""
echo "==> Profiles written:"
echo "    $BENCH_DIR/cpu.prof"
echo "    $BENCH_DIR/mem.prof"
echo ""
echo "==> Opening CPU profile (ctrl-C to exit):"
go tool pprof -http=:6060 "$BENCH_DIR/cpu.prof" &
PPROF_PID=$!

echo "==> Opening MEM profile on :6061 (ctrl-C to exit):"
go tool pprof -http=:6061 "$BENCH_DIR/mem.prof" &
PPROF_MEM_PID=$!

echo ""
echo "CPU flamegraph: http://localhost:6060/ui/flamegraph"
echo "MEM flamegraph: http://localhost:6061/ui/flamegraph"
echo ""
echo "Press ENTER to stop both pprof servers."
read -r || true
