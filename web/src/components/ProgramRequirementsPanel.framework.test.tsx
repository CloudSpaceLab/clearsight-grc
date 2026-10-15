import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadFrameworkClauseCandidates } from "../programFrameworkCandidatesApi";
import type { ProgramAggregate } from "../types";
import { ProgramRequirementsPanel } from "./ProgramRequirementsPanel";

vi.mock("../programFrameworkCandidatesApi", () => ({
  loadFrameworkClauseCandidates: vi.fn(),
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
    { clause_code: "A.1", source_anchor: "Section 1", status: "REVIEW_CANDIDATE" as const,
      total_candidates: 1, candidates: [{ requirement_id: "requirement-1", requirement_code: "R1",
        title: "Access review", applicability: "PARTIALLY_APPLICABLE" }] },
  ],
};

describe("Program framework clause review", () => {
  beforeEach(() => vi.resetAllMocks());

  it("shows review-only existing requirement suggestions and bounded next page", async () => {
    vi.mocked(loadFrameworkClauseCandidates).mockResolvedValueOnce(firstPage).mockResolvedValueOnce({
      ...firstPage, next_after_code: "", candidates: [
        { clause_code: "B.1", source_anchor: "Section 2", status: "UNMAPPED",
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
