import { useState, useEffect, useMemo, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Pencil, Trash2, Shield, AlertTriangle, Search, Building2, User, Bell, CalendarClock, Lock, LockOpen, RotateCcw, Star } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
    enterpriseApi,
    type QuotaPolicy,
    type QuotaPolicyInput,
    type DepartmentQuotaPolicyBinding,
    type UserQuotaPolicy,
    type FeishuDepartment,
    type FeishuUser,
    type QuotaNotifConfig,
    type MyStatsResponse,
    type PromotedModelPolicy,
    type PromotedModelPolicyInput,
    type PromotedModelCandidate,
    type PromotedPricingMode,
} from "@/api/enterprise"
import { toast } from "sonner"
import { ALL_FILTER, getTimeRange } from "@/lib/enterprise"
import { useHasPermission } from "@/lib/permissions"
import useAuthStore from "@/store/auth"
import { Separator } from "@/components/ui/separator"

const defaultPolicy: QuotaPolicyInput = {
    name: "",
    tier1_ratio: 0.7,
    tier2_ratio: 0.9,
    tier1_rpm_multiplier: 1.0,
    tier1_tpm_multiplier: 1.0,
    tier2_rpm_multiplier: 0.5,
    tier2_tpm_multiplier: 0.5,
    tier3_rpm_multiplier: 0.1,
    tier3_tpm_multiplier: 0.1,
    block_at_tier3: false,
    tier2_blocked_models: "",
    tier3_blocked_models: "",
    tier2_price_input_threshold: 0,
    tier2_price_output_threshold: 0,
    tier2_price_condition: "or",
    tier3_price_input_threshold: 0,
    tier3_price_output_threshold: 0,
    tier3_price_condition: "or",
    period_quota: 0,
    period_type: 3,
}

function toDateTimeLocal(value?: string | null) {
    if (!value) return ""
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return ""

    const pad = (n: number) => String(n).padStart(2, "0")
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function dateTimeLocalToISO(value: string) {
    if (!value) return null
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return null
    return date.toISOString()
}

function formatBindingTime(value?: string | null, fallback?: string | null) {
    const raw = value || fallback
    if (!raw) return "—"
    const date = new Date(raw)
    if (Number.isNaN(date.getTime())) return "—"
    return date.toLocaleString()
}

function formatExpiryTime(value: string | null | undefined, permanentLabel: string) {
    if (!value) return permanentLabel
    return formatBindingTime(value)
}

function formatTokenPrice(price?: number, unit?: number) {
    if (price == null) return "-"
    const normalized = unit && unit > 0 ? price / unit * 1_000_000 : price * 1_000_000
    return `¥${normalized.toFixed(4)} / M`
}

function optionalNumber(value: string) {
    return value === "" ? undefined : Number(value)
}

const promotedPriceFields = [
    "per_request_price",
    "input_price",
    "image_input_price",
    "audio_input_price",
    "output_price",
    "image_output_price",
    "thinking_mode_output_price",
    "cached_price",
    "cache_creation_price",
    "web_search_price",
] as const

function computePromotedDiscountPrice(basePrice: PromotedModelCandidate["base_price"] | undefined, discountRate: number) {
    const next = { ...(basePrice || {}) }
    for (const field of promotedPriceFields) {
        const value = next[field]
        if (typeof value === "number") {
            next[field] = value * discountRate
        }
    }
    if (Array.isArray(next.conditional_prices)) {
        next.conditional_prices = next.conditional_prices.map((item) => ({
            ...item,
            price: computePromotedDiscountPrice(item.price, discountRate),
        }))
    }
    return next
}

function parsePromotedAuditSnapshot(raw?: string) {
    if (!raw) return null
    try {
        return JSON.parse(raw) as Partial<PromotedModelPolicy>
    } catch {
        return null
    }
}

function promotedAuditDetails(audit: { before?: string; after?: string }, label: (key: string) => string) {
    const before = parsePromotedAuditSnapshot(audit.before)
    const after = parsePromotedAuditSnapshot(audit.after)
    const current = after || before
    if (!current) return []

    const details: string[] = []
    const modelLabel = current.display_name || current.model
    if (modelLabel) details.push(`${label("enterprise.quota.promoted.model")}: ${modelLabel}`)
    if (current.model && current.display_name) details.push(`ID: ${current.model}`)
    if (current.recommend_badge) details.push(`${label("enterprise.quota.promoted.badge")}: ${current.recommend_badge}`)
    if (current.pricing_mode) {
        details.push(current.pricing_mode === "discount"
            ? `${label("enterprise.quota.promoted.pricingModeDiscount")} ${Math.round((current.discount_rate || 0) * 100)}%`
            : label("enterprise.quota.promoted.pricingModeManual"))
    }
    if (current.override_price) {
        details.push(`${label("enterprise.quota.promoted.inputPrice")}: ${formatTokenPrice(current.override_price.input_price, current.override_price.input_price_unit)}`)
        details.push(`${label("enterprise.quota.promoted.outputPrice")}: ${formatTokenPrice(current.override_price.output_price, current.override_price.output_price_unit)}`)
    }
    details.push(`${label("enterprise.quota.promoted.lock")}: ${current.price_locked ? label("enterprise.quota.promoted.locked") : label("enterprise.quota.promoted.unlocked")}`)
    return details
}

function isPastDateTimeLocal(value: string) {
    const date = new Date(value)
    return !Number.isNaN(date.getTime()) && date.getTime() <= Date.now()
}

/** Clamp-on-blur number input that avoids the broken onChange-clamp-every-keystroke pattern. */
function MultiplierInput({
    value,
    onChange,
    min = 0,
    max = 10,
    step = 0.01,
    disabled,
    className,
}: {
    value: number
    onChange: (val: number) => void
    min?: number
    max?: number
    step?: number
    disabled?: boolean
    className?: string
}) {
    const [localValue, setLocalValue] = useState(String(value))

    useEffect(() => {
        // Only sync when the external value actually differs from what we have,
        // to avoid clobbering mid-edit text like "1." or "0.0".
        if (parseFloat(localValue) !== value) {
            setLocalValue(String(value))
        }
    }, [value]) // eslint-disable-line react-hooks/exhaustive-deps

    const commit = useCallback((raw: string) => {
        const parsed = parseFloat(raw)
        // Round to 2dp to avoid floating-point drift (0.1+0.1+0.1 = 0.30000000000000004)
        const clamped = isNaN(parsed)
            ? min
            : Math.round(Math.max(min, Math.min(max, parsed)) * 100) / 100
        setLocalValue(String(clamped))
        onChange(clamped)
    }, [min, max, onChange])

    return (
        <Input
            type="number"
            value={localValue}
            onChange={(e) => setLocalValue(e.target.value)}
            onBlur={() => commit(localValue)}
            onKeyDown={(e) => { if (e.key === "Enter") commit(localValue) }}
            step={step}
            min={min}
            max={max}
            disabled={disabled}
            className={className}
        />
    )
}

function PriceBlockingRule({
    inputThreshold,
    outputThreshold,
    condition,
    onInputChange,
    onOutputChange,
    onConditionChange,
}: {
    inputThreshold: number
    outputThreshold: number
    condition: string
    onInputChange: (v: number) => void
    onOutputChange: (v: number) => void
    onConditionChange: (v: string) => void
}) {
    const { t } = useTranslation()
    return (
        <div className="border-t pt-3 mt-1">
            <Label className="text-xs font-medium">{t("enterprise.quota.priceBlockingRule")}</Label>
            <p className="text-xs text-muted-foreground mb-2">{t("enterprise.quota.priceBlockingHint")}</p>
            <div className="grid grid-cols-[1fr_auto_1fr] items-end gap-2">
                <div>
                    <Label className="text-xs">Input (¥/M)</Label>
                    <Input
                        type="number"
                        min={0}
                        step={0.001}
                        value={inputThreshold || ""}
                        onChange={(e) => onInputChange(Number(e.target.value))}
                        placeholder="0"
                        className="h-8 text-xs"
                    />
                </div>
                <Select
                    value={condition || "or"}
                    onValueChange={onConditionChange}
                >
                    <SelectTrigger className="w-16 h-8 text-xs">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="and">{t("enterprise.quota.priceConditionAnd")}</SelectItem>
                        <SelectItem value="or">{t("enterprise.quota.priceConditionOr")}</SelectItem>
                    </SelectContent>
                </Select>
                <div>
                    <Label className="text-xs">Output (¥/M)</Label>
                    <Input
                        type="number"
                        min={0}
                        step={0.001}
                        value={outputThreshold || ""}
                        onChange={(e) => onOutputChange(Number(e.target.value))}
                        placeholder="0"
                        className="h-8 text-xs"
                    />
                </div>
            </div>
        </div>
    )
}

function TierIndicator({ ratio, label }: { ratio: number; label: string }) {
    const clampedRatio = Math.max(0, Math.min(1, ratio))
    return (
        <div className="flex items-center gap-2">
            <div
                className="h-2 rounded-full bg-gradient-to-r from-green-500 via-yellow-500 to-red-500"
                style={{ width: "60px" }}
            >
                <div
                    className="h-2 w-1 bg-black rounded-full relative"
                    style={{ marginLeft: `${clampedRatio * 100}%`, transform: "translateX(-50%)" }}
                />
            </div>
            <span className="text-xs text-muted-foreground">{label}: {(ratio * 100).toFixed(0)}%</span>
        </div>
    )
}

function PolicyForm({
    policy,
    onChange,
}: {
    policy: QuotaPolicyInput
    onChange: (policy: QuotaPolicyInput) => void
}) {
    const { t } = useTranslation()

    return (
        <div className="space-y-6">
            {/* Policy Name */}
            <div className="space-y-2">
                <Label htmlFor="name">{t("enterprise.quota.policyName")}</Label>
                <Input
                    id="name"
                    value={policy.name}
                    onChange={(e) => onChange({ ...policy, name: e.target.value })}
                    placeholder={t("enterprise.quota.policyNamePlaceholder")}
                />
            </div>

            {/* Period Quota */}
            <div className="space-y-4">
                <h4 className="font-medium">{t("enterprise.quota.periodQuota")}</h4>
                <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                        <Label>{t("enterprise.quota.periodQuota")}</Label>
                        <Input
                            type="number"
                            value={policy.period_quota}
                            onChange={(e) => onChange({ ...policy, period_quota: Math.max(0, parseFloat(e.target.value) || 0) })}
                            min={0}
                            step={10}
                        />
                        <p className="text-xs text-muted-foreground">{t("enterprise.quota.periodQuotaHint")}</p>
                    </div>
                    <div className="space-y-2">
                        <Label>{t("enterprise.quota.periodType")}</Label>
                        <Select
                            value={String(policy.period_type)}
                            onValueChange={(v) => onChange({ ...policy, period_type: parseInt(v) })}
                        >
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="1">{t("enterprise.quota.daily")}</SelectItem>
                                <SelectItem value="2">{t("enterprise.quota.weekly")}</SelectItem>
                                <SelectItem value="3">{t("enterprise.quota.monthly")}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                </div>
            </div>

            {/* Tier Thresholds */}
            <div className="space-y-4">
                <h4 className="font-medium">{t("enterprise.quota.tierThresholds")}</h4>
                <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                        <Label>{t("enterprise.quota.tier1Ratio")}</Label>
                        <div className="flex items-center gap-2">
                            <Input
                                type="number"
                                value={(policy.tier1_ratio * 100).toFixed(0)}
                                onChange={(e) => {
                                    const val = Math.max(0, Math.min(100, parseFloat(e.target.value) || 0))
                                    onChange({ ...policy, tier1_ratio: val / 100 })
                                }}
                                min={0}
                                max={100}
                                step={5}
                                className="w-24"
                            />
                            <span className="text-sm text-muted-foreground">%</span>
                        </div>
                        <p className="text-xs text-muted-foreground">{t("enterprise.quota.tier1RatioDesc")}</p>
                    </div>
                    <div className="space-y-2">
                        <Label>{t("enterprise.quota.tier2Ratio")}</Label>
                        <div className="flex items-center gap-2">
                            <Input
                                type="number"
                                value={(policy.tier2_ratio * 100).toFixed(0)}
                                onChange={(e) => {
                                    const val = Math.max(0, Math.min(100, parseFloat(e.target.value) || 0))
                                    onChange({ ...policy, tier2_ratio: val / 100 })
                                }}
                                min={0}
                                max={100}
                                step={5}
                                className="w-24"
                            />
                            <span className="text-sm text-muted-foreground">%</span>
                        </div>
                        <p className="text-xs text-muted-foreground">{t("enterprise.quota.tier2RatioDesc")}</p>
                    </div>
                </div>
            </div>

            {/* Tier Multipliers */}
            <div className="space-y-4">
                <h4 className="font-medium">{t("enterprise.quota.tierMultipliers")}</h4>

                {/* Tier 1 — compact horizontal row */}
                <Card className="p-4 border-green-200 bg-green-50/50 dark:bg-green-950/20">
                    <h5 className="text-sm font-medium text-green-700 dark:text-green-400 mb-3">
                        {t("enterprise.quota.tier1")}
                    </h5>
                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <Label className="text-xs">RPM</Label>
                            <MultiplierInput
                                value={policy.tier1_rpm_multiplier}
                                onChange={(val) => onChange({ ...policy, tier1_rpm_multiplier: val })}
                                className="h-8"
                            />
                        </div>
                        <div>
                            <Label className="text-xs">TPM</Label>
                            <MultiplierInput
                                value={policy.tier1_tpm_multiplier}
                                onChange={(val) => onChange({ ...policy, tier1_tpm_multiplier: val })}
                                className="h-8"
                            />
                        </div>
                    </div>
                </Card>

                {/* Tier 2 */}
                <Card className="p-4 border-yellow-200 bg-yellow-50/50 dark:bg-yellow-950/20">
                    <h5 className="text-sm font-medium text-yellow-700 dark:text-yellow-400 mb-3">
                        {t("enterprise.quota.tier2")}
                    </h5>
                    <div className="space-y-3">
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <Label className="text-xs">RPM</Label>
                                <MultiplierInput
                                    value={policy.tier2_rpm_multiplier}
                                    onChange={(val) => onChange({ ...policy, tier2_rpm_multiplier: val })}
                                    className="h-8"
                                />
                            </div>
                            <div>
                                <Label className="text-xs">TPM</Label>
                                <MultiplierInput
                                    value={policy.tier2_tpm_multiplier}
                                    onChange={(val) => onChange({ ...policy, tier2_tpm_multiplier: val })}
                                    className="h-8"
                                />
                            </div>
                        </div>
                        <div>
                            <Label className="text-xs">{t("enterprise.quota.blockedModels")}</Label>
                            <Textarea
                                value={policy.tier2_blocked_models ? (() => { try { return JSON.parse(policy.tier2_blocked_models).join("\n") } catch { return policy.tier2_blocked_models } })() : ""}
                                onChange={(e) => {
                                    const lines = e.target.value.split("\n").map(s => s.trim()).filter(Boolean)
                                    onChange({ ...policy, tier2_blocked_models: lines.length > 0 ? JSON.stringify(lines) : "" })
                                }}
                                placeholder={t("enterprise.quota.blockedModelsHint")}
                                rows={2}
                                className="text-xs"
                            />
                        </div>
                        <PriceBlockingRule
                            inputThreshold={policy.tier2_price_input_threshold}
                            outputThreshold={policy.tier2_price_output_threshold}
                            condition={policy.tier2_price_condition}
                            onInputChange={(v) => onChange({ ...policy, tier2_price_input_threshold: v })}
                            onOutputChange={(v) => onChange({ ...policy, tier2_price_output_threshold: v })}
                            onConditionChange={(v) => onChange({ ...policy, tier2_price_condition: v as "and" | "or" })}
                        />
                    </div>
                </Card>

                {/* Tier 3 */}
                <Card className="p-4 border-red-200 bg-red-50/50 dark:bg-red-950/20">
                    <h5 className="text-sm font-medium text-red-700 dark:text-red-400 mb-3">
                        {t("enterprise.quota.tier3")}
                    </h5>
                    <div className="space-y-3">
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <Label className="text-xs">RPM</Label>
                                <MultiplierInput
                                    value={policy.tier3_rpm_multiplier}
                                    onChange={(val) => onChange({ ...policy, tier3_rpm_multiplier: val })}
                                    disabled={policy.block_at_tier3}
                                    className="h-8"
                                />
                            </div>
                            <div>
                                <Label className="text-xs">TPM</Label>
                                <MultiplierInput
                                    value={policy.tier3_tpm_multiplier}
                                    onChange={(val) => onChange({ ...policy, tier3_tpm_multiplier: val })}
                                    disabled={policy.block_at_tier3}
                                    className="h-8"
                                />
                            </div>
                        </div>
                        <div>
                            <Label className="text-xs">{t("enterprise.quota.blockedModels")}</Label>
                            <Textarea
                                value={policy.tier3_blocked_models ? (() => { try { return JSON.parse(policy.tier3_blocked_models).join("\n") } catch { return policy.tier3_blocked_models } })() : ""}
                                onChange={(e) => {
                                    const lines = e.target.value.split("\n").map(s => s.trim()).filter(Boolean)
                                    onChange({ ...policy, tier3_blocked_models: lines.length > 0 ? JSON.stringify(lines) : "" })
                                }}
                                placeholder={t("enterprise.quota.blockedModelsHint")}
                                rows={2}
                                className="text-xs"
                            />
                        </div>
                        <PriceBlockingRule
                            inputThreshold={policy.tier3_price_input_threshold}
                            outputThreshold={policy.tier3_price_output_threshold}
                            condition={policy.tier3_price_condition}
                            onInputChange={(v) => onChange({ ...policy, tier3_price_input_threshold: v })}
                            onOutputChange={(v) => onChange({ ...policy, tier3_price_output_threshold: v })}
                            onConditionChange={(v) => onChange({ ...policy, tier3_price_condition: v as "and" | "or" })}
                        />
                    </div>
                </Card>
            </div>

            {/* Block at Tier 3 */}
            <div className="flex items-center justify-between p-4 border rounded-lg border-red-200 bg-red-50/30 dark:bg-red-950/10">
                <div className="flex items-center gap-3">
                    <AlertTriangle className="w-5 h-5 text-red-500" />
                    <div>
                        <Label htmlFor="block">{t("enterprise.quota.blockAtTier3")}</Label>
                        <p className="text-xs text-muted-foreground">{t("enterprise.quota.blockAtTier3Desc")}</p>
                    </div>
                </div>
                <Switch
                    id="block"
                    checked={policy.block_at_tier3}
                    onCheckedChange={(checked) => onChange({ ...policy, block_at_tier3: checked })}
                />
            </div>
        </div>
    )
}

function BindingExpiryDialog({
    open,
    title,
    description,
    value,
    isSaving,
    onValueChange,
    onClose,
    onSave,
}: {
    open: boolean
    title: string
    description: string
    value: string
    isSaving: boolean
    onValueChange: (value: string) => void
    onClose: () => void
    onSave: () => void
}) {
    const { t } = useTranslation()

    return (
        <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>{title}</DialogTitle>
                    <DialogDescription>{description}</DialogDescription>
                </DialogHeader>
                <div className="space-y-2">
                    <Label>{t("enterprise.quota.expiresAt" as never)}</Label>
                    <Input
                        type="datetime-local"
                        value={value}
                        onChange={(e) => onValueChange(e.target.value)}
                    />
                    <p className="text-xs text-muted-foreground">
                        {t("enterprise.quota.expiresAtHint" as never)}
                    </p>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={onClose}>
                        {t("common.cancel")}
                    </Button>
                    <Button variant="outline" onClick={() => onValueChange("")}>
                        {t("enterprise.quota.clearExpiry" as never)}
                    </Button>
                    <Button onClick={onSave} disabled={isSaving}>
                        {isSaving ? t("common.saving") : t("common.save")}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}

// ─── Department Binding Tab ─────────────────────────────────────────────────

function DepartmentBindingTab({ policies, canManage }: { policies: QuotaPolicy[]; canManage: boolean }) {
    const { t } = useTranslation()
    const queryClient = useQueryClient()

    // ── Filter state ──
    const [filterLevel1, setFilterLevel1] = useState<string>("")
    const [filterLevel2, setFilterLevel2] = useState<string>("")

    // ── Query results state ──
    const [queryDepts, setQueryDepts] = useState<FeishuDepartment[]>([])
    const [hasQueried, setHasQueried] = useState(false)
    const [checkedDeptIds, setCheckedDeptIds] = useState<Set<string>>(new Set())
    const [bindPolicyId, setBindPolicyId] = useState<string>("")
    const [bindExpiresAt, setBindExpiresAt] = useState("")
    const [editingBinding, setEditingBinding] = useState<DepartmentQuotaPolicyBinding | null>(null)
    const [editingExpiresAt, setEditingExpiresAt] = useState("")

    // Fetch level1 departments (always)
    const { data: deptLevels } = useQuery({
        queryKey: ["enterprise", "department-levels"],
        queryFn: () => enterpriseApi.getDepartmentLevels(),
    })

    // Fetch level2 for selected level1 filter
    const { data: level2Data } = useQuery({
        queryKey: ["enterprise", "department-levels", filterLevel1],
        queryFn: () => enterpriseApi.getDepartmentLevels(filterLevel1),
        enabled: !!filterLevel1 && filterLevel1 !== ALL_FILTER,
    })

    const { data: bindings, isLoading: bindingsLoading } = useQuery({
        queryKey: ["enterprise", "dept-bindings"],
        queryFn: () => enterpriseApi.listDepartmentPolicyBindings(),
    })

    const batchBindMutation = useMutation({
        mutationFn: ({ department_ids, quota_policy_id, expires_at }: { department_ids: string[]; quota_policy_id: number; expires_at: string | null }) =>
            enterpriseApi.batchBindPolicyToDepartments(department_ids, quota_policy_id, expires_at),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "dept-bindings"] })
            setCheckedDeptIds(new Set())
            setBindPolicyId("")
            setBindExpiresAt("")
            toast.success(t("enterprise.quota.batchBindSuccess"))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const updateExpiryMutation = useMutation({
        mutationFn: ({ department_id, expires_at }: { department_id: string; expires_at: string | null }) =>
            enterpriseApi.updateDepartmentPolicyBindingExpiry(department_id, expires_at),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "dept-bindings"] })
            setEditingBinding(null)
            setEditingExpiresAt("")
            toast.success(t("enterprise.quota.expiryUpdated" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const unbindMutation = useMutation({
        mutationFn: (department_id: string) => enterpriseApi.unbindPolicyFromDepartment(department_id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "dept-bindings"] })
            toast.success(t("enterprise.quota.unbindSuccess"))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const level1Depts = deptLevels?.level1_departments || []
    const level2Depts = level2Data?.level2_departments || []
    const bindingList = useMemo(() => bindings?.bindings || [], [bindings?.bindings])

    // Reset level2 filter when level1 changes
    useEffect(() => { setFilterLevel2("") }, [filterLevel1])

    const handleQuery = () => {
        let results: FeishuDepartment[] = []
        const hasLevel1 = filterLevel1 && filterLevel1 !== ALL_FILTER
        const hasLevel2 = filterLevel2 && filterLevel2 !== ALL_FILTER
        if (hasLevel2) {
            // Show only the selected level2 department
            const found = level2Depts.find((d) => d.department_id === filterLevel2)
            results = found ? [found] : []
        } else if (hasLevel1) {
            // Show the level1 + all its level2 children
            const l1 = level1Depts.find((d) => d.department_id === filterLevel1)
            results = [...(l1 ? [l1] : []), ...level2Depts]
        } else {
            // No filter — show all level1 departments
            results = [...level1Depts]
        }
        setQueryDepts(results)
        setHasQueried(true)
        setCheckedDeptIds(new Set())
    }

    const toggleCheck = (id: string) => {
        setCheckedDeptIds((prev) => {
            const next = new Set(prev)
            if (next.has(id)) next.delete(id)
            else next.add(id)
            return next
        })
    }

    const toggleAllChecked = () => {
        if (checkedDeptIds.size === queryDepts.length) {
            setCheckedDeptIds(new Set())
        } else {
            setCheckedDeptIds(new Set(queryDepts.map((d) => d.department_id)))
        }
    }

    const handleBatchBind = () => {
        if (checkedDeptIds.size === 0 || !bindPolicyId) return
        if (bindExpiresAt && isPastDateTimeLocal(bindExpiresAt)) {
            toast.error(t("enterprise.quota.expiryMustBeFuture" as never))
            return
        }
        batchBindMutation.mutate({
            department_ids: Array.from(checkedDeptIds),
            quota_policy_id: parseInt(bindPolicyId),
            expires_at: dateTimeLocalToISO(bindExpiresAt),
        })
    }

    const openExpiryEditor = (binding: DepartmentQuotaPolicyBinding) => {
        setEditingBinding(binding)
        setEditingExpiresAt(toDateTimeLocal(binding.expires_at))
    }

    const saveExpiry = () => {
        if (!editingBinding) return
        if (editingExpiresAt && isPastDateTimeLocal(editingExpiresAt)) {
            toast.error(t("enterprise.quota.expiryMustBeFuture" as never))
            return
        }
        updateExpiryMutation.mutate({
            department_id: editingBinding.department_id,
            expires_at: dateTimeLocalToISO(editingExpiresAt),
        })
    }

    // Pre-build lookup for O(1) binding check in render
    const bindingByDeptId = useMemo(
        () => new Map(bindingList.map((b: DepartmentQuotaPolicyBinding) => [b.department_id, b])),
        [bindingList],
    )

    return (
        <div className="space-y-6">
            {/* Step 1: Filter & Query */}
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Building2 className="w-5 h-5" />
                        {t("enterprise.quota.selectDepartments")}
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="flex items-end gap-4">
                        <div className="space-y-2 min-w-[180px]">
                            <Label>{t("enterprise.quota.level1Department")}</Label>
                            <Select value={filterLevel1} onValueChange={setFilterLevel1}>
                                <SelectTrigger>
                                    <SelectValue placeholder={t("enterprise.quota.allDepartments")} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={ALL_FILTER}>{t("enterprise.quota.allDepartments")}</SelectItem>
                                    {level1Depts.map((d) => (
                                        <SelectItem key={d.department_id} value={d.department_id}>
                                            {d.name || d.department_id}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="space-y-2 min-w-[180px]">
                            <Label>{t("enterprise.quota.level2Department")}</Label>
                            <Select
                                value={filterLevel2}
                                onValueChange={setFilterLevel2}
                                disabled={!filterLevel1 || filterLevel1 === ALL_FILTER || level2Depts.length === 0}
                            >
                                <SelectTrigger>
                                    <SelectValue placeholder={t("enterprise.quota.allDepartments")} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={ALL_FILTER}>{t("enterprise.quota.allDepartments")}</SelectItem>
                                    {level2Depts.map((d) => (
                                        <SelectItem key={d.department_id} value={d.department_id}>
                                            {d.name || d.department_id}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                        <Button onClick={handleQuery}>
                            <Search className="w-4 h-4 mr-1.5" />
                            {t("enterprise.quota.query")}
                        </Button>
                    </div>
                </CardContent>
            </Card>

            {/* Step 2: Query Results with checkboxes */}
            {hasQueried && (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center justify-between">
                            <span>{t("enterprise.quota.queryResults")} ({queryDepts.length})</span>
                            {canManage && queryDepts.length > 0 && checkedDeptIds.size > 0 && (
                                <div className="flex items-center gap-3">
                                    <span className="text-sm font-normal text-muted-foreground">
                                        {t("enterprise.quota.selectedCount", { count: checkedDeptIds.size })}
                                    </span>
                                    <Select value={bindPolicyId} onValueChange={setBindPolicyId}>
                                        <SelectTrigger className="w-[180px]">
                                            <SelectValue placeholder={t("enterprise.quota.selectPolicy")} />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {policies.map((p) => (
                                                <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <div className="relative">
                                        <CalendarClock className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                                        <Input
                                            type="datetime-local"
                                            value={bindExpiresAt}
                                            onChange={(e) => setBindExpiresAt(e.target.value)}
                                            className="w-[210px] pl-8"
                                            title={t("enterprise.quota.expiresAt" as never)}
                                            aria-label={t("enterprise.quota.expiresAt" as never)}
                                        />
                                    </div>
                                    <Button
                                        onClick={handleBatchBind}
                                        disabled={!bindPolicyId || batchBindMutation.isPending}
                                        size="sm"
                                    >
                                        {t("enterprise.quota.bindSelected")}
                                    </Button>
                                </div>
                            )}
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        {queryDepts.length === 0 ? (
                            <div className="text-center py-8 text-muted-foreground">{t("enterprise.quota.noQueryResults")}</div>
                        ) : (
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <TableHead className="w-12">
                                            <input
                                                type="checkbox"
                                                checked={checkedDeptIds.size === queryDepts.length && queryDepts.length > 0}
                                                onChange={toggleAllChecked}
                                                className="rounded"
                                            />
                                        </TableHead>
                                        <TableHead>{t("enterprise.quota.department")}</TableHead>
                                        <TableHead>{t("enterprise.quota.memberCount")}</TableHead>
                                        <TableHead>{t("enterprise.quota.currentPolicy")}</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {queryDepts.map((d) => {
                                        const existing = bindingByDeptId.get(d.department_id)
                                        return (
                                            <TableRow key={d.department_id}>
                                                <TableCell>
                                                    <input
                                                        type="checkbox"
                                                        checked={checkedDeptIds.has(d.department_id)}
                                                        onChange={() => toggleCheck(d.department_id)}
                                                        className="rounded"
                                                    />
                                                </TableCell>
                                                <TableCell className="font-medium">{d.name || d.department_id}</TableCell>
                                                <TableCell>{d.member_count > 0 ? `${d.member_count} ${t("enterprise.quota.membersUnit")}` : "—"}</TableCell>
                                                <TableCell>
                                                    {existing?.quota_policy ? (
                                                        <Badge variant="secondary">{existing.quota_policy.name}</Badge>
                                                    ) : (
                                                        <span className="text-muted-foreground">{t("enterprise.quota.noPolicy")}</span>
                                                    )}
                                                </TableCell>
                                            </TableRow>
                                        )
                                    })}
                                </TableBody>
                            </Table>
                        )}
                    </CardContent>
                </Card>
            )}

            {/* Step 3: Current Bindings */}
            <Card>
                <CardHeader>
                    <CardTitle>{t("enterprise.quota.departmentBinding")}</CardTitle>
                </CardHeader>
                <CardContent>
                    {bindingsLoading ? (
                        <div className="text-center py-8 text-muted-foreground">{t("common.loading")}</div>
                    ) : bindingList.length === 0 ? (
                        <div className="text-center py-8 text-muted-foreground">{t("enterprise.quota.noDeptBindings")}</div>
                    ) : (
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>{t("enterprise.quota.level1Department")}</TableHead>
                                    <TableHead>{t("enterprise.quota.level2Department")}</TableHead>
                                    <TableHead>{t("enterprise.quota.policy")}</TableHead>
                                    <TableHead>{t("enterprise.quota.memberCount")}</TableHead>
                                    <TableHead>{t("enterprise.quota.overrideCount")}</TableHead>
                                    <TableHead>{t("enterprise.quota.effectiveAt" as never)}</TableHead>
                                    <TableHead>{t("enterprise.quota.expiresAt" as never)}</TableHead>
                                    <TableHead>{t("enterprise.quota.policyUpdatedAt")}</TableHead>
                                    <TableHead className="w-24">{t("common.edit")}</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {bindingList.map((b: DepartmentQuotaPolicyBinding) => (
                                    <TableRow key={b.id}>
                                        <TableCell>{b.level1_name || "—"}</TableCell>
                                        <TableCell>{b.level2_name || "—"}</TableCell>
                                        <TableCell>
                                            <Badge variant="secondary">
                                                {b.quota_policy?.name || `#${b.quota_policy_id}`}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>
                                            {b.member_count ?? 0}{t("enterprise.quota.membersUnit") ? ` ${t("enterprise.quota.membersUnit")}` : ""}
                                        </TableCell>
                                        <TableCell>
                                            {b.override_count ?? 0}{t("enterprise.quota.membersUnit") ? ` ${t("enterprise.quota.membersUnit")}` : ""}
                                        </TableCell>
                                        <TableCell className="text-sm text-muted-foreground">
                                            {formatBindingTime(b.effective_at, b.created_at)}
                                        </TableCell>
                                        <TableCell className="text-sm">
                                            {b.expires_at ? (
                                                <span>{formatExpiryTime(b.expires_at, t("enterprise.quota.neverExpires" as never))}</span>
                                            ) : (
                                                <Badge variant="outline">{t("enterprise.quota.neverExpires" as never)}</Badge>
                                            )}
                                        </TableCell>
                                        <TableCell className="text-sm text-muted-foreground">
                                            {new Date(b.updated_at).toLocaleString()}
                                        </TableCell>
                                        <TableCell>
                                            {canManage && (
                                                <div className="flex items-center gap-1">
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        onClick={() => openExpiryEditor(b)}
                                                        title={t("enterprise.quota.editExpiry" as never)}
                                                    >
                                                        <CalendarClock className="w-4 h-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        className="text-red-500 hover:text-red-600"
                                                        onClick={() => unbindMutation.mutate(b.department_id)}
                                                        disabled={unbindMutation.isPending}
                                                    >
                                                        {t("enterprise.quota.unbind")}
                                                    </Button>
                                                </div>
                                            )}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    )}
                </CardContent>
            </Card>

            <BindingExpiryDialog
                open={!!editingBinding}
                title={t("enterprise.quota.editExpiry" as never)}
                description={editingBinding?.quota_policy?.name || `#${editingBinding?.quota_policy_id ?? ""}`}
                value={editingExpiresAt}
                isSaving={updateExpiryMutation.isPending}
                onValueChange={setEditingExpiresAt}
                onClose={() => {
                    setEditingBinding(null)
                    setEditingExpiresAt("")
                }}
                onSave={saveExpiry}
            />
        </div>
    )
}

// ─── User Override Tab ──────────────────────────────────────────────────────

function UserOverrideTab({ policies, canManage }: { policies: QuotaPolicy[]; canManage: boolean }) {
    const { t } = useTranslation()
    const queryClient = useQueryClient()

    // Search & filter state
    const [searchKeyword, setSearchKeyword] = useState("")
    const [selectedPolicyFilters, setSelectedPolicyFilters] = useState<Set<string>>(new Set())
    const [queryUsers, setQueryUsers] = useState<FeishuUser[]>([])
    const [hasQueried, setHasQueried] = useState(false)
    const [checkedOpenIds, setCheckedOpenIds] = useState<Set<string>>(new Set())
    const [bindPolicyId, setBindPolicyId] = useState<string>("")
    const [bindExpiresAt, setBindExpiresAt] = useState("")
    const [editingBinding, setEditingBinding] = useState<UserQuotaPolicy | null>(null)
    const [editingExpiresAt, setEditingExpiresAt] = useState("")

    // Search users query (triggered on demand, fetch larger set for client-side policy filtering)
    const searchQuery = useQuery({
        queryKey: ["enterprise", "feishu-users-search", searchKeyword],
        queryFn: () => enterpriseApi.getFeishuUsers(1, 500, searchKeyword || undefined),
        enabled: false,
    })

    const { data: userBindings, isLoading: userBindingsLoading } = useQuery({
        queryKey: ["enterprise", "user-bindings"],
        queryFn: () => enterpriseApi.listUserPolicyBindings(),
    })

    const batchBindMutation = useMutation({
        mutationFn: ({ open_ids, quota_policy_id, expires_at }: { open_ids: string[]; quota_policy_id: number; expires_at: string | null }) =>
            enterpriseApi.batchBindPolicyToUsers(open_ids, quota_policy_id, expires_at),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "user-bindings"] })
            setCheckedOpenIds(new Set())
            setBindPolicyId("")
            setBindExpiresAt("")
            toast.success(t("enterprise.quota.bindSuccess"))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const updateExpiryMutation = useMutation({
        mutationFn: ({ open_id, expires_at }: { open_id: string; expires_at: string | null }) =>
            enterpriseApi.updateUserPolicyBindingExpiry(open_id, expires_at),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "user-bindings"] })
            setEditingBinding(null)
            setEditingExpiresAt("")
            toast.success(t("enterprise.quota.expiryUpdated" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const unbindMutation = useMutation({
        mutationFn: (open_id: string) => enterpriseApi.unbindPolicyFromUser(open_id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "user-bindings"] })
            toast.success(t("enterprise.quota.unbindSuccess"))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const bindingList = userBindings?.bindings || []

    const togglePolicyFilter = (policyName: string) => {
        setSelectedPolicyFilters(prev => {
            const next = new Set(prev)
            if (next.has(policyName)) next.delete(policyName)
            else next.add(policyName)
            return next
        })
    }

    const handleQuery = async () => {
        const result = await searchQuery.refetch()
        const users = result.data?.users || []
        setQueryUsers(users)
        setHasQueried(true)
        setCheckedOpenIds(new Set())
    }

    // Apply client-side policy filter on query results
    const filteredUsers = useMemo(() => queryUsers.filter(u => {
        if (selectedPolicyFilters.size === 0) return true
        const policyName = u.effective_policy || ""
        return selectedPolicyFilters.has(policyName)
    }), [queryUsers, selectedPolicyFilters])

    const toggleUser = (openId: string) => {
        setCheckedOpenIds(prev => {
            const next = new Set(prev)
            if (next.has(openId)) next.delete(openId)
            else next.add(openId)
            return next
        })
    }

    const toggleAllUsers = () => {
        if (checkedOpenIds.size === filteredUsers.length) {
            setCheckedOpenIds(new Set())
        } else {
            setCheckedOpenIds(new Set(filteredUsers.map(u => u.open_id)))
        }
    }

    const handleBatchBind = () => {
        if (checkedOpenIds.size === 0 || !bindPolicyId) return
        if (bindExpiresAt && isPastDateTimeLocal(bindExpiresAt)) {
            toast.error(t("enterprise.quota.expiryMustBeFuture" as never))
            return
        }
        batchBindMutation.mutate({
            open_ids: Array.from(checkedOpenIds),
            quota_policy_id: parseInt(bindPolicyId),
            expires_at: dateTimeLocalToISO(bindExpiresAt),
        })
    }

    const openExpiryEditor = (binding: UserQuotaPolicy) => {
        setEditingBinding(binding)
        setEditingExpiresAt(toDateTimeLocal(binding.expires_at))
    }

    const saveExpiry = () => {
        if (!editingBinding) return
        if (editingExpiresAt && isPastDateTimeLocal(editingExpiresAt)) {
            toast.error(t("enterprise.quota.expiryMustBeFuture" as never))
            return
        }
        updateExpiryMutation.mutate({
            open_id: editingBinding.open_id,
            expires_at: dateTimeLocalToISO(editingExpiresAt),
        })
    }

    return (
        <div className="space-y-6">
            {/* Card 1: Search & Policy Filter */}
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <User className="w-5 h-5" />
                        {t("enterprise.quota.searchUser")}
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="flex items-end gap-4">
                        <div className="flex-1 space-y-2">
                            <Label>{t("enterprise.quota.searchUser")}</Label>
                            <div className="relative">
                                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                                <Input
                                    value={searchKeyword}
                                    onChange={(e) => setSearchKeyword(e.target.value)}
                                    placeholder={t("enterprise.quota.searchUser")}
                                    className="pl-9"
                                    onKeyDown={(e) => { if (e.key === "Enter") handleQuery() }}
                                />
                            </div>
                        </div>
                        <div className="space-y-2 min-w-[200px]">
                            <Label>{t("enterprise.quota.filterByPolicy")}</Label>
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button variant="outline" className="w-full justify-start gap-1.5">
                                        <Shield className="w-4 h-4" />
                                        {selectedPolicyFilters.size === 0
                                            ? t("enterprise.quota.allPolicies")
                                            : t("enterprise.quota.selectedCount", { count: selectedPolicyFilters.size })}
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="start" className="w-56">
                                    <DropdownMenuLabel>{t("enterprise.quota.filterByPolicy")}</DropdownMenuLabel>
                                    <DropdownMenuSeparator />
                                    {policies.map((p) => (
                                        <DropdownMenuCheckboxItem
                                            key={p.id}
                                            checked={selectedPolicyFilters.has(p.name)}
                                            onCheckedChange={() => togglePolicyFilter(p.name)}
                                        >
                                            {p.name}
                                        </DropdownMenuCheckboxItem>
                                    ))}
                                    <DropdownMenuSeparator />
                                    <DropdownMenuCheckboxItem
                                        checked={selectedPolicyFilters.has("")}
                                        onCheckedChange={() => togglePolicyFilter("")}
                                    >
                                        {t("enterprise.quota.noPolicy")}
                                    </DropdownMenuCheckboxItem>
                                </DropdownMenuContent>
                            </DropdownMenu>
                        </div>
                        <Button onClick={handleQuery} disabled={searchQuery.isFetching}>
                            <Search className="w-4 h-4 mr-1" />
                            {t("enterprise.quota.query")}
                        </Button>
                    </div>
                </CardContent>
            </Card>

            {/* Card 2: Search Results with Checkboxes */}
            {hasQueried && (
                <Card>
                    <CardHeader>
                        <div className="flex items-center justify-between">
                            <CardTitle>
                                {t("enterprise.quota.queryResults")}
                                {selectedPolicyFilters.size > 0 && queryUsers.length !== filteredUsers.length && (
                                    <span className="ml-2 text-sm font-normal text-muted-foreground">
                                        ({filteredUsers.length}/{queryUsers.length})
                                    </span>
                                )}
                            </CardTitle>
                            {canManage && filteredUsers.length > 0 && checkedOpenIds.size > 0 && (
                                <div className="flex items-center gap-3">
                                    <span className="text-sm text-muted-foreground">
                                        {t("enterprise.quota.selectedCount", { count: checkedOpenIds.size })}
                                    </span>
                                    <Select value={bindPolicyId} onValueChange={setBindPolicyId}>
                                        <SelectTrigger className="w-48">
                                            <SelectValue placeholder={t("enterprise.quota.selectPolicy")} />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {policies.map((p) => (
                                                <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <div className="relative">
                                        <CalendarClock className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                                        <Input
                                            type="datetime-local"
                                            value={bindExpiresAt}
                                            onChange={(e) => setBindExpiresAt(e.target.value)}
                                            className="w-[210px] pl-8"
                                            title={t("enterprise.quota.expiresAt" as never)}
                                            aria-label={t("enterprise.quota.expiresAt" as never)}
                                        />
                                    </div>
                                    <Button
                                        onClick={handleBatchBind}
                                        disabled={!bindPolicyId || batchBindMutation.isPending}
                                        size="sm"
                                    >
                                        {t("enterprise.quota.bindSelected")}
                                    </Button>
                                </div>
                            )}
                        </div>
                    </CardHeader>
                    <CardContent>
                        {filteredUsers.length === 0 ? (
                            <div className="text-center py-8 text-muted-foreground">
                                {t("enterprise.quota.noUserResults")}
                            </div>
                        ) : (
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <TableHead className="w-12">
                                            <input
                                                type="checkbox"
                                                checked={checkedOpenIds.size === filteredUsers.length && filteredUsers.length > 0}
                                                onChange={toggleAllUsers}
                                            />
                                        </TableHead>
                                        <TableHead>{t("enterprise.quota.user")}</TableHead>
                                        <TableHead>{t("enterprise.users.email")}</TableHead>
                                        <TableHead>{t("enterprise.quota.level1Department")}</TableHead>
                                        <TableHead>{t("enterprise.quota.currentPolicy")}</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {filteredUsers.map((u) => (
                                        <TableRow key={u.open_id}>
                                            <TableCell>
                                                <input
                                                    type="checkbox"
                                                    checked={checkedOpenIds.has(u.open_id)}
                                                    onChange={() => toggleUser(u.open_id)}
                                                />
                                            </TableCell>
                                            <TableCell className="font-medium">{u.name || u.open_id}</TableCell>
                                            <TableCell className="text-sm text-muted-foreground">{u.email || "-"}</TableCell>
                                            <TableCell className="text-sm">{u.level1_dept_name || "-"}</TableCell>
                                            <TableCell>
                                                {u.effective_policy ? (
                                                    <Badge variant={u.policy_source === "user" ? "default" : "outline"}>
                                                        {u.effective_policy}
                                                    </Badge>
                                                ) : (
                                                    <span className="text-muted-foreground">-</span>
                                                )}
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        )}
                    </CardContent>
                </Card>
            )}

            {/* Card 3: Current User Overrides */}
            <Card>
                <CardHeader>
                    <CardTitle>{t("enterprise.quota.personalOverride")}</CardTitle>
                </CardHeader>
                <CardContent>
                    {userBindingsLoading ? (
                        <div className="text-center py-8 text-muted-foreground">{t("common.loading")}</div>
                    ) : bindingList.length === 0 ? (
                        <div className="text-center py-8 text-muted-foreground">{t("enterprise.quota.noUserOverrides")}</div>
                    ) : (
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>{t("enterprise.quota.user")}</TableHead>
                                    <TableHead>{t("enterprise.quota.policy")}</TableHead>
                                    <TableHead>{t("enterprise.quota.effectiveAt" as never)}</TableHead>
                                    <TableHead>{t("enterprise.quota.expiresAt" as never)}</TableHead>
                                    <TableHead className="w-24">{t("common.edit")}</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {bindingList.map((b: UserQuotaPolicy) => (
                                    <TableRow key={b.id}>
                                        <TableCell>{b.user_name || b.open_id}</TableCell>
                                        <TableCell>
                                            <Badge variant="outline">
                                                {b.quota_policy?.name || `#${b.quota_policy_id}`}
                                            </Badge>
                                        </TableCell>
                                        <TableCell className="text-sm text-muted-foreground">
                                            {formatBindingTime(b.effective_at, b.created_at)}
                                        </TableCell>
                                        <TableCell className="text-sm">
                                            {b.expires_at ? (
                                                <span>{formatExpiryTime(b.expires_at, t("enterprise.quota.neverExpires" as never))}</span>
                                            ) : (
                                                <Badge variant="outline">{t("enterprise.quota.neverExpires" as never)}</Badge>
                                            )}
                                        </TableCell>
                                        <TableCell>
                                            {canManage && (
                                                <div className="flex items-center gap-1">
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        onClick={() => openExpiryEditor(b)}
                                                        title={t("enterprise.quota.editExpiry" as never)}
                                                    >
                                                        <CalendarClock className="w-4 h-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        className="text-red-500 hover:text-red-600"
                                                        onClick={() => unbindMutation.mutate(b.open_id)}
                                                        disabled={unbindMutation.isPending}
                                                    >
                                                        {t("enterprise.quota.unbind")}
                                                    </Button>
                                                </div>
                                            )}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    )}
                </CardContent>
            </Card>

            <BindingExpiryDialog
                open={!!editingBinding}
                title={t("enterprise.quota.editExpiry" as never)}
                description={editingBinding?.user_name || editingBinding?.open_id || ""}
                value={editingExpiresAt}
                isSaving={updateExpiryMutation.isPending}
                onValueChange={setEditingExpiresAt}
                onClose={() => {
                    setEditingBinding(null)
                    setEditingExpiresAt("")
                }}
                onSave={saveExpiry}
            />
        </div>
    )
}

// ─── Notification Config Tab ────────────────────────────────────────────────

function renderTemplate(tmpl: string, vars: Record<string, string>): string {
    return Object.entries(vars).reduce((s, [k, v]) => s.split(`{${k}}`).join(v), tmpl)
}

const TIER_MAP = {
    tier2:   { titleField: "tier2_title"   as const, bodyField: "tier2_body"   as const, color: "orange" as const },
    tier3:   { titleField: "tier3_title"   as const, bodyField: "tier3_body"   as const, color: "red"    as const },
    exhaust: { titleField: "exhaust_title" as const, bodyField: "exhaust_body" as const, color: "red"    as const },
    policy_change: { titleField: "policy_change_title" as const, bodyField: "policy_change_body" as const, color: "green" as const },
} as const

function notifPeriodTypeLabel(tp: string): string {
    return tp === "daily" ? "日" : tp === "weekly" ? "周" : "月"
}

const DEFAULT_NOTIF_CONFIG: QuotaNotifConfig = {
    enabled: false,
    tier2_title: "AI 用量提醒",
    tier2_body: "您好 {name}，您本{period_type}的 AI 用量已达 {usage_pct}（阈值 {tier_threshold}，周期额度 {period_quota}），已进入二级限速，RPM/TPM 有所降低，请注意控制用量。",
    tier3_title: "AI 用量紧张提醒",
    tier3_body: "您好 {name}，您本{period_type}的 AI 用量已达 {usage_pct}（阈值 {tier_threshold}，周期额度 {period_quota}），已进入三级限速，请控制用量以避免服务中断。",
    exhaust_title: "AI 用量已耗尽",
    exhaust_body: "您好 {name}，您本{period_type}的 AI 用量已耗尽（周期额度 {period_quota}），所有请求将被拒绝，请联系管理员或等待下一周期重置。",
    admin_alert_enabled: false,
    admin_alert_threshold: 0.8,
    admin_alert_title: "成员额度用量告警",
    admin_alert_body: "{name} 本{period_type}的 AI 用量已达 {usage_pct}（告警阈值 {admin_threshold}，周期额度 {period_quota}），请关注。",
    policy_change_title: "AI 额度策略变更通知",
    policy_change_body: "您好 {name}，您的 AI 额度策略已变更为「{policy_name}」（周期额度 {period_quota}/{period_type}，阈值 {tier1_ratio}/{tier2_ratio}）。如有疑问请联系管理员。",
}

function NotifConfigTab({ canManage }: { canManage: boolean }) {
    const { t } = useTranslation()
    const queryClient = useQueryClient()
    const [draft, setDraft] = useState<QuotaNotifConfig | null>(null)
    const [selectedTier, setSelectedTier] = useState<"tier2" | "tier3" | "exhaust" | "policy_change">("tier2")

    const adminName = useAuthStore(s => s.enterpriseUser?.name ?? "管理员")

    const { data: serverCfg, isLoading } = useQuery({
        queryKey: ["enterprise", "quota-notif-config"],
        queryFn: () => enterpriseApi.getNotifConfig(),
    })

    const { start, end } = useMemo(() => getTimeRange("30d"), [])
    const { data: statsData } = useQuery<MyStatsResponse>({
        queryKey: ["my-stats-preview", start, end],
        queryFn: () => enterpriseApi.getMyStats(start, end),
    })
    const quota = statsData?.quota

    const saveMutation = useMutation({
        mutationFn: (cfg: QuotaNotifConfig) => enterpriseApi.updateNotifConfig(cfg),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-notif-config"] })
            toast.success(t("enterprise.quota.notif.saveSuccess" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const cfg: QuotaNotifConfig = draft ?? serverCfg ?? DEFAULT_NOTIF_CONFIG

    const VARIABLES = selectedTier === "policy_change" ? [
        { key: "{name}", label: t("enterprise.quota.notif.varName" as never) },
        { key: "{policy_name}", label: t("enterprise.quota.notif.varPolicyName" as never) },
        { key: "{period_quota}", label: t("enterprise.quota.notif.varPeriodQuota" as never) },
        { key: "{period_type}", label: t("enterprise.quota.notif.varPeriodType" as never) },
        { key: "{tier1_ratio}", label: t("enterprise.quota.notif.varTier1Ratio" as never) },
        { key: "{tier2_ratio}", label: t("enterprise.quota.notif.varTier2Ratio" as never) },
    ] : [
        { key: "{name}", label: t("enterprise.quota.notif.varName" as never) },
        { key: "{usage_pct}", label: t("enterprise.quota.notif.varUsagePct" as never) },
        { key: "{tier_threshold}", label: t("enterprise.quota.notif.varTierThreshold" as never) },
        { key: "{period_quota}", label: t("enterprise.quota.notif.varPeriodQuota" as never) },
        { key: "{period_type}", label: t("enterprise.quota.notif.varPeriodType" as never) },
    ]

    const { titleField, bodyField, color } = TIER_MAP[selectedTier]

    const previewVars = useMemo(() => {
        if (selectedTier === "policy_change") {
            return {
                name: adminName,
                policy_name: "标准版额度策略",
                period_quota: quota ? `¥${quota.period_quota.toFixed(2)}` : "¥100.00",
                period_type: quota ? notifPeriodTypeLabel(quota.period_type) : "月",
                tier1_ratio: `${((quota?.tier1_ratio ?? 0.7) * 100).toFixed(0)}%`,
                tier2_ratio: `${((quota?.tier2_ratio ?? 0.9) * 100).toFixed(0)}%`,
            } as Record<string, string>
        }
        const tierThresholdStr =
            selectedTier === "tier2" ? `${((quota?.tier1_ratio ?? 0.7) * 100).toFixed(0)}%`
            : selectedTier === "tier3" ? `${((quota?.tier2_ratio ?? 0.9) * 100).toFixed(0)}%`
            : "100%"
        return {
            name: adminName,
            usage_pct: quota && quota.period_quota > 0
                ? `${((quota.period_used / quota.period_quota) * 100).toFixed(1)}%`
                : "75.0%",
            period_quota: quota ? `¥${quota.period_quota.toFixed(2)}` : "¥100.00",
            period_type: quota ? notifPeriodTypeLabel(quota.period_type) : "月",
            tier_threshold: tierThresholdStr,
        } as Record<string, string>
    }, [selectedTier, quota, adminName])

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text).catch(() => {})
        toast.success(`已复制 ${text}`)
    }

    if (isLoading) {
        return <div className="text-center py-8 text-muted-foreground">{t("common.loading")}</div>
    }

    return (
        <div className="space-y-6">
            {/* P2P warning — only shown when backend reports P2P is not configured */}
            {serverCfg && !serverCfg.p2p_available && (
                <Card className="border-amber-200 bg-amber-50/50 dark:bg-amber-950/20">
                    <CardContent className="pt-4 pb-3">
                        <div className="flex gap-2 items-start text-sm text-amber-700 dark:text-amber-400">
                            <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
                            <span>{t("enterprise.quota.notif.noP2PWarning" as never)}</span>
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Enable switch */}
            <Card>
                <CardContent className="pt-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <Label className="text-base font-medium">{t("enterprise.quota.notif.enable" as never)}</Label>
                            <p className="text-sm text-muted-foreground mt-0.5">{t("enterprise.quota.notif.enableDesc" as never)}</p>
                        </div>
                        <Switch
                            checked={cfg.enabled}
                            onCheckedChange={(v) => setDraft({ ...cfg, enabled: v })}
                            disabled={!canManage}
                        />
                    </div>
                </CardContent>
            </Card>

            {/* Single template editor with tier selector */}
            <Card>
                <CardHeader className="pb-3">
                    <Select value={selectedTier} onValueChange={(v) => setSelectedTier(v as typeof selectedTier)}>
                        <SelectTrigger className="w-52">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="tier2">{t("enterprise.quota.notif.tier2" as never)}</SelectItem>
                            <SelectItem value="tier3">{t("enterprise.quota.notif.tier3" as never)}</SelectItem>
                            <SelectItem value="exhaust">{t("enterprise.quota.notif.exhaust" as never)}</SelectItem>
                            <SelectItem value="policy_change">{t("enterprise.quota.notif.policyChange" as never)}</SelectItem>
                        </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground">{t(`enterprise.quota.notif.${selectedTier}Desc` as never)}</p>
                </CardHeader>
                <CardContent className="space-y-3">
                    <div className="space-y-1">
                        <Label className="text-xs">{t("enterprise.quota.notif.msgTitle" as never)}</Label>
                        <Input
                            value={cfg[titleField]}
                            onChange={(e) => setDraft({ ...cfg, [titleField]: e.target.value })}
                            disabled={!canManage}
                        />
                    </div>
                    <div className="space-y-1">
                        <Label className="text-xs">{t("enterprise.quota.notif.msgBody" as never)}</Label>
                        <Textarea
                            value={cfg[bodyField]}
                            onChange={(e) => setDraft({ ...cfg, [bodyField]: e.target.value })}
                            rows={3}
                            disabled={!canManage}
                        />
                    </div>

                    <Separator />

                    <div>
                        <p className="text-xs text-muted-foreground mb-1.5">
                            {t("enterprise.quota.notif.preview" as never)}
                            {!quota && (
                                <span className="ml-1 opacity-70">{t("enterprise.quota.notif.previewSampleHint" as never)}</span>
                            )}
                        </p>
                        <div className="rounded-lg border bg-muted/40 p-3 space-y-1.5 mt-2">
                            <div className="flex items-center gap-2">
                                <div className={`w-1 h-5 rounded-full ${color === "orange" ? "bg-orange-400" : color === "green" ? "bg-green-400" : "bg-red-400"}`} />
                                <span className="font-medium text-sm">
                                    {renderTemplate(cfg[titleField], previewVars)}
                                </span>
                            </div>
                            <p className="text-sm text-muted-foreground whitespace-pre-wrap leading-relaxed">
                                {renderTemplate(cfg[bodyField], previewVars)}
                            </p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Variable hints */}
            <Card>
                <CardContent className="pt-4">
                    <p className="text-xs font-medium text-muted-foreground mb-2">{t("enterprise.quota.notif.variables" as never)}</p>
                    <div className="flex flex-wrap gap-2">
                        {VARIABLES.map(({ key, label }) => (
                            <button
                                key={key}
                                onClick={() => copyToClipboard(key)}
                                className="inline-flex items-center gap-1 px-2 py-1 rounded border text-xs font-mono bg-muted hover:bg-muted/80 transition-colors"
                            >
                                {label}
                            </button>
                        ))}
                    </div>
                </CardContent>
            </Card>

            {/* Actions */}
            {canManage && (
                <div className="flex justify-end gap-2">
                    <Button
                        variant="outline"
                        onClick={() => {
                            setDraft(DEFAULT_NOTIF_CONFIG)
                            toast.success(t("enterprise.quota.notif.resetSuccess" as never))
                        }}
                    >
                        {t("enterprise.quota.notif.resetDefaults" as never)}
                    </Button>
                    <Button
                        onClick={() => draft && saveMutation.mutate(draft)}
                        disabled={saveMutation.isPending}
                    >
                        {saveMutation.isPending ? t("common.saving") : t("enterprise.quota.notif.save" as never)}
                    </Button>
                </div>
            )}
        </div>
    )
}

function promotedFormDefaults(): PromotedModelPolicyInput {
    return {
        model: "",
        enabled: true,
        override_price: {
            input_price_unit: 1,
            output_price_unit: 1,
        },
        pricing_mode: "discount",
        discount_rate: 0.4,
        price_locked: false,
    }
}

function PromotedModelsTab({ policies, canManage }: { policies: QuotaPolicy[]; canManage: boolean }) {
    const { t } = useTranslation()
    const queryClient = useQueryClient()
    const [selectedPolicyId, setSelectedPolicyId] = useState<number | null>(policies[0]?.id ?? null)
    const [editing, setEditing] = useState<PromotedModelPolicy | null>(null)
    const [dialogOpen, setDialogOpen] = useState(false)
    const [overrideLocked, setOverrideLocked] = useState(false)
    const [lockedFilter, setLockedFilter] = useState<"all" | "locked" | "unlocked">("all")
    const [deleteTarget, setDeleteTarget] = useState<PromotedModelPolicy | null>(null)
    const [form, setForm] = useState<PromotedModelPolicyInput>(promotedFormDefaults)
    const [selectedCandidate, setSelectedCandidate] = useState<PromotedModelCandidate | null>(null)

    const policyId = selectedPolicyId ?? policies[0]?.id

    useEffect(() => {
        if (!policyId && policies[0]?.id) {
            setSelectedPolicyId(policies[0].id)
        }
    }, [policies, policyId])

    const entriesQuery = useQuery({
        queryKey: ["enterprise", "quota-promoted-models", policyId],
        queryFn: () => enterpriseApi.listPromotedModels(policyId!),
        enabled: !!policyId,
    })

    const auditsQuery = useQuery({
        queryKey: ["enterprise", "quota-promoted-model-audits", policyId],
        queryFn: () => enterpriseApi.listPromotedModelAudits(policyId!),
        enabled: !!policyId,
    })

    const candidatesQuery = useQuery({
        queryKey: ["enterprise", "quota-promoted-model-candidates", policyId, form.model, form.channel_id],
        queryFn: () => enterpriseApi.listPromotedModelCandidates(policyId!, { keyword: form.model, channel_id: form.channel_id }),
        enabled: dialogOpen && !!policyId,
    })

    const invalidatePromoted = () => {
        queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-promoted-models", policyId] })
        queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-promoted-model-audits", policyId] })
    }

    const createMutation = useMutation({
        mutationFn: (payload: PromotedModelPolicyInput) => enterpriseApi.createPromotedModel(policyId!, payload),
        onSuccess: () => {
            invalidatePromoted()
            setDialogOpen(false)
            setSelectedCandidate(null)
            setForm(promotedFormDefaults())
            toast.success(t("enterprise.quota.promoted.saved" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const updateMutation = useMutation({
        mutationFn: (payload: PromotedModelPolicyInput & { override_locked?: boolean }) =>
            enterpriseApi.updatePromotedModel(policyId!, editing!.id, payload),
        onSuccess: () => {
            invalidatePromoted()
            setDialogOpen(false)
            setEditing(null)
            setOverrideLocked(false)
            setSelectedCandidate(null)
            toast.success(t("enterprise.quota.promoted.saved" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const deleteMutation = useMutation({
        mutationFn: (entry: PromotedModelPolicy) => enterpriseApi.deletePromotedModel(policyId!, entry.id),
        onSuccess: () => {
            invalidatePromoted()
            setDeleteTarget(null)
            toast.success(t("enterprise.quota.promoted.deleted" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const rollbackMutation = useMutation({
        mutationFn: (entry: PromotedModelPolicy) => enterpriseApi.rollbackPromotedModel(policyId!, entry.id, Math.max(1, entry.version - 1)),
        onSuccess: () => {
            invalidatePromoted()
            toast.success(t("enterprise.quota.promoted.rollbackSuccess" as never))
        },
        onError: (err: Error) => toast.error(err.message),
    })

    const openCreate = () => {
        setEditing(null)
        setOverrideLocked(false)
        setSelectedCandidate(null)
        setForm(promotedFormDefaults())
        setDialogOpen(true)
    }

    const openEdit = (entry: PromotedModelPolicy) => {
        setEditing(entry)
        setOverrideLocked(false)
        setSelectedCandidate(null)
        setForm({
            model: entry.model,
            channel_id: entry.channel_id || undefined,
            display_name: entry.display_name || "",
            recommend_badge: entry.recommend_badge || "",
            sort_order: entry.sort_order || 0,
            enabled: entry.enabled,
            override_price: entry.override_price || promotedFormDefaults().override_price,
            pricing_mode: entry.pricing_mode || (entry.discount_rate > 0 ? "discount" : "manual"),
            discount_rate: entry.discount_rate || 0,
            price_locked: entry.price_locked,
            effective_at: entry.effective_at || null,
            expires_at: entry.expires_at || null,
        })
        setDialogOpen(true)
    }

    const save = () => {
        if (!policyId) return
        if (!form.model.trim()) {
            toast.error(t("enterprise.quota.promoted.modelRequired" as never))
            return
        }
        if (form.expires_at && isPastDateTimeLocal(form.expires_at)) {
            toast.error(t("enterprise.quota.expiryMustBeFuture" as never))
            return
        }

        const payload: PromotedModelPolicyInput = {
            ...form,
            channel_id: form.channel_id || undefined,
            display_name: form.display_name?.trim(),
            recommend_badge: form.recommend_badge?.trim(),
            pricing_mode: form.pricing_mode,
            discount_rate: form.pricing_mode === "discount" ? form.discount_rate : 0,
            effective_at: dateTimeLocalToISO(form.effective_at || ""),
            expires_at: dateTimeLocalToISO(form.expires_at || ""),
        }

        if (editing) {
            updateMutation.mutate({ ...payload, override_locked: overrideLocked })
        } else {
            createMutation.mutate(payload)
        }
    }

    const setPricingMode = (mode: PromotedPricingMode) => {
        const discountRate = form.discount_rate || 0.4
        setForm({
            ...form,
            pricing_mode: mode,
            discount_rate: mode === "discount" ? discountRate : 0,
            override_price: mode === "discount" && selectedCandidate
                ? computePromotedDiscountPrice(selectedCandidate.base_price, discountRate)
                : form.override_price,
        })
    }

    const setDiscountPercent = (value: string) => {
        const percent = optionalNumber(value)
        const discountRate = percent == null ? 0 : percent / 100
        setForm({
            ...form,
            discount_rate: discountRate,
            override_price: selectedCandidate
                ? computePromotedDiscountPrice(selectedCandidate.base_price, discountRate)
                : form.override_price,
        })
    }

    const selectCandidate = (candidate: PromotedModelCandidate) => {
        const discountRate = form.discount_rate || 0.4
        const channelID = candidate.channels[0]?.id || form.channel_id
        setSelectedCandidate(candidate)
        setForm({
            ...form,
            model: candidate.model,
            channel_id: channelID,
            override_price: form.pricing_mode === "discount"
                ? computePromotedDiscountPrice(candidate.base_price, discountRate)
                : form.override_price,
        })
    }

    const showCandidateDropdown = !editing &&
        dialogOpen &&
        !!form.model &&
        selectedCandidate?.model !== form.model &&
        (candidatesQuery.data?.candidates?.length || 0) > 0

    const entries = (entriesQuery.data?.entries || []).filter((entry) => {
        if (lockedFilter === "locked") return entry.price_locked
        if (lockedFilter === "unlocked") return !entry.price_locked
        return true
    })

    const auditRows = auditsQuery.data?.audits || []

    if (policies.length === 0) {
        return (
            <Card>
                <CardContent className="text-center py-8 text-muted-foreground">
                    {t("enterprise.quota.promoted.noPolicies" as never)}
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between gap-3">
                    <CardTitle className="flex items-center gap-2">
                        <Star className="h-5 w-5" />
                        {t("enterprise.quota.promoted.title" as never)}
                    </CardTitle>
                    <div className="flex items-center gap-2">
                        <Select value={policyId ? String(policyId) : ""} onValueChange={(v) => setSelectedPolicyId(Number(v))}>
                            <SelectTrigger className="w-56">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                {policies.map((p) => (
                                    <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Select value={lockedFilter} onValueChange={(v) => setLockedFilter(v as typeof lockedFilter)}>
                            <SelectTrigger className="w-36">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("common.all")}</SelectItem>
                                <SelectItem value="locked">{t("enterprise.quota.promoted.locked" as never)}</SelectItem>
                                <SelectItem value="unlocked">{t("enterprise.quota.promoted.unlocked" as never)}</SelectItem>
                            </SelectContent>
                        </Select>
                        {canManage && (
                            <Button onClick={openCreate}>
                                <Plus className="h-4 w-4 mr-2" />
                                {t("enterprise.quota.promoted.add" as never)}
                            </Button>
                        )}
                    </div>
                </div>
            </CardHeader>
            <CardContent className="space-y-4">
                {entriesQuery.isLoading ? (
                    <div className="text-center py-8 text-muted-foreground">{t("common.loading")}</div>
                ) : entries.length === 0 ? (
                    <div className="text-center py-8 text-muted-foreground">{t("enterprise.quota.promoted.empty" as never)}</div>
                ) : (
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>{t("enterprise.quota.promoted.model" as never)}</TableHead>
                                <TableHead>{t("enterprise.quota.promoted.badge" as never)}</TableHead>
                                <TableHead>{t("enterprise.quota.promoted.overridePrice" as never)}</TableHead>
                                <TableHead>{t("enterprise.quota.promoted.lock" as never)}</TableHead>
                                <TableHead>{t("enterprise.quota.promoted.status" as never)}</TableHead>
                                <TableHead>{t("enterprise.quota.promoted.version" as never)}</TableHead>
                                <TableHead className="w-32">{t("common.edit")}</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {entries.map((entry) => (
                                <TableRow key={entry.id}>
                                    <TableCell>
                                        <div className="font-medium">{entry.display_name || entry.model}</div>
                                        <div className="text-xs text-muted-foreground">{entry.model}</div>
                                        {entry.channel_id > 0 && (
                                            <div className="text-xs text-muted-foreground">
                                                {t("enterprise.quota.promoted.referenceChannel" as never)} #{entry.channel_id}
                                            </div>
                                        )}
                                    </TableCell>
                                    <TableCell>{entry.recommend_badge || "-"}</TableCell>
                                    <TableCell>
                                        <Badge variant="outline" className="mb-1 text-[10px]">
                                            {entry.pricing_mode === "discount"
                                                ? `${t("enterprise.quota.promoted.pricingModeDiscount" as never)} ${Math.round((entry.discount_rate || 0) * 100)}%`
                                                : t("enterprise.quota.promoted.pricingModeManual" as never)}
                                        </Badge>
                                        <div>{formatTokenPrice(entry.override_price?.input_price, entry.override_price?.input_price_unit)}</div>
                                        <div className="text-xs text-muted-foreground">{formatTokenPrice(entry.override_price?.output_price, entry.override_price?.output_price_unit)}</div>
                                    </TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-1.5">
                                            {entry.price_locked ? (
                                                <Lock className="h-4 w-4" />
                                            ) : (
                                                <LockOpen className="h-4 w-4 text-muted-foreground" />
                                            )}
                                            <span className="text-xs text-muted-foreground">
                                                {entry.price_locked ? t("enterprise.quota.promoted.locked" as never) : t("enterprise.quota.promoted.unlocked" as never)}
                                            </span>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={entry.enabled ? "default" : "secondary"}>
                                            {entry.enabled ? t("enterprise.quota.promoted.enabled" as never) : t("enterprise.quota.promoted.disabled" as never)}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>{entry.version}</TableCell>
                                    <TableCell>
                                        {canManage && (
                                            <div className="flex items-center gap-1">
                                                <Button variant="ghost" size="icon" onClick={() => openEdit(entry)}>
                                                    <Pencil className="h-4 w-4" />
                                                </Button>
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    disabled={entry.version <= 1 || rollbackMutation.isPending}
                                                    onClick={() => rollbackMutation.mutate(entry)}
                                                >
                                                    <RotateCcw className="h-4 w-4" />
                                                </Button>
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    className="text-red-500 hover:text-red-600"
                                                    onClick={() => setDeleteTarget(entry)}
                                                >
                                                    <Trash2 className="h-4 w-4" />
                                                </Button>
                                            </div>
                                        )}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
                <div className="text-xs text-muted-foreground">{t("enterprise.quota.promoted.referenceChannelHint" as never)}</div>

                <Separator />

                <div className="space-y-2">
                    <h3 className="text-sm font-medium">{t("enterprise.quota.promoted.audit" as never)}</h3>
                    {auditsQuery.isLoading ? (
                        <div className="text-xs text-muted-foreground">{t("common.loading")}</div>
                    ) : auditRows.length === 0 ? (
                        <div className="text-xs text-muted-foreground">{t("enterprise.quota.promoted.noAudit" as never)}</div>
                    ) : (
                        auditRows.slice(0, 10).map((audit) => {
                            const details = promotedAuditDetails(audit, (key) => t(key as never) as string)
                            return (
                                <div key={audit.id} className="rounded border px-3 py-2 text-xs">
                                    <div className="flex items-center justify-between gap-2">
                                        <div className="flex items-center gap-2">
                                            <Badge variant="outline" className="h-5 text-[10px]">{audit.action}</Badge>
                                            {audit.operator_name && <span className="text-muted-foreground">{audit.operator_name}</span>}
                                        </div>
                                        <span className="text-muted-foreground">{formatBindingTime(audit.created_at)}</span>
                                    </div>
                                    <div className="mt-2 grid gap-1 text-muted-foreground sm:grid-cols-2 lg:grid-cols-3">
                                        {details.length > 0 ? details.map((item) => (
                                            <div key={item} className="truncate">{item}</div>
                                        )) : (
                                            <div>{audit.summary || "-"}</div>
                                        )}
                                    </div>
                                </div>
                            )
                        })
                    )}
                </div>

                <Dialog open={dialogOpen} onOpenChange={(open) => {
                    setDialogOpen(open)
                    if (!open) {
                        setEditing(null)
                        setOverrideLocked(false)
                        setSelectedCandidate(null)
                    }
                }}>
                    <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                        <DialogHeader>
                            <DialogTitle>{editing ? t("enterprise.quota.promoted.edit" as never) : t("enterprise.quota.promoted.add" as never)}</DialogTitle>
                            <DialogDescription>{t("enterprise.quota.promoted.formDescription" as never)}</DialogDescription>
                        </DialogHeader>
                        <div className="space-y-4">
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <Label>{t("enterprise.quota.promoted.model" as never)}</Label>
                                    <div className="relative">
                                        <Input
                                            value={form.model}
                                            disabled={!!editing}
                                            placeholder={t("enterprise.quota.promoted.modelSearchPlaceholder" as never)}
                                            onChange={(e) => {
                                                setSelectedCandidate(null)
                                                setForm({ ...form, model: e.target.value })
                                            }}
                                        />
                                        {showCandidateDropdown && (
                                            <div className="absolute z-50 mt-1 max-h-64 w-full overflow-auto rounded-md border bg-popover p-1 shadow-md">
                                                {candidatesQuery.data?.candidates.map((candidate) => (
                                                    <button
                                                        key={candidate.model}
                                                        type="button"
                                                        className="flex w-full items-start justify-between gap-3 rounded-sm px-2 py-2 text-left hover:bg-muted"
                                                        onClick={() => selectCandidate(candidate)}
                                                    >
                                                        <div className="min-w-0 flex-1">
                                                            <div className="break-all font-mono text-xs leading-5" title={candidate.model}>{candidate.model}</div>
                                                            <div className="mt-1 text-[11px] text-muted-foreground">
                                                                {candidate.type_name || candidate.type} · {formatTokenPrice(candidate.base_price?.input_price, candidate.base_price?.input_price_unit)} / {formatTokenPrice(candidate.base_price?.output_price, candidate.base_price?.output_price_unit)}
                                                            </div>
                                                        </div>
                                                        {candidate.channels[0] && (
                                                            <Badge variant="secondary" className="shrink-0 text-[10px]">
                                                                #{candidate.channels[0].id}
                                                            </Badge>
                                                        )}
                                                    </button>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                </div>
                                <div>
                                    <Label>{t("enterprise.quota.promoted.displayName" as never)}</Label>
                                    <Input value={form.display_name || ""} onChange={(e) => setForm({ ...form, display_name: e.target.value })} />
                                </div>
                            </div>
                            <div className="grid grid-cols-3 gap-3">
                                <div>
                                    <Label>{t("enterprise.quota.promoted.badge" as never)}</Label>
                                    <Input value={form.recommend_badge || ""} onChange={(e) => setForm({ ...form, recommend_badge: e.target.value })} />
                                </div>
                                <div>
                                    <Label>{t("enterprise.quota.promoted.sortOrder" as never)}</Label>
                                    <Input type="number" value={form.sort_order ?? 0} onChange={(e) => setForm({ ...form, sort_order: Number(e.target.value) })} />
                                </div>
                                <div>
                                    <Label>{t("enterprise.quota.promoted.referenceChannel" as never)}</Label>
                                    <Input type="number" value={form.channel_id || ""} onChange={(e) => setForm({ ...form, channel_id: Number(e.target.value) || undefined })} />
                                </div>
                            </div>
                            <div className="space-y-3 rounded border p-3">
                                <div className="flex items-center justify-between gap-3">
                                    <div>
                                        <Label>{t("enterprise.quota.promoted.pricingMode" as never)}</Label>
                                        <p className="text-xs text-muted-foreground">{t("enterprise.quota.promoted.pricingModeHint" as never)}</p>
                                    </div>
                                    <Tabs value={form.pricing_mode} onValueChange={(v) => setPricingMode(v as PromotedPricingMode)}>
                                        <TabsList>
                                            <TabsTrigger value="discount">{t("enterprise.quota.promoted.pricingModeDiscount" as never)}</TabsTrigger>
                                            <TabsTrigger value="manual">{t("enterprise.quota.promoted.pricingModeManual" as never)}</TabsTrigger>
                                        </TabsList>
                                    </Tabs>
                                </div>
                                {form.pricing_mode === "discount" ? (
                                    <div className="grid grid-cols-3 gap-3">
                                        <div>
                                            <Label>{t("enterprise.quota.promoted.discountPercent" as never)}</Label>
                                            <Input
                                                type="number"
                                                min={0}
                                                max={100}
                                                step="1"
                                                value={Math.round((form.discount_rate || 0) * 100)}
                                                onChange={(e) => setDiscountPercent(e.target.value)}
                                            />
                                        </div>
                                        <div>
                                            <Label>{t("enterprise.quota.promoted.inputPrice" as never)}</Label>
                                            <div className="h-9 rounded-md border bg-muted px-3 py-2 text-sm">
                                                {formatTokenPrice(form.override_price.input_price, form.override_price.input_price_unit)}
                                            </div>
                                        </div>
                                        <div>
                                            <Label>{t("enterprise.quota.promoted.outputPrice" as never)}</Label>
                                            <div className="h-9 rounded-md border bg-muted px-3 py-2 text-sm">
                                                {formatTokenPrice(form.override_price.output_price, form.override_price.output_price_unit)}
                                            </div>
                                        </div>
                                    </div>
                                ) : (
                                    <div className="grid grid-cols-2 gap-3">
                                        <div>
                                            <Label>{t("enterprise.quota.promoted.inputPrice" as never)}</Label>
                                            <Input
                                                type="number"
                                                step="0.0000001"
                                                value={form.override_price.input_price ?? ""}
                                                onChange={(e) => setForm({
                                                    ...form,
                                                    override_price: { ...form.override_price, input_price: optionalNumber(e.target.value), input_price_unit: form.override_price.input_price_unit || 1 },
                                                })}
                                            />
                                        </div>
                                        <div>
                                            <Label>{t("enterprise.quota.promoted.outputPrice" as never)}</Label>
                                            <Input
                                                type="number"
                                                step="0.0000001"
                                                value={form.override_price.output_price ?? ""}
                                                onChange={(e) => setForm({
                                                    ...form,
                                                    override_price: { ...form.override_price, output_price: optionalNumber(e.target.value), output_price_unit: form.override_price.output_price_unit || 1 },
                                                })}
                                            />
                                        </div>
                                    </div>
                                )}
                            </div>
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <Label>{t("enterprise.quota.effectiveAt" as never)}</Label>
                                    <Input
                                        type="datetime-local"
                                        value={toDateTimeLocal(form.effective_at)}
                                        onChange={(e) => setForm({ ...form, effective_at: e.target.value })}
                                    />
                                </div>
                                <div>
                                    <Label>{t("enterprise.quota.expiresAt" as never)}</Label>
                                    <Input
                                        type="datetime-local"
                                        value={toDateTimeLocal(form.expires_at)}
                                        onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
                                    />
                                </div>
                            </div>
                            <div className="flex items-center justify-between rounded border p-3">
                                <div>
                                    <Label>{t("enterprise.quota.promoted.enabled" as never)}</Label>
                                    <p className="text-xs text-muted-foreground">{t("enterprise.quota.promoted.enabledHint" as never)}</p>
                                </div>
                                <Switch checked={form.enabled} onCheckedChange={(v) => setForm({ ...form, enabled: v })} />
                            </div>
                            <div className="flex items-center justify-between rounded border p-3">
                                <div>
                                    <Label>{t("enterprise.quota.promoted.lock" as never)}</Label>
                                    <p className="text-xs text-muted-foreground">{t("enterprise.quota.promoted.lockHint" as never)}</p>
                                </div>
                                <Switch checked={!!form.price_locked} onCheckedChange={(v) => setForm({ ...form, price_locked: v })} />
                            </div>
                            {editing?.price_locked && (
                                <div className="flex items-center justify-between rounded border border-amber-300 bg-amber-50 p-3 dark:bg-amber-950/20">
                                    <div>
                                        <Label>{t("enterprise.quota.promoted.overrideLocked" as never)}</Label>
                                        <p className="text-xs text-muted-foreground">{t("enterprise.quota.promoted.overrideLockedHint" as never)}</p>
                                    </div>
                                    <Switch checked={overrideLocked} onCheckedChange={setOverrideLocked} />
                                </div>
                            )}
                        </div>
                        <DialogFooter>
                            <Button variant="outline" onClick={() => setDialogOpen(false)}>{t("common.cancel")}</Button>
                            <Button onClick={save} disabled={createMutation.isPending || updateMutation.isPending}>
                                {(createMutation.isPending || updateMutation.isPending) ? t("common.saving") : t("common.save")}
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>

                <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
                    <AlertDialogContent>
                        <AlertDialogHeader>
                            <AlertDialogTitle>{t("enterprise.quota.promoted.deleteTitle" as never)}</AlertDialogTitle>
                            <AlertDialogDescription>
                                {t("enterprise.quota.promoted.deleteDesc", { model: deleteTarget?.model })}
                            </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
                            <AlertDialogAction
                                className="bg-red-500 hover:bg-red-600"
                                onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget)}
                                disabled={deleteMutation.isPending}
                            >
                                {deleteMutation.isPending ? t("common.deleting") : t("common.delete")}
                            </AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            </CardContent>
        </Card>
    )
}

// ─── Main Page ──────────────────────────────────────────────────────────────

export default function QuotaPoliciesPage() {
    const { t } = useTranslation()
    const queryClient = useQueryClient()
    const canManage = useHasPermission('quota_manage_manage')
    const [editingPolicy, setEditingPolicy] = useState<QuotaPolicy | null>(null)
    const [isCreating, setIsCreating] = useState(false)
    const [formData, setFormData] = useState<QuotaPolicyInput>(defaultPolicy)
    const [deleteTarget, setDeleteTarget] = useState<QuotaPolicy | null>(null)

    const { data, isLoading } = useQuery({
        queryKey: ["enterprise", "quota-policies"],
        queryFn: () => enterpriseApi.listQuotaPolicies(),
    })

    const createMutation = useMutation({
        mutationFn: enterpriseApi.createQuotaPolicy,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-policies"] })
            setIsCreating(false)
            setFormData(defaultPolicy)
            toast.success(t("enterprise.quota.createSuccess"))
        },
        onError: (err: Error) => {
            toast.error(err.message)
        },
    })

    const updateMutation = useMutation({
        mutationFn: ({ id, data }: { id: number; data: QuotaPolicyInput }) =>
            enterpriseApi.updateQuotaPolicy(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-policies"] })
            setEditingPolicy(null)
            toast.success(t("enterprise.quota.updateSuccess"))
        },
        onError: (err: Error) => {
            toast.error(err.message)
        },
    })

    const deleteMutation = useMutation({
        mutationFn: enterpriseApi.deleteQuotaPolicy,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enterprise", "quota-policies"] })
            setDeleteTarget(null)
            toast.success(t("enterprise.quota.deleteSuccess"))
        },
        onError: (err: Error) => {
            toast.error(err.message)
        },
    })

    const policies = data?.policies || []

    const periodTypeLabel = (pt: number) => {
        switch (pt) {
            case 1: return t("enterprise.quota.daily")
            case 2: return t("enterprise.quota.weekly")
            case 3: return t("enterprise.quota.monthly")
            default: return "-"
        }
    }

    const handleCreate = () => {
        setFormData(defaultPolicy)
        setIsCreating(true)
    }

    const handleEdit = (policy: QuotaPolicy) => {
        setEditingPolicy(policy)
        setFormData({
            name: policy.name,
            tier1_ratio: policy.tier1_ratio,
            tier2_ratio: policy.tier2_ratio,
            tier1_rpm_multiplier: policy.tier1_rpm_multiplier,
            tier1_tpm_multiplier: policy.tier1_tpm_multiplier,
            tier2_rpm_multiplier: policy.tier2_rpm_multiplier,
            tier2_tpm_multiplier: policy.tier2_tpm_multiplier,
            tier3_rpm_multiplier: policy.tier3_rpm_multiplier,
            tier3_tpm_multiplier: policy.tier3_tpm_multiplier,
            block_at_tier3: policy.block_at_tier3,
            tier2_blocked_models: policy.tier2_blocked_models || "",
            tier3_blocked_models: policy.tier3_blocked_models || "",
            tier2_price_input_threshold: policy.tier2_price_input_threshold || 0,
            tier2_price_output_threshold: policy.tier2_price_output_threshold || 0,
            tier2_price_condition: policy.tier2_price_condition || "or",
            tier3_price_input_threshold: policy.tier3_price_input_threshold || 0,
            tier3_price_output_threshold: policy.tier3_price_output_threshold || 0,
            tier3_price_condition: policy.tier3_price_condition || "or",
            period_quota: policy.period_quota,
            period_type: policy.period_type,
        })
    }

    const handleSave = () => {
        if (!formData.name.trim()) {
            toast.error(t("enterprise.quota.nameRequired"))
            return
        }
        if (formData.tier1_ratio <= 0 || formData.tier1_ratio >= formData.tier2_ratio || formData.tier2_ratio > 1) {
            toast.error(t("enterprise.quota.ratioError"))
            return
        }

        if (editingPolicy) {
            updateMutation.mutate({ id: editingPolicy.id, data: formData })
        } else {
            createMutation.mutate(formData)
        }
    }

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold">{t("enterprise.quota.title")}</h1>
                    <p className="text-muted-foreground">{t("enterprise.quota.description")}</p>
                </div>
            </div>

            <Tabs defaultValue="policies">
                <TabsList>
                    <TabsTrigger value="policies">
                        <Shield className="w-4 h-4 mr-1.5" />
                        {t("enterprise.quota.policyListTab")}
                    </TabsTrigger>
                    <TabsTrigger value="departments">
                        <Building2 className="w-4 h-4 mr-1.5" />
                        {t("enterprise.quota.departmentBinding")}
                    </TabsTrigger>
                    <TabsTrigger value="users">
                        <User className="w-4 h-4 mr-1.5" />
                        {t("enterprise.quota.userOverride")}
                    </TabsTrigger>
                    <TabsTrigger value="promoted">
                        <Star className="w-4 h-4 mr-1.5" />
                        {t("enterprise.quota.promoted.tab" as never)}
                    </TabsTrigger>
                    <TabsTrigger value="notif">
                        <Bell className="w-4 h-4 mr-1.5" />
                        {t("enterprise.quota.notif.tab" as never)}
                    </TabsTrigger>
                </TabsList>

                {/* Tab 1: Policy List */}
                <TabsContent value="policies">
                    <Card>
                        <CardHeader>
                            <div className="flex items-center justify-between">
                                <CardTitle className="flex items-center gap-2">
                                    <Shield className="w-5 h-5" />
                                    {t("enterprise.quota.policyList")} ({policies.length})
                                </CardTitle>
                                {canManage && (
                                    <Button onClick={handleCreate}>
                                        <Plus className="w-4 h-4 mr-2" />
                                        {t("enterprise.quota.createPolicy")}
                                    </Button>
                                )}
                            </div>
                        </CardHeader>
                        <CardContent>
                            {isLoading ? (
                                <div className="text-center py-8 text-muted-foreground">{t("common.loading")}</div>
                            ) : policies.length === 0 ? (
                                <div className="text-center py-8 text-muted-foreground">{t("enterprise.quota.noPolicies")}</div>
                            ) : (
                                <Table>
                                    <TableHeader>
                                        <TableRow>
                                            <TableHead>{t("enterprise.quota.name")}</TableHead>
                                            <TableHead>{t("enterprise.quota.periodQuota")}</TableHead>
                                            <TableHead>{t("enterprise.quota.thresholds")}</TableHead>
                                            <TableHead>{t("enterprise.quota.tier1")}</TableHead>
                                            <TableHead>{t("enterprise.quota.tier2")}</TableHead>
                                            <TableHead>{t("enterprise.quota.tier3")}</TableHead>
                                            <TableHead className="w-24">{t("common.edit")}</TableHead>
                                        </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                        {policies.map((policy) => (
                                            <TableRow key={policy.id}>
                                                <TableCell className="font-medium">{policy.name}</TableCell>
                                                <TableCell>
                                                    {policy.period_quota > 0 ? (
                                                        <div className="text-sm">
                                                            <span className="font-medium">{policy.period_quota}</span>
                                                            <span className="text-muted-foreground ml-1">/ {periodTypeLabel(policy.period_type)}</span>
                                                        </div>
                                                    ) : (
                                                        <span className="text-xs text-muted-foreground">-</span>
                                                    )}
                                                </TableCell>
                                                <TableCell>
                                                    <div className="space-y-1">
                                                        <TierIndicator ratio={policy.tier1_ratio} label="T1" />
                                                        <TierIndicator ratio={policy.tier2_ratio} label="T2" />
                                                    </div>
                                                </TableCell>
                                                <TableCell>
                                                    <div className="text-xs space-y-0.5">
                                                        <div>RPM: {policy.tier1_rpm_multiplier}x</div>
                                                        <div>TPM: {policy.tier1_tpm_multiplier}x</div>
                                                    </div>
                                                </TableCell>
                                                <TableCell>
                                                    <div className="text-xs space-y-0.5">
                                                        <div>RPM: {policy.tier2_rpm_multiplier}x</div>
                                                        <div>TPM: {policy.tier2_tpm_multiplier}x</div>
                                                    </div>
                                                </TableCell>
                                                <TableCell>
                                                    {policy.block_at_tier3 ? (
                                                        <span className="text-xs text-red-500 font-medium">{t("enterprise.quota.blocked")}</span>
                                                    ) : (
                                                        <div className="text-xs space-y-0.5">
                                                            <div>RPM: {policy.tier3_rpm_multiplier}x</div>
                                                            <div>TPM: {policy.tier3_tpm_multiplier}x</div>
                                                        </div>
                                                    )}
                                                </TableCell>
                                                <TableCell>
                                                    {canManage && (
                                                        <div className="flex items-center gap-1">
                                                            <Button variant="ghost" size="icon" onClick={() => handleEdit(policy)}>
                                                                <Pencil className="w-4 h-4" />
                                                            </Button>
                                                            <Button
                                                                variant="ghost"
                                                                size="icon"
                                                                className="text-red-500 hover:text-red-600"
                                                                onClick={() => setDeleteTarget(policy)}
                                                            >
                                                                <Trash2 className="w-4 h-4" />
                                                            </Button>
                                                        </div>
                                                    )}
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            )}
                        </CardContent>
                    </Card>
                </TabsContent>

                {/* Tab 2: Department Binding */}
                <TabsContent value="departments">
                    <DepartmentBindingTab policies={policies} canManage={canManage} />
                </TabsContent>

                {/* Tab 3: User Override */}
                <TabsContent value="users">
                    <UserOverrideTab policies={policies} canManage={canManage} />
                </TabsContent>

                {/* Tab 4: Promoted Models */}
                <TabsContent value="promoted">
                    <PromotedModelsTab policies={policies} canManage={canManage} />
                </TabsContent>

                {/* Tab 5: Notification Settings */}
                <TabsContent value="notif">
                    <NotifConfigTab canManage={canManage} />
                </TabsContent>
            </Tabs>

            {/* Create/Edit Dialog */}
            <Dialog open={isCreating || !!editingPolicy} onOpenChange={(open) => {
                if (!open) {
                    setIsCreating(false)
                    setEditingPolicy(null)
                    setFormData(defaultPolicy)
                }
            }}>
                <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>
                            {editingPolicy ? t("enterprise.quota.editPolicy") : t("enterprise.quota.createPolicy")}
                        </DialogTitle>
                        <DialogDescription>
                            {t("enterprise.quota.formDescription")}
                        </DialogDescription>
                    </DialogHeader>
                    <PolicyForm policy={formData} onChange={setFormData} />
                    <DialogFooter>
                        <Button variant="outline" onClick={() => { setIsCreating(false); setEditingPolicy(null) }}>
                            {t("common.cancel")}
                        </Button>
                        <Button
                            onClick={handleSave}
                            disabled={createMutation.isPending || updateMutation.isPending}
                        >
                            {(createMutation.isPending || updateMutation.isPending) ? t("common.saving") : t("common.save")}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation */}
            <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t("enterprise.quota.deleteConfirmTitle")}</AlertDialogTitle>
                        <AlertDialogDescription>
                            {t("enterprise.quota.deleteConfirmDesc", { name: deleteTarget?.name })}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
                        <AlertDialogAction
                            className="bg-red-500 hover:bg-red-600"
                            onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
                            disabled={deleteMutation.isPending}
                        >
                            {deleteMutation.isPending ? t("common.deleting") : t("common.delete")}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}
