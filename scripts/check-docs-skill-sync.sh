#!/usr/bin/env sh
set -eu

skill_file=".agents/skills/go-memoize-package/SKILL.md"
mode="${1:-worktree}"

case "$mode" in
  --staged)
    changed_files=$(git diff --name-only --cached | sort -u)
    ;;
  worktree)
    changed_files=$(
      {
        git diff --name-only --cached
        git diff --name-only
        git ls-files --others --exclude-standard
      } | sort -u
    )
    ;;
  *)
    echo "usage: $0 [--staged]" >&2
    exit 2
    ;;
esac

docs_changed=$(printf '%s\n' "$changed_files" | awk '
  /^README\.md$/ { found=1 }
  /^docs\// && $0 !~ /^docs\/superpowers\// { found=1 }
  /^examples\// { found=1 }
  /^adapters\/redis\/examples\// { found=1 }
  END { if (found) print "yes" }
')

if [ "$docs_changed" = "yes" ]; then
  if [ ! -f "$skill_file" ]; then
    echo "Public docs or examples changed, but $skill_file is missing." >&2
    exit 1
  fi

  missing=""
  for required in \
    "## Public API Rules" \
    "## Background And Loader Semantics" \
    "## Docs Sync Rule" \
    "background.Value.Get()" \
    "memoize.New[K,V]" \
    "RecordMetric(memoize.MetricEvent)"
  do
    if ! grep -Fq "$required" "$skill_file"; then
      missing="${missing}\n- ${required}"
    fi
  done

  if [ -n "$missing" ]; then
    printf '%s\n' "Public docs or examples changed, but $skill_file is missing sync anchors:" >&2
    printf '%b\n' "$missing" >&2
    printf '%s\n' "Update the go-memoize-package skill before finishing docs-heavy work." >&2
    exit 1
  fi
fi

exit 0
