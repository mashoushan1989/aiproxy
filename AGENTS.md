# Repository Guidelines

## Project Structure & Module Organization

AI Proxy is an enterprise AI gateway. `core/` contains the Gin/GORM backend; `core/enterprise/` uses the `enterprise` build tag for Feishu SSO, quota, analytics, and provider sync. `web/` is the React/Vite admin panel. `mcp-servers/` contains MCP servers, and `openapi-mcp/` contains the OpenAPI-to-MCP converter. Scripts live in `scripts/`, deployment config in `deploy/`, and docs in `docs/`.

## Build, Test, and Development Commands

- `make branch NAME=fix/passthrough-mode-gate`: start a scoped task branch.
- `make check`: run the local quality gate.
- `make check-quick`: run core backend tests while iterating.
- `bash scripts/build.sh`: rebuild frontend, Swagger, enterprise binary, and verify output.
- `cd core && go test -tags enterprise -v -timeout 30s -count=1 ./...`: backend tests.
- `cd core && golangci-lint run --path-mode=abs --build-tags=enterprise`: Go lint.
- `cd web && pnpm install && pnpm run dev`: install deps and start Vite.
- `cd web && pnpm run build && pnpm run lint`: type-check/build and ESLint.
- `docker compose up -d pgsql redis`: start local PostgreSQL and Redis.

## Coding Style & Naming Conventions

Use Go 1.26, `gofmt`/`gofumpt`, and `golangci-lint` v2 rules. Backend changes should compile and test with `-tags enterprise`. Keep provider adaptors under `core/relay/adaptor/<provider>/` and register them with channel types from `core/model/chtype.go`. Frontend uses TypeScript, React 19, Tailwind, Radix UI, and Zustand.

## Architecture Principles

Optimize for exact request/response pass-through. Avoid unnecessary normalization, mutation, protocol conversion, default injection, or compatibility rewrites unless required by a documented provider contract or existing API behavior. Test required transformations.

## Testing Guidelines

Place Go tests next to implementation files as `*_test.go`, with focused `Test...` names. Prefer table-driven tests for conversion, routing, billing, quota, and adaptor behavior. Run the smallest relevant package first, then `make check` or `make check-backend`.

## Commit & Pull Request Guidelines

Do not develop directly on `main` for normal work. Before edits, check `git status --short --branch`; if on `main`, run `make branch NAME=type/short-name`. Keep one logical change per commit and avoid staging unrelated dirty files. Use Conventional Commit style such as `fix(my-access): ...` or `feat(sync): ...`. PRs should explain behavior, risk, verification, linked context, and UI screenshots when relevant.

## Security & Configuration Tips

Do not commit keys, database URLs, or production logs. Use `config.example.yaml`, `config.md`, and environment variables. Before connecting to online servers, read `DEPLOYMENT.md` and relevant `deploy/` files. Deploy with `scripts/deploy.sh` or Docker; avoid ad hoc server builds that may omit `enterprise`.
