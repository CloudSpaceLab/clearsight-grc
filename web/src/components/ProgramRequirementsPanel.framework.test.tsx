import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { decideProgramFrameworkMapping, loadFrameworkClauseCandidates, loadProgramFrameworkMappings, proposeProgramFrameworkMapping } from "../programFrameworkCandidatesApi";
import type { ProgramAggregate } from "../types";
import { ProgramRequirementsPanel } from "./ProgramRequirementsPanel";

vi.mock("../programFrameworkCandidatesApi", () => ({
  loadFrameworkClauseCandidates: vi.fn(),
  loadProgramFrameworkMappings: vi.fn(),
  proposeProgramFrameworkMapping: vi.fn(),
  decideProgramFrameworkMapping: vi.fn(),
}));

const aggregate = {
  program: { id: "program-1", tenant_id: "bank", name: "Security", version: 4 },
  framework_adoptions: [{
    id: "adoption-1", edition_id: "edition-1", publisher_code: "EXAMPLE",
    framework_code: "SECURITY", edition_code: "2026", status: "ACTIVE",
    proposed_by: "owner", rationale: "Adopted", proposed_at: "2026-10-15T09:00:00Z", version: 2,
  }],
  requirements: [{ id: "requirement-1", code: "R1", title: "Access review", statement: "Review access",
    status: "APPROVED", source_anchor: "Section 1", effective_from: "2026-10-01T00:00:00Z" }],
  applicability: [],
} as unknown as ProgramAggregate;

const firstPage = {
  program_id: "program-1", program_version: 4, adoption_id: "adoption-1", edition_id: "edition-1",
  next_after_code: "A.1", candidates: [
    { clause_id: "clause-1", clause_code: "A.1", source_anchor: "Section 1", status: "REVIEW_CANDIDATE" as const,
      total_candidates: 1, candidates: [{ requirement_id: "requirement-1", requirement_code: "R1",
        title: "Access review", applicability: "PARTIALLY_APPLICABLE" }] },
  ],
};

describe("Program framework clause review", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(loadProgramFrameworkMappings).mockResolvedValue({
      adoption_id: "adoption-1", program_version: 4,
      next_after_id: "", items: [],
    });
  });

  it("shows review-only existing requirement suggestions and bounded next page", async () => {
    vi.mocked(loadFrameworkClauseCandidates).mockResolvedValueOnce(firstPage).mockResolvedValueOnce({
      ...firstPage, next_after_code: "", candidates: [
        { clause_id: "clause-2", clause_code: "B.1", source_anchor: "Section 2", status: "UNMAPPED",
          total_candidates: 0, candidates: [] },
      ],
    });
    render(<ProgramRequirementsPanel aggregate={aggregate} operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Review clauses" }));
    await waitFor(() => expect(screen.getByRole("region", { name: "Clause A.1" })).toBeTruthy());
    expect(screen.getByText("Review suggested match")).toBeTruthy();
    expect(screen.getByText("R1 · Access review · Partially Applicable")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Approve match" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Next clauses" }));
    await waitFor(() => expect(screen.getByRole("region", { name: "Clause B.1" })).toBeTruthy());
    expect(screen.getByText("No matching requirement")).toBeTruthy();
    expect(loadFrameworkClauseCandidates).toHaveBeenNthCalledWith(2, "program-1", "adoption-1", "A.1");
  });


  it("opens the exact active standard's authorized clauses from the Program overview handoff", async () => {
    vi.mocked(loadFrameworkClauseCandidates).mockResolvedValue(firstPage);
    render(<ProgramRequirementsPanel aggregate={aggregate} operations={[]} reviewAdoptionID="adoption-1"
      onUpdated={vi.fn()} onReload={vi.fn()}/>);
    await waitFor(() => expect(screen.getByRole("region", {name: "Clause A.1"})).toBeTruthy());
    expect(screen.getByRole("button", {name: "Hide clauses"})).toBeTruthy();
    expect(loadFrameworkClauseCandidates).toHaveBeenCalledWith("program-1", "adoption-1", "");
    expect(loadProgramFrameworkMappings).toHaveBeenCalledWith("program-1", "adoption-1");
  });

  it("does not trust a stale or unrelated standard ID from navigation", async () => {
    vi.mocked(loadFrameworkClauseCandidates).mockResolvedValue(firstPage);
    const {rerender} = render(<ProgramRequirementsPanel aggregate={aggregate} operations={[]}
      reviewAdoptionID="unknown-adoption" onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(screen.getByRole("button", {name: "Review clauses"})).toBeTruthy();
    expect(loadFrameworkClauseCandidates).not.toHaveBeenCalled();
    expect(loadProgramFrameworkMappings).not.toHaveBeenCalled();
    rerender(<ProgramRequirementsPanel aggregate={{
      ...aggregate, framework_adoptions: aggregate.framework_adoptions?.map((value) => ({...value,status:"PENDING" as const})),
    }} operations={[]} reviewAdoptionID="adoption-1" onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(screen.queryByRole("button", {name: "Review clauses"})).toBeNull();
    expect(loadFrameworkClauseCandidates).not.toHaveBeenCalled();
  });
  it("hides standard review for Programs without an active edition", () => {
    render(<ProgramRequirementsPanel aggregate={{ ...aggregate, framework_adoptions: [
      { id: "adoption-1", edition_id: "edition-1", publisher_code: "EXAMPLE", framework_code: "SECURITY",
        edition_code: "2026", status: "PENDING", proposed_by: "owner", rationale: "Pending",
        proposed_at: "2026-10-15T09:00:00Z", version: 1 },
    ] }} operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(screen.queryByRole("button", { name: "Review clauses" })).toBeNull();
  });

  it("does not report an outdated Program score or accept stale candidate pages", async () => {
    vi.mocked(loadFrameworkClauseCandidates).mockResolvedValue({
      ...firstPage, program_version: 3,
    });
    render(<ProgramRequirementsPanel aggregate={aggregate} operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Review clauses" }));
    await waitFor(() => expect(screen.getByRole("alert").textContent).toContain("Clause matches could not be loaded."));
    expect(screen.queryByText("Review suggested match")).toBeNull();
  });
});

describe("Program framework mapping review", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(loadFrameworkClauseCandidates).mockResolvedValue(firstPage);
    vi.mocked(loadProgramFrameworkMappings).mockResolvedValue({
      adoption_id: "adoption-1", program_version: 4, next_after_id: "", items: [],
    });
  });

  it("proposes an existing Requirement link without creating a new obligation", async () => {
    const onUpdated = vi.fn();
    vi.mocked(proposeProgramFrameworkMapping).mockResolvedValue({
      mapping: {
        id: "map-1",clause_id:"clause-1",adoption_id:"adoption-1",requirement_id:"requirement-1",
        mapping_kind:"PARTIAL",status:"PENDING",version:1,mapped_by:"owner",rationale:"Limited mapping",
      },
      program: { ...aggregate, program: { ...aggregate.program, version: 5 } },
    });
    render(<ProgramRequirementsPanel aggregate={aggregate} operations={[{
      command: "program.framework.mapping.propose",label:"Propose link",
      can_act: true,responsibility:"OWNER",reason:"",
    }]} onUpdated={onUpdated} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button",{name:"Review clauses"}));
    await waitFor(() => expect(screen.getByRole("button",{name:"Propose link"})).toBeTruthy());
    fireEvent.click(screen.getByRole("button",{name:"Propose link"}));
    fireEvent.change(screen.getByLabelText("Mapping"),{target:{value:"PARTIAL"}});
    fireEvent.change(screen.getByLabelText("Link rationale"),{target:{value:"Limited mapping"}});
    fireEvent.click(screen.getByRole("button",{name:"Submit link for approval"}));
    await waitFor(() => expect(proposeProgramFrameworkMapping).toHaveBeenCalledWith(
      "program-1","adoption-1",4,"clause-1","requirement-1","PARTIAL","Limited mapping",
    ));
    expect(onUpdated).toHaveBeenCalled();
  });

  it("recovers a pending mapping after reload and enforces reviewer operations", async () => {
    const onUpdated = vi.fn();
    vi.mocked(loadProgramFrameworkMappings).mockResolvedValue({
      adoption_id:"adoption-1",program_version:4,next_after_id:"",
      items:[{
        id:"mapping-1",clause_id:"clause-1",clause_code:"A.1",
        requirement_id:"requirement-1",adoption_id:"adoption-1",
        mapping_kind:"PARTIAL",rationale:"Limited",mapped_by:"owner",
        status:"PENDING",version:1,
      }],
    });
    vi.mocked(decideProgramFrameworkMapping).mockResolvedValue({
      mapping: {
        id:"mapping-1",clause_id:"clause-1",adoption_id:"adoption-1",requirement_id:"requirement-1",
        mapping_kind:"PARTIAL",status:"APPROVED",version:2,mapped_by:"owner",reviewed_by:"checker",
        rationale:"Limited",
      },
      program: {...aggregate,program:{...aggregate.program,version:5}},
    });
    const reviewer={command:"program.framework.mapping.approve",label:"Approve link",can_act:true,responsibility:"AUTHORIZER",reason:""};
    render(<ProgramRequirementsPanel aggregate={aggregate} operations={[reviewer]} onUpdated={onUpdated} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button",{name:"Review clauses"}));
    await waitFor(() => expect(screen.getByText("Links awaiting review")).toBeTruthy());
    expect(screen.queryByRole("button",{name:"Propose link"})).toBeNull();
    fireEvent.click(screen.getByRole("button",{name:"Review link"}));
    fireEvent.change(screen.getByLabelText("Decision rationale for A.1"),{target:{value:"Mapped to approved obligation"}});
    fireEvent.click(screen.getByRole("button",{name:"Approve link"}));
    await waitFor(() => expect(decideProgramFrameworkMapping).toHaveBeenCalledWith(
      "program-1","adoption-1","mapping-1",4,"approve","Mapped to approved obligation",
    ));
    expect(onUpdated).toHaveBeenCalled();
  });
  it("keeps reviewer rationale scoped to the selected mapping", async () => {
    vi.mocked(loadProgramFrameworkMappings).mockResolvedValue({
      adoption_id:"adoption-1",program_version:4,next_after_id:"",
      items:[
        {id:"mapping-1",clause_id:"clause-1",clause_code:"A.1",requirement_id:"requirement-1",adoption_id:"adoption-1",mapping_kind:"EXACT",rationale:"",mapped_by:"owner",status:"PENDING",version:1},
        {id:"mapping-2",clause_id:"clause-2",clause_code:"A.2",requirement_id:"requirement-1",adoption_id:"adoption-1",mapping_kind:"PARTIAL",rationale:"",mapped_by:"owner",status:"PENDING",version:1},
      ],
    });
    render(<ProgramRequirementsPanel aggregate={aggregate} operations={[{
      command:"program.framework.mapping.approve", label:"Approve link",
      can_act:true,responsibility:"AUTHORIZER",reason:"",
    }]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button",{name:"Review clauses"}));
    await waitFor(() => expect(screen.getAllByRole("button",{name:"Review link"})).toHaveLength(2));
    fireEvent.click(screen.getAllByRole("button",{name:"Review link"})[0]!);
    fireEvent.change(screen.getByLabelText("Decision rationale for A.1"),{target:{value:"Specific to A.1"}});
    fireEvent.click(screen.getByRole("button",{name:"Cancel"}));
    fireEvent.click(screen.getAllByRole("button",{name:"Review link"})[1]!);
    expect((screen.getByLabelText("Decision rationale for A.2") as HTMLTextAreaElement).value).toBe("");
    expect(screen.getByRole("button",{name:"Approve link"}).hasAttribute("disabled")).toBe(true);
    expect(decideProgramFrameworkMapping).not.toHaveBeenCalled();
  });

});
