#!/usr/bin/env bash
#
# Start a scoped task branch with the naming convention used by this repo.

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: bash scripts/new-task-branch.sh <type/short-name>

Examples:
  bash scripts/new-task-branch.sh fix/passthrough-mode-gate
  bash scripts/new-task-branch.sh chore/dev-workflow-guardrails
EOF
}

if [[ $# -ne 1 || "$1" == "-h" || "$1" == "--help" ]]; then
  usage
  exit 0
fi

branch="$1"

if [[ ! "${branch}" =~ ^(fix|feat|chore|docs|test|refactor|perf|build|ci)/[a-z0-9._-]+$ ]]; then
  echo "Invalid branch name: ${branch}" >&2
  echo "Expected: fix|feat|chore|docs|test|refactor|perf|build|ci/<lowercase-short-name>" >&2
  exit 1
fi

current_branch="$(git branch --show-current)"

if [[ -z "${current_branch}" ]]; then
  echo "Cannot create a task branch from a detached HEAD." >&2
  exit 1
fi

if git show-ref --verify --quiet "refs/heads/${branch}"; then
  echo "Branch already exists: ${branch}" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Working tree has local changes; they will remain on the new branch." >&2
fi

git switch -c "${branch}"
