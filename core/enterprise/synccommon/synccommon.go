//go:build enterprise

// Package synccommon provides provider-agnostic utilities shared by the PPIO
// and Novita sync implementations.
package synccommon

import (
	"strings"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

// ModelConfigSyncPriceLockedKey marks synced model configs whose local Price
// should not be overwritten by provider catalog sync.
const ModelConfigSyncPriceLockedKey model.ModelConfigKey = "sync_price_locked"

// IsAnthropicModelName returns true if the model name pattern strongly suggests
// Anthropic protocol support, used as fallback when the upstream endpoints field
// is empty or incomplete.
func IsAnthropicModelName(id string) bool {
	return strings.HasPrefix(id, "claude-") || strings.HasPrefix(id, "pa/claude-")
}

// MultimodalCategoryToModelType maps the multimodal API's category field
// (e.g. "image_gen", "video_gen", "audio_gen") to model_type strings used by
// provider-specific mode inference. Returns "" for unrecognized categories so
// callers can skip non-multimodal entries that the multimodal API sometimes includes.
func MultimodalCategoryToModelType(category string) string {
	switch category {
	case "image_gen":
		return "image"
	case "video_gen":
		return "video"
	case "audio_gen":
		return "audio"
	default:
		return ""
	}
}

// SyncProgressEvent represents a progress event sent via SSE during a model sync.
// Shared by all provider sync implementations to ensure a consistent wire format.
type SyncProgressEvent struct {
	Type     string `json:"type"`    // "progress", "success", "error"
	Step     string `json:"step"`    // "fetching", "comparing", "syncing", "complete"
	Message  string `json:"message"` // Human-readable message
	Progress int    `json:"progress,omitempty"`
	Data     any    `json:"data,omitempty"` // Additional data (e.g., SyncResult on success)
}

// SendProgress delivers a progress event to callback if one is registered.
// step == "complete" upgrades the event type to "success".
func SendProgress(
	callback func(event SyncProgressEvent),
	step, message string,
	progress int,
	data any,
) {
	if callback == nil {
		return
	}

	eventType := "progress"
	if step == "complete" {
		eventType = "success"
	}

	callback(SyncProgressEvent{
		Type:     eventType,
		Step:     step,
		Message:  message,
		Progress: progress,
		Data:     data,
	})
}

// InferToolChoice returns true when a model is likely to support tool_choice.
// Feature-list signals ("tool_use", "function_calling", "tools") take highest
// priority; failing that, all "chat" models are assumed to support tool calling.
func InferToolChoice(modelType string, features []string) bool {
	for _, f := range features {
		switch f {
		case "tool_use", "function_calling", "tools":
			return true
		}
	}

	return modelType == "chat"
}

// ToModelConfigKeys converts map[string]any to map[model.ModelConfigKey]any
// without a JSON round-trip.
func ToModelConfigKeys(m map[string]any) map[model.ModelConfigKey]any {
	out := make(map[model.ModelConfigKey]any, len(m))

	for k, v := range m {
		out[model.ModelConfigKey(k)] = v
	}

	return out
}

// ComparableModelConfig returns config suitable for provider metadata
// comparisons. Local sync-control markers are intentionally excluded because
// upstream provider APIs do not return them.
func ComparableModelConfig(config map[model.ModelConfigKey]any) map[model.ModelConfigKey]any {
	out := make(map[model.ModelConfigKey]any, len(config))

	for k, v := range config {
		if strings.HasPrefix(string(k), "sync_") {
			continue
		}

		out[k] = v
	}

	return out
}

// IsSyncPriceLocked reports whether provider sync should preserve the existing
// local Price for this model config.
func IsSyncPriceLocked(config map[model.ModelConfigKey]any) bool {
	locked, _ := config[ModelConfigSyncPriceLockedKey].(bool)
	return locked
}

// PreserveSyncLocalConfig copies local sync-control markers from existing into
// next. Provider metadata remains fully refreshed.
func PreserveSyncLocalConfig(existing, next map[model.ModelConfigKey]any) {
	if IsSyncPriceLocked(existing) {
		next[ModelConfigSyncPriceLockedKey] = true
	}
}

// IsLocalOnlyMode returns true for model types that are generated locally and
// not sourced from the standard V1/V2 remote model list API. These must be
// excluded from delete detection during sync diagnostics to avoid false positives.
func IsLocalOnlyMode(t mode.Mode) bool {
	return t == mode.WebSearch || t == mode.PPIONative
}

// AdjustTierBounds returns the effective [minTokens, maxTokens] for a tiered-billing
// tier, bumping minTokens by 1 when it overlaps with prevMax (the previous tier's max).
//
// Providers use inclusive boundaries like [0,128000],[128000,∞] but aiproxy requires
// non-overlapping ranges. Pass prevMax=0 for the first tier (index 0).
func AdjustTierBounds(minTokens, maxTokens, prevMax int64) (int64, int64) {
	if minTokens > 0 && prevMax > 0 && minTokens <= prevMax {
		minTokens = prevMax + 1
	}

	return minTokens, maxTokens
}

// PromoteFirstTierToBasePrice fills in base InputPrice/OutputPrice from the
// first conditional tier when the upstream model is pure tiered-billing (base=0
// but all requests hit a tier). This ensures display layers show a meaningful
// price instead of "free".
func PromoteFirstTierToBasePrice(price *model.Price) {
	if price.InputPrice != 0 || price.OutputPrice != 0 {
		return
	}
	if len(price.ConditionalPrices) == 0 {
		return
	}
	first := price.ConditionalPrices[0].Price
	price.InputPrice = first.InputPrice
	price.InputPriceUnit = first.InputPriceUnit
	price.OutputPrice = first.OutputPrice
	price.OutputPriceUnit = first.OutputPriceUnit
}
