package ppio

import "testing"

func TestModelListIncludesDeepSeekV4Pro(t *testing.T) {
	for _, modelConfig := range ModelList {
		if modelConfig.Model == "deepseek/deepseek-v4-pro" {
			return
		}
	}

	t.Fatal("ModelList does not include deepseek/deepseek-v4-pro")
}
