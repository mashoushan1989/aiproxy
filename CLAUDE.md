# CLAUDE.md

## Project Overview

AI Proxy — production AI gateway (OpenAI/Anthropic/Gemini-compatible). 40+ providers, multi-tenant (Group/Token), quota, rate limiting, usage analytics. Enterprise fork of labring/aiproxy.

## Build

```bash
bash scripts/build.sh                                                    # full build
cd core && go build -tags enterprise -trimpath -ldflags "-s -w" -o aiproxy  # backend only
cd core && go test -tags enterprise -v -timeout 30s -count=1 ./...         # tests
cd core && golangci-lint run --path-mode=abs --fix --build-tags=enterprise  # lint
cd web && pnpm install && pnpm run build                                   # frontend
```

**Always use `-tags enterprise`.** Local dev: `docker compose up -d pgsql redis && cd core && go run -tags enterprise .`

## Deploy

**Always use `deploy.sh` (zero-downtime Docker). NEVER bare `go build`, NEVER manual `docker stop/run`.**

```bash
bash scripts/deploy.sh                # full deploy
bash scripts/deploy.sh --no-pull      # current code (also for env-only changes)
bash scripts/deploy.sh --rollback     # emergency rollback
bash scripts/deploy-all.sh            # multi-node (domestic → overseas)
```

Server access — see `DEPLOYMENT.md`. SSH shortcuts:
- Domestic: `ssh -p 2222 -i ~/.ssh/id_ed25519 "ash@ppuser@1.13.81.31"@jump-new.paigod.work`
- Overseas: `ssh -p 2222 -i ~/.ssh/id_ed25519 "ash.ma@ppuser@10.195.9.13"@jump.pplabs.tech`

## Architecture

```
Client → Router → TokenAuth → Distribute → Relay Controller → Adaptor → Provider
                                  ↓              ↓              ↓
                            Group/Token     RPM/TPM Limit   Plugin Chain
```

- `core/middleware/distributor.go` — Central orchestrator
- `core/relay/adaptor/` — ~40 provider adaptors, self-register via `registry.Register()` in `init()`
- `core/relay/plugin/` — cache, web-search, thinksplit, streamfake, cachefollow, patch, monitor, timeout
- Protocol conversion: OpenAI ↔ Claude ↔ Gemini ↔ Responses API. Native mode preferred (`NativeModeChecker`)
- Sync ownership: `model_configs.synced_from` (`'ppio'`/`'novita'` = sync-owned; `''` = manual). Sync must not touch unowned rows

## Key Env Vars

`SQL_DSN`, `LOG_SQL_DSN`, `REDIS_CONN_STRING`, `ADMIN_KEY`, `NODE_CHANNEL_SET` (e.g. `overseas`), `GLOBAL_BACKGROUND_TASKS_ENABLED` (overseas must be `false`)

## Enterprise

Build tag `enterprise` required. All `core/enterprise/*.go` must start with `//go:build enterprise`.
Subsystems: `feishu/` (SSO + org sync), `analytics/` (9 endpoints), `quota/` (progressive tiers).

## Preferences

- 中文交流，代码注释英文
- 中文 commit message，Conventional Commits
- 修改后运行相关测试
- golangci-lint v2 (`.golangci.yml`)
