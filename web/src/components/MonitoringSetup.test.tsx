import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createFormMonitoringCheck, createFormTemplate, createMonitoringLinkedIssue, loadCollectionSummaries, loadFormTemplates, loadMonitoringChecks, loadMonitoringResults, startFormCollection } from "../monitoringApi";
import { createProgram, loadProgramSetupCandidates } from "../continuityCommands";
import type { ProgramAggregate } from "../types";
import { FormBuilder } from "./FormBuilder";
import { MonitoringSetup } from "./MonitoringSetup";
import { ProgramSetupWorkspace } from "./ProgramSetupWorkspace";
import { createRESTBinding, prepareRESTSource } from "../sourceConfigApi";
import { DataSourceBuilder } from "./DataSourceBuilder";
import { loadProgramOperations } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";

vi.mock("../monitoringApi", () => ({
  createFormTemplate: vi.fn(),
  createFormMonitoringCheck: vi.fn(),
  loadCollectionSummaries: vi.fn(),
  loadFormTemplates: vi.fn(),
  loadMonitoringChecks: vi.fn(),
  loadMonitoringResults: vi.fn(),
  evaluateMonitoringSource: vi.fn(),
  createMonitoringLinkedIssue: vi.fn(),
  startFormCollection: vi.fn(),
  transitionFormTemplate: vi.fn(),
  transitionMonitoringCheck: vi.fn(),
}));

vi.mock("../continuityCommands", () => ({ createProgram: vi.fn(), addProgramRequirement: vi.fn(), loadProgramSetupCandidates: vi.fn() }));
vi.mock("../sourceConfigApi", () => ({ prepareRESTSource: vi.fn(), createRESTBinding: vi.fn() }));
vi.mock("../programOperationsApi", () => ({ loadProgramOperations: vi.fn() }));

const program: ProgramAggregate = {
  state_label: "Setup in progress",
  program: { id: "program-1", tenant_id: "bank-1", code: "MOBILE", name: "Mobile banking", type: "CHANNEL", status: "DRAFT", owning_function: "Digital Banking", owner_principal_id: "owner-1", scope: {}, effective_from: "2026-08-17T00:00:00Z", created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z", version: 1 },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
};

const ownerOperations: ProgramOperation[] = [
  { command: "program.monitoring.form.define", label: "Create a collection form", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current responsibility." },
  { command: "program.monitoring.define", label: "Add a monitoring check", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current responsibility." },
];

const allMonitoringOperations: ProgramOperation[] = [
  ...ownerOperations,
  { command: "program.monitoring.collect", subresource_id: "form-1", label: "Collect responses", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current responsibility." },
  { command: "program.monitoring.transition", subresource_id: "check-draft", label: "Change draft check status", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current responsibility.", allowed_targets: ["PENDING_APPROVAL"] },
  { command: "program.monitoring.transition", subresource_id: "check-pending", label: "Change pending check status", responsibility: "REVIEWER", can_act: true, reason: "You hold the current responsibility.", allowed_targets: ["ACTIVE", "REJECTED"] },
  { command: "program.monitoring.evaluate", subresource_id: "check-source", label: "Check source now", responsibility: "PERFORMER", can_act: true, reason: "You hold the current responsibility." },
];

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadFormTemplates).mockResolvedValue([]);
  vi.mocked(loadMonitoringChecks).mockResolvedValue([]);
  vi.mocked(loadMonitoringResults).mockResolvedValue([]);
  vi.mocked(loadCollectionSummaries).mockResolvedValue([]);
  vi.mocked(loadProgramSetupCandidates).mockResolvedValue({
    owner_candidates: [{ id: "owner-1", display_name: "Data Protection Officer", kind: "PERSON", role: "DPO" }],
    approval_authority_candidates: [{ id: "cro-1", display_name: "Chief Risk Officer", kind: "PERSON", role: "CRO" }],
    has_more: false, generated_at: "2026-08-26T00:00:00Z",
  });
});

describe("monitoring setup", () => {
  it("uses the exact form-definition operation only for the collection form builder", async () => {
    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={[
      { command: "program.monitoring.form.define", label: "Create a collection form", responsibility: "ACCOUNTABLE_OWNER", can_act: false, reason: "Assigned to the Program owner." },
      { command: "program.monitoring.define", label: "Add a monitoring check", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current responsibility." },
    ]}/>);

    fireEvent.click(screen.getByRole("button", { name: "Add monitoring check" }));
    expect((screen.getByRole("button", { name: "Collection form" }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: "Connected data" }) as HTMLButtonElement).disabled).toBe(false);
  });

  it("builds a five-question password reset form with explicit risk scoring", async () => {
    vi.mocked(createFormTemplate).mockResolvedValue({
      id: "form-1", tenant_id: "bank-1", code: "PASSWORD-RESET-REVIEW", name: "Password reset security review", purpose: "Confirm that password reset safeguards operated during the reporting period.",
      fields: [], status: "DRAFT", is_current: false, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    });
    const onSaved = vi.fn();
    render(<FormBuilder programID="program-1" onSaved={onSaved} onCancel={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Use password reset example" }));
    expect(screen.getAllByLabelText("Question")).toHaveLength(5);
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledWith("program-1", expect.objectContaining({
      code: "PASSWORD-RESET-REVIEW",
      fields: expect.arrayContaining([expect.objectContaining({
        type: "yes_no", options: ["Yes", "No"], scoring: expect.objectContaining({ answer_scores: { Yes: 0, No: 100 }, critical_answers: ["No"] }),
      })]),
    })));
    expect(onSaved).toHaveBeenCalled();
  });

  it("keeps Program monitoring in the page and offers the two supported input choices", async () => {
    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={ownerOperations}/>);
    expect(await screen.findByRole("heading", { name: "Monitoring" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Add monitoring check" }));
    expect(screen.getByRole("button", { name: "Collection form" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Connected data" })).toBeTruthy();
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("does not collect a form until its exact Program check is active", async () => {
    vi.mocked(loadFormTemplates).mockResolvedValue([{
      id: "form-1", tenant_id: "bank-1", code: "RESET", name: "Password reset review", purpose: "Confirm safeguards", fields: [],
      status: "ACTIVE", is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);

    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={ownerOperations}/>);

    expect(await screen.findByRole("button", { name: "Set collection schedule" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Collect responses" })).toBeNull();
  });

  it("adds the recommended expiry and reminder policy to a Program collection", async () => {
    const form = {
      id: "form-1", tenant_id: "bank-1", code: "RESET", name: "Password reset review", purpose: "Confirm safeguards", fields: [],
      status: "ACTIVE" as const, is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    };
    vi.mocked(loadFormTemplates).mockResolvedValue([form]);
    vi.mocked(createFormMonitoringCheck).mockResolvedValue({
      id: "check-1", tenant_id: "bank-1", program_id: "program-1", code: "RESET-CHECK", name: form.name, claim: form.purpose, input_kind: "FORM",
      form_template_id: form.id, form_template_version: form.version, collection_policy: { validity_months: 12, renewal_window_days: 30, reminder_count: 3 },
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 10080, minimum_coverage: 1, failure_action: "REVIEW",
      status: "DRAFT", is_current: false, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    });
    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={ownerOperations}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Set collection schedule" }));
    fireEvent.click(screen.getByRole("button", { name: "Add collection to Program" }));

    await waitFor(() => expect(createFormMonitoringCheck).toHaveBeenCalledWith("program-1", form, { validity_months: 12, renewal_window_days: 30, reminder_count: 3 }));
    expect(screen.getAllByRole("heading", { name: form.name })).toHaveLength(1);
  });

  it("shows the latest risk and coverage for each monitoring check", async () => {
    vi.mocked(loadFormTemplates).mockResolvedValue([{
      id: "form-1", tenant_id: "bank-1", code: "RESET", name: "Password reset review", purpose: "Confirm safeguards",
      fields: [{ id: "identity", label: "Was identity verified?", type: "single_select", required: true, options: ["Yes", "No"] }],
      status: "ACTIVE", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);
    vi.mocked(loadMonitoringChecks).mockResolvedValue([{
      id: "check-1", tenant_id: "bank-1", program_id: "program-1", code: "RESET", name: "Password reset review", claim: "Safeguards operated", input_kind: "FORM",
      form_template_id: "form-1", form_template_version: 1, thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1,
      failure_action: "RECOMMEND_MATTER", status: "ACTIVE", is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);
    vi.mocked(loadMonitoringResults).mockResolvedValue([{
      id: "result-1", monitoring_check_id: "check-1", monitoring_check_version: 2, evaluated_at: "2026-08-17T12:00:00Z",
      evaluation: { score: 100, band: "CRITICAL", coverage: 1, rule_results: [{ field_id: "identity", outcome: "FAIL", points: 100, critical: true, reason: "Answer evaluated against the active form rule." }] },
    }]);

    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={[]}/>);

    expect(await screen.findByText("100% risk")).toBeTruthy();
    expect(screen.getByText("100% coverage")).toBeTruthy();
    expect(screen.getByText("Critical")).toBeTruthy();
    fireEvent.click(screen.getByText("Review result"));
    expect(screen.getByText("Was identity verified?")).toBeTruthy();
  });

  it("creates a channel Program from business fields without technical identifiers", async () => {
    vi.mocked(createProgram).mockResolvedValue(program);
    const onCreated = vi.fn();
    render(<ProgramSetupWorkspace actorPrincipalID="owner-1" canConfigureSources onCreated={onCreated} onClose={vi.fn()}/>);
    expect((await screen.findByLabelText("Accountable owner") as HTMLSelectElement).value).toBe("owner-1");
    expect((screen.getByLabelText("Approval authority") as HTMLSelectElement).value).toBe("cro-1");
    fireEvent.change(screen.getByLabelText("Program name"), { target: { value: "Mobile banking" } });
    fireEvent.change(screen.getByLabelText("Code"), { target: { value: "MOBILE" } });
    fireEvent.change(screen.getByLabelText("Owning function"), { target: { value: "Digital Banking" } });
    fireEvent.change(screen.getByLabelText("Scope"), { target: { value: "Retail mobile banking channel" } });
    fireEvent.click(screen.getByRole("button", { name: "Create Program" }));
    await waitFor(() => expect(createProgram).toHaveBeenCalledWith(expect.objectContaining({ name: "Mobile banking", type: "CHANNEL", scopeDescription: "Retail mobile banking channel", ownerCandidateID: "owner-1", approvalAuthorityCandidateID: "cro-1" })));
    expect(onCreated).toHaveBeenCalledWith(program);
  });

  it("tests an HTTPS status endpoint and lets the user select an observed field", async () => {
    const prepared = {
      source: { id: "source-1", tenant_id: "bank-1", code: "FACE-SDK", name: "Live face verification", type: "SYSTEM", authority_class: "INTERNAL_CONTROL", expected_freshness_minutes: 60, health: "UNKNOWN", status: "ACTIVE", version: 1 },
      connection: { connection_id: "connection-1", source_id: "source-1", version: 1, code: "FACE-SDK-REST", name: "Endpoint", status: "DRAFT" },
      view: { view_id: "view-1", connection_id: "connection-1", connection_version: 1, source_id: "source-1", version: 2, code: "FACE-SDK-STATUS", name: "Status", native_schema: [{ name: "sdk_present", native_type: "json:boolean", nullable: false }] },
    };
    vi.mocked(prepareRESTSource).mockResolvedValue(prepared);
    vi.mocked(createRESTBinding).mockResolvedValue({ binding_id: "binding-1", view_id: "view-1", view_version: 3, source_id: "source-1", version: 1, code: "FACE-SDK-MONITOR", name: "Monitoring", status: "DRAFT", selected_fields: ["sdk_present"] });
    const onSaved = vi.fn();
    render(<DataSourceBuilder onSaved={onSaved} onCancel={vi.fn()}/>);
    fireEvent.change(screen.getByLabelText("Source name"), { target: { value: "Live face verification" } });
    fireEvent.change(screen.getByLabelText("Code"), { target: { value: "FACE-SDK" } });
    fireEvent.change(screen.getByLabelText("Status endpoint"), { target: { value: "https://status.example/sdk" } });
    fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
    expect(await screen.findByLabelText("Status field")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Monitoring statement"), { target: { value: "The live face verification SDK is enabled on mobile banking." } });
    fireEvent.change(screen.getByLabelText("Expected value"), { target: { value: "true" } });
    fireEvent.click(screen.getByRole("button", { name: "Use this source" }));
    await waitFor(() => expect(createRESTBinding).toHaveBeenCalledWith(prepared, "sdk_present"));
    expect(onSaved).toHaveBeenCalledWith(expect.objectContaining({ binding_id: "binding-1" }), expect.objectContaining({ field: "sdk_present", expected: "true", claim: "The live face verification SDK is enabled on mobile banking." }));
  });

  it("does not start source configuration without configuration access", async () => {
    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources={false} operations={ownerOperations}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Add monitoring check" }));
    expect(screen.getByRole("button", { name: "Connected data" }).hasAttribute("disabled")).toBe(true);
    expect(screen.getByText("A GRC administrator can connect a new source.")).toBeTruthy();
  });

  it("creates the linked issue for an eligible latest result and opens the returned issue", async () => {
    vi.mocked(loadMonitoringChecks).mockResolvedValue([{
      id: "check-1", tenant_id: "bank-1", program_id: "program-1", code: "ACCESS", name: "Access status", claim: "Access controls remain current", input_kind: "SOURCE",
      binding_id: "binding-1", binding_version: 1, thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1,
      reviewer_principal_id: "reviewer-1", failure_action: "RECOMMEND_MATTER", status: "ACTIVE", is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);
    vi.mocked(loadMonitoringResults).mockResolvedValue([{
      id: "result-1", monitoring_check_id: "check-1", monitoring_check_version: 2, evaluated_at: "2026-08-17T12:00:00Z",
      evaluation: { score: 80, band: "CRITICAL", coverage: 1, rule_results: [{ rule_id: "status-rule", field_id: "status", outcome: "FAIL", points: 80, reason: "The expected status was not returned." }] },
    }]);
    vi.mocked(createMonitoringLinkedIssue).mockResolvedValue({ matter: { id: "matter-1", reference: "MAT-0001" }, created: true });
    const onOpenMatter = vi.fn();
    const operation: ProgramOperation = {
      command: "program.monitoring.issue.create", subresource_id: "check-1", label: "Create linked issue for Access status", responsibility: "REVIEWER",
      can_act: true, assigned_to: { id: "reviewer-1", display_name: "Control assurance reviewer", kind: "PERSON", role: "Reviewer" }, reason: "You hold the current responsibility.",
    };

    render(<MonitoringSetup aggregate={program} actorPrincipalID="reviewer-1" canConfigureSources={false} operations={[operation]} onOpenMatter={onOpenMatter}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Create linked issue" }));

    await waitFor(() => expect(createMonitoringLinkedIssue).toHaveBeenCalledWith("result-1"));
    expect(onOpenMatter).toHaveBeenCalledWith("matter-1");
    expect(screen.getByText("Linked issue MAT-0001 is ready for Control Assurance review.")).toBeTruthy();
  });

  it("shows why the eligible linked-issue action is disabled and hides it for a low result", async () => {
    const check = {
      id: "check-1", tenant_id: "bank-1", program_id: "program-1", code: "ACCESS", name: "Access status", claim: "Access controls remain current", input_kind: "SOURCE" as const,
      binding_id: "binding-1", binding_version: 1, thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1,
      reviewer_principal_id: "reviewer-1", failure_action: "RECOMMEND_MATTER" as const, status: "ACTIVE" as const, is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    };
    vi.mocked(loadMonitoringChecks).mockResolvedValue([check]);
    vi.mocked(loadMonitoringResults).mockResolvedValue([{
      id: "result-1", monitoring_check_id: "check-1", monitoring_check_version: 2, evaluated_at: "2026-08-17T12:00:00Z", evaluation: { score: 80, band: "HIGH", coverage: 1 },
    }]);
    const operation: ProgramOperation = {
      command: "program.monitoring.issue.create", subresource_id: "check-1", label: "Create linked issue for Access status", responsibility: "REVIEWER", can_act: false,
      assigned_to: { id: "reviewer-1", display_name: "Control assurance reviewer", kind: "PERSON", role: "Reviewer" }, reason: "Assigned to Control assurance reviewer for the current Program state.",
    };
    const view = render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources={false} operations={[operation]}/>);
    const disabled = await screen.findByRole("button", { name: "Create linked issue" });
    expect(disabled.hasAttribute("disabled")).toBe(true);
    expect(screen.getByText(operation.reason)).toBeTruthy();

    view.unmount();
    for (const [band, score] of [["LOW", 0], ["MODERATE", 30], ["NOT_ASSESSED", undefined]] as const) {
      vi.mocked(loadMonitoringResults).mockResolvedValue([{
        id: `result-${band}`, monitoring_check_id: "check-1", monitoring_check_version: 2, evaluated_at: "2026-08-17T13:00:00Z", evaluation: { score, band, coverage: 1 },
      }]);
      const candidate = render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources={false} operations={[operation]}/>);
      await screen.findByLabelText("Latest result for Access status");
      expect(screen.queryByRole("button", { name: "Create linked issue" })).toBeNull();
      candidate.unmount();
    }
  });

  it("starts a permitted collection without sending browser-selected respondent or reviewer identities", async () => {
    vi.mocked(loadFormTemplates).mockResolvedValue([{
      id: "form-1", tenant_id: "bank-1", program_id: "program-1", legal_entity_id: "entity-1", code: "RESET", name: "Password reset review", purpose: "Confirm safeguards", fields: [],
      status: "ACTIVE", is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);
    vi.mocked(loadMonitoringChecks).mockResolvedValue([{
      id: "check-1", tenant_id: "bank-1", program_id: "program-1", code: "RESET", name: "Password reset review", claim: "Safeguards operated", input_kind: "FORM",
      form_template_id: "form-1", form_template_version: 2, thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1,
      failure_action: "REVIEW", status: "ACTIVE", is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);
    vi.mocked(startFormCollection).mockResolvedValue({ id: "request-1" } as never);

    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={allMonitoringOperations}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Collect responses" }));
    expect(screen.getByLabelText("Period starts").getAttribute("type")).toBe("date");
    expect(screen.getByLabelText("Period ends").getAttribute("type")).toBe("date");
    expect(screen.getByLabelText("Due").getAttribute("type")).toBe("datetime-local");
    fireEvent.submit(screen.getByRole("button", { name: "Create request" }).closest("form")!);

    await waitFor(() => expect(startFormCollection).toHaveBeenCalledWith(expect.objectContaining({ id: "form-1" }), expect.not.objectContaining({
      respondentPrincipalID: expect.anything(), reviewerPrincipalID: expect.anything(),
    })));
  });

  it("keeps monitoring records readable but hides every mutation while Program responsibilities are unavailable", async () => {
    vi.mocked(loadFormTemplates).mockResolvedValue([{
      id: "form-draft", tenant_id: "bank-1", code: "DRAFT", name: "Draft owner review", purpose: "Confirm ownership", fields: [], status: "DRAFT", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }, {
      id: "form-pending", tenant_id: "bank-1", code: "PENDING", name: "Pending owner review", purpose: "Confirm ownership", fields: [], status: "PENDING_APPROVAL", submitted_by: "maker-1", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }, {
      id: "form-active", tenant_id: "bank-1", code: "ACTIVE", name: "Active owner review", purpose: "Confirm ownership", fields: [], status: "ACTIVE", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }, {
      id: "form-linked", tenant_id: "bank-1", code: "LINKED", name: "Linked active review", purpose: "Confirm ownership", fields: [], status: "ACTIVE", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);
    vi.mocked(loadMonitoringChecks).mockResolvedValue([{
      id: "check-source", tenant_id: "bank-1", program_id: "program-1", code: "SOURCE", name: "Live source check", claim: "The connected source remains healthy", input_kind: "SOURCE", binding_id: "binding-1", binding_version: 1,
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1, failure_action: "RECOMMEND_MATTER", status: "ACTIVE", is_current: true, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }, {
      id: "check-draft", tenant_id: "bank-1", program_id: "program-1", code: "CHECK-DRAFT", name: "Draft monitoring check", claim: "The draft check remains visible", input_kind: "FORM", form_template_id: "form-draft", form_template_version: 1,
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1, failure_action: "RECOMMEND_MATTER", status: "DRAFT", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }, {
      id: "check-pending", tenant_id: "bank-1", program_id: "program-1", code: "CHECK-PENDING", name: "Pending monitoring check", claim: "The pending check remains visible", input_kind: "FORM", form_template_id: "form-pending", form_template_version: 1,
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1, failure_action: "RECOMMEND_MATTER", status: "PENDING_APPROVAL", submitted_by: "maker-1", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }, {
      id: "check-form-active", tenant_id: "bank-1", program_id: "program-1", code: "CHECK-ACTIVE", name: "Active collection check", claim: "The collection remains visible", input_kind: "FORM", form_template_id: "form-linked", form_template_version: 1,
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1, failure_action: "RECOMMEND_MATTER", status: "ACTIVE", is_current: true, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);

    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={[]}/>);

    expect(await screen.findByRole("heading", { name: "Active owner review" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Live source check" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Draft monitoring check" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Pending monitoring check" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Active collection check" })).toBeTruthy();
    expect(screen.getByText("Monitoring changes are disabled until current Program responsibilities are available. Existing checks and results remain available.")).toBeTruthy();
    for (const name of ["Add monitoring check", "Send for approval", "Approve form", "Set collection schedule", "Collect responses", "Approve check", "Check source now"]) {
      expect(screen.queryByRole("button", { name })).toBeNull();
    }
  });

  it("keeps monitoring retry available when mutations are disabled", async () => {
    vi.mocked(loadFormTemplates).mockRejectedValueOnce(new Error("forms unavailable")).mockResolvedValue([]);
    vi.mocked(loadMonitoringChecks).mockRejectedValueOnce(new Error("checks unavailable")).mockResolvedValue([]);
    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources operations={[]}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Try again" }));
    await waitFor(() => {
      expect(loadFormTemplates).toHaveBeenCalledTimes(2);
      expect(loadMonitoringChecks).toHaveBeenCalledTimes(2);
    });
    expect(screen.queryByRole("button", { name: "Add monitoring check" })).toBeNull();
  });

  it("shows each monitoring command only to its current assigned responsibility", async () => {
    vi.mocked(loadMonitoringChecks).mockResolvedValue([{
      id: "check-pending", tenant_id: "bank-1", program_id: "program-1", code: "PENDING", name: "Pending check", claim: "The check awaits review.", input_kind: "SOURCE", binding_id: "binding-1", binding_version: 1,
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1, owner_principal_id: "owner-1", reviewer_principal_id: "reviewer-1", failure_action: "REVIEW", status: "PENDING_APPROVAL", submitted_by: "owner-1", is_current: false, version: 2, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }, {
      id: "check-source", tenant_id: "bank-1", program_id: "program-1", code: "ACTIVE", name: "Active source", claim: "The source remains healthy.", input_kind: "SOURCE", binding_id: "binding-2", binding_version: 1,
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1, owner_principal_id: "owner-1", reviewer_principal_id: "reviewer-1", failure_action: "REVIEW", status: "ACTIVE", is_current: true, version: 3, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    }]);
    const reviewerOperations: ProgramOperation[] = [{
      command: "program.monitoring.transition", subresource_id: "check-pending", label: "Approve Pending check", responsibility: "REVIEWER", can_act: true,
      assigned_to: { id: "reviewer-1", display_name: "Controls reviewer", kind: "PERSON", role: "Reviewer" }, reason: "You hold the current responsibility.", allowed_targets: ["ACTIVE", "REJECTED"],
    }, {
      command: "program.monitoring.evaluate", subresource_id: "check-source", label: "Check Active source now", responsibility: "PERFORMER", can_act: false,
      assigned_to: { id: "performer-1", display_name: "Monitoring analyst", kind: "PERSON", role: "Analyst" }, reason: "Assigned to Monitoring analyst.",
    }];

    render(<MonitoringSetup aggregate={program} actorPrincipalID="reviewer-1" canConfigureSources operations={reviewerOperations}/>);

    expect(await screen.findByRole("button", { name: "Approve check" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add monitoring check" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Check source now" })).toBeNull();
    expect(screen.getByText("Assigned to Monitoring analyst.")).toBeTruthy();
  });
  vi.mocked(loadProgramOperations).mockResolvedValue({ program_id: "program-1", program_version: 1, authority_available: true, operations: ownerOperations, generated_at: "2026-08-26T00:00:00Z" });
});
