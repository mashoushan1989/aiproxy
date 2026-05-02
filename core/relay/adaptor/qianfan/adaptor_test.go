//nolint:testpackage
package qianfan

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdaptorSetupRequestHeaderWithAppID(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		&coremodel.Channel{
			Key: "test-key",
			Configs: coremodel.ChannelConfigs{
				"appid": " app-test ",
			},
		},
		mode.ChatCompletions,
		"ernie-4.5-turbo-128k",
		coremodel.ModelConfig{},
	)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"https://qianfan.baidubce.com/v2/chat/completions",
		nil,
	)

	err := adaptor.SetupRequestHeader(m, nil, nil, req)

	require.NoError(t, err)
	assert.Equal(t, "Bearer test-key", req.Header.Get("Authorization"))
	assert.Equal(t, "app-test", req.Header.Get("Appid"))
}

func TestAdaptorSetupRequestHeaderWithoutAppID(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		&coremodel.Channel{
			Key: "test-key",
		},
		mode.ChatCompletions,
		"ernie-4.5-turbo-128k",
		coremodel.ModelConfig{},
	)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"https://qianfan.baidubce.com/v2/chat/completions",
		nil,
	)

	err := adaptor.SetupRequestHeader(m, nil, nil, req)

	require.NoError(t, err)
	assert.Equal(t, "Bearer test-key", req.Header.Get("Authorization"))
	assert.Empty(t, req.Header.Get("Appid"))
}

func TestAdaptorMetadataConfigSchema(t *testing.T) {
	adaptor := &Adaptor{}
	metaInfo := adaptor.Metadata()

	properties, ok := metaInfo.ConfigSchema["properties"].(map[string]any)
	require.True(t, ok)

	field, ok := properties["appid"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", field["type"])
}

func TestAdaptorSupportModeGemini(t *testing.T) {
	adaptor := &Adaptor{}

	assert.True(t, adaptor.SupportMode(mode.Gemini))
}

func TestAdaptorConvertRequestGemini(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.Gemini,
		"ernie-4.5-turbo-128k",
		coremodel.ModelConfig{},
	)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1beta/models/ernie-4.5-turbo-128k:streamGenerateContent",
		strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
	)
	require.NoError(t, err)

	result, err := adaptor.ConvertRequest(m, nil, req)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)

	var openAIReq relaymodel.GeneralOpenAIRequest
	require.NoError(t, json.Unmarshal(body, &openAIReq))

	assert.Equal(t, "ernie-4.5-turbo-128k", openAIReq.Model)
	assert.True(t, openAIReq.Stream)
}
