package ppio

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

func newContractMeta(m mode.Mode) *meta.Meta {
	return meta.NewMeta(&model.Channel{
		ID:      54,
		Type:    model.ChannelTypePPIO,
		BaseURL: baseURL,
		Key:     "upstream-key",
	}, m, "gpt-test", model.ModelConfig{
		Model: "gpt-test",
		Type:  m,
	})
}

func TestConvertRequest_ChatPassthroughPreservesBody(t *testing.T) {
	body := `{"model":"gpt-test","messages":[{"role":"user","content":"hello"}],"temperature":0.2}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("X-Stainless-Arch", "arm64")

	result, err := (&Adaptor{}).ConvertRequest(newContractMeta(mode.ChatCompletions), nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	gotBody, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read converted body: %v", err)
	}

	if string(gotBody) != body {
		t.Fatalf("body was modified:\nwant %s\ngot  %s", body, string(gotBody))
	}

	if got := result.Header.Get("X-Stainless-Arch"); got != "arm64" {
		t.Fatalf("X-Stainless-Arch: want arm64, got %q", got)
	}
}

func TestDoResponse_ChatPassthroughPreservesSuccessBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	body := `{"id":"chatcmpl_1","choices":[{"message":{"content":"hello"}}]}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	_, relayErr := (&Adaptor{}).DoResponse(newContractMeta(mode.ChatCompletions), nil, c, resp)
	if relayErr != nil {
		t.Fatalf("DoResponse returned error: %v", relayErr)
	}

	if got := recorder.Body.String(); got != body {
		t.Fatalf("body was modified:\nwant %s\ngot  %s", body, got)
	}
}

func TestGetRequestURL_ResponsesPreservesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/responses/resp_123/input_items?limit=20&after=item_1",
		nil,
	)

	m := newContractMeta(mode.ResponsesInputItems)
	m.ResponseID = "resp_123"

	got, err := (&Adaptor{}).GetRequestURL(m, nil, c)
	if err != nil {
		t.Fatalf("GetRequestURL returned error: %v", err)
	}

	const want = "https://api.ppinfra.com/openai/v1/responses/resp_123/input_items?limit=20&after=item_1"
	if got.URL != want {
		t.Fatalf("URL: want %s, got %s", want, got.URL)
	}
}
