# Async Usage Local Design

## Goal

Add delayed upstream usage reconciliation without changing model routing or passthrough behavior. Only adaptors that explicitly return `DoResponseResult.AsyncUsage` are polled later.

## Scope

- OpenAI and Azure Responses/video job usage polling.
- Log row status tracking through `async_usage_status`.
- Deferred balance consumption and summary/log usage update when usage becomes available.
- Global background task gate controls the poller together with other global schedulers.

## Non-goals

- No PPIO/Novita passthrough polling.
- No global `SupportMode` contract changes.
- No protocol conversion changes for pure passthrough channels.
- No automatic async usage for every OpenAI-compatible provider.

## Follow-up

Converted Chat/Anthropic/Gemini requests that internally use Responses can be enabled later per adaptor after validating response IDs and upstream fetch URLs.
