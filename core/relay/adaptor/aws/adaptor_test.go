//nolint:testpackage
package aws

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/stretchr/testify/require"
)

func TestConvertRequestFallsBackToOriginModelForARN(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		&model.Channel{},
		mode.ChatCompletions,
		"claude-opus-4-7",
		model.ModelConfig{},
	)
	m.ActualModel = "arn:aws:bedrock:us-east-1:123456789012:provisioned-model/test"

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		bytes.NewBufferString(
			`{"model":"claude-opus-4-7","messages":[{"role":"user","content":"hello"}]}`,
		),
	)

	_, err := adaptor.ConvertRequest(m, nil, req)
	require.NoError(t, err)
}
