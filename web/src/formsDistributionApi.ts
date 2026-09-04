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
  issued_access_routes?: Array<{ route_id: string; selector: string; policy: DistributionAccessPolicy; expires_at: string }>;
};

export type DistributionAmendmentImpact = {
  deadline_changed?: boolean;
  route_expiry_changed?: boolean;
  reminder_policy_changed?: boolean;
  recipients_added?: number;
  recipients_revoked?: number;
};

export type AmendDistributionInput = {
  expected_version: number;
  deadline?: string;
  route_expires_at?: string;
  reminder_policy?: Record<string, unknown>;
  add_recipients?: CreateDistributionRecipient[];
  revoke_recipient_ids?: string[];
};

export type SupersessionFieldDecision = { field_id: string; reason?: string };
export type DistributionSupersessionPreview = {
  distribution_id: string;
  expected_version: number;
  expected_workspace_version: number;
  target_form_template_id: string;
  target_form_version: number;
  compatible_fields: SupersessionFieldDecision[];
  excluded_fields: SupersessionFieldDecision[];
};
export type SupersedeDistributionInput = {
  expected_version: number;
  expected_workspace_version: number;
  target_form_version: number;
  carry_forward: boolean;
  confirmed_field_ids: string[];
};
export type DistributionSupersessionResult = {
  previous: DistributionDetail;
  replacement: DistributionDetail;
  carried_field_ids: string[];
  issued_access_routes?: DistributionDetail["issued_access_routes"];
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
  score?: ResponseScore;
};
export type ResponseRevisionPage = { items: ResponseRevision[] };

export type ResponseScoreMode = "NONE" | "RISK" | "COMPLIANCE";
export type ResponseScoreDirection = "HIGH_IS_POOR" | "LOW_IS_POOR";
export type ResponseConcernBand = "LOW" | "MODERATE" | "HIGH" | "CRITICAL";
export type ResponseScoreState = "NOT_CONFIGURED" | "FINAL" | "PROVISIONAL" | "FAILED";
export type ResponseSort = "CONCERN_DESC" | "COMPLETED_DESC" | "RAW_ASC" | "RAW_DESC";

export type ResponseScoreContribution = {
  id: string;
  outcome: string;
  points: number;
  weight: number;
};

export type ResponseScoreRule = {
  id: string;
  matched: boolean;
  outcome: string;
  effect: string;
  value?: number;
  weight?: number;
};

export type ResponseScore = {
  mode?: ResponseScoreMode;
  direction?: ResponseScoreDirection;
  raw_score?: number;
  adverse_score?: number;
  band?: ResponseConcernBand;
  coverage?: number;
  final?: boolean;
  state: ResponseScoreState;
  profile_version?: string;
  profile_checksum?: string;
  evaluator_version?: string;
  failure_code?: string;
  calculated_at?: string;
  contribution_results?: ResponseScoreContribution[];
  rule_results?: ResponseScoreRule[];
};

export type CompletedResponseSummary = {
  id: string;
  distribution_id: string;
  form_template_id: string;
  form_template_version: number;
  title: string;
  subject_type: string;
  subject_id: string;
  revision: number;
  current: boolean;
  state: "PROVISIONAL" | "FINAL";
  score?: ResponseScore;
  completed_at: string;
};

export type CompletedResponsePage = { items: CompletedResponseSummary[]; next_cursor?: string };
export type CompletedResponseDetail = { response: CompletedResponseSummary; revision: ResponseRevision };
export type CompletedResponseQuery = {
  form_template_id?: string;
  form_template_version?: number;
  subject_type?: string;
  subject_id?: string;
  modes?: ResponseScoreMode[];
  bands?: ResponseConcernBand[];
  states?: ResponseScoreState[];
  raw_min?: number;
  raw_max?: number;
  adverse_min?: number;
  adverse_max?: number;
  completed_from?: string;
  completed_until?: string;
  current_only?: boolean;
  sort?: ResponseSort;
  cursor?: string;
  limit?: number;
};

function queryString(values: Record<string, string | number | undefined>) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== "") params.set(key, String(value));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

function normalizeDistribution(raw: Record<string, unknown>): Distribution {
  return {
    id: raw.id as string, legal_entity_id: raw.legal_entity_id as string | undefined,
    form_template_id: raw.form_template_id as string, form_template_version: raw.form_template_version as number,
    subject_type: raw.subject_type as string, subject_id: raw.subject_id as string, title: raw.title as string,
    purpose: raw.purpose as string, access_policy: raw.access_policy as DistributionAccessPolicy, status: raw.status as DistributionStatus,
    deadline: raw.deadline as string, route_expires_at: raw.route_expires_at as string,
    reminder_policy: raw.reminder_policy as Record<string, unknown> | undefined, created_by: raw.created_by as string | undefined,
    version: raw.version as number, created_at: raw.created_at as string, updated_at: raw.updated_at as string,
  };
}

function normalizeRecipient(raw: Record<string, unknown>): DistributionRecipient {
  return {
    id: raw.id as string, role: raw.role as DistributionRecipientRole, type: raw.type as DistributionRecipientType,
    principal_id: raw.principal_id as string | undefined, request_id: raw.request_id as string | undefined,
    audience_hint: raw.audience_hint as string | undefined, contact_label: raw.contact_label as string | undefined,
    state: raw.state as DistributionRecipientState, version: raw.version as number,
  };
}

function normalizeWorkspace(raw: Record<string, unknown>): DistributionWorkspace {
  return { id: raw.id as string, status: raw.status as DistributionWorkspace["status"], version: raw.version as number, updated_at: raw.updated_at as string };
}

function normalizeScore(raw: unknown): ResponseScore | undefined {
  if (!raw || typeof raw !== "object") return undefined;
  const value = raw as Record<string, unknown>;
  const optionalNumber = (key: string) => typeof value[key] === "number" ? value[key] as number : undefined;
  const optionalString = (key: string) => typeof value[key] === "string" && value[key] ? value[key] as string : undefined;
  return {
    mode: optionalString("mode") as ResponseScoreMode | undefined,
    direction: optionalString("direction") as ResponseScoreDirection | undefined,
    raw_score: optionalNumber("raw_score"), adverse_score: optionalNumber("adverse_score"),
    band: optionalString("band") as ResponseConcernBand | undefined,
    coverage: optionalNumber("coverage"), final: typeof value.final === "boolean" ? value.final : undefined,
    state: (optionalString("state") ?? "NOT_CONFIGURED") as ResponseScoreState,
    profile_version: optionalString("profile_version"), profile_checksum: optionalString("profile_checksum"),
    evaluator_version: optionalString("evaluator_version"), failure_code: optionalString("failure_code"),
    calculated_at: optionalString("calculated_at"),
    contribution_results: Array.isArray(value.contribution_results) ? value.contribution_results as ResponseScoreContribution[] : undefined,
    rule_results: Array.isArray(value.rule_results) ? value.rule_results as ResponseScoreRule[] : undefined,
  };
}

function normalizeResponseRevision(raw: Record<string, unknown>): ResponseRevision {
  return {
    id: raw.id as string, revision: raw.revision as number, supersedes_revision_id: raw.supersedes_revision_id as string | undefined,
    achieved_assurance: raw.achieved_assurance as ResponseRevision["achieved_assurance"], signoff_summary: raw.signoff_summary as Record<string, unknown> | undefined,
    compliance_score: raw.compliance_score as number | undefined, scored_weight_coverage: (raw.scored_weight_coverage as number | undefined) ?? 0,
    state: raw.state as ResponseRevision["state"], critical_field_results: raw.critical_field_results as Array<Record<string, unknown>> | undefined,
    scoring_policy_version: raw.scoring_policy_version as string | undefined, current: raw.current as boolean, created_at: raw.created_at as string,
    score: normalizeScore(raw.score),
  };
}

function normalizeCompletedResponse(raw: Record<string, unknown>): CompletedResponseSummary {
  return {
    id: raw.id as string, distribution_id: raw.distribution_id as string,
    form_template_id: raw.form_template_id as string, form_template_version: raw.form_template_version as number,
    title: raw.title as string, subject_type: raw.subject_type as string, subject_id: raw.subject_id as string,
    revision: raw.revision as number, current: raw.current as boolean, state: raw.state as CompletedResponseSummary["state"],
    score: normalizeScore(raw.score), completed_at: raw.completed_at as string,
  };
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

export async function amendDistribution(id: string, input: AmendDistributionInput): Promise<{ detail: DistributionDetail; impact: DistributionAmendmentImpact; issued_access_routes?: DistributionDetail["issued_access_routes"] }> {
  const raw = await requestJSON<{ distribution: Parameters<typeof normalizeDetail>[0]; impact: DistributionAmendmentImpact; issued_access_routes?: DistributionDetail["issued_access_routes"] }>(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}/amend`, { method: "POST", body: JSON.stringify(input) });
  return { detail: normalizeDetail(raw.distribution), impact: raw.impact, issued_access_routes: raw.issued_access_routes };
}

export async function previewDistributionSupersession(id: string, expectedVersion: number, targetFormVersion: number): Promise<DistributionSupersessionPreview> {
  const raw = await requestJSON<{ preview: DistributionSupersessionPreview }>(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}/supersede`, { method: "POST", body: JSON.stringify({ expected_version: expectedVersion, target_form_version: targetFormVersion, confirm: false }) });
  return raw.preview;
}

export async function supersedeDistribution(id: string, input: SupersedeDistributionInput): Promise<DistributionSupersessionResult> {
  const raw = await requestJSON<{ previous: Parameters<typeof normalizeDetail>[0]; replacement: Parameters<typeof normalizeDetail>[0]; carried_field_ids?: string[]; issued_access_routes?: DistributionDetail["issued_access_routes"] }>(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}/supersede`, { method: "POST", body: JSON.stringify({ ...input, confirm: true }) });
  return { previous: normalizeDetail(raw.previous), replacement: normalizeDetail(raw.replacement), carried_field_ids: raw.carried_field_ids ?? [], issued_access_routes: raw.issued_access_routes };
}

export function loadRecipientCandidates(search: string, limit = 12): Promise<RecipientCandidatePage> {
  return requestJSON(apiBase, `/api/v1/forms/recipient-candidates${queryString({ search: search.trim() || undefined, limit })}`);
}

export function transitionDistribution(id: string, version: number, action: "lock" | "reopen" | "revoke"): Promise<DistributionDetail> {
  return requestJSON<Parameters<typeof normalizeDetail>[0]>(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}/${action}`, { method: "POST", body: JSON.stringify({ expected_version: version }) }).then(normalizeDetail);
}

export function loadResponseRevisions(id: string, limit = 100): Promise<ResponseRevisionPage> {
  return requestJSON<{ items: Record<string, unknown>[] }>(apiBase, `/api/v1/forms/distributions/${encodeURIComponent(id)}/responses${queryString({ limit })}`).then((raw) => ({ items: (raw.items ?? []).map(normalizeResponseRevision) }));
}

export async function loadCompletedResponses(query: CompletedResponseQuery = {}): Promise<CompletedResponsePage> {
  const params = new URLSearchParams();
  const scalar: Record<string, string | number | boolean | undefined> = {
    form_template_id: query.form_template_id, form_template_version: query.form_template_version,
    subject_type: query.subject_type, subject_id: query.subject_id,
    raw_min: query.raw_min, raw_max: query.raw_max, adverse_min: query.adverse_min, adverse_max: query.adverse_max,
    completed_from: query.completed_from, completed_until: query.completed_until,
    current_only: query.current_only, sort: query.sort, cursor: query.cursor, limit: query.limit,
  };
  for (const [key, value] of Object.entries(scalar)) if (value !== undefined && value !== "") params.set(key, String(value));
  for (const mode of query.modes ?? []) params.append("mode", mode);
  for (const band of query.bands ?? []) params.append("band", band);
  for (const state of query.states ?? []) params.append("score_state", state);
  const encoded = params.toString();
  const raw = await requestJSON<{ items: Record<string, unknown>[]; next_cursor?: string }>(apiBase, `/api/v1/forms/responses${encoded ? `?${encoded}` : ""}`);
  return { items: (raw.items ?? []).map(normalizeCompletedResponse), next_cursor: raw.next_cursor };
}

export async function loadCompletedResponse(id: string): Promise<CompletedResponseDetail> {
  const raw = await requestJSON<{ response: Record<string, unknown>; revision: Record<string, unknown> }>(apiBase, `/api/v1/forms/responses/${encodeURIComponent(id)}`);
  return { response: normalizeCompletedResponse(raw.response), revision: normalizeResponseRevision(raw.revision) };
}
