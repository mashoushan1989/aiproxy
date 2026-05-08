import { COST_FIELDS, TIME_DIMENSIONS, sortRowsByTime } from "./types"

export type SortOrder = "asc" | "desc"

export function getEffectiveSortBy(
    sortBy: string | undefined,
    dimensions: string[],
    measures: string[],
): string | undefined {
    if (sortBy) return sortBy
    if (dimensions.some((dimension) => TIME_DIMENSIONS.has(dimension))) return undefined
    if (measures.includes("used_amount")) return "used_amount"
    return measures.find((measure) => COST_FIELDS.has(measure)) ?? measures[0]
}

export function getRowsForReportView(
    rows: Record<string, unknown>[],
    dimensions: string[],
    sortBy: string | undefined,
): Record<string, unknown>[] {
    if (sortBy) return rows
    return sortRowsByTime(rows, dimensions)
}
