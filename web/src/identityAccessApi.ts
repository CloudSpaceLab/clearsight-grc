import { requestJSON, requestVoid as requestNoContent } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type IdentitySource = {
  id: string;
  code: string;
  status: "ACTIVE" | "REVOKED" | string;
  identity_issuer?: string;
  subject_attribute: "externalId" | "userName" | string;
  active_users: number;
  active_groups: number;
  last_activity_at?: string;
};

export type IdentityPerson = {
  id: string;
  display_name: string;
  status: string;
  user_name?: string;
  source_code?: string;
  source_state?: string;
};

export type IdentityGroup = {
  id: string;
  display_name: string;
  external_id?: string;
  source_code: string;
  source_state: string;
  member_count: number;
};

export type IdentityRole = { id: string; code: string; name: string; capabilities: string[] };
export type IdentityLegalEntity = { id: string; code: string; name: string };
export type GroupRoleBinding = {
  id: string;
  group_id: string;
  group_name: string;
  role_template_id: string;
  role_code: string;
  legal_entity_id: string;
  legal_entity: string;
  department_path: string[];
  valid_from: string;
  valid_until?: string;
};
export type EscalationSequence = {
  ID: string;
  Trigger: string;
  Steps: Array<{
    After: number;
    Responsibility: string;
    DepartmentLevelsUp?: number;
    SourceRoles?: string[];
    TargetRoles?: string[];
    TargetGroupIDs?: string[];
  }>;
};
export type EscalationGuardRevision = {
  policy_id: string;
  tenant_id?: string;
  version: number;
  base_version: number;
  maker_id: string;
  created_at: string;
  sequences?: EscalationSequence[];
};
export type EscalationPolicy = {
  policy_id: string;
  code: string;
  name: string;
  version: number;
  record_version: number;
  sequences: EscalationSequence[];
  pending_revision?: EscalationGuardRevision;
};
export type IdentityAccessOverview = {
  sign_in: { mode: string; issuer?: string; authentication?: string; assurance_level?: string };
  actor_principal_id: string;
  can_configure: boolean;
  can_configure_escalation: boolean;
  sources: IdentitySource[];
  people: IdentityPerson[];
  groups: IdentityGroup[];
  roles: IdentityRole[];
  legal_entities: IdentityLegalEntity[];
  bindings: GroupRoleBinding[];
  escalation: { pending_timers: number; escalated_tasks: number; unresolved_24h: number; failed_timers: number };
  escalation_policies: EscalationPolicy[];
};
export type EscalationPreview = {
  policy_id: string;
  policy_code: string;
  policy_version: number;
  sequence_id: string;
  trigger: string;
  steps: Array<{
    index: number;
    after: string;
    responsibility: string;
    scope: string;
    department_path?: string[];
    source_roles?: string[];
    target_roles?: string[];
    target_group_ids?: string[];
  }>;
};

function request<T>(path: string, init?: RequestInit): Promise<T> {
  return requestJSON<T>(apiBase, path, init);
}

export function loadIdentityAccessOverview(): Promise<IdentityAccessOverview> {
  return request<IdentityAccessOverview>("/api/v1/access/overview?limit=50");
}

export function createIdentitySource(input: { code: string; identity_issuer?: string; subject_attribute: "externalId" | "userName" }): Promise<{ source: IdentitySource; token: string }> {
  return request("/api/v1/access/scim-sources", { method: "POST", body: JSON.stringify(input) });
}

export function rotateIdentitySourceToken(id: string): Promise<{ token: string }> {
  return request(`/api/v1/access/scim-sources/${encodeURIComponent(id)}/rotate-token`, { method: "POST", body: "{}" });
}

export function revokeIdentitySource(id: string): Promise<void> {
  return requestNoContent(apiBase, `/api/v1/access/scim-sources/${encodeURIComponent(id)}/revoke`, { method: "POST", body: "{}" });
}

export function createGroupRoleBinding(input: { group_id: string; role_template_id: string; department_path: string[] }): Promise<GroupRoleBinding> {
  return request("/api/v1/access/group-role-bindings", { method: "POST", body: JSON.stringify(input) });
}

export function retireGroupRoleBinding(id: string): Promise<void> {
  return requestNoContent(apiBase, `/api/v1/access/group-role-bindings/${encodeURIComponent(id)}/retire`, { method: "POST", body: "{}" });
}

export function proposeEscalationGuardRevision(input: {
  policy_id: string;
  sequence_id: string;
  step_index: number;
  source_roles: string[];
  target_roles: string[];
  target_group_ids: string[];
  expected_policy_version: number;
}): Promise<EscalationGuardRevision> {
  return request("/api/v1/access/escalation-guard-revisions", { method: "POST", body: JSON.stringify(input) });
}

export function approveEscalationGuardRevision(policyID: string, revisionVersion: number, input: { expected_policy_version: number; rationale: string }): Promise<void> {
  return requestNoContent(apiBase, `/api/v1/access/escalation-guard-revisions/${encodeURIComponent(policyID)}/${revisionVersion}/approve`, { method: "POST", body: JSON.stringify(input) });
}

export function previewEscalation(input: { policy_id: string; sequence_id: string; department_path: string[]; revision_version?: number }): Promise<EscalationPreview> {
  return request("/api/v1/access/escalations/preview", { method: "POST", body: JSON.stringify(input) });
}
