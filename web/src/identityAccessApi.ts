import { requestJSON } from "./http";

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
  Steps: Array<{ After: number; Responsibility: string; DepartmentLevelsUp?: number }>;
};
export type EscalationPolicy = { policy_id: string; code: string; name: string; version: number; sequences: EscalationSequence[] };
export type IdentityAccessOverview = {
  sign_in: { mode: string; issuer?: string; authentication?: string; assurance_level?: string };
  can_configure: boolean;
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
  steps: Array<{ index: number; after: string; responsibility: string; scope: string; department_path?: string[] }>;
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

export async function revokeIdentitySource(id: string): Promise<void> {
  const response = await fetch(`${apiBase}/api/v1/access/scim-sources/${encodeURIComponent(id)}/revoke`, { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}" });
  if (!response.ok) await requestJSON<never>(apiBase, `/api/v1/access/scim-sources/${encodeURIComponent(id)}/revoke`, { method: "POST", body: "{}" });
}

export function createGroupRoleBinding(input: { group_id: string; role_template_id: string; department_path: string[] }): Promise<GroupRoleBinding> {
  return request("/api/v1/access/group-role-bindings", { method: "POST", body: JSON.stringify(input) });
}

export async function retireGroupRoleBinding(id: string): Promise<void> {
  const response = await fetch(`${apiBase}/api/v1/access/group-role-bindings/${encodeURIComponent(id)}/retire`, { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}" });
  if (!response.ok) await requestJSON<never>(apiBase, `/api/v1/access/group-role-bindings/${encodeURIComponent(id)}/retire`, { method: "POST", body: "{}" });
}

export function previewEscalation(input: { policy_id: string; sequence_id: string; department_path: string[] }): Promise<EscalationPreview> {
  return request("/api/v1/access/escalations/preview", { method: "POST", body: JSON.stringify(input) });
}
