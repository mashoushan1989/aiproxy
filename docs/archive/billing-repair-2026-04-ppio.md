# PPIO 2026-04 Billing Investigation Note

## Verified Findings

- The discrepancy is real in local billing data. It is not mainly caused by Novita traffic or by a small amount of direct-key traffic.
- The official PPIO workbook total for `2026-04-01` to `2026-04-24` is `831,882.0178`.
- The same upstream key inside AI Proxy local PPIO channels `3/4/5` sums to `943,337.7720`.
- The delta is `+111,455.7543` (`+13.4%`).
- Almost the whole month-level delta is concentrated in `2026-04-01` to `2026-04-07`.
- Production already contains the runtime fixes from:
  - `d90f521`
  - `50f9b70`
  - `4314dc7`
- `logs`, `summaries`, and `summary_minutes` are not a pristine historical truth set for this window.

## Root-Cause Judgment

There are two separate problem classes in the April history:

1. a historical usage / rewrite problem concentrated in `2026-04-01` to `2026-04-07`
2. a pricing-path problem around Claude cache pricing before the runtime fix landed

The workbook is still the best external truth source for reconciliation, but it is not fine-grained enough to justify rewriting production facts safely.

## Current Decision

We are **not** performing a historical data repair on production.

Specifically, we are not proceeding with:

- backfilling `logs`
- rebuilding summary tables
- resyncing accumulators
- resetting quota/cache state for this incident

## What This Document Is For Now

This note remains useful as:

- a factual investigation summary
- a reconciliation reference against the PPIO workbook
- context for finance, support, and internal review

It is no longer a production repair plan.

## Supported Read-Only Workflow

Use [`scripts/ppio_billing_reconcile.py`](/Users/ash/Documents/GitHub/aiproxy/scripts/ppio_billing_reconcile.py) to normalize the workbook into `day + model` truth and compare it with local aggregates.

Example:

```bash
python3 scripts/ppio_billing_reconcile.py \
  --workbook "/Users/ash/Downloads/API_Key-账单 (16).xlsx" \
  --local-csv "/Users/ash/Documents/GitHub/aiproxy/tmp_ppio_logs_apr.csv" \
  --start 2026-04-01 \
  --end 2026-04-24 \
  --output-dir tmp/ppio_reconcile \
  --emit-sql tmp/ppio_reconcile/ppio_truth.sql
```

This produces read-only reconciliation artifacts:

- `tmp/ppio_reconcile/ppio_truth_by_day_model.csv`
- `tmp/ppio_reconcile/local_by_day_model.csv`
- `tmp/ppio_reconcile/ppio_vs_local_by_day_model.csv`
- `tmp/ppio_reconcile/ppio_truth.sql`

`ppio_truth.sql` only loads workbook truth into a temp table. It does not mutate production data by itself.

## Recommended Operational Use

The findings from this investigation should be used to:

1. explain the discrepancy internally
2. support manual financial reconciliation
3. identify which users / groups were most affected in reporting
4. guide future prevention work in runtime billing logic

They should not be used to justify a direct rewrite of historical production billing rows without a new explicit decision.
