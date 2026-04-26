# PPIO 2026-04 Billing Repair Impact Assessment

## Scope

This note evaluates what the April 2026 PPIO billing repair will change inside AI Proxy, what is and is not one-to-one across data layers, the main remaining bugs/risks, and what users are most likely to notice in April reports.

This assessment is based on:

- workbook truth from `tmp/ppio_reconcile/ppio_truth.sql`
- repair scope from `tmp/ppio_repair_scope/phase_a_repair_scope.sql`
- dry-run preview from `tmp/ppio_billing_repair_runs/billing_fix_202604_ppio_phase_a_*`
- production code paths under `core/model` and `core/enterprise`

## Executive Summary

- The repair changes AI Proxy's internal billing facts and rollups. It does not change PPIO's upstream real charge.
- `logs`, `group_summaries`, token/group quota accumulators, and channel accumulators are not the same layer and are not all automatically kept in sync by a historical backfill.
- `logs.token_id -> tokens.id`, `logs.group_id -> groups.id`, and `logs.channel_id -> channels.id` are stable mappings.
- Workbook truth is only `day + model` granularity. It is not one-to-one with individual requests, tokens, or groups.
- `group_summaries` is keyed by `group_id + token_name + model + hour_timestamp`, not by `token_id`, and it has no `channel_id`. This makes user-facing reports correctable in total, but not reconstructable back to an exact upstream channel split.
- April 2026 user-visible distortion is concentrated in `2026-04-01` through `2026-04-07`. From `2026-04-08` onward, local-vs-PPIO drift is already down to about `+1.13%`.

## What The Repair Actually Changes

### 1. Upstream billing vs internal AI Proxy billing

The repair is a reconciliation against the upstream PPIO workbook. It does not change what PPIO billed. It changes AI Proxy's internal interpretation of that usage.

Practical impact:

- PPIO upstream statement: unchanged
- AI Proxy `logs.used_amount` and token fields: repaired
- enterprise custom report / quota views: only repaired if `group_summaries` are rebuilt
- token total quota and group used amount: only repaired if DB accumulators and Redis caches are synchronized
- channel totals in admin views: only repaired if `channels.used_amount` is explicitly recomputed

### 2. Which product surfaces depend on which table

Enterprise custom report reads `GroupSummary`, not raw `logs`:

- `core/enterprise/analytics/custom_report.go:714`

User quota status also computes period usage from `group_summaries`:

- `core/enterprise/access_info.go:756`

Token validation and total quota checks use token cache / token used amount:

- `core/model/token.go:434`

So a logs-only repair is insufficient for user-facing correctness.

## One-to-One Mapping: What Is Stable And What Is Not

### Stable one-to-one identifiers

These are stable entity mappings and can be reconciled deterministically:

- `logs.token_id -> tokens.id`
- `logs.group_id -> groups.id`
- `logs.channel_id -> channels.id`

### Not one-to-one

These are not stable one-to-one and need special handling:

- workbook truth -> `logs`
  - workbook only provides `day + model` truth
  - per-request / per-group / per-token corrections are inferred, not directly observed
- `group_summaries.token_name` -> token entity
  - `GroupSummaryUnique` is `group_id + token_name + model + hour_timestamp`
  - there is no `token_id` in `group_summaries`
  - token names can change later via `UpdateTokenName`
- `group_summaries` -> upstream channel split
  - `group_summaries` has no `channel_id`
  - once data is rolled up there, PPIO-only usage cannot be perfectly separated back out by channel

Code evidence:

- `core/model/groupsummary.go:19`
- `core/model/summary.go:335`
- `core/model/token.go:1211`

## Current April 2026 Drift

### Whole-month drift

For `2026-04-01` through `2026-04-24`:

- PPIO workbook: `831,882.0178`
- AI Proxy local logs on PPIO channels `3/4/5`: `943,337.7720`
- delta: `+111,455.7543`
- drift: `+13.4%`

### Where the distortion is concentrated

For `2026-04-01` through `2026-04-07`:

- workbook: `115,167.1040`
- local: `218,509.3863`
- delta: `+103,342.2823`

For `2026-04-08` through `2026-04-24`:

- workbook: `716,714.9138`
- local: `724,828.3857`
- delta: `+8,113.4719`
- drift: `+1.13%`

Interpretation:

- the major accounting distortion is an early-April historical issue
- after `2026-04-08`, the collection path is already close enough to normal reconciliation tolerance

### Biggest day-model anomalies in the repair scope

From the dry-run summary:

- `2026-04-07 pa/claude-opus-4-6`: `12,976.1 -> 50,503.3`, delta `+37,527.2`
- `2026-04-07 pa/claude-sonnet-4-6`: `10,214.1 -> 40,164.9`, delta `+29,950.8`
- `2026-04-03 pa/claude-sonnet-4-6`: delta `+3,500.6`
- `2026-04-05 pa/claude-sonnet-4-6`: delta `+3,397.9`
- `2026-04-02 pa/claude-opus-4-6`: delta `+3,008.5`

These remain the strongest evidence that the main anomaly window is not random user noise.

## What Users Will Notice

### 1. Most user-facing correction will feel like April usage going down

Because the anomaly is mostly overcount, the repair will reduce April spend for the groups and token labels most active in the affected PPIO models during `2026-04-01` through `2026-04-07`.

Phase A dry-run top affected groups:

| group_name | current | proposed | delta |
| --- | ---: | ---: | ---: |
| 文也 | 18128.9160 | 9199.1418 | -8929.7742 |
| 德里克 | 11928.2272 | 5693.5046 | -6234.7226 |
| 鸣人 | 10180.9346 | 4429.2180 | -5751.7167 |
| 卡妙 | 13336.9941 | 8424.9954 | -4911.9987 |
| 镜悬 | 9559.1110 | 4973.0000 | -4586.1110 |
| 温莉 | 7252.0903 | 2982.6849 | -4269.4054 |
| 藤丸 | 7263.6435 | 3404.2737 | -3859.3699 |
| 白小飞 | 5365.4802 | 1535.7463 | -3829.7338 |
| 兆南 | 6097.4201 | 2462.4955 | -3634.9246 |
| 麦迪 | 3880.2674 | 996.5023 | -2883.7651 |

### 2. Token-name views and token-entity views are close, but not identical

Phase A dry-run top affected token labels:

| token_name | current | proposed | delta |
| --- | ---: | ---: | ---: |
| 文也 | 18128.9160 | 9199.1418 | -8929.7742 |
| claude code | 11653.1853 | 4827.6962 | -6825.4890 |
| 德里克 | 11928.2272 | 5693.5046 | -6234.7226 |
| claude-code | 13126.1245 | 7312.6892 | -5813.4354 |
| aa | 5365.4802 | 1535.7463 | -3829.7338 |
| 兆南 | 6097.4201 | 2462.4955 | -3634.9246 |
| Agent团队 | 5007.4070 | 1873.0751 | -3134.3319 |
| 藤丸 | 5001.3589 | 1951.7405 | -3049.6184 |
| 麦迪 | 3880.2674 | 996.5023 | -2883.7651 |
| book | 6422.1218 | 3833.3884 | -2588.7334 |

Important nuance:

- these are token labels from `logs.token_name`
- they are not guaranteed to be a stable token entity key forever
- if a token was renamed, historical report rows remain on the old name

### 3. Users are unlikely to feel large corrections after April 8

Because `2026-04-08` through `2026-04-24` is only `+1.13%` high, users are unlikely to perceive a dramatic difference in that later window unless they look at very fine-grained model/day breakdowns.

## Current Accumulator Split: Logs vs Tokens vs Groups vs Channels

### Token accumulators

All-time token accumulators are close to `logs` overall. Largest sampled drift:

- token `146`: `token_used 31254.4636` vs `log_used 30833.0179`, delta `+421.4457`

This suggests token totals are not the primary broken layer right now.

### Group accumulators

All-time group accumulators are also close to `logs`. Largest sampled drift:

- group `feishu_ou_b67111220b894e1ed6833cd39739cbfd`: `group_used 31254.4636` vs `log_used 30833.0179`, delta `+421.4457`

This suggests group totals are mostly aligned historically, but still need explicit sync after a downward repair.

### Channel accumulators

Channel accumulators are not reliable as a repair truth source.

Largest sampled drift:

- channel `4` `PPIO (Anthropic)`: `channel_used 642693.8131` vs `log_used 718428.6068`, delta `-75,734.7937`
- channel `3` `PPIO (OpenAI)`: delta `-268.1298`

This is strong evidence that `channels.used_amount` must be recomputed explicitly during the formal repair.

## Known Risks And Potential Bugs

### 1. Downward repair will not automatically fix Redis quota caches

Both token and group cache update scripts only allow increases:

- `core/model/cache.go:266`
- `core/model/cache.go:549`

That means:

- if repaired usage goes down
- Redis may still hold the old higher `used_amount`
- users can keep seeing stale quota exhaustion or stale remaining-balance state

This is a real bug unless repair explicitly resets or resynchronizes those cache keys.

### 2. Pure proportional scaling is not enough

The dry-run found `180` scoped `day + model` rows with no matching local log rows.

That means:

- some workbook truth buckets exist with no current local rows
- a logs-only rescale of existing rows cannot close the full truth gap
- formal `apply` needs a policy for truth-only buckets

### 3. Rebuilding `group_summaries` cannot restore exact channel provenance

Because `GroupSummaryUnique` has no `channel_id`, a repaired group summary can make user totals correct while still not preserving the exact upstream channel split.

This is acceptable if the goal is billing/report correctness, but it must be stated clearly as a limit.

### 4. Token rename can split historical report continuity

`UpdateTokenName` exists, and `group_summaries` keys on `token_name`, not `token_id`.

So two things can both be true:

- token entity totals are correct
- token-name report rows remain fragmented across old and new labels

This is not introduced by the repair, but it affects how results should be explained and validated.

### 5. Non-billing rollup fields should not be blindly rewritten

Fields like:

- `request_count`
- `status_xxx_count`
- latency-derived fields
- retry counters

do not have workbook truth support.

Code paths show these are derived independently during rollup:

- `core/model/batch.go:869`
- `core/model/batch.go:879`
- `core/model/batch.go:897`
- `core/model/batch.go:1030`
- `core/model/batch.go:1036`
- `core/model/batch.go:1053`

Best practice for the repair is:

- correct billing fields and token counters from truth-constrained logic
- preserve or rebuild operational counters only from trusted local request facts
- never invent request counts from the workbook

## Repair Implications

To make the repair user-correct, not just reconciliation-correct, the formal run should cover four layers:

1. `logs`
   - repair `used_amount` and billing-related token fields inside the scoped PPIO history
2. `group_summaries` and `group_summary_minutes`
   - rebuild user-facing report and quota aggregates from repaired facts
3. `tokens.used_amount`, `groups.used_amount`, and related Redis caches
   - synchronize accumulators so total quota checks match repaired billing
4. `channels.used_amount`
   - explicitly recompute channel totals; do not trust historical accumulator state

If any of the four is skipped, a bad intermediate state is likely:

- reconciliation looks fixed
- but quota remains stale
- or user report remains stale
- or admin channel usage still shows a different number

## Recommended Validation Matrix

After `apply`, the minimum validation set should be:

1. `logs` vs workbook
   - exact match on scoped `day + model`
2. `group_summaries` vs repaired `logs`
   - match on `group_id + token_name + model + hour`
3. token/group accumulators vs repaired `logs`
   - totals must match after cache sync
4. channels vs repaired `logs`
   - totals must match after recompute
5. user spot checks
   - spot-check top affected groups from the table above

## Bottom Line

The repair is justified and will mostly be perceived as an April usage decrease for a limited set of heavy PPIO users in the first week of the month.

The main implementation risk is not the truth source anymore. The main implementation risk is partial repair:

- fixing `logs` without `group_summaries`
- fixing DB without Redis cache sync
- fixing user-facing totals without channel recompute

As of now, the safest formal approach remains:

- truth-constrained historical repair on `logs`
- deterministic rebuild of user-facing summary tables
- explicit synchronization of token/group accumulators and Redis caches
- explicit recomputation of channel accumulators
