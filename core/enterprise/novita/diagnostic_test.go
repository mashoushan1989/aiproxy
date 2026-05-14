//go:build enterprise

package novita

import (
	"strings"
	"testing"

	"github.com/labring/aiproxy/core/enterprise/synccommon"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

func TestCompareModelConfigsV2SkipsPriceChangesWhenSyncPriceLocked(t *testing.T) {
	local := &model.ModelConfig{
		Type: mode.ChatCompletions,
		Config: map[model.ModelConfigKey]any{
			synccommon.ModelConfigSyncPriceLockedKey: true,
		},
		Price: model.Price{
			InputPrice:  model.ZeroNullFloat64(0.0000145),
			OutputPrice: model.ZeroNullFloat64(0.000087),
			CachedPrice: model.ZeroNullFloat64(0.000001452),
			ConditionalPrices: []model.ConditionalPrice{
				{Condition: model.PriceCondition{InputTokenMin: 1}},
			},
		},
	}

	remote := &NovitaModelV2{
		ID:                           "pa/gpt-5.5",
		ModelType:                    "chat",
		Endpoints:                    []string{"chat/completions"},
		InputTokenPricePerM:          362500000,
		OutputTokenPricePerM:         2175000000,
		SupportPromptCache:           true,
		CacheReadInputTokenPricePerM: 36300000,
		IsTieredBilling:              true,
		TieredBillingConfigs: []TieredBillingConfig{
			{MinTokens: 0, MaxTokens: 128000},
			{MinTokens: 128000, MaxTokens: 0},
		},
	}

	changes := compareModelConfigsV2(local, remote, 1)

	for _, change := range changes {
		if strings.Contains(change, "price") || strings.Contains(change, "tiered_billing_count") {
			t.Fatalf("locked price change was reported: %q in %#v", change, changes)
		}
	}
}
