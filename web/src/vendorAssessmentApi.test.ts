import { afterEach, describe, expect, it, vi } from "vitest";
import {
  completeVendorAssessment,
  loadCurrentVendorAssessment,
  requestVendorAssessmentClarification,
  sendVendorAssessmentRequest,
  startVendorAssessment,
  startVendorAssessmentReview,
} from "./vendorAssessmentApi";

const assessment = {
  id: "assessment-1",
  tenant_id: "bank",
  legal_entity_id: "entity",
  relationship_id: "relationship-1",
  review_kind: "ONBOARDING",
  stable_episode_key: "episode-1",
  status: "READY_TO_SEND",
  form_template_id: "form-1",
  form_template_version: 3,
  review_matter_id: "matter-1",
  review_due_at: "2026-09-30T17:00:00Z",
  started_by_principal_id: "owner-1",
  started_at: "2026-08-26T10:00:00Z",
  version: 2,
  created_at: "2026-08-26T10:00:00Z",
  updated_at: "2026-08-26T10:02:00Z",
};

describe("vendor assessment API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("loads the current relationship-scoped assessment using the route identifier", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify(assessment), { status: 200 })));
    vi.stubGlobal("fetch", fetchMock);

    const result = await loadCurrentVendorAssessment("relationship/1");

    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/v1/vendors/relationship%2F1/assessments/current");
    expect(result).toEqual({ assessment, setup: undefined });
  });

  it("preserves setup status from the current-assessment envelope", async () => {
    const envelope = { assessment, setup: { assessment_id: "assessment-1", state: "FAILED", attempts: 3, failure_code: "MATTER_CREATE_FAILED", terminal_at: "2026-08-26T10:05:00Z" } };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(envelope), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(loadCurrentVendorAssessment("relationship-1")).resolves.toEqual(envelope);
  });

  it("starts an assessment without browser-supplied identity or scope", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(assessment), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    await startVendorAssessment("relationship-1", {
      relationship_version: 4,
      form_template_id: "form-1",
      form_template_version: 3,
      review_due_at: "2026-09-30T17:00:00Z",
    });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendors/relationship-1/assessments");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({
      relationship_version: 4,
      form_template_id: "form-1",
      form_template_version: 3,
      review_due_at: "2026-09-30T17:00:00Z",
    });
  });

  it("sends the recipient only to the immediate secure-request command", async () => {
    const outcome = { assessment: { ...assessment, status: "COLLECTING", version: 3 }, request: { id: "request-1" }, state: "DELIVERED" };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(outcome), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await sendVendorAssessmentRequest("assessment/1", {
      expected_version: 2,
      audience: "security@vendor.example",
      deadline: "2026-09-20T17:00:00Z",
      invitation_ttl_minutes: 1440,
    });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendor-assessments/assessment%2F1/send-request");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({
      expected_version: 2,
      audience: "security@vendor.example",
      deadline: "2026-09-20T17:00:00Z",
      invitation_ttl_minutes: 1440,
    });
  });

  it("uses versioned commands for review, clarification and conclusion", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify(assessment), { status: 200 })));
    vi.stubGlobal("fetch", fetchMock);

    await startVendorAssessmentReview("assessment-1", { expected_version: 3 });
    await requestVendorAssessmentClarification("assessment-1", {
      expected_version: 4,
      request_fields: ["security_testing"],
      message: "Provide the current independent security test report.",
      deadline: "2026-09-12T17:00:00Z",
    });
    await completeVendorAssessment("assessment-1", {
      expected_version: 5,
      conclusion: "SATISFACTORY_WITH_CONDITIONS",
      rationale: "The reviewed evidence supports onboarding subject to the recorded finding.",
      uncertainty: "The next independent test is due in March 2027.",
      next_review_recommended_at: "2027-03-01T09:00:00Z",
    });

    expect(fetchMock.mock.calls.map((call) => call[0])).toEqual([
      "/api/v1/vendor-assessments/assessment-1/review/start",
      "/api/v1/vendor-assessments/assessment-1/clarifications",
      "/api/v1/vendor-assessments/assessment-1/complete",
    ]);
  });
});
