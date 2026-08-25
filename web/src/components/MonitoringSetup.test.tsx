import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createFormTemplate, loadFormTemplates, loadMonitoringChecks, loadMonitoringResults } from "../monitoringApi";
import { createProgram } from "../continuityCommands";
import type { ProgramAggregate } from "../types";
import { FormBuilder } from "./FormBuilder";
import { MonitoringSetup } from "./MonitoringSetup";
import { ProgramSetupWorkspace } from "./ProgramSetupWorkspace";
import { createRESTBinding, prepareRESTSource } from "../sourceConfigApi";
import { DataSourceBuilder } from "./DataSourceBuilder";

vi.mock("../monitoringApi", () => ({
  createFormTemplate: vi.fn(),
  createFormMonitoringCheck: vi.fn(),
  loadFormTemplates: vi.fn(),
  loadMonitoringChecks: vi.fn(),
  loadMonitoringResults: vi.fn(),
  evaluateMonitoringSource: vi.fn(),
  startFormCollection: vi.fn(),
  transitionFormTemplate: vi.fn(),
  transitionMonitoringCheck: vi.fn(),
}));

vi.mock("../continuityCommands", () => ({ createProgram: vi.fn(), addProgramRequirement: vi.fn() }));
vi.mock("../sourceConfigApi", () => ({ prepareRESTSource: vi.fn(), createRESTBinding: vi.fn() }));

const program: ProgramAggregate = {
  state_label: "Setup in progress",
  program: { id: "program-1", tenant_id: "bank-1", code: "MOBILE", name: "Mobile banking", type: "CHANNEL", status: "DRAFT", owning_function: "Digital Banking", owner_principal_id: "owner-1", scope: {}, effective_from: "2026-08-17T00:00:00Z", created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z", version: 1 },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadFormTemplates).mockResolvedValue([]);
  vi.mocked(loadMonitoringChecks).mockResolvedValue([]);
  vi.mocked(loadMonitoringResults).mockResolvedValue([]);
});

describe("monitoring setup", () => {
  it("builds a five-question password reset form with explicit risk scoring", async () => {
    vi.mocked(createFormTemplate).mockResolvedValue({
      id: "form-1", tenant_id: "bank-1", code: "PASSWORD-RESET-REVIEW", name: "Password reset security review", purpose: "Confirm that password reset safeguards operated during the reporting period.",
      fields: [], status: "DRAFT", is_current: false, version: 1, created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z",
    });
    const onSaved = vi.fn();
    render(<FormBuilder onSaved={onSaved} onCancel={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Use password reset review" }));
    expect(screen.getAllByLabelText("Question")).toHaveLength(5);
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledWith(expect.objectContaining({
      code: "PASSWORD-RESET-REVIEW",
      fields: expect.arrayContaining([expect.objectContaining({
        type: "yes_no", options: ["Yes", "No"], scoring: expect.objectContaining({ answer_scores: { Yes: 0, No: 100 }, critical_answers: ["No"] }),
      })]),
    })));
    expect(onSaved).toHaveBeenCalled();
  });

  it("keeps Program monitoring in the page and offers the two supported input choices", async () => {
    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources/>);
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

    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources/>);

    expect(await screen.findByRole("button", { name: "Create monitoring check" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Collect responses" })).toBeNull();
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

    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources/>);

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
    fireEvent.change(screen.getByLabelText("Program name"), { target: { value: "Mobile banking" } });
    fireEvent.change(screen.getByLabelText("Code"), { target: { value: "MOBILE" } });
    fireEvent.change(screen.getByLabelText("Owning function"), { target: { value: "Digital Banking" } });
    fireEvent.change(screen.getByLabelText("Scope"), { target: { value: "Retail mobile banking channel" } });
    fireEvent.click(screen.getByRole("button", { name: "Create Program" }));
    await waitFor(() => expect(createProgram).toHaveBeenCalledWith(expect.objectContaining({ name: "Mobile banking", type: "CHANNEL", scopeDescription: "Retail mobile banking channel" })));
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
    render(<MonitoringSetup aggregate={program} actorPrincipalID="owner-1" canConfigureSources={false}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Add monitoring check" }));
    expect(screen.getByRole("button", { name: "Connected data" }).hasAttribute("disabled")).toBe(true);
    expect(screen.getByText("A GRC administrator can connect a new source.")).toBeTruthy();
  });
});
