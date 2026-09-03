import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
const basePath = "/api/v1/ai-governance/gateway-configs";

export type GatewayEnvironment = "PRODUCTION" | "TEST" | "DEVELOPMENT";
export type GatewayProviderKind = "OPENAI" | "ANTHROPIC";
export type GatewayProviderState = "ENABLED" | "SUSPENDED";
export type GatewayTransportStatus = "DRAFT" | "PENDING_APPROVAL" | "APPROVED" | "ACTIVE" | "SUSPENDED" | "RETIRED";
export type GatewayTransportTransition = "submit" | "approve" | "activate" | "suspend" | "retire";

export type GatewayProviderConfig = {
  id: string;
  name: string;
  kind: GatewayProviderKind;
  base_url: string;
  secret_ref: string;
  api_version?: string;
  timeout_ms?: number;
  require_usage?: boolean;
  regions?: string[];
  state: GatewayProviderState;
};

export type GatewayRouteConfig = {
  id: string;
  provider_id: string;
  model: string;
  weight: number;
  input_microusd_per_million_tokens: number;
  output_microusd_per_million_tokens: number;
};

export type GatewayModelConfig = {
  alias: string;
  routes: GatewayRouteConfig[];
};

export type GatewayTransportDefinition = {
  circuit_breaker: {
    failure_threshold: number;
    open_duration_ms: number;
  };
  providers: GatewayProviderConfig[];
  models: GatewayModelConfig[];
};

export type GatewayTransportRevision = {
  id: string;
  tenant_id: string;
  environment: GatewayEnvironment;
  definition: GatewayTransportDefinition;
  status: GatewayTransportStatus;
  maker_id: string;
  checker_id?: string;
  change_reason: string;
  checksum: string;
  version: number;
  record_version: number;
  submitted_at?: string;
  approved_at?: string;
  activated_at?: string;
  suspended_at?: string;
  retired_at?: string;
  created_at: string;
  updated_at: string;
};

export type GatewayRuntimeStatus = {
  configured: boolean;
  available: boolean;
  tenant_id: string;
  environment: GatewayEnvironment | "";
  desired_revision: number;
  desired_checksum?: string;
  applied_revision: number;
  applied_checksum?: string;
  degraded: boolean;
  error_code?: string;
};

export type GatewayTransportControlState = {
  revisions: GatewayTransportRevision[];
  runtimeStatus: GatewayRuntimeStatus;
};

export async function loadGatewayTransportState(environment: GatewayEnvironment): Promise<GatewayTransportControlState> {
  const response = await requestJSON<{ items: GatewayTransportRevision[]; runtime_status: GatewayRuntimeStatus }>(apiBase, `${basePath}?environment=${environment}&limit=100`);
  return {
    revisions: response.items.sort((left, right) => right.version - left.version),
    runtimeStatus: response.runtime_status,
  };
}

export async function loadActiveGatewayTransport(environment: GatewayEnvironment): Promise<GatewayTransportRevision | null> {
  try {
    return await requestJSON<GatewayTransportRevision>(apiBase, `${basePath}/active?environment=${environment}`);
  } catch (error) {
    if (error instanceof Error && /not found/i.test(error.message)) return null;
    throw error;
  }
}

export async function createGatewayTransportRevision(input: {
  environment: GatewayEnvironment;
  definition: GatewayTransportDefinition;
  changeReason: string;
}): Promise<GatewayTransportRevision> {
  return requestJSON<GatewayTransportRevision>(apiBase, basePath, {
    method: "POST",
    body: JSON.stringify({
      environment: input.environment,
      definition: input.definition,
      change_reason: input.changeReason.trim(),
    }),
  });
}

export async function transitionGatewayTransport(
  revisionId: string,
  action: GatewayTransportTransition,
  expectedVersion: number,
): Promise<GatewayTransportRevision> {
  return requestJSON<GatewayTransportRevision>(apiBase, `${basePath}/${encodeURIComponent(revisionId)}/${action}`, {
    method: "POST",
    body: JSON.stringify({ expected_version: expectedVersion }),
  });
}
