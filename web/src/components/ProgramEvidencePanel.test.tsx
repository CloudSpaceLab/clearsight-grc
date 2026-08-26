import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadEvidenceSources } from "../api";
import { addProgramEvidenceContract, recordProgramEvidenceAssessment, reviseProgramEvidenceContract, transitionProgramEvidenceContract } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";
import { ProgramEvidencePanel } from "./ProgramEvidencePanel";

vi.mock("../api", () => ({ loadEvidenceSources: vi.fn() }));
vi.mock("../programOperationsApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../programOperationsApi")>()),
  addProgramEvidenceContract: vi.fn(),
  recordProgramEvidenceAssessment: vi.fn(),
  reviseProgramEvidenceContract: vi.fn(),
  transitionProgramEvidenceContract: vi.fn(),
}));
vi.mock("./MonitoringSetup", () => ({ MonitoringSetup: () => <section><h3>Monitoring</h3></section> }));

const aggregate: ProgramAggregate = {
  state_label: "Evidence incomplete",
  program: { id: "program-1", tenant_id: "bank", legal_entity_id: "entity-1", code: "PRIVACY", name: "Privacy", type: "REGULATORY", status: "ACTIVE", owning_function: "Compliance", scope: {}, effective_from: "2026-01-01T00:00:00Z", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z", version: 4 },
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
  it("offers only backend-supported conclusions and executable failure handling", async () => {
    render(<ProgramEvidencePanel aggregate={aggregate} operations={operations} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Record evidence result" }));
    const conclusion = screen.getByLabelText("Conclusion") as HTMLSelectElement;
    expect(Array.from(conclusion.options).map((option) => [option.value, option.text])).toEqual([
      ["SUPPORTED", "Supported"],
      ["PARTIALLY_SUPPORTED", "Partly supported"],
      ["UNSUPPORTED", "Unsupported"],
      ["CONTRADICTED", "Contradicted"],
      ["INDETERMINATE", "Indeterminate"],
      ["EXPIRED", "Expired"],
    ]);

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Define evidence check" }));
    expect(screen.getByText("Create a linked issue when a result does not support the claim.")).toBeTruthy();
    const contradiction = screen.getByLabelText("Contradiction handling") as HTMLSelectElement;
    expect(Array.from(contradiction.options).map((option) => [option.value, option.text])).toEqual([
      ["HOLD", "Hold the check"], ["REVIEW", "Require review"], ["FAIL", "Fail the check"],
    ]);
    expect(screen.queryByRole("option", { name: "Escalate" })).toBeNull();
  });

  it("loads source choices for the Program's exact legal entity", async () => {
    render(<ProgramEvidencePanel aggregate={aggregate} operations={operations} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(await screen.findByText("No evidence result recorded")).toBeTruthy();
    expect(loadEvidenceSources).toHaveBeenCalledWith("entity-1");
  });

  it("creates draft evidence checks and exposes owner revision and reviewer status actions", async () => {
    const draftAggregate = { ...aggregate, evidence_contracts: [{ ...aggregate.evidence_contracts[0]!, status: "DRAFT", version: 1 }] };
    const lifecycleOperations = [
      ...operations,
      { command: "program.evidence.revise", subresource_id: "contract-1", label: "Edit evidence", responsibility: "OWNER", can_act: true, reason: "You can edit this draft." },
      { command: "program.evidence.transition", subresource_id: "contract-1", label: "Review evidence status", responsibility: "REVIEWER", can_act: true, reason: "You can review this check.", allowed_targets: ["ACTIVE", "RETIRED"] },
    ];
    vi.mocked(addProgramEvidenceContract).mockResolvedValue(draftAggregate);
    vi.mocked(reviseProgramEvidenceContract).mockResolvedValue(draftAggregate);
    vi.mocked(transitionProgramEvidenceContract).mockResolvedValue(draftAggregate);
    render(<ProgramEvidencePanel aggregate={draftAggregate} operations={lifecycleOperations} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Define evidence check" }));
    fireEvent.change(screen.getByLabelText("Evidence code"), { target: { value: "CHECK-2" } });
    fireEvent.change(screen.getByLabelText("Evidence check name"), { target: { value: "Filing receipt" } });
    fireEvent.change(screen.getByLabelText("What must the evidence prove?"), { target: { value: "The filing was accepted." } });
    fireEvent.click(screen.getByRole("button", { name: "Save evidence check" }));
    expect(addProgramEvidenceContract).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ status: "DRAFT" }));

    expect(screen.getByRole("button", { name: "Edit Filing evidence" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Review Filing evidence status" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Record evidence result" })).toBeNull();
    expect(screen.getByText(/reviewer must activate a Draft evidence check first/)).toBeTruthy();
  });

  it("explains that editing an active check requires reviewer reactivation", async () => {
    render(<ProgramEvidencePanel aggregate={aggregate} operations={[...operations, { command: "program.evidence.revise", subresource_id: "contract-1", label: "Edit evidence", responsibility: "OWNER", can_act: true, reason: "You can edit this check." }]} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit Filing evidence" }));
    expect(screen.getByText(/returns this evidence check to Draft/)).toBeTruthy();
    expect(screen.getByText(/current reviewer must activate it again/)).toBeTruthy();
  });

  it("keeps manual evidence definition usable when connected sources are unavailable", async () => {
    vi.mocked(loadEvidenceSources).mockRejectedValue(new Error("source registry unavailable"));
    vi.mocked(addProgramEvidenceContract).mockResolvedValue(aggregate);
    render(<ProgramEvidencePanel aggregate={aggregate} operations={operations} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Define evidence check" }));
    expect(await screen.findByText(/You can save a manual check without a connected source/)).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Evidence code"), { target: { value: "MANUAL" } });
    fireEvent.change(screen.getByLabelText("Evidence check name"), { target: { value: "Manual review" } });
    fireEvent.change(screen.getByLabelText("What must the evidence prove?"), { target: { value: "A reviewer confirms the retained evidence." } });
    fireEvent.click(screen.getByRole("button", { name: "Save evidence check" }));
    expect(addProgramEvidenceContract).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ acceptableSourceIDs: [], failureAction: "MATTER", status: "DRAFT" }));
  });

  it("shows a bounded labelled evidence result history without exposing reviewer identifiers", async () => {
    vi.mocked(loadEvidenceSources).mockResolvedValue([{ id: "source-1", tenant_id: "bank", code: "REG", name: "Regulatory filing register", type: "REGISTER", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 1440, health: "HEALTHY", status: "ACTIVE", version: 1 }]);
    const assessments = Array.from({ length: 22 }, (_, index) => ({ id: `assessment-${index}`, contract_id: "contract-1", conclusion: index ? "SUPPORTED" : "NOT_SUPPORTED", coverage: .9, basis: { summary: `Result basis ${index}` }, assessed_by: index ? "reviewer-hidden" : "reviewer-1", assessed_at: new Date(Date.UTC(2026, 7, 25 - index)).toISOString() }));
    const value = { ...aggregate, evidence_contracts: [{ ...aggregate.evidence_contracts[0]!, acceptable_source_ids: ["source-1"] }], evidence_assessments: assessments };
    render(<ProgramEvidencePanel aggregate={value} operations={[...operations, { ...operations[1]!, candidates: [{ id: "reviewer-1", display_name: "Ada Okafor", kind: "PERSON", role: "Reviewer" }] }]} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);

    fireEvent.click(screen.getByText("View evidence result history (22)"));
    expect(await screen.findByText(/Showing 20 of 22 stored results for Program version 4/)).toBeTruthy();
    expect(screen.getByText(/assessed 2026-08-25 by Ada Okafor/)).toBeTruthy();
    expect(screen.getAllByText(/Sources: Regulatory filing register/)).toHaveLength(20);
    expect(screen.getByText(/2 additional results/)).toBeTruthy();
    expect(screen.queryByText(/reviewer-hidden/)).toBeNull();
  });

  it("keeps retired source provenance and stored reviewer labels out of active source choices", async () => {
    vi.mocked(loadEvidenceSources).mockResolvedValue([
      { id: "source-active", tenant_id: "bank", code: "LIVE", name: "Current filing register", type: "REGISTER", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 1440, health: "HEALTHY", status: "ACTIVE", version: 2 },
      { id: "source-retired", tenant_id: "bank", code: "OLD", name: "Retired filing register", type: "REGISTER", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 1440, health: "RETIRED", status: "RETIRED", version: 4 },
    ]);
    const value = {
      ...aggregate,
      evidence_contracts: [{ ...aggregate.evidence_contracts[0]!, acceptable_source_ids: ["source-retired"] }],
      evidence_assessments: [{ id: "assessment-1", contract_id: "contract-1", conclusion: "SUPPORTED", coverage: 1, assessed_by: "reviewer-private", assessed_at: "2026-08-25T00:00:00Z" }],
    };
    render(<ProgramEvidencePanel aggregate={value} operations={operations} responsibleParties={[{ scope: "EVIDENCE_ASSESSMENT", subresource_id: "assessment-1", responsibility: "REVIEWER", display_name: "Ada Okafor", kind: "PERSON" }]} actorPrincipalID="actor-1" canConfigureSources canOperate onUpdated={vi.fn()} onReload={vi.fn()}/>);

    expect(await screen.findByText(/Accepted sources: Retired filing register/)).toBeTruthy();
    expect(screen.getByText(/Assessed 2026-08-25 by Ada Okafor/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Define evidence check" }));
    expect(screen.getByRole("checkbox", { name: "Current filing register" })).toBeTruthy();
    expect(screen.queryByRole("checkbox", { name: "Retired filing register" })).toBeNull();
    expect(screen.queryByText("reviewer-private")).toBeNull();
  });

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
