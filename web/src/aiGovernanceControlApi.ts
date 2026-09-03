import { loadContext } from "./api";
import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type GatewayBaselineAction = "DENY" | "REQUIRE_APPROVAL";

export type GatewayBaselineDraftInput = {
  code: string;
  name: string;
  organizationInstruction: string;
  highRiskAction: GatewayBaselineAction;
  blockInstructionExfiltration: boolean;
};

export type GatewayBaselinePolicy = {
  id: string;
  code: string;
  name: string;
  status: string;
  rollout_mode: "SHADOW" | "ENFORCE" | string;
  version: number;
  record_version: number;
};

export async function createGatewayBaselineDraft(input: GatewayBaselineDraftInput): Promise<GatewayBaselinePolicy> {
  const context = await loadContext();
  const rules = [
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
  return requestJSON<GatewayBaselinePolicy>(apiBase, "/api/v1/ai-governance/policies", {
    method: "POST",
    body: JSON.stringify({
      tenant_id: context.tenant.id,
      code: input.code.trim(),
      name: input.name.trim(),
      action_class: "AI_GATEWAY_BASELINE",
      eligibility: { scope: "ORGANIZATION", environments: ["production", "test", "development"] },
      blast_radius_limit: { scope: "ALL_REGISTERED_AI_WORKLOADS" },
      verification_contract: { activation: "MAKER_CHECKER", rollout: "SHADOW_FIRST" },
      rollout_mode: "SHADOW",
      definition: {
        rules,
        default_action: "ALLOW",
        response_control: { allow_streaming: true },
      },
    }),
  });
}

export async function submitGatewayBaselineDraft(policyId: string, expectedVersion: number): Promise<GatewayBaselinePolicy> {
  return requestJSON<GatewayBaselinePolicy>(apiBase, `/api/v1/ai-governance/policies/${encodeURIComponent(policyId)}/submit`, {
    method: "POST",
    body: JSON.stringify({ expected_version: expectedVersion }),
  });
}
