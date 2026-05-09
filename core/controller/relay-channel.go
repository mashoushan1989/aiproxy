package controller

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/config"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/monitor"
	"github.com/labring/aiproxy/core/relay/adaptors"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/plugin/cachefollow"
	log "github.com/sirupsen/logrus"
)

const (
	AIProxyChannelHeader = "Aiproxy-Channel"
	// maxRetryErrorRate is the maximum error rate threshold for channel retry selection
	// Channels with error rate higher than this will be filtered out during retry
	maxRetryErrorRate = 0.75
)

func GetChannelFromHeader(
	header string,
	mc *model.ModelCaches,
	availableSet []string,
	model string,
	m mode.Mode,
) (*model.Channel, error) {
	channelIDInt, err := strconv.ParseInt(header, 10, 64)
	if err != nil {
		return nil, err
	}

	for _, set := range availableSet {
		enabledChannels := mc.EnabledModel2ChannelsBySet[set][model]
		if len(enabledChannels) > 0 {
			for _, channel := range enabledChannels {
				if int64(channel.ID) == channelIDInt {
					a, ok := adaptors.GetAdaptor(channel.Type)
					if !ok {
						return nil, fmt.Errorf("adaptor not found for channel %d", channel.ID)
					}

					if !a.SupportMode(m) {
						return nil, fmt.Errorf("channel %d not supported by adaptor", channel.ID)
					}

					return channel, nil
				}
			}
		}

		disabledChannels := mc.DisabledModel2ChannelsBySet[set][model]
		if len(disabledChannels) > 0 {
			for _, channel := range disabledChannels {
				if int64(channel.ID) == channelIDInt {
					a, ok := adaptors.GetAdaptor(channel.Type)
					if !ok {
						return nil, fmt.Errorf("adaptor not found for channel %d", channel.ID)
					}

					if !a.SupportMode(m) {
						return nil, fmt.Errorf("channel %d not supported by adaptor", channel.ID)
					}

					return channel, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("channel %d not found for model `%s`", channelIDInt, model)
}

func needPinChannel(m mode.Mode) bool {
	switch m {
	case mode.VideoGenerationsGetJobs,
		mode.VideoGenerationsContent,
		mode.ResponsesGet,
		mode.ResponsesDelete,
		mode.ResponsesCancel,
		mode.ResponsesInputItems:
		return true
	default:
		return false
	}
}

func GetChannelFromRequest(
	c *gin.Context,
	mc *model.ModelCaches,
	availableSet []string,
	modelName string,
	m mode.Mode,
) (*model.Channel, error) {
	channelID := middleware.GetChannelID(c)
	if channelID == 0 {
		if needPinChannel(m) {
			return nil, fmt.Errorf("%s need pinned channel", m)
		}
		return nil, nil
	}

	for _, set := range availableSet {
		enabledChannels := mc.EnabledModel2ChannelsBySet[set][modelName]
		if len(enabledChannels) > 0 {
			for _, channel := range enabledChannels {
				if channel.ID == channelID {
					a, ok := adaptors.GetAdaptor(channel.Type)
					if !ok {
						return nil, fmt.Errorf(
							"adaptor not found for pinned channel %d",
							channel.ID,
						)
					}

					if !a.SupportMode(m) {
						return nil, fmt.Errorf(
							"pinned channel %d not supported by adaptor",
							channel.ID,
						)
					}

					return channel, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("pinned channel %d not found for model `%s`", channelID, modelName)
}

var (
	ErrChannelsNotFound  = errors.New("channels not found")
	ErrChannelsExhausted = errors.New("channels exhausted")
)

func getRandomChannel(
	mc *model.ModelCaches,
	availableSet []string,
	modelName string,
	mode mode.Mode,
	preferChannelIDs []int,
	errorRates map[int64]float64,
	maxErrorRate float64,
	ignoreChannelMap ...map[int64]struct{},
) (*model.Channel, []*model.Channel, error) {
	channelMap := make(map[int]*model.Channel)
	if len(availableSet) != 0 {
		for _, set := range availableSet {
			channels := mc.EnabledModel2ChannelsBySet[set][modelName]
			for _, channel := range channels {
				a, ok := adaptors.GetAdaptor(channel.Type)
				if !ok {
					continue
				}

				if !a.SupportMode(mode) {
					continue
				}

				channelMap[channel.ID] = channel
			}
		}
	} else {
		for _, sets := range mc.EnabledModel2ChannelsBySet {
			for _, channel := range sets[modelName] {
				a, ok := adaptors.GetAdaptor(channel.Type)
				if !ok {
					continue
				}

				if !a.SupportMode(mode) {
					continue
				}

				channelMap[channel.ID] = channel
			}
		}
	}

	// If no model-specific channels were found, fall back to passthrough channels
	// (channels with allow_passthrough_unknown=true) that can handle any model.
	if len(channelMap) == 0 {
		passthroughIter := func(channels []*model.Channel) {
			for _, channel := range channels {
				a, ok := adaptors.GetAdaptor(channel.Type)
				if !ok {
					continue
				}

				if !a.SupportMode(mode) {
					continue
				}

				channelMap[channel.ID] = channel
			}
		}

		if len(availableSet) != 0 {
			for _, set := range availableSet {
				passthroughIter(mc.PassthroughChannelsBySet[set])
			}
		} else {
			for _, channels := range mc.PassthroughChannelsBySet {
				passthroughIter(channels)
			}
		}
	}

	migratedChannels := make([]*model.Channel, 0, len(channelMap))
	for _, channel := range channelMap {
		migratedChannels = append(migratedChannels, channel)
	}

	if channel := pickPreferredChannel(
		migratedChannels,
		mode,
		preferChannelIDs,
		errorRates,
		maxErrorRate,
		ignoreChannelMap...,
	); channel != nil {
		return channel, migratedChannels, nil
	}

	channel, err := ignoreChannel(
		migratedChannels,
		mode,
		errorRates,
		maxErrorRate,
		ignoreChannelMap...,
	)

	return channel, migratedChannels, err
}

func pickPreferredChannel(
	channels []*model.Channel,
	mode mode.Mode,
	preferChannelIDs []int,
	errorRates map[int64]float64,
	maxErrorRate float64,
	ignoreChannelIDs ...map[int64]struct{},
) *model.Channel {
	if len(preferChannelIDs) == 0 {
		return nil
	}

	channelMap := make(map[int]*model.Channel, len(channels))
	for _, channel := range channels {
		channelMap[channel.ID] = channel
	}

	seen := make(map[int]struct{}, len(preferChannelIDs))
	for _, channelID := range preferChannelIDs {
		if _, ok := seen[channelID]; ok {
			continue
		}

		seen[channelID] = struct{}{}

		channel, ok := channelMap[channelID]
		if !ok {
			continue
		}

		filtered := filterChannels(
			[]*model.Channel{channel},
			mode,
			errorRates,
			maxErrorRate,
			ignoreChannelIDs...,
		)
		if len(filtered) > 0 {
			return filtered[0]
		}
	}

	return nil
}

func getPriority(channel *model.Channel, errorRate float64) int32 {
	priority := channel.GetPriority()

	if errorRate > 1 {
		errorRate = 1
	} else if errorRate < 0.1 {
		errorRate = 0.1
	}

	return int32(float64(priority) / errorRate)
}

func ignoreChannel(
	channels []*model.Channel,
	mode mode.Mode,
	errorRates map[int64]float64,
	maxErrorRate float64,
	ignoreChannelIDs ...map[int64]struct{},
) (*model.Channel, error) {
	if len(channels) == 0 {
		return nil, ErrChannelsNotFound
	}

	channels = filterChannels(channels, mode, errorRates, maxErrorRate, ignoreChannelIDs...)
	if len(channels) == 0 {
		return nil, ErrChannelsExhausted
	}

	if len(channels) == 1 {
		return channels[0], nil
	}

	var totalWeight int32

	cachedPrioritys := make([]int32, len(channels))
	for i, ch := range channels {
		priority := getPriority(ch, errorRates[int64(ch.ID)])
		totalWeight += priority
		cachedPrioritys[i] = priority
	}

	if totalWeight == 0 {
		return channels[rand.IntN(len(channels))], nil
	}

	r := rand.Int32N(totalWeight)
	for i, ch := range channels {
		r -= cachedPrioritys[i]
		if r < 0 {
			return ch, nil
		}
	}

	return channels[rand.IntN(len(channels))], nil
}

// getChannelWithFallback selects a channel by trying each set in availableSet
// in order. Within a set it first filters by error rate; if all channels in the
// set are exhausted it retries without the error-rate cap. Only when the current
// set has NO channels at all does it advance to the next set. This ensures
// overseas nodes prefer their own channels and only fall back to default when
// the preferred set cannot serve the model.
func getChannelWithFallback(
	cache *model.ModelCaches,
	availableSet []string,
	modelName string,
	mode mode.Mode,
	preferChannelIDs []int,
	errorRates map[int64]float64,
	ignoreChannelIDs map[int64]struct{},
) (*model.Channel, []*model.Channel, error) {
	// Fast path: single set (domestic nodes) — no ordering needed.
	if len(availableSet) <= 1 {
		return getChannelFromSingleSet(
			cache,
			availableSet,
			modelName,
			mode,
			preferChannelIDs,
			errorRates,
			ignoreChannelIDs,
		)
	}

	// Multi-set path: try each set in priority order.
	strict := config.GetStrictNodeSet()

	for i, set := range availableSet {
		singleSet := []string{set}

		channel, migratedChannels, err := getRandomChannel(
			cache,
			singleSet,
			modelName,
			mode,
			preferChannelIDs,
			errorRates,
			maxRetryErrorRate,
			ignoreChannelIDs,
		)
		if err == nil {
			return channel, migratedChannels, nil
		}

		// No channels registered for this model in this set — try next set.
		// In strict mode, the FIRST set's NotFound is an unconditional hard
		// fail too: the primary set is the only acceptable destination.
		if errors.Is(err, ErrChannelsNotFound) {
			if strict && i == 0 {
				logShadowStrictWouldReject(modelName, set, "not_found")
				return nil, nil, err
			}

			logShadowStrictWouldReject(modelName, set, "not_found_soft_fallback")

			continue
		}

		// Channels exist but all exceeded error threshold — retry without
		// the error-rate cap but still respecting bans (ignoreChannelIDs).
		// Unlike the single-set fast path which drops bans as a last resort,
		// here we keep bans so that a fully-banned set falls through to the
		// next set rather than resurrecting a banned channel.
		if errors.Is(err, ErrChannelsExhausted) {
			channel, migratedChannels, err = getRandomChannel(
				cache,
				singleSet,
				modelName,
				mode,
				nil,
				errorRates,
				0,
				ignoreChannelIDs,
			)
			if err == nil {
				return channel, migratedChannels, nil
			}

			// Still exhausted (all banned). In strict mode, the primary set
			// is final — hard fail rather than route to a different set.
			if strict && i == 0 {
				logShadowStrictWouldReject(modelName, set, "exhausted")
				return nil, migratedChannels, err
			}

			logShadowStrictWouldReject(modelName, set, "exhausted_soft_fallback")

			continue
		}

		// Unexpected error — return immediately.
		return nil, migratedChannels, err
	}

	return nil, nil, ErrChannelsNotFound
}

// getChannelFromSingleSet is the original two-phase selection for a single set
// (or when availableSet is empty). Kept as a fast path to avoid per-set loop
// overhead for the common domestic-node case.
func getChannelFromSingleSet(
	cache *model.ModelCaches,
	availableSet []string,
	modelName string,
	mode mode.Mode,
	preferChannelIDs []int,
	errorRates map[int64]float64,
	ignoreChannelIDs map[int64]struct{},
) (*model.Channel, []*model.Channel, error) {
	channel, migratedChannels, err := getRandomChannel(
		cache,
		availableSet,
		modelName,
		mode,
		preferChannelIDs,
		errorRates,
		maxRetryErrorRate,
		ignoreChannelIDs,
	)
	if err == nil {
		return channel, migratedChannels, nil
	}

	if !errors.Is(err, ErrChannelsExhausted) {
		return nil, migratedChannels, err
	}

	return getRandomChannel(
		cache,
		availableSet,
		modelName,
		mode,
		nil,
		errorRates,
		0,
	)
}

type initialChannel struct {
	channel           *model.Channel
	designatedChannel bool
	preferChannelIDs  []int
	ignoreChannelIDs  map[int64]struct{}
	errorRates        map[int64]float64
	migratedChannels  []*model.Channel
}

func getInitialChannel(c *gin.Context, modelName string, m mode.Mode) (*initialChannel, error) {
	log := common.GetLogger(c)

	group := middleware.GetGroup(c)
	availableSet := group.GetAvailableSets()

	if channelHeader := c.Request.Header.Get(AIProxyChannelHeader); channelHeader != "" {
		if group.Status != model.GroupStatusInternal {
			return nil, errors.New("channel header is not allowed in non-internal group")
		}

		channel, err := GetChannelFromHeader(
			channelHeader,
			middleware.GetModelCaches(c),
			availableSet,
			modelName,
			m,
		)
		if err != nil {
			return nil, err
		}

		log.Data["designated_channel"] = "true"

		return &initialChannel{channel: channel, designatedChannel: true}, nil
	}

	channel, err := GetChannelFromRequest(
		c,
		middleware.GetModelCaches(c),
		availableSet,
		modelName,
		m,
	)
	if err != nil {
		return nil, err
	}

	if channel != nil {
		return &initialChannel{channel: channel, designatedChannel: true}, nil
	}

	mc := middleware.GetModelCaches(c)

	ignoreChannelIDs, err := monitor.GetBannedChannelsMapWithModel(c.Request.Context(), modelName)
	if err != nil {
		log.Errorf("get %s auto banned channels failed: %+v", modelName, err)
	}

	log.Debugf("%s model banned channels: %+v", modelName, ignoreChannelIDs)

	errorRates, err := monitor.GetModelChannelErrorRate(c.Request.Context(), modelName)
	if err != nil {
		log.Errorf("get channel model error rates failed: %+v", err)
	}

	preferChannelIDs := getPreferChannelIDs(c, modelName, m)
	if len(preferChannelIDs) > 0 {
		log.Data["prefer_channels"] = fmt.Sprintf("%v", preferChannelIDs)
	}

	channel, migratedChannels, err := getChannelWithFallback(
		mc,
		availableSet,
		modelName,
		m,
		preferChannelIDs,
		errorRates,
		ignoreChannelIDs,
	)
	if err != nil {
		return nil, err
	}

	return &initialChannel{
		channel:          channel,
		preferChannelIDs: preferChannelIDs,
		ignoreChannelIDs: ignoreChannelIDs,
		errorRates:       errorRates,
		migratedChannels: migratedChannels,
	}, nil
}

func supportsPromptCacheKeyMode(m mode.Mode) bool {
	switch m {
	case mode.Responses, mode.ChatCompletions:
		return true
	default:
		return false
	}
}

func supportsCacheFollowMode(m mode.Mode) bool {
	switch m {
	case mode.Responses, mode.ChatCompletions, mode.Gemini, mode.Anthropic:
		return true
	default:
		return false
	}
}

func getCacheFollowConfig(c *gin.Context) (cachefollow.Config, bool) {
	modelConfig := middleware.GetModelConfig(c)

	pluginConfig := cachefollow.Config{}
	if err := modelConfig.LoadPluginConfig(cachefollow.PluginName, &pluginConfig); err != nil {
		return cachefollow.Config{}, false
	}

	if !pluginConfig.Enable {
		return cachefollow.Config{}, false
	}

	return pluginConfig, true
}

func getPreferChannelIDs(c *gin.Context, modelName string, m mode.Mode) []int {
	pluginConfig, ok := getCacheFollowConfig(c)
	if !supportsCacheFollowMode(m) || !ok {
		return nil
	}

	group := middleware.GetGroup(c)
	token := middleware.GetToken(c)
	user := middleware.GetRequestUser(c)
	preferChannelIDs := make([]int, 0, 6)
	seen := make(map[int]struct{}, 6)

	appendChannelID := func(storeID string) {
		if storeID == "" {
			return
		}

		store, err := model.CacheGetStore(group.ID, token.ID, storeID)
		if err != nil || store.ChannelID == 0 {
			return
		}

		if !store.ExpiresAt.IsZero() && !store.ExpiresAt.After(time.Now()) {
			return
		}

		if _, ok := seen[store.ChannelID]; ok {
			return
		}

		seen[store.ChannelID] = struct{}{}
		preferChannelIDs = append(preferChannelIDs, store.ChannelID)
	}

	if supportsPromptCacheKeyMode(m) {
		if promptCacheKey := middleware.GetPromptCacheKey(c); promptCacheKey != "" {
			appendChannelID(
				model.PromptCacheStoreID(
					modelName,
					promptCacheKey,
					model.CacheKeyTypeStable,
				),
			)
			appendChannelID(
				model.PromptCacheStoreID(
					modelName,
					promptCacheKey,
					model.CacheKeyTypeRecent,
				),
			)
		}
	}

	if user != "" {
		appendChannelID(model.CacheFollowUserStoreID(modelName, user, model.CacheKeyTypeStable))
		appendChannelID(model.CacheFollowUserStoreID(modelName, user, model.CacheKeyTypeRecent))
	}

	if pluginConfig.EnableGenericFollow {
		appendChannelID(model.CacheFollowStoreID(modelName, model.CacheKeyTypeStable))
		appendChannelID(model.CacheFollowStoreID(modelName, model.CacheKeyTypeRecent))
	}

	if len(preferChannelIDs) == 0 {
		return nil
	}

	return preferChannelIDs
}

func getWebSearchChannel(
	ctx context.Context,
	mc *model.ModelCaches,
	modelName string,
) (*model.Channel, error) {
	ignoreChannelIDs, _ := monitor.GetBannedChannelsMapWithModel(ctx, modelName)
	errorRates, _ := monitor.GetModelChannelErrorRate(ctx, modelName)

	channel, _, err := getChannelWithFallback(
		mc,
		nil,
		modelName,
		mode.ChatCompletions,
		nil,
		errorRates,
		ignoreChannelIDs)
	if err != nil {
		return nil, err
	}

	return channel, nil
}

func getRetryChannel(state *retryState, currentRetry, totalRetries int) (*model.Channel, error) {
	if state.exhausted {
		if state.lastHasPermissionChannel == nil {
			return nil, ErrChannelsExhausted
		}

		// Check if lastHasPermissionChannel has high error rate
		// If so, return exhausted to prevent retrying with a bad channel
		channelID := int64(state.lastHasPermissionChannel.ID)
		if errorRate, ok := state.errorRates[channelID]; ok && errorRate > maxRetryErrorRate {
			return nil, ErrChannelsExhausted
		}

		return state.lastHasPermissionChannel, nil
	}

	// For the last retry, filter out all previously failed channels if there are other options
	if currentRetry == totalRetries-1 && len(state.failedChannelIDs) > 0 {
		// Check if there are channels available after filtering out failed channels
		newChannel, err := ignoreChannel(
			state.migratedChannels,
			state.meta.Mode,
			state.errorRates,
			maxRetryErrorRate,
			state.ignoreChannelIDs,
			state.failedChannelIDs,
		)
		if err == nil {
			return newChannel, nil
		}
		// If no channels available after filtering, fall back to not using failed channels filter
	}

	newChannel, err := ignoreChannel(
		state.migratedChannels,
		state.meta.Mode,
		state.errorRates,
		maxRetryErrorRate,
		state.ignoreChannelIDs,
	)
	if err != nil {
		if !errors.Is(err, ErrChannelsExhausted) || state.lastHasPermissionChannel == nil {
			return nil, err
		}

		// Check if lastHasPermissionChannel has high error rate before using it
		channelID := int64(state.lastHasPermissionChannel.ID)
		if errorRate, ok := state.errorRates[channelID]; ok && errorRate > maxRetryErrorRate {
			return nil, ErrChannelsExhausted
		}

		state.exhausted = true

		return state.lastHasPermissionChannel, nil
	}

	return newChannel, nil
}

func filterChannels(
	channels []*model.Channel,
	mode mode.Mode,
	errorRates map[int64]float64,
	maxErrorRate float64,
	ignoreChannel ...map[int64]struct{},
) []*model.Channel {
	return filterChannelsWithRoutingPolicy(
		channels,
		mode,
		model.RoutingPolicyPureThenConvert,
		errorRates,
		maxErrorRate,
		ignoreChannel...,
	)
}

// filterByErrorRate returns channels whose error rate is at or below the threshold.
// Returns the input slice unchanged when no channel exceeds the threshold.
func filterByErrorRate(
	channels []*model.Channel,
	errorRates map[int64]float64,
	maxErrorRate float64,
) []*model.Channel {
	// Fast path: find first channel that needs filtering.
	firstBad := -1

	for i, ch := range channels {
		if rate, ok := errorRates[int64(ch.ID)]; ok && rate > maxErrorRate {
			firstBad = i

			break
		}
	}

	if firstBad < 0 {
		return channels
	}

	result := make([]*model.Channel, firstBad, len(channels))
	copy(result, channels[:firstBad])

	for _, ch := range channels[firstBad+1:] {
		if rate, ok := errorRates[int64(ch.ID)]; ok && rate > maxErrorRate {
			continue
		}

		result = append(result, ch)
	}

	return result
}

// shadowStrictRejectLog rate-limits the strict-mode shadow log to prevent
// disk fill-up under high QPS × high fallback rate. Each (model,set,reason)
// tuple emits at most one entry per shadowStrictLogInterval. Operators read
// these warnings during the shadow-observation window before flipping
// STRICT_NODE_SET=true; emitting one summary line per minute per tuple is
// enough to inform the rollout decision and won't blow up log storage.
const shadowStrictLogInterval = time.Minute

// shadowStrictMaxKeys caps the rate-limit map so an unbounded number of
// distinct (model,set,reason) tuples can't leak memory over a long-running
// process. When the cap is hit we drop the whole map: at worst this re-emits
// every active tuple once, which is what the shadow log is for anyway.
const shadowStrictMaxKeys = 4096

type shadowStrictKey struct {
	model, set, reason string
}

var (
	shadowStrictMu       sync.Mutex
	shadowStrictLastSeen = make(map[shadowStrictKey]time.Time)
)

// logShadowStrictWouldReject records a one-line WARN for routing decisions
// affected by strict mode. Behavior depends on whether strict is enabled:
//   - strict=false (shadow mode): emits "shadow_strict_would_reject" — used
//     during the rollout-observation window to preview impact.
//   - strict=true (enforcement): emits "strict_node_set_reject" — gives ops a
//     greppable log line when strict mode actually 404s a request, so root
//     cause is visible in production.
//
// Rate-limited to one entry per (model, set, reason) tuple per minute to
// protect log storage under high QPS × high reject rate.
func logShadowStrictWouldReject(modelName, set, reason string) {
	key := shadowStrictKey{model: modelName, set: set, reason: reason}

	shadowStrictMu.Lock()

	if len(shadowStrictLastSeen) >= shadowStrictMaxKeys {
		shadowStrictLastSeen = make(map[shadowStrictKey]time.Time)
	}

	last, ok := shadowStrictLastSeen[key]

	now := time.Now()
	if ok && now.Sub(last) < shadowStrictLogInterval {
		shadowStrictMu.Unlock()
		return
	}

	shadowStrictLastSeen[key] = now
	shadowStrictMu.Unlock()

	if config.GetStrictNodeSet() {
		log.Warnf("strict_node_set_reject model=%s set=%s reason=%s", modelName, set, reason)
		return
	}

	log.Warnf("shadow_strict_would_reject model=%s set=%s reason=%s", modelName, set, reason)
}
