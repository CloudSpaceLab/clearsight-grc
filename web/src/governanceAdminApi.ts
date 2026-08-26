import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type GovernancePolicyRecord = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  code: string;
  name: string;
  status: string;
  current_version: number;
  maker_id: string;
  checker_id?: string;
  effective_from?: string;
  effective_until?: string;
  version: number;
};

export type GovernanceDelegationRecord = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  from_principal_id: string;
  to_principal_id: string;
  responsibility: string;
  scope: Record<string, unknown>;
  starts_at: string;
  ends_at: string;
  status: string;
  reason: string;
  maker_id: string;
  approver_id?: string;
  version: number;
};

export type GovernanceInventory = {
  policies: GovernancePolicyRecord[];
  delegations: GovernanceDelegationRecord[];
  policiesAvailable: boolean;
  delegationsAvailable: boolean;
};

export type GovernanceDelegationCandidate = {
  principal_id: string;
  display_name: string;
  context_label?: string;
  can_give: boolean;
  can_receive: boolean;
};

export type GovernanceDelegationCandidatePage = { items: GovernanceDelegationCandidate[]; has_more: boolean };

export type CreateGovernancePolicyDraftRequest = {
  legalEntityId: string;
  code: string;
  name: string;
  responsibility: string;
  roleCode: string;
  objectType?: "*" | "PROGRAM" | "MATTER";
  decisionType?: string;
  minMateriality: number;
  priority: number;
  effectiveFrom?: string;
};

export type CreateGovernanceDelegationRequest = {
  legalEntityId: string;
  fromPrincipalId: string;
  toPrincipalId: string;
  responsibility: string;
  startsAt: string;
  endsAt: string;
  reason: string;
};

function request<T>(path: string, init?: RequestInit): Promise<T> {
  return requestJSON<T>(apiBase, path, init);
}

export async function loadGovernanceInventory(): Promise<GovernanceInventory> {
  const [policies, delegations] = await Promise.all([
    settle(request<{ items?: GovernancePolicyRecord[] }>("/api/v1/governance/policies?limit=51")),
    settle(request<{ items?: GovernanceDelegationRecord[] }>("/api/v1/governance/delegations?limit=51")),
  ] as const);
  return {
    policies: policies.value?.items ?? [],
    delegations: delegations.value?.items ?? [],
    policiesAvailable: policies.available,
    delegationsAvailable: delegations.available,
  };
}

function settle<T>(operation: Promise<T>): Promise<{ value?: T; available: boolean }> {
  return operation.then((value) => ({ value, available: true })).catch(() => ({ available: false }));
}

export function createGovernanceDelegation(input: CreateGovernanceDelegationRequest): Promise<GovernanceDelegationRecord> {
  return request("/api/v1/governance/delegations", {
    method: "POST",
    body: JSON.stringify({
      legal_entity_id: input.legalEntityId,
      from_principal_id: input.fromPrincipalId,
      to_principal_id: input.toPrincipalId,
      responsibility: input.responsibility,
      scope: { legal_entity_id: input.legalEntityId },
      starts_at: input.startsAt,
      ends_at: input.endsAt,
      reason: input.reason,
    }),
  });
}

export function searchGovernanceDelegationCandidates(responsibility: string, query = ""): Promise<GovernanceDelegationCandidatePage> {
  const params = new URLSearchParams({ responsibility, limit: "50" });
  if (query.trim()) params.set("q", query.trim());
  return request(`/api/v1/governance/delegation-candidates?${params.toString()}`);
}

export function createGovernancePolicyDraft(input: CreateGovernancePolicyDraftRequest): Promise<GovernancePolicyRecord> {
  const ruleBase = slug(input.name) || "policy";
  const rule: Record<string, unknown> = {
    id: ruleBase.endsWith("-route") ? ruleBase : `${ruleBase}-route`,
    legal_entity_id: input.legalEntityId,
    responsibility: input.responsibility,
    min_materiality: input.minMateriality,
    priority: input.priority,
    selector: { kind: "ROLE", ref: input.roleCode },
  };
  if (input.objectType && input.objectType !== "*") rule.object_type = input.objectType;
  if (input.decisionType?.trim()) rule.decision_type = input.decisionType.trim();
  return request("/api/v1/governance/policies", {
    method: "POST",
    body: JSON.stringify({
      legal_entity_id: input.legalEntityId,
      code: input.code.trim(),
      name: input.name.trim(),
      ...(input.effectiveFrom ? { effective_from: input.effectiveFrom } : {}),
      definition: { rules: [rule] },
    }),
  });
}

export function transitionGovernancePolicy(id: string, action: "submit" | "approve" | "reject" | "retire", expectedVersion: number, rationale?: string): Promise<GovernancePolicyRecord> {
  return request(`/api/v1/governance/policies/${encodeURIComponent(id)}/${action}`, {
    method: "POST",
    body: JSON.stringify(transitionBody(expectedVersion, rationale)),
  });
}

function slug(value: string) {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 80);
}

export function transitionGovernanceDelegation(id: string, action: "submit" | "approve" | "revoke", expectedVersion: number, rationale?: string): Promise<GovernanceDelegationRecord> {
  return request(`/api/v1/governance/delegations/${encodeURIComponent(id)}/${action}`, {
    method: "POST",
    body: JSON.stringify(transitionBody(expectedVersion, rationale)),
  });
}

function transitionBody(expectedVersion: number, rationale?: string) {
  return rationale ? { expected_version: expectedVersion, rationale } : { expected_version: expectedVersion };
}
