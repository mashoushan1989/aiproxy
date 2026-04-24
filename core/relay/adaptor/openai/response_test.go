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
