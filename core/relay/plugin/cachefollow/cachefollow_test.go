//nolint:testpackage
package cachefollow

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type recordingStore struct {
	stores map[string]adaptor.StoreCache
	saved  []adaptor.StoreCache
}

func (s *recordingStore) GetStore(_ string, _ int, id string) (adaptor.StoreCache, error) {
	if s.stores == nil {
		return adaptor.StoreCache{}, gorm.ErrRecordNotFound
	}

	store, ok := s.stores[id]
	if !ok {
		return adaptor.StoreCache{}, gorm.ErrRecordNotFound
	}

	return store, nil
}

func (s *recordingStore) SaveStore(cache adaptor.StoreCache) error {
	if s.stores == nil {
		s.stores = make(map[string]adaptor.StoreCache)
	}

	s.stores[cache.ID] = cache
	s.saved = append(s.saved, cache)

	return nil
}

type doResponseFunc struct {
	fn func(*meta.Meta, adaptor.Store, *gin.Context, *http.Response) (adaptor.DoResponseResult, adaptor.Error)
}

func (f doResponseFunc) DoResponse(
	m *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	return f.fn(m, store, c, resp)
}

func newTestContext(t *testing.T) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/chat/completions",
		nil,
	)

	return c
}

func TestDoResponseRecordsPromptAndUserMappingsByDefault(t *testing.T) {
	t.Parallel()

	c := newTestContext(t)
	store := &recordingStore{}
	requestMeta := &meta.Meta{
		Mode:           mode.ChatCompletions,
		OriginModel:    "gpt-5",
		PromptCacheKey: "cache-key",
		User:           "user-1",
		ModelConfig: model.ModelConfig{
			Model: "gpt-5",
			Plugin: map[string]map[string]any{
				PluginName: {"enable": true},
			},
		},
		Group:   model.GroupCache{ID: "group-1"},
		Token:   model.TokenCache{ID: 7},
		Channel: meta.ChannelMeta{ID: 9},
	}

	start := time.Now()
	_, relayErr := (&Plugin{}).DoResponse(
		requestMeta,
		store,
		c,
		&http.Response{StatusCode: http.StatusOK},
		doResponseFunc{
			fn: func(_ *meta.Meta, _ adaptor.Store, c *gin.Context, _ *http.Response) (adaptor.DoResponseResult, adaptor.Error) {
				c.Status(http.StatusOK)
				_, _ = c.Writer.Write(
					[]byte(`data: {"response":{"prompt_cache_retention":"1h"}}` + "\n\n"),
				)

				return adaptor.DoResponseResult{
					Usage: model.Usage{CachedTokens: 4},
				}, nil
			},
		},
	)
	end := time.Now()

	require.Nil(t, relayErr)
	require.Len(t, store.saved, 4)
	assert.Equal(
		t,
		model.PromptCacheStoreID("gpt-5", "cache-key", model.CacheKeyTypeStable),
		store.saved[0].ID,
	)
	assert.Equal(
		t,
		model.PromptCacheStoreID("gpt-5", "cache-key", model.CacheKeyTypeRecent),
		store.saved[1].ID,
	)
	assert.Equal(
		t,
		model.CacheFollowUserStoreID("gpt-5", "user-1", model.CacheKeyTypeStable),
		store.saved[2].ID,
	)
	assert.Equal(
		t,
		model.CacheFollowUserStoreID("gpt-5", "user-1", model.CacheKeyTypeRecent),
		store.saved[3].ID,
	)
	assert.True(t, store.saved[0].ExpiresAt.After(start.Add(time.Hour-time.Second)))
	assert.True(t, store.saved[0].ExpiresAt.Before(end.Add(time.Hour+time.Second)))
}

func TestDoResponseSkipsWhenPluginDisabled(t *testing.T) {
	t.Parallel()

	c := newTestContext(t)
	store := &recordingStore{}
	requestMeta := &meta.Meta{
		Mode:        mode.ChatCompletions,
		OriginModel: "gpt-5",
		ModelConfig: model.ModelConfig{Model: "gpt-5"},
		Group:       model.GroupCache{ID: "group-1"},
		Token:       model.TokenCache{ID: 7},
		Channel:     meta.ChannelMeta{ID: 9},
	}

	_, relayErr := (&Plugin{}).DoResponse(
		requestMeta,
		store,
		c,
		&http.Response{StatusCode: http.StatusOK},
		doResponseFunc{
			fn: func(_ *meta.Meta, _ adaptor.Store, c *gin.Context, _ *http.Response) (adaptor.DoResponseResult, adaptor.Error) {
				c.Status(http.StatusOK)
				_, _ = c.Writer.Write([]byte(`{"ok":true}`))

				return adaptor.DoResponseResult{Usage: model.Usage{CachedTokens: 4}}, nil
			},
		},
	)

	require.Nil(t, relayErr)
	assert.Empty(t, store.saved)
}

func TestSaveRecentStoreMappingDebouncesUpdates(t *testing.T) {
	t.Parallel()

	id := model.CacheFollowStoreID("gpt-5", model.CacheKeyTypeRecent)
	store := &recordingStore{
		stores: map[string]adaptor.StoreCache{
			id: {
				ID:        id,
				GroupID:   "group-1",
				TokenID:   7,
				ChannelID: 3,
				Model:     "gpt-5",
				UpdatedAt: time.Now().Add(-5 * time.Second),
				ExpiresAt: time.Now().Add(time.Minute),
			},
		},
	}

	err := saveRecentStoreMapping(
		store,
		id,
		&meta.Meta{
			OriginModel: "gpt-5",
			Group:       model.GroupCache{ID: "group-1"},
			Token:       model.TokenCache{ID: 7},
			Channel:     meta.ChannelMeta{ID: 9},
		},
		time.Now().Add(time.Minute),
		30*time.Second,
	)

	require.NoError(t, err)
	assert.Empty(t, store.saved)
}
