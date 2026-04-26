//go:build enterprise

package novita

import (
	"context"
	"slices"

	"github.com/labring/aiproxy/core/controller"
	"github.com/labring/aiproxy/core/enterprise/synccommon"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
)

// discoverGroup collapses concurrent auto-discovery calls for the same model
// into a single execution, preventing redundant DB writes and cache rebuilds.
var discoverGroup singleflight.Group

func init() {
	controller.RegisterPassthroughSuccessHook(onPassthroughFirstSuccess)
}

// onPassthroughFirstSuccess is called (in a background goroutine) after a
// passthrough-unknown request succeeds for the first time. For Novita channels
// it fetches pricing from the management API and registers a ModelConfig so
// subsequent requests are billed correctly and the model appears in users'
// "My Access" model list.
func onPassthroughFirstSuccess(
	ctx context.Context,
	channelID int,
	channelType model.ChannelType,
	modelName string,
) {
	switch channelType {
	case model.ChannelTypeNovita:
		_, _, _ = discoverGroup.Do(modelName, func() (any, error) {
			doDiscoverChat(ctx, channelID, modelName)
			return nil, nil
		})
	case model.ChannelTypeNovitaMultimodal:
		_, _, _ = discoverGroup.Do(modelName, func() (any, error) {
			doDiscoverMultimodal(ctx, channelID, modelName)
			return nil, nil
		})
	}
}

// doDiscoverChat handles auto-discovery for Novita OpenAI-compatible channels.
// Looks up the model in the V2 management API to get pricing and config.
func doDiscoverChat(ctx context.Context, channelID int, modelName string) {
	// PLEASE READ before changing this branch (and the multimodal mirror below):
	//
	// This path is currently UNREACHABLE — the hook only fires when
	// meta.ModelConfig.Type == unknownMode (relay-controller.go:304), which
	// requires the row not to exist yet. By the time modelExists() is true the
	// hook would already have skipped us.
	//
	// The finalizeChatDiscovery call would propagate the model into
	// channel.Models of all peer channels in the same set. That is
	// INTENTIONALLY suppressed: hand-mapped alias rows kept ONLY in
	// channel.model_mapping (e.g. claude-opus-4-6 -> pa/claude-opus-4-6) MUST
	// stay out of channel.Models so they don't appear in /v1/models — they're
	// typo-tolerance fallbacks, not first-class entries.
	//
	// If a future change makes the hook fire for already-registered rows, you
	// MUST re-evaluate whether propagation matches the alias design contract.
	// The conservative replacement is `return` only.
	if modelExists(modelName) {
		finalizeChatDiscovery(channelID, modelName, nil, "already-registered")
		return
	}

	client, err := NewNovitaClient()
	if err != nil {
		log.Printf("novita autodiscover: client creation failed: %v", err)
		return
	}

	cfg := GetNovitaConfig()
	if cfg.MgmtToken == "" {
		registerFallbackModel(channelID, modelName)
		return
	}

	remoteModel := discoverV2Model(ctx, client, cfg.MgmtToken, modelName)
	if remoteModel == nil {
		registerFallbackModel(channelID, modelName)
		return
	}

	if err := registerNovitaChatModel(modelName, remoteModel, cfg.ExchangeRate); err != nil {
		log.Printf("novita autodiscover: failed to register %s: %v", modelName, err)
		return
	}

	finalizeChatDiscovery(channelID, modelName, remoteModel, "V2 API")
}

// doDiscoverMultimodal handles auto-discovery for Novita native multimodal channels.
// Fetches pricing from the multimodal console API, falls back to V2 management API.
func doDiscoverMultimodal(ctx context.Context, channelID int, modelName string) {
	// PLEASE READ: this path is unreachable today — see the chat doDiscoverChat
	// guardrail comment for the design contract. Keep `return` semantics here
	// if propagation logic is removed elsewhere.
	if modelExists(modelName) {
		finalizeMultimodalDiscovery(channelID, modelName, "already-registered")
		return
	}

	client, err := NewNovitaClient()
	if err != nil {
		log.Printf("novita autodiscover: client creation failed: %v", err)
		registerMultimodalFallback(channelID, modelName)

		return
	}

	cfg := GetNovitaConfig()

	var perRequestPrice float64

	if cfg.MgmtToken != "" {
		perRequestPrice = discoverMultimodalPrice(
			ctx,
			client,
			cfg.MgmtToken,
			modelName,
			cfg.ExchangeRate,
		)

		// Fallback: try V2 management API for token-based pricing
		if perRequestPrice == 0 {
			if remoteModel := discoverV2Model(
				ctx,
				client,
				cfg.MgmtToken,
				modelName,
			); remoteModel != nil {
				if err := registerNovitaChatModel(
					modelName,
					remoteModel,
					cfg.ExchangeRate,
				); err != nil {
					log.Printf("novita autodiscover: failed to register %s: %v", modelName, err)
					return
				}

				finalizeMultimodalDiscovery(channelID, modelName, "V2 API")

				return
			}
		}
	}

	// Register with per-request pricing (or zero if no pricing found).
	// SyncedFrom intentionally empty — autodiscover writes "non-sync" rows
	// that the regular sync MUST NOT touch.
	// Default to PPIONative for genuinely unknown multimodal models;
	// the next daily sync will not correct the type since SyncedFrom is empty.
	mc := model.ModelConfig{
		Model: modelName,
		Owner: model.ModelOwnerNovita,
		Type:  mode.PPIONative,
		RPM:   60,
		TPM:   1000000,
	}

	if perRequestPrice > 0 {
		mc.Price.PerRequestPrice = model.ZeroNullFloat64(perRequestPrice)
	}

	if err := model.OnConflictDoNothing().Create(&mc).Error; err != nil {
		log.Printf("novita autodiscover: failed to register %s: %v", modelName, err)
		return
	}

	finalizeMultimodalDiscovery(channelID, modelName, "multimodal API")
}

// ─── helpers ────────────────────────────────────────────────────────────────

// modelExists checks if a ModelConfig entry already exists for the given model.
func modelExists(modelName string) bool {
	var count int64
	if err := model.DB.Model(&model.ModelConfig{}).
		Where("model = ?", modelName).
		Count(&count).Error; err != nil {
		log.Printf("novita autodiscover: count check failed for %s: %v", modelName, err)
		return false
	}

	return count > 0
}

// discoverV2Model searches the V2 management API for a model by name.
func discoverV2Model(
	ctx context.Context,
	client *NovitaClient,
	mgmtToken, modelName string,
) *NovitaModelV2 {
	all, err := client.FetchAllModels(ctx, mgmtToken)
	if err != nil {
		log.Printf("novita autodiscover: FetchAllModels failed (non-fatal): %v", err)
		return nil
	}

	for i := range all {
		if all[i].ID == modelName {
			return &all[i]
		}
	}

	return nil
}

// discoverMultimodalPrice fetches the multimodal model catalog and returns the
// minimum SKU price for the given model. Returns 0 if not found.
func discoverMultimodalPrice(
	ctx context.Context,
	client *NovitaClient,
	mgmtToken, modelName string,
	exchangeRate float64,
) float64 {
	mmModels, err := client.FetchMultimodalModels(ctx, mgmtToken)
	if err != nil {
		log.Printf("novita autodiscover: FetchMultimodalModels failed (non-fatal): %v", err)
		return 0
	}

	for i := range mmModels {
		if mmModels[i].FusionConfig.Name != modelName {
			continue
		}

		skuCodes := mmModels[i].collectSKUCodes()
		if len(skuCodes) == 0 {
			return 0
		}

		prices, priceErr := client.FetchMultimodalPrices(ctx, mgmtToken, skuCodes)
		if priceErr != nil {
			log.Printf(
				"novita autodiscover: FetchMultimodalPrices failed (non-fatal): %v",
				priceErr,
			)

			return 0
		}

		return mmModels[i].minSKUPrice(prices, exchangeRate)
	}

	return 0
}

// registerNovitaChatModel creates a ModelConfig entry for a Novita chat model
// using V2 management API data.
//
// SyncedFrom intentionally empty — autodiscover-registered rows are not
// managed by the regular sync (per CanSyncOwn).
func registerNovitaChatModel(
	modelName string,
	remoteModel *NovitaModelV2,
	exchangeRate float64,
) error {
	mc := model.ModelConfig{
		Model:  modelName,
		Owner:  model.ModelOwnerNovita,
		Type:   modeFromEndpoints(remoteModel.ModelType, remoteModel.Endpoints),
		RPM:    60,
		TPM:    1000000,
		Config: synccommon.ToModelConfigKeys(buildConfigFromV2Model(remoteModel)),
	}

	if remoteModel.RPM > 0 {
		mc.RPM = int64(remoteModel.RPM)
	}

	if remoteModel.TPM > 0 {
		mc.TPM = int64(remoteModel.TPM)
	}

	setPriceFromV2Model(&mc.Price, remoteModel, exchangeRate)

	return model.OnConflictDoNothing().Create(&mc).Error
}

// registerFallbackModel creates a zero-cost ModelConfig entry when no pricing
// data is available.
//
// SyncedFrom intentionally empty — autodiscover/fallback rows are not managed
// by the regular sync. Admin must manually update pricing.
func registerFallbackModel(channelID int, modelName string) {
	mc := model.ModelConfig{
		Model: modelName,
		Owner: model.ModelOwnerNovita,
		Type:  mode.ChatCompletions,
		RPM:   60,
		TPM:   1000000,
	}

	if err := model.OnConflictDoNothing().Create(&mc).Error; err != nil {
		log.Printf("novita autodiscover: failed to register fallback %s: %v", modelName, err)
		return
	}

	finalizeChatDiscovery(channelID, modelName, nil, "fallback, zero-cost")
}

// registerMultimodalFallback creates a zero-cost multimodal ModelConfig entry.
//
// SyncedFrom intentionally empty — autodiscover/fallback rows are not managed
// by the regular sync.
func registerMultimodalFallback(channelID int, modelName string) {
	mc := model.ModelConfig{
		Model: modelName,
		Owner: model.ModelOwnerNovita,
		Type:  mode.PPIONative,
		RPM:   60,
		TPM:   1000000,
	}

	if err := model.OnConflictDoNothing().Create(&mc).Error; err != nil {
		log.Printf(
			"novita autodiscover: failed to register multimodal fallback %s: %v",
			modelName,
			err,
		)

		return
	}

	finalizeMultimodalDiscovery(channelID, modelName, "fallback")
}

// finalizeChatDiscovery propagates the model to all peer Novita chat channels
// in the same set as the origin channel — and to peer Anthropic-protocol
// channels when the upstream model declares the anthropic endpoint. Refreshes
// the cache. Called after successful model registration (or for already-
// registered rows so channel.Models gets caught up).
func finalizeChatDiscovery(
	channelID int,
	modelName string,
	remoteModel *NovitaModelV2,
	source string,
) {
	if err := synccommon.AddModelToPeerChannels(
		model.DB,
		channelID,
		model.ChannelTypeNovita,
		modelName,
	); err != nil {
		log.Printf(
			"novita autodiscover: failed to propagate %s to peer chat channels: %v",
			modelName,
			err,
		)
	}

	if remoteModel != nil && slices.Contains(remoteModel.Endpoints, "anthropic") {
		addModelToAnthropicChannel(channelID, modelName)
	}

	if err := model.InitModelConfigAndChannelCache(); err != nil {
		log.Printf(
			"novita autodiscover: cache refresh failed after registering %s: %v",
			modelName,
			err,
		)
	}

	log.Printf("novita autodiscover: registered model %s (%s)", modelName, source)
}

// addModelToAnthropicChannel adds modelName to all Novita Anthropic channels
// that share at least one Set with originChannelID. Anthropic channels share
// ChannelTypeAnthropic with non-Novita channels, so we filter by base_url.
func addModelToAnthropicChannel(originChannelID int, modelName string) {
	var origin model.Channel
	if err := model.DB.Select("id, sets").First(&origin, originChannelID).Error; err != nil {
		log.Printf(
			"novita autodiscover: read origin channel %d for anthropic peer fan-out: %v",
			originChannelID,
			err,
		)

		return
	}

	originSets := origin.GetSets()

	var channels []model.Channel
	if err := model.DB.Where(novitaChannelWhere(), novitaChannelArgs()...).
		Where("type = ? AND status = ?", model.ChannelTypeAnthropic, 1).
		Find(&channels).Error; err != nil {
		return
	}

	for i := range channels {
		if !synccommon.SetsIntersect(originSets, channels[i].GetSets()) {
			continue
		}

		if slices.Contains(channels[i].Models, modelName) {
			continue
		}

		channels[i].Models = append(channels[i].Models, modelName)
		if err := model.DB.Save(&channels[i]).Error; err != nil {
			log.Printf(
				"novita autodiscover: failed to add %s to Anthropic channel %d: %v",
				modelName,
				channels[i].ID,
				err,
			)
		}
	}
}

// finalizeMultimodalDiscovery propagates the model to all peer Novita
// multimodal channels in the same set as the origin channel and refreshes the
// cache. Called after successful model registration (or when a pre-existing
// row was reused).
func finalizeMultimodalDiscovery(channelID int, modelName, source string) {
	if err := synccommon.AddModelToPeerChannels(
		model.DB,
		channelID,
		model.ChannelTypeNovitaMultimodal,
		modelName,
	); err != nil {
		log.Printf(
			"novita autodiscover: failed to propagate %s to peer multimodal channels: %v",
			modelName,
			err,
		)
	}

	if err := model.InitModelConfigAndChannelCache(); err != nil {
		log.Printf(
			"novita autodiscover: cache refresh failed after registering %s: %v",
			modelName,
			err,
		)
	}

	log.Printf("novita autodiscover: registered model %s (%s)", modelName, source)
}
