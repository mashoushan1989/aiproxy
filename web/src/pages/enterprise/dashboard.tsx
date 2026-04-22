import { useState, useMemo, useEffect, useRef } from "react"
import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useQuery } from "@tanstack/react-query"
import { BarChart2, DollarSign, Hash, Building2, ArrowRight, TrendingUp, TrendingDown, Minus, ArrowUpDown, ArrowUp, ArrowDown, Settings2, Filter, Layers3, Sparkles, Activity, Users } from "lucide-react"
import * as echarts from "echarts"
import { type DateRange } from "react-day-picker"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { DateRangePicker } from "@/components/common/DateRangePicker"
import { enterpriseApi, type DepartmentSummary, type ModelDistributionItem, type FeishuDepartment } from "@/api/enterprise"
import { ROUTES } from "@/routes/constants"
import { type TimeRange, getTimeRange, formatNumber, formatAmount, formatRate, useDarkMode, getEChartsTheme } from "@/lib/enterprise"
import { cn } from "@/lib/utils"

// Column configuration for department summary table
type DeptSortField = "department_name" | "member_count" | "active_users" | "request_count" | "used_amount" | "total_tokens" | "input_tokens" | "output_tokens" | "success_rate" | "avg_cost" | "avg_cost_per_user" | "unique_models"
type SortDirection = "asc" | "desc"

interface DeptColumnConfig {
    key: DeptSortField
    labelKey: string
    align: "left" | "right"
    defaultVisible: boolean
    format?: (value: number) => string
    renderCell?: (dept: DepartmentSummary) => React.ReactNode
}

const DEPT_COLUMNS: DeptColumnConfig[] = [
    { key: "department_name", labelKey: "enterprise.dashboard.department", align: "left", defaultVisible: true },
    { key: "request_count", labelKey: "enterprise.dashboard.requests", align: "right", defaultVisible: true, format: formatNumber },
    { key: "used_amount", labelKey: "enterprise.dashboard.amount", align: "right", defaultVisible: true, format: formatAmount },
    { key: "total_tokens", labelKey: "enterprise.dashboard.tokens", align: "right", defaultVisible: false, format: formatNumber },
    { key: "input_tokens", labelKey: "enterprise.dashboard.inputTokens", align: "right", defaultVisible: false, format: formatNumber },
    { key: "output_tokens", labelKey: "enterprise.dashboard.outputTokens", align: "right", defaultVisible: false, format: formatNumber },
    { key: "active_users", labelKey: "enterprise.dashboard.activeUsers", align: "right", defaultVisible: true },
    { key: "member_count", labelKey: "enterprise.dashboard.memberCount", align: "right", defaultVisible: false },
    { key: "success_rate", labelKey: "enterprise.dashboard.successRate", align: "right", defaultVisible: true,
        format: formatRate },
    { key: "avg_cost", labelKey: "enterprise.dashboard.avgCost", align: "right", defaultVisible: true, format: formatAmount },
    { key: "avg_cost_per_user", labelKey: "enterprise.dashboard.avgCostPerUser", align: "right", defaultVisible: true, format: formatAmount },
    { key: "unique_models", labelKey: "enterprise.dashboard.uniqueModels", align: "right", defaultVisible: false, format: formatNumber },
]

// Driven automatically by the hierarchy filter — gives drill-down without extra UI.
type AggLevel = "level1" | "level2" | "leaf"

function rollupDepartments(
    items: DepartmentSummary[],
    level: "level1" | "level2",
    nameMap: Map<string, string>,
): DepartmentSummary[] {
    const isL1 = level === "level1"
    const acc = new Map<string, DepartmentSummary>()
    const successWeighted = new Map<string, number>()
    for (const d of items) {
        const key = (isL1 ? d.level1_dept_id : d.level2_dept_id) || d.department_id
        if (!key) continue
        let cur = acc.get(key)
        if (!cur) {
            cur = {
                department_id: key,
                department_name: nameMap.get(key) || d.department_name || key,
                level1_dept_id: isL1 ? key : d.level1_dept_id,
                level2_dept_id: isL1 ? "" : key,
                member_count: 0,
                active_users: 0,
                request_count: 0,
                used_amount: 0,
                total_tokens: 0,
                input_tokens: 0,
                output_tokens: 0,
                success_rate: 0,
                avg_cost: 0,
                avg_cost_per_user: 0,
                unique_models: 0,
            }
            acc.set(key, cur)
        }
        cur.member_count += d.member_count
        cur.active_users += d.active_users
        cur.request_count += d.request_count
        cur.used_amount += d.used_amount
        cur.total_tokens += d.total_tokens
        cur.input_tokens += d.input_tokens
        cur.output_tokens += d.output_tokens
        cur.unique_models += d.unique_models
        successWeighted.set(key, (successWeighted.get(key) ?? 0) + (d.success_rate / 100) * d.request_count)
    }
    for (const [key, cur] of acc) {
        if (cur.request_count > 0) {
            cur.success_rate = ((successWeighted.get(key) ?? 0) / cur.request_count) * 100
            cur.avg_cost = cur.used_amount / cur.request_count
        }
        if (cur.active_users > 0) {
            cur.avg_cost_per_user = cur.used_amount / cur.active_users
        }
    }
    return [...acc.values()]
}

function MetricCard({
    title,
    value,
    icon: Icon,
    changePct,
}: {
    title: string
    value: string | number
    icon: React.ComponentType<{ className?: string }>
    changePct?: number
}) {
    return (
        <Card className="border border-gray-100 dark:border-gray-800">
            <CardContent className="p-6">
                <div className="flex items-center justify-between">
                    <div>
                        <p className="text-sm text-muted-foreground">{title}</p>
                        <p className="text-2xl font-bold mt-1">{value}</p>
                        {changePct !== undefined && (
                            <div className="flex items-center gap-1 mt-1">
                                {changePct > 0 ? (
                                    <TrendingUp className="w-3.5 h-3.5 text-green-500" />
                                ) : changePct < 0 ? (
                                    <TrendingDown className="w-3.5 h-3.5 text-red-500" />
                                ) : (
                                    <Minus className="w-3.5 h-3.5 text-muted-foreground" />
                                )}
                                <span
                                    className={`text-xs font-medium ${
                                        changePct > 0
                                            ? "text-green-500"
                                            : changePct < 0
                                              ? "text-red-500"
                                              : "text-muted-foreground"
                                    }`}
                                >
                                    {changePct > 0 ? "+" : ""}
                                    {changePct.toFixed(1)}%
                                </span>
                            </div>
                        )}
                    </div>
                    <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-[#6A6DE6]/10 to-[#8A8DF7]/10 flex items-center justify-center">
                        <Icon className="w-5 h-5 text-[#6A6DE6]" />
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}

function InsightTile({
    title,
    value,
    detail,
    icon: Icon,
}: {
    title: string
    value: string
    detail: string
    icon: React.ComponentType<{ className?: string }>
}) {
    return (
        <div className="rounded-2xl border border-border/60 bg-background/80 p-4 shadow-sm">
            <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                    <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{title}</p>
                    <p className="mt-2 truncate text-base font-semibold text-foreground">{value}</p>
                    <p className="mt-1 text-sm text-muted-foreground">{detail}</p>
                </div>
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-muted/70">
                    <Icon className="h-4 w-4 text-[#6A6DE6]" />
                </div>
            </div>
        </div>
    )
}

function DepartmentPieChart({ departments }: { departments: DepartmentSummary[] }) {
    const chartRef = useRef<HTMLDivElement>(null)
    const chartInstance = useRef<echarts.ECharts | null>(null)
    const isDark = useDarkMode()

    useEffect(() => {
        if (!chartRef.current) return

        if (!chartInstance.current) {
            chartInstance.current = echarts.init(chartRef.current)
        }

        const theme = getEChartsTheme(isDark)
        const data = departments
            .filter((d) => d.used_amount > 0)
            .sort((a, b) => b.used_amount - a.used_amount)
            .slice(0, 10)
            .map((d) => ({
                name: d.department_name || d.department_id,
                value: Math.round(d.used_amount * 100) / 100,
            }))

        chartInstance.current.setOption({
            tooltip: {
                trigger: "item",
                formatter: "{b}: ¥{c} ({d}%)",
            },
            series: [
                {
                    type: "pie",
                    radius: ["40%", "70%"],
                    avoidLabelOverlap: true,
                    itemStyle: {
                        borderRadius: 6,
                        borderColor: theme.borderColor,
                        borderWidth: 2,
                    },
                    label: {
                        show: true,
                        formatter: "{b}",
                        color: theme.textColor,
                    },
                    data,
                },
            ],
        })

        const handleResize = () => chartInstance.current?.resize()
        window.addEventListener("resize", handleResize)

        return () => {
            window.removeEventListener("resize", handleResize)
            chartInstance.current?.dispose()
            chartInstance.current = null
        }
    }, [departments, isDark])

    return <div ref={chartRef} className="h-[420px] w-full" />
}

function ModelDistributionChart({ models }: { models: ModelDistributionItem[] }) {
    const chartRef = useRef<HTMLDivElement>(null)
    const chartInstance = useRef<echarts.ECharts | null>(null)
    const isDark = useDarkMode()
    const { t } = useTranslation()

    useEffect(() => {
        if (!chartRef.current || models.length === 0) return

        if (!chartInstance.current) {
            chartInstance.current = echarts.init(chartRef.current)
        }

        const theme = getEChartsTheme(isDark)
        const top10 = models.slice(0, 10)
        chartInstance.current.setOption({
            tooltip: {
                trigger: "axis",
                axisPointer: { type: "shadow" },
            },
            grid: {
                left: "3%",
                right: "4%",
                bottom: "3%",
                top: "8%",
                containLabel: true,
            },
            xAxis: {
                type: "category",
                data: top10.map((m) => {
                    const parts = m.model.split("/")
                    return parts[parts.length - 1]
                }),
                axisLabel: { rotate: 30, fontSize: 10, color: theme.subTextColor },
            },
            yAxis: {
                type: "value",
                name: t("enterprise.department.chartAmount"),
                nameTextStyle: { color: theme.subTextColor },
                axisLabel: { color: theme.subTextColor },
                splitLine: { lineStyle: { color: theme.splitLineColor } },
            },
            series: [
                {
                    type: "bar",
                    data: top10.map((m) => ({
                        value: Math.round(m.used_amount * 100) / 100,
                        itemStyle: { borderRadius: [4, 4, 0, 0] },
                    })),
                    itemStyle: {
                        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                            { offset: 0, color: "#6A6DE6" },
                            { offset: 1, color: "#8A8DF7" },
                        ]),
                    },
                },
            ],
        })

        const handleResize = () => chartInstance.current?.resize()
        window.addEventListener("resize", handleResize)

        return () => {
            window.removeEventListener("resize", handleResize)
            chartInstance.current?.dispose()
            chartInstance.current = null
        }
    }, [models, isDark, t])

    return <div ref={chartRef} className="h-[420px] w-full" />
}

export default function EnterpriseDashboard() {
    const { t } = useTranslation()
    const navigate = useNavigate()
    const [timeRange, setTimeRange] = useState<TimeRange>("7d")
    const [customDateRange, setCustomDateRange] = useState<DateRange | undefined>()

    // Hierarchical department filters (multi-select)
    const [selectedLevel1s, setSelectedLevel1s] = useState<Set<string>>(new Set())
    const [selectedLevel2s, setSelectedLevel2s] = useState<Set<string>>(new Set())

    // Column visibility
    const [visibleColumns, setVisibleColumns] = useState<Set<DeptSortField>>(() => {
        return new Set(DEPT_COLUMNS.filter(c => c.defaultVisible).map(c => c.key))
    })

    // Sorting
    const [sortField, setSortField] = useState<DeptSortField>("used_amount")
    const [sortDirection, setSortDirection] = useState<SortDirection>("desc")

    const { start, end } = useMemo(() => {
        if (timeRange === "custom" && customDateRange?.from) {
            const startTs = Math.floor(customDateRange.from.getTime() / 1000)
            const endTs = customDateRange.to
                ? Math.floor(customDateRange.to.getTime() / 1000) + 86399 // End of day
                : Math.floor(Date.now() / 1000)
            return getTimeRange("custom", startTs, endTs)
        }
        return getTimeRange(timeRange)
    }, [timeRange, customDateRange])

    // Fetch all level1 departments
    const { data: deptLevels } = useQuery({
        queryKey: ["enterprise", "department-levels"],
        queryFn: () => enterpriseApi.getDepartmentLevels(),
    })

    const level1Departments = useMemo(() => deptLevels?.level1_departments ?? [], [deptLevels])

    // Fetch level2 departments for all selected level1s
    const selectedLevel1Array = useMemo(() => [...selectedLevel1s].sort(), [selectedLevel1s])
    const { data: allLevel2Departments = [] } = useQuery({
        queryKey: ["enterprise", "department-levels-l2", selectedLevel1Array],
        queryFn: async () => {
            const results = await Promise.all(
                selectedLevel1Array.map(id => enterpriseApi.getDepartmentLevels(id))
            )
            const seen = new Set<string>()
            const merged: FeishuDepartment[] = []
            for (const r of results) {
                for (const d of (r.level2_departments || [])) {
                    if (!seen.has(d.department_id)) {
                        seen.add(d.department_id)
                        merged.push(d)
                    }
                }
            }
            return merged
        },
        enabled: selectedLevel1s.size > 0,
    })

    // Build department filter for API — pass all selected IDs for consistent filtering.
    const departmentFilters = useMemo(() => {
        if (selectedLevel2s.size > 0) return [...selectedLevel2s]
        if (selectedLevel1s.size > 0) return [...selectedLevel1s]
        return undefined
    }, [selectedLevel1s, selectedLevel2s])

    const { data, isLoading } = useQuery({
        queryKey: ["enterprise", "department-summary", start, end],
        queryFn: () => enterpriseApi.getDepartmentSummary(start, end),
    })

    const { data: comparisonData } = useQuery({
        queryKey: ["enterprise", "comparison", start, end, departmentFilters],
        queryFn: () => enterpriseApi.getComparison(departmentFilters, start, end),
    })

    const { data: modelData } = useQuery({
        queryKey: ["enterprise", "model-distribution", start, end, departmentFilters],
        queryFn: () => enterpriseApi.getModelDistribution(departmentFilters, start, end),
    })

    const level1NameMap = useMemo(() => {
        const m = new Map<string, string>()
        for (const d of level1Departments) m.set(d.department_id, d.name || d.department_id)
        return m
    }, [level1Departments])

    const level2NameMap = useMemo(() => {
        const m = new Map<string, string>()
        for (const d of allLevel2Departments) m.set(d.department_id, d.name || d.department_id)
        return m
    }, [allLevel2Departments])

    const aggLevel: AggLevel = selectedLevel2s.size > 0
        ? "leaf"
        : selectedLevel1s.size > 0 ? "level2" : "level1"

    const departments = useMemo(() => {
        const allDepts = data?.departments || []
        let filtered = allDepts
        if (selectedLevel2s.size > 0) {
            filtered = allDepts.filter(d =>
                selectedLevel2s.has(d.department_id) || selectedLevel2s.has(d.level2_dept_id)
            )
        } else if (selectedLevel1s.size > 0) {
            filtered = allDepts.filter(d =>
                selectedLevel1s.has(d.department_id) || selectedLevel1s.has(d.level1_dept_id)
            )
        }

        if (selectedLevel2s.size > 0) return filtered
        const level: "level1" | "level2" = selectedLevel1s.size > 0 ? "level2" : "level1"
        return rollupDepartments(filtered, level, level === "level1" ? level1NameMap : level2NameMap)
    }, [data?.departments, selectedLevel1s, selectedLevel2s, level1NameMap, level2NameMap])

    const models = modelData?.distribution || []
    const changes = comparisonData?.changes
    const activeFilterCount = selectedLevel1s.size + selectedLevel2s.size + (timeRange === "custom" ? 1 : 0)
    const customRangeLabel = timeRange === "custom" && customDateRange?.from
        ? `${customDateRange.from.toLocaleDateString()}${customDateRange.to ? ` - ${customDateRange.to.toLocaleDateString()}` : ""}`
        : null

    // Sort departments
    const sortedDepartments = useMemo(() => {
        if (!departments.length) return departments
        return [...departments].sort((a, b) => {
            if (sortField === "department_name") {
                const aVal = a.department_name || a.department_id || ""
                const bVal = b.department_name || b.department_id || ""
                const cmp = aVal.localeCompare(bVal, "zh-CN")
                return sortDirection === "asc" ? cmp : -cmp
            }
            const aNum = Number(a[sortField]) || 0
            const bNum = Number(b[sortField]) || 0
            return sortDirection === "asc" ? aNum - bNum : bNum - aNum
        })
    }, [departments, sortField, sortDirection])

    const totals = useMemo(() => {
        return departments.reduce(
            (acc, d) => ({
                requests: acc.requests + (d.request_count || 0),
                amount: acc.amount + (d.used_amount || 0),
                tokens: acc.tokens + (d.total_tokens || 0),
                activeUsers: acc.activeUsers + (d.active_users || 0),
            }),
            { requests: 0, amount: 0, tokens: 0, activeUsers: 0 },
        )
    }, [departments])
    const topDepartment = sortedDepartments[0]
    const topModel = models[0]
    const avgTokensPerRequest = totals.requests > 0 ? totals.tokens / totals.requests : 0
    const avgAmountPerRequest = totals.requests > 0 ? totals.amount / totals.requests : 0

    const handleSort = (field: DeptSortField) => {
        if (sortField === field) {
            setSortDirection(prev => prev === "asc" ? "desc" : "asc")
        } else {
            setSortField(field)
            setSortDirection(field === "department_name" ? "asc" : "desc")
        }
    }

    const toggleColumn = (key: DeptSortField) => {
        setVisibleColumns(prev => {
            const next = new Set(prev)
            if (next.has(key)) {
                if (key !== "department_name") {
                    next.delete(key)
                    if (sortField === key) {
                        setSortField("used_amount")
                        setSortDirection("desc")
                    }
                }
            } else {
                next.add(key)
            }
            return next
        })
    }

    const toggleLevel1 = (deptId: string) => {
        setSelectedLevel1s(prev => {
            const next = new Set(prev)
            if (next.has(deptId)) next.delete(deptId)
            else next.add(deptId)
            return next
        })
        setSelectedLevel2s(new Set())
    }

    const toggleLevel2 = (deptId: string) => {
        setSelectedLevel2s(prev => {
            const next = new Set(prev)
            if (next.has(deptId)) next.delete(deptId)
            else next.add(deptId)
            return next
        })
    }

    const renderSortIcon = (field: DeptSortField) => {
        if (sortField !== field) {
            return <ArrowUpDown className="ml-1 h-3 w-3 opacity-50" />
        }
        return sortDirection === "asc"
            ? <ArrowUp className="ml-1 h-3 w-3" />
            : <ArrowDown className="ml-1 h-3 w-3" />
    }

    const visibleColumnConfigs = DEPT_COLUMNS.filter(c => visibleColumns.has(c.key))
    const clearFilters = () => {
        setSelectedLevel1s(new Set())
        setSelectedLevel2s(new Set())
        setTimeRange("7d")
        setCustomDateRange(undefined)
    }

    const getCellValue = (dept: DepartmentSummary, col: DeptColumnConfig) => {
        if (col.key === "department_name") {
            return (
                <div className="min-w-0">
                    <div className="truncate font-medium">{dept.department_name || dept.department_id}</div>
                    <div className="text-xs text-muted-foreground">
                        {formatNumber(dept.request_count)} · {dept.active_users} {t("enterprise.dashboard.activeUsers")}
                    </div>
                </div>
            )
        }
        if (col.renderCell) {
            return col.renderCell(dept)
        }
        const value = dept[col.key as keyof DepartmentSummary]
        if (col.format && typeof value === "number") {
            return col.format(value)
        }
        return value
    }

    return (
        <div className="space-y-6 bg-[radial-gradient(circle_at_top,_rgba(106,109,230,0.08),_transparent_28%),linear-gradient(180deg,rgba(248,250,252,0.95),rgba(255,255,255,0.98))] p-6 dark:bg-[radial-gradient(circle_at_top,_rgba(106,109,230,0.18),_transparent_24%),linear-gradient(180deg,rgba(2,6,23,0.98),rgba(2,6,23,0.94))]">
            {/* Header with filters */}
            <div className="rounded-[28px] border border-border/60 bg-background/88 p-6 shadow-lg shadow-slate-200/40 backdrop-blur-sm dark:shadow-none">
                <div className="flex flex-wrap items-start justify-between gap-4">
                    <div className="space-y-3">
                        <Badge variant="outline" className="gap-1 rounded-full border-border/70 bg-background/70 px-3 py-1 text-xs text-muted-foreground">
                            <Sparkles className="h-3.5 w-3.5 text-[#6A6DE6]" />
                            {t("enterprise.dashboard.title")}
                        </Badge>
                        <div>
                            <h1 className="text-3xl font-semibold tracking-tight">{t("enterprise.dashboard.title")}</h1>
                            <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
                                {t("enterprise.dashboard.subtitle")}
                            </p>
                        </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                        {activeFilterCount > 0 && (
                            <Badge variant="secondary" className="gap-1 rounded-full px-3 py-1">
                                <Filter className="h-3.5 w-3.5" />
                                {t("enterprise.dashboard.filtersApplied", { count: activeFilterCount })}
                            </Badge>
                        )}
                        {selectedLevel1s.size > 0 && (
                            <Badge variant="outline" className="rounded-full px-3 py-1">
                                {t("enterprise.dashboard.level1Dept")} {selectedLevel1s.size}
                            </Badge>
                        )}
                        {selectedLevel2s.size > 0 && (
                            <Badge variant="outline" className="rounded-full px-3 py-1">
                                {t("enterprise.dashboard.level2Dept")} {selectedLevel2s.size}
                            </Badge>
                        )}
                        {customRangeLabel && (
                            <Badge variant="outline" className="rounded-full px-3 py-1">
                                {customRangeLabel}
                            </Badge>
                        )}
                    </div>
                </div>

                <div className="mt-6 flex flex-wrap items-center gap-3">
                    {/* Level 1 Department Filter (multi-select) */}
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="outline" className="w-44 justify-start gap-1.5 bg-background/80">
                                <Building2 className="w-4 h-4 shrink-0" />
                                <span className="truncate">
                                    {selectedLevel1s.size === 0
                                        ? t("enterprise.dashboard.allLevel1Depts")
                                        : t("enterprise.dashboard.level1Dept")}
                                </span>
                                {selectedLevel1s.size > 0 && (
                                    <Badge variant="secondary" className="ml-auto h-5 px-1.5 text-xs">
                                        {selectedLevel1s.size}
                                    </Badge>
                                )}
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="start" className="w-56 max-h-80 overflow-y-auto">
                            <DropdownMenuLabel>{t("enterprise.dashboard.level1Dept")}</DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            <DropdownMenuCheckboxItem
                                checked={selectedLevel1s.size === 0}
                                onCheckedChange={(checked) => {
                                    if (checked) {
                                        setSelectedLevel1s(new Set())
                                        setSelectedLevel2s(new Set())
                                    }
                                }}
                            >
                                {t("enterprise.dashboard.allLevel1Depts")}
                            </DropdownMenuCheckboxItem>
                            <DropdownMenuSeparator />
                            {level1Departments.map((dept) => (
                                <DropdownMenuCheckboxItem
                                    key={dept.department_id}
                                    checked={selectedLevel1s.has(dept.department_id)}
                                    onCheckedChange={() => toggleLevel1(dept.department_id)}
                                >
                                    {dept.name || dept.department_id}
                                </DropdownMenuCheckboxItem>
                            ))}
                        </DropdownMenuContent>
                    </DropdownMenu>

                    {/* Level 2 Department Filter (multi-select) */}
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="outline" className="w-44 justify-start gap-1.5 bg-background/80" disabled={selectedLevel1s.size === 0}>
                                <Building2 className="w-4 h-4 shrink-0" />
                                <span className="truncate">
                                    {selectedLevel2s.size === 0
                                        ? t("enterprise.dashboard.allLevel2Depts")
                                        : t("enterprise.dashboard.level2Dept")}
                                </span>
                                {selectedLevel2s.size > 0 && (
                                    <Badge variant="secondary" className="ml-auto h-5 px-1.5 text-xs">
                                        {selectedLevel2s.size}
                                    </Badge>
                                )}
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="start" className="w-56 max-h-80 overflow-y-auto">
                            <DropdownMenuLabel>{t("enterprise.dashboard.level2Dept")}</DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            <DropdownMenuCheckboxItem
                                checked={selectedLevel2s.size === 0}
                                onCheckedChange={(checked) => {
                                    if (checked) setSelectedLevel2s(new Set())
                                }}
                            >
                                {t("enterprise.dashboard.allLevel2Depts")}
                            </DropdownMenuCheckboxItem>
                            <DropdownMenuSeparator />
                            {allLevel2Departments.map((dept) => (
                                <DropdownMenuCheckboxItem
                                    key={dept.department_id}
                                    checked={selectedLevel2s.has(dept.department_id)}
                                    onCheckedChange={() => toggleLevel2(dept.department_id)}
                                >
                                    {dept.name || dept.department_id}
                                </DropdownMenuCheckboxItem>
                            ))}
                        </DropdownMenuContent>
                    </DropdownMenu>

                    {/* Time Range Selector */}
                    <div className="flex flex-wrap items-center gap-2">
                        <Select value={timeRange} onValueChange={(v) => {
                            setTimeRange(v as TimeRange)
                            if (v !== "custom") {
                                setCustomDateRange(undefined)
                            }
                        }}>
                            <SelectTrigger className="w-40 bg-background/80">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="7d">{t("enterprise.dashboard.last7Days")}</SelectItem>
                                <SelectItem value="30d">{t("enterprise.dashboard.last30Days")}</SelectItem>
                                <SelectItem value="month">{t("enterprise.dashboard.thisMonth")}</SelectItem>
                                <SelectItem value="last_week">{t("enterprise.dashboard.lastWeek")}</SelectItem>
                                <SelectItem value="last_month">{t("enterprise.dashboard.lastMonth")}</SelectItem>
                                <SelectItem value="custom">{t("enterprise.dashboard.customRange")}</SelectItem>
                            </SelectContent>
                        </Select>

                        {/* Custom Date Range Picker */}
                        {timeRange === "custom" && (
                            <DateRangePicker
                                value={customDateRange}
                                onChange={setCustomDateRange}
                                placeholder={t("enterprise.dashboard.selectDateRange")}
                                className="w-64 bg-background/80"
                            />
                        )}

                        {activeFilterCount > 0 && (
                            <Button variant="ghost" size="sm" onClick={clearFilters}>
                                {t("common.clearFilters")}
                            </Button>
                        )}
                    </div>
                </div>
            </div>

            {/* Metric cards */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                <MetricCard
                    title={t("enterprise.dashboard.totalRequests")}
                    value={isLoading ? "..." : formatNumber(totals.requests)}
                    icon={BarChart2}
                    changePct={changes?.request_count_pct}
                />
                <MetricCard
                    title={t("enterprise.dashboard.totalAmount")}
                    value={isLoading ? "..." : formatAmount(totals.amount)}
                    icon={DollarSign}
                    changePct={changes?.used_amount_pct}
                />
                <MetricCard
                    title={t("enterprise.dashboard.totalTokens")}
                    value={isLoading ? "..." : formatNumber(totals.tokens)}
                    icon={Hash}
                    changePct={changes?.total_tokens_pct}
                />
                <MetricCard
                    title={t("enterprise.dashboard.activeUsersTotal")}
                    value={isLoading ? "..." : formatNumber(totals.activeUsers)}
                    icon={Users}
                />
            </div>

            <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
                <InsightTile
                    title={t("enterprise.dashboard.topDepartment")}
                    value={topDepartment?.department_name || t("common.noResult")}
                    detail={topDepartment
                        ? t("enterprise.dashboard.topDeptDetail", {
                            amount: formatAmount(topDepartment.used_amount),
                            requestCount: formatNumber(topDepartment.request_count),
                        })
                        : t("enterprise.dashboard.noHighlights")}
                    icon={Building2}
                />
                <InsightTile
                    title={t("enterprise.dashboard.topModel")}
                    value={topModel?.model || t("common.noResult")}
                    detail={topModel
                        ? t("enterprise.dashboard.topModelDetail", {
                            amount: formatAmount(topModel.used_amount),
                            share: topModel.percentage.toFixed(1),
                        })
                        : t("enterprise.dashboard.noHighlights")}
                    icon={Layers3}
                />
                <InsightTile
                    title={t("enterprise.dashboard.analysisSignal")}
                    value={totals.requests > 0
                        ? t("enterprise.dashboard.analysisValue", { amount: formatAmount(avgAmountPerRequest) })
                        : t("common.noResult")}
                    detail={totals.requests > 0
                        ? t("enterprise.dashboard.analysisDetail", { tokens: formatNumber(Math.round(avgTokensPerRequest)) })
                        : t("enterprise.dashboard.noHighlights")}
                    icon={Activity}
                />
            </div>

            {/* Main content: table + chart */}
            <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.3fr)_minmax(360px,0.7fr)]">
                {/* Department table */}
                <Card className="overflow-hidden border border-border/60 bg-background/92 shadow-lg shadow-slate-200/40 dark:shadow-none">
                    <CardHeader>
                        <div className="flex items-center justify-between">
                            <div>
                                <div className="flex items-center gap-2">
                                    <CardTitle className="text-lg">{t("enterprise.dashboard.departmentSummary")}</CardTitle>
                                    <Badge variant="outline" className="rounded-full px-2 py-0.5 text-xs font-normal text-muted-foreground">
                                        {t(`enterprise.dashboard.aggLevel.${aggLevel}` as never)}
                                    </Badge>
                                </div>
                                <p className="mt-1 text-sm text-muted-foreground">
                                    {t("enterprise.dashboard.departmentSummaryHint", { count: sortedDepartments.length })}
                                </p>
                            </div>
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button variant="outline" size="icon" className="h-8 w-8">
                                        <Settings2 className="h-4 w-4" />
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end" className="w-48">
                                    <DropdownMenuLabel>{t("enterprise.dashboard.columns")}</DropdownMenuLabel>
                                    <DropdownMenuSeparator />
                                    {DEPT_COLUMNS.map((col) => (
                                        <DropdownMenuCheckboxItem
                                            key={col.key}
                                            checked={visibleColumns.has(col.key)}
                                            onCheckedChange={() => toggleColumn(col.key)}
                                            disabled={col.key === "department_name"}
                                        >
                                            {t(col.labelKey as never)}
                                        </DropdownMenuCheckboxItem>
                                    ))}
                                </DropdownMenuContent>
                            </DropdownMenu>
                        </div>
                    </CardHeader>
                    <CardContent>
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="border-b text-muted-foreground">
                                        <th className="py-3 px-2 text-left font-medium">#</th>
                                        {visibleColumnConfigs.map((col) => (
                                            <th
                                                key={col.key}
                                                className={cn(
                                                    "py-3 px-2 font-medium cursor-pointer select-none hover:text-foreground transition-colors",
                                                    col.align === "right" ? "text-right" : "text-left",
                                                )}
                                                onClick={() => handleSort(col.key)}
                                            >
                                                <span className={cn(
                                                    "inline-flex items-center",
                                                    col.align === "right" && "justify-end"
                                                )}>
                                                    {t(col.labelKey as never)}
                                                    {renderSortIcon(col.key)}
                                                </span>
                                            </th>
                                        ))}
                                        <th className="text-right py-3 px-2 font-medium" />
                                    </tr>
                                </thead>
                                <tbody>
                                    {isLoading ? (
                                        <tr>
                                            <td colSpan={visibleColumnConfigs.length + 2} className="text-center py-8 text-muted-foreground">
                                                {t("common.loading")}
                                            </td>
                                        </tr>
                                    ) : sortedDepartments.length === 0 ? (
                                        <tr>
                                            <td colSpan={visibleColumnConfigs.length + 2} className="text-center py-8 text-muted-foreground">
                                                {t("common.noResult")}
                                            </td>
                                        </tr>
                                    ) : (
                                        sortedDepartments.map((dept, index) => (
                                            <tr
                                                key={dept.department_id}
                                                className="border-b last:border-0 hover:bg-muted/50 cursor-pointer transition-colors"
                                                onClick={() =>
                                                    navigate(`${ROUTES.ENTERPRISE_DEPARTMENT}/${dept.department_id}`)
                                                }
                                            >
                                                <td className="py-3 px-2">
                                                    <Badge variant={index < 3 ? "default" : "secondary"} className="min-w-8 justify-center">
                                                        {index + 1}
                                                    </Badge>
                                                </td>
                                                {visibleColumnConfigs.map((col) => (
                                                    <td
                                                        key={col.key}
                                                        className={cn(
                                                            "py-3 px-2",
                                                            col.align === "right" ? "text-right" : "text-left",
                                                        )}
                                                    >
                                                        {getCellValue(dept, col)}
                                                    </td>
                                                ))}
                                                <td className="py-3 px-2 text-right">
                                                    <Button variant="ghost" size="sm">
                                                        <ArrowRight className="w-4 h-4" />
                                                    </Button>
                                                </td>
                                            </tr>
                                        ))
                                    )}
                                </tbody>
                            </table>
                        </div>
                    </CardContent>
                </Card>

                {/* Pie chart */}
                <Card className="border border-border/60 bg-background/92 shadow-lg shadow-slate-200/40 dark:shadow-none">
                    <CardHeader>
                        <div>
                            <CardTitle className="text-lg">{t("enterprise.dashboard.departmentChart")}</CardTitle>
                            <p className="mt-1 text-sm text-muted-foreground">
                                {t("enterprise.dashboard.departmentChartHint")}
                            </p>
                        </div>
                    </CardHeader>
                    <CardContent>
                        {departments.length > 0 ? (
                            <DepartmentPieChart departments={departments} />
                        ) : (
                            <div className="flex h-[420px] items-center justify-center text-muted-foreground">
                                {isLoading ? t("common.loading") : t("common.noResult")}
                            </div>
                        )}
                    </CardContent>
                </Card>
            </div>

            {/* Model Distribution */}
            <Card className="border border-border/60 bg-background/92 shadow-lg shadow-slate-200/40 dark:shadow-none">
                <CardHeader>
                    <div>
                        <CardTitle className="text-lg">{t("enterprise.dashboard.modelDistribution")}</CardTitle>
                        <p className="mt-1 text-sm text-muted-foreground">
                            {t("enterprise.dashboard.modelDistributionHint")}
                        </p>
                    </div>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
                        <div>
                            {models.length > 0 ? (
                                <ModelDistributionChart models={models} />
                            ) : (
                                <div className="flex h-[420px] items-center justify-center text-muted-foreground">
                                    {isLoading ? t("common.loading") : t("common.noResult")}
                                </div>
                            )}
                        </div>
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="border-b text-muted-foreground">
                                        <th className="text-left py-2 px-2 font-medium">{t("enterprise.dashboard.model")}</th>
                                        <th className="text-right py-2 px-2 font-medium">{t("enterprise.dashboard.requests")}</th>
                                        <th className="text-right py-2 px-2 font-medium">{t("enterprise.dashboard.amount")}</th>
                                        <th className="text-right py-2 px-2 font-medium">{t("enterprise.dashboard.share")}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {models.slice(0, 10).map((m) => (
                                        <tr key={m.model} className="border-b last:border-0">
                                            <td className="py-2 px-2 font-medium text-xs truncate max-w-[180px]">
                                                {m.model}
                                            </td>
                                            <td className="py-2 px-2 text-right">{formatNumber(m.request_count)}</td>
                                            <td className="py-2 px-2 text-right">{formatAmount(m.used_amount)}</td>
                                            <td className="py-2 px-2 text-right">{m.percentage.toFixed(1)}%</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
