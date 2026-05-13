package model

import (
	"net/http"
	"testing"

	"github.com/labring/aiproxy/core/relay/mode"
)

func TestInferRequestShape(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		relayMode  mode.Mode
		wantProto  PassthroughProtocol
		wantFamily EndpointFamily
	}{
		{
			name:       "openai chat",
			method:     http.MethodPost,
			path:       "/v1/chat/completions",
			relayMode:  mode.ChatCompletions,
			wantProto:  PassthroughProtocolOpenAI,
			wantFamily: EndpointFamilyChat,
		},
		{
			name:       "openai responses",
			method:     http.MethodPost,
			path:       "/v1/responses",
			relayMode:  mode.Responses,
			wantProto:  PassthroughProtocolOpenAI,
			wantFamily: EndpointFamilyResponses,
		},
		{
			name:       "openai responses input items",
			method:     http.MethodGet,
			path:       "/v1/responses/resp_123/input_items",
			relayMode:  mode.ResponsesInputItems,
			wantProto:  PassthroughProtocolOpenAI,
			wantFamily: EndpointFamilyResponses,
		},
		{
			name:       "anthropic messages",
			method:     http.MethodPost,
			path:       "/v1/messages",
			relayMode:  mode.Anthropic,
			wantProto:  PassthroughProtocolAnthropic,
			wantFamily: EndpointFamilyMessages,
		},
		{
			name:       "gemini v1beta generate content",
			method:     http.MethodPost,
			path:       "/v1beta/models/gemini-2.5-pro:generateContent",
			relayMode:  mode.Gemini,
			wantProto:  PassthroughProtocolGemini,
			wantFamily: EndpointFamilyGeminiGenerateContent,
		},
		{
			name:       "native v3",
			method:     http.MethodPost,
			path:       "/v3/seedream-5.0-lite",
			relayMode:  mode.PPIONative,
			wantProto:  PassthroughProtocolNativeV3,
			wantFamily: EndpointFamilyNativeV3,
		},
		{
			name:       "web search adapted",
			method:     http.MethodPost,
			path:       "/v1/web-search",
			relayMode:  mode.WebSearch,
			wantProto:  PassthroughProtocolOpenAI,
			wantFamily: EndpointFamilyWebSearch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferRequestShape(tt.method, tt.path, tt.relayMode)

			if got.Protocol != tt.wantProto {
				t.Fatalf("Protocol: want %q, got %q", tt.wantProto, got.Protocol)
			}

			if got.EndpointFamily != tt.wantFamily {
				t.Fatalf("EndpointFamily: want %q, got %q", tt.wantFamily, got.EndpointFamily)
			}

			if got.Method != tt.method {
				t.Fatalf("Method: want %q, got %q", tt.method, got.Method)
			}

			if got.OriginalPath != tt.path {
				t.Fatalf("OriginalPath: want %q, got %q", tt.path, got.OriginalPath)
			}
		})
	}
}
