package controller

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptors"
	"github.com/labring/aiproxy/core/relay/mode"
)

func filterChannelsWithRoutingPolicy(
	channels []*model.Channel,
	mode mode.Mode,
	routingPolicy model.RoutingPolicy,
	errorRates map[int64]float64,
	maxErrorRate float64,
	ignoreChannel ...map[int64]struct{},
) []*model.Channel {
	var pure, adaptedNative, conversion []*model.Channel
	shape := model.InferRequestShape("", "", mode)

	for _, channel := range channels {
		if channel.Status != model.ChannelStatusEnabled {
			continue
		}

		a, ok := adaptors.GetAdaptor(channel.Type)
		if !ok || !a.SupportMode(mode) {
			continue
		}

		if shouldIgnoreRouteChannel(channel.ID, ignoreChannel...) {
			continue
		}

		capability := a.Metadata().PassthroughCapability
		routeKind := model.RouteKind(channel.Configs.GetString(model.ChannelConfigRouteKind))

		if routeKind != model.RouteKindAdaptedPassthrough &&
			routeKind != model.RouteKindConversion &&
			model.SupportsPurePassthrough(channel, shape, capability) {
			pure = append(pure, channel)
			continue
		}

		if routingPolicy == model.RoutingPolicyPureOnly {
			continue
		}

		if model.IsPassthroughChannel(channel.Type) {
			if routeKind != model.RouteKindPurePassthrough &&
				routeKind != model.RouteKindConversion &&
				model.SupportsAdaptedPassthrough(channel, shape, capability) {
				adaptedNative = append(adaptedNative, channel)
			}

			continue
		}

		if routeKind != model.RouteKindPurePassthrough &&
			routeKind != model.RouteKindAdaptedPassthrough &&
			(routeKind == model.RouteKindConversion || isConversionRoute(channel, a, mode)) {
			conversion = append(conversion, channel)
		} else if routeKind != model.RouteKindConversion {
			adaptedNative = append(adaptedNative, channel)
		}
	}

	if routingPolicy == model.RoutingPolicyConvertOnly {
		return filterRouteCandidates(conversion, errorRates, maxErrorRate)
	}

	if len(pure) > 0 {
		return filterRouteCandidates(pure, errorRates, maxErrorRate)
	}

	if routingPolicy == model.RoutingPolicyPureOnly {
		return nil
	}

	if len(adaptedNative) > 0 {
		return filterRouteCandidates(adaptedNative, errorRates, maxErrorRate)
	}

	return filterRouteCandidates(conversion, errorRates, maxErrorRate)
}

func shouldIgnoreRouteChannel(channelID int, ignoreChannel ...map[int64]struct{}) bool {
	chid := int64(channelID)
	for _, ignores := range ignoreChannel {
		if ignores == nil {
			continue
		}

		if _, ok := ignores[chid]; ok {
			return true
		}
	}

	return false
}

func isConversionRoute(channel *model.Channel, a adaptor.Adaptor, mode mode.Mode) bool {
	checker, isChecker := a.(adaptor.NativeModeChecker)
	if !isChecker || checker.NativeMode(mode) {
		return false
	}

	return !model.IsPassthroughChannel(channel.Type)
}

func filterRouteCandidates(
	channels []*model.Channel,
	errorRates map[int64]float64,
	maxErrorRate float64,
) []*model.Channel {
	if len(channels) == 0 {
		return nil
	}

	if maxErrorRate != 0 {
		if healthy := filterByErrorRate(channels, errorRates, maxErrorRate); len(healthy) > 0 {
			return healthy
		}

		return channels
	}

	return channels
}
