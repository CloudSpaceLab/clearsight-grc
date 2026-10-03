import { requestJSON } from "./http";
import type { RiskAggregate, RiskAppetitePosition, RiskControlLink, RiskIndicatorKind, RiskIndicatorLink, RiskPage, RiskRecord, RiskStatus } from "./riskTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type RiskListParams = {
  status?: RiskStatus;
  category?: string;
  ownerPrincipalID?: string;
  search?: string;
  appetitePosition?: RiskAppetitePosition;
  organizationScopeID?: string;
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
  if (params.organizationScopeID?.trim()) query.set("organization_scope_id", params.organizationScopeID.trim());
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


export type LinkRiskControlResponse = {
  risk: RiskRecord;
  control: RiskControlLink;
};

export function linkRiskControl(riskID: string, expectedRiskVersion: number, catalogLinkID: string): Promise<LinkRiskControlResponse> {
  return requestJSON<LinkRiskControlResponse>(
    apiBase,
    `/api/v1/risks/${encodeURIComponent(riskID)}/controls`,
    {
      method: "POST",
      body: JSON.stringify({
        expected_risk_version: expectedRiskVersion,
        catalog_link_id: catalogLinkID,
      }),
    },
  );
}


export type LinkRiskIndicatorResponse = {
  risk: RiskRecord;
  indicator: RiskIndicatorLink;
};

export function linkRiskIndicator(
  riskID: string,
  expectedRiskVersion: number,
  monitoringCheckID: string,
  monitoringCheckVersion: number,
  kind: RiskIndicatorKind,
): Promise<LinkRiskIndicatorResponse> {
  return requestJSON<LinkRiskIndicatorResponse>(
    apiBase,
    `/api/v1/risks/${encodeURIComponent(riskID)}/indicators`,
    {
      method: "POST",
      body: JSON.stringify({
        expected_risk_version: expectedRiskVersion,
        monitoring_check_id: monitoringCheckID,
        monitoring_check_version: monitoringCheckVersion,
        kind,
      }),
    },
  );
}
