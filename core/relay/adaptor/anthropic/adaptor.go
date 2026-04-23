package anthropic

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/passthrough"
	"github.com/labring/aiproxy/core/relay/adaptor/registry"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

// Adaptor serves channel type 14. The pure_passthrough channel config selects
// between byte-level relay (PPIO/Novita Anthropic-compat) and protocol
// conversion (direct Anthropic API, plus OpenAI/Gemini cross-protocol).
// Dispatch happens per-method inside case mode.Anthropic only; ChatCompletions
// and Gemini always go through the protocol converter.
type Adaptor struct {
	passthroughDelegate passthrough.AnthropicAdaptor
}

func init() {
	registry.Register(model.ChannelTypeAnthropic, &Adaptor{})
}

const baseURL = "https://api.anthropic.com/v1"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) NativeMode(m mode.Mode) bool {
	return m == mode.Anthropic
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Anthropic ||
		m == mode.Gemini
}

func (a *Adaptor) GetRequestURL(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
) (adaptor.RequestURL, error) {
	// Passthrough: preserve client path and query verbatim.
	if a.shouldPassthrough(meta) {
		return a.passthroughDelegate.GetRequestURL(meta, store, c)
	}

	u := meta.Channel.BaseURL

	pu, err := url.Parse(u)
	if err != nil {
		return adaptor.RequestURL{}, err
	}

	result := pu.JoinPath("/messages")
	result.RawQuery = c.Request.URL.RawQuery

	return adaptor.RequestURL{
		Method: http.MethodPost,
		URL:    result.String(),
	}, nil
}

const (
	AnthropicVersion = "2023-06-01"
	//nolint:gosec
	AnthropicTokenHeader = "X-Api-Key"
	AnthropicBeta        = "Anthropic-Beta"
)

func ModelDefaultMaxTokens(model string) int {
	switch {
	case strings.Contains(model, "opus-4-5"):
		return 64000
	case strings.Contains(model, "sonnet-4-5"):
		return 64000
	case strings.Contains(model, "4-1"):
		return 204800
	case strings.Contains(model, "sonnet-4-"):
		return 64000
	case strings.Contains(model, "opus-4-"):
		return 32768
	case strings.Contains(model, "3-7"):
		return 131072
	default:
		return 4096
	}
}

// GetMaxTokens returns the effective default max_tokens for a model.
// Prefers ModelConfig.MaxOutputTokens over the hardcoded ModelDefaultMaxTokens.
func GetMaxTokens(m *meta.Meta) int {
	if maxOut, ok := m.ModelConfig.MaxOutputTokens(); ok && maxOut > 0 {
		return maxOut
	}
	return ModelDefaultMaxTokens(m.ActualModel)
}

func FixBetasStringWithModel(model, betas string, deleteFunc ...func(e string) bool) string {
	return strings.Join(FixBetasWithModel(model, strings.Split(betas, ","), deleteFunc...), ",")
}

func modelBetaSupported(model, beta string) bool {
	switch beta {
	case "computer-use-2025-01-24":
		return strings.Contains(model, "3-7-sonnet")
	case "token-efficient-tools-2025-02-19":
		return strings.Contains(model, "3-7-sonnet") ||
			strings.Contains(model, "4-") ||
			strings.Contains(model, "-4")
	case "interleaved-thinking-2025-05-14":
		return strings.Contains(model, "4-") ||
			strings.Contains(model, "-4")
	case "output-128k-2025-02-19":
		return strings.Contains(model, "3-7-sonnet")
	case "dev-full-thinking-2025-05-14":
		return strings.Contains(model, "4-") ||
			strings.Contains(model, "-4")
	case "context-1m-2025-08-07":
		return (strings.Contains(model, "4-") ||
			strings.Contains(model, "-4")) &&
			strings.Contains(model, "sonnet")
	case "context-management-2025-06-27":
		return strings.Contains(model, "4-5") ||
			strings.Contains(model, "-4-5")
	default:
		return true
	}
}

func FixBetasWithModel(model string, betas []string, deleteFunc ...func(e string) bool) []string {
	return slices.DeleteFunc(betas, func(beta string) bool {
		for _, v := range deleteFunc {
			if v != nil && v(beta) {
				return true
			}
		}

		return !modelBetaSupported(model, beta)
	})
}

// anthropicConfigCacheKey scopes the parsed Config cache on meta.values.
const anthropicConfigCacheKey = "anthropic.cfg"

// loadAnthropicConfig returns the channel-level Anthropic Config, memoized on
// meta for the lifetime of the request. Avoids repeat sonic.Marshal+Unmarshal
// of ChannelConfigs (~0.2ms each) when multiple Adaptor methods need the same
// flags on a single request.
func loadAnthropicConfig(m *meta.Meta) Config {
	if cached, ok := m.Get(anthropicConfigCacheKey); ok {
		if cfg, ok := cached.(Config); ok {
			return cfg
		}
	}

	cfg := Config{}
	_ = m.ChannelConfigs.LoadConfig(&cfg)
	m.Set(anthropicConfigCacheKey, cfg)

	return cfg
}

// shouldPassthrough is the single dispatch guard used across Adaptor methods:
// delegate to the byte-level passthrough path only when mode.Anthropic AND
// pure_passthrough=true. ChatCompletions/Gemini (cross-protocol conversion)
// and mode.Anthropic with pure_passthrough=false (direct Anthropic API) fall
// through to the legacy protocol-handling path.
func (a *Adaptor) shouldPassthrough(m *meta.Meta) bool {
	return m.Mode == mode.Anthropic && loadAnthropicConfig(m).PurePassthrough
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) error {
	// Passthrough: client already has Anthropic-Version/Beta; only install
	// the channel key. Cross-protocol paths below rebuild the body and need
	// Anthropic-Version fallback injection.
	if a.shouldPassthrough(meta) {
		return a.passthroughDelegate.SetupRequestHeader(meta, store, c, req)
	}

	req.Header.Set(AnthropicTokenHeader, meta.Channel.Key)

	anthropicVersion := c.Request.Header.Get("Anthropic-Version")
	if anthropicVersion == "" {
		anthropicVersion = AnthropicVersion
	}

	req.Header.Set("Anthropic-Version", anthropicVersion)

	// Pass through Anthropic-Beta as-is. Pre-filtering by model name causes
	// Header/Body desync (e.g. stripping a beta header while the body still
	// contains the gated field → 400 "Extra inputs are not permitted").
	// Constrained backends (AWS Bedrock, VertexAI) apply their own filtering.
	rawBetas := c.Request.Header.Get(AnthropicBeta)

	if rawBetas != "" {
		req.Header.Set(AnthropicBeta, rawBetas)
	}

	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	switch meta.Mode {
	case mode.ChatCompletions:
		data, err := OpenAIConvertRequest(meta, req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}

		data2, err := sonic.Marshal(data)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}

		return adaptor.ConvertResult{
			Header: http.Header{
				"Content-Type":   {"application/json"},
				"Content-Length": {strconv.Itoa(len(data2))},
			},
			Body: bytes.NewReader(data2),
		}, nil
	case mode.Anthropic:
		// Passthrough delegates to passthrough.AnthropicAdaptor which inherits
		// ForwardClientHeaders + ReplaceModelInBody from passthrough.Adaptor.
		if a.shouldPassthrough(meta) {
			return a.passthroughDelegate.ConvertRequest(meta, store, req)
		}

		return ConvertRequest(meta, req)
	case mode.Gemini:
		return ConvertGeminiRequest(meta, req)
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (*http.Response, error) {
	return utils.DoRequestWithMeta(req, meta)
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	switch meta.Mode {
	case mode.ChatCompletions:
		if utils.IsStreamResponse(resp) {
			return OpenAIStreamHandler(meta, c, resp)
		}
		return OpenAIHandler(meta, c, resp)
	case mode.Anthropic:
		if a.shouldPassthrough(meta) {
			return a.passthroughDelegate.DoResponse(meta, store, c, resp)
		}

		if utils.IsStreamResponse(resp) {
			return StreamHandler(meta, c, resp)
		}

		return Handler(meta, c, resp)
	case mode.Gemini:
		if utils.IsStreamResponse(resp) {
			return GeminiStreamHandler(meta, c, resp)
		}
		return GeminiHandler(meta, c, resp)
	default:
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("unsupported mode: %s", meta.Mode),
			"unsupported_mode",
			http.StatusBadRequest,
		)
	}
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "Support native Endpoint: /v1/messages",
		Models: ModelList,
		ConfigSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"disable_context_management": map[string]any{
					"type":        "boolean",
					"title":       "Disable Context Management",
					"description": "Remove the entire context_management field before sending the request upstream.",
				},
				"supported_context_management_edits_type": map[string]any{
					"type":        "array",
					"title":       "Supported Context Management Edit Types",
					"description": "Only keep the listed context management edit types. Leave empty to keep all edit types.",
					"items": map[string]any{
						"type": "string",
					},
				},
				"remove_tools_examples": map[string]any{
					"type":        "boolean",
					"title":       "Remove Tool Examples",
					"description": "Strip tool input_examples from the request body before relay.",
				},
				"remove_tools_custom_defer_loading": map[string]any{
					"type":        "boolean",
					"title":       "Remove Tool Defer Loading",
					"description": "Strip tool defer_loading from the request body before relay.",
				},
				"skip_image_conversion": map[string]any{
					"type":        "boolean",
					"title":       "Skip Image URL to Base64 Conversion",
					"description": "Skip converting image URLs to base64. Enable this for Anthropic-compatible upstreams that natively support URL image sources to reduce latency.",
				},
				"strip_cache_ttl": map[string]any{
					"type":        "boolean",
					"title":       "Strip Cache Control TTL",
					"description": "Remove cache_control.ttl from the request. Enable this for upstreams that do not support the prompt-caching-scope beta.",
				},
				"pure_passthrough": map[string]any{
					"type":        "boolean",
					"title":       "Pure Passthrough (Anthropic Protocol)",
					"description": "Forward Anthropic-protocol requests verbatim without any body transformation. Only authentication headers are replaced. Usage is captured via SSE scanning.",
				},
			},
		},
	}
}
