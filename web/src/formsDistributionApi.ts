import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type DistributionAccessPolicy = "DIRECT_MAGIC_LINK" | "SHARED_LINK_EMAIL_OTP" | "DIRECT_LINK_EMAIL_OTP";
export type DistributionStatus = "DRAFT" | "READY" | "OPEN" | "LOCKED" | "COMPLETED" | "EXPIRED" | "REVOKED" | "SUPERSEDED";
export type DistributionDueState = "OPEN" | "OVERDUE" | "CLOSED";
export type DistributionRecipientState = "PENDING" | "DELIVERED" | "VERIFIED" | "REVOKED" | "COMPLETED";
export type DistributionRecipientRole = "TO" | "CC";
export type DistributionRecipientType = "INTERNAL_PRINCIPAL" | "EXTERNAL_AUDIENCE";

export type Distribution = {
  id: string;
  legal_entity_id?: string;
  form_template_id: string;
  form_template_version: number;
  subject_type: string;
  subject_id: string;
  title: string;
  purpose: string;
  access_policy: DistributionAccessPolicy;
  status: DistributionStatus;
  deadline: string;
  route_expires_at: string;
  reminder_policy?: Record<string, unknown>;
  created_by?: string;
  version: number;
  created_at: string;
  updated_at: string;
};

export type DistributionRecipient = {
  id: string;
  role: DistributionRecipientRole;
  type: DistributionRecipientType;
  principal_id?: string;
  request_id?: string;
  audience_hint?: string;
  contact_label?: string;
  state: DistributionRecipientState;
  version: number;
};

export type DistributionWorkspace = {
  id: string;
  status: "OPEN" | "LOCKED" | "COMPLETED" | "REVOKED";
  version: number;
  updated_at: string;
};

export type DistributionDetail = {
  distribution: Distribution;
  recipients: DistributionRecipient[];
  workspace: DistributionWorkspace;
  issued_access_routes?: Array<{ route_id?: string; route_selector?: string; recipient_id?: string; expires_at?: string }>;
};

export type DistributionPage = { items: Distribution[]; next_cursor?: string };
export type DistributionQuery = {
  status?: DistributionStatus;
  due_state?: DistributionDueState;
  subject_type?: string;
  subject_id?: string;
  owner?: string;
  cursor?: string;
  limit?: number;
};

export type RecipientCandidate = { principal_id: string; display_name: string; context_label?: string };
export type RecipientCandidatePage = { items: RecipientCandidate[]; has_more: boolean };

export type CreateDistributionRecipient = {
  role: DistributionRecipientRole;
  type: DistributionRecipientType;
  principal_id?: string;
  address?: string;
  audience_hint?: string;
  contact_label?: string;
};

export type CreateDistributionInput = {
  form_template_id: string;
  form_template_version: number;
  subject_type: string;
  subject_id: string;
  title: string;
  purpose: string;
  access_policy: DistributionAccessPolicy;
  estimated_minutes: number;
  deadline: string;
  route_expires_at: string;
  reminder_policy?: Record<string, unknown>;
  recipients: CreateDistributionRecipient[];
};

export type ResponseRevision = {
  id: string;
  revision: number;
  supersedes_revision_id?: string;
  achieved_assurance: "LINK_POSSESSION" | "EMAIL_VERIFIED";
  signoff_summary?: Record<string, unknown>;
  compliance_score?: number;
  scored_weight_coverage: number;
  state: "PROVISIONAL" | "FINAL";
  critical_field_results?: Array<Record<string, unknown>>;
  scoring_policy_version?: string;
  current: boolean;
  created_at: string;
};
export type ResponseRevisionPage = { items: ResponseRevision[] };

function queryString(values: Record<string, string | number | undefined>) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== "") params.set(key, String(value));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

function normalizeDistribution(raw: Record<string, unknown>): Distribution {
  const pick = <T>(snake: string, pascal: string) => (raw[snake] ?? raw[pascal]) as T;
  return {
    id: pick<string>("id", "ID"),
    legal_entity_id: pick<string | undefined>("legal_entity_id", "LegalEntityID"),
    form_template_id: pick<string>("form_template_id", "FormTemplateID"),
    form_template_version: pick<number>("form_template_version", "FormTemplateVersion"),
    subject_type: pick<string>("subject_type", "SubjectType"),
    subject_id: pick<string>("subject_id", "SubjectID"),
    title: pick<string>("title", "Title"),
    purpose: pick<string>("purpose", "Purpose"),
    access_policy: pick<DistributionAccessPolicy>("access_policy", "AccessPolicy"),
    status: pick<DistributionStatus>("status", "Status"),
    deadline: pick<string>("deadline", "Deadline"),
    route_expires_at: pick<string>("route_expires_at", "RouteExpiresAt"),
    reminder_policy: pick<Record<string, unknown> | undefined>("reminder_policy", "ReminderPolicy"),
    created_by: pick<string | undefined>("created_by", "CreatedBy"),
    version: pick<number>("version", "Version"),
    created_at: pick<string>("created_at", "CreatedAt"),
    updated_at: pick<string>("updated_at", "UpdatedAt"),
  };
}

function normalizeRecipient(raw: Record<string, unknown>): DistributionRecipient {
  const pick = <T>(snake: string, pascal: string) => (raw[snake] ?? raw[pascal]) as T;
  return {
    id: pick<string>("id", "ID"), role: pick<DistributionRecipientRole>("role", "Role"), type: pick<DistributionRecipientType>("type", "Type"),
    principal_id: pick<string | undefined>("principal_id", "PrincipalID"), request_id: pick<string | undefined>("request_id", "RequestID"),
    audience_hint: pick<string | undefined>("audience_hint", "AudienceHint"), contact_label: pick<string | undefined>("contact_label", "ContactLabel"),
    state: pick<DistributionRecipientState>("state", "State"), version: pick<number>("version", "Version"),
  };
}

function normalizeWorkspace(raw: Record<string, unknown>): DistributionWorkspace {
  const pick = <T>(snake: string, pascal: string) => (raw[snake] ?? raw[pascal]) as T;
  return { id: pick<string>("id", "ID"), status: pick<DistributionWorkspace["status"]>("status", "Status"), version: pick<number>("version", "Version"), updated_at: pick<string>("updated_at", "UpdatedAt") };
}

function normalizeDetail(raw: { distribution: Record<string, unknown>; recipients: Record<string, unknown>[]; workspace: Record<string, unknown>; issued_access_routes?: DistributionDetail["issued_access_routes"] }): DistributionDetail {
  return { distribution: normalizeDistribution(raw.distribution), recipients: (raw.recipients ?? []).map(normalizeRecipient), workspace: normalizeWorkspace(raw.workspace), issued_access_routes: raw.issued_access_routes };
}

export async function loadDistributionPage(query: DistributionQuery = {}): Promise<DistributionPage> {
  const raw = await requestJSON<{ items: Record<string, unknown>[]; next_cursor?: string }>(apiBase, `/api/v1/forms/distributions${queryString(query)}`);
  return { items: (raw.items ?? []).map(normalizeDistribution), next_cursor: raw.next_cursor };
}

export async function loadDistribution(id: string): Promise<DistributionDetail> {
  return normalizeDetail(await requestJSON(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}`));
}

export async function createDistribution(input: CreateDistributionInput): Promise<DistributionDetail> {
  return normalizeDetail(await requestJSON(apiBase, "/api/v1/forms/distributions", { method: "POST", body: JSON.stringify(input) }));
}

export function loadRecipientCandidates(search: string, limit = 12): Promise<RecipientCandidatePage> {
  return requestJSON(apiBase, `/api/v1/forms/recipient-candidates${queryString({ search: search.trim() || undefined, limit })}`);
}

export function transitionDistribution(id: string, version: number, action: "lock" | "reopen" | "revoke"): Promise<DistributionDetail> {
  return requestJSON<Parameters<typeof normalizeDetail>[0]>(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}/${action}`, { method: "POST", body: JSON.stringify({ expected_version: version }) }).then(normalizeDetail);
}

export function loadResponseRevisions(id: string, limit = 100): Promise<ResponseRevisionPage> {
  return requestJSON(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}/responses${queryString({ limit })}`);
}
