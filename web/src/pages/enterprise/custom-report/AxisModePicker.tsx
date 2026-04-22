import { Settings2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuLabel,
    DropdownMenuRadioGroup,
    DropdownMenuRadioItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { getLabel, type AxisMode } from "./types"

export function AxisModePicker({
    measures,
    axisMode,
    rightAxisMeasures,
    onAxisModeChange,
    onRightAxisMeasuresChange,
}: {
    measures: string[]
    axisMode: AxisMode
    rightAxisMeasures: string[]
    onAxisModeChange: (mode: AxisMode) => void
    onRightAxisMeasuresChange: (measures: string[]) => void
}) {
    const { t, i18n } = useTranslation()
    const lang = i18n.resolvedLanguage || i18n.language
    const modeLabel = axisMode === "single"
        ? t("enterprise.customReport.axisSingle", "Single Axis")
        : axisMode === "custom"
            ? t("enterprise.customReport.axisCustomDual", "Custom Dual Axis")
            : t("enterprise.customReport.axisAutoDual", "Auto Dual Axis")

    const toggleRightAxisMeasure = (measure: string, checked: boolean | "indeterminate") => {
        const isChecked = checked === true
        const next = new Set(rightAxisMeasures)
        if (isChecked) {
            next.add(measure)
        } else {
            next.delete(measure)
        }

        // Keep at least one measure on each axis in custom mode.
        if (next.size === 0 || next.size === measures.length) {
            return
        }

        onRightAxisMeasuresChange(measures.filter((m) => next.has(m)))
    }

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button
                    variant="outline"
                    size="sm"
                    className="gap-1.5 border-[#6A6DE6]/30 bg-[#6A6DE6]/8 text-xs text-[#6266e2] hover:bg-[#6A6DE6]/14 hover:text-[#565bd9] dark:border-[#8A8DF7]/35 dark:bg-[#8A8DF7]/12 dark:text-[#B9BCFF]"
                >
                    <Settings2 className="w-3.5 h-3.5" />
                    <span>{t("enterprise.customReport.axisConfig", "轴配置")}</span>
                    <span className="rounded-full bg-white/75 px-2 py-0.5 text-[10px] font-medium text-slate-600 dark:bg-white/10 dark:text-slate-200">
                        {modeLabel}
                    </span>
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-[240px] p-2">
                <DropdownMenuLabel>{t("enterprise.customReport.axisMode", "坐标轴模式")}</DropdownMenuLabel>
                <DropdownMenuRadioGroup value={axisMode} onValueChange={(v) => onAxisModeChange(v as AxisMode)}>
                    <DropdownMenuRadioItem value="single">
                        {t("enterprise.customReport.axisSingle", "单轴")}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="auto">
                        {t("enterprise.customReport.axisAutoDual", "自动双轴")}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="custom">
                        {t("enterprise.customReport.axisCustomDual", "自定义双轴")}
                    </DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>

                {axisMode === "custom" && measures.length >= 2 && (
                    <>
                        <DropdownMenuSeparator />
                        <DropdownMenuLabel>{t("enterprise.customReport.axisRightMeasures", "右轴指标")}</DropdownMenuLabel>
                        {measures.map((measure) => (
                            <DropdownMenuCheckboxItem
                                key={measure}
                                checked={rightAxisMeasures.includes(measure)}
                                onCheckedChange={(checked) => toggleRightAxisMeasure(measure, checked)}
                            >
                                {getLabel(measure, lang)}
                            </DropdownMenuCheckboxItem>
                        ))}
                    </>
                )}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
