import { ArrowDownAZ, ArrowUpAZ } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { getLabel } from "./types"
import type { SortOrder } from "./reportSorting"

export function ReportSortControl({
    columns,
    sortBy,
    sortOrder,
    lang,
    onSortChange,
}: {
    columns: { key: string }[]
    sortBy: string | undefined
    sortOrder: SortOrder
    lang: string
    onSortChange: (key: string, order: SortOrder) => void
}) {
    const defaultKey = "__default__"
    const activeKey = sortBy ?? defaultKey
    const firstColumnKey = columns[0]?.key ?? ""
    const sortableKey = sortBy ?? firstColumnKey

    return (
        <div className="flex items-center gap-1 rounded-xl border border-border/70 bg-background px-2 py-1">
            <span className="hidden text-xs text-muted-foreground sm:inline">
                {lang.startsWith("zh") ? "排序" : "Sort"}
            </span>
            <Select
                value={activeKey}
                onValueChange={(value) => {
                    if (value !== defaultKey) onSortChange(value, sortOrder)
                }}
            >
                <SelectTrigger className="h-8 w-[136px] border-0 px-2 shadow-none focus:ring-0 sm:w-[168px]">
                    <SelectValue placeholder={lang.startsWith("zh") ? "默认顺序" : "Default order"} />
                </SelectTrigger>
                <SelectContent>
                    {!sortBy && (
                        <SelectItem value={defaultKey}>
                            {lang.startsWith("zh") ? "默认顺序" : "Default order"}
                        </SelectItem>
                    )}
                    {columns.map((column) => (
                        <SelectItem key={column.key} value={column.key}>
                            {getLabel(column.key, lang)}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
            <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={() => onSortChange(sortableKey, sortOrder === "desc" ? "asc" : "desc")}
                disabled={!sortableKey}
                title={sortOrder === "desc"
                    ? (lang.startsWith("zh") ? "降序" : "Descending")
                    : (lang.startsWith("zh") ? "升序" : "Ascending")}
            >
                {sortOrder === "desc" ? <ArrowDownAZ className="h-4 w-4" /> : <ArrowUpAZ className="h-4 w-4" />}
            </Button>
        </div>
    )
}
