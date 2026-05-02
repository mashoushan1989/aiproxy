# Cache Follow Plugin

`cachefollow` is an opt-in plugin that improves upstream prompt-cache locality by
remembering the channel that previously produced cache-related usage for the same
model scope.

## Local Merge Rules

- Disabled by default. A model must set `plugin.cachefollow.enable=true`.
- Channel preference is advisory only. The selector still applies available-set,
  adaptor support, native-mode preference, banned-channel, and error-rate checks.
- Preferred channels are only considered inside the currently selected set. A
  remembered default-set channel cannot pull an overseas request out of its
  primary set.
- Passthrough fallback remains unchanged. A preferred passthrough channel is only
  usable when it is already in the current passthrough candidate set.
- Generic follow is disabled by default and requires
  `enable_generic_follow=true`.

## Configuration

```json
{
  "plugin": {
    "cachefollow": {
      "enable": true,
      "followed_channel_ttl_seconds": 180,
      "recent_channel_update_debounce_seconds": 30
    }
  }
}
```

`prompt_cache_key` mappings are used for `responses` and `chat.completions`.
`user` mappings are used for `responses`, `chat.completions`, `gemini`, and
`anthropic`.
