import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import type { MatterAggregate, ProgramAggregate } from "../types";
import { MattersWorkspace } from "./MattersWorkspace";
import { ProgramsWorkspace } from "./ProgramsWorkspace";
import { loadMatter, loadMatterSummaries, loadProgram, loadProgramSummaries } from "../api";
import { createMatter } from "../continuityCommands";
import { loadMatterOperations } from "../matterOperationsApi";

vi.mock("../api", () => ({
  loadMatter: vi.fn(),
  loadMatterSummaries: vi.fn(),
  loadProgram: vi.fn(),
  loadProgramSummaries: vi.fn(),
}));

vi.mock("../continuityCommands", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../continuityCommands")>()),
  createMatter: vi.fn(),
}));

vi.mock("../matterOperationsApi", () => ({ loadMatterOperations: vi.fn() }));

beforeAll(() => {
  Object.defineProperty(Element.prototype, "scrollIntoView", { configurable: true, value: vi.fn() });
});

const programDetail: ProgramAggregate = {
  state_label: "Evidence incomplete",
  program: {
    id: "program-outside-page", tenant_id: "bank-demo", code: "OUTSIDE", name: "Program outside first page", type: "REGULATORY", status: "ACTIVE", owning_function: "Compliance", scope: {}, effective_from: "2026-01-01T00:00:00Z", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-06T10:00:00Z", version: 2,
  },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
  current_state: { id: "state-1", overall_state: "EVIDENCE_INSUFFICIENT", dimensions: {}, reasons: [{ code: "MISSING", summary: "Evidence is missing." }], open_matter_count: 1, generated_at: "2026-08-06T10:00:00Z", program_version: 2, projection_version: 4 },
};

const matterDetail: MatterAggregate = {
  type_label: "Finding",
  status_label: "Decision needed",
  next_action: "Choose a treatment",
  matter: {
    id: "matter-outside-page", tenant_id: "bank-demo", reference: "FND-99", type: "FINDING", status: "DECISION_REQUIRED", priority: 4, title: "Matter outside first page", summary: "A material issue needs a decision.", scope: {}, known_facts: {}, missing_facts: [], contradictions: [], created_at: "2026-08-01T10:00:00Z", updated_at: "2026-08-06T10:00:00Z", version: 2,
  },
  links: [], decisions: [], actions: [], verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: [] },
};

describe("exact workspace targets", () => {
  it("renders a directly fetched Program even when it is outside the first summary page", async () => {
    vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-06T10:00:00Z" });
    vi.mocked(loadProgram).mockResolvedValue(programDetail);

    render(<ProgramsWorkspace targetID={programDetail.program.id}/>);

    expect(await screen.findByText("Program outside first page")).toBeTruthy();
    expect(await screen.findByRole("heading", { name: "Why this status" })).toBeTruthy();
    expect(loadProgram).toHaveBeenCalledWith(programDetail.program.id);
  });

  it("renders a directly fetched Matter even when it is outside the first summary page", async () => {
    vi.mocked(loadMatterSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-06T10:00:00Z" });
    vi.mocked(loadMatter).mockResolvedValue(matterDetail);
    vi.mocked(loadMatterOperations).mockResolvedValue({ matter_id: matterDetail.matter.id, matter_version: 2, authority_available: true, operations: [], generated_at: "2026-08-06T10:00:00Z" });

    render(<MattersWorkspace targetID={matterDetail.matter.id}/>);

    expect(await screen.findByText("Matter outside first page")).toBeTruthy();
    expect(await screen.findByRole("heading", { name: "Current handoff" })).toBeTruthy();
    expect(loadMatter).toHaveBeenCalledWith(matterDetail.matter.id);
  });

  it("does not leave delayed target scrolling after the dedicated Matter workspace unmounts", async () => {
    vi.useFakeTimers();
    try {
      vi.mocked(loadMatterSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-06T10:00:00Z" });
      vi.mocked(loadMatter).mockResolvedValue(matterDetail);
      vi.mocked(loadMatterOperations).mockResolvedValue({ matter_id: matterDetail.matter.id, matter_version: 2, authority_available: true, operations: [], generated_at: "2026-08-06T10:00:00Z" });

      const view = render(<MattersWorkspace targetID={matterDetail.matter.id}/>);
      await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
        await Promise.resolve();
      });

      expect(screen.getByText("Matter outside first page")).toBeTruthy();
      view.unmount();
      expect(vi.getTimerCount()).toBe(0);
    } finally {
      vi.useRealTimers();
    }
  });

  it("creates and opens an issue or change from the empty workspace", async () => {
    const created = { ...matterDetail, matter: { ...matterDetail.matter, id: "matter-created", title: "Mobile banking control gap", type: "CONTROL_GAP", status: "DRAFT" } };
    vi.mocked(loadMatterSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-17T10:00:00Z" });
    vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-17T10:00:00Z" });
    vi.mocked(createMatter).mockResolvedValue(created);

    render(<MattersWorkspace/>);
    fireEvent.click(await screen.findByRole("button", { name: "Create issue or change" }));
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Mobile banking control gap" } });
    fireEvent.change(screen.getByLabelText("What happened or changed?"), { target: { value: "Face verification did not return a successful result." } });
    fireEvent.change(screen.getByLabelText("Affected area"), { target: { value: "Mobile banking" } });
    fireEvent.click(screen.getByRole("button", { name: "Create issue or change" }));

    await waitFor(() => expect(createMatter).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("Issue or change created.")).toBeTruthy();
    expect(screen.getByText("Mobile banking control gap")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Current handoff" })).toBeTruthy();
  });
});
