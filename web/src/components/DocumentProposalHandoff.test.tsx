import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DocumentImport, DocumentProposal } from "../documentTypes";
import { authorizeDocumentProposalHandoff } from "../documentApi";
import { DocumentProposalHandoff } from "./DocumentProposalHandoff";

vi.mock("../api", () => ({
  loadPrograms: vi.fn().mockResolvedValue([{ program: { id: "program-1", code: "P1", name: "Program one", version: 7, legal_entity_id: "entity-a" } }]),
}));

vi.mock("../documentApi", () => ({
  authorizeDocumentProposalHandoff: vi.fn(),
  loadDocumentImport: vi.fn(),
  reviewDocumentProposalHandoff: vi.fn(),
}));

const proposal: DocumentProposal = {
  id: "proposal-1",
  kind: "REQUIREMENT_CANDIDATE",
  title: "Access review",
  statement: "Review access quarterly.",
  confidence: 0.9,
  anchor: { section_id: "section-1", quote: "Review access quarterly." },
  status: "ACCEPTED",
  handoff: {
    id: "handoff-1",
    status: "AWAITING_AUTHORIZATION",
    intake_principal_id: "intake-a",
    reviewer_principal_id: "reviewer-b",
    target_type: "REQUIREMENT",
    target_program_id: "program-1",
    target_program_version: 7,
    draft_title: "Quarterly access review",
    draft_statement: "The bank shall review access quarterly.",
    updated_at: "2026-08-22T14:00:00Z",
    version: 2,
    route: { status: "DIRECT", responsibility: "AUTHORIZER", principal_id: "authorizer-c", is_current_actor: true },
  },
};

const approved: DocumentImport = {
  id: "document-1",
  tenant_id: "tenant-a",
  legal_entity_id: "entity-a",
  file_name: "notice.md",
  media_type: "text/markdown",
  purpose: "Review notice",
  source_type: "REGULATORY",
  size_bytes: 10,
  sha256: "a".repeat(64),
  storage_key: "document-imports/tenant-a/document-1/notice.md",
  artifact_status: "STORED_UNSCANNED",
  extraction_status: "EXTRACTED",
  extraction_method: "PLAIN_TEXT_V2",
  analysis_status: "REVIEW_REQUIRED",
  analysis_method: "DETERMINISTIC_RULES_V2",
  limitations: [],
  sections: [],
  proposals: [{ ...proposal, handoff: { ...proposal.handoff!, status: "APPROVED", authorizer_principal_id: "authorizer-c", authorization_note: "approved", result_object_type: "REQUIREMENT", result_object_id: "requirement-1", version: 3 } }],
  created_by: "intake-a",
  created_at: "2026-08-22T13:00:00Z",
  updated_at: "2026-08-22T14:01:00Z",
  version: 4,
};

describe("DocumentProposalHandoff", () => {
  beforeEach(() => {
    vi.mocked(authorizeDocumentProposalHandoff).mockResolvedValue(approved);
  });

  it("propagates the authoritative document after authorization", async () => {
    const onDocumentUpdated = vi.fn();
    render(<DocumentProposalHandoff documentID="document-1" documentVersion={3} legalEntityID="entity-a" proposal={proposal} locked={false} onDocumentUpdated={onDocumentUpdated}/>);

    fireEvent.change(await screen.findByRole("textbox", { name: "Authorization rationale" }), { target: { value: "approved" } });
    fireEvent.click(screen.getByRole("button", { name: "Authorize conversion" }));

    await waitFor(() => expect(onDocumentUpdated).toHaveBeenCalledWith(approved));
    expect(await screen.findByText("Requirement created")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Open requirement" }).getAttribute("href")).toBe("#programs/program-1/requirements-controls/requirement/requirement-1");
  });

  it.each([
    ["REQUIREMENT", "requirement", "Open requirement"],
    ["CONTROL_OBJECTIVE", "control-objective", "Open control objective"],
  ])("opens the stored approved %s result", (resultType, kind, label) => {
    render(<DocumentProposalHandoff documentID="document-1" documentVersion={4} proposal={{ ...proposal, handoff: { ...approved.proposals[0]!.handoff!, target_program_id: "program/1 #銀行", result_object_type: resultType, result_object_id: "result/1 #銀行" } }} locked={false}/>);
    expect(screen.getByRole("link", { name: label }).getAttribute("href")).toBe(`#programs/program%2F1%20%23%E9%8A%80%E8%A1%8C/requirements-controls/${kind}/result%2F1%20%23%E9%8A%80%E8%A1%8C`);
    expect(screen.queryByText(/canonical/i)).toBeNull();
  });

  it.each([
    { result_object_id: undefined }, { target_program_id: undefined }, { result_object_type: "UNKNOWN" },
    { result_object_type: undefined }, { result_object_id: " " },
    { status: "REJECTED" as const }, { status: "CONVERSION_FAILED" as const },
  ])("does not guess a destination for incomplete or unsuccessful receipts %j", (overrides) => {
    render(<DocumentProposalHandoff documentID="document-1" documentVersion={4} proposal={{ ...proposal, handoff: { ...approved.proposals[0]!.handoff!, ...overrides } }} locked={false}/>);
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.queryByText(/canonical/i)).toBeNull();
    if (!overrides.status) expect(screen.getByText(/Reload this import/)).toBeTruthy();
  });

  it("explains a missing review record without implementation narration", () => {
    render(<DocumentProposalHandoff documentID="document-1" documentVersion={4} proposal={{ ...proposal, handoff: undefined }} locked={false}/>);
    expect(screen.getByText("Accepted · review details unavailable")).toBeTruthy();
    expect(screen.getByText(/Reload this import/)).toBeTruthy();
    expect(screen.queryByRole("link")).toBeNull();
  });
});
