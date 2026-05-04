package minimax

import (
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Config struct {
	UseChatCompletionsPath bool `json:"use_chat_completions_path"`
}

var channelConfigCache utils.ChannelConfigCache[Config]

func loadConfig(meta *meta.Meta) (Config, error) {
	return channelConfigCache.Load(meta, Config{})
}
