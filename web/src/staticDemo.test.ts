import { beforeEach, describe, expect, it, vi } from "vitest";
import fixtures from "./staticDemoFixtures.json";
import { canRespondToEvidenceRequest } from "./evidenceAuthorization";
import type { EvidenceRequest } from "./types";
// @ts-expect-error The static-only runtime is a browser asset with no TypeScript declarations.
await import("./staticDemoWorkflowRuntime.js");

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
  it("loads one base-path runtime and can recover after an asset loads without installing it", async () => {
    vi.resetModules(); vi.stubEnv("VITE_STATIC_DEMO", "true"); const module = await import("./staticDemo");
    const scope = globalThis as typeof globalThis & { ClearSightStaticWorkflowRuntime?: unknown }, runtime = scope.ClearSightStaticWorkflowRuntime; delete scope.ClearSightStaticWorkflowRuntime;
    const first = module.loadStaticDemoWorkflowRuntime("/clearsight-grc/");
    const second = module.loadStaticDemoWorkflowRuntime("/clearsight-grc/");
    const scripts = document.head.querySelectorAll<HTMLScriptElement>('script[data-clearsight-static-runtime="true"]');
    expect(scripts).toHaveLength(1);
    expect(scripts[0]!.src).toContain("/clearsight-grc/");
    expect(scripts[0]!.src).toContain("staticDemoWorkflowRuntime");
    const firstFailure = expect(first).rejects.toThrow("did not initialize");
    const secondFailure = expect(second).rejects.toThrow("did not initialize");
    scripts[0]!.dispatchEvent(new Event("load"));
    await Promise.all([firstFailure, secondFailure]);
    expect(scripts[0]!.isConnected).toBe(false);

    const retry = module.loadStaticDemoWorkflowRuntime("/clearsight-grc/");
    const retryScript = document.head.querySelector<HTMLScriptElement>('script[data-clearsight-static-runtime="true"]')!;
    scope.ClearSightStaticWorkflowRuntime = runtime;
    retryScript.dispatchEvent(new Event("load"));
    await retry;
    retryScript.remove();
  });

  it("loads fixtures from the configured deployment base path", async () => {
    vi.resetModules();
    vi.stubEnv("VITE_STATIC_DEMO", "true");
    const module = await import("./staticDemo");
    const requested: string[] = [];

    await module.loadStaticDemoFixtures(async (input) => {
      requested.push(String(input));
      return new Response(JSON.stringify(fixtures));
    }, "/clearsight-grc/");

    expect(requested).toHaveLength(1);
    expect(requested[0]).toContain("/clearsight-grc/");
    expect(requested[0]).toContain("staticDemoFixtures");
  });

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

  it("returns an onboarding guide for the requested workspace surface", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    const today = await staticDemoRequest<{ code: string; surface: string }>("/api/v1/onboarding/guide?surface=TODAY");
    const vendors = await staticDemoRequest<{ code: string; surface: string; role_codes: string[]; steps: Array<{ id: string; action: string; intent?: string }> }>("/api/v1/onboarding/guide?surface=VENDORS&code=vendor-operations-first-run");

    expect(today).toMatchObject({ code: "executive-first-run", surface: "TODAY" });
    expect(vendors).toMatchObject({ code: "vendor-operations-first-run", surface: "VENDORS" });
    expect(vendors.role_codes).toContain("BUSINESS_OWNER");
    expect(vendors.steps.find((step) => step.id === "finish")).toMatchObject({ action: "Open next vendor task", intent: "open-vendor-next-action" });
    await expect(staticDemoRequest("/api/v1/onboarding/guide?surface=UNKNOWN")).rejects.toMatchObject({ status: 404, code: "not_found" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
    await expect(staticDemoRequest("/api/v1/onboarding/guide?surface=VENDORS&code=executive-first-run")).rejects.toMatchObject({ status: 404, code: "not_found" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
    await expect(staticDemoRequest("/api/v1/onboarding/guide?surface=TODAY&code=vendor-operations-first-run")).rejects.toMatchObject({ status: 404, code: "not_found" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
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

  it("switches static demo personas so distinct owners and reviewers can complete their work", async () => {
    const { staticDemoRequest } = await demo();
    const accounts = await staticDemoRequest<{ accounts: Array<{ username: string; label: string }> }>("/api/v1/demo/accounts");
    expect(accounts.accounts.map((item) => item.label)).toEqual(expect.arrayContaining(["Data Protection Officer", "Data Protection Compliance Officer"]));

    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    expect((await staticDemoRequest<any>("/api/v1/context")).actor.id).toBe("role-dpo");
    expect((await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/operations?tenant_id=bank-demo")).operations.find((item: any) => item.command === "matter.details.update").can_act).toBe(true);
    expect((await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo")).operations.find((item: any) => item.command === "program.details.update").can_act).toBe(true);

    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpco@demo.clearsight" }) });
    const matterOperations = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/operations?tenant_id=bank-demo");
    expect(matterOperations.operations.find((item: any) => item.command === "matter.outcome.record").can_act).toBe(true);
    const programOperations = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo");
    expect(programOperations.operations.find((item: any) => item.command === "program.evidence.assess" && item.subresource_id === "contract-return").can_act).toBe(true);
    expect(programOperations.operations.filter((item: any) => item.command === "program.evidence.assess" && item.subresource_id === "contract-training")).toEqual([expect.objectContaining({ can_act: false, assigned_to: expect.objectContaining({ id: "role-training-reviewer" }) })]);
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

  it("executes the complete stateful Matter workflow through the static transport", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    const operations = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/operations?tenant_id=bank-demo");
    expect(operations.matter_version).toBe(8);
    expect(operations.operations.map((item: any) => `${item.command}:${item.subresource_id ?? ""}`)).toEqual(expect.arrayContaining([
      "matter.details.update:", "matter.context.change:", "matter.assign:", "matter.action.add:",
      "matter.action.update:action-1", "matter.action.assign:action-1", "matter.action.transition:action-1",
      "matter.outcome.define:",
      "matter.outcome.supersede:verify-1", "matter.outcome.retire:verify-1", "matter.outcome.record:verify-1", "matter.unlink:link-1",
    ]));

    let aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change");
    const staleVersion = aggregate.matter.version;
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/details", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, title: "Implement the revised GAID return", summary: "Update the return and retain approval evidence.", priority: 3, due_at: "2027-03-01T12:00:00Z", scope: { filing_year: 2027 }, rationale: "The approved filing plan changed." }) });
    expect(aggregate.matter).toMatchObject({ title: "Implement the revised GAID return", priority: 3, version: staleVersion + 1 });
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/context-changes", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, kind: "RESOLVE_MISSING", label: "Final DPCO review date", key: "dpco_review_date", value: "1 March 2027", rationale: "The DPCO approved the timetable." }) });
    expect(aggregate.matter.known_facts.dpco_review_date).toBe("1 March 2027");
    expect(aggregate.matter.missing_facts).not.toContain("Final DPCO review date");
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/assignment", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, owner_principal_id: "role-deputy-dpo", rationale: "The deputy owns this filing." }) });
    expect(aggregate.matter.owner_principal_id).toBe("role-deputy-dpo");
    expect((await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/operations?tenant_id=bank-demo")).operations.find((item: any) => item.command === "matter.details.update")).toMatchObject({ can_act: false, assigned_to: { id: "role-deputy-dpo" } });

    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/actions", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, title: "Confirm filing receipt", description: "Retain the accepted receipt.", owner_principal_id: "role-privacy-control", due_at: "2027-03-02T12:00:00Z" }) });
    const action = aggregate.actions.at(-1);
    expect(action).toMatchObject({ title: "Confirm filing receipt", status: "PLANNED", version: 1 });
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/actions/${action.id}`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, title: "Confirm accepted filing receipt", description: "Retain the regulator acceptance receipt.", due_at: "2027-03-03T12:00:00Z", rationale: "Clarify the required evidence." }) });
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/actions/${action.id}/assignment`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, owner_principal_id: "role-dpo", rationale: "The DPO will retain the receipt." }) });
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/actions/${action.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, to: "IN_PROGRESS", rationale: "Receipt collection started." }) });
    expect(aggregate.actions.find((item: any) => item.id === action.id)).toMatchObject({ title: "Confirm accepted filing receipt", owner_principal_id: "role-dpo", status: "IN_PROGRESS", version: 4 });

    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/decisions", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, type: "FILING_CHANNEL", status: "APPROVED", options: ["Portal"], selected_option: "Portal", rationale: "Use the regulator portal." }) });
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/responses", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, purpose: "Annual return", audience: "NDPC", manifest: { receipt_required: true } }) });
    const response = aggregate.response_packages.at(-1);
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpco@demo.clearsight" }) });
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/responses/${response.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, to: "IN_REVIEW", rationale: "The response is ready for review." }) });
    expect(aggregate.response_packages.at(-1).status).toBe("IN_REVIEW");
    const responseOperations = (await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/operations?tenant_id=bank-demo")).operations.filter((item: any) => item.command === "matter.response.transition" && item.subresource_id === response.id);
    expect(responseOperations).toEqual(expect.arrayContaining([
      expect.objectContaining({ responsibility: "SIGNATORY", assigned_to: expect.objectContaining({ id: "role-cro" }), allowed_targets: ["APPROVED"] }),
      expect.objectContaining({ responsibility: "REVIEWER", assigned_to: expect.objectContaining({ id: "role-dpco" }), allowed_targets: expect.arrayContaining(["REJECTED", "DRAFT"]) }),
      expect.objectContaining({ responsibility: "PROPOSER", assigned_to: expect.objectContaining({ id: "role-dpo" }), allowed_targets: ["WITHDRAWN"] }),
    ]));
    const currentFilingDecision = aggregate.decisions.at(-1);
    expect((await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/operations?tenant_id=bank-demo")).operations.filter((item: any) => item.command === "matter.decision.record" && item.subresource_id === currentFilingDecision.id)).toEqual(expect.arrayContaining([
      expect.objectContaining({ responsibility: "PROPOSER", allowed_targets: ["PROPOSED"] }),
      expect.objectContaining({ responsibility: "AUTHORIZER", allowed_targets: expect.arrayContaining(["SUPERSEDED", "EXPIRED"]) }),
    ]));
    const responseStatusHistory = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/responses/${response.id}/history?tenant_id=bank-demo&limit=20`);
    expect(responseStatusHistory.items.map((item: any) => item.status)).toEqual(["DRAFT", "IN_REVIEW"]);

    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/verification-contracts/verify-1/retire", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, rationale: "Replace the original sample check with the filing receipt check." }) });
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/verification-contracts", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, action_id: action.id, expected_outcome: "The regulator accepted the annual return.", baseline: {}, scope: { filing_year: 2027 }, threshold: { accepted: true }, observation_period_minutes: 0, reviewer_candidate_id: "role-dpco", failure_response: "REOPEN" }) });
    const check = aggregate.verification_contracts.at(-1);
    expect(check).toMatchObject({ status: "ACTIVE", authority_principal_id: "role-dpco", version: 1 });
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/verification-contracts/${check.id}/supersede`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, action_id: action.id, expected_outcome: "The regulator accepted the complete return.", baseline: {}, scope: { filing_year: 2027 }, threshold: { accepted: true }, observation_period_minutes: 0, reviewer_candidate_id: "role-dpco", failure_response: "REOPEN", rationale: "Clarify completeness." }) });
    const replacement = aggregate.verification_contracts.at(-1);
    expect(aggregate.verification_contracts.find((item: any) => item.id === check.id).status).toBe("RETIRED");
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/verification-contracts", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, expected_outcome: "A temporary duplicate check.", baseline: {}, scope: {}, threshold: {}, observation_period_minutes: 0, reviewer_candidate_id: "role-dpco", failure_response: "BLOCK_CLOSE" }) });
    const temporaryCheck = aggregate.verification_contracts.at(-1);
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/verification-contracts/${temporaryCheck.id}/retire`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, rationale: "The duplicate check is not required." }) });
    expect(aggregate.verification_contracts.at(-1).status).toBe("RETIRED");
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/actions/action-1/transition", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, to: "IMPLEMENTED", rationale: "The evidence checklist is complete." }) });
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/actions/${action.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, to: "IMPLEMENTED", rationale: "The accepted receipt is retained." }) });
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/verification-results", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, contract_id: replacement.id, result: "PASS", observations: { accepted: true }, evidence_references: ["receipt-1"], rationale: "The accepted receipt was checked.", observed_at: "2027-03-03T15:00:00Z" }) });
    expect(aggregate.verification_results.at(-1)).toMatchObject({ contract_id: replacement.id, result: "PASS", reviewer_principal_id: "role-dpco" });
    expect(aggregate.closure).toEqual({ ready: true, reasons: [] });
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/links", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, program_id: "program-ndpa", relationship: "IMPLEMENTS" }) });
    const link = aggregate.links.at(-1);
    aggregate = await staticDemoRequest<any>(`/api/v1/matters/matter-gaid-change/links/${link.id}/retirement`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, rationale: "This relationship was replaced." }) });
    expect(aggregate.links.some((item: any) => item.id === link.id)).toBe(false);
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/transition", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, to: "VERIFICATION", rationale: "Implementation is ready for final closure." }) });
    expect(aggregate.status_label).toBe("Outcome verification");
    expect((await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/operations?tenant_id=bank-demo")).operations.find((item: any) => item.command === "matter.transition" && item.allowed_targets?.includes("CLOSED"))).toBeTruthy();
    aggregate = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/transition", { method: "POST", body: JSON.stringify({ expected_version: aggregate.matter.version, to: "CLOSED", rationale: "The accepted filing receipt confirms the outcome." }) });
    expect(aggregate).toMatchObject({ status_label: "Closed", matter: { status: "CLOSED" } });
    const historic = await staticDemoRequest<any>("/api/v1/matters/matter-gaid-change/history?tenant_id=bank-demo&at=2026-08-06T15%3A30%3A00Z");
    expect(historic.matter).toMatchObject({ title: "Implement GAID annual-return evidence requirements", version: 8 });

    await expect(staticDemoRequest("/api/v1/matters/matter-gaid-change/details", { method: "POST", body: JSON.stringify({ expected_version: staleVersion, title: "Stale" }) })).rejects.toMatchObject({ status: 409, code: "version_conflict" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
  });

  it("revises, assigns and transitions Program safeguards and evidence checks statefully", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    let aggregate = await staticDemoRequest<any>("/api/v1/programs/program-ndpa");
    const safeguard = aggregate.control_implementations[0];
    const staleProgramVersion = aggregate.program.version;
    aggregate = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/control-implementations/${safeguard.id}/details`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, expected_implementation_version: safeguard.version ?? 1, name: "Annual privacy return review", description: "DPO review and evidence sign-off.", implementation_type: "PROCESS", scope: { filing_year: 2027 }, effective_from: "2026-01-01T00:00:00Z", rationale: "Align the safeguard with the current filing cycle." }) });
    expect(aggregate.control_implementations[0]).toMatchObject({ name: "Annual privacy return review", version: 2 });
    const operations = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo");
    const digest = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/review-digest?tenant_id=bank-demo");
    expect(operations.program_version).toBe(aggregate.program.version);
    expect(digest.current_program_version).toBe(aggregate.program.version);
    expect((await staticDemoRequest<any>("/api/v1/programs/program-ndpa/history?tenant_id=bank-demo&at=2026-08-06T15%3A30%3A00Z")).program).toMatchObject({ name: "Nigeria Data Protection Programme", version: 12 });
    aggregate = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/control-implementations/${safeguard.id}/assignment`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, expected_implementation_version: 2, owner_principal_id: "role-privacy-control", rationale: "The privacy control owner performs the review." }) });
    aggregate = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/control-implementations/${safeguard.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, expected_implementation_version: 3, to: "RETIRED", rationale: "The safeguard was replaced." }) });
    expect(aggregate.control_implementations[0]).toMatchObject({ owner_principal_id: "role-privacy-control", status: "RETIRED", version: 4 });

    const evidence = aggregate.evidence_contracts[0];
    aggregate = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/evidence-contracts/${evidence.id}/revision`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, expected_contract_version: evidence.version ?? 1, name: "Annual return acceptance evidence", claim: "The regulator accepted the annual return.", acceptable_source_ids: ["source-ndpc"], population_scope: { filing_year: 2027 }, freshness_minutes: 525600, minimum_coverage: 1, independence_required: true, contradiction_policy: "REVIEW", failure_action: "MATTER", rationale: "Measure the accepted filing." }) });
    const revised = aggregate.evidence_contracts.find((item: any) => item.id === evidence.id);
    expect(revised).toMatchObject({ status: "DRAFT", version: 2, configured_by: "role-dpo" });
    await expect(staticDemoRequest(`/api/v1/programs/program-ndpa/evidence-contracts/${revised.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, expected_contract_version: 2, to: "ACTIVE", rationale: "Attempt self-approval." }) })).rejects.toMatchObject({ status: 409, code: "maker_checker_required" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpco@demo.clearsight" }) });
    aggregate = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/evidence-contracts/${revised.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, expected_contract_version: 2, to: "ACTIVE", rationale: "Independent review approved this check." }) });
    expect(aggregate.evidence_contracts.find((item: any) => item.id === evidence.id)).toMatchObject({ status: "ACTIVE", version: 3 });
    expect((await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo")).operations.find((item: any) => item.command === "program.evidence.assess" && item.subresource_id === evidence.id).can_act).toBe(true);
    aggregate = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/evidence-contracts/${evidence.id}/assessments`, { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, contract_id: evidence.id, conclusion: "SUPPORTED", coverage: 1, basis: { summary: "The accepted receipt was reviewed." }, assessed_at: "2026-08-26T12:00:00Z" }) });
    expect(aggregate.evidence_assessments.at(-1)).toMatchObject({ contract_id: evidence.id, assessed_by: "role-dpco" });

    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    aggregate = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/assignment", { method: "POST", body: JSON.stringify({ expected_version: aggregate.program.version, owner_principal_id: "role-deputy-dpo", rationale: "The deputy now owns the Program." }) });
    expect((await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo")).operations.find((item: any) => item.command === "program.details.update")).toMatchObject({ can_act: false, assigned_to: { id: "role-deputy-dpo" } });

    await expect(staticDemoRequest(`/api/v1/programs/program-ndpa/control-implementations/${safeguard.id}/details`, { method: "POST", body: JSON.stringify({ expected_version: staleProgramVersion, expected_implementation_version: 4 }) })).rejects.toMatchObject({ status: 409, code: "version_conflict" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
  });

  it("runs Program-scoped monitoring forms and linked issue creation with maker-checker state", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    let operations = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo");
    expect(operations.operations.find((item: any) => item.command === "program.monitoring.form.define").can_act).toBe(true);

    let form = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/form-templates", { method: "POST", body: JSON.stringify({ code: "RETURN-OWNER", name: "Return owner confirmation", purpose: "Confirm every return section owner.", fields: [{ id: "owner", label: "Owner", type: "text", required: true }] }) });
    expect((await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo")).operations.find((item: any) => item.command === "program.monitoring.form.transition" && item.subresource_id === form.id).allowed_targets).toEqual(["PENDING_APPROVAL"]);
    form = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/form-templates/${form.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: form.version, to: "PENDING_APPROVAL" }) });
    expect(form).toMatchObject({ status: "PENDING_APPROVAL", submitted_by: "role-dpo" });
    await expect(staticDemoRequest(`/api/v1/programs/program-ndpa/form-templates/${form.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: form.version, to: "ACTIVE" }) })).rejects.toMatchObject({ status: 409, code: "maker_checker_required" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpco@demo.clearsight" }) });
    form = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/form-templates/${form.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: form.version, to: "ACTIVE" }) });
    expect(form).toMatchObject({ status: "ACTIVE", approved_by: "role-dpco" });
    operations = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/operations?tenant_id=bank-demo");
    expect(operations.operations.find((item: any) => item.command === "program.monitoring.collect" && item.subresource_id === form.id)).toBeTruthy();
    const collection = await staticDemoRequest<any>(`/api/v1/programs/program-ndpa/form-templates/${form.id}/collections`, { method: "POST", body: JSON.stringify({ form_template_version: form.version, period_start: "2026-08-01T00:00:00Z", period_end: "2026-08-31T00:00:00Z", deadline: "2026-09-02T00:00:00Z" }) });
    expect(collection.id).toContain("evidence-monitoring-");
    expect((await staticDemoRequest<any>("/api/v1/evidence/requests")).items).toEqual(expect.arrayContaining([expect.objectContaining({ id: collection.id })]));
    expect(await staticDemoRequest<any>(`/api/v1/evidence/requests/${collection.id}`)).toMatchObject({ id: collection.id, deadline: "2026-09-02T00:00:00Z" });

    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    let check = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/monitoring-checks", { method: "POST", body: JSON.stringify({ code: "RETURN-OWNER-CHECK", name: "Return owner confirmation", claim: "Every section has an owner.", input_kind: "FORM", form_template_id: form.id, form_template_version: form.version, freshness_minutes: 10080, minimum_coverage: 1, failure_action: "RECOMMEND_MATTER" }) });
    check = await staticDemoRequest<any>(`/api/v1/monitoring-checks/${check.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: check.version, to: "PENDING_APPROVAL" }) });
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpco@demo.clearsight" }) });
    check = await staticDemoRequest<any>(`/api/v1/monitoring-checks/${check.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: check.version, to: "ACTIVE" }) });
    expect(check).toMatchObject({ status: "ACTIVE", submitted_by: "role-dpo", approved_by: "role-dpco" });

    const linked = await staticDemoRequest<any>("/api/v1/monitoring-results/monitor-result-1/linked-issue", { method: "POST", body: "{}" });
    expect(linked).toMatchObject({ created: true, matter: { id: "matter-monitor-return", known_facts: { monitoring_result_id: "monitor-result-1" } } });
    expect((await staticDemoRequest<any>(`/api/v1/matters/${linked.matter.id}`)).matter.id).toBe(linked.matter.id);
  });

  it("configures connected monitoring data and isolates evaluated results and linked issues by check", async () => {
    const { staticDemoRequest } = await demo();
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    const source = await staticDemoRequest<any>("/api/v1/evidence/sources", { method: "POST", body: JSON.stringify({ legal_entity_id: "bank-ng", code: "RETURN-STATUS", name: "Annual return status", type: "SYSTEM", authority_class: "INTERNAL_CONTROL", owner_principal_id: "role-dpo", endpoint: "https://records.example.test/returns", expected_freshness_minutes: 60 }) });
    const connection = await staticDemoRequest<any>(`/api/v1/config/sources/${source.id}/connections`, { method: "POST", body: JSON.stringify({ code: "RETURN-STATUS-REST", name: "Annual return status endpoint", adapter_kind: "REST_JSON", adapter_version: "rest-json-v1", definition: { base_url: "https://records.example.test" }, declared_capabilities: ["INSPECT", "PAGE"] }) });
    let view = await staticDemoRequest<any>(`/api/v1/config/source-connections/${connection.connection_id}/views`, { method: "POST", body: JSON.stringify({ connection_version: connection.version, code: "RETURN-STATUS-VIEW", name: "Annual return status view", definition: { path: "/returns" }, output_kind: "RECORDS" }) });
    view = (await staticDemoRequest<any>(`/api/v1/config/source-views/${view.view_id}/inspect?version=${view.version}`, { method: "POST", body: JSON.stringify({ stable_keys: [] }) })).view;
    view = (await staticDemoRequest<any>(`/api/v1/config/source-views/${view.view_id}/inspect?version=${view.version}`, { method: "POST", body: JSON.stringify({ stable_keys: ["status"] }) })).view;
    const binding = await staticDemoRequest<any>(`/api/v1/config/source-views/${view.view_id}/bindings`, { method: "POST", body: JSON.stringify({ view_version: view.version, code: "RETURN-STATUS-MONITOR", name: "Annual return status monitoring", purpose: "monitoring", operations: ["PAGE"], selected_fields: ["status"], key_fields: ["status"] }) });
    let check = await staticDemoRequest<any>("/api/v1/programs/program-ndpa/monitoring-checks", { method: "POST", body: JSON.stringify({ code: "RETURN-STATUS-CHECK", name: "Annual return status", claim: "The annual return is accepted.", input_kind: "SOURCE", binding_id: binding.binding_id, binding_version: binding.version, source_rules: [{ id: "accepted", field: "status", operator: "EQUALS", expected: "accepted", risk_points: 100, critical: true }], failure_action: "RECOMMEND_MATTER" }) });
    check = await staticDemoRequest<any>(`/api/v1/monitoring-checks/${check.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: check.version, to: "PENDING_APPROVAL" }) });
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpco@demo.clearsight" }) });
    check = await staticDemoRequest<any>(`/api/v1/monitoring-checks/${check.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: check.version, to: "ACTIVE" }) });
    const result = await staticDemoRequest<any>(`/api/v1/monitoring-checks/${check.id}/evaluate-source`, { method: "POST", body: JSON.stringify({ check_version: check.version }) });
    expect(result).toMatchObject({ monitoring_check_id: check.id, monitoring_check_version: check.version });
    expect(result.evaluation.rule_results).toEqual([expect.objectContaining({ rule_id: "accepted", field_id: "status" })]);
    expect((await staticDemoRequest<any>(`/api/v1/monitoring-checks/${check.id}/results`)).items).toEqual([result]);
    const linked = await staticDemoRequest<any>(`/api/v1/monitoring-results/${result.id}/linked-issue`, { method: "POST", body: "{}" });
    expect(linked.matter).toMatchObject({ id: `matter-${check.id}`, known_facts: { monitoring_result_id: result.id } });
    expect((await staticDemoRequest<any>(`/api/v1/matters/${linked.matter.id}`)).matter.id).toBe(linked.matter.id);
  });

  it("keeps newly created Programs and issues operable on their returned exact routes", async () => {
    const { staticDemoRequest } = await demo();
    await staticDemoRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    let program = await staticDemoRequest<any>("/api/v1/programs", { method: "POST", body: JSON.stringify({ code: "NEW-PRIV", name: "New privacy Program", type: "PRIVACY", owning_function: "Data Protection Office", owner_candidate_id: "role-dpo", approval_authority_candidate_id: "role-cro", jurisdiction: "Nigeria", scope: {}, effective_from: "2026-08-26T00:00:00Z" }) });
    const createdOperations = await staticDemoRequest<any>(`/api/v1/programs/${program.program.id}/operations?tenant_id=bank-demo`);
    expect(createdOperations.program_id).toBe("program-created");
    expect(createdOperations.operations.find((item: any) => item.command === "program.review.accept").can_act).toBe(false);
    expect(await staticDemoRequest<any>(`/api/v1/programs/${program.program.id}/review-digest`)).toMatchObject({ state: "NO_BASELINE", checkpoint: null, current_program_version: 1, current_projection_version: 0 });
    expect((await staticDemoRequest<any>(`/api/v1/programs/${program.program.id}/form-templates`)).items).toEqual([]);
    expect((await staticDemoRequest<any>(`/api/v1/programs/${program.program.id}/monitoring-checks`)).items).toEqual([]);
    program = await staticDemoRequest<any>(`/api/v1/programs/${program.program.id}/requirements`, { method: "POST", body: JSON.stringify({ expected_version: program.program.version, code: "NEW-01", title: "Maintain current evidence", statement: "The bank must maintain current evidence.", source_anchor: "Approved policy section 1", modality: "MUST", effective_from: "2026-08-26T00:00:00Z" }) });
    expect(program.requirements).toHaveLength(1);
    const form = await staticDemoRequest<any>(`/api/v1/programs/${program.program.id}/form-templates`, { method: "POST", body: JSON.stringify({ code: "NEW-FORM", name: "New evidence form", purpose: "Collect current evidence.", fields: [{ id: "current", label: "Current?", type: "boolean", required: true }] }) });
    expect(form.program_id).toBe("program-created");

    const { staticDemoRequest: matterRequest } = await demo();
    await matterRequest("/api/v1/demo/login", { method: "POST", body: JSON.stringify({ username: "dpo@demo.clearsight" }) });
    let matter = await matterRequest<any>("/api/v1/matters", { method: "POST", body: JSON.stringify({ type: "CONTROL_GAP", priority: 3, title: "Resolve a new evidence gap", summary: "Assign and verify the evidence update.", scope: {}, known_facts: {}, missing_facts: ["Evidence owner"], contradictions: [], due_at: "2026-09-01T00:00:00Z" }) });
    expect((await matterRequest<any>(`/api/v1/matters/${matter.matter.id}/operations?tenant_id=bank-demo`)).matter_id).toBe("matter-created");
    matter = await matterRequest<any>(`/api/v1/matters/${matter.matter.id}/context-changes`, { method: "POST", body: JSON.stringify({ expected_version: matter.matter.version, kind: "RESOLVE_MISSING", key: "evidence_owner", label: "Evidence owner", value: "Privacy Control Owner", rationale: "The owner accepted the work." }) });
    matter = await matterRequest<any>(`/api/v1/matters/${matter.matter.id}/actions`, { method: "POST", body: JSON.stringify({ expected_version: matter.matter.version, title: "Update evidence", description: "Retain current evidence.", owner_principal_id: "role-privacy-control" }) });
    const action = matter.actions[0];
    matter = await matterRequest<any>(`/api/v1/matters/${matter.matter.id}/actions/${action.id}/transition`, { method: "POST", body: JSON.stringify({ expected_version: matter.matter.version, to: "IN_PROGRESS", rationale: "Evidence work started." }) });
    matter = await matterRequest<any>(`/api/v1/matters/${matter.matter.id}/verification-contracts`, { method: "POST", body: JSON.stringify({ expected_version: matter.matter.version, action_id: action.id, expected_outcome: "Current evidence is retained.", observation_period_minutes: 0, reviewer_candidate_id: "role-dpco", failure_response: "BLOCK_CLOSE" }) });
    expect(matter.verification_contracts[0]).toMatchObject({ action_id: action.id, status: "ACTIVE" });
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

  it("supports exact vendor identity and approved-logo state without a remote image URL", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    type DemoVendor = { id: string; legal_name: string; website_domain?: string; version: number };
    const page = await staticDemoRequest<{ items: Array<{ vendor: DemoVendor }> }>("/api/v1/vendors?limit=50");
    const vendor = page.items[0]!.vendor;
    const loaded = await staticDemoRequest<{ vendor: DemoVendor; brand: { state: string; asset_token?: string; version: number } }>(`/api/v1/vendor-identities/${vendor.id}`);
    expect(loaded.vendor.website_domain).toBe("acme.example");
    expect(loaded.brand).toMatchObject({ state: "PENDING", version: 0 });
    expect(loaded.brand.asset_token).toBeUndefined();

    const updated = await staticDemoRequest<typeof loaded>(`/api/v1/vendor-identities/${vendor.id}`, { method: "PUT", body: JSON.stringify({ expected_version: vendor.version, legal_name: "Acme Payments Limited", website_domain: "payments.acme.example" }) });
    expect(updated.vendor).toMatchObject({ legal_name: "Acme Payments Limited", website_domain: "payments.acme.example", version: vendor.version + 1 });
    const logo = new File([new Uint8Array([137, 80, 78, 71])], "approved.png", { type: "image/png" });
    await expect(staticDemoRequest(`/api/v1/vendor-identities/${vendor.id}/brand`, { method: "PUT", headers: { "If-Match": `"${updated.brand.version}"`, "Content-Type": "image/png" }, body: logo })).rejects.toMatchObject({ status: 400, code: "idempotency_key_required" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
    const approved = await staticDemoRequest<typeof loaded>(`/api/v1/vendor-identities/${vendor.id}/brand`, { method: "PUT", headers: { "If-Match": `"${updated.brand.version}"`, "Content-Type": "image/png", "Idempotency-Key": "vendor-brand-static-upload" }, body: logo });
    expect(approved.brand).toMatchObject({ state: "APPROVED_LOGO", version: updated.brand.version + 1 });
    expect(approved.brand.asset_token).toBeUndefined();
    const restored = await staticDemoRequest<typeof loaded>(`/api/v1/vendor-identities/${vendor.id}/brand`, { method: "DELETE", headers: { "If-Match": `"${approved.brand.version}"`, "Idempotency-Key": "vendor-brand-static-remove" } });
    expect(restored.brand).toMatchObject({ state: "PENDING", version: approved.brand.version + 1 });
  });

  it("provides deterministic vendor-brand presentation fixtures for rendered review", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    type Presentation = { vendor: { id: string; website_domain?: string }; brand: { state: string; source?: string; asset_token?: string; version: number } };
    const expected = [
      ["vendor-brand-website", { state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "website-1" }],
      ["vendor-brand-approved", { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-1" }],
      ["vendor-brand-pending", { state: "PENDING" }],
      ["vendor-brand-unavailable", { state: "UNAVAILABLE" }],
      ["vendor-brand-broken", { state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "broken-1" }],
    ] as const;

    for (const [fixture, brand] of expected) {
      window.history.replaceState(null, "", `/?fixture=${fixture}`);
      const page = await staticDemoRequest<{ items: Array<{ vendor: { id: string }; brand: Presentation["brand"] }> }>("/api/v1/vendors?limit=50");
      expect(page.items[0]?.brand).toMatchObject(brand);
      const identity = await staticDemoRequest<Presentation>(`/api/v1/vendor-identities/${page.items[0]!.vendor.id}`);
      expect(identity.brand).toMatchObject(brand);
    }

    window.history.replaceState(null, "", "/?fixture=vendor-identity-brand-errors");
    const current = await staticDemoRequest<Presentation>("/api/v1/vendor-identities/vendor-acme-processing");
    await expect(staticDemoRequest("/api/v1/vendor-identities/vendor-acme-processing", { method: "PUT", body: JSON.stringify({ expected_version: 1, legal_name: "Acme Processing Limited", website_domain: "acme.example" }) })).rejects.toMatchObject({ status: 409, code: "vendor_identity_changed" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
    const logo = new File([new Uint8Array([137, 80, 78, 71])], "approved.png", { type: "image/png" });
    await expect(staticDemoRequest("/api/v1/vendor-identities/vendor-acme-processing/brand", { method: "PUT", headers: { "If-Match": `"${current.brand.version}"`, "Content-Type": "image/png", "Idempotency-Key": "vendor-brand-static-forbidden" }, body: logo })).rejects.toMatchObject({ status: 403, code: "permission_denied" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
  });

  it("applies the production website-hostname rules to vendor identity updates", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    const current = await staticDemoRequest<{ vendor: { id: string; legal_name: string; version: number } }>("/api/v1/vendor-identities/vendor-acme-processing");
    const canonical = await staticDemoRequest<{ vendor: { website_domain?: string; version: number } }>("/api/v1/vendor-identities/vendor-acme-processing", { method: "PUT", body: JSON.stringify({ expected_version: current.vendor.version, legal_name: current.vendor.legal_name, website_domain: "BÜCHER.Example" }) });
    expect(canonical.vendor.website_domain).toBe("xn--bcher-kva.example");
    for (const value of ["vendor.example.", "vendor.example\\path", "%76endor.example", "127.1"]) {
      await expect(staticDemoRequest("/api/v1/vendor-identities/vendor-acme-processing", { method: "PUT", body: JSON.stringify({ expected_version: canonical.vendor.version, legal_name: current.vendor.legal_name, website_domain: value }) })).rejects.toMatchObject({ status: 422, code: "vendor_identity_invalid" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
    }
  });

  it("restores a website icon or monogram state when an approved fixture logo is removed", async () => {
    const { staticDemoRequest } = await demo();
    type Presentation = { vendor: { id: string }; brand: { state: string; source?: string; asset_token?: string; version: number } };
    for (const [fixture, expected] of [
      ["vendor-brand-approved", { state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "website-2" }],
      ["vendor-brand-approved-no-discovered", { state: "UNAVAILABLE" }],
    ] as const) {
      window.history.replaceState(null, "", `/?fixture=${fixture}`);
      const current = await staticDemoRequest<Presentation>("/api/v1/vendor-identities/vendor-acme-processing");
      const restored = await staticDemoRequest<Presentation>("/api/v1/vendor-identities/vendor-acme-processing/brand", { method: "DELETE", headers: { "If-Match": `"${current.brand.version}"`, "Idempotency-Key": `vendor-brand-remove-${fixture}` } });
      expect(restored.brand).toMatchObject(expected);
    }
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

  it("supports vendor work linked to exact Program and Matter records", async () => {
    window.history.replaceState(null, "", "/?fixture=vendor-work-empty");
    const { staticDemoRequest } = await demo();

    const programLinks = await staticDemoRequest<{ items: Array<{ id: string; target_type: string; target_id: string; relationship_id: string }> }>("/api/v1/vendor-links?target_type=PROGRAM&target_id=program-ndpa&limit=50");
    const matterLinks = await staticDemoRequest<{ items: Array<{ id: string; target_type: string; target_id: string; relationship_id: string }> }>("/api/v1/vendor-links?target_type=MATTER&target_id=matter-gaid-change&limit=50");
    expect(programLinks.items[0]).toMatchObject({ target_type: "PROGRAM", target_id: "program-ndpa", relationship_id: "vendor-relationship-payments" });
    expect(matterLinks.items[0]).toMatchObject({ target_type: "MATTER", target_id: "matter-gaid-change", relationship_id: "vendor-relationship-payments" });

    const prepared = await staticDemoRequest<{ id: string; target_type: string; target_id: string; presentation: string; state: string; version: number }>("/api/v1/vendors/vendor-relationship-payments/work/prepare", {
      method: "POST",
      body: JSON.stringify({ relationship_link_id: programLinks.items[0]?.id, purpose: "Confirm payment-service controls", instructions: "Complete the control questions and provide the current assurance report.", form_template_id: "form-vendor-due-diligence", form_template_version: 3, presentation: "WIZARD", vendor_audience: "security@acme.example", due_at: "2026-09-30T23:59:59Z" }),
    });
    expect(prepared).toMatchObject({ target_type: "PROGRAM", target_id: "program-ndpa", presentation: "WIZARD", state: "PREPARING", version: 1 });

    const sent = await staticDemoRequest<{ work: { state: string; delivery_state: string; version: number }; state: string }>(`/api/v1/vendors/vendor-relationship-payments/work/${prepared.id}/send`, { method: "POST", body: JSON.stringify({ expected_version: 1, vendor_audience: "security@acme.example", invitation_ttl_minutes: 1440 }) });
    expect(sent).toMatchObject({ work: { state: "AWAITING_VENDOR", delivery_state: "DELIVERED", version: 2 }, state: "DELIVERED" });
  });

  it("renders exact vendor work responses and deterministic review decisions", async () => {
    window.history.replaceState(null, "", "/?fixture=vendor-work-submitted");
    const { staticDemoRequest } = await demo();
    const page = await staticDemoRequest<{ items: Array<{ id: string; relationship_id: string; current_request_id: string; state: string; version: number }> }>("/api/v1/vendor-work?target_type=MATTER&target_id=matter-gaid-change&limit=20");
    const work = page.items[0]!;
    const response = await staticDemoRequest<{ work: { id: string }; request: { request_id: string; presentation: { default_mode: string } }; response: { request_id: string }; answers: Array<{ visibility: string; value?: unknown }>; documents: Array<{ artifact_status: string }> }>(`/api/v1/vendors/${work.relationship_id}/work/${work.id}/response`);

    expect(work).toMatchObject({ state: "RESPONSE_RECEIVED", current_request_id: "vendor-work-capture-1", version: 4 });
    expect(response.work.id).toBe(work.id);
    expect(response.request).toMatchObject({ request_id: work.current_request_id, presentation: { default_mode: "WIZARD" } });
    expect(response.response.request_id).toBe(work.current_request_id);
    expect(response.answers.some((answer) => answer.visibility === "CONDITIONALLY_OMITTED")).toBe(true);
    expect(response.answers.some((answer) => !answer.value)).toBe(true);
    expect(response.documents.map((document) => document.artifact_status)).toEqual(["AVAILABLE", "QUARANTINED"]);

    const reviewing = await staticDemoRequest<{ state: string; version: number }>(`/api/v1/vendors/${work.relationship_id}/work/${work.id}/review/start`, { method: "POST", body: JSON.stringify({ expected_version: work.version }) });
    expect(reviewing).toMatchObject({ state: "UNDER_REVIEW", version: 5 });
    const accepted = await staticDemoRequest<{ state: string; version: number; review_rationale: string }>(`/api/v1/vendors/${work.relationship_id}/work/${work.id}/accept`, { method: "POST", body: JSON.stringify({ expected_version: 5, rationale: "The response and current assurance report address this request." }) });
    expect(accepted).toMatchObject({ state: "ACCEPTED", version: 6, review_rationale: "The response and current assurance report address this request." });
  });

  it("recovers vendor work after partial invitation delivery", async () => {
    window.history.replaceState(null, "", "/?fixture=vendor-work-partial-delivery");
    const { staticDemoRequest } = await demo();
    const page = await staticDemoRequest<{ items: Array<{ id: string; relationship_id: string; delivery_state: string; version: number }> }>("/api/v1/vendor-work?target_type=PROGRAM&target_id=program-ndpa&limit=20");
    const work = page.items[0]!;
    expect(work.delivery_state).toBe("RETRY_REQUIRED");

    const retried = await staticDemoRequest<{ work: { delivery_state: string; version: number }; state: string }>(`/api/v1/vendors/${work.relationship_id}/work/${work.id}/retry`, { method: "POST", body: JSON.stringify({ expected_version: work.version, vendor_audience: "security@acme.example", invitation_ttl_minutes: 1440 }) });
    expect(retried).toMatchObject({ work: { delivery_state: "DELIVERED", version: work.version + 1 }, state: "DELIVERED" });
  });
});
