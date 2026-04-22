import { useTranslation } from "react-i18next"
import type { CustomReportResponse } from "@/api/enterprise"
import type { AxisMode, ChartType } from "./types"
import { ReportChart } from "./ReportChart"
import { ReportTable } from "./ReportTable"

export function SplitView({
    data,
    dimensions,
    measures,
    chartType,
    axisMode,
    rightAxisMeasures,
    lang,
    sortBy,
    sortOrder,
    onSort,
}: {
    data: CustomReportResponse
    dimensions: string[]
    measures: string[]
    chartType: ChartType
    axisMode: AxisMode
    rightAxisMeasures: string[]
    lang: string
    sortBy: string | undefined
    sortOrder: "asc" | "desc"
    onSort: (key: string, order: "asc" | "desc") => void
}) {
    const { t } = useTranslation()

    return (
        <div className="grid grid-cols-1 gap-0 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)] lg:divide-x">
            {/* Chart */}
            <div className="space-y-4 p-5">
                <div className="rounded-2xl border border-border/60 bg-muted/20 px-4 py-3">
                    <p className="text-sm font-medium text-foreground">{t("enterprise.customReport.chartFocusTitle", "Chart Focus")}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                        {t("enterprise.customReport.chartInspectorHint", "Hover a specific bar or point to inspect a single metric precisely.")}
                    </p>
                </div>
                <ReportChart
                    data={data}
                    dimensions={dimensions}
                    measures={measures}
                    chartType={chartType}
                    axisMode={axisMode}
                    rightAxisMeasures={rightAxisMeasures}
                    lang={lang}
                />
            </div>

            {/* Table */}
            <div className="border-t lg:border-t-0">
                <ReportTable
                    data={data}
                    dimensions={dimensions}
                    sortBy={sortBy}
                    sortOrder={sortOrder}
                    onSort={onSort}
                />
            </div>
        </div>
    )
}
