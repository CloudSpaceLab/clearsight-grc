import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadProgram, loadProgramSummaries } from "../api";
import {
  addProgramRequirement,
  assignProgram,
  determineProgramApplicability,
  loadProgramOperations,
  supersedeProgramRequirement,
  updateProgramDetails,
} from "../programOperationsApi";
import { loadProgramReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";
import { ProgramRecordWorkspace } from "./ProgramRecordWorkspace";
import { ProgramsWorkspace } from "./ProgramsWorkspace";

vi.mock("../api", () => ({ loadProgram: vi.fn(), loadProgramSummaries: vi.fn() }));
vi.mock("../programOperationsApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../programOperationsApi")>()),
  loadProgramOperations: vi.fn(),
  updateProgramDetails: vi.fn(),
  assignProgram: vi.fn(),
  addProgramRequirement: vi.fn(),
  supersedeProgramRequirement: vi.fn(),
  determineProgramApplicability: vi.fn(),
}));
vi.mock("../programReviewApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../programReviewApi")>()),
  loadProgramReviewDigest: vi.fn(),
}));

const aggregate: ProgramAggregate = {
  state_label: "Evidence incomplete",
  program: {
    id: "program-1", tenant_id: "bank", code: "NDPA", name: "Nigeria data protection", type: "PRIVACY",
    status: "DRAFT", owning_function: "Data Protection Office", owner_principal_id: "owner-1", jurisdiction: "Nigeria",
    scope: { business_lines: ["Retail"] }, effective_from: "2026-01-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-25T10:00:00Z", version: 4,
  },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [],
  evidence_contracts: [], evidence_assessments: [], triggers: [],
  current_state: {
    id: "state-1", overall_state: "EVIDENCE_INSUFFICIENT", dimensions: {},
    reasons: [{ code: "NO_EVIDENCE", summary: "Two applicable requirements do not have evidence checks." }],
    open_matter_count: 1, generated_at: "2026-08-25T09:50:00Z", program_version: 3, projection_version: 8,
  },
};

const operations = {
  program_id: "program-1", program_version: 4, authority_available: true, generated_at: "2026-08-25T10:00:00Z",
  operations: [
    { command: "program.details.update", label: "Edit Program details", responsibility: "ACCOUNTABLE_OWNER", can_act: false, assigned_to: { id: "owner-1", display_name: "Data Protection Officer", kind: "PERSON", role: "DPO" }, reason: "Assigned to Data Protection Officer." },
    { command: "program.transition", label: "Approve Program activation", responsibility: "AUTHORIZER", can_act: true, assigned_to: { id: "cro", display_name: "Chief Risk Officer", kind: "PERSON", role: "CRO" }, reason: "You hold the current responsibility.", allowed_targets: ["ACTIVE", "RETIRED"] },
  ],
};

const digest = {
  program_id: "program-1", state: "CURRENT", review_required: false,
  current_program_version: 4, current_projection_version: 8, current_overall: "EVIDENCE_INSUFFICIENT" as const,
  open_matter_count: 1, changes: [], changes_total: 0, changes_omitted: 0, history_truncated: false,
  current_exceptions: [], current_exceptions_total: 0, new_exceptions: [], new_exceptions_total: 0,
  resolved_exceptions: [], resolved_exceptions_total: 0,
};

describe("Program record workspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadProgram).mockResolvedValue(aggregate);
    vi.mocked(loadProgramOperations).mockResolvedValue(operations);
    vi.mocked(loadProgramReviewDigest).mockResolvedValue(digest);
    vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-25T10:00:00Z" });
	vi.mocked(updateProgramDetails).mockResolvedValue(aggregate);
	vi.mocked(assignProgram).mockResolvedValue(aggregate);
	vi.mocked(addProgramRequirement).mockResolvedValue(aggregate);
	vi.mocked(supersedeProgramRequirement).mockResolvedValue(aggregate);
	vi.mocked(determineProgramApplicability).mockResolvedValue(aggregate);
  });

  it("shows owner, calculated-state freshness, reasons and one dominant action", async () => {
    render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

    expect(await screen.findByRole("heading", { name: "Nigeria data protection" })).toBeTruthy();
    expect(screen.getAllByText("Data Protection Officer").length).toBeGreaterThan(0);
    expect(screen.getByText("Updating status")).toBeTruthy();
    expect(screen.getByText("Assessed version 3 · current version 4")).toBeTruthy();
    expect(screen.getByText("Two applicable requirements do not have evidence checks.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Approve Program activation" })).toBeTruthy();
    expect(screen.getAllByTestId("program-dominant-action")).toHaveLength(1);
    await waitFor(() => {
      expect(loadProgram).toHaveBeenCalledWith("program-1");
      expect(loadProgramOperations).toHaveBeenCalledWith("program-1");
      expect(loadProgramReviewDigest).toHaveBeenCalledWith("program-1");
    });
  });

  it("keeps portfolio search on the list route and uses the dedicated record on a target route", async () => {
    const list = render(<ProgramsWorkspace/>);
    expect(await screen.findByRole("search")).toBeTruthy();
    list.unmount();

    render(<ProgramsWorkspace targetID="program-1"/>);
    expect(await screen.findByRole("heading", { name: "Nigeria data protection" })).toBeTruthy();
    expect(screen.queryByRole("search")).toBeNull();
  });

  it("lets the current owner edit Program details and choose an eligible successor", async () => {
	vi.mocked(loadProgramOperations).mockResolvedValue({
	  ...operations,
	  operations: [
		{ ...operations.operations[0]!, can_act: true, reason: "You hold the current Program owner responsibility." },
		{ command: "program.assign", label: "Change Program owner", responsibility: "ACCOUNTABLE_OWNER", can_act: true, assigned_to: operations.operations[0]!.assigned_to, candidates: [operations.operations[0]!.assigned_to!, { id: "owner-2", display_name: "Deputy Data Protection Officer", kind: "PERSON", role: "Deputy DPO" }], reason: "You can assign an eligible owner." },
	  ],
	});
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	await screen.findByRole("heading", { name: "Scope and ownership" });
	const detailsPanel = document.getElementById("program-details-panel")!;
	fireEvent.click(within(detailsPanel).getByRole("button", { name: "Edit Program details" }));
	fireEvent.change(screen.getByLabelText("Owning function"), { target: { value: "Privacy and Data Protection" } });
	fireEvent.change(screen.getByLabelText("Business lines"), { target: { value: "Retail, Corporate" } });
	fireEvent.change(screen.getByLabelText("Reason for this change"), { target: { value: "Use the approved operating model." } });
	fireEvent.click(screen.getByRole("button", { name: "Save Program details" }));
	await waitFor(() => expect(updateProgramDetails).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ owningFunction: "Privacy and Data Protection", scope: expect.objectContaining({ business_lines: ["Retail", "Corporate"] }), rationale: "Use the approved operating model." })));

	fireEvent.click(within(detailsPanel).getByRole("button", { name: "Change Program owner" }));
	fireEvent.change(screen.getByLabelText("New Program owner"), { target: { value: "owner-2" } });
	fireEvent.change(screen.getByLabelText("Reason for changing owner"), { target: { value: "The deputy now holds the DPO position." } });
	fireEvent.click(screen.getByRole("button", { name: "Save Program owner" }));
	await waitFor(() => expect(assignProgram).toHaveBeenCalledWith("program-1", 4, "owner-2", "The deputy now holds the DPO position."));
  });

  it("supports source-anchored requirements, supersession and applicability decisions", async () => {
	const requirement = { id: "requirement-1", code: "CAR-01", title: "File the annual return", statement: "The bank must file its annual compliance return.", source_anchor: "GAID 2025, section 7", modality: "MUST", actor: "The bank", action: "file", object: "the annual return", status: "APPROVED", effective_from: "2026-01-01T00:00:00Z" };
	const value = { ...aggregate, requirements: [requirement] };
	vi.mocked(loadProgram).mockResolvedValue(value);
	vi.mocked(addProgramRequirement).mockResolvedValue(value);
	vi.mocked(supersedeProgramRequirement).mockResolvedValue(value);
	vi.mocked(determineProgramApplicability).mockResolvedValue(value);
	vi.mocked(loadProgramOperations).mockResolvedValue({ ...operations, operations: [
	  { command: "program.requirement.add", label: "Add a requirement", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current owner responsibility." },
	  { command: "program.requirement.supersede", subresource_id: "requirement-1", label: "Replace File the annual return", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current owner responsibility." },
	  { command: "program.applicability.decide", label: "Decide whether requirements apply", responsibility: "AUTHORIZER", can_act: true, assigned_to: { id: "cro", display_name: "Chief Risk Officer", kind: "PERSON", role: "CRO" }, reason: "You hold the current authorizer responsibility." },
	] });
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	fireEvent.click(await screen.findByRole("button", { name: "Add requirement" }));
	fireEvent.change(screen.getByLabelText("Requirement code"), { target: { value: "CAR-02" } });
	fireEvent.change(screen.getByLabelText("Requirement title"), { target: { value: "Keep filing evidence" } });
	fireEvent.change(screen.getByLabelText("Requirement statement"), { target: { value: "The bank must keep its filing receipt." } });
	fireEvent.change(screen.getByLabelText("Official source and section"), { target: { value: "GAID 2025, section 7.3" } });
	fireEvent.click(screen.getByRole("button", { name: "Save requirement" }));
	await waitFor(() => expect(addProgramRequirement).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ code: "CAR-02", sourceAnchor: "GAID 2025, section 7.3" })));

	fireEvent.click(screen.getByRole("button", { name: "Replace requirement" }));
	fireEvent.change(screen.getByLabelText("Replacement statement"), { target: { value: "The bank must file through a licensed DPCO." } });
	fireEvent.change(screen.getByLabelText("Replacement source and section"), { target: { value: "GAID 2025, section 7.2" } });
	fireEvent.change(screen.getByLabelText("Reason for replacing this requirement"), { target: { value: "The regulator changed the filing channel." } });
	fireEvent.click(screen.getByRole("button", { name: "Save replacement requirement" }));
	await waitFor(() => expect(supersedeProgramRequirement).toHaveBeenCalledWith("program-1", "requirement-1", 4, expect.objectContaining({ statement: "The bank must file through a licensed DPCO.", rationale: "The regulator changed the filing channel." })));

	fireEvent.click(screen.getByRole("button", { name: "Record applicability" }));
	fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "APPLICABLE" } });
	fireEvent.change(screen.getByLabelText("Applicability rationale"), { target: { value: "The licensed entity files this return." } });
	fireEvent.click(screen.getByRole("button", { name: "Save applicability decision" }));
	await waitFor(() => expect(determineProgramApplicability).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ requirementID: "requirement-1", status: "APPLICABLE", rationale: "The licensed entity files this return." })));
  });

  it("keeps unavailable owner actions read-only and names the responsible person", async () => {
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);
	expect((await screen.findAllByText("Data Protection Officer")).length).toBeGreaterThan(0);
	expect(screen.getByText("Assigned to Data Protection Officer.")).toBeTruthy();
	expect(within(document.getElementById("program-details-panel")!).queryByRole("button", { name: "Edit Program details" })).toBeNull();
  });
});
