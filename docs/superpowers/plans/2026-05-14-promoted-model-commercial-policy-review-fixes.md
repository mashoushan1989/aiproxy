# Promoted Model Commercial Policy Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix review findings that can make promoted commercial pricing diverge from actual billing or frontend contracts.

**Architecture:** Centralize effective promoted entry and price resolution in enterprise quota, expose a controller-level price wrapper so every endpoint-specific request price can be commercially resolved after its base price is calculated, and normalize API DTOs so frontend sees `ModelPrice` arrays instead of storage-only commercial strings. Make mutation plus audit writes transactional.

**Tech Stack:** Go enterprise build tags, GORM, Gin handlers, existing `model.Price` billing structs, React/TypeScript frontend.

---

### Task 1: Effective Promoted Entry Selection

**Files:**
- Modify: `core/enterprise/quota/promoted_price.go`
- Modify: `core/enterprise/quota/promoted_price_test.go`
- Modify: `core/enterprise/access_info.go`

- [ ] **Step 1: Add failing test for inactive first row**

Add a test where the highest-priority promoted row is future-dated and a later row is active. `ResolvePromotedModelPrice` must pick the active row, not fallback.

Run: `cd core && go test -tags enterprise ./enterprise/quota -run TestResolvePromotedModelPriceSkipsInactiveRows -count=1`
Expected before fix: FAIL with fallback price selected.

- [ ] **Step 2: Implement shared effective entry lookup**

Create `activePromotedModelEntries(policyID int, modelName string, now time.Time) ([]entmodels.PromotedModelPolicy, error)` and make `ResolvePromotedModelPrice` iterate active rows before choosing the first sorted match.

- [ ] **Step 3: Reuse active lookup in My Access**

Replace duplicate active filtering in `core/enterprise/access_info.go` with a quota helper or matching query behavior so My Access and billing choose the same row.

- [ ] **Step 4: Verify**

Run: `cd core && go test -tags enterprise ./enterprise/quota ./enterprise -run 'TestResolvePromotedModelPrice|TestGetMyAccess' -count=1`
Expected: PASS.

### Task 2: Apply Commercial Resolver To Endpoint-Specific Prices

**Files:**
- Modify: `core/controller/relay-controller.go`
- Test: Add/update controller or quota tests where practical.

- [ ] **Step 1: Add failing test for wrapper behavior**

Add a focused unit test for a helper that takes a base request price function and applies enterprise commercial pricing unless group override is present.

Run: `cd core && go test ./controller -run TestResolveRequestPriceAppliesEnterpriseResolver -count=1`
Expected before fix: FAIL or helper missing.

- [ ] **Step 2: Implement helper**

Introduce `resolveRequestPrice(c, mc, basePriceFunc)` in `relay-controller.go`. It should:
- compute endpoint-specific base price using the existing function when present;
- preserve `GroupModelConfig.OverridePrice` precedence;
- call `middleware.EnterprisePriceResolver(group, mc.Model, basePrice)` when configured.

- [ ] **Step 3: Wire all relay modes**

In `relay`, replace direct `relayController.GetRequestPrice(c, mc)` with the helper so images/edits and future endpoint-specific functions share the commercial resolver.

- [ ] **Step 4: Verify**

Run: `cd core && go test ./controller -count=1`
Expected: PASS.

### Task 3: Price-Based Quota Blocking Uses Effective Price

**Files:**
- Modify: `core/enterprise/quota/hook.go`
- Test: `core/enterprise/quota/hook_test.go`

- [ ] **Step 1: Add failing test**

Add a test proving a promoted discount below threshold is not blocked when the base `ModelConfig.Price` is above threshold.

Run: `cd core && go test -tags enterprise ./enterprise/quota -run TestApplyPolicyTiersUsesPromotedPriceForPriceBlocking -count=1`
Expected before fix: FAIL because base price blocks.

- [ ] **Step 2: Implement effective price helper**

Add a quota helper that resolves price for a group/model using group override first, promoted commercial second, fallback third. Use it in `applyPolicyTiers` or call path where group context is available.

- [ ] **Step 3: Verify**

Run: `cd core && go test -tags enterprise ./enterprise/quota -count=1`
Expected: PASS.

### Task 4: Normalize Promoted Model API Price DTO

**Files:**
- Modify: `core/enterprise/quota/promoted_model_handler.go`
- Modify: `core/enterprise/quota/promoted_model_handler_test.go`
- Modify: `web/src/api/enterprise.ts` only if type adjustment is needed.

- [ ] **Step 1: Add failing handler test**

Create/list a promoted policy with conditional prices and assert JSON response has `override_price.conditional_prices` as an array, not a JSON string.

Run: `cd core && go test -tags enterprise ./enterprise/quota -run TestPromotedModelHandlersReturnModelPriceDTO -count=1`
Expected before fix: FAIL because storage `CommercialPrice.ConditionalPrices` serializes as string.

- [ ] **Step 2: Add response DTO**

Create response structs in handler/service layer that expose `base_price` and `override_price` as `model.Price`, while storage continues using `CommercialPrice`.

- [ ] **Step 3: Verify frontend contract**

Run: `cd web && pnpm run build`
Expected: PASS.

### Task 5: Transactional Mutations And Audit

**Files:**
- Modify: `core/enterprise/quota/promoted_model_policy.go`
- Test: `core/enterprise/quota/promoted_model_policy_test.go`

- [ ] **Step 1: Add transactional audit failure test**

Inject a transaction path or invalid audit write scenario that proves mutation rolls back when audit write fails.

Run: `cd core && go test -tags enterprise ./enterprise/quota -run TestPromotedModelPolicyAuditFailureRollsBack -count=1`
Expected before fix: FAIL because mutation persists.

- [ ] **Step 2: Wrap create/update/delete/rollback in DB transactions**

Use `model.DB.Transaction(func(tx *gorm.DB) error { ... })`, pass `tx` into validation/query/save/audit helpers, and return only committed state.

- [ ] **Step 3: Verify**

Run: `cd core && go test -tags enterprise ./enterprise/quota -count=1`
Expected: PASS.

### Task 6: Final Verification

**Files:** none.

- [ ] Run: `cd core && go test ./controller -count=1`
- [ ] Run: `cd core && go test -tags enterprise ./enterprise/quota ./enterprise -count=1`
- [ ] Run: `cd web && pnpm run lint`
- [ ] Run: `cd web && pnpm run build`
- [ ] Commit final fixes with a focused message.

