#!/bin/bash
# Responses API Billing Impact Assessment
# 评估受影响的日志数量和金额

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Responses API Billing Impact Assessment ===${NC}"
echo ""

# 数据库连接信息（请根据实际环境修改）
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-aiproxy}"
DB_NAME="${DB_NAME:-aiproxy}"
DB_PASSWORD="${DB_PASSWORD:-}"

# 时间范围（请根据实际情况修改）
START_DATE="${START_DATE:-2024-12-01}"
END_DATE="${END_DATE:-2025-01-01}"

echo -e "${YELLOW}Configuration:${NC}"
echo "  Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo "  User: $DB_USER"
echo "  Time range: $START_DATE ~ $END_DATE"
echo ""
echo -e "${YELLOW}Note: Set environment variables to override defaults:${NC}"
echo "  DB_HOST, DB_PORT, DB_USER, DB_NAME, START_DATE, END_DATE"
echo ""

read -p "Press Enter to continue or Ctrl+C to abort..."
echo ""

run_psql() {
    if [[ -n "$DB_PASSWORD" ]]; then
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" "$@"
    else
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" "$@"
    fi
}

# 临时 SQL 文件
TMPFILE=$(mktemp /tmp/assess_responses_api.XXXXXX.sql)
trap "rm -f $TMPFILE" EXIT

# 生成 SQL 查询
cat > "$TMPFILE" <<EOF
-- ============================================================================
-- 1. 总体统计
-- ============================================================================
\echo ''
\echo '=== 1. Overall Statistics ==='
\echo ''

SELECT
    COUNT(*) as total_logs,
    COUNT(CASE WHEN id IN (SELECT log_id FROM request_details) THEN 1 END) as has_detail,
    COUNT(CASE WHEN id NOT IN (SELECT log_id FROM request_details) THEN 1 END) as missing_detail,
    ROUND(COUNT(CASE WHEN id IN (SELECT log_id FROM request_details) THEN 1 END)::numeric / COUNT(*)::numeric * 100, 1) as has_detail_pct,
    ROUND(SUM(used_amount)::numeric, 2) as total_amount_usd,
    MIN(created_at) as earliest,
    MAX(created_at) as latest
FROM logs
WHERE
    endpoint = 'POST /v1/responses'
    AND cached_tokens = 0
    AND created_at >= '$START_DATE'
    AND created_at < '$END_DATE';

-- ============================================================================
-- 2. 按模型统计
-- ============================================================================
\echo ''
\echo '=== 2. Statistics by Model ==='
\echo ''

SELECT
    model,
    COUNT(*) as log_count,
    SUM(input_tokens) as total_input_tokens,
    SUM(cached_tokens) as total_cached_tokens,
    SUM(reasoning_tokens) as total_reasoning_tokens,
    SUM(output_tokens) as total_output_tokens,
    ROUND(SUM(used_amount)::numeric, 2) as total_amount_usd
FROM logs
WHERE
    endpoint = 'POST /v1/responses'
    AND cached_tokens = 0
    AND created_at >= '$START_DATE'
    AND created_at < '$END_DATE'
GROUP BY model
ORDER BY total_amount_usd DESC
LIMIT 10;

-- ============================================================================
-- 3. 按日期统计
-- ============================================================================
\echo ''
\echo '=== 3. Statistics by Date ==='
\echo ''

SELECT
    DATE(created_at) as day,
    COUNT(*) as log_count,
    ROUND(SUM(used_amount)::numeric, 2) as total_amount_usd
FROM logs
WHERE
    endpoint = 'POST /v1/responses'
    AND cached_tokens = 0
    AND created_at >= '$START_DATE'
    AND created_at < '$END_DATE'
GROUP BY DATE(created_at)
ORDER BY day DESC
LIMIT 10;

-- ============================================================================
-- 4. 按 group_id 统计（Top 10）
-- ============================================================================
\echo ''
\echo '=== 4. Top 10 Groups by Amount ==='
\echo ''

SELECT
    group_id,
    COUNT(*) as log_count,
    ROUND(SUM(used_amount)::numeric, 2) as total_amount_usd
FROM logs
WHERE
    endpoint = 'POST /v1/responses'
    AND cached_tokens = 0
    AND created_at >= '$START_DATE'
    AND created_at < '$END_DATE'
GROUP BY group_id
ORDER BY total_amount_usd DESC
LIMIT 10;

-- ============================================================================
-- 5. 抽样检查（5 条有 request_detail 的日志）
-- ============================================================================
\echo ''
\echo '=== 5. Sample Logs with request_detail ==='
\echo ''

SELECT
    l.id,
    l.request_id,
    l.model,
    l.created_at,
    l.input_tokens,
    l.cached_tokens,
    l.reasoning_tokens,
    l.output_tokens,
    ROUND(l.input_amount::numeric, 6) as input_amount,
    ROUND(l.cached_amount::numeric, 6) as cached_amount,
    ROUND(l.reasoning_amount::numeric, 6) as reasoning_amount,
    ROUND(l.output_amount::numeric, 6) as output_amount,
    ROUND(l.used_amount::numeric, 6) as used_amount
FROM logs l
WHERE
    l.endpoint = 'POST /v1/responses'
    AND l.cached_tokens = 0
    AND l.created_at >= '$START_DATE'
    AND l.created_at < '$END_DATE'
    AND l.id IN (SELECT log_id FROM request_details)
ORDER BY l.created_at DESC
LIMIT 5;

-- ============================================================================
-- 6. 修复建议
-- ============================================================================
\echo ''
\echo '=== 6. Repair Recommendation ==='
\echo ''

WITH stats AS (
    SELECT
        COUNT(*) as total_logs,
        COUNT(CASE WHEN id IN (SELECT log_id FROM request_details) THEN 1 END) as has_detail,
        SUM(used_amount) as total_amount
    FROM logs
    WHERE
        endpoint = 'POST /v1/responses'
        AND cached_tokens = 0
        AND created_at >= '$START_DATE'
        AND created_at < '$END_DATE'
)
SELECT
    CASE
        WHEN has_detail::numeric / total_logs::numeric > 0.8 THEN
            'Recommendation: Use Method A (Precise Repair via Go script)'
        ELSE
            'Recommendation: Use Method B (Reconciliation via PPIO workbook)'
    END as recommendation,
    ROUND(has_detail::numeric / total_logs::numeric * 100, 1) || '%' as has_detail_ratio,
    'Estimated refund: $' || ROUND(total_amount::numeric * 0.5, 2) || ' (assuming 50% overcharge)' as estimated_impact
FROM stats;

\echo ''
\echo '=== Assessment Complete ==='
\echo ''
\echo 'Next steps:'
\echo '1. Review the statistics above'
\echo '2. If has_detail_ratio > 80%, use: scripts/repair_responses_api_billing.go'
\echo '3. Otherwise, use: scripts/repair_responses_api_reconcile.py'
\echo '4. See docs/README_REPAIR.md for detailed instructions'
\echo ''
EOF

# 运行 SQL 查询
echo -e "${GREEN}Running assessment queries...${NC}"
echo ""

run_psql -f "$TMPFILE"

# 保存结果到文件
OUTPUT_FILE="tmp/responses_api_assessment_$(date +%Y%m%d_%H%M%S).txt"
mkdir -p tmp
run_psql -f "$TMPFILE" > "$OUTPUT_FILE" 2>&1

echo ""
echo -e "${GREEN}Assessment results saved to: $OUTPUT_FILE${NC}"
echo ""
