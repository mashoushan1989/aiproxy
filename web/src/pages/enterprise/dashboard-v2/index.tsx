import { type ComponentType, useMemo, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { BarChart3, Bot, DatabaseZap, Users } from "lucide-react"
import { type DateRange } from "react-day-picker"
import { useTranslation } from "react-i18next"
import { DateRangePicker } from "@/components/common/DateRangePicker"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
    enterpriseAnalyticsV2Api,
    type AnalyticsV2Params,
} from "@/api/enterprise-analytics-v2"
import type { DepartmentSummary, ModelDistributionItem, UserRankingItem } from "@/api/enterprise"
import { computeTimeRangeTs, formatAmount, formatNumber, formatRate, type TimeRange } from "@/lib/enterprise"

function MetricCard({
    title,
    value,
    detail,
    icon: Icon,
}: {
    title: string
    value: string
    detail: string
    icon: ComponentType<{ className?: string }>
}) {
    return (
        <Card className="border border-border/60">
            <CardContent className="p-5">
                <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0">
                        <p className="text-sm text-muted-foreground">{title}</p>
                        <p className="mt-1 truncate text-2xl font-semibold text-foreground">{value}</p>
                        <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
                    </div>
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                        <Icon className="h-5 w-5 text-foreground" />
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}

function EmptyRow({ colSpan, loading }: { colSpan: number; loading: boolean }) {
    const { t } = useTranslation()
    return (
        <TableRow>
            <TableCell colSpan={colSpan} className="h-28 text-center text-muted-foreground">
                {loading ? t("common.loading") : t("common.noResult")}
            </TableCell>
        </TableRow>
    )
}

export default function EnterpriseDashboardV2() {
    const { t } = useTranslation()
    const [timeRange, setTimeRange] = useState<TimeRange>("7d")
    const [customDateRange, setCustomDateRange] = useState<DateRange | undefined>()

    const { start, end } = useMemo(
        () => computeTimeRangeTs(timeRange, customDateRange),
        [customDateRange, timeRange],
    )
    const params = useMemo<AnalyticsV2Params>(
        () => ({ startTimestamp: start, endTimestamp: end, granularity: "daily", limit: 20 }),
        [end, start],
    )

    const departmentsQuery = useQuery({
        queryKey: ["enterprise", "analytics-v2", "departments", params],
        queryFn: () => enterpriseAnalyticsV2Api.getDepartmentSummaryV2(params),
    })
    const rankingQuery = useQuery({
        queryKey: ["enterprise", "analytics-v2", "ranking", params],
        queryFn: () => enterpriseAnalyticsV2Api.getUserRankingV2(params),
    })
    const modelsQuery = useQuery({
        queryKey: ["enterprise", "analytics-v2", "models", params],
        queryFn: () => enterpriseAnalyticsV2Api.getModelDistributionV2(params),
    })

    const departments = useMemo(
        () => departmentsQuery.data?.departments ?? [],
        [departmentsQuery.data?.departments],
    )
    const ranking = useMemo(
        () => rankingQuery.data?.ranking ?? [],
        [rankingQuery.data?.ranking],
    )
    const models = useMemo(
        () => modelsQuery.data?.distribution ?? [],
        [modelsQuery.data?.distribution],
    )
    const isFetching = departmentsQuery.isFetching || rankingQuery.isFetching || modelsQuery.isFetching

    const totals = useMemo(() => {
        return departments.reduce(
            (acc, dept) => ({
                requests: acc.requests + (dept.request_count || 0),
                amount: acc.amount + (dept.used_amount || 0),
                tokens: acc.tokens + (dept.total_tokens || 0),
                activeUsers: acc.activeUsers + (dept.active_users || 0),
            }),
            { requests: 0, amount: 0, tokens: 0, activeUsers: 0 },
        )
    }, [departments])

    const topDepartments = useMemo(
        () => [...departments].sort((a, b) => (b.used_amount || 0) - (a.used_amount || 0)).slice(0, 20),
        [departments],
    )
    const topUsers = useMemo(
        () => [...ranking].sort((a, b) => (a.rank || 0) - (b.rank || 0)).slice(0, 10),
        [ranking],
    )
    const topModels = useMemo(
        () => [...models].sort((a, b) => (b.used_amount || 0) - (a.used_amount || 0)).slice(0, 10),
        [models],
    )

    const rangeLabel = new Date(start * 1000).toLocaleDateString() + " - " + new Date(end * 1000).toLocaleDateString()

    return (
        <div className="space-y-6 p-6">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                <div>
                    <div className="flex flex-wrap items-center gap-2">
                        <h1 className="text-2xl font-semibold text-foreground">{t("enterprise.dashboard.title")}</h1>
                        <Badge variant="secondary">v2</Badge>
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">
                        {rangeLabel}
                        {isFetching ? " · " + t("common.loading") : ""}
                    </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                    <Select value={timeRange} onValueChange={(value) => setTimeRange(value as TimeRange)}>
                        <SelectTrigger className="w-[160px]">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="7d">{t("enterprise.dashboard.last7Days")}</SelectItem>
                            <SelectItem value="30d">{t("enterprise.dashboard.last30Days")}</SelectItem>
                            <SelectItem value="month">{t("enterprise.dashboard.thisMonth")}</SelectItem>
                            <SelectItem value="last_month">{t("enterprise.dashboard.lastMonth")}</SelectItem>
                            <SelectItem value="custom">{t("enterprise.dashboard.customRange")}</SelectItem>
                        </SelectContent>
                    </Select>
                    {timeRange === "custom" && (
                        <DateRangePicker value={customDateRange} onChange={setCustomDateRange} />
                    )}
                </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <MetricCard
                    title={t("enterprise.dashboard.requests")}
                    value={formatNumber(totals.requests)}
                    detail={`${departments.length} ${t("enterprise.dashboard.department")}`}
                    icon={BarChart3}
                />
                <MetricCard
                    title={t("enterprise.dashboard.amount")}
                    value={formatAmount(totals.amount)}
                    detail="Server scoped"
                    icon={DatabaseZap}
                />
                <MetricCard
                    title={t("enterprise.dashboard.tokens")}
                    value={formatNumber(totals.tokens)}
                    detail={`${models.length} ${t("enterprise.dashboard.model")}`}
                    icon={Bot}
                />
                <MetricCard
                    title={t("enterprise.dashboard.activeUsers")}
                    value={formatNumber(totals.activeUsers)}
                    detail={`${ranking.length} ${t("enterprise.ranking.userName")}`}
                    icon={Users}
                />
            </div>

            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(360px,0.55fr)]">
                <DepartmentTable departments={topDepartments} loading={departmentsQuery.isLoading} />
                <ModelTable models={topModels} loading={modelsQuery.isLoading} />
            </div>

            <UserRankingTable ranking={topUsers} loading={rankingQuery.isLoading} />
        </div>
    )
}

function DepartmentTable({ departments, loading }: { departments: DepartmentSummary[]; loading: boolean }) {
    const { t } = useTranslation()
    return (
        <Card className="border border-border/60">
            <CardHeader>
                <CardTitle className="text-base">{t("enterprise.dashboard.departmentSummary")}</CardTitle>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t("enterprise.dashboard.department")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.requests")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.amount")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.activeUsers")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.successRate")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {departments.length === 0 && <EmptyRow colSpan={5} loading={loading} />}
                        {departments.map((dept) => (
                            <TableRow key={dept.department_id}>
                                <TableCell className="max-w-[260px] truncate font-medium">
                                    {dept.department_name || dept.department_id}
                                </TableCell>
                                <TableCell className="text-right">{formatNumber(dept.request_count)}</TableCell>
                                <TableCell className="text-right">{formatAmount(dept.used_amount)}</TableCell>
                                <TableCell className="text-right">{formatNumber(dept.active_users)}</TableCell>
                                <TableCell className="text-right">{formatRate(dept.success_rate)}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </CardContent>
        </Card>
    )
}

function ModelTable({ models, loading }: { models: ModelDistributionItem[]; loading: boolean }) {
    const { t } = useTranslation()
    return (
        <Card className="border border-border/60">
            <CardHeader>
                <CardTitle className="text-base">{t("enterprise.dashboard.modelDistribution")}</CardTitle>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t("enterprise.dashboard.model")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.requests")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.share")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {models.length === 0 && <EmptyRow colSpan={3} loading={loading} />}
                        {models.map((model) => (
                            <TableRow key={model.model}>
                                <TableCell className="max-w-[220px] truncate font-medium">{model.model}</TableCell>
                                <TableCell className="text-right">{formatNumber(model.request_count)}</TableCell>
                                <TableCell className="text-right">{model.percentage.toFixed(1)}%</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </CardContent>
        </Card>
    )
}

function UserRankingTable({ ranking, loading }: { ranking: UserRankingItem[]; loading: boolean }) {
    const { t } = useTranslation()
    return (
        <Card className="border border-border/60">
            <CardHeader>
                <CardTitle className="text-base">{t("enterprise.ranking.title")}</CardTitle>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-16">{t("enterprise.ranking.rank")}</TableHead>
                            <TableHead>{t("enterprise.ranking.userName")}</TableHead>
                            <TableHead>{t("enterprise.dashboard.department")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.requests")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.amount")}</TableHead>
                            <TableHead className="text-right">{t("enterprise.dashboard.tokens")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {ranking.length === 0 && <EmptyRow colSpan={6} loading={loading} />}
                        {ranking.map((user) => (
                            <TableRow key={`${user.rank}-${user.group_id}`}>
                                <TableCell>{user.rank}</TableCell>
                                <TableCell className="max-w-[240px] truncate font-medium">
                                    {user.user_name || user.group_id}
                                </TableCell>
                                <TableCell className="max-w-[240px] truncate">
                                    {user.department_name || user.department_id || "-"}
                                </TableCell>
                                <TableCell className="text-right">{formatNumber(user.request_count)}</TableCell>
                                <TableCell className="text-right">{formatAmount(user.used_amount)}</TableCell>
                                <TableCell className="text-right">{formatNumber(user.total_tokens)}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </CardContent>
        </Card>
    )
}
