# Repository Guidance

- Backend Go modules live in `core`, `openapi-mcp`, and `mcp-servers`. The admin UI lives in `web`.
- Prefer narrow verification: run `go test ./path` from the affected Go module before `go test ./...`.
- Common backend checks: `cd core && go test ./...` and `cd core && go build -o aiproxy .`.
- Common frontend checks: `cd web && pnpm run lint` and `cd web && pnpm run build`.
- Use `pnpm` for `web`; the package declares `pnpm@10.32.0`.
- Keep generated frontend build output in `core/public/dist` only when the task explicitly requires packaging the UI.
- For relay, billing, model sync, and enterprise behavior, add or update adjacent Go tests when changing logic.
- Do not touch production credentials, local env files, or billing repair artifacts unless the task explicitly targets them.
