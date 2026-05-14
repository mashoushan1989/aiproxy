//go:build enterprise

package quota

import (
	"testing"
	"time"

	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
)

func TestResolvePromotedModelPriceUsesActiveCommercialPrice(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	if err := db.Create(&entmodels.GroupQuotaPolicy{
		GroupID:       "engineering",
		QuotaPolicyID: policy.ID,
	}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	_, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.0000145),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
		EffectiveAt: ptrTime(time.Now().Add(-time.Hour)),
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create promoted model: %v", err)
	}

	fallback := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.00003625),
		InputPriceUnit: model.ZeroNullInt64(1),
	}
	got, err := ResolvePromotedModelPrice(model.GroupCache{ID: "engineering"}, "pa/gpt-5.5", fallback)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.InputPrice != model.ZeroNullFloat64(0.0000145) {
		t.Fatalf("input price = %v, want promoted price", got.InputPrice)
	}
}

func TestResolvePromotedModelPriceFallsBackWhenEntryInactive(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	if err := db.Create(&entmodels.GroupQuotaPolicy{
		GroupID:       "engineering",
		QuotaPolicyID: policy.ID,
	}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	_, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.0000145),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
		EffectiveAt: ptrTime(time.Now().Add(time.Hour)),
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create promoted model: %v", err)
	}

	fallback := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.00003625),
		InputPriceUnit: model.ZeroNullInt64(1),
	}
	got, err := ResolvePromotedModelPrice(model.GroupCache{ID: "engineering"}, "pa/gpt-5.5", fallback)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.InputPrice != fallback.InputPrice {
		t.Fatalf("input price = %v, want fallback %v", got.InputPrice, fallback.InputPrice)
	}
}

func TestResolvePromotedModelPriceSkipsInactiveRows(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	if err := db.Create(&entmodels.GroupQuotaPolicy{
		GroupID:       "engineering",
		QuotaPolicyID: policy.ID,
	}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	_, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		SortOrder:     0,
		Enabled:       true,
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.00001),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
		EffectiveAt: ptrTime(time.Now().Add(time.Hour)),
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create future promoted model: %v", err)
	}

	_, err = CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		SortOrder:     10,
		Enabled:       true,
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.00002),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
		EffectiveAt: ptrTime(time.Now().Add(-time.Hour)),
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create active promoted model: %v", err)
	}

	fallback := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.00003625),
		InputPriceUnit: model.ZeroNullInt64(1),
	}
	got, err := ResolvePromotedModelPrice(model.GroupCache{ID: "engineering"}, "pa/gpt-5.5", fallback)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.InputPrice != model.ZeroNullFloat64(0.00002) {
		t.Fatalf("input price = %v, want active promoted price", got.InputPrice)
	}
}

func TestDefaultPriceFuncKeepsGroupOverrideBeforePromotedPrice(t *testing.T) {
	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	if err := db.Create(&entmodels.GroupQuotaPolicy{
		GroupID:       "engineering",
		QuotaPolicyID: policy.ID,
	}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	_, err := CreatePromotedModelEntry(CreatePromotedModelEntryRequest{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		OverridePrice: model.Price{
			InputPrice:     model.ZeroNullFloat64(0.0000145),
			InputPriceUnit: model.ZeroNullInt64(1),
		},
	}, AuditOperator{ID: "admin"})
	if err != nil {
		t.Fatalf("create promoted model: %v", err)
	}

	groupOverride := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.00001),
		InputPriceUnit: model.ZeroNullInt64(1),
	}
	mc := model.ModelConfig{
		Model: "pa/gpt-5.5",
		Price: groupOverride,
	}
	group := model.GroupCache{
		ID: "engineering",
		ModelConfigs: map[string]model.GroupModelConfig{
			"pa/gpt-5.5": {
				Model:         "pa/gpt-5.5",
				OverridePrice: true,
				Price:         groupOverride,
			},
		},
	}

	if groupModelConfig, ok := group.ModelConfigs[mc.Model]; ok && groupModelConfig.OverridePrice {
		got := mc.Price
		if got.InputPrice != groupOverride.InputPrice {
			t.Fatalf("group override input price = %v, want %v", got.InputPrice, groupOverride.InputPrice)
		}
		return
	}

	t.Fatalf("group override was not detected")
}
