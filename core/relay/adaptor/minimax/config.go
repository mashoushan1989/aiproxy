package minimax

import "github.com/labring/aiproxy/core/relay/meta"

type Config struct {
	UseChatCompletionsPath bool `json:"use_chat_completions_path"`
}

func loadConfig(meta *meta.Meta) (Config, error) {
	cfg := Config{}
	if meta == nil {
		return cfg, nil
	}

	return cfg, meta.ChannelConfigs.LoadConfig(&cfg)
}
