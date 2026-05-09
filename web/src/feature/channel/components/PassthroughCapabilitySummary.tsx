import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ShieldCheck } from 'lucide-react'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import type { ChannelPassthroughCapability } from '@/types/channel'

interface PassthroughCapabilitySummaryProps {
    capability?: ChannelPassthroughCapability | null
    configsText?: string
    onConfigsTextChange?: (value: string) => void
}

type ConfigObject = Record<string, unknown>
type RouteKind = 'pure_passthrough' | 'adapted_passthrough' | 'conversion'

function parseConfigObject(value?: string): ConfigObject {
    const raw = value?.trim()
    if (!raw) return {}

    try {
        const parsed = JSON.parse(raw) as unknown
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
            return {}
        }
        return parsed as ConfigObject
    } catch {
        return {}
    }
}

function stringList(value: unknown): string[] {
    if (!Array.isArray(value)) return []
    return value.filter((item): item is string => typeof item === 'string' && item.length > 0)
}

function unique(values: string[]): string[] {
    return Array.from(new Set(values))
}

function normalizeRouteKind(value: unknown, supportsPure: boolean): RouteKind {
    if (value === 'pure_passthrough' || value === 'adapted_passthrough' || value === 'conversion') {
        return value
    }
    return supportsPure ? 'pure_passthrough' : 'conversion'
}

function stringifyConfig(config: ConfigObject) {
    const cleaned = Object.fromEntries(
        Object.entries(config).filter(([, value]) => value !== undefined && value !== null && value !== '')
    )
    return Object.keys(cleaned).length === 0 ? '' : JSON.stringify(cleaned, null, 2)
}

export function PassthroughCapabilitySummary({
    capability,
    configsText,
    onConfigsTextChange,
}: PassthroughCapabilitySummaryProps) {
    const { t } = useTranslation()

    const summary = useMemo(() => {
        const configs = parseConfigObject(configsText)
        const pureFamilies = unique([
            ...(capability?.endpointFamilies || []),
            ...stringList(configs.passthrough_endpoint_families),
        ])
        const adaptedFamilies = unique([
            ...(capability?.adaptedEndpointFamilies || []),
            ...stringList(configs.adapted_passthrough_endpoint_families),
        ])
        const supportsPure = pureFamilies.length > 0 || Boolean(capability?.purePassthrough)
        const supportsAdapted = adaptedFamilies.length > 0

        return {
            routeKind: normalizeRouteKind(configs.route_kind, supportsPure),
            supportsPure,
            supportsAdapted,
        }
    }, [capability, configsText])

    const updateRouteKind = (routeKind: string) => {
        const configs = parseConfigObject(configsText)
        if (routeKind === 'pure_passthrough' && capability?.purePassthrough) {
            delete configs.route_kind
        } else {
            configs.route_kind = routeKind
        }
        onConfigsTextChange?.(stringifyConfig(configs))
    }
    const selectedDescription = (() => {
        switch (summary.routeKind) {
            case 'adapted_passthrough':
                return t('channel.dialog.passthrough.adapted_passthrough')
            case 'conversion':
                return t('channel.dialog.passthrough.conversion')
            case 'pure_passthrough':
            default:
                return t('channel.dialog.passthrough.pure_passthrough')
        }
    })()

    const capabilityHint = (() => {
        if (summary.supportsPure && summary.supportsAdapted) {
            return t('channel.dialog.passthrough.supportsPureAndAdapted')
        }
        if (summary.supportsPure) {
            return t('channel.dialog.passthrough.supportsPureOnly')
        }
        if (summary.supportsAdapted) {
            return t('channel.dialog.passthrough.supportsAdaptedOnly')
        }
        return t('channel.dialog.passthrough.supportsConversionOnly')
    })()

    return (
        <div className="rounded-lg border bg-muted/30 p-4">
            <div className="space-y-3">
                <div className="space-y-1">
                    <div className="flex items-center gap-2 text-sm font-medium">
                        <ShieldCheck className="h-4 w-4 text-primary" />
                        {t('channel.dialog.passthrough.title')}
                    </div>
                    <p className="text-xs text-muted-foreground">
                        {t('channel.dialog.passthrough.description')}
                    </p>
                </div>

                <div className="space-y-2">
                    <p className="text-xs font-medium text-muted-foreground">
                        {t('channel.dialog.passthrough.routeMode')}
                    </p>
                    <Select value={summary.routeKind} onValueChange={updateRouteKind}>
                        <SelectTrigger>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="pure_passthrough" disabled={!summary.supportsPure}>
                                {summary.supportsPure
                                    ? t('channel.dialog.passthrough.modePure')
                                    : t('channel.dialog.passthrough.modePureUnavailable')}
                            </SelectItem>
                            <SelectItem value="adapted_passthrough" disabled={!summary.supportsAdapted}>
                                {summary.supportsAdapted
                                    ? t('channel.dialog.passthrough.modeAdapted')
                                    : t('channel.dialog.passthrough.modeAdaptedUnavailable')}
                            </SelectItem>
                            <SelectItem value="conversion">
                                {t('channel.dialog.passthrough.modeConversion')}
                            </SelectItem>
                        </SelectContent>
                    </Select>
                </div>

                <p className="text-xs text-muted-foreground">
                    {selectedDescription}
                </p>

                <p className="text-xs text-muted-foreground">
                    {capabilityHint}
                </p>
            </div>
        </div>
    )
}
