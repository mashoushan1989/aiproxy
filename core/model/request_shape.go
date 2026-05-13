package model

import (
	"strings"

	"github.com/labring/aiproxy/core/relay/mode"
)

func InferRequestShape(method, path string, relayMode mode.Mode) RequestShape {
	return RequestShape{
		Protocol:       inferRequestProtocol(path, relayMode),
		EndpointFamily: inferEndpointFamily(path, relayMode),
		Method:         method,
		OriginalPath:   path,
	}
}

func inferRequestProtocol(path string, relayMode mode.Mode) PassthroughProtocol {
	switch relayMode {
	case mode.Anthropic:
		return PassthroughProtocolAnthropic
	case mode.Gemini:
		return PassthroughProtocolGemini
	case mode.PPIONative:
		return PassthroughProtocolNativeV3
	case mode.ChatCompletions,
		mode.Completions,
		mode.Responses,
		mode.ResponsesGet,
		mode.ResponsesDelete,
		mode.ResponsesCancel,
		mode.ResponsesInputItems,
		mode.ResponsesCompact,
		mode.ResponsesInputTokens,
		mode.Embeddings,
		mode.ImagesGenerations,
		mode.ImagesEdits,
		mode.AudioSpeech,
		mode.AudioTranscription,
		mode.AudioTranslation,
		mode.Rerank,
		mode.Moderations,
		mode.VideoGenerationsJobs,
		mode.VideoGenerationsGetJobs,
		mode.VideoGenerationsContent,
		mode.WebSearch:
		return PassthroughProtocolOpenAI
	default:
		if strings.HasPrefix(path, "/v1/") || path == "/v1" {
			return PassthroughProtocolOpenAI
		}
	}

	return ""
}

func inferEndpointFamily(path string, relayMode mode.Mode) EndpointFamily {
	switch relayMode {
	case mode.ChatCompletions:
		return EndpointFamilyChat
	case mode.Completions:
		return EndpointFamilyCompletions
	case mode.Responses,
		mode.ResponsesGet,
		mode.ResponsesDelete,
		mode.ResponsesCancel,
		mode.ResponsesInputItems,
		mode.ResponsesCompact,
		mode.ResponsesInputTokens:
		return EndpointFamilyResponses
	case mode.Embeddings:
		return EndpointFamilyEmbeddings
	case mode.ImagesGenerations, mode.ImagesEdits:
		return EndpointFamilyImages
	case mode.AudioSpeech, mode.AudioTranscription, mode.AudioTranslation:
		return EndpointFamilyAudio
	case mode.Rerank:
		return EndpointFamilyRerank
	case mode.Moderations:
		return EndpointFamilyModerations
	case mode.VideoGenerationsJobs, mode.VideoGenerationsGetJobs, mode.VideoGenerationsContent:
		return EndpointFamilyVideoJobs
	case mode.Anthropic:
		return EndpointFamilyMessages
	case mode.Gemini:
		return EndpointFamilyGeminiGenerateContent
	case mode.PPIONative:
		return EndpointFamilyNativeV3
	case mode.WebSearch:
		return EndpointFamilyWebSearch
	default:
		if strings.Contains(path, "/responses") {
			return EndpointFamilyResponses
		}
	}

	return ""
}
