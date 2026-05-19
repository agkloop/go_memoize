#!/usr/bin/env sh
set -eu

root=$(git rev-parse --show-toplevel)
cd "$root"

require_optional=${REQUIRE_OPTIONAL_TOOLS:-0}

log() {
  printf '\n==> %s\n' "$1"
}

run_optional() {
  tool=$1
  shift
  if command -v "$tool" >/dev/null 2>&1; then
    "$tool" "$@"
    return
  fi

  if [ "$require_optional" = "1" ]; then
    printf '%s\n' "Required tool not found on PATH: $tool" >&2
    exit 1
  fi

  printf '%s\n' "Skipping optional check; $tool is not installed." >&2
}

check_tidy() {
  module_dir=$1
  mod_file=$module_dir/go.mod
  sum_file=$module_dir/go.sum

  log "go mod tidy check: $module_dir"
  (
    cd "$module_dir"
    go mod tidy
  )
  git diff --exit-code -- "$mod_file" "$sum_file"
}

go_files=$(
  for file in $(git ls-files '*.go'); do
    if [ -f "$file" ]; then
      printf '%s\n' "$file"
    fi
  done
)

log "docs/skill sync"
scripts/check-docs-skill-sync.sh

log "gofmt check"
if [ -n "$go_files" ]; then
  unformatted=$(gofmt -l $go_files)
  if [ -n "$unformatted" ]; then
    printf '%s\n' "$unformatted" >&2
    printf '%s\n' "Run gofmt on the files above." >&2
    exit 1
  fi
fi

log "public docs/examples legacy import guard"
if git grep -n -E 'github\.com/agkloop/go_memoize/(v2|helpers)|go_memoize/(v2|helpers)' -- README.md docs examples adapters/redis/examples ':!docs/superpowers'; then
  printf '%s\n' "Public docs/examples must not use legacy module or helper import paths." >&2
  exit 1
fi

check_tidy .
check_tidy adapters/redis

log "root go vet"
go vet ./...

log "redis adapter go vet"
(
  cd adapters/redis
  go vet ./...
)

log "root tests"
go test ./... -count=1

log "root race tests"
go test ./... -race -count=1

log "redis adapter tests"
(
  cd adapters/redis
  go test ./... -count=1
)

log "redis adapter race tests"
(
  cd adapters/redis
  go test ./... -race -count=1
)

log "benchmark smoke"
go test ./benchmarks/ -bench=. -benchmem -benchtime=100ms -count=1

log "govulncheck root"
run_optional govulncheck ./...

log "govulncheck redis adapter"
if command -v govulncheck >/dev/null 2>&1; then
  (
    cd adapters/redis
    govulncheck ./...
  )
elif [ "$require_optional" = "1" ]; then
  printf '%s\n' "Required tool not found on PATH: govulncheck" >&2
  exit 1
else
  printf '%s\n' "Skipping optional check; govulncheck is not installed." >&2
fi

log "golangci-lint root"
run_optional golangci-lint run ./...

log "golangci-lint redis adapter"
if command -v golangci-lint >/dev/null 2>&1; then
  (
    cd adapters/redis
    golangci-lint run ./...
  )
elif [ "$require_optional" = "1" ]; then
  printf '%s\n' "Required tool not found on PATH: golangci-lint" >&2
  exit 1
else
  printf '%s\n' "Skipping optional check; golangci-lint is not installed." >&2
fi

log "full verification passed"
