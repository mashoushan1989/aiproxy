package model

import "testing"

func TestInferPassthroughCapabilityDefaults(t *testing.T) {
	tests := []struct {
		name       string
		channel    *Channel
		defaults   ChannelCapability
		wantPure   bool
		wantProto  PassthroughProtocol
		wantAuth   PassthroughAuthScheme
		wantPath   PassthroughPathPolicy
		wantFamily []EndpointFamily
	}{
		{
			name: "ppio openai compatible",
			channel: &Channel{
				Type: ChannelTypePPIO,
			},
			defaults: ChannelCapability{
				PurePassthrough: true,
				Protocol:        PassthroughProtocolOpenAI,
				AuthScheme:      PassthroughAuthSchemeBearer,
				PathPolicy:      PassthroughPathPolicyStripV1,
				EndpointFamilies: []EndpointFamily{
					EndpointFamilyChat,
					EndpointFamilyCompletions,
					EndpointFamilyResponses,
					EndpointFamilyEmbeddings,
					EndpointFamilyImages,
					EndpointFamilyAudio,
					EndpointFamilyRerank,
					EndpointFamilyModerations,
					EndpointFamilyVideoJobs,
				},
			},
			wantPure:  true,
			wantProto: PassthroughProtocolOpenAI,
			wantAuth:  PassthroughAuthSchemeBearer,
			wantPath:  PassthroughPathPolicyStripV1,
			wantFamily: []EndpointFamily{
				EndpointFamilyChat,
				EndpointFamilyCompletions,
				EndpointFamilyResponses,
				EndpointFamilyEmbeddings,
				EndpointFamilyImages,
				EndpointFamilyAudio,
				EndpointFamilyRerank,
				EndpointFamilyModerations,
				EndpointFamilyVideoJobs,
			},
		},
		{
			name: "novita native multimodal",
			channel: &Channel{
				Type: ChannelTypeNovitaMultimodal,
			},
			defaults: ChannelCapability{
				PurePassthrough: true,
				Protocol:        PassthroughProtocolNativeNovitaV3,
				AuthScheme:      PassthroughAuthSchemeBearer,
				PathPolicy:      PassthroughPathPolicyPreserve,
				EndpointFamilies: []EndpointFamily{
					EndpointFamilyNativeV3,
				},
			},
			wantPure:  true,
			wantProto: PassthroughProtocolNativeNovitaV3,
			wantAuth:  PassthroughAuthSchemeBearer,
			wantPath:  PassthroughPathPolicyPreserve,
			wantFamily: []EndpointFamily{
				EndpointFamilyNativeV3,
			},
		},
		{
			name: "anthropic without pure passthrough",
			channel: &Channel{
				Type: ChannelTypeAnthropic,
			},
			defaults: ChannelCapability{
				Protocol:   PassthroughProtocolAnthropic,
				AuthScheme: PassthroughAuthSchemeXAPIKey,
				PathPolicy: PassthroughPathPolicyStripV1,
				EndpointFamilies: []EndpointFamily{
					EndpointFamilyMessages,
				},
			},
			wantPure:   false,
			wantProto:  "",
			wantAuth:   "",
			wantPath:   "",
			wantFamily: nil,
		},
		{
			name: "anthropic pure passthrough",
			channel: &Channel{
				Type: ChannelTypeAnthropic,
				Configs: ChannelConfigs{
					ChannelConfigPurePassthrough: true,
				},
			},
			defaults: ChannelCapability{
				Protocol:   PassthroughProtocolAnthropic,
				AuthScheme: PassthroughAuthSchemeXAPIKey,
				PathPolicy: PassthroughPathPolicyStripV1,
				EndpointFamilies: []EndpointFamily{
					EndpointFamilyMessages,
				},
			},
			wantPure:  true,
			wantProto: PassthroughProtocolAnthropic,
			wantAuth:  PassthroughAuthSchemeXAPIKey,
			wantPath:  PassthroughPathPolicyStripV1,
			wantFamily: []EndpointFamily{
				EndpointFamilyMessages,
			},
		},
		{
			name: "gemini native explicit pure passthrough",
			channel: &Channel{
				Type: ChannelTypeGoogleGemini,
				Configs: ChannelConfigs{
					ChannelConfigPurePassthrough: true,
				},
			},
			defaults: ChannelCapability{
				Protocol:   PassthroughProtocolGemini,
				AuthScheme: PassthroughAuthSchemeXGoogAPIKey,
				PathPolicy: PassthroughPathPolicyPreserve,
				EndpointFamilies: []EndpointFamily{
					EndpointFamilyGeminiGenerateContent,
					EndpointFamilyEmbeddings,
				},
			},
			wantPure:  true,
			wantProto: PassthroughProtocolGemini,
			wantAuth:  PassthroughAuthSchemeXGoogAPIKey,
			wantPath:  PassthroughPathPolicyPreserve,
			wantFamily: []EndpointFamily{
				EndpointFamilyGeminiGenerateContent,
				EndpointFamilyEmbeddings,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferPassthroughCapability(tt.channel, tt.defaults)

			if got.PurePassthrough != tt.wantPure {
				t.Fatalf("PurePassthrough: want %v, got %v", tt.wantPure, got.PurePassthrough)
			}

			if got.Protocol != tt.wantProto {
				t.Fatalf("Protocol: want %q, got %q", tt.wantProto, got.Protocol)
			}

			if got.AuthScheme != tt.wantAuth {
				t.Fatalf("AuthScheme: want %q, got %q", tt.wantAuth, got.AuthScheme)
			}

			if got.PathPolicy != tt.wantPath {
				t.Fatalf("PathPolicy: want %q, got %q", tt.wantPath, got.PathPolicy)
			}

			if !endpointFamiliesEqual(got.EndpointFamilies, tt.wantFamily) {
				t.Fatalf("EndpointFamilies: want %v, got %v", tt.wantFamily, got.EndpointFamilies)
			}
		})
	}
}

func TestInferPassthroughCapabilityDoesNotHardcodeProviderDefaults(t *testing.T) {
	got := InferPassthroughCapability(&Channel{Type: ChannelTypePPIO})
	if got.PurePassthrough {
		t.Fatal("model layer must not infer PPIO passthrough defaults without adaptor-provided capability")
	}
}

func TestInferPassthroughCapabilityConfigOverrides(t *testing.T) {
	channel := &Channel{
		Type: ChannelTypeOpenAI,
		Configs: ChannelConfigs{
			ChannelConfigPurePassthrough:                    true,
			ChannelConfigPassthroughProtocol:                string(PassthroughProtocolOpenAI),
			ChannelConfigPassthroughAuthScheme:              string(PassthroughAuthSchemeBearer),
			ChannelConfigPassthroughPathPolicy:              string(PassthroughPathPolicyPreserve),
			ChannelConfigPassthroughEndpointFamilies:        []any{string(EndpointFamilyChat), string(EndpointFamilyResponses)},
			ChannelConfigAdaptedPassthroughEndpointFamilies: []any{string(EndpointFamilyWebSearch)},
			ChannelConfigPassthroughModelMappingPolicy:      string(PassthroughModelMappingBodyModel),
		},
	}

	got := InferPassthroughCapability(channel)

	if !got.PurePassthrough {
		t.Fatal("PurePassthrough: want true")
	}

	if got.Protocol != PassthroughProtocolOpenAI {
		t.Fatalf("Protocol: want openai, got %q", got.Protocol)
	}

	if got.PathPolicy != PassthroughPathPolicyPreserve {
		t.Fatalf("PathPolicy: want preserve, got %q", got.PathPolicy)
	}

	wantFamilies := []EndpointFamily{EndpointFamilyChat, EndpointFamilyResponses}
	if !endpointFamiliesEqual(got.EndpointFamilies, wantFamilies) {
		t.Fatalf("EndpointFamilies: want %v, got %v", wantFamilies, got.EndpointFamilies)
	}

	if got.ModelMappingPolicy != PassthroughModelMappingBodyModel {
		t.Fatalf("ModelMappingPolicy: want body_model, got %q", got.ModelMappingPolicy)
	}

	if !endpointFamiliesEqual(got.AdaptedEndpointFamilies, []EndpointFamily{EndpointFamilyWebSearch}) {
		t.Fatalf("AdaptedEndpointFamilies: want [web_search], got %v", got.AdaptedEndpointFamilies)
	}
}

func TestInferPassthroughCapabilityRouteKindPureEnablesPurePassthrough(t *testing.T) {
	channel := &Channel{
		Type: ChannelTypeAnthropic,
		Configs: ChannelConfigs{
			ChannelConfigRouteKind: string(RouteKindPurePassthrough),
		},
	}

	got := InferPassthroughCapability(channel, ChannelCapability{
		Protocol: PassthroughProtocolAnthropic,
		EndpointFamilies: []EndpointFamily{
			EndpointFamilyMessages,
		},
	})
	if !got.PurePassthrough {
		t.Fatal("route_kind=pure_passthrough should enable pure passthrough capability")
	}

	if !SupportsPurePassthrough(channel, RequestShape{
		Protocol:       PassthroughProtocolAnthropic,
		EndpointFamily: EndpointFamilyMessages,
	}, ChannelCapability{
		Protocol: PassthroughProtocolAnthropic,
		EndpointFamilies: []EndpointFamily{
			EndpointFamilyMessages,
		},
	}) {
		t.Fatal("route_kind=pure_passthrough should support Anthropic messages")
	}
}

func TestSupportsPurePassthrough(t *testing.T) {
	ppio := &Channel{Type: ChannelTypePPIO}
	ppioCapability := ChannelCapability{
		PurePassthrough: true,
		Protocol:        PassthroughProtocolOpenAI,
		EndpointFamilies: []EndpointFamily{
			EndpointFamilyResponses,
		},
		AdaptedEndpointFamilies: []EndpointFamily{
			EndpointFamilyWebSearch,
		},
	}
	anthropicPure := &Channel{
		Type: ChannelTypeAnthropic,
		Configs: ChannelConfigs{
			ChannelConfigPurePassthrough: true,
		},
	}
	anthropicCapability := ChannelCapability{
		Protocol: PassthroughProtocolAnthropic,
		EndpointFamilies: []EndpointFamily{
			EndpointFamilyMessages,
		},
	}

	if !SupportsPurePassthrough(ppio, RequestShape{
		Protocol:       PassthroughProtocolOpenAI,
		EndpointFamily: EndpointFamilyResponses,
	}, ppioCapability) {
		t.Fatal("PPIO OpenAI-compatible should support OpenAI responses pure passthrough")
	}

	if SupportsPurePassthrough(ppio, RequestShape{
		Protocol:       PassthroughProtocolAnthropic,
		EndpointFamily: EndpointFamilyMessages,
	}, ppioCapability) {
		t.Fatal("PPIO OpenAI-compatible must not support Anthropic pure passthrough")
	}

	if !SupportsPurePassthrough(anthropicPure, RequestShape{
		Protocol:       PassthroughProtocolAnthropic,
		EndpointFamily: EndpointFamilyMessages,
	}, anthropicCapability) {
		t.Fatal("Anthropic pure channel should support Anthropic messages")
	}

	if SupportsPurePassthrough(anthropicPure, RequestShape{
		Protocol:       PassthroughProtocolOpenAI,
		EndpointFamily: EndpointFamilyResponses,
	}, anthropicCapability) {
		t.Fatal("Anthropic pure channel must not support OpenAI responses")
	}

	if SupportsPurePassthrough(ppio, RequestShape{
		Protocol:       PassthroughProtocolOpenAI,
		EndpointFamily: EndpointFamilyWebSearch,
	}, ppioCapability) {
		t.Fatal("WebSearch must not be classified as pure passthrough")
	}

	if !SupportsAdaptedPassthrough(ppio, RequestShape{
		Protocol:       PassthroughProtocolOpenAI,
		EndpointFamily: EndpointFamilyWebSearch,
	}, ppioCapability) {
		t.Fatal("WebSearch should be classified as adapted passthrough")
	}

	if SupportsAdaptedPassthrough(ppio, RequestShape{
		Protocol:       PassthroughProtocolOpenAI,
		EndpointFamily: EndpointFamilyResponses,
	}, ppioCapability) {
		t.Fatal("Responses must not be classified as adapted passthrough")
	}

	novitaNative := &Channel{Type: ChannelTypeNovitaMultimodal}
	novitaNativeCapability := ChannelCapability{
		PurePassthrough: true,
		Protocol:        PassthroughProtocolNativeNovitaV3,
		EndpointFamilies: []EndpointFamily{
			EndpointFamilyNativeV3,
		},
	}
	if !SupportsPurePassthrough(novitaNative, RequestShape{
		Protocol:       PassthroughProtocolNativeV3,
		EndpointFamily: EndpointFamilyNativeV3,
	}, novitaNativeCapability) {
		t.Fatal("Novita native multimodal should support generic native v3 requests")
	}
}

func endpointFamiliesEqual(a, b []EndpointFamily) bool {
	if len(a) != len(b) {
		return false
	}

	seen := make(map[EndpointFamily]int, len(a))
	for _, family := range a {
		seen[family]++
	}

	for _, family := range b {
		seen[family]--
		if seen[family] < 0 {
			return false
		}
	}

	return true
}
