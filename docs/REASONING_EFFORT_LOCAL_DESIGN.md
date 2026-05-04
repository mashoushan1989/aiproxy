# Local Reasoning Effort Merge

This branch keeps only the low-risk reasoning-effort subset from upstream.

## Scope

- Chat Completions converted to OpenAI Responses maps `reasoning_effort` to
  `reasoning.effort`.
- Gemini requests converted through the OpenAI adaptor map explicit
  `generationConfig.thinkingConfig` to OpenAI `reasoning_effort` or Responses
  `reasoning.effort`.
- Claude requests converted through the OpenAI adaptor map explicit `thinking`
  to OpenAI `reasoning_effort` or Responses `reasoning.effort`.

## Boundaries

- Pure passthrough channels remain byte-level passthrough. PPIO/Novita request
  bodies are not parsed or rewritten by this branch.
- Ordinary OpenAI-compatible Chat Completions forwarding is not converted unless
  an existing conversion path already runs.
- Empty Gemini `thinkingConfig` is treated as unspecified, not as an instruction
  to disable reasoning.
- The broader upstream cross-provider mappings are intentionally left out until
  each provider strategy is reviewed independently.
