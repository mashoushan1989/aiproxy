# PPIO Billing Reconciliation Execution Note

## Current Decision

The historical April 2026 PPIO discrepancy will **not** be fixed by mutating production billing facts.

This means we are **not** proceeding with any of the following on production history:

- rewriting `logs`
- rebuilding `summaries`
- rebuilding `summary_minutes`
- rebuilding `group_summaries`
- rebuilding `group_summary_minutes`
- recalculating `tokens.used_amount`
- recalculating `groups.used_amount`
- recalculating `channels.used_amount`
- resetting Redis quota/cache state for this historical incident

## What Remains In Scope

Only read-only and documentary work remains in scope:

1. normalize workbook truth from PPIO
2. compare workbook truth with local aggregates
3. identify affected `day + model` buckets
4. estimate user / token / group impact in reports
5. archive findings and evidence-quality judgments
6. keep optional SQL drafts only as historical research artifacts, not execution plans

## Why We Are Stopping Short Of Mutation

The main reasons are:

- workbook truth is only authoritative at `day + model`
- most affected rows do not have enough retained request detail for deterministic replay
- a large share of the correction would still be residual-backed inference
- user-facing and channel-facing tables do not have perfectly reversible semantics
- any historical rewrite would mix true facts with inferred facts in the finest-grained local data

That tradeoff is not acceptable for the current decision.

## Allowed Tooling

The following tooling remains valid because it is read-only:

- [`scripts/ppio_billing_reconcile.py`](/Users/ash/Documents/GitHub/aiproxy/scripts/ppio_billing_reconcile.py)
- [`scripts/ppio_billing_repair_scope.py`](/Users/ash/Documents/GitHub/aiproxy/scripts/ppio_billing_repair_scope.py)

Their purpose is:

- workbook normalization
- scoped discrepancy analysis
- reporting support

They are not authorization to mutate production history.

## How To Use Existing Findings

Existing findings should now be treated as:

- reconciliation evidence
- user-impact estimates
- operational context for support / finance / internal review

They should **not** be treated as approval to execute a repair backfill.

## If This Is Revisited Later

If the team later wants to revisit a historical correction, that should be reopened as a separate decision with explicit approval and a fresh review of:

- evidence quality
- blast radius
- user/accounting consequences
- reversibility

Until then, the operative policy is:

- **diagnose**
- **document**
- **do not mutate production billing history**
