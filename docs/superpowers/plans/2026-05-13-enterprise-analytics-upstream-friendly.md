# Enterprise Analytics Upstream-Friendly Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade Enterprise Analytics for ToB private deployment while keeping future upstream open-source merges cheap and predictable.

**Architecture:** Put commercial analytics behavior in additive enterprise-only packages and keep existing upstream-facing files thin. New scope, validation, query, audit, and org-directory logic lives behind adapters; existing handlers and routes only delegate to the new layer under feature flags or API v2 paths.

**Tech Stack:** Go 1.24, enterprise build tag, Gin, GORM, existing `model.GroupSummary`, existing enterprise org models, React, TanStack Query, pnpm.

---

## Maintenance Objective

This plan optimizes for long-term fork maintenance:

- Keep upstream file diffs small and mechanical.
- Prefer new enterprise-only packages over rewriting existing analytics files.
- Preserve existing function signatures where possible.
- Add v2 APIs and feature flags instead of replacing behavior in place.
- Keep `model.GroupSummary` read-only from commercial analytics code.
- Register new migrations through one stable enterprise analytics migration entry.

## Why This Replaces The Earlier Execution Strategy

The first commercialization plan correctly identified product gaps, but it proposed broad edits to high-conflict files such as:

- `core/enterprise/analytics/handler.go`
- `core/enterprise/analytics/department.go`
- `core/enterprise/analytics/ranking.go`
- `core/enterprise/analytics/model_distribution.go`
- `core/enterprise/analytics/custom_report.go`
- `web/src/pages/enterprise/dashboard.tsx`
- `web/src/pages/enterprise/custom-report/index.tsx`

Those files are likely to keep changing upstream. This revised plan moves most new behavior into additive packages and leaves old code available as compatibility paths.

## Package Boundaries

Create a new package tree:

```text
core/enterprise/analyticsx/
  scope.go
  request.go
  service.go
  org_directory.go
  audit.go
  migration.go
  export.go
  custom_report.go
  rollup.go
  aggregate.go
```

Rules:

- `analyticsx` may import existing `core/enterprise/analytics` DTOs for compatibility.
- Existing `core/enterprise/analytics` should not import deep commercial logic except thin adapters.
- Route changes are limited to registering v2 endpoints or delegating old endpoints behind a feature flag.
- Enterprise migrations call one stable `analyticsx.AutoMigrate(db)` entry.

Frontend additive tree:

```text
web/src/api/enterprise-analytics-v2.ts
web/src/pages/enterprise/dashboard-v2/
web/src/pages/enterprise/custom-report-v2/
```

Rules:

- Existing dashboard and custom-report pages remain mostly intact.
- V2 components can be mounted behind a feature flag or route.
- Existing API client types are not broken.

## Feature Flags

Backend flags:

```text
ENTERPRISE_ANALYTICS_SCOPE_ENFORCED=false
ENTERPRISE_ANALYTICS_V2_ENABLED=false
ENTERPRISE_ANALYTICS_AGGREGATES_ENABLED=false
ENTERPRISE_ANALYTICS_ASYNC_EXPORT_ENABLED=false
```

Frontend flags:

```text
VITE_ENTERPRISE_ANALYTICS_V2=false
```

Rollout rule:

- New code ships disabled.
- Admin-only comparison mode can be enabled first.
- Scoped enforcement is enabled only after parity tests pass.

## Target Flow

```text
Gin route
  -> existing auth and permission middleware
  -> analyticsx.ResolveScope
  -> analyticsx.ParseRequest
  -> analyticsx.Service
  -> analyticsx.OrgDirectory
  -> GroupSummary or analyticsx aggregate tables
  -> v2 response or legacy-compatible DTO
```

## Compatibility Contract

- Existing endpoint paths remain available.
- Existing response fields remain available until frontend has migrated.
- Existing Feishu analytics helpers are not deleted in the first pass.
- Existing `group_id` remains the usage subject.
- Existing report templates remain readable; missing visibility is treated as `private`.
- Existing `model.GroupSummary` schema is not modified for analytics commercialization.

## File Structure

- Create `core/enterprise/analyticsx/scope.go`
  Defines `Scope`, `Resolver`, and scope intersection helpers.
- Create `core/enterprise/analyticsx/scope_test.go`
  Tests admin, member, explicit org ownership, and scope intersection.
- Create `core/enterprise/analyticsx/request.go`
  Defines bounded request parsing independent of existing handlers.
- Create `core/enterprise/analyticsx/request_test.go`
  Tests time range, limit, granularity, and pagination validation.
- Create `core/enterprise/analyticsx/org_directory.go`
  Reads provider-neutral org/user/group mappings, with Feishu compatibility lookup.
- Create `core/enterprise/analyticsx/org_directory_test.go`
  Tests descendant expansion, group mapping, and compatibility IDs.
- Create `core/enterprise/analyticsx/service.go`
  Provides scope-aware dashboard, ranking, distribution, trend, and comparison queries.
- Create `core/enterprise/analyticsx/service_test.go`
  Tests no global fallback, out-of-scope filters, and query timeout context use.
- Create `core/enterprise/analyticsx/export.go`
  Builds export datasets through the same service queries.
- Create `core/enterprise/analyticsx/audit.go`
  Defines audit event model and persistence.
- Create `core/enterprise/analyticsx/audit_test.go`
  Tests success/failure audit events and sensitive-data exclusion.
- Create `core/enterprise/analyticsx/migration.go`
  Owns analyticsx migration registration.
- Modify `core/enterprise/models/migrate.go`
  Adds one stable call to `analyticsx.AutoMigrate(db)`.
- Modify `core/enterprise/analytics/register.go`
  Registers v2 routes when enabled.
- Modify `core/enterprise/analytics/handler.go`
  Adds minimal adapter calls only where legacy endpoints are explicitly switched to scoped mode.
- Create `web/src/api/enterprise-analytics-v2.ts`
  Adds v2 API types and client methods.
- Create `web/src/pages/enterprise/dashboard-v2/index.tsx`
  Uses backend server-rollup data.
- Modify `web/src/pages/enterprise/dashboard.tsx`
  Adds a small feature-flag switch to render v2 dashboard when enabled.

## Tasks

### Task 1: Add Analyticsx Skeleton And Flags

**Files:**
- Create: `core/enterprise/analyticsx/config.go`
- Create: `core/enterprise/analyticsx/config_test.go`
- Create: `core/enterprise/analyticsx/doc.go`
- Modify: `core/enterprise/analytics/register.go`

- [ ] **Step 1: Write flag tests**

Create tests:

```go
func TestConfigDefaultsKeepAnalyticsV2Disabled(t *testing.T)
func TestConfigReadsAnalyticsV2Enabled(t *testing.T)
func TestConfigReadsScopeEnforced(t *testing.T)
```

- [ ] **Step 2: Run failing tests**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run TestConfig -count=1
```

Expected: fail because `analyticsx` does not exist.

- [ ] **Step 3: Implement config**

Add:

```go
type Config struct {
	V2Enabled          bool
	ScopeEnforced     bool
	AggregatesEnabled bool
	AsyncExportEnabled bool
}
```

Read environment variables:

```text
ENTERPRISE_ANALYTICS_V2_ENABLED
ENTERPRISE_ANALYTICS_SCOPE_ENFORCED
ENTERPRISE_ANALYTICS_AGGREGATES_ENABLED
ENTERPRISE_ANALYTICS_ASYNC_EXPORT_ENABLED
```

Default all values to `false`.

- [ ] **Step 4: Add route registration seam**

In `core/enterprise/analytics/register.go`, add only a small conditional registration block:

```go
if analyticsx.LoadConfig().V2Enabled {
	analyticsx.RegisterRoutes(analytics, permMiddleware)
}
```

- [ ] **Step 5: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run TestConfig -count=1
```

Expected: pass.

### Task 2: Scope Resolver Without Rewriting Legacy Queries

**Files:**
- Create: `core/enterprise/analyticsx/scope.go`
- Create: `core/enterprise/analyticsx/scope_test.go`

- [ ] **Step 1: Write scope tests**

Create tests:

```go
func TestResolveScopeAdminIsGlobal(t *testing.T)
func TestResolveScopeMemberUsesOwnGroup(t *testing.T)
func TestIntersectGroupsDoesNotWidenAccess(t *testing.T)
func TestIntersectGroupsEmptyRequestMeansAllowedGroups(t *testing.T)
```

- [ ] **Step 2: Run failing tests**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run 'TestResolveScope|TestIntersectGroups' -count=1
```

Expected: fail until scope exists.

- [ ] **Step 3: Implement additive scope types**

Use names local to `analyticsx` to avoid conflicts:

```go
type Scope struct {
	WorkspaceID       string
	Role              string
	CallerGroupID     string
	CallerUserID      string
	AllowedOrgUnitIDs []string
	AllowedUserIDs    []string
	AllowedGroupIDs   []string
	AllowedModels     []string
	AllOrgUnits       bool
	AllUsers          bool
	AllGroups         bool
}
```

Implement:

```go
func IntersectGroupIDs(scope Scope, requested []string) []string
```

Rules:

- Admin scope can return requested groups or all groups depending on request.
- Non-admin scope cannot return groups outside `AllowedGroupIDs`.
- Empty request means all allowed groups for non-admin users.
- Empty intersection returns an empty slice and is not treated as global.

- [ ] **Step 4: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run 'TestResolveScope|TestIntersectGroups' -count=1
```

Expected: pass.

### Task 3: Request Validation As A Standalone Adapter

**Files:**
- Create: `core/enterprise/analyticsx/request.go`
- Create: `core/enterprise/analyticsx/request_test.go`

- [ ] **Step 1: Write request tests**

Create tests:

```go
func TestParseRequestDefaultsToSevenDays(t *testing.T)
func TestParseRequestRejectsEndBeforeStart(t *testing.T)
func TestParseRequestRejectsInteractiveRangeOverNinetyDays(t *testing.T)
func TestParseRequestRequiresDailyGranularityForLongHourlyRange(t *testing.T)
func TestParseRequestClampsPagination(t *testing.T)
```

- [ ] **Step 2: Implement request type**

Add:

```go
type Filter struct {
	StartTimestamp int64
	EndTimestamp   int64
	Granularity    string
	OrgUnitIDs     []string
	GroupIDs       []string
	UserIDs        []string
	Models         []string
	Limit          int
	Page           int
	PerPage        int
}
```

Validation constants:

```go
const (
	DefaultRangeDays = 7
	MaxInteractiveRangeDays = 90
	MaxHourlyRangeDays = 31
	MaxLimit = 1000
	MaxPageSize = 500
)
```

- [ ] **Step 3: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run TestParseRequest -count=1
```

Expected: pass.

### Task 4: Org Directory Adapter

**Files:**
- Create: `core/enterprise/analyticsx/org_directory.go`
- Create: `core/enterprise/analyticsx/org_directory_test.go`

- [ ] **Step 1: Write org directory tests**

Create tests:

```go
func TestOrgDirectoryDescendantsUseCanonicalOrgUnits(t *testing.T)
func TestOrgDirectoryMapsOrgUnitsToGroupIDs(t *testing.T)
func TestOrgDirectorySupportsFeishuCompatibilityIDs(t *testing.T)
func TestOrgDirectoryEmptyMatchReturnsEmptyGroups(t *testing.T)
```

- [ ] **Step 2: Implement interface**

Add:

```go
type OrgDirectory interface {
	DescendantOrgUnitIDs(ctx context.Context, workspaceID string, orgUnitIDs []string) ([]string, error)
	GroupIDsForOrgUnits(ctx context.Context, workspaceID string, orgUnitIDs []string) ([]string, error)
	GroupIDsForUsers(ctx context.Context, workspaceID string, userIDs []string) ([]string, error)
}
```

Implementation reads:

- `enterprise_org_units`
- `enterprise_users`
- `enterprise_user_org_units`
- `groups.workspace_id`
- `groups.owner_user_id`
- `groups.org_unit_id`

Feishu compatibility lookup is allowed only inside this adapter.

- [ ] **Step 3: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run TestOrgDirectory -count=1
```

Expected: pass.

### Task 5: Scope-Aware Service As A Parallel Path

**Files:**
- Create: `core/enterprise/analyticsx/service.go`
- Create: `core/enterprise/analyticsx/service_test.go`

- [ ] **Step 1: Write service tests**

Create tests:

```go
func TestServiceDepartmentSummaryUsesOnlyScopedGroups(t *testing.T)
func TestServiceUserRankingOutOfScopeDepartmentReturnsEmpty(t *testing.T)
func TestServiceModelDistributionNeverFallsBackToGlobal(t *testing.T)
func TestServiceQueriesUseContextTimeout(t *testing.T)
```

- [ ] **Step 2: Implement service**

Add:

```go
type Service struct {
	DB           *gorm.DB
	LogDB        *gorm.DB
	OrgDirectory OrgDirectory
	Timeout      time.Duration
}
```

Methods:

```go
func (s Service) DepartmentSummaries(ctx context.Context, scope Scope, filter Filter) ([]analytics.DepartmentSummary, error)
func (s Service) UserRanking(ctx context.Context, scope Scope, filter Filter) ([]analytics.UserRankingEntry, int, error)
func (s Service) ModelDistribution(ctx context.Context, scope Scope, filter Filter) ([]analytics.ModelDistributionEntry, error)
```

Rules:

- Accept and return existing analytics DTOs to reduce frontend/API churn.
- Do not modify legacy query function signatures.
- Use `LogDB.WithContext(ctx)` for all summary queries.
- Empty group intersection returns empty data.

- [ ] **Step 3: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run TestService -count=1
```

Expected: pass.

### Task 6: V2 Routes With Thin Registration

**Files:**
- Create: `core/enterprise/analyticsx/routes.go`
- Create: `core/enterprise/analyticsx/handler.go`
- Modify: `core/enterprise/analytics/register.go`

- [ ] **Step 1: Add v2 endpoints**

Register under existing analytics group:

```text
GET /analytics/v2/department
GET /analytics/v2/user/ranking
GET /analytics/v2/model/distribution
GET /analytics/v2/export
```

Use the same permission middleware keys as current endpoints.

- [ ] **Step 2: Keep old endpoints untouched**

Do not replace legacy endpoint behavior in this task. The only existing-file change is route registration gated by `ENTERPRISE_ANALYTICS_V2_ENABLED`.

- [ ] **Step 3: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analytics ./enterprise/analyticsx -count=1
```

Expected: pass.

### Task 7: Scoped Export And Audit In Analyticsx

**Files:**
- Create: `core/enterprise/analyticsx/export.go`
- Create: `core/enterprise/analyticsx/audit.go`
- Create: `core/enterprise/analyticsx/audit_test.go`
- Create: `core/enterprise/analyticsx/migration.go`
- Modify: `core/enterprise/models/migrate.go`

- [ ] **Step 1: Write audit tests**

Create tests:

```go
func TestAuditEventPersistsExportSuccess(t *testing.T)
func TestAuditEventPersistsExportFailure(t *testing.T)
func TestAuditEventDoesNotStoreSecrets(t *testing.T)
func TestExportDatasetUsesServiceScopedFilters(t *testing.T)
```

- [ ] **Step 2: Implement audit model**

Add:

```go
type AuditEvent struct {
	ID           int
	WorkspaceID  string
	ActorGroupID string
	Action       string
	ScopeSummary string
	FilterJSON   string
	ResultStatus string
	RowCount     int
	ErrorMessage  string
	CreatedAt    time.Time
}
```

- [ ] **Step 3: Add stable migration seam**

In `analyticsx/migration.go`:

```go
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&AuditEvent{})
}
```

In `core/enterprise/models/migrate.go`, add only:

```go
if err := analyticsx.AutoMigrate(db); err != nil {
	return err
}
```

- [ ] **Step 4: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run 'TestAudit|TestExportDataset' -count=1
cd core && go test -tags enterprise ./enterprise/models -count=1
```

Expected: pass.

### Task 8: Frontend V2 Additive Entry

**Files:**
- Create: `web/src/api/enterprise-analytics-v2.ts`
- Create: `web/src/pages/enterprise/dashboard-v2/index.tsx`
- Modify: `web/src/pages/enterprise/dashboard.tsx`

- [ ] **Step 1: Add v2 API client**

Expose:

```ts
getDepartmentSummaryV2(params)
getUserRankingV2(params)
getModelDistributionV2(params)
exportAnalyticsV2(params)
```

- [ ] **Step 2: Add isolated dashboard v2 component**

Component consumes backend rollup data directly and does not perform distinct metric rollup in the browser.

- [ ] **Step 3: Add tiny feature switch to existing dashboard**

In `dashboard.tsx`, add only:

```tsx
if (import.meta.env.VITE_ENTERPRISE_ANALYTICS_V2 === "true") {
  return <EnterpriseDashboardV2 />
}
```

- [ ] **Step 4: Verify**

Run:

```bash
cd web && pnpm run build
cd web && pnpm run lint
```

Expected: build succeeds; lint either succeeds or reports existing unrelated failures.

### Task 9: Aggregates As A Disabled Sidecar

**Files:**
- Create: `core/enterprise/analyticsx/aggregate.go`
- Create: `core/enterprise/analyticsx/aggregate_test.go`
- Modify: `core/enterprise/analyticsx/migration.go`

- [ ] **Step 1: Write aggregate tests**

Create tests:

```go
func TestAggregateHourlyIsIdempotent(t *testing.T)
func TestAggregateDailyRollsUpHourlyRows(t *testing.T)
func TestAggregateDoesNotModifyGroupSummary(t *testing.T)
```

- [ ] **Step 2: Implement sidecar tables**

Create analyticsx-owned models:

```text
enterprise_analyticsx_hourly
enterprise_analyticsx_daily
enterprise_analyticsx_monthly
```

Do not modify `model.GroupSummary`.

- [ ] **Step 3: Keep runtime disabled**

Dashboard uses aggregates only when `ENTERPRISE_ANALYTICS_AGGREGATES_ENABLED=true`.

- [ ] **Step 4: Verify**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -run TestAggregate -count=1
```

Expected: pass.

## Architect Review

### Maintainability

Assessment: acceptable after this revision.

Reasons:

- Most commercial behavior is isolated in `analyticsx`.
- Existing upstream files receive small, reviewable changes.
- New migrations are hidden behind one stable entry.
- Existing DTOs and endpoint paths remain available.
- V2 APIs allow parity testing before replacing legacy behavior.

Residual risk:

- `register.go` and `models/migrate.go` remain merge touchpoints.
- Frontend `dashboard.tsx` still receives a small feature-flag import.
- If upstream changes auth context keys, `analyticsx.Scope` resolver must adapt.

### Security

Assessment: directionally strong, but Phase 0 must ship before report-product features.

Required guardrails:

- Empty scope intersection must never become global.
- Export must share the same service query path as page views.
- V2 endpoints must use the same permission middleware keys as legacy analytics endpoints.
- Audit events must not store secrets or full export content.

### Performance

Assessment: incremental and safe.

Reasons:

- Query bounds are introduced before aggregate tables.
- Aggregates are sidecar and disabled by default.
- `GroupSummary` remains source of truth until parity is proven.

Required guardrails:

- All service queries must use context timeouts.
- Large report/export queries must remain blocked or paginated until async export is enabled.
- Aggregates must be idempotent and replayable.

### Product

Assessment: good sequencing.

Reasons:

- Security and correctness come before template sharing and scheduling.
- Dashboard v2 can be validated by admins before user-wide rollout.
- Existing report templates remain compatible.

### Upstream Merge Risk

Assessment: medium-low if implemented as written.

The biggest merge risks are reduced from broad analytics rewrites to:

- one route registration seam;
- one migration registration seam;
- one frontend feature switch;
- additive files that rarely conflict with upstream.

## Branch And Worktree Recommendation

Current repository state:

```text
current branch: feat/private-cloud-workspace-governance
dirty files: .gitignore and untracked docs
worktree directory: .worktrees does not exist and is not ignored
```

Recommendation:

- Do not start implementation in the current working tree.
- First commit or stash the current documentation/governance work.
- Then create a new isolated development branch:

```text
feat/enterprise-analytics-upstream-friendly
```

Preferred execution:

- Use a git worktree for implementation once `.worktrees/` is added to `.gitignore`, or use a global worktree directory outside the repo.
- Run baseline checks in the new worktree before coding:

```bash
cd core && go test -tags enterprise ./enterprise/analytics ./enterprise/models -count=1
cd web && pnpm run build
```

Decision:

- Open a new branch for development.
- Prefer a worktree because the current branch already contains private-cloud workspace governance work and untracked planning docs.
- Do not mix this analytics implementation into `feat/private-cloud-workspace-governance`; the scope is independent and would make review and rollback harder.

## Verification Matrix

Backend narrow checks:

```bash
cd core && go test -tags enterprise ./enterprise/analyticsx -count=1
```

Backend compatibility checks:

```bash
cd core && go test -tags enterprise ./enterprise/analytics ./enterprise/models -count=1
```

Frontend checks:

```bash
cd web && pnpm run build
cd web && pnpm run lint
```

Manual smoke checks after enabling v2:

```bash
curl -sS -H "Authorization: Bearer $ADMIN_KEY" "http://127.0.0.1:3000/api/enterprise/analytics/v2/department"
curl -sS -H "Authorization: Bearer $ADMIN_KEY" "http://127.0.0.1:3000/api/enterprise/analytics/v2/user/ranking"
curl -sS -H "Authorization: Bearer $ADMIN_KEY" "http://127.0.0.1:3000/api/enterprise/analytics/v2/model/distribution"
```

Security review checks:

- Non-admin user cannot query out-of-scope department, user, group, or model.
- Export row counts match scoped API row counts.
- Invalid time ranges return `400`.
- Feature flags disabled means legacy behavior remains unchanged.

Merge-maintenance review checks:

- No existing analytics function signature is changed.
- No `model.GroupSummary` schema change exists.
- Existing endpoints still compile and return legacy DTOs.
- New code lives under `analyticsx` except route/migration seams.
- `git diff --stat` shows most lines in new files.

## Self Review

Spec coverage:

- Data scope, export consistency, query bounds, org decoupling, frontend rollup, aggregates, and audit are covered.
- Upstream merge maintainability is explicitly covered by package boundaries, feature flags, and branch guidance.

Placeholder scan:

- No placeholder markers are used.
- Every task has exact files, tests, commands, and expected results.

Type consistency:

- The plan consistently uses `analyticsx.Scope`, `analyticsx.Filter`, `analyticsx.Service`, and `analyticsx.OrgDirectory`.
- V2 API naming is consistent across backend and frontend tasks.

Architect decision:

- Proceed with a new branch/worktree after current workspace cleanup.
- Implement Tasks 1-7 first as the minimum commercial safety milestone.
- Defer Task 9 aggregates until scoped v2 parity is proven.

## Implementation Status

Updated: 2026-05-13.

Completed on branch `feat/enterprise-analytics-upstream-friendly`:

- Tasks 1-4: additive `analyticsx` foundation, feature flags, scope resolver, bounded request parsing, and provider-neutral org directory adapter.
- Task 5: scoped analytics service for department summaries, user ranking, and model distribution.
- Task 6: feature-flagged `/enterprise/analytics/v2/*` routes with legacy endpoints preserved.
- Task 7: scoped export dataset builder, v2 export route, export audit persistence, and secret/raw actor redaction.
- Task 8: additive frontend v2 API client and dashboard entry behind `VITE_ENTERPRISE_ANALYTICS_V2`.
- Task 9: disabled hourly/daily/monthly aggregate sidecar tables on LogDB with explicit aggregation functions; no runtime read path uses aggregates yet.

Verification completed:

```bash
cd core && go test -tags enterprise ./enterprise/analytics ./enterprise/analyticsx ./enterprise/models ./model -count=1
cd web && pnpm run build
```

Known verification caveat:

- `cd web && pnpm run lint` still fails on pre-existing unrelated files (`RequirePermission.tsx`, `custom-report/ReportChart.tsx`) while the Task 8 touched files pass targeted ESLint.

Merge readiness notes:

- This worktree branch also contains earlier identity-source, org-sync, and passthrough/channel commits outside the enterprise analytics scope. For easier review and future upstream merges, prefer either splitting PRs by subsystem or clearly labeling one stacked PR series.
- Aggregate tables are migrated in enterprise builds but remain disabled as a read path. Enabling `ENTERPRISE_ANALYTICS_AGGREGATES_ENABLED` still requires a follow-up parity comparison task before dashboard/report queries read from sidecar tables.
- Hook-suggested docs (`CLAUDE.md`, `USER-GUIDE.md`, `enterprise/docs/progress.md`) are not present in this worktree; this plan document is the current implementation record.
