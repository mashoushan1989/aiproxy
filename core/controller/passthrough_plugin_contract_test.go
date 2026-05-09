package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/novita"
	"github.com/labring/aiproxy/core/relay/adaptor/novitaml"
	"github.com/labring/aiproxy/core/relay/adaptor/ppio"
	"github.com/labring/aiproxy/core/relay/adaptor/ppioml"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

func newPassthroughPluginContractMeta(
	chType model.ChannelType,
	baseURL string,
	m mode.Mode,
) *meta.Meta {
	return meta.NewMeta(&model.Channel{
		ID:      int(chType),
		Type:    chType,
		BaseURL: baseURL,
		Key:     "upstream-key",
	}, m, "gpt-test", model.ModelConfig{
		Model: "gpt-test",
		Type:  m,
	})
}

type passthroughContractCase struct {
	name    string
	adaptor adaptor.Adaptor
	meta    *meta.Meta
	method  string
	path    string
	body    string
	want20m bool
}

func passthroughOpenAICompatibleContractCases() []passthroughContractCase {
	modes := []struct {
		name   string
		mode   mode.Mode
		method string
		path   string
		body   string
	}{
		{
			name:   "chat",
			mode:   mode.ChatCompletions,
			method: http.MethodPost,
			path:   "/v1/chat/completions",
			body:   `{"model":"gpt-test","messages":[{"role":"user","content":"hello"}],"stream":true}`,
		},
		{
			name:   "completions",
			mode:   mode.Completions,
			method: http.MethodPost,
			path:   "/v1/completions",
			body:   `{"model":"gpt-test","prompt":"hello","stream":true}`,
		},
		{
			name:   "responses",
			mode:   mode.Responses,
			method: http.MethodPost,
			path:   "/v1/responses",
			body:   `{"model":"gpt-test","input":"hello","stream":true}`,
		},
		{
			name:   "embeddings",
			mode:   mode.Embeddings,
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body:   `{"model":"gpt-test","input":"hello"}`,
		},
		{
			name:   "images_generations",
			mode:   mode.ImagesGenerations,
			method: http.MethodPost,
			path:   "/v1/images/generations",
			body:   `{"model":"gpt-test","prompt":"a mountain"}`,
		},
		{
			name:   "audio_speech",
			mode:   mode.AudioSpeech,
			method: http.MethodPost,
			path:   "/v1/audio/speech",
			body:   `{"model":"gpt-test","input":"hello","voice":"alloy"}`,
		},
		{
			name:   "rerank",
			mode:   mode.Rerank,
			method: http.MethodPost,
			path:   "/v1/rerank",
			body:   `{"model":"gpt-test","query":"hello","documents":["hello world"]}`,
		},
		{
			name:   "moderations",
			mode:   mode.Moderations,
			method: http.MethodPost,
			path:   "/v1/moderations",
			body:   `{"model":"gpt-test","input":"hello"}`,
		},
		{
			name:   "video_generations_jobs",
			mode:   mode.VideoGenerationsJobs,
			method: http.MethodPost,
			path:   "/v1/video/generations/jobs",
			body:   `{"model":"gpt-test","prompt":"hello"}`,
		},
	}

	providers := []struct {
		name    string
		adaptor adaptor.Adaptor
		chType  model.ChannelType
		baseURL string
	}{
		{
			name:    "ppio",
			adaptor: &ppio.Adaptor{},
			chType:  model.ChannelTypePPIO,
			baseURL: "https://api.ppinfra.com/v3/openai",
		},
		{
			name:    "novita",
			adaptor: &novita.Adaptor{},
			chType:  model.ChannelTypeNovita,
			baseURL: "https://api.novita.ai/v3/openai",
		},
	}

	cases := make([]passthroughContractCase, 0, len(providers)*len(modes))
	for _, provider := range providers {
		for _, modeCase := range modes {
			cases = append(cases, passthroughContractCase{
				name:    provider.name + "/" + modeCase.name,
				adaptor: provider.adaptor,
				meta: newPassthroughPluginContractMeta(
					provider.chType,
					provider.baseURL,
					modeCase.mode,
				),
				method: modeCase.method,
				path:   modeCase.path,
				body:   modeCase.body,
				want20m: modeCase.mode == mode.ChatCompletions ||
					modeCase.mode == mode.Completions ||
					modeCase.mode == mode.Responses,
			})
		}
	}

	return cases
}

func passthroughNativeMultimodalContractCases() []passthroughContractCase {
	return []passthroughContractCase{
		{
			name:    "ppio_multimodal/sync",
			adaptor: &ppioml.Adaptor{},
			meta: newPassthroughPluginContractMeta(
				model.ChannelTypePPIOMultimodal,
				"https://api.ppinfra.com",
				mode.PPIONative,
			),
			method: http.MethodPost,
			path:   "/v3/seedream-5.0-lite",
			body:   `{"prompt":"a mountain","size":"1024x1024"}`,
		},
		{
			name:    "ppio_multimodal/async",
			adaptor: &ppioml.Adaptor{},
			meta: newPassthroughPluginContractMeta(
				model.ChannelTypePPIOMultimodal,
				"https://api.ppinfra.com",
				mode.PPIONative,
			),
			method: http.MethodPost,
			path:   "/v3/async/veo-3.1",
			body:   `{"prompt":"a mountain video","duration":5}`,
		},
		{
			name:    "novita_multimodal/sync",
			adaptor: &novitaml.Adaptor{},
			meta: newPassthroughPluginContractMeta(
				model.ChannelTypeNovitaMultimodal,
				"https://api.novita.ai",
				mode.PPIONative,
			),
			method: http.MethodPost,
			path:   "/v3/seedream-5.0-lite",
			body:   `{"prompt":"a mountain","size":"1024x1024"}`,
		},
		{
			name:    "novita_multimodal/async",
			adaptor: &novitaml.Adaptor{},
			meta: newPassthroughPluginContractMeta(
				model.ChannelTypeNovitaMultimodal,
				"https://api.novita.ai",
				mode.PPIONative,
			),
			method: http.MethodPost,
			path:   "/v3/async/veo-3.1",
			body:   `{"prompt":"a mountain video","duration":5}`,
		},
	}
}

func passthroughPluginContractCases() []passthroughContractCase {
	cases := passthroughOpenAICompatibleContractCases()
	cases = append(cases, passthroughNativeMultimodalContractCases()...)
	return cases
}

func TestDefaultPluginChainDoesNotModifyPassthroughProtocolRequests(t *testing.T) {
	tests := passthroughPluginContractCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				tt.method,
				tt.path,
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Stainless-Arch", "arm64")

			wrapped := wrapPlugin(req.Context(), &model.ModelCaches{}, tt.adaptor)
			result, err := wrapped.ConvertRequest(tt.meta, nil, req)
			if err != nil {
				t.Fatalf("ConvertRequest returned error: %v", err)
			}

			gotBody, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatalf("read converted body: %v", err)
			}

			if string(gotBody) != tt.body {
				t.Fatalf("body was modified:\nwant %s\ngot  %s", tt.body, string(gotBody))
			}

			if got := result.Header.Get("X-Stainless-Arch"); got != "arm64" {
				t.Fatalf("X-Stainless-Arch: want arm64, got %q", got)
			}

			if tt.want20m && tt.meta.RequestTimeout != 20*60*1_000_000_000 {
				t.Fatalf("passthrough timeout: want 20m0s, got %s", tt.meta.RequestTimeout)
			}
		})
	}
}

func TestDefaultPluginChainDoesNotModifyPassthroughProtocolResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := passthroughPluginContractCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(tt.method, tt.path, nil)

			body := `{"id":"upstream_1","result":{"content":"hello"}}`
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(strings.NewReader(body)),
			}

			wrapped := wrapPlugin(c.Request.Context(), &model.ModelCaches{}, tt.adaptor)
			_, relayErr := wrapped.DoResponse(tt.meta, nil, c, resp)
			if relayErr != nil {
				t.Fatalf("DoResponse returned error: %v", relayErr)
			}

			if got := recorder.Body.String(); got != body {
				t.Fatalf("body was modified:\nwant %s\ngot  %s", body, got)
			}
		})
	}
}
