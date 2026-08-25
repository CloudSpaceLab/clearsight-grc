import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadProgram, loadProgramSummaries } from "../api";
import { loadProgramOperations } from "../programOperationsApi";
import { loadProgramReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";
import { ProgramRecordWorkspace } from "./ProgramRecordWorkspace";
import { ProgramsWorkspace } from "./ProgramsWorkspace";

vi.mock("../api", () => ({ loadProgram: vi.fn(), loadProgramSummaries: vi.fn() }));
vi.mock("../programOperationsApi", () => ({ loadProgramOperations: vi.fn() }));
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
  });

  it("shows owner, calculated-state freshness, reasons and one dominant action", async () => {
    render(<ProgramRecordWorkspace programID="program-1" onBack={vi.fn()}/>);

    expect(await screen.findByRole("heading", { name: "Nigeria data protection" })).toBeTruthy();
    expect(screen.getByText("Data Protection Officer")).toBeTruthy();
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
});
