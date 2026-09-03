import { loadContext } from "./api";
import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
const gatewayBaselineCode = "ORG_AI_BASELINE";
const gatewayBaselineActionClass = "AI_GATEWAY_BASELINE";

export type GatewayBaselineAction = "DENY" | "REQUIRE_APPROVAL";
export type GatewayBaselineTransition = "submit" | "approve" | "activate" | "suspend" | "retire";

export type GatewayBaselineDraftInput = {
  name: string;
  organizationInstruction: string;
  highRiskAction: GatewayBaselineAction;
  blockInstructionExfiltration: boolean;
};

type GatewayPolicyDefinition = {
  bindings?: Array<Record<string, unknown>>;
  rules?: Array<Record<string, unknown>>;
  default_action?: string;
  response_control?: Record<string, unknown>;
};

export type GatewayBaselinePolicy = {
  id: string;
  tenant_id: string;
  code: string;
  name: string;
  action_class: string;
  eligibility: Record<string, unknown>;
  blast_radius_limit: Record<string, unknown>;
  verification_contract: Record<string, unknown>;
  definition: GatewayPolicyDefinition;
  status: string;
  rollout_mode: "SHADOW" | "ENFORCE" | string;
  maker_id: string;
  checker_id?: string;
  version: number;
  record_version: number;
};

export async function loadGatewayBaselines(): Promise<GatewayBaselinePolicy[]> {
  const response = await requestJSON<{ items: GatewayBaselinePolicy[] }>(apiBase, "/api/v1/ai-governance/policies?limit=100");
  return response.items
    .filter((policy) => policy.code === gatewayBaselineCode && policy.action_class === gatewayBaselineActionClass)
    .sort((left, right) => right.version - left.version);
}

export async function createGatewayBaselineDraft(input: GatewayBaselineDraftInput): Promise<GatewayBaselinePolicy> {
  const context = await loadContext();
  const rules: Array<Record<string, unknown>> = [
    {
      id: "org-baseline-instruction",
      priority: 1,
      fact_key: "gateway.prompt_injection_risk",
      operator: "EXISTS",
      action: "ALLOW",
      reason_code: "ORG_BASELINE_APPLIED",
      obligations: [{ code: "ORG_INSTRUCTION", detail: input.organizationInstruction.trim() }],
    },
    {
      id: "prompt-injection-high",
      priority: 100,
      fact_key: "gateway.prompt_injection_risk",
      operator: "EQ",
      value: "HIGH",
      action: input.highRiskAction,
      reason_code: "PROMPT_INJECTION_HIGH",
    },
  ];
  if (input.blockInstructionExfiltration) {
    rules.push({
      id: "instruction-exfiltration",
      priority: 110,
      fact_key: "gateway.instruction_exfiltration_attempt",
      operator: "EQ",
      value: "true",
      action: "DENY",
      reason_code: "INSTRUCTION_EXFILTRATION_ATTEMPT",
    });
  }
  return createGatewayBaselineRevision({
    tenantId: context.tenant.id,
    name: input.name,
    rolloutMode: "SHADOW",
    definition: { rules, default_action: "ALLOW", response_control: { allow_streaming: true } },
  });
}

export async function createGatewayEnforcementRevision(source: GatewayBaselinePolicy): Promise<GatewayBaselinePolicy> {
  const context = await loadContext();
  return createGatewayBaselineRevision({
    tenantId: context.tenant.id,
    name: source.name,
    rolloutMode: "ENFORCE",
    definition: source.definition,
    eligibility: source.eligibility,
    blastRadiusLimit: source.blast_radius_limit,
    verificationContract: source.verification_contract,
  });
}

export async function transitionGatewayBaseline(policyId: string, action: GatewayBaselineTransition, expectedVersion: number): Promise<GatewayBaselinePolicy> {
  return requestJSON<GatewayBaselinePolicy>(apiBase, `/api/v1/ai-governance/policies/${encodeURIComponent(policyId)}/${action}`, {
    method: "POST",
    body: JSON.stringify({ expected_version: expectedVersion }),
  });
}

async function createGatewayBaselineRevision(input: {
  tenantId: string;
  name: string;
  rolloutMode: "SHADOW" | "ENFORCE";
  definition: GatewayPolicyDefinition;
  eligibility?: Record<string, unknown>;
  blastRadiusLimit?: Record<string, unknown>;
  verificationContract?: Record<string, unknown>;
}): Promise<GatewayBaselinePolicy> {
  return requestJSON<GatewayBaselinePolicy>(apiBase, "/api/v1/ai-governance/policies", {
    method: "POST",
    body: JSON.stringify({
      tenant_id: input.tenantId,
      code: gatewayBaselineCode,
      name: input.name.trim(),
      action_class: gatewayBaselineActionClass,
      eligibility: input.eligibility ?? { scope: "ORGANIZATION", environments: ["production", "test", "development"] },
      blast_radius_limit: input.blastRadiusLimit ?? { scope: "ALL_REGISTERED_AI_WORKLOADS" },
      verification_contract: input.verificationContract ?? { activation: "MAKER_CHECKER", rollout: "SHADOW_FIRST" },
      rollout_mode: input.rolloutMode,
      definition: input.definition,
    }),
  });
}
