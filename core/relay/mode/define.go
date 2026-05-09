package mode

import "fmt"

type Mode int

func (m Mode) String() string {
	switch m {
	case Unknown:
		return "Unknown"
	case ChatCompletions:
		return "ChatCompletions"
	case Completions:
		return "Completions"
	case Embeddings:
		return "Embeddings"
	case Moderations:
		return "Moderations"
	case ImagesGenerations:
		return "ImagesGenerations"
	case ImagesEdits:
		return "ImagesEdits"
	case AudioSpeech:
		return "AudioSpeech"
	case AudioTranscription:
		return "AudioTranscription"
	case AudioTranslation:
		return "AudioTranslation"
	case Rerank:
		return "Rerank"
	case ParsePdf:
		return "ParsePdf"
	case Anthropic:
		return "Anthropic"
	case VideoGenerationsJobs:
		return "VideoGenerationsJobs"
	case VideoGenerationsGetJobs:
		return "VideoGenerationsGetJobs"
	case VideoGenerationsContent:
		return "VideoGenerationsContent"
	case Responses:
		return "Responses"
	case ResponsesGet:
		return "ResponsesGet"
	case ResponsesDelete:
		return "ResponsesDelete"
	case ResponsesCancel:
		return "ResponsesCancel"
	case ResponsesInputItems:
		return "ResponsesInputItems"
	case ResponsesCompact:
		return "ResponsesCompact"
	case ResponsesInputTokens:
		return "ResponsesInputTokens"
	case Gemini:
		return "Gemini"
	case WebSearch:
		return "WebSearch"
	case PPIONative:
		return "PPIONative"
	default:
		return fmt.Sprintf("Mode(%d)", m)
	}
}

const (
	// Mode values are persisted in model_configs.type. New modes must be appended
	// with explicit values to avoid reinterpreting existing production rows.
	Unknown                 Mode = 0
	ChatCompletions         Mode = 1
	Completions             Mode = 2
	Embeddings              Mode = 3
	Moderations             Mode = 4
	ImagesGenerations       Mode = 5
	ImagesEdits             Mode = 6
	AudioSpeech             Mode = 7
	AudioTranscription      Mode = 8
	AudioTranslation        Mode = 9
	Rerank                  Mode = 10
	ParsePdf                Mode = 11
	Anthropic               Mode = 12
	VideoGenerationsJobs    Mode = 13
	VideoGenerationsGetJobs Mode = 14
	VideoGenerationsContent Mode = 15
	Responses               Mode = 16
	ResponsesGet            Mode = 17
	ResponsesDelete         Mode = 18
	ResponsesCancel         Mode = 19
	ResponsesInputItems     Mode = 20
	Gemini                  Mode = 21
	WebSearch               Mode = 22
	// PPIONative is for PPIO multimodal models (image/video/audio) that use
	// model-ID-embedded URL paths (/v3/{model-id} or /v3/async/{model-id}).
	// The request/response bodies are forwarded verbatim in PPIO's native format.
	PPIONative Mode = 23

	ResponsesCompact     Mode = 24
	ResponsesInputTokens Mode = 25
)
