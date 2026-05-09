package model

type PassthroughProtocol string

const (
	PassthroughProtocolOpenAI         PassthroughProtocol = "openai"
	PassthroughProtocolAnthropic      PassthroughProtocol = "anthropic"
	PassthroughProtocolGemini         PassthroughProtocol = "gemini"
	PassthroughProtocolNativeV3       PassthroughProtocol = "native_v3"
	PassthroughProtocolNativePPIOV3   PassthroughProtocol = "native_ppio_v3"
	PassthroughProtocolNativeNovitaV3 PassthroughProtocol = "native_novita_v3"
)

type PassthroughAuthScheme string

const (
	PassthroughAuthSchemeBearer      PassthroughAuthScheme = "bearer"
	PassthroughAuthSchemeXAPIKey     PassthroughAuthScheme = "x-api-key"
	PassthroughAuthSchemeXGoogAPIKey PassthroughAuthScheme = "x-goog-api-key"
	PassthroughAuthSchemeCustom      PassthroughAuthScheme = "custom"
)

type PassthroughPathPolicy string

const (
	PassthroughPathPolicyPreserve    PassthroughPathPolicy = "preserve"
	PassthroughPathPolicyStripV1     PassthroughPathPolicy = "strip_v1"
	PassthroughPathPolicyPathBaseMap PassthroughPathPolicy = "path_base_map"
)

type PassthroughModelMappingPolicy string

const (
	PassthroughModelMappingNone      PassthroughModelMappingPolicy = "none"
	PassthroughModelMappingBodyModel PassthroughModelMappingPolicy = "body_model"
	PassthroughModelMappingPathModel PassthroughModelMappingPolicy = "path_model"
)

type EndpointFamily string

const (
	EndpointFamilyChat                  EndpointFamily = "chat"
	EndpointFamilyCompletions           EndpointFamily = "completions"
	EndpointFamilyResponses             EndpointFamily = "responses"
	EndpointFamilyEmbeddings            EndpointFamily = "embeddings"
	EndpointFamilyImages                EndpointFamily = "images"
	EndpointFamilyAudio                 EndpointFamily = "audio"
	EndpointFamilyRerank                EndpointFamily = "rerank"
	EndpointFamilyModerations           EndpointFamily = "moderations"
	EndpointFamilyVideoJobs             EndpointFamily = "video_jobs"
	EndpointFamilyMessages              EndpointFamily = "messages"
	EndpointFamilyGeminiGenerateContent EndpointFamily = "gemini_generate_content"
	EndpointFamilyNativeV3              EndpointFamily = "native_v3"
	EndpointFamilyWebSearch             EndpointFamily = "web_search"
)

type RoutingPolicy string

const (
	RoutingPolicyPureOnly        RoutingPolicy = "pure_only"
	RoutingPolicyPureThenConvert RoutingPolicy = "pure_then_convert"
	RoutingPolicyConvertOnly     RoutingPolicy = "convert_only"
)

type RouteKind string

const (
	RouteKindPurePassthrough    RouteKind = "pure_passthrough"
	RouteKindAdaptedPassthrough RouteKind = "adapted_passthrough"
	RouteKindConversion         RouteKind = "conversion"
)

type RequestShape struct {
	Protocol       PassthroughProtocol `json:"protocol,omitempty"`
	EndpointFamily EndpointFamily      `json:"endpointFamily,omitempty"`
	Method         string              `json:"method,omitempty"`
	OriginalPath   string              `json:"originalPath,omitempty"`
}

type ChannelCapability struct {
	PurePassthrough         bool                          `json:"purePassthrough,omitempty"`
	Protocol                PassthroughProtocol           `json:"protocol,omitempty"`
	AuthScheme              PassthroughAuthScheme         `json:"authScheme,omitempty"`
	PathPolicy              PassthroughPathPolicy         `json:"pathPolicy,omitempty"`
	ModelMappingPolicy      PassthroughModelMappingPolicy `json:"modelMappingPolicy,omitempty"`
	EndpointFamilies        []EndpointFamily              `json:"endpointFamilies,omitempty"`
	AdaptedEndpointFamilies []EndpointFamily              `json:"adaptedEndpointFamilies,omitempty"`
}

const (
	ChannelConfigPassthroughProtocol                = "passthrough_protocol"
	ChannelConfigPassthroughAuthScheme              = "passthrough_auth_scheme"
	ChannelConfigPassthroughPathPolicy              = "passthrough_path_policy"
	ChannelConfigPassthroughModelMappingPolicy      = "passthrough_model_mapping_policy"
	ChannelConfigPassthroughEndpointFamilies        = "passthrough_endpoint_families"
	ChannelConfigAdaptedPassthroughEndpointFamilies = "adapted_passthrough_endpoint_families"
	ChannelConfigRouteKind                          = "route_kind"
)

func InferPassthroughCapability(channel *Channel, defaults ...ChannelCapability) ChannelCapability {
	if channel == nil {
		return ChannelCapability{}
	}

	var capability ChannelCapability
	if len(defaults) > 0 {
		capability = defaults[0]
	}

	applyPassthroughCapabilityConfig(&capability, channel.Configs)

	if !capability.PurePassthrough {
		return ChannelCapability{}
	}

	return capability
}

func SupportsPurePassthrough(channel *Channel, shape RequestShape, defaults ...ChannelCapability) bool {
	capability := InferPassthroughCapability(channel, defaults...)
	if !capability.PurePassthrough || !passthroughProtocolMatches(capability.Protocol, shape.Protocol) {
		return false
	}

	return endpointFamilyMatches(capability.EndpointFamilies, shape.EndpointFamily)
}

func SupportsAdaptedPassthrough(channel *Channel, shape RequestShape, defaults ...ChannelCapability) bool {
	capability := InferPassthroughCapability(channel, defaults...)
	if !capability.PurePassthrough || !passthroughProtocolMatches(capability.Protocol, shape.Protocol) {
		return false
	}

	return endpointFamilyMatches(capability.AdaptedEndpointFamilies, shape.EndpointFamily)
}

func endpointFamilyMatches(families []EndpointFamily, want EndpointFamily) bool {
	for _, family := range families {
		if family == want {
			return true
		}
	}

	return false
}

func passthroughProtocolMatches(channelProtocol, requestProtocol PassthroughProtocol) bool {
	if channelProtocol == requestProtocol {
		return true
	}

	if requestProtocol == PassthroughProtocolNativeV3 {
		return channelProtocol == PassthroughProtocolNativePPIOV3 ||
			channelProtocol == PassthroughProtocolNativeNovitaV3
	}

	return false
}

func applyPassthroughCapabilityConfig(capability *ChannelCapability, configs ChannelConfigs) {
	if configs == nil {
		return
	}

	if configs.GetBool(ChannelConfigPurePassthrough) ||
		RouteKind(configs.GetString(ChannelConfigRouteKind)) == RouteKindPurePassthrough {
		capability.PurePassthrough = true
	}

	if v := configs.GetString(ChannelConfigPassthroughProtocol); v != "" {
		capability.Protocol = PassthroughProtocol(v)
	}

	if v := configs.GetString(ChannelConfigPassthroughAuthScheme); v != "" {
		capability.AuthScheme = PassthroughAuthScheme(v)
	}

	if v := configs.GetString(ChannelConfigPassthroughPathPolicy); v != "" {
		capability.PathPolicy = PassthroughPathPolicy(v)
	}

	if v := configs.GetString(ChannelConfigPassthroughModelMappingPolicy); v != "" {
		capability.ModelMappingPolicy = PassthroughModelMappingPolicy(v)
	}

	if families := configs.GetStringSlice(ChannelConfigPassthroughEndpointFamilies); len(families) > 0 {
		capability.EndpointFamilies = make([]EndpointFamily, 0, len(families))
		for _, family := range families {
			capability.EndpointFamilies = append(capability.EndpointFamilies, EndpointFamily(family))
		}
	}

	if families := configs.GetStringSlice(ChannelConfigAdaptedPassthroughEndpointFamilies); len(families) > 0 {
		capability.AdaptedEndpointFamilies = make([]EndpointFamily, 0, len(families))
		for _, family := range families {
			capability.AdaptedEndpointFamilies = append(
				capability.AdaptedEndpointFamilies,
				EndpointFamily(family),
			)
		}
	}
}
