package adaptors

import (
	"fmt"

	"github.com/labring/aiproxy/core/model"
)

var knownChannelConfigKeys = map[string]struct{}{
	model.ChannelConfigPathBaseMapKey:                     {},
	model.ChannelConfigAllowPassthroughUnknown:            {},
	model.ChannelConfigPurePassthrough:                    {},
	model.ChannelConfigPassthroughProtocol:                {},
	model.ChannelConfigPassthroughAuthScheme:              {},
	model.ChannelConfigPassthroughPathPolicy:              {},
	model.ChannelConfigPassthroughModelMappingPolicy:      {},
	model.ChannelConfigPassthroughEndpointFamilies:        {},
	model.ChannelConfigAdaptedPassthroughEndpointFamilies: {},
	model.ChannelConfigRouteKind:                          {},
}

func ValidateChannelConfigs(channelType model.ChannelType, configs model.ChannelConfigs) error {
	if len(configs) == 0 {
		return nil
	}

	a, ok := GetAdaptor(channelType)
	if !ok {
		return fmt.Errorf("invalid channel type: %d", channelType)
	}

	allowed := configKeysFromSchema(a.Metadata().ConfigSchema)

	for key := range configs {
		if key == model.ChannelConfigRouteKind {
			continue
		}

		if _, known := knownChannelConfigKeys[key]; !known {
			continue
		}

		if _, ok := allowed[key]; !ok {
			return fmt.Errorf(
				"config %q is not supported by channel type %s(%d)",
				key,
				channelType.String(),
				channelType,
			)
		}
	}

	return nil
}

func configKeysFromSchema(schema map[string]any) map[string]struct{} {
	result := map[string]struct{}{}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return result
	}

	for key := range properties {
		result[key] = struct{}{}
	}

	return result
}
