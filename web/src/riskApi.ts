import { requestJSON } from "./http";
import type { RiskAggregate, RiskAppetitePosition, RiskPage, RiskStatus } from "./riskTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type RiskListParams = {
  status?: RiskStatus;
  category?: string;
  ownerPrincipalID?: string;
  search?: string;
  appetitePosition?: RiskAppetitePosition;
  cursor?: string;
  limit?: number;
};

export function listRisks(params: RiskListParams = {}, signal?: AbortSignal): Promise<RiskPage> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.category?.trim()) query.set("category", params.category.trim());
  if (params.ownerPrincipalID?.trim()) query.set("owner_principal_id", params.ownerPrincipalID.trim());
  if (params.search?.trim()) query.set("search", params.search.trim());
  if (params.appetitePosition) query.set("appetite_position", params.appetitePosition);
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  const suffix = query.size ? `?${query.toString()}` : "";
  return requestJSON<RiskPage>(apiBase, `/api/v1/risks${suffix}`, signal ? { signal } : undefined);
}

export function getRisk(id: string, signal?: AbortSignal): Promise<RiskAggregate> {
  return requestJSON<RiskAggregate>(
    apiBase,
    `/api/v1/risks/${encodeURIComponent(id)}`,
    signal ? { signal } : undefined,
  );
}
