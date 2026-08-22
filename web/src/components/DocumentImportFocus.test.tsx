import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DocumentCoverage, DocumentImport, DocumentImportSummary } from "../documentTypes";
import { loadDocumentCoverage, loadDocumentImport, loadDocumentImports } from "../documentApi";
import { DocumentImportWorkspace } from "./DocumentImportWorkspace";

vi.mock("../documentApi", async () => {
  const actual = await vi.importActual<typeof import("../documentApi")>("../documentApi");
  return {
    ...actual,
    loadDocumentCoverage: vi.fn(),
    loadDocumentImport: vi.fn(),
    loadDocumentImports: vi.fn(),
  };
});

vi.mock("./DocumentProposalHandoff", () => ({
  DocumentProposalHandoff: () => <div>Focused governed handoff</div>,
}));

const targetID = "22222222-2222-4222-8222-222222222222";
const proposalID = "33333333-3333-4333-8333-333333333333";

const targetDocument: DocumentImport = {
  id: targetID,
  tenant_id: "bank-demo",
  legal_entity_id: "bank-ng",
  file_name: "target-notice.md",
  media_type: "text/markdown",
  purpose: "Review target notice",
  source_type: "REGULATORY",
  size_bytes: 128,
  sha256: "a".repeat(64),
  storage_key: "document-imports/bank-demo/target",
  artifact_status: "STORED_UNSCANNED",
  extraction_status: "EXTRACTED",
  extraction_method: "PLAIN_TEXT_V2",
  analysis_status: "REVIEW_REQUIRED",
  analysis_method: "DETERMINISTIC_RULES_V2",
  limitations: [],
  sections: [{ id: "section-1", sequence: 1, title: "Records", text: "The bank must retain records." }],
  proposals: [{
    id: proposalID,
    kind: "REQUIREMENT_CANDIDATE",
    title: "Retain records",
    statement: "The bank must retain records.",
    confidence: 0.9,
    anchor: { section_id: "section-1", quote: "The bank must retain records." },
    status: "ACCEPTED",
    reviewed_by: "intake-1",
    reviewed_at: "2026-08-22T10:00:00Z",
    handoff: {
      id: "handoff-1",
      status: "AWAITING_REVIEW",
      intake_principal_id: "intake-1",
      draft_title: "Retain records",
      draft_statement: "The bank must retain records.",
      updated_at: "2026-08-22T10:00:01Z",
      version: 1,
      route: { status: "DIRECT", responsibility: "REVIEWER", principal_id: "reviewer-1", is_current_actor: true },
    },
  }],
  sections_total: 1,
  sections_omitted: 0,
  proposals_total: 1,
  proposals_omitted: 0,
  content_truncated: false,
  processed_at: "2026-08-22T10:00:00Z",
  created_by: "intake-1",
  created_at: "2026-08-22T09:59:00Z",
  updated_at: "2026-08-22T10:00:01Z",
  version: 3,
};

function summary(id: string, fileName: string): DocumentImportSummary {
  return {
    id,
    tenant_id: "bank-demo",
    legal_entity_id: "bank-ng",
    file_name: fileName,
    media_type: "text/markdown",
    purpose: "Review notice",
    source_type: "REGULATORY",
    size_bytes: 128,
    sha256: "b".repeat(64),
    artifact_status: "STORED_UNSCANNED",
    extraction_status: "EXTRACTED",
    analysis_status: "REVIEW_REQUIRED",
    sections_total: 1,
    sections_omitted: 0,
    proposals_total: 1,
    proposals_omitted: 0,
    pending_proposal_count: 0,
    reviewed_proposal_count: 1,
    content_truncated: false,
    processed_at: "2026-08-22T10:00:00Z",
    created_at: "2026-08-22T09:59:00Z",
    updated_at: "2026-08-22T10:00:01Z",
    version: 3,
  };
}

const coverage: DocumentCoverage = {
  tenant_id: "bank-demo",
  legal_entity_id: "bank-ng",
  document_id: targetID,
  document_sha256: targetDocument.sha256,
  status: "READY",
  candidates: [],
  suggestions: [],
  matters: [],
  metrics: {
    estimated_verified: { numerator: 0, denominator: 0 },
    verified: { numerator: 0, denominator: 0 },
    requirement_mapped: { numerator: 0, denominator: 0 },
    control_implemented: { numerator: 0, denominator: 0 },
    evidence_supported: { numerator: 0, denominator: 0 },
  },
  limitations: [],
  version: 1,
};

describe("DocumentImportWorkspace focused route", () => {
  beforeEach(() => {
    Object.defineProperty(Element.prototype, "scrollIntoView", { value: vi.fn(), configurable: true });
    window.history.replaceState(null, "", `#imports/${targetID}/${proposalID}`);
    vi.mocked(loadDocumentImports).mockResolvedValue([
      summary("11111111-1111-4111-8111-111111111111", "other-notice.md"),
      summary(targetID, targetDocument.file_name),
    ]);
    vi.mocked(loadDocumentImport).mockImplementation(async (id) => {
      if (id !== targetID) throw new Error(`unexpected document ${id}`);
      return targetDocument;
    });
    vi.mocked(loadDocumentCoverage).mockResolvedValue(coverage);
  });

  it("loads the route-bound document and opens the exact proposal handoff", async () => {
    render(<DocumentImportWorkspace/>);

    expect(await screen.findByRole("heading", { name: "target-notice.md" })).toBeTruthy();
    await waitFor(() => expect(loadDocumentImport).toHaveBeenCalledWith(targetID));

    const proposal = window.document.getElementById(`document-proposal-${proposalID}`);
    expect(proposal).toBeTruthy();
    expect(proposal?.closest("details")?.open).toBe(true);
    expect(screen.getByText("Focused governed handoff")).toBeTruthy();
  });
});
