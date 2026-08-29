import { afterEach, describe, expect, it, vi } from "vitest";
import {
  FormWorkspaceConflictError,
  loadCaptureDraft,
  saveCaptureDraft,
  saveFormResponseWorkspace,
  submitCaptureSession,
  submitInternalCaptureRequest,
  uploadCaptureSessionArtifact,
  uploadInternalCaptureArtifact,
} from "./captureApi";

vi.mock("./api", () => ({
  loadContext: vi.fn().mockResolvedValue({ tenant: { id: "tenant-demo" } }),
}));

describe("capture API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("writes structured answers for internal requests while accepting legacy scalar callers", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      request_id: "request/1",
      status: "SUBMITTED",
      submitted_at: "2026-08-26T12:00:00Z",
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);

    await submitInternalCaptureRequest("request/1", 7, {
      contact: "security@vendor.example",
      services: { values: ["Payments", "Hosting"] },
      certificate: {
        document: {
          artifact_id: "artifact-1",
          document_type: "ISO 27001 certificate",
          reference: "CERT-2026-81",
          issued_by: "Accredited Certification Limited",
          issued_on: "2026-01-10",
          expires_on: "2027-01-09",
        },
      },
    });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/evidence/requests/request%2F1/submissions?tenant_id=tenant-demo");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({
      tenant_id: "tenant-demo",
      expected_version: 7,
      answers: {
        contact: { text: "security@vendor.example" },
        services: { values: ["Payments", "Hosting"] },
        certificate: {
          document: {
            artifact_id: "artifact-1",
            document_type: "ISO 27001 certificate",
            reference: "CERT-2026-81",
            issued_by: "Accredited Certification Limited",
            issued_on: "2026-01-10",
            expires_on: "2027-01-09",
          },
        },
      },
    });
  });

  it("writes typed answers through the request-scoped external session", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      request_id: "request-2",
      status: "SUBMITTED",
      submitted_at: "2026-08-26T12:00:00Z",
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);

    await submitCaptureSession("session-secret", 3, {
      approved: { text: "Yes" },
      evidence: { artifact_ids: ["artifact-2"] },
    });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/evidence/session/submissions");
    expect(new Headers((call[1] as RequestInit).headers).get("Authorization")).toBe("Bearer session-secret");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({
      expected_version: 3,
      answers: {
        approved: { text: "Yes" },
        evidence: { artifact_ids: ["artifact-2"] },
      },
    });
  });

  it("loads and saves a draft through the bearer session without client scope fields", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ answers: {}, presentation_mode: "WIZARD", version: 0 }), { status: 200, headers: { "Content-Type": "application/json" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ answers: { approved: { text: "Yes" } }, presentation_mode: "WIZARD", version: 1, updated_at: "2026-08-26T12:00:00Z" }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);

    await loadCaptureDraft("session-secret");
    await saveCaptureDraft("session-secret", {
      answers: { approved: { text: "Yes" } }, presentation_mode: "WIZARD", expected_version: 0,
    });

    const [loadCall, saveCall] = fetchMock.mock.calls;
    if (!loadCall || !saveCall) throw new Error("draft requests were not made");
    expect(loadCall[0]).toBe("/api/v1/evidence/session/draft");
    expect(new Headers((loadCall[1] as RequestInit).headers).get("Authorization")).toBe("Bearer session-secret");
    expect(saveCall[0]).toBe("/api/v1/evidence/session/draft");
    expect((saveCall[1] as RequestInit).method).toBe("PUT");
    expect(new Headers((saveCall[1] as RequestInit).headers).get("Authorization")).toBe("Bearer session-secret");
    expect(JSON.parse(String((saveCall[1] as RequestInit).body))).toEqual({
      answers: { approved: { text: "Yes" } }, presentation_mode: "WIZARD", expected_version: 0,
    });
  });

  it("preserves field-level conflict details from shared workspace PATCH", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      current_version: 9,
      changed_fields: [{ field_id: "owner", server_value: { text: "Server owner" }, sequence: 4 }],
    }), { status: 409, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(saveFormResponseWorkspace("session-secret", {
      expected_version: 8,
      presentation_mode: "AUTOMATIC",
      edits: [{ field_id: "owner", value: { text: "Local owner" }, base_sequence: 3 }],
    })).rejects.toMatchObject<FormWorkspaceConflictError>({
      name: "FormWorkspaceConflictError",
      conflict: {
        current_version: 9,
        changed_fields: [{ field_id: "owner", server_value: { text: "Server owner" }, sequence: 4 }],
      },
    });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("workspace PATCH was not made");
    expect(call[0]).toBe("/api/v1/evidence/session/workspace");
    expect((call[1] as RequestInit).method).toBe("PATCH");
    expect(new Headers((call[1] as RequestInit).headers).get("Authorization")).toBe("Bearer session-secret");
  });

  it("binds internal and external artifact uploads to the selected form field", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ id: "artifact-1" }), { status: 201, headers: { "Content-Type": "application/json" } })));
    vi.stubGlobal("fetch", fetchMock);
    const file = new File(["evidence"], "evidence.txt", { type: "text/plain" });

    await uploadInternalCaptureArtifact("request-1", file, "documents");
    await uploadCaptureSessionArtifact("session-secret", file, "documents");

    const internalBody = fetchMock.mock.calls[0]?.[1]?.body as FormData;
    const externalCall = fetchMock.mock.calls[1];
    const externalBody = externalCall?.[1]?.body as FormData;
    expect(internalBody.get("request_id")).toBe("request-1");
    expect(internalBody.get("field_id")).toBe("documents");
    expect(externalBody.get("field_id")).toBe("documents");
    expect(new Headers(externalCall?.[1]?.headers).get("Authorization")).toBe("Bearer session-secret");
  });
});
