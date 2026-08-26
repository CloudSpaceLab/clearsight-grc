import { StrictMode } from "react";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadEvidenceSources, loadMatterSummaries, loadProgram, loadProgramSummaries } from "../api";
import {
  addProgramEvidenceContract,
  addProgramRequirement,
	addProgramControlImplementation,
	addProgramControlObjective,
  assignProgram,
  determineProgramApplicability,
  loadProgramOperations,
	linkProgramRequirementControl,
  recordProgramEvidenceAssessment,
  supersedeProgramRequirement,
  transitionProgram,
  updateProgramDetails,
} from "../programOperationsApi";
import { acceptProgramReview, loadProgramReviewDigest } from "../programReviewApi";
import type { ProgramReviewDigest as ReviewDigest } from "../programReviewApi";
import { ApiError } from "../http";
import type { MatterAggregate, ProgramAggregate } from "../types";
import { createMatter } from "../continuityCommands";
import { ProgramRecordWorkspace } from "./ProgramRecordWorkspace";
import { ProgramsWorkspace } from "./ProgramsWorkspace";

vi.mock("../api", () => ({ loadProgram: vi.fn(), loadProgramSummaries: vi.fn(), loadMatterSummaries: vi.fn(), loadEvidenceSources: vi.fn() }));
vi.mock("../continuityCommands", async (importOriginal) => ({ ...(await importOriginal<typeof import("../continuityCommands")>()), createMatter: vi.fn() }));
vi.mock("../programOperationsApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../programOperationsApi")>()),
  loadProgramOperations: vi.fn(),
  updateProgramDetails: vi.fn(),
  assignProgram: vi.fn(),
  addProgramRequirement: vi.fn(),
  supersedeProgramRequirement: vi.fn(),
  determineProgramApplicability: vi.fn(),
	addProgramControlObjective: vi.fn(),
	addProgramControlImplementation: vi.fn(),
	linkProgramRequirementControl: vi.fn(),
  addProgramEvidenceContract: vi.fn(),
  recordProgramEvidenceAssessment: vi.fn(),
  transitionProgram: vi.fn(),
}));
vi.mock("../programReviewApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../programReviewApi")>()),
  acceptProgramReview: vi.fn(),
  loadProgramReviewDigest: vi.fn(),
}));
vi.mock("./MonitoringSetup", () => ({ MonitoringSetup: ({ canOperate = true }: { canOperate?: boolean }) => <section aria-label="Program monitoring"><h3>Monitoring</h3>{canOperate ? <button type="button">Add monitoring check</button> : <p>Monitoring changes are disabled until current Program responsibilities are available.</p>}</section> }));

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

const changedDigest = {
  ...digest,
  state: "CHANGED" as const,
  review_required: true,
  changes: [{ kind: "PROGRAM", summary: "The Program scope changed after the last review." }],
  changes_total: 1,
};

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => { resolve = next; });
  return { promise, resolve };
}

describe("Program record workspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadProgram).mockResolvedValue(aggregate);
    vi.mocked(loadProgramOperations).mockResolvedValue(operations);
	vi.mocked(loadProgramReviewDigest).mockResolvedValue(digest);
	vi.mocked(acceptProgramReview).mockResolvedValue(digest);
    vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-25T10:00:00Z" });
	vi.mocked(loadEvidenceSources).mockResolvedValue([]);
	vi.mocked(loadMatterSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-25T10:00:00Z" });
	vi.mocked(updateProgramDetails).mockResolvedValue(aggregate);
	vi.mocked(assignProgram).mockResolvedValue(aggregate);
	vi.mocked(addProgramRequirement).mockResolvedValue(aggregate);
	vi.mocked(supersedeProgramRequirement).mockResolvedValue(aggregate);
	vi.mocked(determineProgramApplicability).mockResolvedValue(aggregate);
	vi.mocked(addProgramControlObjective).mockResolvedValue(aggregate);
	vi.mocked(addProgramControlImplementation).mockResolvedValue(aggregate);
	vi.mocked(linkProgramRequirementControl).mockResolvedValue(aggregate);
	vi.mocked(addProgramEvidenceContract).mockResolvedValue(aggregate);
	vi.mocked(recordProgramEvidenceAssessment).mockResolvedValue(aggregate);
	vi.mocked(transitionProgram).mockResolvedValue(aggregate);
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

  it("keeps the Program visible and retries only responsibilities when responsibility loading fails", async () => {
	vi.mocked(loadProgramReviewDigest).mockResolvedValue(changedDigest);
	vi.mocked(loadProgramOperations)
	  .mockRejectedValueOnce(new Error("routing unavailable"))
	  .mockResolvedValue({
		...operations,
		operations: operations.operations.map((operation) => ({ ...operation, can_act: true })),
	  });
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	expect(await screen.findByRole("heading", { name: "Nigeria data protection" })).toBeTruthy();
	expect(screen.getByText("Two applicable requirements do not have evidence checks.")).toBeTruthy();
	expect(await screen.findByRole("heading", { name: "1 change since your last review" })).toBeTruthy();
	expect(screen.queryByRole("button", { name: "Mark current state reviewed" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Approve Program activation" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Record new issue" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Add monitoring check" })).toBeNull();
	expect(screen.getByText("New issues cannot be recorded until current Program responsibilities are available.")).toBeTruthy();
	expect(screen.getByText("Monitoring changes are disabled until current Program responsibilities are available.")).toBeTruthy();

	fireEvent.click(screen.getByRole("button", { name: "Retry responsibilities" }));
	expect(await screen.findByRole("button", { name: "Approve Program activation" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Record new issue" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Add monitoring check" })).toBeTruthy();
	expect(loadProgramOperations).toHaveBeenCalledTimes(2);
	expect(loadProgram).toHaveBeenCalledTimes(1);
	expect(loadProgramReviewDigest).toHaveBeenCalledTimes(1);
  });

  it("keeps degraded Program labels readable without exposing stored principal or source identifiers", async () => {
	vi.mocked(loadProgram).mockResolvedValue({
	  ...aggregate,
	  program: { ...aggregate.program, owner_principal_id: "program-owner-internal" },
	  control_objectives: [{ id: "objective-1", code: "COMPLETE", name: "Complete filing", outcome: "All sections are filed.", status: "ACTIVE" }],
	  control_implementations: [{ id: "implementation-1", objective_id: "objective-1", name: "Filing checklist", description: "Review every section.", implementation_type: "CHECKLIST", owner_principal_id: "safeguard-owner-internal", status: "IMPLEMENTED" }],
	  evidence_contracts: [{ id: "contract-1", code: "FILED", name: "Filing proof", claim: "The return was filed.", acceptable_source_ids: ["source-internal"], status: "ACTIVE", freshness_minutes: 1440, minimum_coverage: 1, independence_required: true, contradiction_policy: "REVIEW", failure_action: "MATTER" }],
	  evidence_assessments: [{ id: "assessment-1", contract_id: "contract-1", conclusion: "SUPPORTED", coverage: 1, assessed_by: "reviewer-internal", assessed_at: "2026-08-25T10:00:00Z", valid_until: "2099-08-25T10:00:00Z" }],
	});
	vi.mocked(loadProgramOperations).mockRejectedValue(new Error("routing unavailable"));

	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	expect(await screen.findByRole("heading", { name: "Nigeria data protection" })).toBeTruthy();
	expect(screen.queryByText(/program-owner-internal|safeguard-owner-internal|source-internal|reviewer-internal/)).toBeNull();
	expect(screen.getAllByText("Recorded Program owner unavailable").length).toBeGreaterThan(0);
	expect(screen.getByText(/Recorded safeguard owner unavailable/)).toBeTruthy();
	expect(screen.getByText(/Accepted sources: Source name unavailable/)).toBeTruthy();
	expect(screen.getByText(/Reviewer name unavailable/)).toBeTruthy();
  });

  it("shows review history without acknowledgement when the live review operation cannot act", async () => {
	vi.mocked(loadProgramReviewDigest).mockResolvedValue(changedDigest);
	vi.mocked(loadProgramOperations).mockResolvedValue({
	  ...operations,
	  operations: [...operations.operations, {
		command: "program.review.accept", label: "Mark current state reviewed", responsibility: "REVIEWER", can_act: false,
		reason: "This review is assigned to the current reviewer.",
	  }],
	});
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	expect(await screen.findByRole("heading", { name: "1 change since your last review" })).toBeTruthy();
	expect(screen.getByText("The Program scope changed after the last review.")).toBeTruthy();
	expect(screen.queryByRole("button", { name: "Mark current state reviewed" })).toBeNull();
  });

  it.each([
	["Program version", { current_program_version: 5 }],
	["calculated-status version", { current_projection_version: 7 }],
  ])("reloads review status when the %s does not match the displayed Program", async (_label, mismatch) => {
	vi.mocked(loadProgramReviewDigest)
	  .mockResolvedValueOnce({ ...changedDigest, ...mismatch })
	  .mockResolvedValue(changedDigest);
	vi.mocked(loadProgramOperations).mockResolvedValue({
	  ...operations,
	  operations: [...operations.operations, {
		command: "program.review.accept", label: "Mark current state reviewed", responsibility: "REVIEWER", can_act: true,
		reason: "You hold the current review responsibility.",
	  }],
	});
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	expect(await screen.findByText("The Program scope changed after the last review.")).toBeTruthy();
	expect(screen.queryByRole("button", { name: "Mark current state reviewed" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Approve Program activation" })).toBeNull();
	fireEvent.click(screen.getByRole("button", { name: "Reload review status" }));

	const reviewPanel = document.getElementById("program-review-panel")!;
	expect(await within(reviewPanel).findByRole("button", { name: "Mark current state reviewed" })).toBeTruthy();
	expect(loadProgramReviewDigest).toHaveBeenCalledTimes(2);
	expect(loadProgram).toHaveBeenCalledTimes(1);
	expect(loadProgramOperations).toHaveBeenCalledTimes(1);
  });

  it("loads once and accepts the current Program command under StrictMode", async () => {
	const pending = deferred<ProgramAggregate>();
	vi.mocked(loadProgramOperations).mockResolvedValue({
	  ...operations,
	  operations: [{ ...operations.operations[0]!, can_act: true }],
	});
	vi.mocked(updateProgramDetails).mockReturnValue(pending.promise);
	render(<StrictMode><ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/></StrictMode>);

	const detailsPanel = (await screen.findByRole("heading", { name: "Scope and ownership" })).closest("article")!;
	await waitFor(() => {
	  expect(loadProgram).toHaveBeenCalledTimes(1);
	  expect(loadProgramOperations).toHaveBeenCalledTimes(1);
	  expect(loadProgramReviewDigest).toHaveBeenCalledTimes(1);
	});
	fireEvent.click(within(detailsPanel).getByRole("button", { name: "Edit Program details" }));
	fireEvent.change(screen.getByLabelText("Reason for this change"), { target: { value: "Keep the current command valid." } });
	fireEvent.click(screen.getByRole("button", { name: "Save Program details" }));
	await act(async () => {
	  pending.resolve({ ...aggregate, program: { ...aggregate.program, name: "Updated current Program", version: 5 } });
	  await pending.promise;
	});
	expect(await screen.findByRole("heading", { name: "Updated current Program" })).toBeTruthy();
  });

  it("disables Program mutations until responsibility data matches the displayed version", async () => {
	vi.mocked(loadProgramOperations)
	  .mockResolvedValueOnce({ ...operations, program_version: 3 })
	  .mockResolvedValue(operations);
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	expect(await screen.findByRole("heading", { name: "Nigeria data protection" })).toBeTruthy();
	expect(screen.queryByRole("button", { name: "Approve Program activation" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Record new issue" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Add monitoring check" })).toBeNull();
	fireEvent.click(screen.getByRole("button", { name: "Reload Program data" }));

	expect(await screen.findByRole("button", { name: "Approve Program activation" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Record new issue" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Add monitoring check" })).toBeTruthy();
	expect(loadProgram).toHaveBeenCalledTimes(2);
	expect(loadProgramOperations).toHaveBeenCalledTimes(2);
	expect(loadProgramReviewDigest).toHaveBeenCalledTimes(2);
  });

  it("reloads the Program, responsibilities and review after a command conflict", async () => {
	vi.mocked(loadProgramOperations).mockResolvedValue({
	  ...operations,
	  operations: [{ ...operations.operations[0]!, can_act: true }],
	});
	vi.mocked(updateProgramDetails).mockRejectedValue(new ApiError(409, "version conflict", "version_conflict"));
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	const detailsPanel = (await screen.findByRole("heading", { name: "Scope and ownership" })).closest("article")!;
	fireEvent.click(within(detailsPanel).getByRole("button", { name: "Edit Program details" }));
	fireEvent.change(screen.getByLabelText("Reason for this change"), { target: { value: "Use the current approved scope." } });
	fireEvent.click(screen.getByRole("button", { name: "Save Program details" }));
	fireEvent.click(await screen.findByRole("button", { name: "Reload Program" }));

	await waitFor(() => {
	  expect(loadProgram).toHaveBeenCalledTimes(2);
	  expect(loadProgramOperations).toHaveBeenCalledTimes(2);
	  expect(loadProgramReviewDigest).toHaveBeenCalledTimes(2);
	});
  });

  it("ignores a completed Program command after navigation targets another Program", async () => {
	const pending = deferred<ProgramAggregate>();
	const secondAggregate: ProgramAggregate = {
	  ...aggregate,
	  program: { ...aggregate.program, id: "program-2", code: "AML", name: "Anti-money laundering", version: 4 },
	};
	vi.mocked(loadProgram).mockImplementation((id) => Promise.resolve(id === "program-2" ? secondAggregate : aggregate));
	vi.mocked(loadProgramOperations).mockImplementation((id) => Promise.resolve({
	  ...operations,
	  program_id: id,
	  operations: [{ ...operations.operations[0]!, can_act: true }],
	}));
	vi.mocked(loadProgramReviewDigest).mockImplementation((id) => Promise.resolve({ ...digest, program_id: id }));
	vi.mocked(updateProgramDetails).mockReturnValue(pending.promise);
	const view = render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	const detailsPanel = (await screen.findByRole("heading", { name: "Scope and ownership" })).closest("article")!;
	fireEvent.click(within(detailsPanel).getByRole("button", { name: "Edit Program details" }));
	fireEvent.change(screen.getByLabelText("Reason for this change"), { target: { value: "Update the first Program." } });
	fireEvent.click(screen.getByRole("button", { name: "Save Program details" }));
	view.rerender(<ProgramRecordWorkspace programID="program-2" onBack={vi.fn()}/>);
	expect(await screen.findByRole("heading", { name: "Anti-money laundering" })).toBeTruthy();

	pending.resolve({ ...aggregate, program: { ...aggregate.program, name: "Stale completed Program", version: 5 } });
	await waitFor(() => expect(screen.queryByRole("heading", { name: "Stale completed Program" })).toBeNull());
	expect(screen.getByRole("heading", { name: "Anti-money laundering" })).toBeTruthy();
  });

  it("ignores a completed review acknowledgement after navigation targets another Program", async () => {
	const pending = deferred<ReviewDigest>();
	const secondAggregate: ProgramAggregate = {
	  ...aggregate,
	  program: { ...aggregate.program, id: "program-2", code: "AML", name: "Anti-money laundering", version: 4 },
	};
	const reviewOperation = {
	  command: "program.review.accept", label: "Mark current state reviewed", responsibility: "REVIEWER", can_act: true,
	  reason: "You hold the current review responsibility.",
	};
	vi.mocked(loadProgram).mockImplementation((id) => Promise.resolve(id === "program-2" ? secondAggregate : aggregate));
	vi.mocked(loadProgramOperations).mockImplementation((id) => Promise.resolve({
	  ...operations, program_id: id, operations: [...operations.operations, reviewOperation],
	}));
	vi.mocked(loadProgramReviewDigest).mockImplementation((id) => Promise.resolve({
	  ...changedDigest,
	  program_id: id,
	  changes: [{ kind: "PROGRAM", summary: id === "program-2" ? "The second Program review is still outstanding." : "The first Program review is still outstanding." }],
	}));
	vi.mocked(acceptProgramReview).mockReturnValue(pending.promise);
	const view = render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	await screen.findByText("The first Program review is still outstanding.");
	const reviewPanel = document.getElementById("program-review-panel")!;
	fireEvent.click(await within(reviewPanel).findByRole("button", { name: "Mark current state reviewed" }));
	expect(acceptProgramReview).toHaveBeenCalledWith("program-1", 4, 8);
	view.rerender(<ProgramRecordWorkspace programID="program-2" onBack={vi.fn()}/>);
	expect(await screen.findByText("The second Program review is still outstanding.")).toBeTruthy();

	await act(async () => {
	  pending.resolve({ ...changedDigest, program_id: "program-1", changes: [{ kind: "PROGRAM", summary: "Late review result from the first Program." }] });
	  await pending.promise;
	});
	await waitFor(() => expect(screen.queryByText("Late review result from the first Program.")).toBeNull());
	expect(screen.getByText("The second Program review is still outstanding.")).toBeTruthy();
  });

  it("keeps linked-issue retry and navigation available when Program responsibilities fail", async () => {
	const linked = {
	  matter: { id: "matter-1", tenant_id: "bank", reference: "MAT-001", type: "CONTROL_GAP", status: "OPEN", priority: 4, title: "Annual return evidence is incomplete", summary: "Two sections need approved evidence.", scope: {}, known_facts: {}, missing_facts: [], contradictions: [], created_at: "2026-08-20T10:00:00Z", updated_at: "2026-08-25T10:00:00Z", version: 2 },
	  type_label: "Control gap", status_label: "Open", next_action: "Assign the evidence owners", program_count: 1, open_action_count: 1, outcome_check_count: 0,
	};
	vi.mocked(loadProgramOperations).mockRejectedValue(new Error("routing unavailable"));
	vi.mocked(loadMatterSummaries)
	  .mockRejectedValueOnce(new Error("linked issues unavailable"))
	  .mockResolvedValue({ items: [linked], generated_at: "2026-08-25T10:00:00Z" });
	const onOpenMatter = vi.fn();
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()} onOpenMatter={onOpenMatter}/>);

	const issues = await screen.findByRole("heading", { name: "Linked issues and changes" });
	const panel = issues.closest("article")!;
	fireEvent.click(await within(panel).findByRole("button", { name: "Try again" }));
	fireEvent.click(await within(panel).findByRole("button", { name: "Open MAT-001" }));

	expect(loadMatterSummaries).toHaveBeenCalledTimes(2);
	expect(onOpenMatter).toHaveBeenCalledWith("matter-1");
  });

  it("keeps the Program visible and retries only review status when review loading fails", async () => {
	vi.mocked(loadProgramReviewDigest)
	  .mockRejectedValueOnce(new Error("review unavailable"))
	  .mockResolvedValue(digest);
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

	expect(await screen.findByRole("heading", { name: "Nigeria data protection" })).toBeTruthy();
	expect(screen.getByText("Two applicable requirements do not have evidence checks.")).toBeTruthy();
	expect(screen.queryByRole("button", { name: "Approve Program activation" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Record new issue" })).toBeNull();
	expect(screen.queryByRole("button", { name: "Add monitoring check" })).toBeNull();

	fireEvent.click(screen.getByRole("button", { name: "Retry review status" }));
	expect(await screen.findByRole("button", { name: "Approve Program activation" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Record new issue" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Add monitoring check" })).toBeTruthy();
	expect(loadProgramReviewDigest).toHaveBeenCalledTimes(2);
	expect(loadProgram).toHaveBeenCalledTimes(1);
	expect(loadProgramOperations).toHaveBeenCalledTimes(1);
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

  it("defines safeguard objectives, assigns eligible owners and links requirement coverage", async () => {
	const value: ProgramAggregate = {
	  ...aggregate,
	  requirements: [{ id: "requirement-1", code: "CAR-01", title: "File the annual return", statement: "The bank must file its annual compliance return.", status: "APPROVED", source_anchor: "GAID 2025, section 7" }],
	  control_objectives: [{ id: "objective-1", code: "CAR-COMPLETE", name: "Complete return", outcome: "Every required section is filed.", status: "ACTIVE" }],
	  control_implementations: [{ id: "implementation-1", objective_id: "objective-1", name: "Annual return checklist", description: "Owners confirm each section.", implementation_type: "CHECKLIST", owner_principal_id: "control-owner", status: "IMPLEMENTED" }],
	};
	vi.mocked(loadProgram).mockResolvedValue(value);
	vi.mocked(addProgramControlObjective).mockResolvedValue(value);
	vi.mocked(addProgramControlImplementation).mockResolvedValue(value);
	vi.mocked(linkProgramRequirementControl).mockResolvedValue(value);
	vi.mocked(loadProgramOperations).mockResolvedValue({ ...operations, operations: [{
	  command: "program.safeguard.define", label: "Define safeguards", responsibility: "ACCOUNTABLE_OWNER", can_act: true,
	  assigned_to: { id: "owner-1", display_name: "Data Protection Officer", kind: "PERSON", role: "DPO" },
	  candidates: [{ id: "control-owner", display_name: "Privacy control owner", kind: "PERSON", role: "Control owner" }],
	  reason: "You hold the current Program owner responsibility.",
	}] });
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);
	await screen.findByRole("heading", { name: "Safeguards and coverage" });
	const panel = document.getElementById("program-safeguards-panel")!;

	fireEvent.click(within(panel).getByRole("button", { name: "Add control objective" }));
	fireEvent.change(screen.getByLabelText("Objective code"), { target: { value: "CAR-ACCURATE" } });
	fireEvent.change(screen.getByLabelText("Objective name"), { target: { value: "Accurate annual return" } });
	fireEvent.change(screen.getByLabelText("Intended outcome"), { target: { value: "Every filed section agrees with current bank records." } });
	fireEvent.click(screen.getByRole("button", { name: "Save control objective" }));
	await waitFor(() => expect(addProgramControlObjective).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ code: "CAR-ACCURATE", status: "ACTIVE" })));

	fireEvent.click(within(panel).getByRole("button", { name: "Add safeguard" }));
	fireEvent.change(screen.getByLabelText("Safeguard owner"), { target: { value: "control-owner" } });
	fireEvent.change(screen.getByLabelText("Safeguard name"), { target: { value: "Return accuracy review" } });
	fireEvent.change(screen.getByLabelText("How the safeguard works"), { target: { value: "The control owner reconciles every section before filing." } });
	fireEvent.click(screen.getByRole("button", { name: "Save safeguard" }));
	await waitFor(() => expect(addProgramControlImplementation).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ ownerPrincipalID: "control-owner", name: "Return accuracy review" })));

	fireEvent.click(within(panel).getByRole("button", { name: "Link requirement to safeguard" }));
	fireEvent.click(screen.getByRole("button", { name: "Save coverage link" }));
	await waitFor(() => expect(linkProgramRequirementControl).toHaveBeenCalledWith("program-1", 4, "requirement-1", "implementation-1"));
  });

  it("defines evidence expectations, records reviewer results and keeps monitoring separate", async () => {
	const value: ProgramAggregate = {
	  ...aggregate,
	  requirements: [{ id: "requirement-1", code: "CAR-01", title: "File the annual return", statement: "The bank must file its annual compliance return.", status: "APPROVED" }],
	  control_implementations: [{ id: "implementation-1", name: "Annual return checklist", description: "Owners confirm each section.", implementation_type: "CHECKLIST", status: "IMPLEMENTED" }],
	  evidence_contracts: [{ id: "contract-1", requirement_id: "requirement-1", code: "CAR-EVIDENCE", name: "Annual return filing evidence", claim: "The complete annual return was filed by the deadline.", acceptable_source_ids: ["source-1"], status: "ACTIVE", freshness_minutes: 43200, minimum_coverage: .95, independence_required: true, contradiction_policy: "REVIEW", failure_action: "MATTER" }],
	  evidence_assessments: [],
	};
	vi.mocked(loadProgram).mockResolvedValue(value);
	vi.mocked(loadEvidenceSources).mockResolvedValue([{ id: "source-1", tenant_id: "bank", code: "RETURN", name: "Annual return register", type: "REGISTER", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 1440, health: "HEALTHY", status: "ACTIVE", version: 1 }]);
	vi.mocked(addProgramEvidenceContract).mockResolvedValue(value);
	vi.mocked(recordProgramEvidenceAssessment).mockResolvedValue(value);
	vi.mocked(loadProgramOperations).mockResolvedValue({ ...operations, operations: [
	  { command: "program.evidence.define", label: "Define an evidence check", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You hold the current owner responsibility." },
	  { command: "program.evidence.assess", label: "Record evidence check results", responsibility: "REVIEWER", can_act: true, assigned_to: { id: "reviewer-1", display_name: "Compliance assurance reviewer", kind: "PERSON", role: "Reviewer" }, reason: "You hold the current reviewer responsibility." },
	] });
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);
	await screen.findByRole("heading", { name: "Evidence checks and results" });
	const panel = document.getElementById("program-evidence-panel")!;
	expect(within(panel).getByRole("heading", { name: "Monitoring" })).toBeTruthy();

	fireEvent.click(within(panel).getByRole("button", { name: "Define evidence check" }));
	fireEvent.change(screen.getByLabelText("Evidence code"), { target: { value: "CAR-COMPLETE" } });
	fireEvent.change(screen.getByLabelText("Evidence check name"), { target: { value: "Complete annual return evidence" } });
	fireEvent.change(screen.getByLabelText("What must the evidence prove?"), { target: { value: "Every required return section was filed." } });
	fireEvent.click(await screen.findByLabelText("Annual return register"));
	fireEvent.change(screen.getByLabelText("Maximum evidence age (days)"), { target: { value: "30" } });
	fireEvent.change(screen.getByLabelText("Required population coverage (%)"), { target: { value: "95" } });
	fireEvent.click(screen.getByLabelText("Independent review required"));
	fireEvent.click(screen.getByRole("button", { name: "Save evidence check" }));
	await waitFor(() => expect(addProgramEvidenceContract).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ code: "CAR-COMPLETE", acceptableSourceIDs: ["source-1"], freshnessMinutes: 43200, minimumCoverage: .95, independenceRequired: true })));

	fireEvent.click(within(panel).getByRole("button", { name: "Record evidence result" }));
	fireEvent.change(screen.getByLabelText("Conclusion"), { target: { value: "PARTIALLY_SUPPORTED" } });
	fireEvent.change(screen.getByLabelText("Population coverage (%)"), { target: { value: "89" } });
	fireEvent.change(screen.getByLabelText("Assessment basis"), { target: { value: "89 of 100 filing sections have current evidence." } });
	fireEvent.change(screen.getByLabelText("Evidence references"), { target: { value: "Return register export\nFiling receipt" } });
	fireEvent.click(screen.getByRole("button", { name: "Save evidence result" }));
	await waitFor(() => expect(recordProgramEvidenceAssessment).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ contractID: "contract-1", conclusion: "PARTIALLY_SUPPORTED", coverage: .89, basis: { summary: "89 of 100 filing sections have current evidence.", evidence_references: ["Return register export", "Filing receipt"] } })));
  });

  it("shows only exactly linked issues and opens newly created work", async () => {
	const linked = {
	  matter: { id: "matter-1", tenant_id: "bank", reference: "MAT-001", type: "CONTROL_GAP", status: "OPEN", priority: 4, title: "Annual return evidence is incomplete", summary: "Two sections need approved evidence.", scope: {}, known_facts: {}, missing_facts: [], contradictions: [], created_at: "2026-08-20T10:00:00Z", updated_at: "2026-08-25T10:00:00Z", version: 2 },
	  type_label: "Control gap", status_label: "Open", next_action: "Assign the evidence owners", program_count: 1, open_action_count: 1, outcome_check_count: 0,
	};
	const created: MatterAggregate = { type_label: "Control gap", status_label: "Draft", next_action: "Start initial review", matter: { ...linked.matter, id: "matter-new", reference: "MAT-NEW", title: "New Program issue", status: "DRAFT", version: 1 }, links: [{ id: "link-new", program_id: "program-1", relationship: "AFFECTS" }], decisions: [], actions: [], verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: [] } };
	vi.mocked(loadMatterSummaries).mockResolvedValue({ items: [linked], generated_at: "2026-08-25T10:00:00Z" });
	vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [{ program: aggregate.program, state_label: aggregate.state_label, overall_state: "EVIDENCE_INSUFFICIENT", reasons: [], reasons_total: 0, reasons_omitted: 0, open_matter_count: 1, requirement_count: 0, safeguard_count: 0, evidence_check_count: 0, program_version: 4, assessed_program_version: 3, projection_version: 8, projection_stale: true }], generated_at: "2026-08-25T10:00:00Z" });
	vi.mocked(createMatter).mockResolvedValue(created);
	const onOpenMatter = vi.fn();
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()} onOpenMatter={onOpenMatter}/>);
	await screen.findByRole("heading", { name: "Linked issues and changes" });
	const issues = document.getElementById("program-issues-panel")!;
	expect(loadMatterSummaries).toHaveBeenCalledWith({ status: "OPEN", programID: "program-1", limit: 20 });
	fireEvent.click(await within(issues).findByRole("button", { name: "Open MAT-001" }));
	expect(onOpenMatter).toHaveBeenCalledWith("matter-1");

	fireEvent.click(within(issues).getByRole("button", { name: "Record new issue" }));
	await screen.findByRole("option", { name: "Nigeria data protection (NDPA)" });
	expect((screen.getByLabelText("Program (optional)") as HTMLSelectElement).value).toBe("program-1");
	fireEvent.change(screen.getByLabelText("Title"), { target: { value: "New Program issue" } });
	fireEvent.change(screen.getByLabelText("What happened or changed?"), { target: { value: "A current evidence gap needs an owner." } });
	fireEvent.change(screen.getByLabelText("Affected area"), { target: { value: "Annual return" } });
	fireEvent.click(screen.getByRole("button", { name: "Create issue or change" }));
	await waitFor(() => expect(createMatter).toHaveBeenCalledWith(expect.objectContaining({ programID: "program-1", title: "New Program issue" })));
	expect(onOpenMatter).toHaveBeenCalledWith("matter-new");
  });

  it("lets the current authorizer change Program operating status with a reason", async () => {
	const updated = { ...aggregate, program: { ...aggregate.program, status: "ACTIVE", version: 5 } };
	vi.mocked(transitionProgram).mockResolvedValue(updated);
	render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);
	await screen.findByRole("heading", { name: "Operating status" });
	fireEvent.change(screen.getByLabelText("Reason for status change"), { target: { value: "The approved requirements, safeguards and evidence checks are in place." } });
	fireEvent.click(screen.getByRole("button", { name: "Activate Program" }));
	await waitFor(() => expect(transitionProgram).toHaveBeenCalledWith("program-1", 4, "ACTIVE", "The approved requirements, safeguards and evidence checks are in place."));
  });
});
