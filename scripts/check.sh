#!/usr/bin/env bash
#
# Unified local quality gate. Defaults to the same broad checks expected before
# handing work off for review; use --quick while iterating on backend changes.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORE_DIR="${ROOT_DIR}/core"
MCP_DIR="${ROOT_DIR}/mcp-servers"
OPENAPI_MCP_DIR="${ROOT_DIR}/openapi-mcp"
WEB_DIR="${ROOT_DIR}/web"

RUN_CORE=1
RUN_MCP=1
RUN_OPENAPI_MCP=1
RUN_WEB=1
RUN_LINT=1

usage() {
  cat <<'EOF'
Usage: bash scripts/check.sh [--quick] [--backend-only] [--no-lint] [--no-web]

Options:
  --quick         Run core backend tests only.
  --backend-only  Run Go module tests and lint; skip frontend checks.
  --no-lint      Skip golangci-lint.
  --no-web       Skip frontend lint/build.

Environment:
  GOCACHE         Defaults to /tmp/aiproxy-go-build to avoid host cache issues.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --quick)
      RUN_MCP=0
      RUN_OPENAPI_MCP=0
      RUN_WEB=0
      RUN_LINT=0
      shift
      ;;
    --backend-only)
      RUN_WEB=0
      shift
      ;;
    --no-lint)
      RUN_LINT=0
      shift
      ;;
    --no-web)
      RUN_WEB=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

export GOCACHE="${GOCACHE:-/tmp/aiproxy-go-build}"

info() { printf "[info] %s\n" "$*"; }
pass() { printf "[ok] %s\n" "$*"; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd go

if [[ "${RUN_CORE}" == "1" ]]; then
  info "Core tests (enterprise tag)"
  (cd "${CORE_DIR}" && go test -tags enterprise -v -timeout 30s -count=1 ./...)
  pass "Core tests passed"
fi

if [[ "${RUN_MCP}" == "1" ]]; then
  info "MCP server tests"
  (cd "${MCP_DIR}" && go test -v -timeout 30s -count=1 ./...)
  pass "MCP server tests passed"
fi

if [[ "${RUN_OPENAPI_MCP}" == "1" ]]; then
  info "OpenAPI MCP tests"
  (cd "${OPENAPI_MCP_DIR}" && go test -v -timeout 30s -count=1 ./...)
  pass "OpenAPI MCP tests passed"
fi

if [[ "${RUN_LINT}" == "1" ]]; then
  require_cmd golangci-lint

  info "Core lint"
  (cd "${CORE_DIR}" && golangci-lint run --path-mode=abs --build-tags=enterprise)
  pass "Core lint passed"

  info "MCP server lint"
  (cd "${MCP_DIR}" && golangci-lint run --path-mode=abs)
  pass "MCP server lint passed"

  info "OpenAPI MCP lint"
  (cd "${OPENAPI_MCP_DIR}" && golangci-lint run --path-mode=abs)
  pass "OpenAPI MCP lint passed"
fi

if [[ "${RUN_WEB}" == "1" ]]; then
  require_cmd pnpm

  info "Frontend install"
  (cd "${WEB_DIR}" && pnpm install --frozen-lockfile)

  info "Frontend lint"
  (cd "${WEB_DIR}" && pnpm run lint)

  info "Frontend build"
  (cd "${WEB_DIR}" && pnpm run build)
  pass "Frontend checks passed"
fi

pass "All requested checks passed"
