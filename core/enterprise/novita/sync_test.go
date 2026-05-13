//go:build enterprise

package novita

import (
	"testing"

	"github.com/labring/aiproxy/core/enterprise/synccommon"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

func TestNovitaResponsesBase(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "default Novita base URL",
			baseURL: "https://api.novita.ai/v3/openai",
			want:    "https://api.novita.ai/openai/v1",
		},
		{
			name:    "custom base URL with /v3/openai suffix",
			baseURL: "https://custom.example.com/v3/openai",
			want:    "https://custom.example.com/openai/v1",
		},
		{
			name:    "base URL without /v3/openai — falls back to default",
			baseURL: "https://other.example.com/api",
			want:    "https://api.novita.ai/openai/v1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := novitaResponsesBase(tc.baseURL); got != tc.want {
				t.Errorf("novitaResponsesBase(%q) = %q, want %q", tc.baseURL, got, tc.want)
			}
		})
	}
}

func TestBuildConfigFromV2Model_ToolChoiceAndVision(t *testing.T) {
	tests := []struct {
		name           string
		model          NovitaModelV2
		wantToolChoice bool
		wantVision     bool
	}{
		{
			name: "Claude chat model with image input",
			model: NovitaModelV2{
				ID:              "claude-sonnet-4-20250514",
				ModelType:       "chat",
				Features:        []string{"tool_use"},
				InputModalities: []string{"text", "image"},
			},
			wantToolChoice: true,
			wantVision:     true,
		},
		{
			name: "Text-only chat model",
			model: NovitaModelV2{
				ID:              "deepseek-v3",
				ModelType:       "chat",
				InputModalities: []string{"text"},
			},
			wantToolChoice: true,
			wantVision:     false,
		},
		{
			name: "Embedding model",
			model: NovitaModelV2{
				ID:              "bge-m3",
				ModelType:       "embedding",
				InputModalities: []string{"text"},
			},
			wantToolChoice: false,
			wantVision:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := buildConfigFromV2Model(&tt.model)

			gotToolChoice, _ := cfg[string(model.ModelConfigToolChoiceKey)].(bool)
			if gotToolChoice != tt.wantToolChoice {
				t.Errorf("tool_choice = %v, want %v", gotToolChoice, tt.wantToolChoice)
			}

			gotVision, _ := cfg[string(model.ModelConfigVisionKey)].(bool)
			if gotVision != tt.wantVision {
				t.Errorf("vision = %v, want %v", gotVision, tt.wantVision)
			}
		})
	}
}

func TestUpdateModelConfigV2PreservesLockedPrice(t *testing.T) {
	db, err := model.OpenSQLite(t.TempDir() + "/novita-sync.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	oldDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
	})

	if err := db.AutoMigrate(&model.ModelConfig{}); err != nil {
		t.Fatalf("migrate model_configs: %v", err)
	}

	manualPrice := model.Price{
		InputPrice:      model.ZeroNullFloat64(0.123),
		InputPriceUnit:  model.ZeroNullInt64(1),
		OutputPrice:     model.ZeroNullFloat64(0.456),
		OutputPriceUnit: model.ZeroNullInt64(1),
		ConditionalPrices: []model.ConditionalPrice{
			{
				Condition: model.PriceCondition{
					InputTokenMin: 1,
					InputTokenMax: 100,
				},
				Price: model.Price{
					InputPrice:      model.ZeroNullFloat64(0.111),
					InputPriceUnit:  model.ZeroNullInt64(1),
					OutputPrice:     model.ZeroNullFloat64(0.222),
					OutputPriceUnit: model.ZeroNullInt64(1),
				},
			},
		},
	}

	existing := model.ModelConfig{
		Model:      "pa/gpt-5.5",
		Owner:      model.ModelOwnerNovita,
		SyncedFrom: synccommon.SyncedFromNovita,
		Type:       mode.ChatCompletions,
		RPM:        60,
		TPM:        1000000,
		Config: map[model.ModelConfigKey]any{
			synccommon.ModelConfigSyncPriceLockedKey: true,
			"old_field":                              "keep-no",
		},
		Price: manualPrice,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed model config: %v", err)
	}

	remote := &NovitaModelV2{
		ID:                           "pa/gpt-5.5",
		ModelType:                    "chat",
		Endpoints:                    []string{"chat/completions", "anthropic"},
		Features:                     []string{"tool_use"},
		InputModalities:              []string{"text"},
		MaxOutputTokens:              8192,
		RPM:                          120,
		TPM:                          2000000,
		InputTokenPricePerM:          999000000,
		OutputTokenPricePerM:         888000000,
		CacheReadInputTokenPricePerM: 777000000,
	}

	if err := updateModelConfigV2(db, remote, 7.25); err != nil {
		t.Fatalf("update model config: %v", err)
	}

	var got model.ModelConfig
	if err := db.Where("model = ?", "pa/gpt-5.5").First(&got).Error; err != nil {
		t.Fatalf("load updated model config: %v", err)
	}

	if got.Price.InputPrice != manualPrice.InputPrice ||
		got.Price.OutputPrice != manualPrice.OutputPrice ||
		len(got.Price.ConditionalPrices) != len(manualPrice.ConditionalPrices) {
		t.Fatalf("price was overwritten: got %#v want %#v", got.Price, manualPrice)
	}

	if got.RPM != 120 || got.TPM != 2000000 {
		t.Fatalf("limits were not refreshed: rpm=%d tpm=%d", got.RPM, got.TPM)
	}

	if locked, _ := got.Config[synccommon.ModelConfigSyncPriceLockedKey].(bool); !locked {
		t.Fatalf("sync price lock marker was not preserved in config: %#v", got.Config)
	}

	if _, ok := got.Config["old_field"]; ok {
		t.Fatalf("unrelated stale config key was preserved: %#v", got.Config)
	}
}
