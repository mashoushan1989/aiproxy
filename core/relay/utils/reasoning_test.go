package utils_test

import (
	"testing"

	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
	"github.com/stretchr/testify/assert"
)

func TestParseGeminiReasoningEmptyConfigIsUnspecified(t *testing.T) {
	t.Parallel()

	reasoning := utils.ParseGeminiReasoning(&relaymodel.GeminiThinkingConfig{})

	assert.False(t, reasoning.Specified)
}

func TestParseGeminiReasoningBudgetToEffort(t *testing.T) {
	t.Parallel()

	budget := 4097
	reasoning := utils.ParseGeminiReasoning(&relaymodel.GeminiThinkingConfig{
		ThinkingBudget: &budget,
	})

	assert.True(t, reasoning.Specified)
	assert.Equal(t, relaymodel.ReasoningEffortMedium, reasoning.Effort)
}

func TestParseClaudeReasoningDisabled(t *testing.T) {
	t.Parallel()

	reasoning := utils.ParseClaudeReasoning(&relaymodel.ClaudeThinking{
		Type: relaymodel.ClaudeThinkingTypeDisabled,
	})

	assert.True(t, reasoning.Specified)
	assert.True(t, reasoning.Disabled)
	assert.Equal(t, relaymodel.ReasoningEffortNone, utils.ReasoningToOpenAIEffort(reasoning))
}
