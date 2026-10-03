import { loadContext } from "./api";
import { requestJSON } from "./http";
import type { AIGovernancePolicy } from "./types";
import type { GatewayBaselinePolicy } from "./aiGovernanceControlApi";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
export const gatewayExceptionCodeRoot = "ORG_AI_BASELINE_EXCEPTION";
export const gatewayExceptionActionClass = "AI_GATEWAY_BASELINE_EXCEPTION";

export type GatewayExceptionTransition = "submit" | "approve" | "activate" | "suspend" | "retire";

export type GatewayExceptionDraftInput = {
  name: string;
  baseline: GatewayBaselinePolicy;
  workloadRecordIds: string[];
  environments: string[];
  waivedRuleIds: string[];
  justification: string;
  effectiveUntil: string;
};

export function isGatewayExceptionPolicy(policy: AIGovernancePolicy) {
  return policy.code.startsWith(`${gatewayExceptionCodeRoot}:`) && policy.action_class === gatewayExceptionActionClass;
}

export async function createGatewayExceptionShadowDraft(input: GatewayExceptionDraftInput): Promise<AIGovernancePolicy> {
  const context = await loadContext();
  const code = `${gatewayExceptionCodeRoot}:${crypto.randomUUID()}`;
  return createGatewayExceptionRevision({
    tenantId: context.tenant.id,
    code,
    name: input.name,
    rolloutMode: "SHADOW",
    baseline: input.baseline,
    workloadRecordIds: input.workloadRecordIds,
    environments: input.environments,
    waivedRuleIds: input.waivedRuleIds,
    justification: input.justification,
    effectiveUntil: input.effectiveUntil,
  });
}

export async function createGatewayExceptionEnforcementRevision(source: AIGovernancePolicy): Promise<AIGovernancePolicy> {
  const context = await loadContext();
  const scope = source.eligibility as GatewayExceptionScope;
  return createGatewayExceptionRevision({
    tenantId: context.tenant.id,
    code: source.code,
    name: source.name,
    rolloutMode: "ENFORCE",
    baseline: {
      id: scope.target_baseline_id,
      version: scope.target_baseline_version,
    } as GatewayBaselinePolicy,
    workloadRecordIds: scope.workload_record_ids,
    environments: scope.environments,
    waivedRuleIds: scope.waived_rule_ids,
    justification: scope.justification,
    effectiveUntil: source.effective_until ?? "",
  });
}

export async function transitionGatewayException(policyId: string, action: GatewayExceptionTransition, expectedVersion: number): Promise<AIGovernancePolicy> {
  return requestJSON<AIGovernancePolicy>(apiBase, `/api/v1/ai-governance/policies/${encodeURIComponent(policyId)}/${action}`, {
    method: "POST",
    body: JSON.stringify({ expected_version: expectedVersion }),
  });
}

export type GatewayExceptionScope = {
  target_baseline_id: string;
  target_baseline_version: number;
  workload_record_ids: string[];
  environments: string[];
  waived_rule_ids: string[];
  justification: string;
};

export function gatewayExceptionScope(policy: AIGovernancePolicy): GatewayExceptionScope | null {
  const value = policy.eligibility as Partial<GatewayExceptionScope>;
  if (!value || typeof value.target_baseline_id !== "string" || typeof value.target_baseline_version !== "number" || !Array.isArray(value.workload_record_ids) || !Array.isArray(value.environments) || !Array.isArray(value.waived_rule_ids) || typeof value.justification !== "string") {
    return null;
  }
  return {
    target_baseline_id: value.target_baseline_id,
    target_baseline_version: value.target_baseline_version,
    workload_record_ids: value.workload_record_ids.filter((item): item is string => typeof item === "string"),
    environments: value.environments.filter((item): item is string => typeof item === "string"),
    waived_rule_ids: value.waived_rule_ids.filter((item): item is string => typeof item === "string"),
    justification: value.justification,
  };
}

async function createGatewayExceptionRevision(input: {
  tenantId: string;
  code: string;
  name: string;
  rolloutMode: "SHADOW" | "ENFORCE";
  baseline: Pick<GatewayBaselinePolicy, "id" | "version">;
  workloadRecordIds: string[];
  environments: string[];
  waivedRuleIds: string[];
  justification: string;
  effectiveUntil: string;
}): Promise<AIGovernancePolicy> {
  const scope: GatewayExceptionScope = {
    target_baseline_id: input.baseline.id,
    target_baseline_version: input.baseline.version,
    workload_record_ids: [...new Set(input.workloadRecordIds)],
    environments: [...new Set(input.environments.map((value) => value.toUpperCase()))],
    waived_rule_ids: [...new Set(input.waivedRuleIds)],
    justification: input.justification.trim(),
  };
  return requestJSON<AIGovernancePolicy>(apiBase, "/api/v1/ai-governance/policies", {
    method: "POST",
    body: JSON.stringify({
      tenant_id: input.tenantId,
      code: input.code,
      name: input.name.trim(),
      action_class: gatewayExceptionActionClass,
      eligibility: scope,
      blast_radius_limit: { workload_record_ids: scope.workload_record_ids, environments: scope.environments },
      verification_contract: {
        activation: "MAKER_CHECKER",
        target_baseline_id: scope.target_baseline_id,
        target_baseline_version: scope.target_baseline_version,
        expires_automatically: true,
      },
      rollout_mode: input.rolloutMode,
      effective_until: input.effectiveUntil,
      definition: { default_action: "ALLOW" },
    }),
  });
}
