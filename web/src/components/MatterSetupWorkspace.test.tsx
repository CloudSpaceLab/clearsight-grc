import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadProgramSummaries } from "../api";
import { createMatter } from "../continuityCommands";
import type { MatterAggregate } from "../types";
import { MatterSetupWorkspace } from "./MatterSetupWorkspace";

vi.mock("../api", () => ({ loadProgramSummaries: vi.fn() }));
vi.mock("../continuityCommands", () => ({ createMatter: vi.fn() }));

const createdMatter: MatterAggregate = {
  type_label: "Control gap",
  status_label: "Draft",
  next_action: "Complete the initial review",
  matter: {
    id: "matter-new", tenant_id: "bank", reference: "MAT-NEW", type: "CONTROL_GAP", status: "DRAFT", priority: 4,
    title: "Face verification is unavailable", summary: "The mobile channel did not return a successful face-verification result.",
    scope: { access: "INTERNAL", area: "Mobile banking" }, known_facts: { notes: "The public status check failed." },
    missing_facts: ["Confirm the SDK version", "Confirm the last successful check"], contradictions: [], owner_principal_id: "actor-1",
    due_at: "2026-09-30T22:59:59.999Z", created_at: "2026-08-17T10:00:00Z", updated_at: "2026-08-17T10:00:00Z", version: 1,
  },
  links: [{ id: "link-1", program_id: "program-mobile", relationship: "AFFECTS" }], decisions: [], actions: [],
  verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: [] },
};

const programSummary = {
  program: {
    id: "program-mobile", tenant_id: "bank", code: "MOBILE", name: "Mobile banking", type: "CHANNEL", status: "ACTIVE",
    owning_function: "Digital Banking", scope: {}, effective_from: "2026-08-01T00:00:00Z", created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-17T10:00:00Z", version: 3,
  },
  state_label: "Up to date", overall_state: "CURRENT" as const, reasons: [], reasons_total: 0, reasons_omitted: 0,
  open_matter_count: 0, requirement_count: 2, safeguard_count: 2, evidence_check_count: 2, program_version: 3,
  assessed_program_version: 3, projection_version: 3, projection_stale: false, state_generated_at: "2026-08-17T10:00:00Z",
};

describe("MatterSetupWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [programSummary], generated_at: "2026-08-17T10:00:00Z" });
    vi.mocked(createMatter).mockResolvedValue(createdMatter);
  });

  it("uses plain business fields and excludes system-generated work types", async () => {
    render(<MatterSetupWorkspace onCreated={vi.fn()} onClose={vi.fn()}/>);

    expect(screen.getByRole("heading", { name: "New issue or change" })).toBeTruthy();
    expect(screen.getByLabelText("Work type")).toBeTruthy();
    expect(screen.getByLabelText("What happened or changed?")).toBeTruthy();
    expect(screen.queryByRole("option", { name: "Failed verification" })).toBeNull();
    expect(screen.queryByRole("option", { name: "Evidence contradiction" })).toBeNull();
    expect(await screen.findByRole("option", { name: "Mobile banking (MOBILE)" })).toBeTruthy();
  });

  it("creates linked work with safe defaults and structured facts", async () => {
    const onCreated = vi.fn();
    render(<MatterSetupWorkspace onCreated={onCreated} onClose={vi.fn()}/>);

    fireEvent.change(screen.getByLabelText("Work type"), { target: { value: "CONTROL_GAP" } });
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Face verification is unavailable" } });
    fireEvent.change(screen.getByLabelText("What happened or changed?"), { target: { value: "The mobile channel did not return a successful face-verification result." } });
    fireEvent.change(screen.getByLabelText("Affected area"), { target: { value: "Mobile banking" } });
    fireEvent.change(screen.getByLabelText("Priority"), { target: { value: "4" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-09-30" } });
    fireEvent.change(await screen.findByLabelText("Program (optional)"), { target: { value: "program-mobile" } });
    fireEvent.change(screen.getByLabelText("What is already known?"), { target: { value: "The public status check failed." } });
    fireEvent.change(screen.getByLabelText("What information is still needed?"), { target: { value: "Confirm the SDK version\n\nConfirm the last successful check" } });
    fireEvent.click(screen.getByRole("button", { name: "Create issue or change" }));

    await waitFor(() => expect(createMatter).toHaveBeenCalledTimes(1));
    expect(createMatter).toHaveBeenCalledWith(expect.objectContaining({
      type: "CONTROL_GAP", priority: 4, title: "Face verification is unavailable",
      summary: "The mobile channel did not return a successful face-verification result.", affectedArea: "Mobile banking",
      knownInformation: "The public status check failed.", missingInformation: ["Confirm the SDK version", "Confirm the last successful check"],
      programID: "program-mobile",
    }));
    expect(vi.mocked(createMatter).mock.calls[0]![0].dueAt).toMatch(/^2026-09-30T\d{2}:59:59\.999Z$/);
    expect(onCreated).toHaveBeenCalledWith(createdMatter);
  });

  it("keeps entered values when creation fails", async () => {
    vi.mocked(createMatter).mockRejectedValue(new Error("You are not authorized to create this work."));
    render(<MatterSetupWorkspace onCreated={vi.fn()} onClose={vi.fn()}/>);

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Password reset review is overdue" } });
    fireEvent.change(screen.getByLabelText("What happened or changed?"), { target: { value: "The weekly response has not been received." } });
    fireEvent.change(screen.getByLabelText("Affected area"), { target: { value: "Mobile banking" } });
    fireEvent.click(screen.getByRole("button", { name: "Create issue or change" }));

    expect((await screen.findByRole("alert")).textContent).toContain("You are not authorized to create this work.");
    expect((screen.getByLabelText("Title") as HTMLInputElement).value).toBe("Password reset review is overdue");
  });
});
