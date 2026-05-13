import apiClient, { get } from "@/api"
import type {
    DepartmentSummary,
    DepartmentSummaryResponse,
    ModelDistributionResponse,
    UserRankingResponse,
} from "@/api/enterprise"

export type AnalyticsGranularityV2 = "hourly" | "daily" | "monthly"

export interface AnalyticsV2Params {
    startTimestamp?: number
    endTimestamp?: number
    granularity?: AnalyticsGranularityV2
    orgUnitIds?: string[]
    groupIds?: string[]
    userIds?: string[]
    models?: string[]
    limit?: number
    page?: number
    perPage?: number
}

export type DepartmentSummaryV2 = DepartmentSummary
export type DepartmentSummaryResponseV2 = DepartmentSummaryResponse
export type UserRankingResponseV2 = UserRankingResponse
export type ModelDistributionResponseV2 = ModelDistributionResponse

function buildAnalyticsV2Params(params?: AnalyticsV2Params): URLSearchParams {
    const search = new URLSearchParams()
    if (!params) return search

    if (params.startTimestamp !== undefined) search.set("start_timestamp", String(params.startTimestamp))
    if (params.endTimestamp !== undefined) search.set("end_timestamp", String(params.endTimestamp))
    if (params.granularity) search.set("granularity", params.granularity)
    if (params.limit !== undefined) search.set("limit", String(params.limit))
    if (params.page !== undefined) search.set("page", String(params.page))
    if (params.perPage !== undefined) search.set("per_page", String(params.perPage))

    appendRepeated(search, "org_unit_id", params.orgUnitIds)
    appendRepeated(search, "group_id", params.groupIds)
    appendRepeated(search, "user_id", params.userIds)
    appendRepeated(search, "model", params.models)

    return search
}

function appendRepeated(search: URLSearchParams, key: string, values?: string[]) {
    for (const value of values ?? []) {
        if (value) search.append(key, value)
    }
}

export const enterpriseAnalyticsV2Api = {
    getDepartmentSummaryV2: (params?: AnalyticsV2Params): Promise<DepartmentSummaryResponseV2> => {
        return get<DepartmentSummaryResponseV2>("/enterprise/analytics/v2/department", {
            params: buildAnalyticsV2Params(params),
        })
    },

    getUserRankingV2: (params?: AnalyticsV2Params): Promise<UserRankingResponseV2> => {
        return get<UserRankingResponseV2>("/enterprise/analytics/v2/user/ranking", {
            params: buildAnalyticsV2Params(params),
        })
    },

    getModelDistributionV2: (params?: AnalyticsV2Params): Promise<ModelDistributionResponseV2> => {
        return get<ModelDistributionResponseV2>("/enterprise/analytics/v2/model/distribution", {
            params: buildAnalyticsV2Params(params),
        })
    },

    exportAnalyticsV2: async (params?: AnalyticsV2Params): Promise<void> => {
        const response = await apiClient.get("/enterprise/analytics/v2/export", {
            params: buildAnalyticsV2Params(params),
            responseType: "blob",
        })
        const url = window.URL.createObjectURL(new Blob([response.data as BlobPart]))
        const link = document.createElement("a")
        link.href = url
        const disposition = response.headers["content-disposition"]
        const filename = disposition
            ? disposition.split("filename=")[1]?.replace(/"/g, "")
            : "enterprise_analytics_v2.xlsx"
        link.setAttribute("download", filename)
        document.body.appendChild(link)
        link.click()
        link.remove()
        window.URL.revokeObjectURL(url)
    },
}
