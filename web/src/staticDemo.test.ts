import { beforeEach, describe, expect, it, vi } from "vitest";

async function demo() {
  vi.resetModules();
  vi.stubEnv("VITE_STATIC_DEMO", "true");
  return import("./staticDemo");
}

beforeEach(() => {
  localStorage.clear();
  window.history.replaceState(null, "", "/");
  vi.unstubAllEnvs();
});

describe("static stakeholder demo transport", () => {
  it("exposes clearly identified reference context and exact records", async () => {
    const { staticDemoRequest } = await demo();
    const context = await staticDemoRequest<{ demo_mode: boolean; actor: { role_codes: string[] }; tenant: { name: string }; capabilities: { config_read: boolean } }>("/api/v1/context");
    const programs = await staticDemoRequest<{ items: Array<{ program: { id: string } }> }>("/api/v1/program-summaries");
    const matters = await staticDemoRequest<{ items: Array<{ matter: { id: string } }> }>("/api/v1/matter-summaries?status=OPEN");
    const evidence = await staticDemoRequest<{ items: Array<{ id: string; estimated_minutes: number }> }>("/api/v1/evidence/requests");
    const readiness = await staticDemoRequest<{ baseline_known: boolean }>("/api/v1/compliance/readiness");
    const projections = await staticDemoRequest<{ items: Array<{ projection: string; state: string; pending: number; failed: number }> }>("/api/v1/operations/projections");

    expect(context.demo_mode).toBe(true);
    expect(context.tenant.name).toContain("Meridian Trust Bank");
    expect(context.actor.role_codes).toContain("CRO");
    expect(context.capabilities.config_read).toBe(true);
    expect(programs.items[0]?.program.id).toBe("program-ndpa");
    expect(matters.items[0]?.matter.id).toBe("matter-gaid-change");
    expect(evidence.items[0]?.id).toBe("evidence-annual-return");
    expect(evidence.items[0]?.estimated_minutes).toBe(2);
    expect(readiness.baseline_known).toBe(false);
    expect(projections.items[0]).toMatchObject({ projection: "program_state", state: "CURRENT", pending: 0, failed: 0 });
  });

  it("returns current candidate-set authority semantics for an exact review context", async () => {
    const { staticDemoRequest } = await demo();
    const resolution = await staticDemoRequest<{ candidate_principals: Array<{ id: string }>; strategy: string }>("/api/v1/authority/resolve", { method: "POST", body: JSON.stringify({ object_type: "PROGRAM", object_id: "program-ndpa", responsibility: "REVIEWER", materiality: 2 }) });
    expect(resolution.strategy).toBe("ANY_OF");
    expect(resolution.candidate_principals).toHaveLength(2);
  });

  it("persists onboarding progress and document review within the demo session", async () => {
    const { staticDemoRequest } = await demo();
    const initial = await staticDemoRequest<{ version: number; current_step: number }>("/api/v1/onboarding/state?guide_code=executive-first-run");
    const updated = await staticDemoRequest<{ version: number; current_step: number }>("/api/v1/onboarding/state?guide_code=executive-first-run", { method: "PUT", body: JSON.stringify({ current_step: 1, completed: false, dismissed: false }) });
    const reviewed = await staticDemoRequest<{ version: number; proposals: Array<{ status: string }> }>("/api/v1/document-imports/document-gaid/proposals/proposal-owner/review", { method: "POST", body: JSON.stringify({ status: "ACCEPTED" }) });

    expect(initial.current_step).toBe(0);
    expect(updated.current_step).toBe(1);
    expect(updated.version).toBe(initial.version + 1);
    expect(reviewed.proposals[0]?.status).toBe("ACCEPTED");
  });

  it("demonstrates source-backed Program coverage and governed gap actions", async () => {
    const { staticDemoRequest } = await demo();
    const coverage = await staticDemoRequest<{ version: number; metrics: { verified: { denominator: number }; requirement_mapped: { numerator: number } }; candidates: Array<{ anchor: { page?: number }; matches: unknown[] }>; suggestions: Array<{ id: string; status: string }> }>("/api/v1/document-imports/document-gaid/coverage?limit=25");
    expect(coverage.metrics.verified.denominator).toBe(2);
    expect(coverage.metrics.requirement_mapped.numerator).toBe(1);
    expect(coverage.candidates[0]?.anchor.page).toBe(3);
    expect(coverage.candidates[0]?.matches).toHaveLength(1);

    const applied = await staticDemoRequest<{ assessment: { version: number; suggestions: Array<{ status: string }> }; object_type: string }>("/api/v1/document-imports/document-gaid/coverage/suggestions/suggestion-review/apply", { method: "POST", body: JSON.stringify({ expected_version: coverage.version }) });
    expect(applied.object_type).toBe("REQUIREMENT");
    expect(applied.assessment.version).toBe(coverage.version + 1);
    expect(applied.assessment.suggestions[0]?.status).toBe("APPLIED");
  });

  it("executes a governed Program transition and rejects stale replays", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    const current = await staticDemoRequest<{ program: { status: string; version: number } }>("/api/v1/programs/program-ndpa");
    const updated = await staticDemoRequest<{ program: { status: string; version: number } }>("/api/v1/programs/program-ndpa/transition", {
      method: "POST",
      body: JSON.stringify({ expected_version: current.program.version, to: "PAUSED", rationale: "Pause while ownership is corrected." }),
    });

    expect(updated.program.status).toBe("PAUSED");
    expect(updated.program.version).toBe(current.program.version + 1);
    await expect(staticDemoRequest("/api/v1/programs/program-ndpa/transition", {
      method: "POST",
      body: JSON.stringify({ expected_version: current.program.version, to: "RETIRED", rationale: "Stale replay." }),
    })).rejects.toMatchObject({ status: 409, code: "version_conflict" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
  });

  it("can deterministically exercise permission and conflict fixtures", async () => {
    window.history.replaceState(null, "", "/?fixture=authority-forbidden");
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    await expect(staticDemoRequest("/api/v1/authority/resolve", { method: "POST", body: "{}" })).rejects.toMatchObject({ status: 403, code: "permission_denied" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
  });

  it("supports the vendor due-diligence journey with the current approved typed form", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    const forms = await staticDemoRequest<{ items: Array<{ id: string; status: string; is_current: boolean; presentation: { default_mode: string; allow_mode_switch: boolean }; fields: Array<{ type: string }> }> }>("/api/v1/form-templates?limit=100&tenant_id=bank-demo");

    expect(forms.items).toHaveLength(1);
    expect(forms.items[0]).toMatchObject({ status: "ACTIVE", is_current: true, presentation: { default_mode: "WIZARD", allow_mode_switch: true } });
    expect(forms.items[0]?.fields.map((field) => field.type)).toEqual(expect.arrayContaining(["email", "yes_no", "single_select", "vendor_document", "attestation"]));

    await expect(staticDemoRequest("/api/v1/vendors/vendor-relationship-payments/assessments/current")).rejects.toMatchObject({ status: 404, code: "vendor_assessment_not_found" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);

    const started = await staticDemoRequest<{ id: string; status: string; version: number }>("/api/v1/vendors/vendor-relationship-payments/assessments", {
      method: "POST",
      body: JSON.stringify({ relationship_version: 1, form_template_id: forms.items[0]?.id, form_template_version: 3, review_due_at: "2099-09-30T23:59:59.000Z" }),
    });
    expect(started).toMatchObject({ status: "READY_TO_SEND", version: 2 });

    const current = await staticDemoRequest<{ assessment: { id: string; status: string } }>("/api/v1/vendors/vendor-relationship-payments/assessments/current");
    expect(current.assessment).toMatchObject({ id: started.id, status: "READY_TO_SEND" });

    const sent = await staticDemoRequest<{ assessment: { status: string; current_request_id: string }; state: string; capture_url?: string; delivery: { status: string; recipient_hint: string } }>(`/api/v1/vendor-assessments/${started.id}/send-request`, {
      method: "POST",
      body: JSON.stringify({ expected_version: started.version, audience: "security@acme.example", deadline: "2099-09-20T23:59:59.000Z", invitation_ttl_minutes: 1440 }),
    });
    expect(sent).toMatchObject({ assessment: { status: "COLLECTING", current_request_id: "vendor-request-payments-2026" }, state: "DELIVERED", delivery: { status: "DELIVERED", recipient_hint: "s***@acme.example" } });
    expect(sent.capture_url).toBeUndefined();
  });

  it("renders a submitted vendor response from bounded review fixture data", async () => {
    window.history.replaceState(null, "", "/?fixture=vendor-submitted");
    const { staticDemoRequest } = await demo();
    const current = await staticDemoRequest<{ assessment: { id: string; status: string; submission_id: string } }>("/api/v1/vendors/vendor-relationship-payments/assessments/current");
    const review = await staticDemoRequest<{ assessment: { status: string }; answers: Array<{ visibility: string }>; coverage: { answered_required: number; required_fields: number }; documents: Array<{ artifact_id: string }>; matters: unknown[] }>(`/api/v1/vendor-assessments/${current.assessment.id}`);

    expect(current.assessment).toMatchObject({ status: "SUBMITTED", submission_id: "vendor-submission-payments-2026" });
    expect(review.assessment.status).toBe("SUBMITTED");
    expect(review.answers.some((answer) => answer.visibility === "VISIBLE")).toBe(true);
    expect(review.coverage).toMatchObject({ answered_required: 4, required_fields: 4 });
    expect(review.documents).toHaveLength(1);
    expect(review.matters).toEqual([]);

    const started = await staticDemoRequest<{ status: string; version: number }>(`/api/v1/vendor-assessments/${current.assessment.id}/review/start`, { method: "POST", body: JSON.stringify({ expected_version: 4 }) });
    expect(started).toMatchObject({ status: "UNDER_REVIEW", version: 5 });
    const completed = await staticDemoRequest<{ status: string; version: number; conclusion: string; conclusion_rationale: string }>(`/api/v1/vendor-assessments/${current.assessment.id}/complete`, { method: "POST", body: JSON.stringify({ expected_version: 5, conclusion: "SATISFACTORY_WITH_CONDITIONS", rationale: "Proceed after the access-control action is complete." }) });
    expect(completed).toMatchObject({ status: "COMPLETED", version: 6, conclusion: "SATISFACTORY_WITH_CONDITIONS", conclusion_rationale: "Proceed after the access-control action is complete." });
  });

  it("provides deterministic vendor recovery, collection and completion fixtures", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    for (const [fixture, status] of [["vendor-ready", "READY_TO_SEND"], ["vendor-collecting", "COLLECTING"], ["vendor-completed", "COMPLETED"]] as const) {
      window.history.replaceState(null, "", `/?fixture=${fixture}`);
      const current = await staticDemoRequest<{ assessment: { status: string } }>("/api/v1/vendors/vendor-relationship-payments/assessments/current");
      expect(current.assessment.status).toBe(status);
    }

    window.history.replaceState(null, "", "/?fixture=vendor-source-degraded");
    await expect(staticDemoRequest("/api/v1/form-templates?limit=100")).rejects.toMatchObject({ status: 503, code: "vendor_forms_unavailable" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);

    window.history.replaceState(null, "", "/?fixture=vendor-partial-delivery");
    const ready = await staticDemoRequest<{ assessment: { id: string; version: number } }>("/api/v1/vendors/vendor-relationship-payments/assessments/current");
    const partial = await staticDemoRequest<{ state: string; capture_url?: string; delivery: { status: string } }>(`/api/v1/vendor-assessments/${ready.assessment.id}/send-request`, { method: "POST", body: JSON.stringify({ expected_version: ready.assessment.version, audience: "security@acme.example", deadline: "2099-09-20T23:59:59Z", invitation_ttl_minutes: 1440 }) });
    expect(partial).toMatchObject({ state: "LINK_CREATED_EMAIL_NOT_SENT", delivery: { status: "FAILED" } });
    expect(partial.capture_url).toContain("capture_invite=");
  });
});
