package adaptors

import (
	"testing"

	"github.com/labring/aiproxy/core/model"
)

func TestValidateChannelConfigsRejectsKnownUnsupportedKey(t *testing.T) {
	err := ValidateChannelConfigs(model.ChannelTypePPIO, model.ChannelConfigs{
		model.ChannelConfigPurePassthrough: true,
	})
	if err == nil {
		t.Fatal("expected unsupported pure_passthrough config to be rejected")
	}
}

func TestValidateChannelConfigsAllowsKnownSupportedKey(t *testing.T) {
	err := ValidateChannelConfigs(model.ChannelTypePPIO, model.ChannelConfigs{
		model.ChannelConfigAllowPassthroughUnknown: true,
	})
	if err != nil {
		t.Fatalf("expected supported config to be allowed, got %v", err)
	}
}

func TestValidateChannelConfigsAllowsUnknownCustomKey(t *testing.T) {
	err := ValidateChannelConfigs(model.ChannelTypePPIO, model.ChannelConfigs{
		"custom_internal_flag": true,
	})
	if err != nil {
		t.Fatalf("expected unknown custom config to remain compatible, got %v", err)
	}
}

func TestValidateChannelConfigsAllowsAnthropicPurePassthrough(t *testing.T) {
	err := ValidateChannelConfigs(model.ChannelTypeAnthropic, model.ChannelConfigs{
		model.ChannelConfigPurePassthrough: true,
	})
	if err != nil {
		t.Fatalf("expected anthropic pure_passthrough config to be allowed, got %v", err)
	}
}
