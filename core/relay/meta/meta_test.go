package meta

import (
	"testing"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

func TestMetaIsPassthroughForPureRouteKind(t *testing.T) {
	channel := &model.Channel{
		Type: model.ChannelTypeAnthropic,
		Configs: model.ChannelConfigs{
			model.ChannelConfigRouteKind: string(model.RouteKindPurePassthrough),
		},
	}

	m := NewMeta(channel, mode.Anthropic, "claude-sonnet-4-20250514", model.ModelConfig{})
	if !m.IsPassthrough() {
		t.Fatal("route_kind=pure_passthrough should mark meta as passthrough")
	}
}
