package middleware

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptors"
	"github.com/labring/aiproxy/core/relay/mode"
)

func hasUnknownPassthroughForMode(caches *model.ModelCaches, sets []string, m mode.Mode) bool {
	if caches == nil {
		return false
	}

	for _, set := range sets {
		for _, channel := range caches.PassthroughChannelsBySet[set] {
			if supportsUnknownPassthroughMode(channel, m) {
				return true
			}
		}
	}

	return false
}

func supportsUnknownPassthroughMode(channel *model.Channel, m mode.Mode) bool {
	if channel == nil ||
		channel.Status != model.ChannelStatusEnabled ||
		!channel.Configs.GetBool(model.ChannelConfigAllowPassthroughUnknown) {
		return false
	}

	if !model.IsPassthroughChannel(channel.Type) &&
		!channel.Configs.GetBool(model.ChannelConfigPurePassthrough) {
		return false
	}

	a, ok := adaptors.GetAdaptor(channel.Type)
	if !ok || !a.SupportMode(m) {
		return false
	}

	if checker, ok := a.(adaptor.NativeModeChecker); ok {
		return checker.NativeMode(m)
	}

	return true
}
