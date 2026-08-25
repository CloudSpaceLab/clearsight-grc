import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadMatter } from "../api";
import { ApiError } from "../http";
import { assignMatter, assignMatterAction, changeMatterContext, loadMatterOperations, updateMatterAction, updateMatterDetails } from "../matterOperationsApi";
import type { MatterOperations } from "../matterOperationsApi";
import { addMatterAction, transitionMatterAction } from "../continuityCommands";
import type { MatterAggregate } from "../types";
import { MatterRecordWorkspace } from "./MatterRecordWorkspace";

vi.mock("../api", () => ({ loadMatter: vi.fn() }));
vi.mock("../matterOperationsApi", () => ({
  assignMatter: vi.fn(),
  assignMatterAction: vi.fn(),
  changeMatterContext: vi.fn(),
  loadMatterOperations: vi.fn(),
  updateMatterAction: vi.fn(),
  updateMatterDetails: vi.fn(),
}));

vi.mock("../continuityCommands", () => ({ addMatterAction: vi.fn(), transitionMatterAction: vi.fn() }));

const detail: MatterAggregate = {
  type_label: "Regulatory change",
  status_label: "Work in progress",
  next_action: "Update the annual return evidence checklist",
  matter: {
    id: "matter-1", tenant_id: "bank-1", reference: "MAT-82BF", type: "REGULATORY_CHANGE", status: "ACTION_IN_PROGRESS",
    priority: 4, title: "Implement GAID 2025 annual return requirements",
    summary: "Update the annual return process, evidence ownership and filing timetable.",
    scope: { affected_area: "Privacy governance" },
    known_facts: { filing_channel: "licensed DPCO", filing_deadline: "31 March", current_evidence_sections: 8, required_sections: 10 },
    missing_facts: ["approved owner for the cross-border transfer section", "final DPCO evidence checklist"], contradictions: [],
    owner_principal_id: "owner-1", due_at: "2026-08-26T16:00:00Z", created_at: "2026-08-01T10:00:00Z",
    updated_at: "2026-08-25T09:00:00Z", version: 7,
  },
  links: [], decisions: [],
  actions: [{ id: "action-1", title: "Update the annual return evidence checklist", description: "Assign the remaining sections and record the review date.", owner_principal_id: "owner-1", status: "IN_PROGRESS", due_at: "2026-08-26T16:00:00Z", version: 2 }],
  verification_contracts: [], verification_results: [], response_packages: [],
  closure: { ready: false, reasons: ["One action has not been implemented.", "The outcome has not been checked."] },
};

const operations: MatterOperations = {
  matter_id: "matter-1", matter_version: 7, authority_available: true, generated_at: "2026-08-25T09:00:00Z",
  operations: [
    { command: "matter.details.update", label: "Edit issue details", responsibility: "ACCOUNTABLE_OWNER", can_act: false, assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" }, reason: "This change is assigned to the Program Owner." },
    { command: "matter.context.change", label: "Update facts and missing information", responsibility: "ACCOUNTABLE_OWNER", can_act: false, assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" }, reason: "This change is assigned to the Program Owner." },
    { command: "matter.action.transition", subresource_id: "action-1", label: "Update action", responsibility: "PERFORMER", can_act: false, assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" }, reason: "This action is assigned to the Program Owner.", allowed_targets: ["IMPLEMENTED", "BLOCKED"] },
  ],
};

const ownerOperations: MatterOperations = {
  ...operations,
  operations: [
    { ...operations.operations[0]!, can_act: true },
    { ...operations.operations[1]!, can_act: true },
    {
      command: "matter.assign", label: "Change issue owner", responsibility: "ACCOUNTABLE_OWNER", can_act: true,
      assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" },
      candidates: [
        { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" },
        { id: "owner-2", display_name: "Privacy Operations Lead", kind: "PERSON", role: "PROGRAM_OWNER" },
      ],
      reason: "You can assign the current accountable owner.",
    },
    operations.operations[2]!,
  ],
};

const actionOwnerOperations: MatterOperations = {
  ...ownerOperations,
  operations: [
    ...ownerOperations.operations,
    {
      command: "matter.action.add", label: "Add an action", responsibility: "ACCOUNTABLE_OWNER", can_act: true,
      assigned_to: ownerOperations.operations[0]!.assigned_to,
      candidates: ownerOperations.operations[2]!.candidates,
      reason: "You can add assigned work for this issue.",
    },
    { command: "matter.action.update", subresource_id: "action-1", label: "Edit action", responsibility: "ACCOUNTABLE_OWNER", can_act: true, assigned_to: ownerOperations.operations[0]!.assigned_to, reason: "You can update this action." },
    { command: "matter.action.assign", subresource_id: "action-1", label: "Change action owner", responsibility: "ACCOUNTABLE_OWNER", can_act: true, assigned_to: ownerOperations.operations[0]!.assigned_to, candidates: ownerOperations.operations[2]!.candidates, reason: "You can assign an eligible performer." },
  ],
};

const performerOperations: MatterOperations = {
  ...operations,
  operations: operations.operations.map((operation) => operation.command === "matter.action.transition"
    ? { ...operation, can_act: true, allowed_targets: ["IMPLEMENTED", "BLOCKED"] }
    : operation),
};

describe("Matter record workspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadMatter).mockResolvedValue(detail);
    vi.mocked(loadMatterOperations).mockResolvedValue(operations);
  });

  it("shows the exact record, current owner, deadline and blocker without inventing a CRO command", async () => {
    const onBack = vi.fn();
    render(<MatterRecordWorkspace matterID="matter-1" onBack={onBack}/>);

    expect(await screen.findByRole("heading", { name: "Implement GAID 2025 annual return requirements" })).toBeTruthy();
    expect(screen.getByLabelText("Current responsibility and timing").textContent).toContain("Assigned to Program Owner");
    expect(screen.getByText("Due 26 Aug 2026")).toBeTruthy();
    expect(screen.getByText("2 missing information items")).toBeTruthy();
    expect(screen.getByText("final DPCO evidence checklist")).toBeTruthy();
    expect(screen.getAllByText("This action is assigned to the Program Owner.").length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Update action" })).toBeNull();
    expect(screen.getAllByTestId("dominant-next-action")).toHaveLength(1);

    fireEvent.click(screen.getByRole("button", { name: "Back to issues and changes" }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("reports a recoverable load failure without showing stale controls", async () => {
    vi.mocked(loadMatterOperations).mockRejectedValue(new Error("routing unavailable"));
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    expect(await screen.findByRole("heading", { name: "Issue responsibilities could not be loaded" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Update action" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(loadMatter).toHaveBeenCalledTimes(2));
  });

  it("updates core issue details and keeps the server version current", async () => {
    vi.mocked(loadMatterOperations).mockResolvedValue(ownerOperations);
    vi.mocked(updateMatterDetails).mockResolvedValue({ ...detail, matter: { ...detail.matter, due_at: "2026-09-01T00:00:00.000Z", version: 8 } });
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Edit issue details" }));
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-09-01" } });
    fireEvent.change(screen.getByLabelText("Reason for this change"), { target: { value: "Use the agreed internal completion date." } });
    fireEvent.click(screen.getByRole("button", { name: "Save issue details" }));

    await waitFor(() => expect(updateMatterDetails).toHaveBeenCalledWith("matter-1", 7, expect.objectContaining({
      title: detail.matter.title,
      dueAt: "2026-09-01T00:00:00.000Z",
      rationale: "Use the agreed internal completion date.",
    })));
    expect(await screen.findByText("Issue details updated.")).toBeTruthy();
  });

  it("corrects recorded information and resolves a missing item with supporting evidence", async () => {
    vi.mocked(loadMatterOperations).mockResolvedValue(ownerOperations);
    vi.mocked(changeMatterContext)
      .mockResolvedValueOnce({ ...detail, matter: { ...detail.matter, known_facts: { ...detail.matter.known_facts, filing_channel: "licensed DPCO portal" }, version: 8 } })
      .mockResolvedValueOnce({ ...detail, matter: { ...detail.matter, missing_facts: [detail.matter.missing_facts[0]], version: 9 } });
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Edit Filing Channel" }));
    fireEvent.change(screen.getByLabelText("Updated value"), { target: { value: "licensed DPCO portal" } });
    fireEvent.change(screen.getByLabelText("Evidence references (optional)"), { target: { value: "DPCO guidance 2026\nportal instructions v2" } });
    fireEvent.change(screen.getByLabelText("Reason for this correction"), { target: { value: "Replace the superseded submission route." } });
    fireEvent.click(screen.getByRole("button", { name: "Save Filing Channel" }));

    await waitFor(() => expect(changeMatterContext).toHaveBeenCalledWith("matter-1", 7, {
      kind: "CORRECT_FACT", key: "filing_channel", label: "Filing Channel", value: "licensed DPCO portal",
      evidenceReferences: ["DPCO guidance 2026", "portal instructions v2"], rationale: "Replace the superseded submission route.",
    }));

    fireEvent.click(screen.getByRole("button", { name: "Add information for final DPCO evidence checklist" }));
    fireEvent.change(screen.getByLabelText("Information to record"), { target: { value: "Checklist v3 approved" } });
    fireEvent.change(screen.getByLabelText("Evidence references (optional)"), { target: { value: "artifact-checklist-v3" } });
    fireEvent.change(screen.getByLabelText("Reason this information resolves the gap"), { target: { value: "The DPCO approved version 3." } });
    fireEvent.click(screen.getByRole("button", { name: "Record missing information" }));

    await waitFor(() => expect(changeMatterContext).toHaveBeenLastCalledWith("matter-1", 8, {
      kind: "RESOLVE_MISSING", key: "final_dpco_evidence_checklist", label: "final DPCO evidence checklist",
      value: "Checklist v3 approved", evidenceReferences: ["artifact-checklist-v3"], rationale: "The DPCO approved version 3.",
    }));
  });

  it("adds a missing-information requirement and changes the accountable owner from eligible candidates", async () => {
    vi.mocked(loadMatterOperations).mockResolvedValue(ownerOperations);
    vi.mocked(changeMatterContext).mockResolvedValue({ ...detail, matter: { ...detail.matter, missing_facts: [...detail.matter.missing_facts, "approved signatory"], version: 8 } });
    vi.mocked(assignMatter).mockResolvedValue({ ...detail, matter: { ...detail.matter, owner_principal_id: "owner-2", version: 9 } });
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Add missing information" }));
    fireEvent.change(screen.getByLabelText("Information needed"), { target: { value: "approved signatory" } });
    fireEvent.change(screen.getByLabelText("Why this information is needed"), { target: { value: "The filing approval route is incomplete without the signatory." } });
    fireEvent.click(screen.getByRole("button", { name: "Add missing item" }));
    await waitFor(() => expect(changeMatterContext).toHaveBeenCalledWith("matter-1", 7, expect.objectContaining({ kind: "ADD_MISSING", label: "approved signatory" })));

    fireEvent.click(screen.getByRole("button", { name: "Change issue owner" }));
    fireEvent.change(screen.getByLabelText("New issue owner"), { target: { value: "owner-2" } });
    fireEvent.change(screen.getByLabelText("Reason for reassignment"), { target: { value: "Assign the current Privacy Operations owner." } });
    fireEvent.click(screen.getByRole("button", { name: "Assign issue owner" }));
    await waitFor(() => expect(assignMatter).toHaveBeenCalledWith("matter-1", 8, "owner-2", "Assign the current Privacy Operations owner."));
  });

  it("keeps permission-limited information visible without edit controls", async () => {
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    expect(await screen.findByText("licensed DPCO")).toBeTruthy();
    expect(screen.getByLabelText("Current responsibility and timing").textContent).toContain("Assigned to Program Owner");
    expect(screen.queryByRole("button", { name: "Edit Filing Channel" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Change issue owner" })).toBeNull();
  });

  it("preserves entered details and offers a reload after a version conflict", async () => {
    vi.mocked(loadMatterOperations).mockResolvedValue(ownerOperations);
    vi.mocked(updateMatterDetails).mockRejectedValue(new ApiError(409, "version conflict", "version_conflict"));
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Edit issue details" }));
    fireEvent.change(screen.getByLabelText("Summary"), { target: { value: "Keep this carefully entered summary." } });
    fireEvent.change(screen.getByLabelText("Reason for this change"), { target: { value: "Clarify the filing work." } });
    fireEvent.click(screen.getByRole("button", { name: "Save issue details" }));

    expect(await screen.findByText("This issue changed since you opened it. Your entries have been kept.")).toBeTruthy();
    expect((screen.getByLabelText("Summary") as HTMLTextAreaElement).value).toBe("Keep this carefully entered summary.");
    fireEvent.click(screen.getByRole("button", { name: "Reload current issue" }));
    await waitFor(() => expect(loadMatter).toHaveBeenCalledTimes(2));
  });

  it("shows each Action owner and deadline and lets the accountable owner add assigned work", async () => {
    vi.mocked(loadMatterOperations).mockResolvedValue(actionOwnerOperations);
    vi.mocked(addMatterAction).mockResolvedValue({ ...detail, matter: { ...detail.matter, version: 8 }, actions: [...detail.actions, { id: "action-2", title: "Confirm section owners", description: "Record the two remaining owners.", owner_principal_id: "owner-2", status: "PLANNED", due_at: "2026-09-02T00:00:00.000Z" }] });
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    expect(await screen.findByText("Program Owner", { selector: ".matter-action-meta strong" })).toBeTruthy();
    expect(screen.getByText("Action due 26 Aug 2026")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Add action" }));
    fireEvent.change(screen.getByLabelText("Action title"), { target: { value: "Confirm section owners" } });
    fireEvent.change(screen.getByLabelText("Action description"), { target: { value: "Record the two remaining owners." } });
    fireEvent.change(screen.getByLabelText("Action owner"), { target: { value: "owner-2" } });
    fireEvent.change(screen.getByLabelText("Action due date"), { target: { value: "2026-09-02" } });
    fireEvent.click(screen.getByRole("button", { name: "Create assigned action" }));

    await waitFor(() => expect(addMatterAction).toHaveBeenCalledWith("matter-1", 7, {
      title: "Confirm section owners", description: "Record the two remaining owners.", ownerPrincipalID: "owner-2", dueAt: "2026-09-02T00:00:00.000Z",
    }));
    expect(await screen.findByText("Action added.")).toBeTruthy();
  });

  it("lets the accountable owner edit and reassign an Action", async () => {
    vi.mocked(loadMatterOperations).mockResolvedValue(actionOwnerOperations);
    vi.mocked(updateMatterAction).mockResolvedValue({ ...detail, matter: { ...detail.matter, version: 8 }, actions: [{ ...detail.actions[0]!, description: "Map every section to its approved source." }] });
    vi.mocked(assignMatterAction).mockResolvedValue({ ...detail, matter: { ...detail.matter, version: 9 }, actions: [{ ...detail.actions[0]!, owner_principal_id: "owner-2" }] });
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Edit Update the annual return evidence checklist" }));
    fireEvent.change(screen.getByLabelText("Action description"), { target: { value: "Map every section to its approved source." } });
    fireEvent.change(screen.getByLabelText("Reason for changing this action"), { target: { value: "Clarify the evidence required for completion." } });
    fireEvent.click(screen.getByRole("button", { name: "Save action" }));
    await waitFor(() => expect(updateMatterAction).toHaveBeenCalledWith("matter-1", "action-1", 7, expect.objectContaining({ description: "Map every section to its approved source." })));

    fireEvent.click(screen.getByRole("button", { name: "Change owner for Update the annual return evidence checklist" }));
    fireEvent.change(screen.getByLabelText("New action owner"), { target: { value: "owner-2" } });
    fireEvent.change(screen.getByLabelText("Reason for action reassignment"), { target: { value: "Assign the process owner who maintains the evidence." } });
    fireEvent.click(screen.getByRole("button", { name: "Assign action owner" }));
    await waitFor(() => expect(assignMatterAction).toHaveBeenCalledWith("matter-1", "action-1", 8, "owner-2", "Assign the process owner who maintains the evidence."));
  });

  it("lets only the current performer update Action status and keeps the outcome pending after implementation", async () => {
    vi.mocked(loadMatterOperations).mockResolvedValue(performerOperations);
    vi.mocked(transitionMatterAction).mockResolvedValue({ ...detail, matter: { ...detail.matter, version: 8 }, actions: [{ ...detail.actions[0]!, status: "IMPLEMENTED", implemented_at: "2026-08-25T12:00:00Z" }] });
    render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Update status for Update the annual return evidence checklist" }));
    fireEvent.change(screen.getByLabelText("Next action status"), { target: { value: "IMPLEMENTED" } });
    fireEvent.click(screen.getByRole("button", { name: "Update action status" }));

    await waitFor(() => expect(transitionMatterAction).toHaveBeenCalledWith("matter-1", "action-1", 7, "IMPLEMENTED", ""));
    expect(await screen.findByText("Work completed; outcome not confirmed")).toBeTruthy();
    expect(screen.getByText("No outcome check has been defined")).toBeTruthy();
  });
});
