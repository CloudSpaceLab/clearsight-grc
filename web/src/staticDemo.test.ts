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

  it("exposes stored sample collection states without deriving dates from viewer time", async () => {
    window.history.replaceState(null, "", "/?fixture=collection-renewal-states");
    const { staticDemoRequest } = await demo();
    const forms = await staticDemoRequest<{ items: Array<{ id: string; name: string }> }>("/api/v1/form-templates?limit=100");
    const checks = await staticDemoRequest<{ items: Array<{ id: string; collection_policy?: { validity_months: number; renewal_window_days: number; reminder_count: number } }> }>("/api/v1/programs/program-ndpa/monitoring-checks?limit=100");
    const summaries = await staticDemoRequest<{ items: Array<{ monitoring_check_id: string; currency_state: string; latest_submission_at?: string; respondent_label?: string; expires_at: string; reminders_sent: number; reminder_count: number; delivery_state: string }> }>("/api/v1/programs/program-ndpa/collection-summaries?limit=100");

    expect(forms.items.some((form) => form.id === "form-no-policy")).toBe(true);
    expect(checks.items.find((check) => check.id === "check-no-policy")?.collection_policy).toBeUndefined();
    expect(checks.items.find((check) => check.id === "check-current")?.collection_policy).toEqual({ validity_months: 12, renewal_window_days: 30, reminder_count: 3 });
    expect(summaries.items.map((summary) => summary.currency_state)).toEqual(expect.arrayContaining(["CURRENT", "RENEWAL_DUE", "RESPONSE_POTENTIALLY_EXPIRED", "AWAITING_RESPONSE", "RENEWAL_BLOCKED"]));
    expect(summaries.items.find((summary) => summary.monitoring_check_id === "check-current")).toMatchObject({
      latest_submission_at: "2026-08-14T10:30:00Z",
      respondent_label: "Ada Okafor · Vendor assurance lead",
      expires_at: "2027-08-14T10:30:00Z",
    });
    expect(summaries.items.find((summary) => summary.monitoring_check_id === "check-awaiting")).toMatchObject({ reminders_sent: 1, reminder_count: 3 });
    expect(summaries.items.find((summary) => summary.monitoring_check_id === "check-blocked")?.delivery_state).toBe("BLOCKED");
  });

  it("provides a deterministic long-name collection fixture", async () => {
    window.history.replaceState(null, "", "/?fixture=collection-long-content");
    const { staticDemoRequest } = await demo();
    const checks = await staticDemoRequest<{ items: Array<{ name: string }> }>("/api/v1/programs/program-ndpa/monitoring-checks?limit=100");
    expect(checks.items[0]?.name).toContain("cross-border payment-processing");
  });
});
