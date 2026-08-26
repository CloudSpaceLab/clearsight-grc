import { afterEach, describe, expect, it, vi } from "vitest";
import {
  issueEvidenceInvitation,
  listEvidenceActiveSessions,
  listEvidenceInvitationMetadata,
  listEvidenceRecipientCandidates,
  replaceEvidenceInvitation,
  revokeEvidenceInvitation,
  revokeEvidenceSession,
} from "./evidenceRequestAdminApi";

describe("evidence requester administration API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("loads sanitized invitation metadata from the bounded nested request route", async () => {
    const fetch = vi.fn().mockResolvedValue(jsonResponse({ items: [{
      id: "invitation-1",
      request_id: "request/2026",
      audience_hint: "a***@supplier.example",
      purpose: "Provide the quarter-end access review",
      expires_at: "2026-09-01T12:00:00Z",
      max_redemptions: 1,
      redemptions: 0,
      created_at: "2026-08-26T12:00:00Z",
    }] }));
    vi.stubGlobal("fetch", fetch);

    const items = await listEvidenceInvitationMetadata("request/2026");

    expect(items).toHaveLength(1);
    expect(items[0]?.audience_hint).toBe("a***@supplier.example");
    expect(items[0]).not.toHaveProperty("token");
    expect(fetch).toHaveBeenCalledWith("/api/v1/evidence/requests/request%2F2026/invitations", expect.objectContaining({
      credentials: "include",
      method: "GET",
    }));
  });

  it("normalizes a null invitation collection to an empty bounded inventory", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ items: null })));

    await expect(listEvidenceInvitationMetadata("request-1")).resolves.toEqual([]);
  });

  it("loads a bounded sanitized active-session page from the exact request route", async () => {
    const fetch = vi.fn().mockResolvedValue(jsonResponse({ items: [{ id: "session-1", audience_hint: "a***@supplier.example", expires_at: "2026-09-01T12:00:00Z", created_at: "2026-08-26T12:00:00Z" }], has_more: true }));
    vi.stubGlobal("fetch", fetch);

    await expect(listEvidenceActiveSessions("request/2026")).resolves.toEqual({
      items: [{ id: "session-1", audience_hint: "a***@supplier.example", expires_at: "2026-09-01T12:00:00Z", created_at: "2026-08-26T12:00:00Z" }],
      has_more: true,
    });
    expect(fetch).toHaveBeenCalledWith("/api/v1/evidence/requests/request%2F2026/sessions?limit=50", expect.objectContaining({ credentials: "include", method: "GET" }));
    expect(fetch.mock.calls[0]?.[0]).not.toContain("token");
  });

  it("loads labelled internal recipient candidates from the exact request route", async () => {
    const fetch = vi.fn().mockResolvedValue(jsonResponse({ items: [{ principal_id: "person-1", display_name: "Ada Okafor", context_label: "Privacy Operations Lead" }], has_more: true }));
    vi.stubGlobal("fetch", fetch);

    await expect(listEvidenceRecipientCandidates("request/1", "  privacy lead  ")).resolves.toEqual({ items: [{ principal_id: "person-1", display_name: "Ada Okafor", context_label: "Privacy Operations Lead" }], has_more: true });
    expect(fetch).toHaveBeenCalledWith("/api/v1/evidence/requests/request%2F1/recipient-candidates?limit=50&q=privacy+lead", expect.objectContaining({ credentials: "include", method: "GET" }));
  });

  it("issues and replaces an invitation without sending identity fields or retaining a prior token", async () => {
    const fetch = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ invitation_id: "invitation-1", token: "one-time-token", audience_hint: "a***@supplier.example", expires_at: "2026-09-01T12:00:00Z" }, 201))
      .mockResolvedValueOnce(jsonResponse({ invitation_id: "invitation-2", token: "replacement-token", audience_hint: "a***@supplier.example", expires_at: "2026-09-02T12:00:00Z" }, 201));
    vi.stubGlobal("fetch", fetch);
    const input = { audience: "assurance@supplier.example", purpose: "Provide the quarter-end access review", ttlMinutes: 1440 };

    const issued = await issueEvidenceInvitation("request-1", input);
    const replaced = await replaceEvidenceInvitation("request-1", "invitation/1", input);

    expect(issued.token).toBe("one-time-token");
    expect(replaced.token).toBe("replacement-token");
    expect(requestBody(fetch, 0)).toEqual({ audience: input.audience, purpose: input.purpose, ttl_minutes: 1440 });
    expect(requestBody(fetch, 0)).not.toHaveProperty("tenant_id");
    expect(requestBody(fetch, 0)).not.toHaveProperty("actor_principal_id");
    expect(fetch.mock.calls[1]?.[0]).toBe("/api/v1/evidence/requests/request-1/invitations/invitation%2F1/replace");
    expect(requestBody(fetch, 1)).toEqual({ audience: input.audience, purpose: input.purpose, ttl_minutes: 1440 });
  });

  it("revokes an invitation or active external session through its request-bounded route", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetch);

    await revokeEvidenceInvitation("request/1", "invitation/1");
    await revokeEvidenceSession("request/1", "session/1");

    expect(fetch).toHaveBeenNthCalledWith(1, "/api/v1/evidence/requests/request%2F1/invitations/invitation%2F1/revoke", expect.objectContaining({ credentials: "include", method: "POST" }));
    expect(fetch).toHaveBeenNthCalledWith(2, "/api/v1/evidence/requests/request%2F1/sessions/session%2F1/revoke", expect.objectContaining({ credentials: "include", method: "POST" }));
  });
});

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function requestBody(fetch: ReturnType<typeof vi.fn>, callIndex: number) {
  const init = fetch.mock.calls[callIndex]?.[1] as RequestInit;
  return JSON.parse(String(init.body)) as Record<string, unknown>;
}
