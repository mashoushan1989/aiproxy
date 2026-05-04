//nolint:testpackage
package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type responseTestStore struct {
	saved []adaptor.StoreCache
}

func (s *responseTestStore) GetStore(string, int, string) (adaptor.StoreCache, error) {
	return adaptor.StoreCache{}, nil
}

func (s *responseTestStore) SaveStore(cache adaptor.StoreCache) error {
	s.saved = append(s.saved, cache)
	return nil
}

func TestResponseHandlerStoreUsesOriginModel(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/responses",
		nil,
	)
	store := &responseTestStore{}
	meta := &meta.Meta{
		OriginModel: "gpt-5",
		ActualModel: "mapped-gpt-5",
		Group:       model.GroupCache{ID: "group-1"},
		Token:       model.TokenCache{ID: 7},
		Channel:     meta.ChannelMeta{ID: 9},
	}

	body := `{
		"id":"resp_store_origin",
		"object":"response",
		"created_at":1,
		"status":"completed",
		"model":"mapped-gpt-5",
		"output":[],
		"parallel_tool_calls":true,
		"store":true
	}`
	resp := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}

	_, err := ResponseHandler(meta, store, c, resp)
	require.Nil(t, err)
	require.Len(t, store.saved, 1)
	assert.Equal(t, "gpt-5", store.saved[0].Model)
}

func TestResponseStreamHandlerStoreUsesOriginModel(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/responses",
		nil,
	)
	store := &responseTestStore{}
	meta := &meta.Meta{
		OriginModel: "gpt-5",
		ActualModel: "mapped-gpt-5",
		Group:       model.GroupCache{ID: "group-1"},
		Token:       model.TokenCache{ID: 7},
		Channel:     meta.ChannelMeta{ID: 9},
	}

	body := "event: response.created\n" +
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_stream_store_origin\",\"object\":\"response\",\"created_at\":1,\"status\":\"in_progress\",\"model\":\"mapped-gpt-5\",\"output\":[],\"parallel_tool_calls\":true,\"store\":true}}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_stream_store_origin\",\"object\":\"response\",\"created_at\":2,\"status\":\"completed\",\"model\":\"mapped-gpt-5\",\"output\":[],\"parallel_tool_calls\":true,\"store\":true}}\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}

	_, err := ResponseStreamHandler(meta, store, c, resp)
	require.Nil(t, err)
	require.Len(t, store.saved, 1)
	assert.Equal(t, "gpt-5", store.saved[0].Model)
}

func TestResponseHandlerWebSearchCountFromToolUsage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/responses",
		nil,
	)
	store := &responseTestStore{}
	meta := &meta.Meta{
		OriginModel: "gpt-5.4",
		ActualModel: "gpt-5.4",
		Group:       model.GroupCache{ID: "group-1"},
		Token:       model.TokenCache{ID: 7},
		Channel:     meta.ChannelMeta{ID: 9},
	}

	body := `{
		"id":"resp_tool_usage_123",
		"object":"response",
		"created_at":1777053463,
		"status":"completed",
		"model":"gpt-5.4",
		"output":[
			{"type":"reasoning","summary":[]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}
		],
		"tool_usage":{"web_search":{"num_requests":1}},
		"usage":{
			"input_tokens":15065,
			"input_tokens_details":{"cached_tokens":10880},
			"output_tokens":256,
			"output_tokens_details":{"reasoning_tokens":81},
			"total_tokens":15321
		}
	}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}

	result, err := ResponseHandler(meta, store, c, resp)
	require.Nil(t, err)
	assert.Equal(t, model.ZeroNullInt64(15321), result.Usage.TotalTokens)
	assert.Equal(t, model.ZeroNullInt64(1), result.Usage.WebSearchCount)
}

func TestResponseNeedsAsyncUsage(t *testing.T) {
	t.Run("queued without usage needs async usage", func(t *testing.T) {
		response := &relaymodel.Response{
			ID:     "resp_queued",
			Status: relaymodel.ResponseStatusQueued,
		}

		assert.True(t, responseNeedsAsyncUsage(response))
	})

	t.Run("in progress without usage needs async usage", func(t *testing.T) {
		response := &relaymodel.Response{
			ID:     "resp_progress",
			Status: relaymodel.ResponseStatusInProgress,
		}

		assert.True(t, responseNeedsAsyncUsage(response))
	})

	t.Run("existing usage does not need async usage", func(t *testing.T) {
		response := &relaymodel.Response{
			ID:     "resp_done",
			Status: relaymodel.ResponseStatusCompleted,
			Usage:  &relaymodel.ResponseUsage{TotalTokens: 10},
		}

		assert.False(t, responseNeedsAsyncUsage(response))
	})

	t.Run("missing upstream id does not need async usage", func(t *testing.T) {
		response := &relaymodel.Response{
			Status: relaymodel.ResponseStatusQueued,
		}

		assert.False(t, responseNeedsAsyncUsage(response))
	})
}

func TestResponseStreamHandlerWebSearchCountFromToolUsage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/responses",
		nil,
	)
	store := &responseTestStore{}
	meta := &meta.Meta{
		OriginModel: "gpt-5.4",
		ActualModel: "gpt-5.4",
		Group:       model.GroupCache{ID: "group-1"},
		Token:       model.TokenCache{ID: 7},
		Channel:     meta.ChannelMeta{ID: 9},
	}

	body := "event: response.created\n" +
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_ws_stream_tool_usage\",\"object\":\"response\",\"created_at\":1777053463,\"status\":\"in_progress\",\"model\":\"gpt-5.4\",\"output\":[],\"tool_usage\":{\"image_gen\":{\"input_tokens\":0,\"input_tokens_details\":{\"image_tokens\":0,\"text_tokens\":0},\"output_tokens\":0,\"output_tokens_details\":{\"image_tokens\":0,\"text_tokens\":0},\"total_tokens\":0},\"web_search\":{\"num_requests\":1}},\"parallel_tool_calls\":true,\"store\":false}}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_ws_stream_tool_usage\",\"object\":\"response\",\"created_at\":1777053474,\"status\":\"completed\",\"model\":\"gpt-5.4\",\"output\":[{\"type\":\"reasoning\",\"summary\":[]},{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"done\"}]}],\"tool_usage\":{\"image_gen\":{\"input_tokens\":0,\"input_tokens_details\":{\"image_tokens\":0,\"text_tokens\":0},\"output_tokens\":0,\"output_tokens_details\":{\"image_tokens\":0,\"text_tokens\":0},\"total_tokens\":0},\"web_search\":{\"num_requests\":1}},\"parallel_tool_calls\":true,\"store\":false,\"usage\":{\"input_tokens\":15065,\"input_tokens_details\":{\"cached_tokens\":10880},\"output_tokens\":256,\"output_tokens_details\":{\"reasoning_tokens\":81},\"total_tokens\":15321}}}\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}

	result, err := ResponseStreamHandler(meta, store, c, resp)
	require.Nil(t, err)
	assert.Equal(t, model.ZeroNullInt64(15321), result.Usage.TotalTokens)
	assert.Equal(t, model.ZeroNullInt64(1), result.Usage.WebSearchCount)
}
