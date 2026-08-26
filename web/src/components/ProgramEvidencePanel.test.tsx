import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadEvidenceSources } from "../api";
import { addProgramEvidenceContract, recordProgramEvidenceAssessment } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";
import { ProgramEvidencePanel } from "./ProgramEvidencePanel";

vi.mock("../api", () => ({ loadEvidenceSources: vi.fn() }));
vi.mock("../programOperationsApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../programOperationsApi")>()),
  addProgramEvidenceContract: vi.fn(),
  recordProgramEvidenceAssessment: vi.fn(),
}));
vi.mock("./MonitoringSetup", () => ({ MonitoringSetup: () => <section><h3>Monitoring</h3></section> }));

const aggregate: ProgramAggregate = {
  state_label: "Evidence incomplete",
  program: { id: "program-1", tenant_id: "bank", code: "PRIVACY", name: "Privacy", type: "REGULATORY", status: "ACTIVE", owning_function: "Compliance", scope: {}, effective_from: "2026-01-01T00:00:00Z", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z", version: 4 },
  requirements: [{ id: "requirement-1", code: "REQ-1", title: "File return", statement: "File the return.", status: "APPROVED" }],
  applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [],
  evidence_contracts: [{ id: "contract-1", requirement_id: "requirement-1", code: "EVIDENCE", name: "Filing evidence", claim: "The return was filed.", acceptable_source_ids: [], status: "ACTIVE", freshness_minutes: 1440, minimum_coverage: 1, independence_required: false, contradiction_policy: "REVIEW", failure_action: "MATTER" }],
  evidence_assessments: [], triggers: [],
};
const operations = [
  { command: "program.evidence.define", label: "Define evidence", responsibility: "OWNER", can_act: true, reason: "You can define evidence." },
  { command: "program.evidence.assess", label: "Assess evidence", responsibility: "REVIEWER", can_act: true, reason: "You can assess evidence." },
];

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadEvidenceSources).mockResolvedValue([]);
});

describe("Program evidence authority gating", () => {
  it.each([
    ["Define evidence check", "Save evidence check"],
    ["Record evidence result", "Save evidence result"],
  ])("closes the %s form without submitting when responsibility readiness is lost", async (openName, saveName) => {
    const view = render(<ProgramEvidencePanel aggregate={aggregate} operations={operations} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(await screen.findByRole("button", { name: openName }));
    expect(screen.getByRole("button", { name: saveName })).toBeTruthy();

    view.rerender(<ProgramEvidencePanel aggregate={aggregate} operations={operations} actorPrincipalID="actor-1" canConfigureSources={false} canOperate={false} onUpdated={vi.fn()} onReload={vi.fn()}/>);

    expect(screen.queryByRole("button", { name: saveName })).toBeNull();
    expect(screen.queryByRole("button", { name: openName })).toBeNull();
    expect(screen.getByText("Evidence changes are disabled until current Program responsibilities are available. Existing evidence checks and results remain visible.")).toBeTruthy();
    expect(addProgramEvidenceContract).not.toHaveBeenCalled();
    expect(recordProgramEvidenceAssessment).not.toHaveBeenCalled();
  });
});
