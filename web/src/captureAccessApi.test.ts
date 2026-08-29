import { afterEach, describe, expect, it, vi } from "vitest";
import {
  loadFormResponseWorkspace,
  redeemFormAccess,
  saveFormResponseWorkspace,
  sendFormAccessOTP,
  startFormAccess,
  submitFormResponseWorkspace,
  verifyFormAccessOTP,
} from "./captureApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));

describe("protected form access API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("starts the access ceremony with only the opaque route selector", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ policy: "DIRECT_MAGIC_LINK", expires_at: "2026-09-01T12:00:00Z" }));
    vi.stubGlobal("fetch", fetchMock);

    await startFormAccess("route-secret");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [path, init] = fetchMock.mock.calls[0] ?? [];
    expect(path).toBe("/api/v1/evidence/access/start");
    expect((init as RequestInit).method).toBe("POST");
    expect(JSON.parse(String((init as RequestInit).body))).toEqual({ route_selector: "route-secret" });
  });

  it("sends and verifies OTP with server-issued selectors and challenge IDs", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ challenge_id: "challenge-1", hint: "a***@example.com", expires_at: "2026-09-01T12:00:00Z" }))
      .mockResolvedValueOnce(jsonResponse({ session_id: "session-1", session_token: "bearer-secret", distribution_id: "distribution-1", request_id: "request-1", audience_hint: "a***@example.com", assurance: "EMAIL_VERIFIED", expires_at: "2026-09-01T12:00:00Z" }));
    vi.stubGlobal("fetch", fetchMock);

    await sendFormAccessOTP("route-secret", "recipient-selector");
    await verifyFormAccessOTP("route-secret", "challenge-1", "123456");

    expect(JSON.parse(String((fetchMock.mock.calls[0]?.[1] as RequestInit).body))).toEqual({
      route_selector: "route-secret",
      recipient_selector: "recipient-selector",
    });
    expect(JSON.parse(String((fetchMock.mock.calls[1]?.[1] as RequestInit).body))).toEqual({
      route_selector: "route-secret",
      challenge_id: "challenge-1",
      code: "123456",
    });
  });

  it("redeems direct links without adding identity fields", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ session_id: "session-1", session_token: "bearer-secret" }));
    vi.stubGlobal("fetch", fetchMock);

    await redeemFormAccess("route-secret");

    const body = JSON.parse(String((fetchMock.mock.calls[0]?.[1] as RequestInit).body));
    expect(body).toEqual({ route_selector: "route-secret" });
    expect(JSON.stringify(body)).not.toMatch(/email|phone|audience/i);
  });

  it("keeps the bearer in the Authorization header for workspace read, save and submit", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ session: {}, request: {}, workspace: {} }))
      .mockResolvedValueOnce(jsonResponse({ workspace: { version: 4 }, answers: {}, presentation_mode: "AUTOMATIC", field_sequences: {} }))
      .mockResolvedValueOnce(jsonResponse({ workspace: {}, revision: {}, submission: {} }, 201));
    vi.stubGlobal("fetch", fetchMock);

    await loadFormResponseWorkspace("bearer-secret");
    await saveFormResponseWorkspace("bearer-secret", {
      expected_version: 3,
      presentation_mode: "AUTOMATIC",
      edits: [{ field_id: "legal_name", value: { text: "Example Ltd" }, base_sequence: 2 }],
    });
    await submitFormResponseWorkspace("bearer-secret", { expected_version: 4 });

    const [load, save, submit] = fetchMock.mock.calls;
    for (const call of [load, save, submit]) {
      expect(new Headers((call?.[1] as RequestInit).headers).get("Authorization")).toBe("Bearer bearer-secret");
      expect(String(call?.[0])).not.toContain("bearer-secret");
      expect(String((call?.[1] as RequestInit).body ?? "")).not.toContain("bearer-secret");
    }
    expect(load?.[0]).toBe("/api/v1/evidence/session/workspace");
    expect((save?.[1] as RequestInit).method).toBe("PATCH");
    expect(submit?.[0]).toBe("/api/v1/evidence/session/workspace/submissions");
  });
});

function jsonResponse(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}
