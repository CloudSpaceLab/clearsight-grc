import { afterEach, describe, expect, it, vi } from "vitest";
import {
  acceptVendorWork,
  cancelVendorWork,
  loadVendorWork,
  prepareVendorWork,
  requestVendorWorkChanges,
  retryVendorWorkDelivery,
  sendVendorWork,
  startVendorWorkReview,
} from "./vendorWorkApi";

const work = {
  id: "work-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1",
  relationship_link_id: "link-1", target_type: "PROGRAM" as const, target_id: "program-1",
  purpose: "Confirm annual resilience controls", instructions: "Complete the form and attach the current test report.",
  owner_principal_id: "owner-1", form_template_id: "form-1", form_template_version: 4,
  presentation: "WIZARD" as const, current_request_id: "capture-1", current_capture_sequence: 1,
  state: "PREPARING" as const, delivery_state: "NOT_SENT" as const,
  due_at: "2026-09-30T17:00:00Z", version: 2, created_at: "2026-08-26T10:00:00Z", updated_at: "2026-08-26T10:01:00Z",
};

describe("vendor work API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("loads bounded work for an exact target", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [work] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(loadVendorWork({ target_type: "PROGRAM", target_id: "program/1", limit: 20 })).resolves.toEqual({ items: [work] });
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/v1/vendor-work?target_type=PROGRAM&target_id=program%2F1&limit=20");
  });

  it("prepares work without browser-supplied scope or actor identity", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(work), { status: 202 }));
    vi.stubGlobal("fetch", fetchMock);

    await prepareVendorWork("relationship/1", {
      relationship_link_id: "link-1", purpose: work.purpose, instructions: work.instructions,
      form_template_id: "form-1", form_template_version: 4, presentation: "WIZARD",
      vendor_audience: "assurance@vendor.example", due_at: "2026-09-30T17:00:00Z",
    });

    const call = fetchMock.mock.calls[0];
    expect(call?.[0]).toBe("/api/v1/vendors/relationship%2F1/work/prepare");
    expect(JSON.parse(String((call?.[1] as RequestInit).body))).toEqual({
      relationship_link_id: "link-1", purpose: work.purpose, instructions: work.instructions,
      form_template_id: "form-1", form_template_version: 4, presentation: "WIZARD",
      vendor_audience: "assurance@vendor.example", due_at: "2026-09-30T17:00:00Z",
    });
  });

  it("uses the relationship and work identifiers for every versioned command", async () => {
    const outcome = { work, state: "DELIVERED" };
    const fetchMock = vi.fn().mockImplementation((_url, init?: RequestInit) => Promise.resolve(new Response(JSON.stringify(String(_url).endsWith("/send") || String(_url).endsWith("/changes") || String(_url).endsWith("/retry") ? outcome : work), { status: 200 })));
    vi.stubGlobal("fetch", fetchMock);

    await sendVendorWork("relationship/1", "work/1", { expected_version: 2, vendor_audience: "assurance@vendor.example", invitation_ttl_minutes: 10080 });
    await startVendorWorkReview("relationship/1", "work/1", { expected_version: 3 });
    await requestVendorWorkChanges("relationship/1", "work/1", { expected_version: 4, message: "Attach the signed schedule.", field_ids: ["signed_schedule"], vendor_audience: "assurance@vendor.example", due_at: "2026-10-07T17:00:00Z", invitation_ttl_minutes: 10080 });
    await acceptVendorWork("relationship/1", "work/1", { expected_version: 5, rationale: "The requested response and document were reviewed." });
    await cancelVendorWork("relationship/1", "work/1", { expected_version: 6, reason: "The Program no longer needs this response." });
    await retryVendorWorkDelivery("relationship/1", "work/1", { expected_version: 7, vendor_audience: "assurance@vendor.example", invitation_ttl_minutes: 10080 });

    expect(fetchMock.mock.calls.map((call) => call[0])).toEqual([
      "/api/v1/vendors/relationship%2F1/work/work%2F1/send",
      "/api/v1/vendors/relationship%2F1/work/work%2F1/review/start",
      "/api/v1/vendors/relationship%2F1/work/work%2F1/changes",
      "/api/v1/vendors/relationship%2F1/work/work%2F1/accept",
      "/api/v1/vendors/relationship%2F1/work/work%2F1/cancel",
      "/api/v1/vendors/relationship%2F1/work/work%2F1/retry",
    ]);
    expect(JSON.parse(String((fetchMock.mock.calls[2]?.[1] as RequestInit).body))).toEqual({ expected_version: 4, message: "Attach the signed schedule.", field_ids: ["signed_schedule"], vendor_audience: "assurance@vendor.example", due_at: "2026-10-07T17:00:00Z", invitation_ttl_minutes: 10080 });
  });
});
