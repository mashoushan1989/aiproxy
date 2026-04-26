# PPIO Billing Repair Executable Design

## Goal

Repair the confirmed historical billing corruption for the PPIO upstream key while preserving downstream enterprise analytics semantics.

The repair must be:

- auditable
- rollbackable
- idempotent
- minimal in blast radius
- consistent across `logs`, `summaries`, `summary_minutes`, `group_summaries`, and `group_summary_minutes`

## Why This Design

The upstream workbook is authoritative only at `day + model`.
It is **not** authoritative at:

- request level
- `group_id` level
- `token_name` level
- minute / hour bucket level

So the safest formal repair is:

1. recompute request-level billing where we still have trustworthy per-request facts
2. treat workbook as aggregate truth only for the unresolved residual
3. repair only billing-related fields in `logs`
4. rebuild only the billing-related fields in summary tables from repaired `logs`
5. leave non-billing fields alone unless we can reconstruct them exactly

This is safer than:

- directly forcing workbook totals into `group_summaries`
- blindly restoring `_fix_round3_backup` or `_fix_round4_backup`
- fully rebuilding every summary field from scratch without proving semantics parity
- blindly scaling every scoped log row even when a subset can be deterministically recomputed

## Confirmed Constraints From Code

- Enterprise custom reports read `GroupSummary`, not raw `logs`:
  [custom_report.go](/Users/ash/Documents/GitHub/aiproxy/core/enterprise/analytics/custom_report.go:714)
- `group_summaries` are keyed by `group_id + token_name + model + hour_timestamp`:
  [batch.go](/Users/ash/Documents/GitHub/aiproxy/core/model/batch.go:854)
- `summaries` are keyed by `channel_id + model + hour_timestamp`:
  [batch.go](/Users/ash/Documents/GitHub/aiproxy/core/model/batch.go:1016)
- Summary cache-hit counters are derived from token presence, not stored independently:
  [batch.go](/Users/ash/Documents/GitHub/aiproxy/core/model/batch.go:897)
  [batch.go](/Users/ash/Documents/GitHub/aiproxy/core/model/batch.go:1053)
- Latency metrics are derived from `created_at - request_at` and `firstByteAt - request_at`:
  [batch.go](/Users/ash/Documents/GitHub/aiproxy/core/model/batch.go:1139)

## Repair Scope Policy

Use a staged scope instead of mutating the whole month in one pass.

### Phase A: mandatory

Repair only the clearly-corrupted window:

- date range: `2026-04-01` to `2026-04-07`
- channels: `3, 4, 5`
- rows selected from workbook-vs-local diff by threshold

Recommended threshold:

- `abs(delta_used_amount) >= 500`
  or
- `abs(ratio_used_amount - 1.0) >= 0.05`

Generate that scope with:

```bash
python3 scripts/ppio_billing_repair_scope.py \
  --diff-csv tmp/ppio_reconcile/ppio_vs_local_by_day_model.csv \
  --start 2026-04-01 \
  --end 2026-04-07 \
  --amount-threshold 500 \
  --ratio-threshold 0.05 \
  --output-dir tmp/ppio_repair_scope \
  --label phase_a
```

### Phase B: optional

Only if Phase A lands cleanly and business still requires tighter reconciliation:

- date range: `2026-04-08` to `2026-04-24`
- same threshold-based selection

Do **not** include Phase B in the first production repair.

## Source Inputs

### Workbook truth

Generate normalized truth:

```bash
python3 scripts/ppio_billing_reconcile.py \
  --workbook "/Users/ash/Downloads/API_Key-账单 (16).xlsx" \
  --local-csv tmp_ppio_logs_apr.csv \
  --start 2026-04-01 \
  --end 2026-04-24 \
  --output-dir tmp/ppio_reconcile \
  --emit-sql tmp/ppio_reconcile/ppio_truth.sql
```

This produces:

- `tmp/ppio_reconcile/ppio_truth_by_day_model.csv`
- `tmp/ppio_reconcile/ppio_vs_local_by_day_model.csv`
- `tmp/ppio_reconcile/ppio_truth.sql`

### Repair scope

Generate a minimal scope:

```bash
python3 scripts/ppio_billing_repair_scope.py \
  --diff-csv tmp/ppio_reconcile/ppio_vs_local_by_day_model.csv \
  --start 2026-04-01 \
  --end 2026-04-07 \
  --amount-threshold 500 \
  --ratio-threshold 0.05 \
  --output-dir tmp/ppio_repair_scope \
  --label phase_a
```

This produces:

- `tmp/ppio_repair_scope/phase_a_repair_scope.csv`
- `tmp/ppio_repair_scope/phase_a_repair_scope.sql`

### Request-level evidence

Request-level recompute is only possible when enough original facts still exist.

For the current Phase A window on production:

- scoped logs: `126,719`
- rows with `request_detail`: `3,110`
- rows with non-empty `request_body`: `3,108`
- rows with non-empty `response_body`: `2,790`
- rows with truncated `request_body`: `2,448`

Interpretation:

- request replay is useful, but it cannot be the only repair path
- the majority of affected rows do not retain enough raw detail for full replay
- the formal repair must therefore be hybrid by design

## Production Execution Model

Run in three explicit modes:

1. `dry-run`
2. `apply`
3. `verify`

Never combine them into one opaque script execution.

## Data Model Strategy

### Tables to back up

Before any mutation, snapshot the exact affected slice into run-scoped backup tables:

- `logs`
- `summaries`
- `summary_minutes`
- `group_summaries`
- `group_summary_minutes`

Recommended naming:

- `_billing_fix_202604_ppio_logs_backup`
- `_billing_fix_202604_ppio_summaries_backup`
- `_billing_fix_202604_ppio_summary_minutes_backup`
- `_billing_fix_202604_ppio_group_summaries_backup`
- `_billing_fix_202604_ppio_group_summary_minutes_backup`

## What To Repair In `logs`

Repair only these billing fields:

- `input_tokens`
- `cached_tokens`
- `cache_creation_tokens`
- `output_tokens`
- `total_tokens`
- `input_amount`
- `cached_amount`
- `cache_creation_amount`
- `output_amount`
- `used_amount`

Preserve these fields:

- `request_id`
- `upstream_id`
- `group_id`
- `token_name`
- `channel_id`
- `code`
- `created_at`
- `request_at`
- `retry_at`
- `ttfb_milliseconds`
- `retry_times`
- `service_tier`
- `user`
- `metadata`

Leave these price fields unchanged in Phase A:

- `input_price`
- `cached_price`
- `cache_creation_price`
- `output_price`
- `conditional_prices`

Reason:
the workbook is aggregate truth, not per-request unit-price truth. Forcing per-row prices from aggregate data would create false precision.

## Hybrid Log Repair Algorithm

For each `(day, model)` inside `tmp_ppio_repair_scope`:

1. classify scoped log rows into:
   - `replayable`
   - `non_replayable`
2. recompute billing for `replayable` rows using the corrected request-level logic
3. aggregate replayed subtotals at `(day, model)`
4. load workbook truth from `tmp_ppio_truth`
5. compute residual truth:
   - `truth - replayed_subtotal`
6. distribute only the residual back onto `non_replayable` scoped rows
7. combine:
   - locked replayed rows
   - residual-adjusted non-replayable rows
8. verify the final `(day, model)` aggregate exactly matches workbook truth within tolerance

### Replayable row rule

A row is `replayable` only if all of the following hold:

- its issue type is one we can deterministically fix from request-level facts
- the row has sufficient original fields to recompute the affected billing buckets
- recomputation does not require inventing missing upstream usage

Practical Phase A rule:

- pricing-formula corrections are replayable when the row already has trustworthy token buckets in `logs`
- raw-body replay is optional and secondary because detail coverage is too low
- rows whose stored token usage is suspected to be corrupted remain `non_replayable`

### Locked-row rule

Replayed rows are locked after request-level recompute.

Workbook residual allocation must never overwrite them.

This is the main guardrail that keeps request-level truth and workbook truth from conflicting.

### Conflict rule

For every `(day, model, metric)`:

- if `replayed_subtotal <= workbook_truth + tolerance`, proceed
- if `replayed_subtotal > workbook_truth + tolerance`, fail the repair for that bucket and require manual review

Never use workbook residual scaling to push down a deterministically recomputed replayed subtotal.

Recommended tolerance:

- token metrics: exact integer match after allocation
- amount metrics: `<= 0.0001`

### Residual pool rule

Workbook residual is allocated only across `non_replayable` rows.

If a `(day, model)` bucket has:

- non-zero residual
- zero eligible non-replayable rows

then the repair must fail closed for that bucket unless an explicit synthetic adjustment policy is enabled.

This prevents silent leakage of workbook truth into the wrong request rows.

### Token distribution rule

Use largest-remainder allocation for exact integer preservation:

- compute raw proportional value per row
- take `floor(raw_value)`
- compute residual at `(day, model, metric)` level
- give `+1` to rows with the largest fractional remainder until the residual is exhausted

Do this independently across the `non_replayable` residual pool for:

- `uncached_tokens`
- `cached_tokens`
- `cache_creation_tokens`
- `output_tokens`

Then set:

- `input_tokens = uncached + cached + cache_creation`
- `total_tokens = input_tokens + output_tokens`

### Amount distribution rule

Use proportional numeric scaling per row and keep sufficient precision.

Do this independently across the `non_replayable` residual pool for:

- `input_amount`
- `cached_amount`
- `cache_creation_amount`
- `output_amount`

Then set:

- `used_amount = input_amount + cached_amount + cache_creation_amount + output_amount + preserved_other_amounts`

### Two concrete bug classes

The hybrid algorithm should treat the currently confirmed bug classes differently:

1. pricing-only bug
   - example: Claude cache pricing mismatch
   - approach: request-level recompute first
   - workbook only reconciles the residual after replay

2. usage-semantics bug
   - example: Anthropic input/cache token normalization mismatch written into `logs`
   - approach: do not trust request-level recompute unless raw per-request facts are sufficient
   - workbook remains the effective truth source for unresolved rows

This distinction is critical. Recomputing amount from already-corrupted usage does not recover truth.

`preserved_other_amounts` means:

- `image_input_amount`
- `audio_input_amount`
- `image_output_amount`
- `web_search_amount`
- `reasoning_amount`

Those remain untouched in Phase A.

## Why Not Full Summary Rebuild

A full rebuild of all summary fields from `logs` is possible, but it is **not** the best first production repair because:

- retry semantics differ between paths
- some summary counters were intentionally derived with path-specific logic
- this incident is about billing corruption, not request-count corruption

So the first formal repair should be:

- full repair of billing fields in `logs`
- targeted rebuild of billing-related fields in summary tables

## Summary Rebuild Strategy

Recompute only these summary fields from repaired `logs`:

### Usage fields

- `input_tokens`
- `output_tokens`
- `total_tokens`
- `cached_tokens`
- `cache_creation_tokens`
- `reasoning_tokens`
- `image_input_tokens`
- `audio_input_tokens`
- `image_output_tokens`
- `web_search_count`

### Amount fields

- `input_amount`
- `output_amount`
- `cached_amount`
- `cache_creation_amount`
- `reasoning_amount`
- `image_input_amount`
- `audio_input_amount`
- `image_output_amount`
- `web_search_amount`
- `used_amount`

### Derived billing counters

- `cache_hit_count`
- `cache_creation_count`

Rebuild the same field families for:

- base rows
- `service_tier_flex_*`
- `service_tier_priority_*`
- `claude_long_context_*`

Preserve these existing summary fields:

- `request_count`
- `retry_count`
- `exception_count`
- `status2xx_count`
- `status4xx_count`
- `status5xx_count`
- `status_other_count`
- `status400_count`
- `status429_count`
- `status500_count`
- `total_time_milliseconds`
- `total_ttfb_milliseconds`

## Rebuild Order

Use this order:

1. repair `logs`
2. rebuild `summary_minutes`
3. roll up to `summaries`
4. rebuild `group_summary_minutes`
5. roll up to `group_summaries`

This is safer than rebuilding hourly and minute tables independently because it gives one canonical minute-level source.

## Dry-Run Output Requirements

The dry-run must emit:

1. scope size
2. affected `(day, model)` list
3. pre-repair aggregate totals
4. target workbook totals
5. replayable vs non-replayable row counts
6. replayed subtotal by `(day, model)`
7. residual to allocate by `(day, model)`
8. planned post-repair totals
9. top 50 largest per-log deltas
7. invariant checks:
   - no negative amounts
   - `input_tokens >= cached_tokens + cache_creation_tokens`
   - `total_tokens = input_tokens + output_tokens`
   - `used_amount >= 0`
8. conflict checks:
   - no bucket where replayed subtotal exceeds workbook truth beyond tolerance
   - no non-zero residual bucket without eligible residual rows

## Apply Mode Requirements

Apply mode must:

1. refuse to run if backup tables already exist and no explicit `--force-run-id` is provided
2. run inside a transaction per table-mutation stage
3. write repair audit rows:
   - run id
   - operator
   - started at
   - completed at
   - scope row count
   - changed log row count
   - replayed row count
   - residual-adjusted row count
4. store per-log before/after deltas in an audit table

Recommended audit table:

- `_billing_fix_202604_ppio_log_delta_audit`

Recommended additional bucket audit table:

- `_billing_fix_202604_ppio_bucket_audit`

Per bucket audit should include:

- day
- model
- workbook truth metrics
- replayed subtotal metrics
- residual metrics
- final metrics
- status (`ok` / `conflict` / `no_residual_pool`)

## Verify Mode Requirements

Verify mode must prove all of the following:

### Aggregate checks

- `logs` match workbook within repair scope at `day + model`
- replayed rows retain their request-level recomputed values after final apply
- `summaries` usage/amount fields match `logs`
- `summary_minutes` usage/amount fields match `logs`
- `group_summaries` usage/amount fields match rebuilt `group_summary_minutes`

### Invariant checks

- no negative `used_amount`
- no negative bucket amounts
- no negative token fields
- `input_tokens = uncached + cached + cache_creation` at aggregated scope
- `total_tokens = input_tokens + output_tokens`

### Business checks

- enterprise custom report totals inside the repaired window align with rebuilt `group_summaries`
- post-repair residual for Phase A should be near zero at `day + model`
- no repaired bucket required workbook to override a locked replayed subtotal

## Rollback Plan

Rollback is table-copy based, not inverse-delta based.

If verification fails:

1. restore the scoped rows from backup tables
2. restore summary tables from their backups
3. rerun verification

Do not attempt arithmetic reverse updates.

## Recommended Production Rollout

1. run `dry-run` on domestic production DB replica or a sanitized clone
2. review top per-log deltas
3. apply Phase A only
4. verify
5. observe for one reporting cycle
6. decide whether Phase B is worth doing

## Immediate Next Implementation Step

Implement a one-off repair utility with this interface:

```bash
scripts/ppio_billing_repair \
  --db-url "$AIPROXY_DSN" \
  --truth-sql tmp/ppio_reconcile/ppio_truth.sql \
  --scope-sql tmp/ppio_repair_scope/phase_a_repair_scope.sql \
  --run-id billing_fix_202604_ppio_phase_a \
  dry-run
```

Current status:

- `dry-run` is implemented in [`scripts/ppio_billing_repair`](/Users/ash/Documents/GitHub/aiproxy/scripts/ppio_billing_repair)
- current `dry-run` is still workbook-proportional preview only and should be upgraded to the hybrid algorithm above before `apply`
- `apply`, `verify`, and `rollback` are intentionally still pending
- for remote execution without a local `psql`, use `--psql-cmd 'ssh ... docker exec -i ... psql ...'`

Supported subcommands:

- `dry-run`
- `apply`
- `verify`
- `rollback`

That utility should own all SQL generation and execution, instead of relying on hand-edited ad hoc SQL in production.
