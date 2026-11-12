#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$root"

for required in README.md LICENSE VERSION Makefile .gitignore .gitattributes; do
  if [ ! -f "$required" ]; then
    echo "missing required file: $required" >&2
    exit 1
  fi
done

version_val="$(tr -d '[:space:]' < VERSION)"
if [ -z "$version_val" ]; then
  echo "VERSION file must not be empty" >&2
  exit 1
fi

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  forbidden=$(git ls-files | grep -E '^(docs/|AGENTS\.md$|CLAUDE\.md$|GEMINI\.md$|\.agents/|\.claude/|\.codex/|\.gemini/|prompts/|\.env($|\.))' || true)
  if [ -n "$forbidden" ]; then
    echo "forbidden private/governance files are tracked by git:" >&2
    echo "$forbidden" >&2
    exit 1
  fi
fi

echo "repo hygiene OK (VERSION=$version_val)"
