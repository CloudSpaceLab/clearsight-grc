import { afterEach, describe, expect, it, vi } from "vitest";
import {
  cancelVendorAssessment,
  completeVendorAssessment,
  createVendorAssessmentDeficiency,
  loadCurrentVendorAssessment,
  reissueVendorAssessmentRequest,
  reviewVendorAssessmentDocument,
  retryVendorAssessmentSetup,
  requestVendorAssessmentClarification,
  sendVendorAssessmentRequest,
  startVendorAssessment,
  startVendorAssessmentReview,
} from "./vendorAssessmentApi";
import * as vendorAssessmentApi from "./vendorAssessmentApi";

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

  it("builds an exact assessment and request scoped document URL", () => {
    const api = vendorAssessmentApi as typeof vendorAssessmentApi & { vendorAssessmentDocumentURL?: (assessmentID: string, requestID: string, artifactID: string) => string };
    expect(typeof api.vendorAssessmentDocumentURL).toBe("function");
    expect(api.vendorAssessmentDocumentURL?.("assessment/1", "request/7", "artifact/3")).toBe("/api/v1/vendor-assessments/assessment%2F1/requests/request%2F7/documents/artifact%2F3/open");
  });

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

  it("sends replacement-link details only to the reissue command", async () => {
    const outcome = { assessment: { ...assessment, status: "COLLECTING", version: 4 }, request: { id: "request-1" }, state: "DELIVERED" };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(outcome), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await reissueVendorAssessmentRequest("assessment/1", {
      expected_version: 3,
      audience: "security@vendor.example",
      invitation_ttl_minutes: 1440,
    });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendor-assessments/assessment%2F1/reissue-request");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({
      expected_version: 3,
      audience: "security@vendor.example",
      invitation_ttl_minutes: 1440,
    });
  });

  it("retries terminal setup using only the current assessment version", async () => {
    const outcome = { assessment: { ...assessment, status: "SETUP_PENDING", version: 4 }, setup: { assessment_id: "assessment-1", state: "READY", attempts: 0, next_attempt_at: "2026-08-26T10:10:00Z" } };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(outcome), { status: 202 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(retryVendorAssessmentSetup("assessment/1", { expected_version: 3 })).resolves.toEqual(outcome);

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendor-assessments/assessment%2F1/setup/retry");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({ expected_version: 3 });
  });

  it("uses versioned commands for review, clarification and conclusion", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify(assessment), { status: 200 })));
    vi.stubGlobal("fetch", fetchMock);

    await startVendorAssessmentReview("assessment-1", { expected_version: 3 });
    await requestVendorAssessmentClarification("assessment-1", {
      expected_version: 4,
      request_fields: ["security_testing"],
      message: "Provide the current independent security test report.",
      audience: "security@vendor.example",
      deadline: "2026-09-12T17:00:00Z",
      invitation_ttl_minutes: 1440,
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

  it("cancels the current assessment with a reason and current version", async () => {
    const cancelled = { ...assessment, status: "CANCELLED", version: 3, cancellation_reason: "The proposed service is no longer being procured." };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(cancelled), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await cancelVendorAssessment("assessment/1", { expected_version: 2, reason: "The proposed service is no longer being procured." });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendor-assessments/assessment%2F1/cancel");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({ expected_version: 2, reason: "The proposed service is no longer being procured." });
  });

  it("sends exact bounded follow-up and document-decision bodies", async () => {
    const clarification = { assessment: { ...assessment, status: "COLLECTING", version: 5 }, state: "DELIVERED" };
    const deficiency = { assessment: { ...assessment, status: "UNDER_REVIEW", version: 6 }, matter: { matter: { id: "matter-2", reference: "MAT-002", title: "Current security test required", status: "OPEN" } } };
    const refreshed = { assessment: { ...assessment, status: "UNDER_REVIEW", version: 7 }, requests: [], answers: [], coverage: { visible_fields: 0, answered_fields: 0, required_fields: 0, answered_required: 0, ratio: 1 }, documents: [], matters: [] };
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(clarification), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(deficiency), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(refreshed), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await requestVendorAssessmentClarification("assessment/1", {
      expected_version: 4,
      request_fields: ["security-testing"],
      message: "Provide the current independent security test report.",
      audience: "security@vendor.example",
      deadline: "2026-09-12T23:59:59.000Z",
      invitation_ttl_minutes: 60,
    });
    await createVendorAssessmentDeficiency("assessment/1", {
      expected_version: 5,
      trigger_key: "security-test-report",
      title: "Current security test required",
      summary: "The submitted report is no longer current for this review.",
      due_at: "2026-09-20T23:59:59.000Z",
    });
    await reviewVendorAssessmentDocument("assessment/1", "artifact/1", {
      expected_version: 6,
      decision: "VALIDATE",
      document_type: "SOC_2_TYPE_II",
      evidence_class: "BANK_VALIDATED",
      valid_until: "2027-05-31",
    });

    expect(fetchMock.mock.calls.map((call) => [call[0], JSON.parse(String((call[1] as RequestInit).body))])).toEqual([
      ["/api/v1/vendor-assessments/assessment%2F1/clarifications", {
        expected_version: 4, request_fields: ["security-testing"], message: "Provide the current independent security test report.", audience: "security@vendor.example", deadline: "2026-09-12T23:59:59.000Z", invitation_ttl_minutes: 60,
      }],
      ["/api/v1/vendor-assessments/assessment%2F1/deficiencies", {
        expected_version: 5, trigger_key: "security-test-report", title: "Current security test required", summary: "The submitted report is no longer current for this review.", due_at: "2026-09-20T23:59:59.000Z",
      }],
      ["/api/v1/vendor-assessments/assessment%2F1/documents/artifact%2F1/validate", {
        expected_version: 6, decision: "VALIDATE", document_type: "SOC_2_TYPE_II", evidence_class: "BANK_VALIDATED", valid_until: "2027-05-31",
      }],
    ]);
  });
});
