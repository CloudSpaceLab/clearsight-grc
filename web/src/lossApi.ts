import { requestJSON } from "./http";
import type { LossAggregate, LossCreateInput, LossEventType, LossInterventionResponse, LossPage, LossRecord, LossRecoveryInput, LossRecoveryResponse, LossRecoveryStatus, LossStatus } from "./lossTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type LossListParams = {
  status?: LossStatus;
  eventType?: LossEventType;
  currency?: string;
  organizationScopeID?: string;
  riskID?: string;
  matterID?: string;
  recoveryStatus?: LossRecoveryStatus;
  search?: string;
  cursor?: string;
  limit?: number;
};

export function listLosses(params: LossListParams = {}, signal?: AbortSignal): Promise<LossPage> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.eventType) query.set("event_type", params.eventType);
  if (params.currency?.trim()) query.set("currency", params.currency.trim());
  if (params.organizationScopeID?.trim()) query.set("organization_scope_id", params.organizationScopeID.trim());
  if (params.riskID?.trim()) query.set("risk_id", params.riskID.trim());
  if (params.matterID?.trim()) query.set("matter_id", params.matterID.trim());
  if (params.recoveryStatus) query.set("recovery_status", params.recoveryStatus);
  if (params.search?.trim()) query.set("search", params.search.trim());
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  const suffix = query.size ? `?${query.toString()}` : "";
  return requestJSON<LossPage>(apiBase, `/api/v1/losses${suffix}`, signal ? { signal } : undefined);
}

export function getLoss(id: string, signal?: AbortSignal): Promise<LossAggregate> {
  return requestJSON<LossAggregate>(
    apiBase,
    `/api/v1/losses/${encodeURIComponent(id)}`,
    signal ? { signal } : undefined,
  );
}

export function openLossIntervention(id: string, expectedVersion: number): Promise<LossInterventionResponse> {
  return requestJSON<LossInterventionResponse>(
    apiBase,
    `/api/v1/losses/${encodeURIComponent(id)}/intervention`,
    {
      method: "POST",
      body: JSON.stringify({ expected_version: expectedVersion }),
    },
  );
}

export function createLoss(input: LossCreateInput): Promise<LossRecord> {
  return requestJSON<LossRecord>(
    apiBase,
    "/api/v1/losses",
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );
}

export function recordLossRecovery(id: string, input: LossRecoveryInput): Promise<LossRecoveryResponse> {
  return requestJSON<LossRecoveryResponse>(
    apiBase,
    `/api/v1/losses/${encodeURIComponent(id)}/recoveries`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );
}
