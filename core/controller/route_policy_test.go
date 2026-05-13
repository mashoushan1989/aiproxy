package controller

import (
	"testing"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

func TestFilterChannelsWithRoutingPolicyPureOnlyRejectsProtocolConversion(t *testing.T) {
	channels := []*model.Channel{
		{
			ID:     1,
			Type:   model.ChannelTypePPIO,
			Status: model.ChannelStatusEnabled,
		},
	}

	got := filterChannelsWithRoutingPolicy(
		channels,
		mode.Anthropic,
		model.RoutingPolicyPureOnly,
		map[int64]float64{},
		0,
	)
	if len(got) != 0 {
		t.Fatalf("expected no pure Anthropic channel, got %d", len(got))
	}
}

func TestFilterChannelsWithRoutingPolicyPureThenConvertRejectsPassthroughConversion(t *testing.T) {
	channels := []*model.Channel{
		{
			ID:     1,
			Type:   model.ChannelTypePPIO,
			Status: model.ChannelStatusEnabled,
		},
	}

	got := filterChannelsWithRoutingPolicy(
		channels,
		mode.Anthropic,
		model.RoutingPolicyPureThenConvert,
		map[int64]float64{},
		0,
	)
	if len(got) != 0 {
		t.Fatalf("expected no fallback channel, got %d", len(got))
	}
}

func TestFilterChannelsWithRoutingPolicyPureThenConvertFallsBackToConvertingAdaptor(t *testing.T) {
	channels := []*model.Channel{
		{
			ID:     1,
			Type:   model.ChannelTypeOpenAI,
			Status: model.ChannelStatusEnabled,
		},
	}

	got := filterChannelsWithRoutingPolicy(
		channels,
		mode.Anthropic,
		model.RoutingPolicyPureThenConvert,
		map[int64]float64{},
		0,
	)
	if len(got) != 1 {
		t.Fatalf("expected conversion fallback channel, got %d", len(got))
	}

	if got[0].ID != 1 {
		t.Fatalf("expected OpenAI conversion channel, got channel id %d", got[0].ID)
	}
}

func TestFilterChannelsWithRoutingPolicyKeepsAdaptedWebSearchPassthrough(t *testing.T) {
	channels := []*model.Channel{
		{
			ID:     1,
			Type:   model.ChannelTypePPIO,
			Status: model.ChannelStatusEnabled,
		},
	}

	got := filterChannelsWithRoutingPolicy(
		channels,
		mode.WebSearch,
		model.RoutingPolicyPureThenConvert,
		map[int64]float64{},
		0,
	)
	if len(got) != 1 {
		t.Fatalf("expected adapted passthrough channel, got %d", len(got))
	}

	if got[0].ID != 1 {
		t.Fatalf("expected PPIO adapted passthrough channel, got channel id %d", got[0].ID)
	}
}

func TestFilterChannelsWithRoutingPolicyConvertOnlySkipsPure(t *testing.T) {
	channels := []*model.Channel{
		{
			ID:     1,
			Type:   model.ChannelTypeAnthropic,
			Status: model.ChannelStatusEnabled,
			Configs: model.ChannelConfigs{
				model.ChannelConfigPurePassthrough: true,
			},
		},
		{
			ID:     2,
			Type:   model.ChannelTypeOpenAI,
			Status: model.ChannelStatusEnabled,
		},
	}

	got := filterChannelsWithRoutingPolicy(
		channels,
		mode.Anthropic,
		model.RoutingPolicyConvertOnly,
		map[int64]float64{},
		0,
	)
	if len(got) != 1 {
		t.Fatalf("expected only conversion candidate, got %d", len(got))
	}

	if got[0].ID != 2 {
		t.Fatalf("expected OpenAI conversion channel, got channel id %d", got[0].ID)
	}
}

func TestFilterChannelsWithRoutingPolicyConversionRouteKindKeepsNativeAdaptor(t *testing.T) {
	channels := []*model.Channel{
		{
			ID:     1,
			Type:   model.ChannelTypeOpenAI,
			Status: model.ChannelStatusEnabled,
			Configs: model.ChannelConfigs{
				model.ChannelConfigRouteKind: string(model.RouteKindConversion),
			},
		},
	}

	got := filterChannelsWithRoutingPolicy(
		channels,
		mode.ChatCompletions,
		model.RoutingPolicyPureThenConvert,
		map[int64]float64{},
		0,
	)
	if len(got) != 1 {
		t.Fatalf("conversion route_kind should keep native adaptor route, got %d", len(got))
	}

	if got[0].ID != 1 {
		t.Fatalf("expected OpenAI native adaptor channel, got channel id %d", got[0].ID)
	}
}

func TestFilterChannelsWithRouteKindOverride(t *testing.T) {
	ppioAdaptedOnly := &model.Channel{
		ID:     1,
		Type:   model.ChannelTypePPIO,
		Status: model.ChannelStatusEnabled,
		Configs: model.ChannelConfigs{
			model.ChannelConfigRouteKind: string(model.RouteKindAdaptedPassthrough),
		},
	}

	if got := filterChannelsWithRoutingPolicy(
		[]*model.Channel{ppioAdaptedOnly},
		mode.ChatCompletions,
		model.RoutingPolicyPureThenConvert,
		map[int64]float64{},
		0,
	); len(got) != 0 {
		t.Fatalf("adapted-only PPIO must not route pure chat requests, got %d", len(got))
	}

	if got := filterChannelsWithRoutingPolicy(
		[]*model.Channel{ppioAdaptedOnly},
		mode.WebSearch,
		model.RoutingPolicyPureThenConvert,
		map[int64]float64{},
		0,
	); len(got) != 1 {
		t.Fatalf("adapted-only PPIO should route adapted WebSearch requests, got %d", len(got))
	}
}
