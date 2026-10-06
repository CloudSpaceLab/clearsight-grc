import { requestJSON } from "./http";
import type { IndicatorDetailBase, RiskIndicatorKind } from "./riskTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type IndicatorInsight = IndicatorDetailBase & {
  risk_count: number;
};

export type IndicatorInsightsPage = {
  items: IndicatorInsight[];
  next_cursor?: string;
  complete: boolean;
  generated_at: string;
};

export type IndicatorInsightsParams = {
  kind?: RiskIndicatorKind;
  cursor?: string;
  limit?: number;
};

export function loadIndicatorInsights(params: IndicatorInsightsParams = {}, signal?: AbortSignal): Promise<IndicatorInsightsPage> {
  const query = new URLSearchParams();
  if (params.kind) query.set("kind", params.kind);
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  const suffix = query.size ? `?${query.toString()}` : "";
  return requestJSON<IndicatorInsightsPage>(
    apiBase,
    `/api/v1/insights/indicators${suffix}`,
    signal ? { signal } : undefined,
  );
}
