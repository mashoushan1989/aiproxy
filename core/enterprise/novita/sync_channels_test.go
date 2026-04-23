//go:build enterprise

package novita

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	novitarelay "github.com/labring/aiproxy/core/relay/adaptor/novita"
)

func setupNovitaChannelTestDB(t *testing.T) {
	t.Helper()

	prevDB := model.DB
	prevUsingSQLite := common.UsingSQLite

	testDB, err := model.OpenSQLite(filepath.Join(t.TempDir(), "novita-sync.db"))
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	model.DB = testDB
	common.UsingSQLite = true
	t.Cleanup(func() {
		model.DB = prevDB
		common.UsingSQLite = prevUsingSQLite
	})

	if err := testDB.AutoMigrate(&model.Channel{}); err != nil {
		t.Fatalf("failed to migrate channel table: %v", err)
	}
}

func TestEnsureNovitaChannelsFromModels_UpdatesChannelConfigs(t *testing.T) {
	setupNovitaChannelTestDB(t)

	channels := []model.Channel{
		{
			Name:    "Novita (OpenAI)",
			Type:    model.ChannelTypeNovita,
			BaseURL: DefaultNovitaAPIBase,
			Key:     "novita-key",
			Status:  model.ChannelStatusEnabled,
		},
		{
			Name:    "Novita (Anthropic)",
			Type:    model.ChannelTypeAnthropic,
			BaseURL: DefaultNovitaAnthropicBase,
			Key:     "novita-key",
			Status:  model.ChannelStatusEnabled,
		},
	}

	for i := range channels {
		if err := model.DB.Create(&channels[i]).Error; err != nil {
			t.Fatalf("failed to seed channel %q: %v", channels[i].Name, err)
		}
	}

	purePassthrough := true
	allowUnknown := true

	info, err := ensureNovitaChannelsFromModels(
		[]string{"claude-sonnet-4-20250514"},
		[]string{"deepseek-v3"},
		nil,   // multimodalModels
		false, // skipChatUpdate
		true,  // skipMultimodalUpdate (nil multimodal → skip)
		false, // autoCreate
		&purePassthrough,
		&allowUnknown,
		NovitaConfigResult{},
	)
	if err != nil {
		t.Fatalf("ensureNovitaChannelsFromModels returned error: %v", err)
	}

	if !info.Novita.Exists {
		t.Fatalf("expected Novita channel info to exist")
	}

	var got []model.Channel
	if err := model.DB.Order("id asc").Find(&got).Error; err != nil {
		t.Fatalf("failed to load updated channels: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(got))
	}

	for _, ch := range got {
		switch ch.Type {
		case model.ChannelTypeNovita:
			pathBaseMap, ok := ch.Configs[model.ChannelConfigPathBaseMapKey].(map[string]any)
			if !ok {
				t.Fatalf(
					"openai channel path_base_map missing or wrong type: %#v",
					ch.Configs[model.ChannelConfigPathBaseMapKey],
				)
			}

			if gotBase := pathBaseMap["/v1/responses"]; gotBase != novitaResponsesBase(
				DefaultNovitaAPIBase,
			) {
				t.Fatalf(
					"responses base = %#v, want %q",
					gotBase,
					novitaResponsesBase(DefaultNovitaAPIBase),
				)
			}

			if gotAllow := ch.Configs.GetBool(
				model.ChannelConfigAllowPassthroughUnknown,
			); !gotAllow {
				t.Fatalf("allow_passthrough_unknown = false, want true")
			}
		case model.ChannelTypeAnthropic:
			if gotPure := ch.Configs.GetBool("pure_passthrough"); !gotPure {
				t.Fatalf("pure_passthrough = false, want true")
			}
		}
	}
}

func seedNovitaChannelsWithModels(t *testing.T) {
	t.Helper()

	channels := []model.Channel{
		{
			Name:    "Novita (OpenAI)",
			Type:    model.ChannelTypeNovita,
			BaseURL: DefaultNovitaAPIBase,
			Key:     "novita-key",
			Status:  model.ChannelStatusEnabled,
			Models:  []string{"stale-openai"},
		},
		{
			Name:    "Novita (Anthropic)",
			Type:    model.ChannelTypeAnthropic,
			BaseURL: DefaultNovitaAnthropicBase,
			Key:     "novita-key",
			Status:  model.ChannelStatusEnabled,
			Models:  []string{"stale-claude"},
		},
		{
			Name:    "Novita (Multimodal)",
			Type:    model.ChannelTypeNovitaMultimodal,
			BaseURL: DefaultNovitaMultimodalBase,
			Key:     "novita-key",
			Status:  model.ChannelStatusEnabled,
			Models:  []string{"stale-flux"},
		},
	}

	for i := range channels {
		if err := model.DB.Create(&channels[i]).Error; err != nil {
			t.Fatalf("failed to seed channel %q: %v", channels[i].Name, err)
		}
	}
}

func novitaModelsByType(t *testing.T) map[model.ChannelType][]string {
	t.Helper()

	var got []model.Channel
	if err := model.DB.Order("id asc").Find(&got).Error; err != nil {
		t.Fatalf("failed to load channels: %v", err)
	}

	out := make(map[model.ChannelType][]string, len(got))
	for _, ch := range got {
		out[ch.Type] = append([]string(nil), ch.Models...)
	}

	return out
}

// Regression: multimodal API fetch failure must not wipe the multimodal channel.
func TestEnsureNovitaChannelsFromModels_SkipMultimodalPreservesChannel(t *testing.T) {
	setupNovitaChannelTestDB(t)
	seedNovitaChannelsWithModels(t)

	_, err := ensureNovitaChannelsFromModels(
		[]string{"claude-sonnet-4-20250514"},
		[]string{"deepseek-v3"},
		nil,
		false, // skipChatUpdate
		true,  // skipMultimodalUpdate
		false,
		nil, nil, NovitaConfigResult{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := novitaModelsByType(t)

	if want := []string{
		"claude-sonnet-4-20250514",
	}; !slices.Equal(
		got[model.ChannelTypeAnthropic],
		want,
	) {
		t.Errorf("anthropic Models = %v, want %v", got[model.ChannelTypeAnthropic], want)
	}

	if want := []string{"deepseek-v3"}; !slices.Equal(got[model.ChannelTypeNovita], want) {
		t.Errorf("openai Models = %v, want %v", got[model.ChannelTypeNovita], want)
	}

	if want := []string{"stale-flux"}; !slices.Equal(got[model.ChannelTypeNovitaMultimodal], want) {
		t.Errorf(
			"multimodal Models = %v, want preserved %v",
			got[model.ChannelTypeNovitaMultimodal],
			want,
		)
	}
}

// Regression: chat API fetch failure must not wipe OpenAI/Anthropic channels.
func TestEnsureNovitaChannelsFromModels_SkipChatPreservesChannels(t *testing.T) {
	setupNovitaChannelTestDB(t)
	seedNovitaChannelsWithModels(t)

	_, err := ensureNovitaChannelsFromModels(
		nil, nil,
		[]string{"flux-schnell"},
		true,  // skipChatUpdate
		false, // skipMultimodalUpdate
		false,
		nil, nil, NovitaConfigResult{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := novitaModelsByType(t)

	if want := []string{"stale-claude"}; !slices.Equal(got[model.ChannelTypeAnthropic], want) {
		t.Errorf("anthropic Models = %v, want preserved %v", got[model.ChannelTypeAnthropic], want)
	}

	if want := []string{"stale-openai"}; !slices.Equal(got[model.ChannelTypeNovita], want) {
		t.Errorf("openai Models = %v, want preserved %v", got[model.ChannelTypeNovita], want)
	}

	if want := []string{
		"flux-schnell",
	}; !slices.Equal(
		got[model.ChannelTypeNovitaMultimodal],
		want,
	) {
		t.Errorf("multimodal Models = %v, want %v", got[model.ChannelTypeNovitaMultimodal], want)
	}
}

// Regression: virtual WebSearch models (novita-tavily-search) are declared
// only in the adaptor ModelList and never returned by /v1/models.
// EnsureNovitaChannels must merge them into the OpenAI channel Models list so
// /v1/web-search routing keeps working. See commit d253822.
func TestEnsureNovitaChannels_InjectsVirtualWebSearchModels(t *testing.T) {
	setupNovitaChannelTestDB(t)
	seedNovitaChannelsWithModels(t)

	remote := []NovitaModelV2{
		{
			ID:        "deepseek-v3",
			ModelType: "chat",
			Endpoints: []string{"chat/completions"},
			Status:    NovitaModelStatusAvailable,
		},
	}

	_, err := EnsureNovitaChannels(
		false, nil, nil,
		NovitaConfigResult{},
		remote,
		[]string{"flux-schnell"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := novitaModelsByType(t)

	virtual := novitarelay.VirtualWebSearchModels()
	if len(virtual) == 0 {
		t.Fatalf("expected adaptor to declare at least one virtual WebSearch model")
	}

	want := append([]string{"deepseek-v3"}, virtual...)
	slices.Sort(want)

	if !slices.Equal(got[model.ChannelTypeNovita], want) {
		t.Errorf("openai Models = %v, want %v", got[model.ChannelTypeNovita], want)
	}

	for _, name := range virtual {
		if !slices.Contains(got[model.ChannelTypeNovita], name) {
			t.Errorf("virtual WebSearch model %q missing from openai channel Models", name)
		}
	}
}

// Regression: when the upstream chat fetch returns nothing (skipChatUpdate),
// the virtual-model injection must NOT fire — channel Models should be
// preserved verbatim so a transient upstream failure can't overwrite the list
// with only virtual models.
func TestEnsureNovitaChannels_SkipChatDoesNotInjectVirtuals(t *testing.T) {
	setupNovitaChannelTestDB(t)
	seedNovitaChannelsWithModels(t)

	_, err := EnsureNovitaChannels(
		false, nil, nil,
		NovitaConfigResult{},
		nil, // empty remote → skipChatUpdate=true
		[]string{"flux-schnell"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := novitaModelsByType(t)

	if want := []string{"stale-openai"}; !slices.Equal(got[model.ChannelTypeNovita], want) {
		t.Errorf("openai Models = %v, want preserved %v", got[model.ChannelTypeNovita], want)
	}
}

// Startup refresh: both sources empty means preserve all channel Models.
func TestEnsureNovitaChannelsFromModels_SkipBothPreservesAll(t *testing.T) {
	setupNovitaChannelTestDB(t)
	seedNovitaChannelsWithModels(t)

	_, err := ensureNovitaChannelsFromModels(
		nil, nil, nil,
		true, true,
		false,
		nil, nil, NovitaConfigResult{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := novitaModelsByType(t)

	if want := []string{"stale-claude"}; !slices.Equal(got[model.ChannelTypeAnthropic], want) {
		t.Errorf("anthropic Models = %v, want preserved %v", got[model.ChannelTypeAnthropic], want)
	}

	if want := []string{"stale-openai"}; !slices.Equal(got[model.ChannelTypeNovita], want) {
		t.Errorf("openai Models = %v, want preserved %v", got[model.ChannelTypeNovita], want)
	}

	if want := []string{"stale-flux"}; !slices.Equal(got[model.ChannelTypeNovitaMultimodal], want) {
		t.Errorf(
			"multimodal Models = %v, want preserved %v",
			got[model.ChannelTypeNovitaMultimodal],
			want,
		)
	}
}
