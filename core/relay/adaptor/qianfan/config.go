package qianfan

import "github.com/labring/aiproxy/core/relay/meta"

type Config struct {
	AppID string `json:"appid"`
}

func loadConfig(meta *meta.Meta) (Config, error) {
	cfg := Config{}
	if meta == nil {
		return cfg, nil
	}

	return cfg, meta.ChannelConfigs.LoadConfig(&cfg)
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
