# PPIO Billing Repair Plan (2026-04)

## Verified Findings

- The discrepancy is real in local billing data. It is not mainly caused by Novita traffic or by a small amount of direct-key traffic.
- The official PPIO workbook total for `2026-04-01` to `2026-04-24` is `831,882.0178`.
- The same upstream key inside AI Proxy local PPIO channels `3/4/5` sums to `943,337.7720`.
- The delta is `+111,455.7543` (`+13.4%`).
- Almost the whole month-level delta is concentrated in `2026-04-01` to `2026-04-07`.
- Production already contains the runtime fixes from:
  - `d90f521` `fix(passthrough): Anthropic input_tokens ...`
  - `50f9b70` `fix(passthrough): TotalTokens ...`
  - `4314dc7` `fix(ppio): 移除 SupportPromptCache 门控 ...`
- The current production values are not pristine history. `_fix_round3_backup` and `_fix_round4_backup` prove that parts of the affected rows were rewritten after the original values were backed up.
- `round3` / `round4` are useful for forensics, but neither round can be treated as a global restore source:
  - On `2026-04-07`, `round3` is much closer than current for `pa/claude-opus-4-6` and `pa/claude-sonnet-4-6`.
  - On `2026-04-05` and `2026-04-06`, current is often closer than `round3`.
  - A naive `coalesce(round4, round3, current)` mix is better than current but still materially wrong.
- `logs`, `summaries`, and `summary_minutes` are already divergent. Any repair must account for summary tables instead of assuming they will self-heal.

## Root-Cause Judgment

There are two separate problems:

1. `2026-04-01` to `2026-04-07` has a historical rewrite problem on Claude-heavy rows.
   The strongest evidence is the gap between current `logs` and `_fix_round3_backup` / `_fix_round4_backup`, especially on `2026-04-07`.

2. There is also a pricing-path issue around Claude cache pricing.
   The `4314dc7` fix explains why cache-related price fields could be wrong before the sync fix landed.

The important conclusion is that backup tables explain the history, but the official PPIO workbook should be treated as the repair truth for the affected month.

## Safe Repair Strategy

1. Use the PPIO workbook as the source of truth at `day + model`.
2. Do not blindly restore `_fix_round3_backup` or `_fix_round4_backup`.
3. Back up all affected rows before any mutation:
   - `logs`
   - `summaries`
   - `summary_minutes`
   - `group_summaries`
   - `group_summary_minutes`
4. Repair `logs` only inside the affected scope:
   - channels `3/4/5`
   - time range `2026-04-01` to `2026-04-24`
   - models present in the workbook
5. Rebuild summary tables from repaired `logs`.
   This matters because enterprise custom reports read `group_summaries`, not raw `logs`.
6. Re-run the reconciliation after rebuild and verify that:
   - local `used_amount` matches workbook by `day + model`
   - `summaries` and `summary_minutes` match `logs`
   - `group_summaries` totals match rebuilt `logs`

## Why Workbook Truth Beats Backup Truth

- The workbook is the upstream bill for the exact PPIO key used by channels `3/4/5`.
- Backup tables only capture selected repair rounds and selected rows.
- The backup rounds are internally inconsistent across windows:
  - some buckets are closer before repair,
  - some are closer after repair,
  - some still need pricing-side correction either way.

That makes the workbook the only stable reconciliation target.

## Repo Support Added

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

This produces:

- `tmp/ppio_reconcile/ppio_truth_by_day_model.csv`
- `tmp/ppio_reconcile/local_by_day_model.csv`
- `tmp/ppio_reconcile/ppio_vs_local_by_day_model.csv`
- `tmp/ppio_reconcile/ppio_truth.sql`

`ppio_truth.sql` only loads workbook truth into a temp table. It does not mutate production data by itself.

## Recommended Next Step

Build a one-off repair SQL in three stages:

1. load `tmp_ppio_truth`
2. repair the affected `logs` rows inside the PPIO scope
3. rebuild `summaries`, `summary_minutes`, `group_summaries`, and `group_summary_minutes` from repaired `logs`

That approach is safer than direct summary-table patching because it preserves the user and group distribution that enterprise reports depend on.
