//go:build enterprise

package models_test

import (
	"testing"

	. "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
)

func TestEnterpriseAutoMigrateBackfillsPromotedDiscountPricingMode(t *testing.T) {
	db, err := model.OpenSQLite(t.TempDir() + "/promoted_migrate.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Group{},
		&model.Channel{},
		&model.ModelConfig{},
		&PromotedModelPolicy{},
	))
	require.NoError(t, db.Create(&PromotedModelPolicy{
		QuotaPolicyID: 1,
		Model:         "pa/gpt-5.5",
		PricingMode:   PromotedModelPricingModeManual,
		DiscountRate:  0.4,
		BasePrice: CommercialPrice{
			InputPrice:      0.00003625,
			InputPriceUnit:  1,
			OutputPrice:     0.0002175,
			OutputPriceUnit: 1,
		},
		OverridePrice: CommercialPrice{
			InputPrice:      0.0000145,
			InputPriceUnit:  1,
			OutputPrice:     0.000087,
			OutputPriceUnit: 1,
		},
	}).Error)

	require.NoError(t, EnterpriseAutoMigrate(db))

	var entry PromotedModelPolicy
	require.NoError(t, db.First(&entry, "model = ?", "pa/gpt-5.5").Error)
	require.Equal(t, PromotedModelPricingModeDiscount, entry.PricingMode)
	require.Equal(t, 0.4, entry.DiscountRate)
}

func TestEnterpriseAutoMigrateKeepsPromotedManualPriceWhenDiscountMetadataDoesNotMatch(t *testing.T) {
	db, err := model.OpenSQLite(t.TempDir() + "/promoted_migrate_manual.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Group{},
		&model.Channel{},
		&model.ModelConfig{},
		&PromotedModelPolicy{},
	))
	require.NoError(t, db.Create(&PromotedModelPolicy{
		QuotaPolicyID: 1,
		Model:         "pa/gpt-5.5",
		PricingMode:   PromotedModelPricingModeManual,
		DiscountRate:  0.4,
		BasePrice: CommercialPrice{
			InputPrice:      0.00003625,
			InputPriceUnit:  1,
			OutputPrice:     0.0002175,
			OutputPriceUnit: 1,
		},
		OverridePrice: CommercialPrice{
			InputPrice:      0.0000123,
			InputPriceUnit:  1,
			OutputPrice:     0.000087,
			OutputPriceUnit: 1,
		},
	}).Error)

	require.NoError(t, EnterpriseAutoMigrate(db))

	var entry PromotedModelPolicy
	require.NoError(t, db.First(&entry, "model = ?", "pa/gpt-5.5").Error)
	require.Equal(t, PromotedModelPricingModeManual, entry.PricingMode)
	require.Equal(t, 0.4, entry.DiscountRate)
	require.Equal(t, 0.0000123, entry.OverridePrice.InputPrice)
}
