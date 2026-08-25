import { afterEach, describe, expect, it, vi } from "vitest";
import { submitCaptureSession, submitInternalCaptureRequest } from "./captureApi";

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
});
