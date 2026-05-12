# Identity Source Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add commercial identity source configuration for private-cloud deployments so admins can configure Feishu app credentials, see future WeCom/DingTalk slots, and run a safe self-check without changing current production login or sync behavior.

**Architecture:** Add provider-neutral enterprise identity-source persistence and a resolver that keeps environment variables as the effective runtime fallback. The first release stores Feishu configuration, exposes admin CRUD and doctor APIs, and adds a User Management UI tab; OAuth and scheduled sync continue using the existing environment-backed Feishu client until a later controlled cutover.

**Tech Stack:** Go 1.24, enterprise build tag, GORM, Gin, existing middleware response helpers, React, TanStack Query, shadcn-style UI components, pnpm.

---

## Scope

Included:
- Provider-neutral identity source table for `feishu`, `wecom`, and `dingtalk` readiness.
- Feishu config resolver with `db` vs `env` source reporting and masked secret output.
- Admin APIs under `/api/enterprise/identity-sources/:provider`.
- Self-check API that validates required fields and, when credentials are configured, probes Feishu tenant and contact permissions without writing synced data.
- User Management tab for identity source configuration, provider distinction, and self-check results.
- Compatibility guarantee: existing `FEISHU_APP_ID`, `FEISHU_APP_SECRET`, `FEISHU_REDIRECT_URI`, `FEISHU_FRONTEND_URL`, and `FEISHU_ALLOWED_TENANTS` continue to work unchanged.

Excluded from this implementation:
- Switching OAuth login or scheduled Feishu sync to DB-managed credentials.
- Real WeCom or DingTalk API clients.
- Multi-workspace UI.
- Secret encryption migration beyond not returning secrets to the browser. If a repo-wide encryption helper is added later, this table can be migrated in place.

## File Structure

- Create `core/enterprise/models/identity_source.go`
  Defines `IdentitySource`, provider constants, table name, and secret masking helpers.
- Modify `core/enterprise/models/migrate.go`
  Auto-migrates `IdentitySource`.
- Create `core/enterprise/identitysource/config.go`
  Resolves effective Feishu config from DB or env, saves admin config, and never returns plaintext secrets.
- Create `core/enterprise/identitysource/config_test.go`
  Tests env fallback, DB override, disabled DB fallback, secret preservation, and masked responses.
- Create `core/enterprise/identitysource/doctor.go`
  Runs non-mutating Feishu self-checks with injectable probes for tests.
- Create `core/enterprise/identitysource/doctor_test.go`
  Tests missing fields, env fallback warning, tenant mismatch warning, and permission failure reporting.
- Create `core/enterprise/identitysource/handler.go`
  Adds Gin handlers for read, update, and check.
- Create `core/enterprise/identitysource/routes.go`
  Registers identity source routes with user-management permissions plus admin role.
- Modify `core/enterprise/router.go`
  Wires identity source routes into enterprise auth.
- Modify `web/src/api/enterprise.ts`
  Adds identity source request/response types and API methods.
- Modify `web/src/pages/enterprise/users.tsx`
  Adds `IdentitySourceConfigTab` and the admin-only tab trigger.
- Modify `web/public/locales/en/translation.json`
  Adds English UI copy for the new tab.
- Modify `web/public/locales/zh/translation.json`
  Adds Chinese UI copy for the new tab.

## Tasks

### Task 1: Model And Resolver

**Files:**
- Create: `core/enterprise/models/identity_source.go`
- Modify: `core/enterprise/models/migrate.go`
- Create: `core/enterprise/identitysource/config.go`
- Create: `core/enterprise/identitysource/config_test.go`

- [ ] Write tests first for resolver behavior:
  - no DB row returns env source and env values;
  - enabled DB row with app id and secret overrides env;
  - disabled DB row falls back to env;
  - update with empty `app_secret` preserves existing secret;
  - response masks the secret as configured without exposing plaintext.
- [ ] Run `cd core && go test -tags enterprise ./enterprise/identitysource -run TestResolve -count=1` and confirm it fails because the package/types do not exist.
- [ ] Add `IdentitySource` model with unique `(workspace_id, provider)` and fields for external org ID, app ID, app secret, redirect URI, frontend URL, sync enabled, enabled, last check status, last check result, timestamps.
- [ ] Add `IdentitySource` to enterprise auto-migrate.
- [ ] Implement resolver and save helpers with default workspace and Feishu env fallback.
- [ ] Run `cd core && go test -tags enterprise ./enterprise/identitysource -count=1`.
- [ ] Run `cd core && go test -tags enterprise ./enterprise/models -count=1`.

### Task 2: Doctor And API

**Files:**
- Create: `core/enterprise/identitysource/doctor.go`
- Create: `core/enterprise/identitysource/doctor_test.go`
- Create: `core/enterprise/identitysource/handler.go`
- Create: `core/enterprise/identitysource/routes.go`
- Modify: `core/enterprise/router.go`

- [ ] Write tests first for doctor result aggregation:
  - missing app credentials fails;
  - env-sourced config returns a compatibility warning;
  - tenant API mismatch returns warning;
  - department/user permission probe failure reports the failing check.
- [ ] Run `cd core && go test -tags enterprise ./enterprise/identitysource -run TestDoctor -count=1` and confirm failure.
- [ ] Implement a probe interface and default Feishu probe using the existing Lark SDK client with supplied app id/secret.
- [ ] Implement self-check result types with `passed`, `warning`, and `failed` levels.
- [ ] Implement handlers:
  - `GET /identity-sources/:provider`;
  - `PUT /identity-sources/:provider`;
  - `POST /identity-sources/:provider/check`.
- [ ] Register routes under enterprise auth. Read requires `user_manage_view`; update and check require `user_manage_manage` and admin role.
- [ ] Run `cd core && go test -tags enterprise ./enterprise/identitysource -count=1`.
- [ ] Run `cd core && go test -tags enterprise ./enterprise/... -count=1`.

### Task 3: User Management UI

**Files:**
- Modify: `web/src/api/enterprise.ts`
- Modify: `web/src/pages/enterprise/users.tsx`
- Modify: `web/public/locales/en/translation.json`
- Modify: `web/public/locales/zh/translation.json`

- [ ] Add TypeScript types for identity source config, update payload, check item, and check response.
- [ ] Add API methods `getIdentitySource`, `updateIdentitySource`, and `checkIdentitySource`.
- [ ] Add `IdentitySourceConfigTab` with:
  - provider display cards for Feishu, WeCom, and DingTalk;
  - Feishu editable fields for external org ID, app ID, app secret replacement, redirect URI, frontend URL, enabled, and sync enabled;
  - current effective source badge (`env` or `db`);
  - save button and run self-check button;
  - self-check result list with pass/warning/fail styling.
- [ ] Add an admin-only tab trigger in User Management.
- [ ] Add zh/en translation keys.
- [ ] Run `cd web && pnpm run build`.
- [ ] Run `cd web && pnpm run lint`; if existing unrelated lint failures remain, report exact files.

### Task 4: Final Verification

**Files:**
- No new files expected.

- [ ] Run `git diff --stat` and inspect changed files for scope.
- [ ] Run `cd core && go test -tags enterprise ./enterprise/identitysource ./enterprise/models -count=1`.
- [ ] Run `cd web && pnpm run build`.
- [ ] Confirm no production credential or env file was touched.
- [ ] Summarize the compatibility boundary: DB identity source config can be saved and checked, while existing online Feishu OAuth/sync behavior remains env-backed in this phase.
