# PPIO Phase A User Adjustment Classification

## Scope

This note classifies the Phase A hybrid dry-run results into three buckets:

- users with clear attributable adjustments
- workbook truth buckets with no current user ownership
- buckets that are attributable but still evidence-limited and should be treated as pending review

Reference artifacts:

- [billing_fix_202604_ppio_phase_a_hybrid_dry_run_summary.md](/Users/ash/Documents/GitHub/aiproxy/tmp/ppio_billing_repair_runs/billing_fix_202604_ppio_phase_a_hybrid_dry_run_summary.md)
- [billing_fix_202604_ppio_phase_a_hybrid_day_model_plan.csv](/Users/ash/Documents/GitHub/aiproxy/tmp/ppio_billing_repair_runs/billing_fix_202604_ppio_phase_a_hybrid_day_model_plan.csv)
- [billing_fix_202604_ppio_phase_a_hybrid_per_log_preview.csv](/Users/ash/Documents/GitHub/aiproxy/tmp/ppio_billing_repair_runs/billing_fix_202604_ppio_phase_a_hybrid_per_log_preview.csv)

## Reading Guide

### Clear attributable adjustment

- there are local `logs`
- usage can be mapped to a concrete `group_name`
- the user/group should be adjusted in April reporting

### No current ownership

- workbook has a truth bucket
- local Phase A scope has zero matching `logs`
- there is no defensible current user/group assignment

### Pending review

- user ownership exists
- but most of the correction is still driven by workbook residual rather than replay-locked request-level evidence
- acceptable for repair, but should be explained as aggregate reconciliation instead of exact request replay truth

## A. Clear Attributable User Adjustments

These groups have clear local ownership and will be adjusted if Phase A is applied.

| group_name | current_used_amount | proposed_used_amount | delta_used_amount | replayable_rows | total_rows |
| --- | ---: | ---: | ---: | ---: | ---: |
| 文也 | 18128.9160 | 9067.2620 | -9061.6540 | 0 | 4656 |
| 德里克 | 11928.2272 | 5770.5296 | -6157.6977 | 352 | 10320 |
| 鸣人 | 10180.7886 | 4452.3025 | -5728.4861 | 220 | 3728 |
| 卡妙 | 13333.5653 | 8398.4008 | -4935.1645 | 97 | 12961 |
| 镜悬 | 9559.1110 | 5098.7069 | -4460.4041 | 172 | 5821 |
| 温莉 | 7252.0903 | 2913.8054 | -4338.2849 | 30 | 4378 |
| 藤丸 | 7263.6435 | 3351.5193 | -3912.1242 | 50 | 3616 |
| 白小飞 | 5365.4802 | 1541.7187 | -3823.7614 | 125 | 1290 |
| 兆南 | 6097.4201 | 2435.5991 | -3661.8210 | 47 | 1221 |
| 麦迪 | 3880.2674 | 1183.4510 | -2696.8163 | 197 | 1250 |
| 罗小黑 | 4266.4084 | 1827.3713 | -2439.0371 | 0 | 1149 |
| 天元 | 5361.6637 | 3030.0844 | -2331.5794 | 30 | 1459 |
| 麦克雷 | 3266.1152 | 982.2779 | -2283.8372 | 0 | 1396 |
| 李洛克 | 4276.4681 | 2067.3900 | -2209.0781 | 0 | 1654 |
| 影山 | 3363.0749 | 1168.5391 | -2194.5358 | 1 | 868 |

Operational reading:

- all rows above have concrete local ownership
- the repair direction is uniformly downward
- a non-zero `replayable_rows` count means at least part of the adjustment is supported by request-level replay locking

## B. No Current User Ownership

These are workbook truth buckets with zero current local logs inside the scoped Phase A window.

Current totals:

- bucket count: `180`
- total amount: `¥72.3096`
- max single bucket: `¥7.8400`

Top examples:

| day | model | truth_used_amount |
| --- | --- | ---: |
| 2026-04-07 | KLING_V30_PRO_I2V_NA | 7.8400 |
| 2026-04-07 | KLING_V30_PRO_T2V_NA | 7.8400 |
| 2026-04-07 | KLING_V30_STD_I2V_NA | 5.8800 |
| 2026-04-07 | KLING_V30_STD_T2V_NA | 5.8800 |
| 2026-04-06 | WAN2_6_T2V_1080P_5S | 5.0000 |
| 2026-04-07 | VIDU_Q3_PRO_T2V_720P_PEAK | 4.6875 |
| 2026-04-07 | KLING_O1_IMG_TO_VIDEO_720P_5S_TO_10S | 4.0600 |
| 2026-04-07 | KLING_O1_TEXT_TO_VIDEO_720P_5S_TO_10S | 4.0600 |
| 2026-04-06 | MINIMAX_MUSIC_2_5_PLUS | 4.0000 |
| 2026-04-01 | pa/gpt-5.4-pro | 3.6866 |

Pattern:

- image / video / music truth-only buckets
- Tavily / search style micro-buckets
- scattered `-dd` or alias model rows

Recommended handling:

- do not force-assign these to users
- keep them in a separate `unowned truth-only adjustment` bucket
- only merge them into user-facing data if a defensible attribution source is found later

## C. Pending Review Buckets

These have local ownership, but the evidence is still mostly aggregate rather than request-exact.

### High-priority pending review

These buckets are large and do have replay-locked rows, but replay still covers only a small share of truth:

| day | model | truth_used_amount | replayable_rows | non_replayable_rows | replayed_locked_used_amount | replay_share_of_truth |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| 2026-04-07 | pa/claude-opus-4-6 | 12976.0784 | 974 | 8800 | 1244.8149 | 0.0959 |
| 2026-04-07 | pa/claude-sonnet-4-6 | 10214.0553 | 1001 | 13599 | 727.7613 | 0.0713 |
| 2026-04-07 | pa/claude-haiku-4-5-20251001 | 260.1725 | 114 | 2277 | 13.1759 | 0.0506 |

Interpretation:

- user ownership is not in doubt
- repair direction is not in doubt
- exact per-request truth still is not fully recoverable
- this is where workbook residual remains the dominant correction source

### Lower-amount but stronger replay-share

These buckets have a somewhat stronger replay share, but the amounts are much smaller:

| day | model | truth_used_amount | replayable_rows | non_replayable_rows | replayed_locked_used_amount | replay_share_of_truth |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| 2026-04-07 | pa/claude-opus-4-6-dd | 22.0755 | 6 | 6 | 4.1031 | 0.1859 |
| 2026-04-07 | pa/claude-opus-4-5-20251101 | 40.1066 | 6 | 46 | 4.7351 | 0.1181 |

These are better candidates for “request replay supported” messaging, but they are not the main money.

## Recommended Communication Posture

### Users in section A

Say:

- the user attribution is clear
- April usage is being corrected downward
- exact request-level reconstruction is partial, but the group-level reconciliation is strong

### Buckets in section B

Say:

- there is upstream truth
- there is no current user-level ownership in local logs
- these stay outside direct user back-allocation for now

### Buckets in section C

Say:

- ownership is clear
- the adjustment is justified
- the final correction is a hybrid of request-level replay and workbook-constrained residual, not a pure request replay

## Bottom Line

For the current Phase A hybrid dry-run:

- the large user adjustments are attributable and actionable
- the unowned portion is small in money terms: `¥72.3096`
- the main open question is not “which users are affected”
- the main open question is “how to represent evidence quality” for large residual-driven buckets during apply and stakeholder communication
