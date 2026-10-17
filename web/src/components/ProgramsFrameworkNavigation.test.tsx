import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadProgramSummaries } from "../api";
import { ProgramsWorkspace } from "./ProgramsWorkspace";

vi.mock("../api", () => ({ loadProgramSummaries: vi.fn() }));
vi.mock("./FrameworkLibraryWorkspace", () => ({
  FrameworkLibraryWorkspace: ({onBack, onChooseProgram}: {
    onBack: () => void;
    onChooseProgram: (edition: {id: string; jurisdiction: string}) => void;
  }) => <>
    <button onClick={onBack}>Return to Programs</button>
    <button onClick={() => onChooseProgram({
      id: "0199f39a-dac6-7be2-a4ca-1f6d2c7d6001", jurisdiction: "GLOBAL",
    })}>Select published edition</button>
  </>,
}));
vi.mock("./ProgramRecordWorkspace", () => ({
  ProgramRecordWorkspace: ({onBack}: {onBack: () => void}) =>
    <button onClick={onBack}>Back to Program register</button>,
}));

const selection = {
  editionID: "0199f39a-dac6-7be2-a4ca-1f6d2c7d6001",
  jurisdiction: "GLOBAL",
};

describe("Framework Library to Program route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.location.hash = "#programs";
  });

  it("opens the library without loading a Program or widening Program scope", () => {
    render(<ProgramsWorkspace programsPage="library"/>);
    expect(screen.getByRole("button", {name: "Select published edition"})).toBeTruthy();
    expect(loadProgramSummaries).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", {name: "Select published edition"}));
    expect(window.location.hash).toBe("#programs?edition_id=0199f39a-dac6-7be2-a4ca-1f6d2c7d6001&edition_jurisdiction=GLOBAL");
  });

  it("carries an edition through the existing bounded Program register", async () => {
    vi.mocked(loadProgramSummaries).mockResolvedValue({
      generated_at: "2026-10-17T00:00:00Z",
      items: [{
        program: {
          id: "program-1", tenant_id: "bank", name: "Information security", code: "ISMS-001",
          type: "ASSURANCE", scope: {}, effective_from: "2026-01-01T00:00:00Z",
          created_at: "2026-01-01T00:00:00Z", updated_at: "2026-10-17T00:00:00Z",
          owning_function: "Risk", status: "ACTIVE", version: 3,
        },
        state_label: "Up to date", overall_state: "CURRENT",
        program_version: 3, assessed_program_version: 3, projection_stale: false,
        requirement_count: 1, safeguard_count: 0, evidence_check_count: 1, open_matter_count: 0,
        reasons: [], reasons_total: 0, reasons_omitted: 0, projection_version: 1,
      }],
    });
    render(<ProgramsWorkspace frameworkEdition={selection}/>);
    const link = await screen.findByRole("link", {name: /Information security/});
    expect(link.getAttribute("href")).toContain("edition_id=0199f39a-dac6-7be2-a4ca-1f6d2c7d6001");
    expect(link.getAttribute("href")).toContain("edition_jurisdiction=GLOBAL");
    expect(screen.getByLabelText("Selected framework edition").textContent).toContain("Choose a Program");
  });

  it("preserves the selected edition when returning from a Program record", () => {
    window.location.hash = "#programs/program-1/overview?q=security&edition_id=0199f39a-dac6-7be2-a4ca-1f6d2c7d6001&edition_jurisdiction=GLOBAL";
    render(<ProgramsWorkspace targetID="program-1" frameworkEdition={selection}/>);
    fireEvent.click(screen.getByRole("button", {name: "Back to Program register"}));
    expect(window.location.hash).toContain("#programs?");
    expect(window.location.hash).toContain("edition_id=0199f39a-dac6-7be2-a4ca-1f6d2c7d6001");
    expect(window.location.hash).toContain("q=security");
  });
});
