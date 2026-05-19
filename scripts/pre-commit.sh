#!/usr/bin/env sh
set -eu

root=$(git rev-parse --show-toplevel)
cd "$root"

log() {
  printf '\n==> %s\n' "$1"
}

go_files=$(
  for file in $(git ls-files '*.go'); do
    if [ -f "$file" ]; then
      printf '%s\n' "$file"
    fi
  done
)

log "docs/skill sync"
scripts/check-docs-skill-sync.sh --staged

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

log "root go vet"
go vet ./...

log "root tests"
go test ./... -count=1

log "redis adapter tests"
(
  cd adapters/redis
  go test ./... -count=1
)

log "pre-commit checks passed"
