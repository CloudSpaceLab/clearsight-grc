import { beforeEach, describe, expect, it, vi } from "vitest";
import { createDistribution, loadDistributionPage, loadRecipientCandidates, loadResponseRevisions } from "./formsDistributionApi";

const fetchMock = vi.fn();
beforeEach(() => { fetchMock.mockReset(); vi.stubGlobal("fetch", fetchMock); });

describe("forms distribution API", () => {
  it("normalizes the legacy list projection and preserves server keyset filters", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ items: [{ ID: "dist-a", FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectType: "VENDOR", SubjectID: "vendor-a", Title: "Review", Purpose: "Collect evidence", AccessPolicy: "DIRECT_LINK_EMAIL_OTP", Status: "OPEN", Deadline: "2026-09-01T10:00:00Z", RouteExpiresAt: "2026-09-01T09:00:00Z", Version: 2, CreatedAt: "2026-08-28T10:00:00Z", UpdatedAt: "2026-08-28T11:00:00Z" }], next_cursor: "next" }), { status: 200, headers: { "Content-Type": "application/json" } }));
    const page = await loadDistributionPage({ status: "OPEN", due_state: "OVERDUE", subject_type: "VENDOR", owner: "owner-a", limit: 25 });
    expect(page.items[0]).toMatchObject({ id: "dist-a", form_template_version: 3, status: "OPEN" });
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain("status=OPEN"); expect(url).toContain("due_state=OVERDUE"); expect(url).toContain("subject_type=VENDOR"); expect(url).toContain("owner=owner-a");
  });

  it("pins the exact form revision when dispatching", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ distribution: { id: "dist-a", form_template_id: "form-a", form_template_version: 7, subject_type: "CONTROL", subject_id: "c1", title: "Review", purpose: "Evidence", access_policy: "DIRECT_MAGIC_LINK", status: "OPEN", deadline: "2026-09-01T10:00:00Z", route_expires_at: "2026-09-01T09:00:00Z", version: 2, created_at: "2026-08-28T10:00:00Z", updated_at: "2026-08-28T10:00:00Z" }, recipients: [], workspace: { ID: "w1", Status: "OPEN", Version: 1, UpdatedAt: "2026-08-28T10:00:00Z" } }), { status: 201, headers: { "Content-Type": "application/json" } }));
    await createDistribution({ form_template_id: "form-a", form_template_version: 7, subject_type: "CONTROL", subject_id: "c1", title: "Review", purpose: "Evidence", access_policy: "DIRECT_MAGIC_LINK", estimated_minutes: 10, deadline: "2026-09-01T10:00:00Z", route_expires_at: "2026-09-01T09:00:00Z", recipients: [{ role: "TO", type: "INTERNAL_PRINCIPAL", principal_id: "p1" }] });
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toMatchObject({ form_template_id: "form-a", form_template_version: 7 });
  });

  it("uses the safe recipient candidate and immutable response routes", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ items: [], has_more: false }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await loadRecipientCandidates("ops", 12);
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/api/v1/forms/recipient-candidates?search=ops&limit=12");
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ items: [{ id: "r1", revision: 2, achieved_assurance: "EMAIL_VERIFIED", scored_weight_coverage: 100, state: "FINAL", current: true, created_at: "2026-08-28T10:00:00Z" }] }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await expect(loadResponseRevisions("dist/a")).resolves.toMatchObject({ items: [{ revision: 2, current: true }] });
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain("/api/v1/forms/distributions/dist%2Fa/responses?limit=100");
  });
});
