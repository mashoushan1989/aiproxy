#!/bin/bash
# Adjust model prices based on PPIO upstream prices × discount factor.
#
# Strategy:
#   1. Call sync preview API to get upstream prices for changed models
#   2. For unchanged models (not in preview diff), current DB price IS upstream price
#   3. Apply discount, show comparison, update after confirmation
#
# Usage:
#   ADMIN_KEY=xxx bash scripts/adjust_model_price.sh
#
# Environment:
#   ADMIN_KEY  — aiproxy admin key (required, or prompted)
#   SSH_HOST   — SSH alias for production server (default: aiproxy-prod)

set -euo pipefail

SSH_HOST="${SSH_HOST:-aiproxy-prod}"
ADMIN_KEY="${ADMIN_KEY:-}"

if [[ -z "$ADMIN_KEY" ]]; then
  read -rp "请输入 ADMIN_KEY: " ADMIN_KEY
fi

# --- Step 1: Model name ---
echo ""
read -rp "请输入模型名称 (如 pa/gpt-5.5): " MODEL_NAME

if [[ -z "$MODEL_NAME" ]]; then
  echo "模型名称不能为空" >&2
  exit 1
fi

MODEL_URL=$(echo -n "$MODEL_NAME" | python3 -c "import sys,urllib.parse; print(urllib.parse.quote(sys.stdin.read(), safe=''))")

# --- Step 2: Fetch current aiproxy config ---
echo ""
echo "正在获取 aiproxy 当前配置..."

CURRENT_RAW=$(ssh -n "$SSH_HOST" \
  "curl -sf -H 'Authorization: Bearer $ADMIN_KEY' \
    http://127.0.0.1:3000/api/model_config/${MODEL_URL}" 2>/dev/null)

if [[ -z "$CURRENT_RAW" ]]; then
  echo "获取当前模型配置失败，请检查模型名称" >&2
  exit 1
fi

CONFIG=$(echo "$CURRENT_RAW" | jq '.data // .')
PRICE=$(echo "$CONFIG" | jq '.price // {}')
SYNCED_FROM=$(echo "$CONFIG" | jq -r '.synced_from // ""')

if [[ "$SYNCED_FROM" != "ppio" ]]; then
  echo "⚠ 该模型 synced_from='$SYNCED_FROM'，非 PPIO 同步模型" >&2
  echo "脚本仅适用于 PPIO 同步模型" >&2
  exit 1
fi

CURRENT_INPUT=$(echo "$PRICE" | jq '.input_price // 0')
CURRENT_OUTPUT=$(echo "$PRICE" | jq '.output_price // 0')
CURRENT_CACHED=$(echo "$PRICE" | jq '.cached_price // 0')
CURRENT_CACHE_CREATION=$(echo "$PRICE" | jq '.cache_creation_price // 0')
HAS_CONDITIONAL=$(echo "$PRICE" | jq 'if (.conditional_prices // []) | length > 0 then true else false end')

# --- Step 3: Fetch PPIO upstream prices ---
echo "正在获取 PPIO 上游价格 (sync preview API)..."

PREVIEW_RAW=$(ssh -n "$SSH_HOST" \
  "curl -sf -X POST \
    -H 'Authorization: Bearer $ADMIN_KEY' \
    -H 'Content-Type: application/json' \
    -d '{}' \
    http://127.0.0.1:3000/api/enterprise/ppio/sync/preview" 2>/dev/null)

if [[ -z "$PREVIEW_RAW" ]]; then
  echo "sync preview API 请求失败" >&2
  exit 1
fi

# Search in all diff lists (add/update/shared)
UPSTREAM_MODEL=$(echo "$PREVIEW_RAW" | jq --arg m "$MODEL_NAME" '
  [.data.changes.add[]?, .data.changes.update[]?, .data.changes.shared[]?]
  | map(select(.model_id == $m)) | .[0] // empty')

if [[ -n "$UPSTREAM_MODEL" && "$UPSTREAM_MODEL" != "null" ]]; then
  # Model found in diff — upstream prices differ from current
  UPSTREAM_INPUT=$(echo "$UPSTREAM_MODEL" | jq '.new_config.input_price // 0')
  UPSTREAM_OUTPUT=$(echo "$UPSTREAM_MODEL" | jq '.new_config.output_price // 0')
  UPSTREAM_CACHED=$(echo "$UPSTREAM_MODEL" | jq '.new_config.cache_read // 0')
  UPSTREAM_CACHE_CREATION=$(echo "$UPSTREAM_MODEL" | jq '.new_config.cache_creation // 0')
  UPSTREAM_IS_TIERED=$(echo "$UPSTREAM_MODEL" | jq '.new_config.is_tiered // false')
  PRICE_SOURCE="sync preview (上游有变更)"
else
  # Model not in diff — current prices ARE upstream prices (already synced)
  UPSTREAM_INPUT="$CURRENT_INPUT"
  UPSTREAM_OUTPUT="$CURRENT_OUTPUT"
  UPSTREAM_CACHED="$CURRENT_CACHED"
  UPSTREAM_CACHE_CREATION="$CURRENT_CACHE_CREATION"
  UPSTREAM_IS_TIERED="$HAS_CONDITIONAL"
  PRICE_SOURCE="当前配置 (与上游一致，无待同步变更)"
fi

# --- Step 4: Display prices ---
echo ""
echo "========== PPIO 上游价格 (元/token) =========="
echo "来源: $PRICE_SOURCE"
printf "%-30s %s\n" "input_price" "$UPSTREAM_INPUT"
printf "%-30s %s\n" "output_price" "$UPSTREAM_OUTPUT"
[[ "$UPSTREAM_CACHED" != "0" ]] && printf "%-30s %s\n" "cached_price" "$UPSTREAM_CACHED"
[[ "$UPSTREAM_CACHE_CREATION" != "0" ]] && printf "%-30s %s\n" "cache_creation_price" "$UPSTREAM_CACHE_CREATION"
[[ "$UPSTREAM_IS_TIERED" == "true" ]] && echo "(阶梯定价模型)"

echo ""
echo "========== aiproxy 当前价格 (元/token) =========="
printf "%-30s %s\n" "input_price" "$CURRENT_INPUT"
printf "%-30s %s\n" "output_price" "$CURRENT_OUTPUT"
[[ "$CURRENT_CACHED" != "0" ]] && printf "%-30s %s\n" "cached_price" "$CURRENT_CACHED"
[[ "$CURRENT_CACHE_CREATION" != "0" ]] && printf "%-30s %s\n" "cache_creation_price" "$CURRENT_CACHE_CREATION"

if [[ "$HAS_CONDITIONAL" == "true" ]]; then
  echo ""
  echo "阶梯定价 (conditional_prices):"
  echo "$PRICE" | jq -r '.conditional_prices[] |
    "  [\(.condition.input_token_min)-\(.condition.input_token_max)] input=\(.price.input_price) output=\(.price.output_price) cached=\(.price.cached_price // "N/A")"'
fi

# --- Step 5: Ask for discount ---
echo ""
read -rp "请输入折扣系数 (基于上游价格，如 0.8=八折, 0.1=一折): " DISCOUNT

if [[ -z "$DISCOUNT" ]]; then
  echo "折扣系数不能为空" >&2
  exit 1
fi

if ! echo "$DISCOUNT" | grep -qE '^[0-9]*\.?[0-9]+$'; then
  echo "折扣系数必须是正数" >&2
  exit 1
fi

# --- Step 6: Calculate target prices ---
TARGET_INPUT=$(awk "BEGIN {printf \"%.15g\", $UPSTREAM_INPUT * $DISCOUNT}")
TARGET_OUTPUT=$(awk "BEGIN {printf \"%.15g\", $UPSTREAM_OUTPUT * $DISCOUNT}")
TARGET_CACHED="0"
TARGET_CACHE_CREATION="0"

[[ "$UPSTREAM_CACHED" != "0" ]] && TARGET_CACHED=$(awk "BEGIN {printf \"%.15g\", $UPSTREAM_CACHED * $DISCOUNT}")
[[ "$UPSTREAM_CACHE_CREATION" != "0" ]] && TARGET_CACHE_CREATION=$(awk "BEGIN {printf \"%.15g\", $UPSTREAM_CACHE_CREATION * $DISCOUNT}")

# --- Step 7: Show comparison ---
echo ""
echo "========== 价格调整预览 (上游 × ${DISCOUNT}) =========="
printf "%-25s %-22s %-22s %-22s\n" "字段" "PPIO上游" "折后目标" "当前价格"
echo "------------------------------------------------------------------------------------"
printf "%-25s %-22s %-22s %-22s\n" "input_price" "$UPSTREAM_INPUT" "$TARGET_INPUT" "$CURRENT_INPUT"
printf "%-25s %-22s %-22s %-22s\n" "output_price" "$UPSTREAM_OUTPUT" "$TARGET_OUTPUT" "$CURRENT_OUTPUT"

if [[ "$UPSTREAM_CACHED" != "0" || "$CURRENT_CACHED" != "0" ]]; then
  printf "%-25s %-22s %-22s %-22s\n" "cached_price" "$UPSTREAM_CACHED" "$TARGET_CACHED" "$CURRENT_CACHED"
fi
if [[ "$UPSTREAM_CACHE_CREATION" != "0" || "$CURRENT_CACHE_CREATION" != "0" ]]; then
  printf "%-25s %-22s %-22s %-22s\n" "cache_creation_price" "$UPSTREAM_CACHE_CREATION" "$TARGET_CACHE_CREATION" "$CURRENT_CACHE_CREATION"
fi

# Handle conditional/tiered prices
if [[ "$HAS_CONDITIONAL" == "true" ]]; then
  echo ""
  echo "阶梯定价调整 (× ${DISCOUNT}):"
  printf "  %-30s %-18s %-18s\n" "Token 范围" "当前 input" "折后 input"
  echo "  ---------------------------------------------------------------"

  echo "$PRICE" | jq -r --argjson d "$DISCOUNT" '.conditional_prices[] |
    "  [\(.condition.input_token_min)-\(.condition.input_token_max)]  \(.price.input_price)  →  \(.price.input_price * $d)"'
fi

# --- Step 8: Build update payload ---
UPDATED_PRICE=$(echo "$PRICE" | jq \
  --argjson ip "$TARGET_INPUT" \
  --argjson op "$TARGET_OUTPUT" \
  '.input_price = $ip | .output_price = $op | .input_price_unit = 1 | .output_price_unit = 1')

if [[ "$TARGET_CACHED" != "0" ]]; then
  UPDATED_PRICE=$(echo "$UPDATED_PRICE" | jq --argjson cp "$TARGET_CACHED" '.cached_price = $cp | .cached_price_unit = 1')
elif [[ "$CURRENT_CACHED" != "0" ]]; then
  # Upstream has no cache price, clear it
  UPDATED_PRICE=$(echo "$UPDATED_PRICE" | jq '.cached_price = 0')
fi

if [[ "$TARGET_CACHE_CREATION" != "0" ]]; then
  UPDATED_PRICE=$(echo "$UPDATED_PRICE" | jq --argjson ccp "$TARGET_CACHE_CREATION" '.cache_creation_price = $ccp | .cache_creation_price_unit = 1')
elif [[ "$CURRENT_CACHE_CREATION" != "0" ]]; then
  UPDATED_PRICE=$(echo "$UPDATED_PRICE" | jq '.cache_creation_price = 0')
fi

# Scale conditional prices by discount
if [[ "$HAS_CONDITIONAL" == "true" ]]; then
  UPDATED_PRICE=$(echo "$UPDATED_PRICE" | jq --argjson d "$DISCOUNT" '
    .conditional_prices = [.conditional_prices[] | {
      condition: .condition,
      price: (.price | to_entries | map(
        if (.key | test("_price$")) and (.key | test("_unit$") | not) and .value != null and (.value | type) == "number" then
          .value *= $d
        else . end
      ) | from_entries)
    }]')
fi

UPDATED_CONFIG=$(echo "$CONFIG" | jq --argjson p "$UPDATED_PRICE" '.price = $p')

# --- Step 9: Confirm ---
echo ""
read -rp "确认执行价格调整？(y/N): " CONFIRM

if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
  echo "已取消"
  exit 0
fi

# --- Step 10: Execute update ---
echo ""
echo "正在更新..."

PAYLOAD_FILE=$(mktemp)
echo "$UPDATED_CONFIG" | jq -c . > "$PAYLOAD_FILE"

UPDATE_RESP=$(cat "$PAYLOAD_FILE" | ssh "$SSH_HOST" \
  "cat | curl -sf -X POST \
    -H 'Authorization: Bearer $ADMIN_KEY' \
    -H 'Content-Type: application/json' \
    -d @- \
    http://127.0.0.1:3000/api/model_config/${MODEL_URL}" 2>/dev/null)

rm -f "$PAYLOAD_FILE"

SUCCESS=$(echo "$UPDATE_RESP" | jq -r '.success // true')
if [[ "$SUCCESS" == "false" ]]; then
  MSG=$(echo "$UPDATE_RESP" | jq -r '.message // "unknown error"')
  echo "更新失败: $MSG" >&2
  exit 1
fi

echo "✅ 价格调整完成"

# --- Step 11: Verify ---
echo ""
echo "验证更新结果..."

VERIFY_RAW=$(ssh -n "$SSH_HOST" \
  "curl -sf -H 'Authorization: Bearer $ADMIN_KEY' \
    http://127.0.0.1:3000/api/model_config/${MODEL_URL}" 2>/dev/null)

VERIFY_PRICE=$(echo "$VERIFY_RAW" | jq '.data.price // {}')

echo ""
echo "========== 更新后价格 =========="
printf "%-30s %s\n" "input_price" "$(echo "$VERIFY_PRICE" | jq '.input_price // 0')"
printf "%-30s %s\n" "output_price" "$(echo "$VERIFY_PRICE" | jq '.output_price // 0')"

V_CACHED=$(echo "$VERIFY_PRICE" | jq '.cached_price // 0')
V_CC=$(echo "$VERIFY_PRICE" | jq '.cache_creation_price // 0')
[[ "$V_CACHED" != "0" ]] && printf "%-30s %s\n" "cached_price" "$V_CACHED"
[[ "$V_CC" != "0" ]] && printf "%-30s %s\n" "cache_creation_price" "$V_CC"

V_HAS_CP=$(echo "$VERIFY_PRICE" | jq 'if (.conditional_prices // []) | length > 0 then true else false end')
if [[ "$V_HAS_CP" == "true" ]]; then
  echo ""
  echo "阶梯定价:"
  echo "$VERIFY_PRICE" | jq -r '.conditional_prices[] |
    "  [\(.condition.input_token_min)-\(.condition.input_token_max)] input=\(.price.input_price) output=\(.price.output_price)"'
fi

echo ""
echo "完成。"
