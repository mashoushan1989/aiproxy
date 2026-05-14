//go:build enterprise

package quota

import (
	"fmt"
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

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := db.AutoMigrate(
		&model.ModelConfig{},
		&model.Channel{},
		&entmodels.QuotaPolicy{},
		&entmodels.FeishuUser{},
		&entmodels.GroupQuotaPolicy{},
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
	if entry.BasePrice.InputPrice != 0.00003625 {
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
	overridePrice, err := commercialPriceFromModelPrice(model.Price{
		InputPrice:     model.ZeroNullFloat64(0.1),
		InputPriceUnit: model.ZeroNullInt64(1),
	})
	if err != nil {
		t.Fatalf("convert price: %v", err)
	}

	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		PriceLocked:   true,
		OverridePrice: overridePrice,
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	_, err = UpdatePromotedModelEntry(entry.ID, UpdatePromotedModelEntryRequest{
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.2),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
	}, AuditOperator{ID: "admin"}, false)
	if err == nil {
		t.Fatalf("expected locked price update to fail")
	}
}

func TestRollbackPromotedModelPolicyCreatesNewVersion(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	entry, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.1),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
		EffectiveAt:    ptrTime(time.Now().Add(-time.Hour)),
		RecommendBadge: "A",
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := UpdatePromotedModelEntry(entry.ID, UpdatePromotedModelEntryRequest{
		RecommendBadge: "B",
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.2),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
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
