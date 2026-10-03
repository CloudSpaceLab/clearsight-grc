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
export type OrganizationScope = {
  id: string;
  legal_entity_id: string;
  parent_scope_id?: string;
  code: string;
  name: string;
  kind: "ORGANIZATION_UNIT" | "BRANCH" | "DEPARTMENT" | "FUNCTION" | "BUSINESS_UNIT" | "CRITICAL_SERVICE";
  department_path: string[];
  origin: "LEGACY_DEPARTMENT_PATH" | "MANAGED";
  status: string;
  valid_from: string;
  valid_until?: string;
  version: number;
};

export type OrganizationScopeOperation = "CREATE" | "UPDATE" | "MOVE" | "RETIRE";
export type OrganizationScopeImpact = {
  child_scopes: number;
  positions: number;
  access_mappings: number;
  open_matters: number;
};
export type OrganizationScopeRevision = {
  id: string;
  scope_id: string;
  operation: OrganizationScopeOperation;
  base_version: number;
  proposed_parent_scope_id?: string;
  proposed_code?: string;
  proposed_name?: string;
  proposed_kind?: OrganizationScope["kind"];
  maker_id: string;
  checker_id?: string;
  status: string;
  rationale?: string;
  impact: OrganizationScopeImpact;
  created_at: string;
  decided_at?: string;
  applied_at?: string;
};
export type ProposeOrganizationScopeInput = {
  scope_id?: string;
  operation: OrganizationScopeOperation;
  parent_scope_id?: string;
  code?: string;
  name?: string;
  kind?: OrganizationScope["kind"];
  expected_version?: number;
};

export type OrganizationPositionOperation = "CREATE" | "UPDATE" | "RETIRE";
export type OrganizationPositionState = {
  code: string;
  title: string;
  function_name?: string;
  organization_scope_id?: string;
  parent_position_id?: string;
  occupant_principal_id?: string;
};
export type OrganizationPositionImpact = {
  child_positions: number;
  responsibility_assignments: number;
  authority_grants: number;
  active_role_bindings: number;
};
export type OrganizationPositionRevision = {
  id: string;
  position_id: string;
  operation: OrganizationPositionOperation;
  base_version: number;
  base: OrganizationPositionState;
  proposed: OrganizationPositionState;
  maker_id: string;
  checker_id?: string;
  status: string;
  rationale?: string;
  impact: OrganizationPositionImpact;
  created_at: string;
  decided_at?: string;
  applied_at?: string;
};
export type ProposeOrganizationPositionInput = {
  position_id?: string;
  operation: OrganizationPositionOperation;
  code?: string;
  title?: string;
  function_name?: string;
  organization_scope_id?: string;
  parent_position_id?: string;
  occupant_principal_id?: string;
  expected_version?: number;
};

export type OrganizationPosition = {
  id: string;
  code: string;
  title: string;
  function_name?: string;
  department_path: string[];
  organization_scope_id?: string;
  parent_position_id?: string;
  parent_position_code?: string;
  parent_position_title?: string;
  occupant_principal_id?: string;
  occupant_name?: string;
  occupant_status?: string;
  role_codes: string[];
  valid_from: string;
  valid_until?: string;
  version: number;
};
export type GroupRoleBinding = {
  id: string;
  group_id: string;
  group_name: string;
  role_template_id: string;
  role_code: string;
  legal_entity_id: string;
  legal_entity: string;
  department_path: string[];
  organization_scope_id?: string;
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
  can_configure_organization: boolean;
  can_configure_escalation: boolean;
  sources: IdentitySource[];
  people: IdentityPerson[];
  groups: IdentityGroup[];
  roles: IdentityRole[];
  legal_entities: IdentityLegalEntity[];
  bindings: GroupRoleBinding[];
  positions: OrganizationPosition[];
  organization_scopes: OrganizationScope[];
  organization_scopes_truncated?: boolean;
  organization_scope_revisions: OrganizationScopeRevision[];
  organization_position_revisions: OrganizationPositionRevision[];
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

export async function loadIdentityAccessOverview(): Promise<IdentityAccessOverview> {
  const overview = await request<IdentityAccessOverview>("/api/v1/access/overview?limit=50");
  return {
    ...overview,
    sources: overview.sources ?? [],
    people: overview.people ?? [],
    groups: overview.groups ?? [],
    roles: overview.roles ?? [],
    legal_entities: overview.legal_entities ?? [],
    bindings: overview.bindings ?? [],
    positions: overview.positions ?? [],
    organization_scopes: overview.organization_scopes ?? [],
    organization_scope_revisions: overview.organization_scope_revisions ?? [],
    organization_position_revisions: overview.organization_position_revisions ?? [],
    escalation_policies: overview.escalation_policies ?? [],
  };
}


export function proposeOrganizationScope(input: ProposeOrganizationScopeInput): Promise<OrganizationScopeRevision> {
  return request("/api/v1/access/organization-scope-revisions", { method: "POST", body: JSON.stringify(input) });
}

export function approveOrganizationScope(id: string, rationale: string): Promise<void> {
  return requestNoContent(apiBase, `/api/v1/access/organization-scope-revisions/${encodeURIComponent(id)}/approve`, { method: "POST", body: JSON.stringify({ rationale }) });
}

export function rejectOrganizationScope(id: string, rationale: string): Promise<void> {
  return requestNoContent(apiBase, `/api/v1/access/organization-scope-revisions/${encodeURIComponent(id)}/reject`, { method: "POST", body: JSON.stringify({ rationale }) });
}

export function proposeOrganizationPosition(input: ProposeOrganizationPositionInput): Promise<OrganizationPositionRevision> {
  return request("/api/v1/access/organization-position-revisions", { method: "POST", body: JSON.stringify(input) });
}

export function approveOrganizationPosition(id: string, rationale: string): Promise<void> {
  return requestNoContent(apiBase, `/api/v1/access/organization-position-revisions/${encodeURIComponent(id)}/approve`, { method: "POST", body: JSON.stringify({ rationale }) });
}

export function rejectOrganizationPosition(id: string, rationale: string): Promise<void> {
  return requestNoContent(apiBase, `/api/v1/access/organization-position-revisions/${encodeURIComponent(id)}/reject`, { method: "POST", body: JSON.stringify({ rationale }) });
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
