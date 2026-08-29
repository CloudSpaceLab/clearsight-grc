const fieldRequestID = "field-visit-atm-042";
const distributionID = "field-distribution-1";
const workspaceID = "field-workspace-1";
const sessionToken = "static-field-agent-session";
let responseDraft = { answers: {} as Record<string, unknown>, presentation_mode: "AUTOMATIC", version: 0 };
let fieldSequences: Record<string, number> = {};

export function resetStaticExternalCaptureDraft() {
  responseDraft = { answers: {}, presentation_mode: "AUTOMATIC", version: 0 };
  fieldSequences = {};
}

const fieldAgentRequest = {
  id: fieldRequestID,
  tenant_id: "bank-demo",
  legal_entity_id: "entity-demo",
  subject_type: "ASSET",
  subject_id: "ATM-LAG-042",
  title: "Verify ATM location after your visit",
  purpose: "Confirm that this ATM is present at the recorded address and provide one clear site photo.",
  why_you: "You were assigned to verify this location after a physical visit.",
  sensitivity: "INTERNAL",
  audience_type: "EXTERNAL",
  estimated_minutes: 3,
  deadline: new Date(Date.now() + 48 * 60 * 60 * 1000).toISOString(),
  known_facts: {
    atm_id: "ATM-LAG-042",
    location: "Meridian Trust Bank, Lekki Phase 1",
    expected_address: "12 Admiralty Way, Lekki Phase 1, Lagos",
    visit_type: "Physical address confirmation",
  },
  fields: [
    { id: "address_matches", label: "Is the ATM at the address above?", type: "single_select", required: true, options: ["Yes", "No"] },
    { id: "atm_identifiable", label: "Is the ATM present and clearly identifiable?", type: "single_select", required: true, options: ["Yes", "No"] },
    { id: "site_photo", label: "Site photo", type: "photo", required: true, description: "Take one clear photo showing the ATM and enough of the surrounding location to identify the site.", accepted_formats: ["image/jpeg", "image/png"] },
    { id: "visit_note", label: "Anything the reviewer should know?", type: "long_text", required: false, description: "Add a note only if something needs explanation." },
    { id: "agent_signature", label: "Signature", type: "signature", required: true, description: "I confirm that I visited this location and that the information and photo above are accurate to the best of my knowledge.", accepted_formats: ["image/png"] },
  ],
  status: "READY",
  version: 1,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

export async function staticExternalCaptureRequest(path: string, init?: RequestInit): Promise<unknown | undefined> {
  const url = new URL(path, "https://clearsight.demo");
  const method = (init?.method ?? "GET").toUpperCase();

  if (url.pathname === "/api/v1/evidence/access/start" && method === "POST") {
    const input = parseBody(init) as { route_selector?: string };
    if (input.route_selector !== "field-agent-demo") throw staticFailure(404, "access_unavailable", "Invitation unavailable");
    return { policy: "DIRECT_MAGIC_LINK", expires_at: fieldAgentRequest.deadline };
  }
  if (url.pathname === "/api/v1/evidence/access/redeem" && method === "POST") {
    const input = parseBody(init) as { route_selector?: string };
    if (input.route_selector !== "field-agent-demo") throw staticFailure(401, "access_unavailable", "Invitation unavailable");
    return redeemedSession();
  }

  // Legacy invitation support remains only for deterministic regression fixtures.
  if (url.pathname === "/api/v1/evidence/invitations/redeem" && method === "POST") {
    const input = parseBody(init) as { token?: string; audience?: string };
    if (input.token !== "field-agent-demo" || input.audience?.trim().toLowerCase() !== "field.agent@example.com") throw staticFailure(401, "invitation_unavailable", "Invitation unavailable");
    return { session_id: "field-session-1", session_token: sessionToken, request_id: fieldRequestID, audience_hint: "f***@example.com", expires_at: fieldAgentRequest.deadline };
  }

  const authorization = new Headers(init?.headers).get("Authorization");
  if (authorization !== `Bearer ${sessionToken}`) return undefined;

  if (url.pathname === "/api/v1/evidence/session/workspace" && method === "GET") return workspacePayload();
  if (url.pathname === "/api/v1/evidence/session/workspace" && method === "PATCH") {
    const input = parseBody(init) as {
      expected_version?: number;
      presentation_mode?: string;
      edits?: Array<{ field_id?: string; value?: unknown; base_sequence?: number }>;
    };
    if (input.expected_version !== responseDraft.version) throw staticFailure(409, "workspace_conflict", "The response workspace changed");
    for (const edit of input.edits ?? []) {
      if (!edit.field_id) continue;
      const currentSequence = fieldSequences[edit.field_id] ?? 0;
      if ((edit.base_sequence ?? 0) !== currentSequence) throw staticFailure(409, "workspace_conflict", "The response workspace changed");
      if (isAnswered(edit.value)) responseDraft.answers[edit.field_id] = structuredClone(edit.value);
      else delete responseDraft.answers[edit.field_id];
      fieldSequences[edit.field_id] = currentSequence + 1;
    }
    responseDraft = {
      answers: responseDraft.answers,
      presentation_mode: input.presentation_mode ?? responseDraft.presentation_mode,
      version: responseDraft.version + 1,
    };
    return responseWorkspace();
  }
  if (url.pathname === "/api/v1/evidence/session/workspace/submissions" && method === "POST") {
    const input = parseBody(init) as { expected_version?: number };
    if (input.expected_version !== responseDraft.version) throw staticFailure(409, "workspace_conflict", "The response workspace changed");
    return {
      workspace: { ...workspaceState(), status: "COMPLETED", version: responseDraft.version + 1 },
      revision: { revision: 1, state: "FINAL", current: true, created_at: new Date().toISOString() },
      submission: { request_id: fieldRequestID, submission_id: "field-submission-1", status: "SUBMITTED", submitted_at: new Date().toISOString(), version: 2 },
    };
  }

  // Legacy session/draft endpoints remain available to older component tests.
  if (url.pathname === "/api/v1/evidence/session" && method === "GET") return { session: { id: "field-session-1", request_id: fieldRequestID, audience_hint: "f***@example.com", expires_at: fieldAgentRequest.deadline }, request: fieldAgentRequest };
  if (url.pathname === "/api/v1/evidence/session/draft" && method === "GET") return structuredClone(responseDraft);
  if (url.pathname === "/api/v1/evidence/session/draft" && method === "PUT") {
    const input = parseBody(init) as { answers?: Record<string, unknown>; presentation_mode?: string; expected_version?: number };
    if (input.expected_version !== responseDraft.version) throw staticFailure(409, "draft_conflict", "The saved response changed");
    responseDraft = { answers: input.answers ?? {}, presentation_mode: input.presentation_mode ?? "AUTOMATIC", version: responseDraft.version + 1 };
    return structuredClone(responseDraft);
  }
  if (url.pathname === "/api/v1/evidence/artifacts" && method === "POST") {
    const file = init?.body instanceof FormData ? init.body.get("file") : null;
    const name = file instanceof File ? file.name : "site-photo.jpg";
    const mediaType = file instanceof File ? file.type || "image/jpeg" : "image/jpeg";
    const signature = name === "signature.png" && mediaType === "image/png";
    return { id: signature ? "artifact-agent-signature" : "artifact-site-photo", request_id: fieldRequestID, file_name: name, media_type: mediaType, size_bytes: file instanceof File ? file.size : 128000, sha256: signature ? "demo-signature-sha256" : "demo-photo-sha256", status: "STORED_UNSCANNED" };
  }
  if (url.pathname === "/api/v1/evidence/session/submissions" && method === "POST") return { request_id: fieldRequestID, submission_id: "field-submission-1", status: "SUBMITTED", submitted_at: new Date().toISOString(), version: 2 };
  return undefined;
}

function workspacePayload() {
  return {
    session: {
      id: "field-distribution-session-1",
      distribution_id: distributionID,
      request_id: fieldRequestID,
      audience_hint: "f***@example.com",
      assurance: "LINK_POSSESSION",
      expires_at: fieldAgentRequest.deadline,
      created_at: new Date().toISOString(),
    },
    request: fieldAgentRequest,
    workspace: responseWorkspace(),
  };
}

function responseWorkspace() {
  return {
    workspace: workspaceState(),
    answers: structuredClone(responseDraft.answers),
    presentation_mode: responseDraft.presentation_mode,
    field_sequences: structuredClone(fieldSequences),
  };
}

function workspaceState() {
  const now = new Date().toISOString();
  return {
    id: workspaceID,
    distribution_id: distributionID,
    status: "OPEN",
    version: responseDraft.version,
    created_at: now,
    updated_at: now,
  };
}

function redeemedSession() {
  return {
    session_id: "field-distribution-session-1",
    session_token: sessionToken,
    distribution_id: distributionID,
    request_id: fieldRequestID,
    audience_hint: "f***@example.com",
    assurance: "LINK_POSSESSION",
    expires_at: fieldAgentRequest.deadline,
  };
}

function isAnswered(value: unknown) {
  if (!value || typeof value !== "object") return false;
  const answer = value as { text?: unknown; values?: unknown[]; artifact_ids?: unknown[]; document?: unknown };
  return typeof answer.text === "string" ? answer.text.length > 0 : Boolean(answer.values?.length || answer.artifact_ids?.length || answer.document);
}

function parseBody(init?: RequestInit) {
  if (typeof init?.body !== "string") return {};
  try { return JSON.parse(init.body) as unknown; } catch { return {}; }
}

function staticFailure(status: number, code: string, message: string) {
  return Object.assign(new Error(message), { staticStatus: status, staticCode: code });
}
