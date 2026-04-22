package novita

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

// https://novita.ai/llm-api

const (
	ModelNovitaTavilySearch = "novita-tavily-search"
)

var ModelList = []model.ModelConfig{
	{
		Model: "meta-llama/llama-3-8b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "meta-llama/llama-3-70b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "nousresearch/hermes-2-pro-llama-3-8b",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "nousresearch/nous-hermes-llama2-13b",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "mistralai/mistral-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "teknium/openhermes-2.5-mistral-7b",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "microsoft/wizardlm-2-8x22b",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMicrosoft,
	},

	// Tavily Search (virtual model for Novita's Tavily search API)
	{
		Model: ModelNovitaTavilySearch,
		Type:  mode.WebSearch,
		Owner: model.ModelOwnerNovita,
	},
}

// VirtualWebSearchModels returns the names of virtual WebSearch models declared
// in ModelList. These models are not returned by the upstream /v1/models API,
// so the sync pipeline must inject them into the channel Models list explicitly
// to keep routing functional.
func VirtualWebSearchModels() []string {
	names := make([]string, 0, 1)

	for i := range ModelList {
		if ModelList[i].Type == mode.WebSearch && ModelList[i].Owner == model.ModelOwnerNovita {
			names = append(names, ModelList[i].Model)
		}
	}

	return names
}
