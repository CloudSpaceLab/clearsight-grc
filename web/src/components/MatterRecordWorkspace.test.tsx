import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadMatter } from "../api";
import { loadMatterOperations } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import { MatterRecordWorkspace } from "./MatterRecordWorkspace";

vi.mock("../api", () => ({ loadMatter: vi.fn() }));
vi.mock("../matterOperationsApi", () => ({ loadMatterOperations: vi.fn() }));

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

const operations = {
  matter_id: "matter-1", matter_version: 7, authority_available: true, generated_at: "2026-08-25T09:00:00Z",
  operations: [
    { command: "matter.details.update", label: "Edit issue details", responsibility: "ACCOUNTABLE_OWNER", can_act: false, assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" }, reason: "This change is assigned to the Program Owner." },
    { command: "matter.context.change", label: "Update facts and missing information", responsibility: "ACCOUNTABLE_OWNER", can_act: false, assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" }, reason: "This change is assigned to the Program Owner." },
    { command: "matter.action.transition", subresource_id: "action-1", label: "Update action", responsibility: "PERFORMER", can_act: false, assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" }, reason: "This action is assigned to the Program Owner.", allowed_targets: ["IMPLEMENTED", "BLOCKED"] },
  ],
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
    expect(screen.getByText("Assigned to Program Owner")).toBeTruthy();
    expect(screen.getByText("Due 26 Aug 2026")).toBeTruthy();
    expect(screen.getByText("2 missing information items")).toBeTruthy();
    expect(screen.getByText("final DPCO evidence checklist")).toBeTruthy();
    expect(screen.getAllByText("This action is assigned to the Program Owner.")).toHaveLength(2);
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
});
