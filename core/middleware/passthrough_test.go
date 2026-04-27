package middleware

import (
	"testing"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

func unknownPassthroughChannel(id int, channelType model.ChannelType) *model.Channel {
	return &model.Channel{
		ID:     id,
		Type:   channelType,
		Status: model.ChannelStatusEnabled,
		Configs: model.ChannelConfigs{
			model.ChannelConfigAllowPassthroughUnknown: true,
		},
	}
}

func pureUnknownPassthroughChannel(id int, channelType model.ChannelType) *model.Channel {
	channel := unknownPassthroughChannel(id, channelType)
	channel.Configs[model.ChannelConfigPurePassthrough] = true

	return channel
}

func TestHasUnknownPassthroughForModeDomestic(t *testing.T) {
	mc := &model.ModelCaches{
		PassthroughChannelsBySet: map[string][]*model.Channel{
			model.ChannelDefaultSet: {
				pureUnknownPassthroughChannel(1, model.ChannelTypeAnthropic),
				unknownPassthroughChannel(2, model.ChannelTypePPIOMultimodal),
			},
		},
	}

	if hasUnknownPassthroughForMode(mc, []string{model.ChannelDefaultSet}, mode.Responses) {
		t.Fatal("responses must not be allowed by anthropic or native multimodal passthrough")
	}

	if !hasUnknownPassthroughForMode(mc, []string{model.ChannelDefaultSet}, mode.Anthropic) {
		t.Fatal("anthropic pure passthrough should allow unknown anthropic models")
	}

	if !hasUnknownPassthroughForMode(mc, []string{model.ChannelDefaultSet}, mode.PPIONative) {
		t.Fatal("native multimodal passthrough should allow unknown native multimodal models")
	}
}

func TestHasUnknownPassthroughForModeDomesticOpenAIResponses(t *testing.T) {
	mc := &model.ModelCaches{
		PassthroughChannelsBySet: map[string][]*model.Channel{
			model.ChannelDefaultSet: {
				unknownPassthroughChannel(3, model.ChannelTypePPIO),
			},
		},
	}

	if !hasUnknownPassthroughForMode(mc, []string{model.ChannelDefaultSet}, mode.Responses) {
		t.Fatal("ppio openai passthrough should allow unknown responses models")
	}
}

func TestHasUnknownPassthroughForModeOverseas(t *testing.T) {
	mc := &model.ModelCaches{
		PassthroughChannelsBySet: map[string][]*model.Channel{
			"overseas": {
				pureUnknownPassthroughChannel(1, model.ChannelTypeAnthropic),
				unknownPassthroughChannel(2, model.ChannelTypeNovitaMultimodal),
			},
			model.ChannelDefaultSet: {
				unknownPassthroughChannel(3, model.ChannelTypePPIO),
			},
		},
	}

	if hasUnknownPassthroughForMode(mc, []string{"overseas"}, mode.Responses) {
		t.Fatal("overseas responses must not be allowed by anthropic or native multimodal passthrough")
	}

	if !hasUnknownPassthroughForMode(mc, []string{"overseas"}, mode.Anthropic) {
		t.Fatal("overseas anthropic pure passthrough should allow unknown anthropic models")
	}

	if !hasUnknownPassthroughForMode(mc, []string{"overseas"}, mode.PPIONative) {
		t.Fatal("overseas native multimodal passthrough should allow unknown native multimodal models")
	}

	if !hasUnknownPassthroughForMode(
		mc,
		[]string{"overseas", model.ChannelDefaultSet},
		mode.Responses,
	) {
		t.Fatal("overseas soft fallback should allow responses when default set has openai passthrough")
	}
}

func TestHasUnknownPassthroughForModeOverseasOpenAIResponses(t *testing.T) {
	mc := &model.ModelCaches{
		PassthroughChannelsBySet: map[string][]*model.Channel{
			"overseas": {
				unknownPassthroughChannel(1, model.ChannelTypeNovita),
			},
		},
	}

	if !hasUnknownPassthroughForMode(mc, []string{"overseas"}, mode.Responses) {
		t.Fatal("novita openai passthrough should allow unknown responses models")
	}
}
