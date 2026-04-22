import { useEffect, useMemo, useRef } from "react"
import * as echarts from "echarts"
import type { CustomReportResponse } from "@/api/enterprise"
import { useDarkMode, getEChartsTheme } from "@/lib/enterprise"
import {
    type AxisMode,
    type ChartType,
    CHART_COLORS,
    PERCENTAGE_FIELDS,
    COST_FIELDS,
    TIME_DIMENSIONS,
    getLabel,
    formatDimValue,
    formatCellValue,
    recommendChartType,
    sortRowsByTime,
} from "./types"

/** Estimate legend rows and compute grid top offset so legend never overlaps chart */
function legendGridTop(itemCount: number, containerWidth = 800): number {
    // Each legend item is roughly 80–120px wide; estimate items per row
    const usableWidth = containerWidth * 0.9
    const avgItemWidth = 100
    const itemsPerRow = Math.max(Math.floor(usableWidth / avgItemWidth), 1)
    const rows = Math.ceil(itemCount / itemsPerRow)
    // Each row ~22px, plus 8px padding
    return Math.max(rows * 22 + 8, 30)
}

/** Build a wrapping legend config with auto scroll for many items */
function wrapLegend(data: string[], textColor: string): echarts.EChartsOption["legend"] {
    const useScroll = data.length > 15
    return {
        data,
        textStyle: { color: textColor, fontSize: 11 },
        type: useScroll ? "scroll" : ("plain" as const),
        width: "90%",
        left: "center",
        top: 0,
        itemWidth: 14,
        itemHeight: 10,
        ...(useScroll ? { pageTextStyle: { color: textColor } } : {}),
    }
}

/** Compute chart container height based on legend count */
function computeChartHeight(legendCount: number, fullscreen: boolean): number {
    if (fullscreen) return 0 // CSS-controlled
    const legendRows = Math.ceil(Math.min(legendCount, 15) / 5)
    const legendHeight = legendRows * 22 + 8
    const minChartArea = legendCount > 8 ? 420 : 380
    const maxHeight = 760
    return Math.min(legendHeight + minChartArea, maxHeight)
}

/** Compute rotation and interval for X-axis labels */
function xAxisLabelConfig(labels: string[]): { rotate: number; interval: number; fontSize: number } {
    const count = labels.length
    const maxLen = labels.reduce((mx, l) => Math.max(mx, l.length), 0)
    if (count <= 7 && maxLen <= 10) return { rotate: 0, interval: 0, fontSize: 11 }
    if (count <= 15) return { rotate: 30, interval: 0, fontSize: 10 }
    if (count <= 31) return { rotate: 45, interval: 0, fontSize: 10 }
    return { rotate: 45, interval: Math.floor(count / 25), fontSize: 9 }
}

// Separate dimensions into primary (X-axis) and secondary (series grouping).
// Time dimensions are preferred as primary; if none, use the first dimension.
function splitDimensions(dimensions: string[]): { primary: string; secondary: string | null } {
    if (dimensions.length <= 1) {
        return { primary: dimensions[0] ?? "", secondary: null }
    }
    const timeDim = dimensions.find((d) => TIME_DIMENSIONS.has(d))
    if (timeDim) {
        const other = dimensions.find((d) => d !== timeDim) ?? null
        return { primary: timeDim, secondary: other }
    }
    return { primary: dimensions[0], secondary: dimensions[1] ?? null }
}

// Build formatted label for a single dimension value
function dimLabel(dimKey: string, row: Record<string, unknown>): string {
    return formatDimValue(dimKey, row[dimKey])
}

function formatTooltipChartValue(measureKey: string, value: number | null | undefined): string {
    if (value == null || Number.isNaN(Number(value))) return "-"
    const n = Number(value)
    if (COST_FIELDS.has(measureKey)) return `¥${n.toFixed(2)}`
    return formatCellValue(measureKey, n)
}

function buildDataZoom(pointCount: number): echarts.EChartsOption["dataZoom"] | undefined {
    if (pointCount <= 12) return undefined
    return [
        {
            type: "inside",
            zoomOnMouseWheel: "shift",
            moveOnMouseMove: true,
            moveOnMouseWheel: true,
            filterMode: "none",
        },
        {
            type: "slider",
            height: 18,
            bottom: 0,
            brushSelect: false,
            filterMode: "none",
        },
    ]
}

function measureUnitFamily(measureKey: string): "percent" | "cost" | "duration" | "speed" | "ratio" | "count" {
    if (PERCENTAGE_FIELDS.has(measureKey)) return "percent"
    if (COST_FIELDS.has(measureKey)) return "cost"
    if (measureKey.includes("latency") || measureKey.includes("ttfb") || measureKey.includes("time_ms")) return "duration"
    if (measureKey === "tokens_per_second" || measureKey === "output_speed") return "speed"
    if (measureKey === "output_input_ratio") return "ratio"
    return "count"
}

function normalizeAxisAssignment(
    measures: string[],
    axisMode: AxisMode,
    rightAxisMeasures: string[],
): { useDualAxis: boolean; rightAxisSet: Set<string> } {
    if (measures.length < 2 || axisMode === "single") {
        return { useDualAxis: false, rightAxisSet: new Set() }
    }

    if (axisMode === "custom") {
        const rightAxisSet = new Set(rightAxisMeasures.filter((measure) => measures.includes(measure)))
        if (rightAxisSet.size === 0 || rightAxisSet.size === measures.length) {
            rightAxisSet.clear()
            rightAxisSet.add(measures[measures.length - 1])
        }
        return { useDualAxis: true, rightAxisSet }
    }

    const percentageMeasures = measures.filter((measure) => PERCENTAGE_FIELDS.has(measure))
    if (percentageMeasures.length > 0 && percentageMeasures.length < measures.length) {
        return { useDualAxis: true, rightAxisSet: new Set(percentageMeasures) }
    }

    if (measures.length === 2 && measureUnitFamily(measures[0]) !== measureUnitFamily(measures[1])) {
        return { useDualAxis: true, rightAxisSet: new Set([measures[1]]) }
    }

    return { useDualAxis: false, rightAxisSet: new Set() }
}

function buildAxisName(measures: string[], lang: string): string {
    if (measures.length === 0) return lang.startsWith("zh") ? "数值" : "Value"
    if (measures.length === 1) return getLabel(measures[0], lang)
    if (measures.every((measure) => PERCENTAGE_FIELDS.has(measure))) return "%"
    if (measures.every((measure) => COST_FIELDS.has(measure))) return lang.startsWith("zh") ? "费用" : "Cost"
    if (measures.every((measure) => measureUnitFamily(measure) === "duration")) return lang.startsWith("zh") ? "时延 (ms)" : "Latency (ms)"
    return lang.startsWith("zh") ? "混合指标" : "Metrics"
}

function buildValueAxis(
    measures: string[],
    lang: string,
    theme: ReturnType<typeof getEChartsTheme>,
    position: "left" | "right",
): echarts.YAXisComponentOption {
    const percentOnly = measures.length > 0 && measures.every((measure) => PERCENTAGE_FIELDS.has(measure))
    return {
        type: "value",
        position,
        name: buildAxisName(measures, lang),
        nameTextStyle: { color: theme.subTextColor },
        axisLabel: {
            color: theme.subTextColor,
            formatter: percentOnly ? "{value}%" : undefined,
        },
        ...(percentOnly ? { min: 0, max: 100 } : {}),
        splitLine: position === "right"
            ? { show: !percentOnly, lineStyle: { color: theme.splitLineColor, opacity: 0.3 } }
            : { lineStyle: { color: theme.splitLineColor } },
    } as echarts.YAXisComponentOption
}

export function ReportChart({
    data,
    dimensions,
    measures,
    chartType,
    axisMode = "auto",
    rightAxisMeasures = [],
    lang,
    fullscreen = false,
}: {
    data: CustomReportResponse
    dimensions: string[]
    measures: string[]
    chartType: ChartType
    axisMode?: AxisMode
    rightAxisMeasures?: string[]
    lang: string
    fullscreen?: boolean
}) {
    const chartRef = useRef<HTMLDivElement>(null)
    const instance = useRef<echarts.ECharts | null>(null)
    const isDark = useDarkMode()

    useEffect(() => {
        if (!chartRef.current || data.rows.length === 0) return

        if (!instance.current) {
            instance.current = echarts.init(chartRef.current)
        }

        const theme = getEChartsTheme(isDark)
        const resolvedType = chartType === "auto" ? recommendChartType(dimensions, measures) : chartType

        const { primary, secondary } = splitDimensions(dimensions)

        // Sort rows by time dimension if present
        const rows = sortRowsByTime(data.rows, dimensions)

        // Build formatted labels from all dimensions (for single-dim charts)
        const labels = rows.map((row) =>
            dimensions.map((d) => formatDimValue(d, row[d])).join(" / "),
        )

        // Only numeric measures
        const numericMeasures = measures.filter((m) => {
            const first = rows[0]?.[m]
            return first !== undefined && !Number.isNaN(Number(first))
        })
        const { useDualAxis, rightAxisSet } = normalizeAxisAssignment(numericMeasures, axisMode, rightAxisMeasures)
        const leftAxisMeasures = numericMeasures.filter((measure) => !rightAxisSet.has(measure))
        const rightAxisMeasureList = numericMeasures.filter((measure) => rightAxisSet.has(measure))

        let option: echarts.EChartsOption

        switch (resolvedType) {
            case "pie": {
                const measure = numericMeasures[0]
                if (!measure) return
                const PIE_TOP_N = 15
                const pieData = rows.slice(0, PIE_TOP_N).map((row, i) => ({
                    name: labels[i],
                    value: Number(row[measure] ?? 0),
                    itemStyle: { color: CHART_COLORS[i % CHART_COLORS.length] },
                }))
                if (rows.length > PIE_TOP_N) {
                    let othersSum = 0
                    for (let i = PIE_TOP_N; i < rows.length; i++) {
                        othersSum += Number(rows[i][measure] ?? 0)
                    }
                    if (othersSum > 0) {
                        pieData.push({
                            name: lang.startsWith("zh") ? "其他" : "Others",
                            value: othersSum,
                            itemStyle: { color: "#CBD5E1" },
                        })
                    }
                }
                const isCost = COST_FIELDS.has(measure)
                option = {
                    tooltip: {
                        trigger: "item",
                        formatter: (p: unknown) => {
                            const params = p as { name: string; value: number; percent: number }
                            return `${params.name}: ${isCost ? formatCellValue(measure, params.value) : params.value} (${params.percent}%)`
                        },
                    },
                    series: [{
                        type: "pie",
                        radius: ["40%", "70%"],
                        data: pieData,
                        label: { show: true, formatter: "{b}\n{d}%", color: theme.textColor },
                    }],
                }
                break
            }

            case "heatmap": {
                const dim0 = dimensions[0]
                const dim1 = dimensions[1]
                const dim0Values = [...new Set(rows.map((r) => formatDimValue(dim0, r[dim0])))]
                const dim1Values = [...new Set(rows.map((r) => formatDimValue(dim1, r[dim1])))]
                const measure = numericMeasures[0]
                if (!measure) return

                const heatData: [number, number, number][] = []
                for (const row of rows) {
                    const x = dim0Values.indexOf(formatDimValue(dim0, row[dim0]))
                    const y = dim1Values.indexOf(formatDimValue(dim1, row[dim1]))
                    if (x >= 0 && y >= 0) {
                        heatData.push([x, y, Number(row[measure] ?? 0)])
                    }
                }

                let minVal = Infinity
                let maxVal = -Infinity
                for (const d of heatData) {
                    if (d[2] < minVal) minVal = d[2]
                    if (d[2] > maxVal) maxVal = d[2]
                }

                option = {
                    tooltip: {
                        position: "top",
                        formatter: (p: unknown) => {
                            const params = p as { value: [number, number, number] }
                            return `${dim0Values[params.value[0]]} × ${dim1Values[params.value[1]]}<br/>${getLabel(measure, lang)}: ${formatCellValue(measure, params.value[2])}`
                        },
                    },
                    grid: { left: "15%", right: "10%", bottom: "15%", top: "5%" },
                    xAxis: {
                        type: "category",
                        data: dim0Values,
                        axisLabel: { rotate: 30, fontSize: 10, color: theme.subTextColor },
                        splitArea: { show: true },
                    },
                    yAxis: {
                        type: "category",
                        data: dim1Values,
                        axisLabel: { fontSize: 10, color: theme.subTextColor },
                        splitArea: { show: true },
                    },
                    visualMap: {
                        min: minVal,
                        max: maxVal,
                        calculable: true,
                        orient: "horizontal",
                        left: "center",
                        bottom: "0",
                        inRange: { color: ["#f0f0ff", "#6A6DE6", "#3a0ca3"] },
                        textStyle: { color: theme.subTextColor },
                    },
                    series: [{
                        type: "heatmap",
                        data: heatData,
                        label: { show: heatData.length <= 100, fontSize: 9 },
                        emphasis: { itemStyle: { shadowBlur: 10, shadowColor: "rgba(0,0,0,0.3)" } },
                    }],
                }
                break
            }

            case "treemap": {
                const measure = numericMeasures[0]
                if (!measure) return

                if (dimensions.length >= 2) {
                    const groups = new Map<string, { name: string; value: number; children: { name: string; value: number }[] }>()
                    for (const row of rows) {
                        const parent = dimLabel(dimensions[0], row)
                        const child = dimLabel(dimensions[1], row)
                        const val = Number(row[measure] ?? 0)
                        if (!groups.has(parent)) {
                            groups.set(parent, { name: parent, value: 0, children: [] })
                        }
                        const g = groups.get(parent)!
                        g.value += val
                        g.children.push({ name: child, value: val })
                    }
                    option = {
                        tooltip: { formatter: (p: unknown) => {
                            const params = p as { name: string; value: number }
                            return `${params.name}: ${formatCellValue(measure, params.value)}`
                        }},
                        series: [{
                            type: "treemap",
                            data: Array.from(groups.values()),
                            label: { show: true, formatter: "{b}" },
                            levels: [
                                { itemStyle: { borderColor: theme.borderColor, borderWidth: 2 } },
                                { itemStyle: { borderColor: theme.borderColor, borderWidth: 1 }, label: { fontSize: 10 } },
                            ],
                        }],
                    }
                } else {
                    const TREE_TOP_N = 30
                    const treeData = rows.slice(0, TREE_TOP_N).map((row, i) => ({
                        name: labels[i],
                        value: Number(row[measure] ?? 0),
                    }))
                    if (rows.length > TREE_TOP_N) {
                        let othersSum = 0
                        for (let i = TREE_TOP_N; i < rows.length; i++) {
                            othersSum += Number(rows[i][measure] ?? 0)
                        }
                        if (othersSum > 0) {
                            treeData.push({
                                name: lang.startsWith("zh") ? "其他" : "Others",
                                value: othersSum,
                            })
                        }
                    }
                    option = {
                        tooltip: {},
                        series: [{
                            type: "treemap",
                            data: treeData,
                            label: { show: true, formatter: "{b}\n{c}" },
                        }],
                    }
                }
                break
            }

            case "radar": {
                const maxValues = numericMeasures.map((m) => {
                    let mx = 1
                    for (const r of rows) {
                        const v = Number(r[m] ?? 0)
                        if (v > mx) mx = v
                    }
                    return mx
                })
                const indicator = numericMeasures.map((m, i) => ({
                    name: getLabel(m, lang),
                    max: maxValues[i],
                }))

                const radarRows = rows.slice(0, 8)
                const radarLegendTop = legendGridTop(radarRows.length)
                option = {
                    tooltip: {},
                    legend: wrapLegend(radarRows.map((_, i) => labels[i]), theme.textColor),
                    radar: { indicator, shape: "polygon", center: ["50%", `${50 + radarLegendTop / 8}%`] },
                    series: [{
                        type: "radar",
                        data: radarRows.map((row, i) => ({
                            name: labels[i],
                            value: numericMeasures.map((m) => Number(row[m] ?? 0)),
                            lineStyle: { color: CHART_COLORS[i % CHART_COLORS.length] },
                            itemStyle: { color: CHART_COLORS[i % CHART_COLORS.length] },
                            areaStyle: { color: CHART_COLORS[i % CHART_COLORS.length], opacity: 0.1 },
                        })),
                    }],
                }
                break
            }

            default: {
                // bar, stacked_bar, line, area
                const isStacked = resolvedType === "stacked_bar"
                const isArea = resolvedType === "area"
                const seriesType = (resolvedType === "stacked_bar" || resolvedType === "bar") ? "bar" : "line"

                if (secondary) {
                    // ── Multi-dimension: primary = X-axis, secondary = series grouping ──
                    const primaryValues = [...new Set(rows.map((r) => formatDimValue(primary, r[primary])))]
                    const rawSecondaryValues = [...new Set(rows.map((r) => formatDimValue(secondary, r[secondary])))]
                    const sortMeasure = numericMeasures[0]
                    const maxVisibleSeries = numericMeasures.length > 1 ? 6 : 10
                    const secondaryTotals = new Map<string, number>()
                    if (sortMeasure) {
                        for (const row of rows) {
                            const sKey = formatDimValue(secondary, row[secondary])
                            secondaryTotals.set(
                                sKey,
                                (secondaryTotals.get(sKey) ?? 0) + Number(row[sortMeasure] ?? 0),
                            )
                        }
                    }
                    const sortedSecondaryValues = [...rawSecondaryValues].sort(
                        (a, b) => (secondaryTotals.get(b) ?? 0) - (secondaryTotals.get(a) ?? 0),
                    )
                    const hiddenSecondaryValues = new Set(
                        sortedSecondaryValues.length > maxVisibleSeries
                            ? sortedSecondaryValues.slice(maxVisibleSeries)
                            : [],
                    )
                    const othersLabel = lang.startsWith("zh") ? "其他" : "Others"
                    const secondaryValues = hiddenSecondaryValues.size > 0
                        ? [...sortedSecondaryValues.slice(0, maxVisibleSeries), othersLabel]
                        : sortedSecondaryValues

                    // Build lookup: primaryLabel -> secondaryLabel -> { measure: value }
                    const lookup = new Map<string, Map<string, Record<string, number>>>()
                    for (const row of rows) {
                        const pKey = formatDimValue(primary, row[primary])
                        const sKey = formatDimValue(secondary, row[secondary])
                        if (!lookup.has(pKey)) lookup.set(pKey, new Map())
                        const sMap = lookup.get(pKey)!
                        if (!sMap.has(sKey)) sMap.set(sKey, {})
                        const rec = sMap.get(sKey)!
                        for (const m of numericMeasures) {
                            rec[m] = Number(row[m] ?? 0)
                        }
                    }

                    const labelCfg = xAxisLabelConfig(primaryValues)

                    // Build series: for each (secondaryValue, measure) pair
                    // If only 1 measure, series name = secondaryValue (cleaner legend)
                    // If multiple measures, series name = "secondaryValue - measureLabel"
                    const allSeries: echarts.EChartsOption["series"] = []
                    const legendData: string[] = []
                    const seriesMeasureMap = new Map<string, string>()
                    let colorIdx = 0
                    const getSeriesData = (seriesKey: string, measureKey: string) => {
                        if (seriesKey !== othersLabel) {
                            return primaryValues.map((pVal) => lookup.get(pVal)?.get(seriesKey)?.[measureKey] ?? 0)
                        }
                        return primaryValues.map((pVal) => {
                            const pLookup = lookup.get(pVal)
                            if (!pLookup) return 0
                            let total = 0
                            for (const hiddenKey of hiddenSecondaryValues) {
                                total += pLookup.get(hiddenKey)?.[measureKey] ?? 0
                            }
                            return total
                        })
                    }

                    if (numericMeasures.length === 1) {
                        const m = numericMeasures[0]
                        for (const sVal of secondaryValues) {
                            legendData.push(sVal)
                            seriesMeasureMap.set(sVal, m)
                                allSeries.push({
                                    name: sVal,
                                    type: seriesType,
                                    yAxisIndex: useDualAxis && rightAxisSet.has(m) ? 1 : 0,
                                    data: getSeriesData(sVal, m),
                                    itemStyle: { color: CHART_COLORS[colorIdx % CHART_COLORS.length] },
                                smooth: seriesType === "line",
                                stack: isStacked ? "total" : undefined,
                                barMaxWidth: 24,
                                emphasis: { focus: "series", blurScope: "coordinateSystem" },
                                ...(isArea ? { areaStyle: { opacity: 0.3 } } : {}),
                            })
                            colorIdx++
                        }
                    } else {
                        for (const sVal of secondaryValues) {
                            for (const m of numericMeasures) {
                                const seriesName = `${sVal} - ${getLabel(m, lang)}`
                                legendData.push(seriesName)
                                seriesMeasureMap.set(seriesName, m)
                                allSeries.push({
                                    name: seriesName,
                                    type: seriesType,
                                    yAxisIndex: useDualAxis && rightAxisSet.has(m) ? 1 : 0,
                                    data: getSeriesData(sVal, m),
                                    itemStyle: { color: CHART_COLORS[colorIdx % CHART_COLORS.length] },
                                    smooth: seriesType === "line",
                                    stack: isStacked ? sVal : undefined,
                                    barMaxWidth: 24,
                                    emphasis: { focus: "series", blurScope: "coordinateSystem" },
                                    ...(isArea ? { areaStyle: { opacity: 0.3 } } : {}),
                                })
                                colorIdx++
                            }
                        }
                    }

                    const gridTop = legendGridTop(legendData.length)

                    option = {
                        tooltip: {
                            trigger: "item",
                            confine: true,
                            backgroundColor: isDark ? "rgba(30,30,40,0.95)" : "rgba(255,255,255,0.96)",
                            borderColor: isDark ? "#444" : "#e5e7eb",
                            textStyle: { color: isDark ? "#e5e7eb" : "#374151", fontSize: 12 },
                            formatter: (param: unknown) => {
                                const p = param as {
                                    name: string
                                    seriesName: string
                                    value: number | null
                                    marker: string
                                }
                                const measureKey = seriesMeasureMap.get(p.seriesName) ?? numericMeasures[0] ?? ""
                                return [
                                    `<div style="font-weight:600;margin-bottom:6px">${p.name}</div>`,
                                    `<div>${p.marker} ${p.seriesName}: <span style="font-weight:600">${formatTooltipChartValue(measureKey, p.value)}</span></div>`,
                                ].join("")
                            },
                        },
                        legend: wrapLegend(legendData, theme.textColor),
                        grid: { left: "3%", right: useDualAxis ? "9%" : "4%", bottom: primaryValues.length > 12 ? 56 : "6%", top: gridTop, containLabel: true },
                        dataZoom: buildDataZoom(primaryValues.length),
                        xAxis: {
                            type: "category",
                            data: primaryValues,
                            axisLabel: { rotate: labelCfg.rotate, interval: labelCfg.interval, fontSize: labelCfg.fontSize, color: theme.subTextColor },
                        },
                        yAxis: useDualAxis
                            ? [
                                buildValueAxis(leftAxisMeasures, lang, theme, "left"),
                                buildValueAxis(rightAxisMeasureList, lang, theme, "right"),
                            ]
                            : buildValueAxis(leftAxisMeasures, lang, theme, "left"),
                        series: allSeries,
                    }
                } else {
                    // ── Single dimension: each measure is a series ──
                    const xLabels = rows.slice(0, 50).map((row) => formatDimValue(primary, row[primary]))
                    const labelCfg = xAxisLabelConfig(xLabels)

                    const singleGridTop = legendGridTop(numericMeasures.length)

                    // Build a measure-aware tooltip so cost fields show ¥ prefix,
                    // percentages show %, and null values display as "-".
                    const measureTooltipFormatter = (params: unknown) => {
                        const list = Array.isArray(params) ? params : [params]
                        type TParam = { axisValueLabel: string; seriesName: string; value: number | null; marker: string }
                        const first = list[0] as TParam | undefined
                        if (!first) return ""
                        let html = `<div style="font-weight:600;margin-bottom:4px">${first.axisValueLabel}</div>`
                        for (const p of list as TParam[]) {
                            const mKey = numericMeasures.find((k) => getLabel(k, lang) === p.seriesName) ?? ""
                            const formatted = p.value == null ? "-" : formatCellValue(mKey, p.value)
                            html += `<div>${p.marker} ${p.seriesName}: ${formatted}</div>`
                        }
                        return html
                    }

                    option = {
                        tooltip: {
                            trigger: "axis",
                            axisPointer: { type: "shadow" },
                            backgroundColor: isDark ? "rgba(30,30,40,0.95)" : "rgba(255,255,255,0.95)",
                            borderColor: isDark ? "#444" : "#e5e7eb",
                            textStyle: { color: isDark ? "#e5e7eb" : "#374151", fontSize: 12 },
                            formatter: measureTooltipFormatter,
                        },
                        legend: wrapLegend(numericMeasures.map((m) => getLabel(m, lang)), theme.textColor),
                        grid: { left: "3%", right: useDualAxis ? "9%" : "4%", bottom: xLabels.length > 12 ? 56 : "3%", top: singleGridTop, containLabel: true },
                        dataZoom: buildDataZoom(xLabels.length),
                        xAxis: {
                            type: "category",
                            data: xLabels,
                            axisLabel: { rotate: labelCfg.rotate, interval: labelCfg.interval, fontSize: labelCfg.fontSize, color: theme.subTextColor },
                        },
                        yAxis: useDualAxis
                            ? [
                                buildValueAxis(leftAxisMeasures, lang, theme, "left"),
                                buildValueAxis(rightAxisMeasureList, lang, theme, "right"),
                            ]
                            : buildValueAxis(leftAxisMeasures, lang, theme, "left"),
                        series: numericMeasures.map((m, i) => {
                            // Preserve null (from safeDivide) so echarts skips the point
                            // instead of plotting 0 for "no data" ratios.
                            const seriesData = rows.slice(0, 50).map((row) => {
                                const v = row[m]
                                return v == null ? null : Number(v)
                            })
                            // Add average markLine for single-dimension line/bar charts
                            const markLine = (seriesType === "line" || seriesType === "bar") && !isStacked && numericMeasures.length <= 2
                                ? {
                                    data: [{ type: "average" as const, name: "Avg" }],
                                    lineStyle: { color: CHART_COLORS[i % CHART_COLORS.length], type: "dashed" as const, opacity: 0.5 },
                                    label: { show: true, formatter: `{c}`, fontSize: 10, color: theme.subTextColor },
                                    silent: true,
                                }
                                : undefined
                            return {
                                name: getLabel(m, lang),
                                type: seriesType,
                                yAxisIndex: useDualAxis && rightAxisSet.has(m) ? 1 : 0,
                                data: seriesData,
                                itemStyle: { color: CHART_COLORS[i % CHART_COLORS.length] },
                                smooth: seriesType === "line",
                                barMaxWidth: 28,
                                emphasis: { focus: "series", blurScope: "coordinateSystem" },
                                ...(isStacked ? { stack: "total" } : {}),
                                ...(isArea ? { areaStyle: { opacity: 0.3 } } : {}),
                                ...(markLine ? { markLine } : {}),
                            }
                        }),
                    }
                }
                break
            }
        }

        instance.current.setOption(option, true)

        const handleResize = () => instance.current?.resize()
        window.addEventListener("resize", handleResize)
        return () => {
            window.removeEventListener("resize", handleResize)
        }
    }, [data, dimensions, measures, chartType, axisMode, rightAxisMeasures, lang, isDark])

    // Estimate legend count for dynamic height
    const legendCount = useMemo(() => {
        const resolvedType = chartType === "auto" ? recommendChartType(dimensions, measures) : chartType
        if (resolvedType === "pie" || resolvedType === "treemap") return 0 // no legend-driven height
        if (resolvedType === "heatmap") return 0

        const { secondary } = splitDimensions(dimensions)
        if (secondary) {
            const secondaryValues = new Set(data.rows.map((r) => formatDimValue(secondary, r[secondary])))
            return measures.length === 1 ? secondaryValues.size : secondaryValues.size * measures.length
        }
        return measures.length
    }, [data, dimensions, measures, chartType])

    const dynamicHeight = computeChartHeight(legendCount, fullscreen)

    // Resize when fullscreen or dynamic height changes
    useEffect(() => {
        const timer = setTimeout(() => instance.current?.resize(), 50)
        return () => clearTimeout(timer)
    }, [fullscreen, dynamicHeight])

    // Clean up on unmount
    useEffect(() => {
        return () => {
            instance.current?.dispose()
            instance.current = null
        }
    }, [])

    return (
        <div
            ref={chartRef}
            className="w-full"
            style={{ height: fullscreen ? "calc(100vh - 120px)" : `${dynamicHeight}px` }}
        />
    )
}
