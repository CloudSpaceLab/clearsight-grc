import { requestJSON } from "./http";
import type { IndicatorPopulationPage, IndicatorPopulationParams } from "./indicatorTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export function loadIndicatorPopulation(params: IndicatorPopulationParams = {}, signal?: AbortSignal): Promise<IndicatorPopulationPage> {
  const query = new URLSearchParams();
  if (params.kind) query.set("kind", params.kind);
  if (params.organizationScopeID?.trim()) query.set("organization_scope_id", params.organizationScopeID.trim());
  if (params.checkID?.trim()) query.set("check_id", params.checkID.trim());
  if (params.cursor?.trim()) query.set("cursor", params.cursor.trim());
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  const suffix = query.size ? `?${query.toString()}` : "";
  return requestJSON<IndicatorPopulationPage>(apiBase, `/api/v1/risk-indicators${suffix}`, signal ? { signal } : undefined);
}
