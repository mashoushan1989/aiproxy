package utils

import relaymodel "github.com/labring/aiproxy/core/relay/model"

func ParseOpenAIReasoning(req *relaymodel.GeneralOpenAIRequest) relaymodel.NormalizedReasoning {
	if req == nil || req.ReasoningEffort == nil {
		return relaymodel.NormalizedReasoning{}
	}

	effort := relaymodel.NormalizeReasoningEffort(*req.ReasoningEffort)
	if effort == "" {
		return relaymodel.NormalizedReasoning{}
	}

	return reasoningFromEffort(effort)
}

func ParseGeminiReasoning(
	config *relaymodel.GeminiThinkingConfig,
) relaymodel.NormalizedReasoning {
	if config == nil {
		return relaymodel.NormalizedReasoning{}
	}

	if config.ThinkingLevel != "" {
		effort := relaymodel.NormalizeReasoningEffort(config.ThinkingLevel)
		if effort != "" {
			return reasoningFromEffort(effort)
		}
	}

	if config.ThinkingBudget != nil {
		if *config.ThinkingBudget <= 0 {
			return relaymodel.NormalizedReasoning{
				Specified: true,
				Disabled:  true,
				Effort:    relaymodel.ReasoningEffortNone,
			}
		}

		budget := *config.ThinkingBudget

		return relaymodel.NormalizedReasoning{
			Specified:    true,
			Effort:       BudgetToEffort(budget),
			BudgetTokens: &budget,
		}
	}

	if config.IncludeThoughts {
		return relaymodel.NormalizedReasoning{
			Specified: true,
			Effort:    relaymodel.ReasoningEffortMedium,
		}
	}

	return relaymodel.NormalizedReasoning{}
}

func ParseClaudeReasoning(
	thinking *relaymodel.ClaudeThinking,
) relaymodel.NormalizedReasoning {
	if thinking == nil {
		return relaymodel.NormalizedReasoning{}
	}

	switch thinking.Type {
	case relaymodel.ClaudeThinkingTypeDisabled:
		return relaymodel.NormalizedReasoning{
			Specified: true,
			Disabled:  true,
			Effort:    relaymodel.ReasoningEffortNone,
		}
	case relaymodel.ClaudeThinkingTypeEnabled:
		reasoning := relaymodel.NormalizedReasoning{
			Specified: true,
		}

		if thinking.BudgetTokens > 0 {
			budget := thinking.BudgetTokens
			reasoning.BudgetTokens = &budget
			reasoning.Effort = BudgetToEffort(budget)
		}

		if reasoning.Effort == "" {
			reasoning.Effort = relaymodel.ReasoningEffortMedium
		}

		return reasoning
	default:
		return relaymodel.NormalizedReasoning{}
	}
}

func ApplyReasoningToOpenAIRequest(
	req *relaymodel.GeneralOpenAIRequest,
	reasoning relaymodel.NormalizedReasoning,
) {
	if req == nil || !reasoning.Specified {
		return
	}

	effort := ReasoningToOpenAIEffort(reasoning)
	if effort == "" {
		return
	}

	effortString := effort
	req.ReasoningEffort = &effortString
	req.Thinking = nil
}

func ApplyReasoningToResponsesRequest(
	req *relaymodel.CreateResponseRequest,
	reasoning relaymodel.NormalizedReasoning,
) {
	if req == nil || !reasoning.Specified {
		return
	}

	effort := ReasoningToOpenAIEffort(reasoning)
	if effort == "" {
		return
	}

	effortString := effort
	req.Reasoning = &relaymodel.ResponseReasoning{
		Effort: &effortString,
	}
}

func reasoningFromEffort(effort relaymodel.ReasoningEffort) relaymodel.NormalizedReasoning {
	return relaymodel.NormalizedReasoning{
		Specified: true,
		Disabled:  effort == relaymodel.ReasoningEffortNone,
		Effort:    effort,
	}
}

func ReasoningToOpenAIEffort(
	reasoning relaymodel.NormalizedReasoning,
) relaymodel.ReasoningEffort {
	if reasoning.Disabled {
		return ""
	}

	if reasoning.Effort != "" {
		return clampReasoningToOpenAIEffort(reasoning.Effort)
	}

	if reasoning.BudgetTokens != nil {
		return clampReasoningToOpenAIEffort(BudgetToEffort(*reasoning.BudgetTokens))
	}

	if reasoning.Specified {
		return relaymodel.ReasoningEffortMedium
	}

	return ""
}

func clampReasoningToOpenAIEffort(
	effort relaymodel.ReasoningEffort,
) relaymodel.ReasoningEffort {
	switch effort {
	case relaymodel.ReasoningEffortNone:
		return ""
	case relaymodel.ReasoningEffortMinimal:
		return relaymodel.ReasoningEffortLow
	case relaymodel.ReasoningEffortLow,
		relaymodel.ReasoningEffortMedium,
		relaymodel.ReasoningEffortHigh:
		return effort
	case relaymodel.ReasoningEffortXHigh:
		return relaymodel.ReasoningEffortHigh
	default:
		return ""
	}
}

func BudgetToEffort(budget int) relaymodel.ReasoningEffort {
	switch {
	case budget <= 0:
		return relaymodel.ReasoningEffortNone
	case budget <= 1024:
		return relaymodel.ReasoningEffortMinimal
	case budget <= 4096:
		return relaymodel.ReasoningEffortLow
	case budget <= 12288:
		return relaymodel.ReasoningEffortMedium
	case budget <= 24576:
		return relaymodel.ReasoningEffortHigh
	default:
		return relaymodel.ReasoningEffortXHigh
	}
}
