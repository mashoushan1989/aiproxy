//go:build enterprise

package ppio

import (
	"testing"

	"github.com/labring/aiproxy/core/enterprise/synccommon"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

func TestPPIOURLHelpers(t *testing.T) {
	cases := []struct {
		name          string
		baseURL       string
		wantResp      string
		wantWebSearch string
	}{
		{
			name:          "default PPIO base URL",
			baseURL:       "https://api.ppinfra.com/v3/openai",
			wantResp:      "https://api.ppinfra.com/openai/v1",
			wantWebSearch: "https://api.ppinfra.com/v3",
		},
		{
			name:          "custom base URL with /v3/openai suffix",
			baseURL:       "https://custom.example.com/v3/openai",
			wantResp:      "https://custom.example.com/openai/v1",
			wantWebSearch: "https://custom.example.com/v3",
		},
		{
			name:          "base URL without /v3/openai — falls back to default",
			baseURL:       "https://other.example.com/api",
			wantResp:      "https://api.ppinfra.com/openai/v1",
			wantWebSearch: "https://api.ppinfra.com/v3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ppioResponsesBase(tc.baseURL); got != tc.wantResp {
				t.Errorf("ppioResponsesBase(%q) = %q, want %q", tc.baseURL, got, tc.wantResp)
			}

			if got := ppioWebSearchBase(tc.baseURL); got != tc.wantWebSearch {
				t.Errorf("ppioWebSearchBase(%q) = %q, want %q", tc.baseURL, got, tc.wantWebSearch)
			}
		})
	}
}

func TestBuildConfigFromPPIOModelV2_ToolChoiceAndVision(t *testing.T) {
	tests := []struct {
		name           string
		model          PPIOModelV2
		wantToolChoice bool
		wantVision     bool
		wantMaxOutput  int64
	}{
		{
			name: "Claude chat model with image input",
			model: PPIOModelV2{
				ID:              "claude-sonnet-4-20250514",
				ModelType:       "chat",
				Features:        []string{"tool_use", "streaming"},
				InputModalities: []string{"text", "image"},
				Endpoints:       []string{"anthropic"},
				MaxOutputTokens: 64000,
			},
			wantToolChoice: true,
			wantVision:     true,
			wantMaxOutput:  64000,
		},
		{
			name: "Chat model text only",
			model: PPIOModelV2{
				ID:              "deepseek-v3",
				ModelType:       "chat",
				Features:        []string{},
				InputModalities: []string{"text"},
				MaxOutputTokens: 8192,
			},
			wantToolChoice: true,
			wantVision:     false,
			wantMaxOutput:  8192,
		},
		{
			name: "Embedding model",
			model: PPIOModelV2{
				ID:              "bge-m3",
				ModelType:       "embedding",
				Features:        []string{},
				InputModalities: []string{"text"},
				MaxOutputTokens: 2048,
			},
			wantToolChoice: false,
			wantVision:     false,
			wantMaxOutput:  2048,
		},
		{
			name: "Image generation model",
			model: PPIOModelV2{
				ID:              "flux-1",
				ModelType:       "image",
				Features:        []string{},
				InputModalities: []string{"text"},
				MaxOutputTokens: 1024,
			},
			wantToolChoice: false,
			wantVision:     false,
			wantMaxOutput:  1024,
		},
		{
			name: "Chat model with image modality but no tool features",
			model: PPIOModelV2{
				ID:              "llava-v1.6",
				ModelType:       "chat",
				Features:        []string{"streaming"},
				InputModalities: []string{"text", "image"},
				MaxOutputTokens: 4096,
			},
			wantToolChoice: true,
			wantVision:     true,
			wantMaxOutput:  4096,
		},
		{
			name: "Anthropic-compatible mimo-v2-pro keeps upstream max output tokens",
			model: PPIOModelV2{
				ID:              "xiaomimimo/mimo-v2-pro",
				ModelType:       "chat",
				Features:        []string{"streaming"},
				InputModalities: []string{"text"},
				Endpoints:       []string{"anthropic"},
				MaxOutputTokens: 131072,
			},
			wantToolChoice: true,
			wantVision:     false,
			wantMaxOutput:  131072,
		},
		{
			name: "Other anthropic non-Claude models keep upstream max output tokens",
			model: PPIOModelV2{
				ID:              "deepseek-r1",
				ModelType:       "chat",
				Features:        []string{"streaming"},
				InputModalities: []string{"text"},
				Endpoints:       []string{"anthropic"},
				MaxOutputTokens: 131072,
			},
			wantToolChoice: true,
			wantVision:     false,
			wantMaxOutput:  131072,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := buildConfigFromPPIOModelV2(&tt.model)

			gotToolChoice, _ := cfg[string(model.ModelConfigToolChoiceKey)].(bool)
			if gotToolChoice != tt.wantToolChoice {
				t.Errorf("tool_choice = %v, want %v", gotToolChoice, tt.wantToolChoice)
			}

			gotVision, _ := cfg[string(model.ModelConfigVisionKey)].(bool)
			if gotVision != tt.wantVision {
				t.Errorf("vision = %v, want %v", gotVision, tt.wantVision)
			}

			gotMaxOutput, _ := cfg["max_output_tokens"].(int64)
			if gotMaxOutput != tt.wantMaxOutput {
				t.Errorf("max_output_tokens = %v, want %v", gotMaxOutput, tt.wantMaxOutput)
			}
		})
	}
}

func TestUpdateModelConfigV2PreservesLockedPrice(t *testing.T) {
	db, err := model.OpenSQLite(t.TempDir() + "/ppio-sync.db")
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
		Owner:      model.ModelOwnerPPIO,
		SyncedFrom: synccommon.SyncedFromPPIO,
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

	remote := &PPIOModelV2{
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

	if err := updateModelConfigV2(db, remote); err != nil {
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
