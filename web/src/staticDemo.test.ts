import { beforeEach, describe, expect, it, vi } from "vitest";
import fixtures from "../public/static-demo-fixtures.json";
import { canRespondToEvidenceRequest } from "./evidenceAuthorization";
import type { EvidenceRequest } from "./types";

async function demo() {
  vi.resetModules();
  vi.stubEnv("VITE_STATIC_DEMO", "true");
  const module = await import("./staticDemo");
  await module.loadStaticDemoFixtures(async () => new Response(JSON.stringify(fixtures)));
  return module;
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

  it("removes current Program and issue relationships while preserving version checks", async () => {
    const { staticDemoRequest } = await demo();
    const program = await staticDemoRequest<{ program: { version: number }; requirement_control_links: Array<{ id: string }> }>("/api/v1/programs/program-ndpa");
    const updatedProgram = await staticDemoRequest<{ program: { version: number }; requirement_control_links: Array<{ id: string }> }>("/api/v1/programs/program-ndpa/control-links/coverage-link-1/retirement", { method: "POST", body: JSON.stringify({ expected_version: program.program.version, rationale: "The coverage mapping is incorrect." }) });
    const matter = await staticDemoRequest<{ matter: { version: number }; links: Array<{ id: string }> }>("/api/v1/matters/matter-gaid-change");
    const updatedMatter = await staticDemoRequest<{ matter: { version: number }; links: Array<{ id: string }> }>("/api/v1/matters/matter-gaid-change/links/link-1/retirement", { method: "POST", body: JSON.stringify({ expected_version: matter.matter.version, rationale: "The issue no longer affects this Program." }) });

    expect(updatedProgram.requirement_control_links).toHaveLength(0);
    expect(updatedProgram.program.version).toBe(program.program.version + 1);
    expect(updatedMatter.links).toHaveLength(0);
    expect(updatedMatter.matter.version).toBe(matter.matter.version + 1);
  });

  it("makes the Today evidence shortcut eligible only for the verified static CRO", async () => {
    const { staticDemoRequest } = await demo();
    const context = await staticDemoRequest<{ actor: { id: string } }>("/api/v1/context");
    const today = await staticDemoRequest<{ items: Array<{ action_target_type?: string; action_target_id?: string }> }>("/api/v1/today");
    const targetID = today.items.find((item) => item.action_target_type === "EVIDENCE_REQUEST")?.action_target_id;

    expect(targetID).toBe("evidence-annual-return");
    const evidence = await staticDemoRequest<EvidenceRequest>(`/api/v1/evidence/requests/${targetID}`);
    expect(evidence).toMatchObject({
      status: "READY",
      audience_type: "INTERNAL",
      recipient: { type: "INTERNAL_PRINCIPAL", state: "ASSIGNED", principal_id: context.actor.id },
    });
    expect(canRespondToEvidenceRequest(evidence, context.actor.id)).toBe(true);
    expect(canRespondToEvidenceRequest(evidence, "role-dpo")).toBe(false);
  });

  it("revalidates the static not-found fixture only after an eligible Today preload", async () => {
    window.history.replaceState(null, "", "/?fixture=capture-not-found");
    const { staticDemoRequest } = await demo();

    const preloadPath = "/api/v1/evidence/requests/evidence-annual-return?request_intent=eligibility_preload";
    expect(canRespondToEvidenceRequest(await staticDemoRequest<EvidenceRequest>(preloadPath), "role-cro")).toBe(true);
    expect(canRespondToEvidenceRequest(await staticDemoRequest<EvidenceRequest>(preloadPath), "role-cro")).toBe(true);
    await expect(staticDemoRequest("/api/v1/evidence/requests/evidence-annual-return")).rejects.toMatchObject({ status: 404, code: "request_not_found" });
  });

  it("revalidates the static terminal fixture only after an eligible Today preload", async () => {
    window.history.replaceState(null, "", "/?fixture=capture-terminal");
    const { staticDemoRequest } = await demo();

    const preloadPath = "/api/v1/evidence/requests/evidence-annual-return?request_intent=eligibility_preload";
    expect(canRespondToEvidenceRequest(await staticDemoRequest<EvidenceRequest>(preloadPath), "role-cro")).toBe(true);
    expect(canRespondToEvidenceRequest(await staticDemoRequest<EvidenceRequest>(preloadPath), "role-cro")).toBe(true);
    const current = await staticDemoRequest<EvidenceRequest>("/api/v1/evidence/requests/evidence-annual-return");
    expect(current.status).toBe("EXPIRED");
    expect(canRespondToEvidenceRequest(current, "role-cro")).toBe(false);
  });

  it("exposes Program responsibilities and executes evidence and monitoring workflows", async () => {
    const { staticDemoRequest } = await demo();
    const operations = await staticDemoRequest<{ operations: Array<{ command: string; assigned_to?: { display_name: string } }> }>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo");
    expect(operations.operations.find((value) => value.command === "program.details.update")?.assigned_to?.display_name).toBe("Data Protection Officer");
    expect(operations.operations.find((value) => value.command === "program.evidence.assess")?.assigned_to?.display_name).toBe("Data Protection Compliance Officer");

    const current = await staticDemoRequest<{ program: { version: number }; evidence_contracts: unknown[] }>("/api/v1/programs/program-ndpa");
    const defined = await staticDemoRequest<{ program: { version: number }; evidence_contracts: unknown[] }>("/api/v1/programs/program-ndpa/evidence-contracts", { method: "POST", body: JSON.stringify({ expected_version: current.program.version, requirement_id: "req-2", code: "RETURN-RECEIPT", name: "Filing receipt", claim: "The annual return was accepted.", acceptable_source_ids: ["source-ndpc"], population_scope: {}, freshness_minutes: 43200, minimum_coverage: 1, independence_required: true, contradiction_policy: "REVIEW", failure_action: "MATTER", status: "ACTIVE" }) });
    expect(defined.evidence_contracts).toHaveLength(current.evidence_contracts.length + 1);
    expect(defined.program.version).toBe(current.program.version + 1);

    const checks = await staticDemoRequest<{ items: Array<{ id: string }> }>("/api/v1/programs/program-ndpa/monitoring-checks?tenant_id=bank-demo");
    const results = await staticDemoRequest<{ items: Array<{ evaluation: { coverage: number } }> }>(`/api/v1/monitoring-checks/${checks.items[0]!.id}/results?tenant_id=bank-demo`);
    expect(results.items[0]?.evaluation.coverage).toBe(.8);
  });

  it("can deterministically exercise permission and conflict fixtures", async () => {
    window.history.replaceState(null, "", "/?fixture=authority-forbidden");
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    await expect(staticDemoRequest("/api/v1/authority/resolve", { method: "POST", body: "{}" })).rejects.toMatchObject({ status: 403, code: "permission_denied" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
  });
});
