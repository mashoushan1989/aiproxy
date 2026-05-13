package adaptors

import (
	"testing"

	"github.com/labring/aiproxy/core/model"
)

func TestChannelMetasExposePassthroughCapability(t *testing.T) {
	tests := []struct {
		name        string
		channelType model.ChannelType
		wantPure    bool
		wantProto   model.PassthroughProtocol
		wantFamily  model.EndpointFamily
		wantAdapted model.EndpointFamily
	}{
		{
			name:        "ppio openai-compatible",
			channelType: model.ChannelTypePPIO,
			wantPure:    true,
			wantProto:   model.PassthroughProtocolOpenAI,
			wantFamily:  model.EndpointFamilyResponses,
			wantAdapted: model.EndpointFamilyWebSearch,
		},
		{
			name:        "novita openai-compatible",
			channelType: model.ChannelTypeNovita,
			wantPure:    true,
			wantProto:   model.PassthroughProtocolOpenAI,
			wantFamily:  model.EndpointFamilyResponses,
			wantAdapted: model.EndpointFamilyWebSearch,
		},
		{
			name:        "anthropic template",
			channelType: model.ChannelTypeAnthropic,
			wantPure:    false,
			wantProto:   model.PassthroughProtocolAnthropic,
			wantFamily:  model.EndpointFamilyMessages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, ok := ChannelMetas[tt.channelType]
			if !ok {
				t.Fatalf("missing channel meta for %s", tt.channelType)
			}
			if meta.PassthroughCapability == nil {
				t.Fatal("missing passthrough capability")
			}

			got := meta.PassthroughCapability
			if got.PurePassthrough != tt.wantPure {
				t.Fatalf("PurePassthrough: want %v, got %v", tt.wantPure, got.PurePassthrough)
			}
			if got.Protocol != tt.wantProto {
				t.Fatalf("Protocol: want %q, got %q", tt.wantProto, got.Protocol)
			}
			if !containsEndpointFamily(got.EndpointFamilies, tt.wantFamily) {
				t.Fatalf("EndpointFamilies: want to include %q, got %v", tt.wantFamily, got.EndpointFamilies)
			}
			if tt.wantAdapted != "" && !containsEndpointFamily(got.AdaptedEndpointFamilies, tt.wantAdapted) {
				t.Fatalf("AdaptedEndpointFamilies: want to include %q, got %v", tt.wantAdapted, got.AdaptedEndpointFamilies)
			}
		})
	}
}

func containsEndpointFamily(families []model.EndpointFamily, want model.EndpointFamily) bool {
	for _, family := range families {
		if family == want {
			return true
		}
	}

	return false
}
