# Promoted Model Pricing Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make promoted model commercial pricing explicit, searchable, and user-visible by adding discount/manual pricing modes, model candidates, and display/sort linkage.

**Architecture:** The backend remains the source of truth for commercial prices. `pricing_mode` records business intent, while `override_price` remains the final billing snapshot consumed by relay and quota logic. The frontend presents a searchable model selector and mutually exclusive pricing modes; user-facing access pages consume display name, badge, and sort metadata.

**Tech Stack:** Go enterprise backend with GORM/Gin tests, React/TypeScript admin UI, pnpm lint/build verification.

---

## File Map

- Modify `core/enterprise/models/promoted_model_policy.go`
  Add `PricingMode` constants and model field.
- Modify `core/enterprise/quota/promoted_model_policy.go`
  Validate pricing mode, compute discount snapshots, preserve lock checks, and keep manual mode compatible.
- Modify `core/enterprise/quota/promoted_model_handler.go`
  Return `pricing_mode`, add candidate response structs and handler.
- Modify `core/enterprise/quota/register.go`
  Register promoted model candidate endpoint.
- Modify `core/enterprise/access_info.go`
  Expose `display_name` and promoted `sort_order` to my-access model list.
- Modify `core/enterprise/models/migrate.go`
  Let AutoMigrate add `pricing_mode`; add guarded backfill if project migration helpers require it.
- Modify tests:
  - `core/enterprise/quota/promoted_model_policy_test.go`
  - `core/enterprise/quota/promoted_model_handler_test.go`
  - `core/enterprise/access_info_test.go`
- Modify `web/src/api/enterprise.ts`
  Add `pricing_mode`, candidate API types, and access model display/sort fields.
- Modify `web/src/pages/enterprise/quota.tsx`
  Replace free text model input with searchable selector and add pricing mode UI.
- Modify `web/src/pages/enterprise/my-access.tsx`
  Render display name and sort promoted models by configured order.
- Modify locale files:
  - `web/public/locales/zh/translation.json`
  - `web/public/locales/en/translation.json`

---

### Task 1: Backend Pricing Mode and Discount Snapshot

**Files:**
- Modify: `core/enterprise/models/promoted_model_policy.go`
- Modify: `core/enterprise/quota/promoted_model_policy.go`
- Test: `core/enterprise/quota/promoted_model_policy_test.go`

- [ ] **Step 1: Write failing tests for discount and manual mode**

Add tests to `core/enterprise/quota/promoted_model_policy_test.go`:

```go
func TestCreatePromotedModelPolicyDiscountModeComputesOverridePrice(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	entry, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		PricingMode:   entmodels.PromotedModelPricingModeDiscount,
		DiscountRate:  0.4,
		PriceLocked:   true,
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create promoted model: %v", err)
	}

	price, err := modelPriceFromCommercialPrice(entry.OverridePrice)
	if err != nil {
		t.Fatalf("decode override price: %v", err)
	}
	if got, want := float64(price.InputPrice), 0.0000058; got != want {
		t.Fatalf("input price = %.10f, want %.10f", got, want)
	}
	if got, want := float64(price.OutputPrice), 0.0000348; got != want {
		t.Fatalf("output price = %.10f, want %.10f", got, want)
	}
	if entry.PricingMode != entmodels.PromotedModelPricingModeDiscount {
		t.Fatalf("pricing mode = %q", entry.PricingMode)
	}
}

func TestCreatePromotedModelPolicyManualModeKeepsOverridePrice(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	entry, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		PricingMode:   entmodels.PromotedModelPricingModeManual,
		OverridePrice: model.Price{
			InputPrice:      model.ZeroNullFloat64(0.000001),
			InputPriceUnit:  model.ZeroNullInt64(1),
			OutputPrice:     model.ZeroNullFloat64(0.000002),
			OutputPriceUnit: model.ZeroNullInt64(1),
		},
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create promoted model: %v", err)
	}

	price, err := modelPriceFromCommercialPrice(entry.OverridePrice)
	if err != nil {
		t.Fatalf("decode override price: %v", err)
	}
	if price.InputPrice != model.ZeroNullFloat64(0.000001) {
		t.Fatalf("manual input price = %v", price.InputPrice)
	}
	if entry.DiscountRate != 0 {
		t.Fatalf("manual discount rate = %v, want 0", entry.DiscountRate)
	}
}

func TestUpdatePromotedModelPolicyRejectsLockedDiscountChangeWithoutForce(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	entry, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		PricingMode:   entmodels.PromotedModelPricingModeDiscount,
		DiscountRate:  0.5,
		PriceLocked:   true,
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create promoted model: %v", err)
	}

	_, err = UpdatePromotedModelEntry(entry.ID, UpdatePromotedModelEntryRequest{
		DisplayName:   entry.DisplayName,
		Enabled:       true,
		PricingMode:   entmodels.PromotedModelPricingModeDiscount,
		DiscountRate:  0.4,
		PriceLocked:   true,
	}, AuditOperator{ID: "admin"}, false)
	if !errors.Is(err, ErrPromotedModelPriceLocked) {
		t.Fatalf("err = %v, want ErrPromotedModelPriceLocked", err)
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestCreatePromotedModelPolicy(DiscountModeComputesOverridePrice|ManualModeKeepsOverridePrice)|TestUpdatePromotedModelPolicyRejectsLockedDiscountChangeWithoutForce'
```

Expected: FAIL because `PricingMode` constants/fields and discount calculation do not exist.

- [ ] **Step 3: Add pricing mode constants and field**

In `core/enterprise/models/promoted_model_policy.go`, add:

```go
const (
	PromotedModelPricingModeManual   = "manual"
	PromotedModelPricingModeDiscount = "discount"
)
```

Add field to `PromotedModelPolicy` after `OverridePrice`:

```go
PricingMode string `json:"pricing_mode" gorm:"size:32;default:manual"`
```

- [ ] **Step 4: Implement pricing mode normalization and discount calculation**

In `core/enterprise/quota/promoted_model_policy.go`, add `PricingMode string` to create/update requests.

Add helpers:

```go
func normalizePromotedPricingMode(mode string) string {
	switch mode {
	case entmodels.PromotedModelPricingModeDiscount:
		return entmodels.PromotedModelPricingModeDiscount
	default:
		return entmodels.PromotedModelPricingModeManual
	}
}

func discountedPrice(base model.Price, rate float64) (model.Price, error) {
	if rate <= 0 || rate > 1 {
		return model.Price{}, errors.New("discount_rate must be greater than 0 and less than or equal to 1")
	}
	price := base
	price.PerRequestPrice = model.ZeroNullFloat64(float64(base.PerRequestPrice) * rate)
	price.InputPrice = model.ZeroNullFloat64(float64(base.InputPrice) * rate)
	price.ImageInputPrice = model.ZeroNullFloat64(float64(base.ImageInputPrice) * rate)
	price.AudioInputPrice = model.ZeroNullFloat64(float64(base.AudioInputPrice) * rate)
	price.OutputPrice = model.ZeroNullFloat64(float64(base.OutputPrice) * rate)
	price.ImageOutputPrice = model.ZeroNullFloat64(float64(base.ImageOutputPrice) * rate)
	price.ThinkingModeOutputPrice = model.ZeroNullFloat64(float64(base.ThinkingModeOutputPrice) * rate)
	price.CachedPrice = model.ZeroNullFloat64(float64(base.CachedPrice) * rate)
	price.CacheCreationPrice = model.ZeroNullFloat64(float64(base.CacheCreationPrice) * rate)
	price.WebSearchPrice = model.ZeroNullFloat64(float64(base.WebSearchPrice) * rate)
	for i := range price.ConditionalPrices {
		price.ConditionalPrices[i].Price.PerRequestPrice = model.ZeroNullFloat64(float64(price.ConditionalPrices[i].Price.PerRequestPrice) * rate)
		price.ConditionalPrices[i].Price.InputPrice = model.ZeroNullFloat64(float64(price.ConditionalPrices[i].Price.InputPrice) * rate)
		price.ConditionalPrices[i].Price.OutputPrice = model.ZeroNullFloat64(float64(price.ConditionalPrices[i].Price.OutputPrice) * rate)
		price.ConditionalPrices[i].Price.CachedPrice = model.ZeroNullFloat64(float64(price.ConditionalPrices[i].Price.CachedPrice) * rate)
		price.ConditionalPrices[i].Price.CacheCreationPrice = model.ZeroNullFloat64(float64(price.ConditionalPrices[i].Price.CacheCreationPrice) * rate)
	}
	return price, nil
}

func promotedOverridePrice(basePrice model.Price, requestedOverride model.Price, pricingMode string, discountRate float64) (model.Price, float64, error) {
	if pricingMode == entmodels.PromotedModelPricingModeDiscount {
		price, err := discountedPrice(basePrice, discountRate)
		return price, discountRate, err
	}
	if err := requestedOverride.ValidateConditionalPrices(); err != nil {
		return model.Price{}, 0, err
	}
	return requestedOverride, 0, nil
}
```

Use this helper in create/update before converting `OverridePrice`.

- [ ] **Step 5: Run tests and verify GREEN**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestCreatePromotedModelPolicy(DiscountModeComputesOverridePrice|ManualModeKeepsOverridePrice)|TestUpdatePromotedModelPolicyRejectsLockedDiscountChangeWithoutForce'
```

Expected: PASS.

---

### Task 2: Candidate Search API

**Files:**
- Modify: `core/enterprise/quota/promoted_model_handler.go`
- Modify: `core/enterprise/quota/register.go`
- Test: `core/enterprise/quota/promoted_model_handler_test.go`

- [ ] **Step 1: Write failing handler test**

Add to `core/enterprise/quota/promoted_model_handler_test.go`:

```go
func TestPromotedModelCandidatesSearchEnabledModels(t *testing.T) {
	router, _ := setupPromotedModelHandlerRouter(t)

	resp := requestJSON(t, router, http.MethodGet, "/api/enterprise/quota/promoted-model-candidates?keyword=gpt", nil)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}

	var out struct {
		Success bool `json:"success"`
		Data struct {
			Candidates []struct {
				Model    string      `json:"model"`
				Channels []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"channels"`
				BasePrice model.Price `json:"base_price"`
			} `json:"candidates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Data.Candidates) == 0 {
		t.Fatalf("expected candidates")
	}
	if out.Data.Candidates[0].Model != "pa/gpt-5.5" {
		t.Fatalf("model = %q", out.Data.Candidates[0].Model)
	}
	if out.Data.Candidates[0].BasePrice.InputPrice == 0 {
		t.Fatalf("expected base price")
	}
}
```

- [ ] **Step 2: Run test and verify RED**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run TestPromotedModelCandidatesSearchEnabledModels
```

Expected: FAIL with 404 or missing handler.

- [ ] **Step 3: Implement candidate handler**

Add handler that reads `keyword` and optional `channel_id`, iterates `model.LoadModelCaches().EnabledModelConfigsMap`, filters by substring, and returns at most 50 candidates with `model`, `type`, `type_name`, `base_price`, and matching channels from `EnabledModel2ChannelsBySet`.

Register under quota view group:

```go
group.GET("/quota/promoted-model-candidates", ListPromotedModelCandidates)
```

- [ ] **Step 4: Run test and verify GREEN**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run TestPromotedModelCandidatesSearchEnabledModels
```

Expected: PASS.

---

### Task 3: My Access Display Name and Sort Linkage

**Files:**
- Modify: `core/enterprise/access_info.go`
- Test: `core/enterprise/access_info_test.go`
- Modify: `web/src/pages/enterprise/my-access.tsx`
- Modify: `web/src/api/enterprise.ts`

- [ ] **Step 1: Write failing backend test**

Extend existing promoted metadata test in `core/enterprise/access_info_test.go` to assert:

```go
if got := promoted.DisplayName; got != "GPT-5.5 推荐版" {
	t.Fatalf("display name = %q", got)
}
if got := promoted.SortOrder; got != 5 {
	t.Fatalf("sort order = %d", got)
}
```

Seed the promoted policy with:

```go
DisplayName: "GPT-5.5 推荐版",
SortOrder:   5,
```

- [ ] **Step 2: Run test and verify RED**

Run:

```bash
cd core && go test -tags enterprise ./enterprise -run TestGetMyAccess_IncludesPromotedModelMetadataAndKeepsNormalModels
```

Expected: FAIL because response does not expose display name or sort order.

- [ ] **Step 3: Add response fields and sorting**

Add to `ModelAccessInfo`:

```go
DisplayName string `json:"display_name,omitempty"`
SortOrder   int    `json:"sort_order,omitempty"`
```

When promoted, set:

```go
info.DisplayName = promotedEntry.DisplayName
info.SortOrder = promotedEntry.SortOrder
```

Sort owner models:

```go
sort.Slice(models, func(i, j int) bool {
	if models[i].IsPromoted != models[j].IsPromoted {
		return models[i].IsPromoted
	}
	if models[i].IsPromoted && models[i].SortOrder != models[j].SortOrder {
		return models[i].SortOrder < models[j].SortOrder
	}
	return models[i].Model < models[j].Model
})
```

- [ ] **Step 4: Run backend test and verify GREEN**

Run:

```bash
cd core && go test -tags enterprise ./enterprise -run TestGetMyAccess_IncludesPromotedModelMetadataAndKeepsNormalModels
```

Expected: PASS.

- [ ] **Step 5: Update frontend access model type and rendering**

In `web/src/api/enterprise.ts`, add to access model:

```ts
display_name?: string
sort_order?: number
```

In `web/src/pages/enterprise/my-access.tsx`, render `m.display_name || m.model` as primary text and keep `m.model` as mono secondary text when display name exists.

---

### Task 4: Admin UI Searchable Model Selector and Pricing Modes

**Files:**
- Modify: `web/src/api/enterprise.ts`
- Modify: `web/src/pages/enterprise/quota.tsx`
- Modify: `web/public/locales/zh/translation.json`
- Modify: `web/public/locales/en/translation.json`

- [ ] **Step 1: Add API types**

In `web/src/api/enterprise.ts`, add:

```ts
export type PromotedPricingMode = "manual" | "discount"

export interface PromotedModelCandidate {
    model: string
    type: number
    type_name: string
    base_price: ModelPrice
    channels: Array<{ id: number; name: string; type: number }>
}
```

Add `pricing_mode: PromotedPricingMode` to `PromotedModelPolicy` and `PromotedModelPolicyInput`.

Add API method:

```ts
listPromotedModelCandidates: (params?: { keyword?: string; channel_id?: number }): Promise<{ candidates: PromotedModelCandidate[] }> => {
    const search = new URLSearchParams()
    if (params?.keyword) search.set("keyword", params.keyword)
    if (params?.channel_id) search.set("channel_id", String(params.channel_id))
    const suffix = search.toString() ? `?${search.toString()}` : ""
    return get<{ candidates: PromotedModelCandidate[] }>(`/enterprise/quota/promoted-model-candidates${suffix}`)
}
```

- [ ] **Step 2: Update form state**

Defaults:

```ts
pricing_mode: "discount",
discount_rate: 0.4,
```

Editing fallback:

```ts
pricing_mode: entry.pricing_mode || (entry.discount_rate > 0 ? "discount" : "manual")
```

- [ ] **Step 3: Add candidate query and selector**

Use `useQuery` with key `["enterprise", "quota-promoted-model-candidates", form.model, form.channel_id]`, enabled when dialog open.

Replace model `Input` with an input plus dropdown list. On selecting candidate:

```ts
setForm({
  ...form,
  model: candidate.model,
  channel_id: candidate.channels[0]?.id || form.channel_id,
  override_price: form.pricing_mode === "discount"
    ? computeDiscountPrice(candidate.base_price, form.discount_rate || 0)
    : form.override_price,
})
```

- [ ] **Step 4: Add mutually exclusive pricing UI**

Add segmented/radio controls:

```tsx
<Tabs value={form.pricing_mode} onValueChange={(v) => switchPricingMode(v as PromotedPricingMode)}>
  <TabsList>
    <TabsTrigger value="discount">{t("enterprise.quota.promoted.pricingModeDiscount" as never)}</TabsTrigger>
    <TabsTrigger value="manual">{t("enterprise.quota.promoted.pricingModeManual" as never)}</TabsTrigger>
  </TabsList>
</Tabs>
```

Discount mode:
- editable percent input
- read-only price preview

Manual mode:
- editable input/output fields
- advanced price fields remain future-safe but do not need a full editor in this task.

- [ ] **Step 5: Save payload**

Payload must include:

```ts
pricing_mode: form.pricing_mode,
discount_rate: form.pricing_mode === "discount" ? form.discount_rate : 0,
override_price: form.override_price,
```

- [ ] **Step 6: Update translations**

Add keys:

```json
"pricingMode": "计价方式",
"pricingModeDiscount": "按基础价格折扣",
"pricingModeManual": "手动指定价格",
"discountPercent": "折扣比例",
"discountPreview": "折扣后价格预览",
"modelSearchPlaceholder": "搜索当前可用模型",
"basePricePreview": "基础价格预览"
```

Add English equivalents.

---

### Task 5: Verification

**Files:** all changed files.

- [ ] **Step 1: Run backend targeted tests**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota ./enterprise ./controller
```

Expected: PASS.

- [ ] **Step 2: Run frontend checks**

Run:

```bash
cd web && pnpm run lint
cd web && pnpm run build
```

Expected: PASS. Existing warnings only if already present; do not introduce new errors.

- [ ] **Step 3: Run diff hygiene**

Run:

```bash
git diff --check
```

Expected: no output.

- [ ] **Step 4: Manual preview**

With local backend/frontend running:
- Open quota policy promoted tab.
- Search `gpt`.
- Select `pa/gpt-5.5`.
- Confirm discount mode computes price preview.
- Switch to manual mode and confirm price fields become editable.
- Save and confirm my-access shows display name, badge, commercial lock, and promoted sorting.

