package qianfan

import (
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Config struct {
	AppID string `json:"appid"`
}

var channelConfigCache utils.ChannelConfigCache[Config]

func loadConfig(meta *meta.Meta) (Config, error) {
	return channelConfigCache.Load(meta, Config{})
}

func configSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"appid": map[string]any{
				"type":        "string",
				"title":       "AppID Header",
				"description": "Optional Qianfan appid header used to distinguish usage and billing by application.",
			},
		},
	}
}
