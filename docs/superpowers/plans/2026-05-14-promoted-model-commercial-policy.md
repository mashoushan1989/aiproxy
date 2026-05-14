# Promoted Model Commercial Policy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build single-model promoted commercial discounts for enterprise quota policies, with visible commercial locks, audit history, rollback, My Access recommendations, and request-time billing resolution for new requests only.

**Architecture:** Add separate enterprise promoted-model policy and audit tables, expose quota-scoped APIs, and resolve commercial price dynamically during request pricing after group override prices. The feature is recommendation-first: reference channel metadata validates/admin-context only and never forces routing.

**Tech Stack:** Go enterprise build tag, GORM, Gin, existing `model.Price` schema, React/TypeScript, TanStack Query, existing quota permissions and i18n.

---

## File Structure

- Create `core/enterprise/models/promoted_model_policy.go`: GORM models, table names, audit action constants, active-window helper, and request structs shared by quota handlers.
- Modify `core/enterprise/models/migrate.go`: add promoted model policy and audit tables to enterprise AutoMigrate.
- Create `core/enterprise/quota/promoted_model_policy.go`: CRUD, validation, audit writing, rollback, and query helpers for promoted model entries.
- Create `core/enterprise/quota/promoted_model_policy_test.go`: model/service-level tests for validation, audit, lock behavior, and rollback.
- Modify `core/enterprise/quota/register.go`: register promoted model APIs under existing quota permissions.
- Create `core/enterprise/quota/promoted_model_handler.go`: Gin handlers for list/create/update/delete/rollback/audit.
- Create `core/enterprise/quota/promoted_model_handler_test.go`: handler tests for success and permission-independent validation paths.
- Modify `core/controller/relay-controller.go`: wrap request price resolution with enterprise promoted commercial price resolver.
- Modify `core/enterprise/quota/promoted_price.go`: add effective promoted price resolver used by relay.
- Create `core/enterprise/quota/promoted_price_test.go`: request-time pricing precedence tests.
- Modify `core/enterprise/access_info.go`: include promoted model metadata in My Access model groups without hiding normal models.
- Modify `web/src/api/enterprise.ts`: add promoted model API types and methods.
- Modify `web/src/pages/enterprise/quota.tsx`: add promoted model management tab, single-entry forms, lock controls, audit drawer/dialog.
- Modify `web/src/pages/enterprise/my-access.tsx`: display promoted models first with badges and discounted prices.
- Modify `web/public/locales/en/translation.json` and `web/public/locales/zh/translation.json`: add UI labels.

## Task 1: Add Promoted Model Data Models And Migration

**Files:**
- Create: `core/enterprise/models/promoted_model_policy.go`
- Modify: `core/enterprise/models/migrate.go`
- Test: compile via `go test -tags enterprise ./enterprise/models`

- [ ] **Step 1: Create promoted model models**

Add `core/enterprise/models/promoted_model_policy.go`:

```go
//go:build enterprise

package models

import (
	"time"

	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

const (
	PromotedModelAuditActionCreate              = "create"
	PromotedModelAuditActionUpdate              = "update"
	PromotedModelAuditActionEnable              = "enable"
	PromotedModelAuditActionDisable             = "disable"
	PromotedModelAuditActionPriceLock           = "price_lock"
	PromotedModelAuditActionPriceUnlock         = "price_unlock"
	PromotedModelAuditActionPriceChange         = "price_change"
	PromotedModelAuditActionForceLockedOverride = "force_locked_override"
	PromotedModelAuditActionDelete              = "delete"
	PromotedModelAuditActionRollback            = "rollback"
)

type PromotedModelPolicy struct {
	ID              int            `json:"id"                gorm:"primaryKey"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-"                 gorm:"index"`
	QuotaPolicyID   int            `json:"quota_policy_id"   gorm:"index;not null"`
	QuotaPolicy     *QuotaPolicy   `json:"quota_policy"      gorm:"foreignKey:QuotaPolicyID"`
	Model           string         `json:"model"             gorm:"size:191;index;not null"`
	ChannelID       int            `json:"channel_id"        gorm:"default:0"`
	DisplayName     string         `json:"display_name"      gorm:"size:191"`
	RecommendBadge  string         `json:"recommend_badge"   gorm:"size:64"`
	SortOrder       int            `json:"sort_order"        gorm:"default:0"`
	Enabled         bool           `json:"enabled"           gorm:"default:true"`
	BasePrice       model.Price    `json:"base_price"        gorm:"embedded;embeddedPrefix:base_"`
	OverridePrice   model.Price    `json:"override_price"    gorm:"embedded;embeddedPrefix:override_"`
	DiscountRate    float64        `json:"discount_rate"     gorm:"default:0"`
	PriceLocked     bool           `json:"price_locked"      gorm:"default:false"`
	EffectiveAt     *time.Time     `json:"effective_at"      gorm:"index"`
	ExpiresAt       *time.Time     `json:"expires_at"        gorm:"index"`
	Version         int            `json:"version"           gorm:"default:1"`
	CreatedBy       string         `json:"created_by"        gorm:"size:191"`
	UpdatedBy       string         `json:"updated_by"        gorm:"size:191"`
}

func (PromotedModelPolicy) TableName() string {
	return "enterprise_promoted_model_policies"
}

func (p PromotedModelPolicy) ActiveAt(now time.Time) bool {
	if !p.Enabled {
		return false
	}
	if p.EffectiveAt != nil && now.Before(*p.EffectiveAt) {
		return false
	}
	if p.ExpiresAt != nil && !now.Before(*p.ExpiresAt) {
		return false
	}
	return true
}

type PromotedModelPolicyAudit struct {
	ID                    int            `json:"id"                       gorm:"primaryKey"`
	CreatedAt             time.Time      `json:"created_at"`
	DeletedAt             gorm.DeletedAt `json:"-"                        gorm:"index"`
	PromotedModelPolicyID int            `json:"promoted_model_policy_id" gorm:"index;not null"`
	QuotaPolicyID         int            `json:"quota_policy_id"          gorm:"index;not null"`
	Action                string         `json:"action"                   gorm:"size:64;not null"`
	Before                string         `json:"before"                   gorm:"type:text"`
	After                 string         `json:"after"                    gorm:"type:text"`
	Summary               string         `json:"summary"                  gorm:"size:512"`
	OperatorID            string         `json:"operator_id"              gorm:"size:191"`
	OperatorName          string         `json:"operator_name"            gorm:"size:191"`
}

func (PromotedModelPolicyAudit) TableName() string {
	return "enterprise_promoted_model_policy_audits"
}
```

- [ ] **Step 2: Register migrations**

In `core/enterprise/models/migrate.go`, add these models to `EnterpriseAutoMigrate` after `QuotaAlertHistory{}`:

```go
		&PromotedModelPolicy{},
		&PromotedModelPolicyAudit{},
```

- [ ] **Step 3: Verify models compile**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/models
```

Expected: package compiles and tests pass.

- [ ] **Step 4: Commit**

```bash
git add core/enterprise/models/promoted_model_policy.go core/enterprise/models/migrate.go
git commit -m "feat: add promoted model policy tables"
```

## Task 2: Add Service Layer With Audit, Locks, And Rollback

**Files:**
- Create: `core/enterprise/quota/promoted_model_policy.go`
- Create: `core/enterprise/quota/promoted_model_policy_test.go`

- [ ] **Step 1: Write service tests**

Create `core/enterprise/quota/promoted_model_policy_test.go` with tests:

```go
//go:build enterprise

package quota

import (
	"testing"
	"time"

	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPromotedModelPolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.ModelConfig{},
		&model.Channel{},
		&entmodels.QuotaPolicy{},
		&entmodels.PromotedModelPolicy{},
		&entmodels.PromotedModelPolicyAudit{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	model.DB = db
	return db
}

func seedPromotedModelFixtures(t *testing.T, db *gorm.DB) entmodels.QuotaPolicy {
	t.Helper()
	policy := entmodels.QuotaPolicy{Name: "Engineering"}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	if err := db.Create(&model.ModelConfig{
		Model: "pa/gpt-5.5",
		Type:  mode.ChatCompletions,
		Price: model.Price{
			InputPrice:      model.ZeroNullFloat64(0.00003625),
			InputPriceUnit:  model.ZeroNullInt64(1),
			OutputPrice:     model.ZeroNullFloat64(0.0002175),
			OutputPriceUnit: model.ZeroNullInt64(1),
		},
	}).Error; err != nil {
		t.Fatalf("seed model: %v", err)
	}
	return policy
}

func TestCreatePromotedModelPolicyDefaultsUnlockedAndAudits(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	entry, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID:  policy.ID,
		Model:          "pa/gpt-5.5",
		DisplayName:    "GPT-5.5",
		RecommendBadge: "Recommended",
		Enabled:        true,
		OverridePrice: model.Price{
			InputPrice:      model.ZeroNullFloat64(0.0000145),
			InputPriceUnit:  model.ZeroNullInt64(1),
			OutputPrice:     model.ZeroNullFloat64(0.000087),
			OutputPriceUnit: model.ZeroNullInt64(1),
		},
	}, AuditOperator{ID: "admin", Name: "Admin"})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if entry.PriceLocked {
		t.Fatalf("new entry should default unlocked")
	}
	if entry.BasePrice.InputPrice != model.ZeroNullFloat64(0.00003625) {
		t.Fatalf("base price snapshot not captured: %#v", entry.BasePrice)
	}

	var audits []entmodels.PromotedModelPolicyAudit
	if err := db.Find(&audits).Error; err != nil {
		t.Fatalf("load audits: %v", err)
	}
	if len(audits) != 1 || audits[0].Action != entmodels.PromotedModelAuditActionCreate {
		t.Fatalf("unexpected audits: %#v", audits)
	}
}

func TestUpdatePromotedModelPolicyRejectsLockedPriceWithoutForce(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)
	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		PriceLocked:   true,
		OverridePrice: model.Price{InputPrice: model.ZeroNullFloat64(0.1), InputPriceUnit: model.ZeroNullInt64(1)},
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	_, err := UpdatePromotedModelEntry(entry.ID, UpdatePromotedModelEntryRequest{
		OverridePrice: model.Price{InputPrice: model.ZeroNullFloat64(0.2), InputPriceUnit: model.ZeroNullInt64(1)},
	}, AuditOperator{ID: "admin"}, false)
	if err == nil {
		t.Fatalf("expected locked price update to fail")
	}
}

func TestRollbackPromotedModelPolicyCreatesNewVersion(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)
	entry, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID:  policy.ID,
		Model:          "pa/gpt-5.5",
		Enabled:        true,
		OverridePrice:  model.Price{InputPrice: model.ZeroNullFloat64(0.1), InputPriceUnit: model.ZeroNullInt64(1)},
		EffectiveAt:    ptrTime(time.Now().Add(-time.Hour)),
		RecommendBadge: "A",
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	updated, err := UpdatePromotedModelEntry(entry.ID, UpdatePromotedModelEntryRequest{
		RecommendBadge: "B",
		OverridePrice:  model.Price{InputPrice: model.ZeroNullFloat64(0.2), InputPriceUnit: model.ZeroNullInt64(1)},
	}, AuditOperator{ID: "admin"}, true)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("want version 2, got %d", updated.Version)
	}

	rolled, err := RollbackPromotedModelEntry(entry.ID, 1, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if rolled.Version != 3 || rolled.RecommendBadge != "A" {
		t.Fatalf("unexpected rollback result: %#v", rolled)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestCreatePromotedModelPolicy|TestUpdatePromotedModelPolicy|TestRollbackPromotedModelPolicy' -count=1
```

Expected: FAIL because service types/functions are undefined.

- [ ] **Step 3: Implement service layer**

Create `core/enterprise/quota/promoted_model_policy.go`:

```go
//go:build enterprise

package quota

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

var ErrPromotedModelPriceLocked = errors.New("promoted model commercial price is locked")

type AuditOperator struct {
	ID   string
	Name string
}

type CreatePromotedModelEntryRequest struct {
	QuotaPolicyID  int         `json:"quota_policy_id"`
	Model          string      `json:"model"`
	ChannelID      int         `json:"channel_id"`
	DisplayName    string      `json:"display_name"`
	RecommendBadge string      `json:"recommend_badge"`
	SortOrder      int         `json:"sort_order"`
	Enabled        bool        `json:"enabled"`
	OverridePrice  model.Price `json:"override_price"`
	DiscountRate   float64     `json:"discount_rate"`
	PriceLocked    bool        `json:"price_locked"`
	EffectiveAt    *time.Time  `json:"effective_at"`
	ExpiresAt      *time.Time  `json:"expires_at"`
}

type UpdatePromotedModelEntryRequest struct {
	DisplayName    string      `json:"display_name"`
	RecommendBadge string      `json:"recommend_badge"`
	SortOrder      int         `json:"sort_order"`
	Enabled        bool        `json:"enabled"`
	OverridePrice  model.Price `json:"override_price"`
	DiscountRate   float64     `json:"discount_rate"`
	PriceLocked    bool        `json:"price_locked"`
	EffectiveAt    *time.Time  `json:"effective_at"`
	ExpiresAt      *time.Time  `json:"expires_at"`
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func validatePromotedModelEntry(policyID int, modelName string, channelID int, overridePrice model.Price, effectiveAt, expiresAt *time.Time) (model.Price, error) {
	if policyID <= 0 {
		return model.Price{}, errors.New("quota_policy_id is required")
	}
	if modelName == "" {
		return model.Price{}, errors.New("model is required")
	}
	if effectiveAt != nil && expiresAt != nil && !expiresAt.After(*effectiveAt) {
		return model.Price{}, errors.New("expires_at must be after effective_at")
	}
	if err := overridePrice.ValidateConditionalPrices(); err != nil {
		return model.Price{}, err
	}
	var policy entmodels.QuotaPolicy
	if err := model.DB.First(&policy, policyID).Error; err != nil {
		return model.Price{}, fmt.Errorf("quota policy not found: %w", err)
	}
	var mc model.ModelConfig
	if err := model.DB.Where("model = ?", modelName).First(&mc).Error; err != nil {
		return model.Price{}, fmt.Errorf("model config not found: %w", err)
	}
	if channelID > 0 {
		var channel model.Channel
		if err := model.DB.First(&channel, channelID).Error; err != nil {
			return model.Price{}, fmt.Errorf("channel not found: %w", err)
		}
		found := false
		for _, m := range channel.Models {
			if m == modelName {
				found = true
				break
			}
		}
		if !found {
			return model.Price{}, errors.New("channel does not include model")
		}
	}
	return mc.Price, nil
}

func CreatePromotedModelEntry(req CreatePromotedModelEntryRequest, op AuditOperator) (*entmodels.PromotedModelPolicy, error) {
	basePrice, err := validatePromotedModelEntry(req.QuotaPolicyID, req.Model, req.ChannelID, req.OverridePrice, req.EffectiveAt, req.ExpiresAt)
	if err != nil {
		return nil, err
	}
	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID:  req.QuotaPolicyID,
		Model:          req.Model,
		ChannelID:      req.ChannelID,
		DisplayName:    req.DisplayName,
		RecommendBadge: req.RecommendBadge,
		SortOrder:      req.SortOrder,
		Enabled:        req.Enabled,
		BasePrice:      basePrice,
		OverridePrice:  req.OverridePrice,
		DiscountRate:   req.DiscountRate,
		PriceLocked:    req.PriceLocked,
		EffectiveAt:    req.EffectiveAt,
		ExpiresAt:      req.ExpiresAt,
		Version:        1,
		CreatedBy:      op.ID,
		UpdatedBy:      op.ID,
	}
	if err := model.DB.Create(&entry).Error; err != nil {
		return nil, err
	}
	if err := writePromotedModelAudit(&entry, entmodels.PromotedModelAuditActionCreate, nil, &entry, op, "created promoted model"); err != nil {
		return nil, err
	}
	return &entry, nil
}

func UpdatePromotedModelEntry(id int, req UpdatePromotedModelEntryRequest, op AuditOperator, overrideLocked bool) (*entmodels.PromotedModelPolicy, error) {
	var existing entmodels.PromotedModelPolicy
	if err := model.DB.First(&existing, id).Error; err != nil {
		return nil, err
	}
	if existing.PriceLocked && !overrideLocked && !samePrice(existing.OverridePrice, req.OverridePrice) {
		return nil, ErrPromotedModelPriceLocked
	}
	if _, err := validatePromotedModelEntry(existing.QuotaPolicyID, existing.Model, existing.ChannelID, req.OverridePrice, req.EffectiveAt, req.ExpiresAt); err != nil {
		return nil, err
	}
	before := existing
	existing.DisplayName = req.DisplayName
	existing.RecommendBadge = req.RecommendBadge
	existing.SortOrder = req.SortOrder
	existing.Enabled = req.Enabled
	existing.OverridePrice = req.OverridePrice
	existing.DiscountRate = req.DiscountRate
	existing.PriceLocked = req.PriceLocked
	existing.EffectiveAt = req.EffectiveAt
	existing.ExpiresAt = req.ExpiresAt
	existing.Version++
	existing.UpdatedBy = op.ID
	if err := model.DB.Save(&existing).Error; err != nil {
		return nil, err
	}
	action := entmodels.PromotedModelAuditActionUpdate
	if overrideLocked && before.PriceLocked && !samePrice(before.OverridePrice, existing.OverridePrice) {
		action = entmodels.PromotedModelAuditActionForceLockedOverride
	} else if before.PriceLocked != existing.PriceLocked {
		if existing.PriceLocked {
			action = entmodels.PromotedModelAuditActionPriceLock
		} else {
			action = entmodels.PromotedModelAuditActionPriceUnlock
		}
	} else if !samePrice(before.OverridePrice, existing.OverridePrice) {
		action = entmodels.PromotedModelAuditActionPriceChange
	}
	if err := writePromotedModelAudit(&existing, action, &before, &existing, op, "updated promoted model"); err != nil {
		return nil, err
	}
	return &existing, nil
}

func DeletePromotedModelEntry(id int, op AuditOperator) error {
	var existing entmodels.PromotedModelPolicy
	if err := model.DB.First(&existing, id).Error; err != nil {
		return err
	}
	if err := model.DB.Delete(&existing).Error; err != nil {
		return err
	}
	return writePromotedModelAudit(&existing, entmodels.PromotedModelAuditActionDelete, &existing, nil, op, "deleted promoted model")
}

func RollbackPromotedModelEntry(id int, version int, op AuditOperator) (*entmodels.PromotedModelPolicy, error) {
	var current entmodels.PromotedModelPolicy
	if err := model.DB.First(&current, id).Error; err != nil {
		return nil, err
	}
	var audits []entmodels.PromotedModelPolicyAudit
	if err := model.DB.Where("promoted_model_policy_id = ?", id).Order("id ASC").Find(&audits).Error; err != nil {
		return nil, err
	}
	var target *entmodels.PromotedModelPolicy
	for i := range audits {
		if audits[i].After == "" {
			continue
		}
		var snapshot entmodels.PromotedModelPolicy
		if err := json.Unmarshal([]byte(audits[i].After), &snapshot); err != nil {
			return nil, err
		}
		if snapshot.Version == version {
			target = &snapshot
			break
		}
	}
	if target == nil {
		return nil, gorm.ErrRecordNotFound
	}
	before := current
	current.DisplayName = target.DisplayName
	current.RecommendBadge = target.RecommendBadge
	current.SortOrder = target.SortOrder
	current.Enabled = target.Enabled
	current.OverridePrice = target.OverridePrice
	current.DiscountRate = target.DiscountRate
	current.PriceLocked = target.PriceLocked
	current.EffectiveAt = target.EffectiveAt
	current.ExpiresAt = target.ExpiresAt
	current.Version++
	current.UpdatedBy = op.ID
	if err := model.DB.Save(&current).Error; err != nil {
		return nil, err
	}
	if err := writePromotedModelAudit(&current, entmodels.PromotedModelAuditActionRollback, &before, &current, op, "rolled back promoted model"); err != nil {
		return nil, err
	}
	return &current, nil
}

func writePromotedModelAudit(entry *entmodels.PromotedModelPolicy, action string, before, after *entmodels.PromotedModelPolicy, op AuditOperator, summary string) error {
	beforeJSON := ""
	afterJSON := ""
	if before != nil {
		data, err := json.Marshal(before)
		if err != nil {
			return err
		}
		beforeJSON = string(data)
	}
	if after != nil {
		data, err := json.Marshal(after)
		if err != nil {
			return err
		}
		afterJSON = string(data)
	}
	return model.DB.Create(&entmodels.PromotedModelPolicyAudit{
		PromotedModelPolicyID: entry.ID,
		QuotaPolicyID:         entry.QuotaPolicyID,
		Action:                action,
		Before:                beforeJSON,
		After:                 afterJSON,
		Summary:               summary,
		OperatorID:            op.ID,
		OperatorName:          op.Name,
	}).Error
}

func samePrice(a, b model.Price) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
```

- [ ] **Step 4: Run service tests**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestCreatePromotedModelPolicy|TestUpdatePromotedModelPolicy|TestRollbackPromotedModelPolicy' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add core/enterprise/quota/promoted_model_policy.go core/enterprise/quota/promoted_model_policy_test.go
git commit -m "feat: manage promoted model policy entries"
```

## Task 3: Add Promoted Model APIs

**Files:**
- Modify: `core/enterprise/quota/register.go`
- Create: `core/enterprise/quota/promoted_model_handler.go`
- Create: `core/enterprise/quota/promoted_model_handler_test.go`

- [ ] **Step 1: Write handler tests**

Create `core/enterprise/quota/promoted_model_handler_test.go`:

```go
//go:build enterprise

package quota

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/stretchr/testify/require"
)

func setupPromotedModelRouter(t *testing.T) *gin.Engine {
	t.Helper()
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)
	require.NotZero(t, policy.ID)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/enterprise"), map[string]gin.HandlerFunc{
		"quota_manage_view":   func(c *gin.Context) { c.Next() },
		"quota_manage_manage": func(c *gin.Context) { c.Next() },
	})
	return r
}

func TestCreateAndListPromotedModelAPI(t *testing.T) {
	r := setupPromotedModelRouter(t)
	body := map[string]any{
		"model": "pa/gpt-5.5",
		"enabled": true,
		"override_price": map[string]any{
			"input_price": 0.0000145,
			"input_price_unit": 1,
			"output_price": 0.000087,
			"output_price_unit": 1,
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/enterprise/quota/policies/1/promoted-models", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/enterprise/quota/policies/1/promoted-models", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "pa/gpt-5.5")
}

func TestPromotedModelAPIValidatesChannelReference(t *testing.T) {
	r := setupPromotedModelRouter(t)
	require.NoError(t, model.DB.Create(&model.Channel{ID: 77, Name: "Other", Models: []string{"other-model"}}).Error)
	require.NoError(t, model.DB.Create(&model.ModelConfig{Model: "other-model", Type: mode.ChatCompletions}).Error)
	body := map[string]any{
		"model": "pa/gpt-5.5",
		"channel_id": 77,
		"enabled": true,
		"override_price": map[string]any{"input_price": 0.1, "input_price_unit": 1},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/enterprise/quota/policies/1/promoted-models", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestPromotedModelRollbackAPI(t *testing.T) {
	r := setupPromotedModelRouter(t)
	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID: 1,
		Model: "pa/gpt-5.5",
		Enabled: true,
		OverridePrice: model.Price{InputPrice: model.ZeroNullFloat64(0.1), InputPriceUnit: model.ZeroNullInt64(1)},
		Version: 1,
	}
	require.NoError(t, model.DB.Create(&entry).Error)
	require.NoError(t, writePromotedModelAudit(&entry, entmodels.PromotedModelAuditActionCreate, nil, &entry, AuditOperator{ID: "test"}, "created"))

	body := map[string]any{"version": 1}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/enterprise/quota/policies/1/promoted-models/1/rollback", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestCreateAndListPromotedModelAPI|TestPromotedModelAPI|TestPromotedModelRollbackAPI' -count=1
```

Expected: FAIL because routes/handlers are missing.

- [ ] **Step 3: Register routes**

Modify `core/enterprise/quota/register.go`:

```go
	policies.GET("/:id/promoted-models", ListPromotedModels)
	policies.GET("/:id/promoted-models/audit", ListPromotedModelAudits)
```

Add to `adminPolicies`:

```go
	adminPolicies.POST("/:id/promoted-models", CreatePromotedModel)
	adminPolicies.PUT("/:id/promoted-models/:entry_id", UpdatePromotedModel)
	adminPolicies.DELETE("/:id/promoted-models/:entry_id", DeletePromotedModel)
	adminPolicies.POST("/:id/promoted-models/:entry_id/rollback", RollbackPromotedModel)
```

- [ ] **Step 4: Implement handlers**

Create `core/enterprise/quota/promoted_model_handler.go` with:

```go
//go:build enterprise

package quota

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

func auditOperatorFromContext(c *gin.Context) AuditOperator {
	return AuditOperator{ID: c.GetString("username"), Name: c.GetString("username")}
}

func policyIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "invalid policy id")
		return 0, false
	}
	return id, true
}

func entryIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("entry_id"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "invalid promoted model id")
		return 0, false
	}
	return id, true
}

func ListPromotedModels(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}
	var entries []entmodels.PromotedModelPolicy
	if err := model.DB.Where("quota_policy_id = ?", policyID).Order("sort_order ASC, id DESC").Find(&entries).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	middleware.SuccessResponse(c, gin.H{"entries": entries})
}

func CreatePromotedModel(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}
	var req CreatePromotedModelEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	req.QuotaPolicyID = policyID
	entry, err := CreatePromotedModelEntry(req, auditOperatorFromContext(c))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	middleware.SuccessResponse(c, entry)
}

func UpdatePromotedModel(c *gin.Context) {
	entryID, ok := entryIDParam(c)
	if !ok {
		return
	}
	var req struct {
		UpdatePromotedModelEntryRequest
		OverrideLocked bool `json:"override_locked"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := UpdatePromotedModelEntry(entryID, req.UpdatePromotedModelEntryRequest, auditOperatorFromContext(c), req.OverrideLocked)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrPromotedModelPriceLocked) {
			status = http.StatusConflict
		}
		middleware.ErrorResponse(c, status, err.Error())
		return
	}
	middleware.SuccessResponse(c, entry)
}

func DeletePromotedModel(c *gin.Context) {
	entryID, ok := entryIDParam(c)
	if !ok {
		return
	}
	if err := DeletePromotedModelEntry(entryID, auditOperatorFromContext(c)); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	middleware.SuccessResponse(c, gin.H{"deleted": true})
}

func RollbackPromotedModel(c *gin.Context) {
	entryID, ok := entryIDParam(c)
	if !ok {
		return
	}
	var req struct {
		Version int `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := RollbackPromotedModelEntry(entryID, req.Version, auditOperatorFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			middleware.ErrorResponse(c, http.StatusNotFound, "rollback version not found")
			return
		}
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	middleware.SuccessResponse(c, entry)
}

func ListPromotedModelAudits(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}
	var audits []entmodels.PromotedModelPolicyAudit
	if err := model.DB.Where("quota_policy_id = ?", policyID).Order("id DESC").Find(&audits).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	middleware.SuccessResponse(c, gin.H{"audits": audits})
}
```

- [ ] **Step 5: Run handler tests**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestCreateAndListPromotedModelAPI|TestPromotedModelAPI|TestPromotedModelRollbackAPI' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add core/enterprise/quota/register.go core/enterprise/quota/promoted_model_handler.go core/enterprise/quota/promoted_model_handler_test.go
git commit -m "feat: add promoted model policy APIs"
```

## Task 4: Add Request-Time Price Resolver

**Files:**
- Create: `core/enterprise/quota/promoted_price.go`
- Create: `core/enterprise/quota/promoted_price_test.go`
- Modify: `core/controller/relay-controller.go`

- [ ] **Step 1: Write resolver tests**

Create `core/enterprise/quota/promoted_price_test.go`:

```go
//go:build enterprise

package quota

import (
	"context"
	"testing"
	"time"

	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
)

func TestResolvePromotedCommercialPriceUsesActiveEntry(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)
	now := time.Now()
	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID: policy.ID,
		Model: "pa/gpt-5.5",
		Enabled: true,
		EffectiveAt: ptrTime(now.Add(-time.Hour)),
		ExpiresAt: ptrTime(now.Add(time.Hour)),
		OverridePrice: model.Price{InputPrice: model.ZeroNullFloat64(0.0000145), InputPriceUnit: model.ZeroNullInt64(1)},
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	price, ok, err := ResolvePromotedCommercialPrice(context.Background(), policy.ID, "pa/gpt-5.5", now)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !ok || price.InputPrice != model.ZeroNullFloat64(0.0000145) {
		t.Fatalf("unexpected price: ok=%v price=%#v", ok, price)
	}
}

func TestResolvePromotedCommercialPriceIgnoresExpiredEntry(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)
	now := time.Now()
	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID: policy.ID,
		Model: "pa/gpt-5.5",
		Enabled: true,
		ExpiresAt: ptrTime(now.Add(-time.Minute)),
		OverridePrice: model.Price{InputPrice: model.ZeroNullFloat64(0.0000145), InputPriceUnit: model.ZeroNullInt64(1)},
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	_, ok, err := ResolvePromotedCommercialPrice(context.Background(), policy.ID, "pa/gpt-5.5", now)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if ok {
		t.Fatalf("expired entry should not resolve")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestResolvePromotedCommercialPrice' -count=1
```

Expected: FAIL because resolver is undefined.

- [ ] **Step 3: Implement resolver**

Create `core/enterprise/quota/promoted_price.go`:

```go
//go:build enterprise

package quota

import (
	"context"
	"time"

	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

func ResolvePromotedCommercialPrice(ctx context.Context, quotaPolicyID int, modelName string, now time.Time) (model.Price, bool, error) {
	var entries []entmodels.PromotedModelPolicy
	if err := model.DB.WithContext(ctx).
		Where("quota_policy_id = ? AND model = ? AND enabled = ?", quotaPolicyID, modelName, true).
		Order("sort_order ASC, id DESC").
		Find(&entries).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.Price{}, false, nil
		}
		return model.Price{}, false, err
	}
	for _, entry := range entries {
		if entry.ActiveAt(now) {
			return entry.OverridePrice, true, nil
		}
	}
	return model.Price{}, false, nil
}
```

- [ ] **Step 4: Wire resolver into relay price path**

Modify `core/controller/relay-controller.go`:

1. Add enterprise quota import:

```go
	enterprisequota "github.com/labring/aiproxy/core/enterprise/quota"
```

2. Replace `defaultPriceFunc` with logic that preserves group overrides:

```go
func defaultPriceFunc(c *gin.Context, mc model.ModelConfig) (model.Price, error) {
	group := middleware.GetGroup(c)
	if groupModelConfig, ok := group.ModelConfigs[mc.Model]; ok && groupModelConfig.OverridePrice {
		return mc.Price, nil
	}

	var feishuUser entmodels.FeishuUser
	if err := model.DB.Where("group_id = ?", group.ID).First(&feishuUser).Error; err != nil {
		return mc.Price, nil
	}

	policy, err := enterprisequota.GetPolicyForUser(c.Request.Context(), feishuUser.OpenID)
	if err != nil || policy == nil {
		return mc.Price, nil
	}

	price, ok, err := enterprisequota.ResolvePromotedCommercialPrice(c.Request.Context(), policy.ID, mc.Model, time.Now())
	if err != nil {
		return model.Price{}, err
	}
	if ok {
		return price, nil
	}

	return mc.Price, nil
}
```

Also add the enterprise models import used by the Feishu user lookup:

```go
	entmodels "github.com/labring/aiproxy/core/enterprise/models"
```

This mirrors the existing quota hook behavior in `core/enterprise/quota/hook.go`: Feishu identity is resolved by matching the current group ID to `FeishuUser.GroupID`. Keep the logic order exactly: group override first, promoted price second, base price third.

- [ ] **Step 5: Run resolver tests and relay compile**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/quota -run 'TestResolvePromotedCommercialPrice' -count=1
cd core && go test -tags enterprise ./controller -run '^$'
```

Expected: resolver tests pass; controller package compiles.

- [ ] **Step 6: Commit**

```bash
git add core/enterprise/quota/promoted_price.go core/enterprise/quota/promoted_price_test.go core/controller/relay-controller.go
git commit -m "feat: resolve promoted model commercial prices"
```

## Task 5: Add My Access Promoted Metadata

**Files:**
- Modify: `core/enterprise/access_info.go`
- Test: add focused test in existing access info tests if practical, otherwise compile `go test -tags enterprise ./enterprise`

- [ ] **Step 1: Extend response types**

In `core/enterprise/access_info.go`, add promoted fields to `ModelAccessInfo`:

```go
	IsPromoted       bool        `json:"is_promoted"`
	RecommendBadge   string      `json:"recommend_badge,omitempty"`
	CommercialLocked bool        `json:"commercial_locked,omitempty"`
	ReferenceChannel int         `json:"reference_channel,omitempty"`
	BasePrice        model.Price `json:"base_price,omitempty"`
```

- [ ] **Step 2: Load promoted entries for current user policy**

Near the beginning of the My Access model listing handler, after the current Feishu user is resolved, call:

```go
effectivePolicy, _ := quota.GetPolicyForUser(c.Request.Context(), feishuUser.OpenID)
promotedByModel := map[string]models.PromotedModelPolicy{}
if effectivePolicy != nil {
	var promoted []models.PromotedModelPolicy
	now := time.Now()
	if err := model.DB.Where("quota_policy_id = ? AND enabled = ?", effectivePolicy.ID, true).
		Order("sort_order ASC, id DESC").
		Find(&promoted).Error; err == nil {
		for _, entry := range promoted {
			if entry.ActiveAt(now) {
				if _, exists := promotedByModel[entry.Model]; !exists {
					promotedByModel[entry.Model] = entry
				}
			}
		}
	}
}
```

- [ ] **Step 3: Apply promoted display metadata without hiding normal models**

When constructing `ModelAccessInfo`, after group override price handling, if `promotedByModel[modelName]` exists and group override did not apply, set displayed price and metadata:

```go
if promoted, exists := promotedByModel[modelName]; exists {
	info.IsPromoted = true
	info.RecommendBadge = promoted.RecommendBadge
	info.CommercialLocked = promoted.PriceLocked
	info.ReferenceChannel = promoted.ChannelID
	info.BasePrice = promoted.BasePrice
	if !groupPriceOverridden {
		info.InputPrice = float64(promoted.OverridePrice.InputPrice)
		info.OutputPrice = float64(promoted.OverridePrice.OutputPrice)
		if int64(promoted.OverridePrice.InputPriceUnit) != 0 {
			info.PriceUnit = int64(promoted.OverridePrice.InputPriceUnit)
		}
	}
}
```

Track `groupPriceOverridden := false` where `gmc.OverridePrice` is applied.

- [ ] **Step 4: Sort promoted models first**

Where models are sorted inside each owner group, place promoted models before non-promoted models, then preserve the existing secondary sort:

```go
if models[i].IsPromoted != models[j].IsPromoted {
	return models[i].IsPromoted
}
```

- [ ] **Step 5: Verify compile**

Run:

```bash
cd core && go test -tags enterprise ./enterprise -run '^$'
```

Expected: enterprise package compiles.

- [ ] **Step 6: Commit**

```bash
git add core/enterprise/access_info.go
git commit -m "feat: show promoted models in my access"
```

## Task 6: Add Frontend API Types

**Files:**
- Modify: `web/src/api/enterprise.ts`

- [ ] **Step 1: Add TypeScript types**

In `web/src/api/enterprise.ts`, import `ModelPrice`:

```ts
import type { ModelPrice } from '@/types/model'
```

Add interfaces near quota policy types:

```ts
export interface PromotedModelPolicy {
    id: number
    created_at: string
    updated_at: string
    quota_policy_id: number
    model: string
    channel_id: number
    display_name: string
    recommend_badge: string
    sort_order: number
    enabled: boolean
    base_price: ModelPrice
    override_price: ModelPrice
    discount_rate: number
    price_locked: boolean
    effective_at?: string | null
    expires_at?: string | null
    version: number
    created_by: string
    updated_by: string
}

export interface PromotedModelPolicyInput {
    model: string
    channel_id?: number
    display_name?: string
    recommend_badge?: string
    sort_order?: number
    enabled: boolean
    override_price: ModelPrice
    discount_rate?: number
    price_locked?: boolean
    effective_at?: string | null
    expires_at?: string | null
}

export interface PromotedModelAudit {
    id: number
    promoted_model_policy_id: number
    quota_policy_id: number
    action: string
    before: string
    after: string
    summary: string
    operator_id: string
    operator_name: string
    created_at: string
}
```

- [ ] **Step 2: Add API methods**

Inside `enterpriseApi`, add:

```ts
    listPromotedModels: (policyId: number): Promise<{ entries: PromotedModelPolicy[] }> => {
        return get(`/enterprise/quota/policies/${policyId}/promoted-models`)
    },

    createPromotedModel: (policyId: number, entry: PromotedModelPolicyInput): Promise<PromotedModelPolicy> => {
        return post(`/enterprise/quota/policies/${policyId}/promoted-models`, entry)
    },

    updatePromotedModel: (
        policyId: number,
        entryId: number,
        entry: PromotedModelPolicyInput & { override_locked?: boolean },
    ): Promise<PromotedModelPolicy> => {
        return put(`/enterprise/quota/policies/${policyId}/promoted-models/${entryId}`, entry)
    },

    deletePromotedModel: (policyId: number, entryId: number): Promise<void> => {
        return del(`/enterprise/quota/policies/${policyId}/promoted-models/${entryId}`)
    },

    rollbackPromotedModel: (policyId: number, entryId: number, version: number): Promise<PromotedModelPolicy> => {
        return post(`/enterprise/quota/policies/${policyId}/promoted-models/${entryId}/rollback`, { version })
    },

    listPromotedModelAudits: (policyId: number): Promise<{ audits: PromotedModelAudit[] }> => {
        return get(`/enterprise/quota/policies/${policyId}/promoted-models/audit`)
    },
```

- [ ] **Step 3: Typecheck frontend**

Run:

```bash
cd web && pnpm run lint
```

Expected: lint passes or only pre-existing unrelated warnings appear.

- [ ] **Step 4: Commit**

```bash
git add web/src/api/enterprise.ts
git commit -m "feat: add promoted model frontend API"
```

## Task 7: Add Quota Page Promoted Models Tab

**Files:**
- Modify: `web/src/pages/enterprise/quota.tsx`
- Modify: `web/public/locales/en/translation.json`
- Modify: `web/public/locales/zh/translation.json`

- [ ] **Step 1: Add UI imports**

In `web/src/pages/enterprise/quota.tsx`, add icons:

```ts
import { Lock, LockOpen, RotateCcw, Star } from "lucide-react"
```

Merge with the existing lucide import rather than creating a duplicate import.

- [ ] **Step 2: Add helper price formatter**

Near existing helper functions, add:

```ts
function formatTokenPrice(price?: number, unit?: number) {
    if (!price) return "-"
    const normalized = unit && unit > 0 ? price / unit * 1_000_000 : price * 1_000_000
    return `¥${normalized.toFixed(4)} / M`
}
```

- [ ] **Step 3: Add promoted model tab component**

Add a `PromotedModelsTab` component before the main page component. Keep it focused on single-entry CRUD:

```tsx
function PromotedModelsTab({ policies, canManage }: { policies: QuotaPolicy[]; canManage: boolean }) {
    const { t } = useTranslation()
    const queryClient = useQueryClient()
    const [selectedPolicyId, setSelectedPolicyId] = useState<number | null>(policies[0]?.id ?? null)
    const [editing, setEditing] = useState<PromotedModelPolicy | null>(null)
    const [dialogOpen, setDialogOpen] = useState(false)
    const [overrideLocked, setOverrideLocked] = useState(false)
    const [lockedFilter, setLockedFilter] = useState<"all" | "locked" | "unlocked">("all")
    const [form, setForm] = useState<PromotedModelPolicyInput>({
        model: "",
        enabled: true,
        override_price: { input_price_unit: 1, output_price_unit: 1 },
        price_locked: false,
    })

    const policyId = selectedPolicyId ?? policies[0]?.id
    const entriesQuery = useQuery({
        queryKey: ["enterprise", "quota-promoted-models", policyId],
        queryFn: () => enterpriseApi.listPromotedModels(policyId!),
        enabled: !!policyId,
    })
    const auditsQuery = useQuery({
        queryKey: ["enterprise", "quota-promoted-model-audits", policyId],
        queryFn: () => enterpriseApi.listPromotedModelAudits(policyId!),
        enabled: !!policyId,
    })

    const createMutation = useMutation({
        mutationFn: (payload: PromotedModelPolicyInput) => enterpriseApi.createPromotedModel(policyId!, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-promoted-models", policyId] })
            setDialogOpen(false)
            toast.success(t("enterprise.quota.promoted.saved" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const updateMutation = useMutation({
        mutationFn: (payload: PromotedModelPolicyInput & { override_locked?: boolean }) =>
            enterpriseApi.updatePromotedModel(policyId!, editing!.id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-promoted-models", policyId] })
            queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-promoted-model-audits", policyId] })
            setDialogOpen(false)
            setEditing(null)
            toast.success(t("enterprise.quota.promoted.saved" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const openCreate = () => {
        setEditing(null)
        setOverrideLocked(false)
        setForm({ model: "", enabled: true, override_price: { input_price_unit: 1, output_price_unit: 1 }, price_locked: false })
        setDialogOpen(true)
    }

    const openEdit = (entry: PromotedModelPolicy) => {
        setEditing(entry)
        setOverrideLocked(false)
        setForm({
            model: entry.model,
            channel_id: entry.channel_id || undefined,
            display_name: entry.display_name,
            recommend_badge: entry.recommend_badge,
            sort_order: entry.sort_order,
            enabled: entry.enabled,
            override_price: entry.override_price || {},
            discount_rate: entry.discount_rate,
            price_locked: entry.price_locked,
            effective_at: entry.effective_at,
            expires_at: entry.expires_at,
        })
        setDialogOpen(true)
    }

    const save = () => {
        if (!form.model.trim()) {
            toast.error(t("enterprise.quota.promoted.modelRequired" as never))
            return
        }
        if (editing) {
            updateMutation.mutate({ ...form, override_locked: overrideLocked })
        } else {
            createMutation.mutate(form)
        }
    }

    const entries = (entriesQuery.data?.entries || []).filter((entry) => {
        if (lockedFilter === "locked") return entry.price_locked
        if (lockedFilter === "unlocked") return !entry.price_locked
        return true
    })

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between gap-3">
                    <CardTitle className="flex items-center gap-2">
                        <Star className="h-5 w-5" />
                        {t("enterprise.quota.promoted.title" as never)}
                    </CardTitle>
                    <div className="flex items-center gap-2">
                        <Select value={policyId?.toString()} onValueChange={(v) => setSelectedPolicyId(Number(v))}>
                            <SelectTrigger className="w-56">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                {policies.map((p) => <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>)}
                            </SelectContent>
                        </Select>
                        <Select value={lockedFilter} onValueChange={(v) => setLockedFilter(v as typeof lockedFilter)}>
                            <SelectTrigger className="w-36">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("common.all")}</SelectItem>
                                <SelectItem value="locked">{t("enterprise.quota.promoted.locked" as never)}</SelectItem>
                                <SelectItem value="unlocked">{t("enterprise.quota.promoted.unlocked" as never)}</SelectItem>
                            </SelectContent>
                        </Select>
                        {canManage && <Button onClick={openCreate}><Plus className="h-4 w-4 mr-2" />{t("enterprise.quota.promoted.add" as never)}</Button>}
                    </div>
                </div>
            </CardHeader>
            <CardContent className="space-y-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t("enterprise.quota.promoted.model" as never)}</TableHead>
                            <TableHead>{t("enterprise.quota.promoted.badge" as never)}</TableHead>
                            <TableHead>{t("enterprise.quota.promoted.overridePrice" as never)}</TableHead>
                            <TableHead>{t("enterprise.quota.promoted.lock" as never)}</TableHead>
                            <TableHead>{t("enterprise.quota.promoted.status" as never)}</TableHead>
                            <TableHead>{t("common.edit")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {entries.map((entry) => (
                            <TableRow key={entry.id}>
                                <TableCell>
                                    <div className="font-medium">{entry.display_name || entry.model}</div>
                                    <div className="text-xs text-muted-foreground">{entry.model}</div>
                                </TableCell>
                                <TableCell>{entry.recommend_badge || "-"}</TableCell>
                                <TableCell>
                                    <div>{formatTokenPrice(entry.override_price?.input_price, entry.override_price?.input_price_unit)}</div>
                                    <div className="text-xs text-muted-foreground">{formatTokenPrice(entry.override_price?.output_price, entry.override_price?.output_price_unit)}</div>
                                </TableCell>
                                <TableCell>{entry.price_locked ? <Lock className="h-4 w-4" /> : <LockOpen className="h-4 w-4 text-muted-foreground" />}</TableCell>
                                <TableCell><Badge variant={entry.enabled ? "default" : "secondary"}>{entry.enabled ? t("common.enabled") : t("common.disabled")}</Badge></TableCell>
                                <TableCell>{canManage && <Button variant="ghost" size="icon" onClick={() => openEdit(entry)}><Pencil className="h-4 w-4" /></Button>}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
                <div className="text-xs text-muted-foreground">{t("enterprise.quota.promoted.referenceChannelHint" as never)}</div>
                <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>{editing ? t("enterprise.quota.promoted.edit" as never) : t("enterprise.quota.promoted.add" as never)}</DialogTitle>
                        </DialogHeader>
                        <div className="space-y-3">
                            <div>
                                <Label>{t("enterprise.quota.promoted.model" as never)}</Label>
                                <Input value={form.model} disabled={!!editing} onChange={(e) => setForm({ ...form, model: e.target.value })} />
                            </div>
                            <div>
                                <Label>{t("enterprise.quota.promoted.badge" as never)}</Label>
                                <Input value={form.recommend_badge || ""} onChange={(e) => setForm({ ...form, recommend_badge: e.target.value })} />
                            </div>
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <Label>Input (¥/token)</Label>
                                    <Input type="number" value={form.override_price.input_price || ""} onChange={(e) => setForm({ ...form, override_price: { ...form.override_price, input_price: Number(e.target.value), input_price_unit: 1 } })} />
                                </div>
                                <div>
                                    <Label>Output (¥/token)</Label>
                                    <Input type="number" value={form.override_price.output_price || ""} onChange={(e) => setForm({ ...form, override_price: { ...form.override_price, output_price: Number(e.target.value), output_price_unit: 1 } })} />
                                </div>
                            </div>
                            <div className="flex items-center justify-between">
                                <Label>{t("enterprise.quota.promoted.lock" as never)}</Label>
                                <Switch checked={!!form.price_locked} onCheckedChange={(v) => setForm({ ...form, price_locked: v })} />
                            </div>
                            {editing?.price_locked && (
                                <div className="flex items-center justify-between rounded border p-2">
                                    <Label>{t("enterprise.quota.promoted.overrideLocked" as never)}</Label>
                                    <Switch checked={overrideLocked} onCheckedChange={setOverrideLocked} />
                                </div>
                            )}
                            <div className="flex items-center justify-between">
                                <Label>{t("common.enabled")}</Label>
                                <Switch checked={form.enabled} onCheckedChange={(v) => setForm({ ...form, enabled: v })} />
                            </div>
                        </div>
                        <DialogFooter>
                            <Button variant="outline" onClick={() => setDialogOpen(false)}>{t("common.cancel")}</Button>
                            <Button onClick={save} disabled={createMutation.isPending || updateMutation.isPending}>{t("common.save")}</Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
                <Separator />
                <div className="space-y-2">
                    <h3 className="text-sm font-medium">{t("enterprise.quota.promoted.audit" as never)}</h3>
                    {(auditsQuery.data?.audits || []).slice(0, 8).map((audit) => (
                        <div key={audit.id} className="text-xs text-muted-foreground">
                            {audit.created_at} · {audit.action} · {audit.summary}
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    )
}
```

Make sure `PromotedModelPolicy`, `PromotedModelPolicyInput`, and `PromotedModelAudit` are imported from `@/api/enterprise`.

- [ ] **Step 4: Add tab to main quota page**

In the main `<TabsList>`, add:

```tsx
<TabsTrigger value="promoted">
    <Star className="w-4 h-4 mr-1.5" />
    {t("enterprise.quota.promoted.tab" as never)}
</TabsTrigger>
```

After user override tab content, add:

```tsx
<TabsContent value="promoted">
    <PromotedModelsTab policies={policies} canManage={canManage} />
</TabsContent>
```

- [ ] **Step 5: Add translations**

Add keys under `enterprise.quota.promoted` in both locale files:

```json
"promoted": {
  "tab": "Promoted Models",
  "title": "Promoted Models",
  "add": "Add promoted model",
  "edit": "Edit promoted model",
  "model": "Model",
  "badge": "Badge",
  "overridePrice": "Commercial price",
  "lock": "Commercial lock",
  "locked": "Locked",
  "unlocked": "Unlocked",
  "status": "Status",
  "saved": "Promoted model saved",
  "audit": "Change history",
  "modelRequired": "Model is required",
  "overrideLocked": "Override locked price",
  "referenceChannelHint": "Reference channel is for admin context only and does not force routing."
}
```

Use Chinese equivalents in `zh`.

- [ ] **Step 6: Verify frontend lint**

Run:

```bash
cd web && pnpm run lint
```

Expected: lint passes or only unrelated existing warnings appear. Fix any errors introduced by this task.

- [ ] **Step 7: Commit**

```bash
git add web/src/pages/enterprise/quota.tsx web/public/locales/en/translation.json web/public/locales/zh/translation.json
git commit -m "feat: add promoted model quota tab"
```

## Task 8: Display Promoted Models In My Access

**Files:**
- Modify: `web/src/api/enterprise.ts`
- Modify: `web/src/pages/enterprise/my-access.tsx`
- Modify: locale files if labels are needed

- [ ] **Step 1: Extend My Access model type**

In `web/src/api/enterprise.ts`, add these fields to the My Access model interface used by `my-access.tsx`:

```ts
is_promoted?: boolean
recommend_badge?: string
commercial_locked?: boolean
reference_channel?: number
base_price?: ModelPrice
```

If the interface is local to `my-access.tsx`, update it there instead.

- [ ] **Step 2: Update table/list display**

In `web/src/pages/enterprise/my-access.tsx`, where model rows render model name and price, add:

```tsx
{model.is_promoted && (
    <Badge variant="default" className="ml-2">
        {model.recommend_badge || t("enterprise.myAccess.promoted" as never)}
    </Badge>
)}
```

Show discounted price using existing price display. Do not hide non-promoted models.

- [ ] **Step 3: Ensure promoted sort is stable**

If frontend sorts models client-side, add promoted-first sorting there:

```ts
if (!!a.is_promoted !== !!b.is_promoted) return a.is_promoted ? -1 : 1
```

If backend already sorts promoted first, do not add a second conflicting sort.

- [ ] **Step 4: Add translations**

Add:

```json
"promoted": "Promoted"
```

under `enterprise.myAccess` in both locale files.

- [ ] **Step 5: Run frontend lint**

Run:

```bash
cd web && pnpm run lint
```

Expected: lint passes.

- [ ] **Step 6: Commit**

```bash
git add web/src/api/enterprise.ts web/src/pages/enterprise/my-access.tsx web/public/locales/en/translation.json web/public/locales/zh/translation.json
git commit -m "feat: show promoted models in my access"
```

## Task 9: Final Verification

**Files:**
- No new files unless fixes are needed.

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
cd core && go test -tags enterprise ./enterprise/models ./enterprise/quota
```

Expected: PASS.

- [ ] **Step 2: Compile relay/controller enterprise path**

Run:

```bash
cd core && go test -tags enterprise ./controller ./middleware
```

Expected: PASS.

- [ ] **Step 3: Run frontend checks**

Run:

```bash
cd web && pnpm run lint
```

Expected: PASS.

- [ ] **Step 4: Review git diff**

Run:

```bash
git status --short
git log --oneline -5
```

Expected: only intended changes remain, with task commits present. Existing unrelated PPIO/Novita diagnostic changes may still be present if they were not handled separately; do not revert them.

- [ ] **Step 5: Commit any verification fixes**

If verification required small fixes, stage the exact files changed by those fixes. For example, if the final fix only touched the relay controller:

```bash
git add core/controller/relay-controller.go
git commit -m "fix: stabilize promoted model policy"
```

If no fixes were required, do not create an empty commit.
