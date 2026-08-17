import { act, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import type { MatterAggregate, ProgramAggregate } from "../types";
import { MattersWorkspace } from "./MattersWorkspace";
import { ProgramsWorkspace } from "./ProgramsWorkspace";
import { loadMatter, loadMatterSummaries, loadProgram, loadProgramSummaries } from "../api";

vi.mock("../api", () => ({
  loadMatter: vi.fn(),
  loadMatterSummaries: vi.fn(),
  loadProgram: vi.fn(),
  loadProgramSummaries: vi.fn(),
}));

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

    render(<MattersWorkspace targetID={matterDetail.matter.id}/>);

    expect(await screen.findByText("Matter outside first page")).toBeTruthy();
    expect(await screen.findByRole("heading", { name: "Current handoff" })).toBeTruthy();
    expect(loadMatter).toHaveBeenCalledWith(matterDetail.matter.id);
  });

  it("clears delayed target scrolling when the Matter workspace unmounts", async () => {
    vi.useFakeTimers();
    try {
      vi.mocked(loadMatterSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-06T10:00:00Z" });
      vi.mocked(loadMatter).mockResolvedValue(matterDetail);

      const view = render(<MattersWorkspace targetID={matterDetail.matter.id}/>);
      await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
        await Promise.resolve();
      });

      expect(screen.getByText("Matter outside first page")).toBeTruthy();
      expect(vi.getTimerCount()).toBeGreaterThan(0);
      view.unmount();
      expect(vi.getTimerCount()).toBe(0);
    } finally {
      vi.useRealTimers();
    }
  });
});
