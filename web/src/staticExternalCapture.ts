const fieldRequestID = "field-visit-atm-042";
const sessionToken = "static-field-agent-session";

const fieldAgentRequest = {
  id: fieldRequestID,
  tenant_id: "bank-demo",
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
  if (url.pathname === "/api/v1/evidence/invitations/redeem" && method === "POST") {
    const input = parseBody(init) as { token?: string; audience?: string };
    if (input.token !== "field-agent-demo" || input.audience?.trim().toLowerCase() !== "field.agent@example.com") throw Object.assign(new Error("Invitation unavailable"), { staticStatus: 401, staticCode: "invitation_unavailable" });
    return { session_id: "field-session-1", session_token: sessionToken, request_id: fieldRequestID, audience_hint: "f***@example.com", expires_at: fieldAgentRequest.deadline };
  }
  const authorization = new Headers(init?.headers).get("Authorization");
  if (authorization !== `Bearer ${sessionToken}`) return undefined;
  if (url.pathname === "/api/v1/evidence/session" && method === "GET") return { session: { id: "field-session-1", request_id: fieldRequestID, audience_hint: "f***@example.com", expires_at: fieldAgentRequest.deadline }, request: fieldAgentRequest };
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

function parseBody(init?: RequestInit) {
  if (typeof init?.body !== "string") return {};
  try { return JSON.parse(init.body) as unknown; } catch { return {}; }
}
