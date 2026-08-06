import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DocumentImport } from "../documentTypes";
import { DocumentImportWorkspace } from "./DocumentImportWorkspace";
import { importDocument, loadDocumentImport, loadDocumentImports, reviewDocumentProposal } from "../documentApi";

vi.mock("../documentApi", () => ({
  importDocument: vi.fn(),
  loadDocumentImport: vi.fn(),
  loadDocumentImports: vi.fn(),
  reviewDocumentProposal: vi.fn(),
}));

const documentRecord: DocumentImport = {
  id: "11111111-1111-4111-8111-111111111111",
  tenant_id: "bank-demo",
  legal_entity_id: "bank-ng",
  file_name: "regulatory-notice.md",
  media_type: "text/markdown",
  purpose: "Assess a regulatory notice",
  source_type: "REGULATORY",
  size_bytes: 128,
  sha256: "a".repeat(64),
  storage_key: "document-imports/bank-demo/notice",
  artifact_status: "STORED_UNSCANNED",
  extraction_status: "EXTRACTED",
  extraction_method: "MARKDOWN_TEXT_V1",
  analysis_status: "REVIEW_REQUIRED",
  analysis_method: "DETERMINISTIC_RULES_V1",
  limitations: ["The artifact has not passed a production malware-scanning service."],
  sections: [{
    id: "22222222-2222-4222-8222-222222222222",
    sequence: 1,
    title: "Records",
    text: "The bank must retain records for five years.",
  }],
  proposals: [{
    id: "33333333-3333-4333-8333-333333333333",
    kind: "REQUIREMENT_CANDIDATE",
    title: "Possible requirement",
    statement: "The bank must retain records for five years.",
    confidence: 0.86,
    anchor: {
      section_id: "22222222-2222-4222-8222-222222222222",
      quote: "The bank must retain records for five years.",
    },
    status: "PENDING_REVIEW",
  }],
  created_by: "reviewer-1",
  created_at: "2026-08-06T10:00:00Z",
  updated_at: "2026-08-06T10:00:00Z",
  version: 1,
};

const acceptedDocument: DocumentImport = {
  ...documentRecord,
  version: 2,
  proposals: [{
    ...documentRecord.proposals[0]!,
    status: "ACCEPTED",
    reviewed_by: "reviewer-2",
    reviewed_at: "2026-08-06T10:05:00Z",
  }],
};

beforeEach(() => {
  vi.mocked(loadDocumentImports).mockResolvedValue([documentRecord]);
  vi.mocked(loadDocumentImport).mockResolvedValue(documentRecord);
  vi.mocked(importDocument).mockResolvedValue(documentRecord);
  vi.mocked(reviewDocumentProposal).mockResolvedValue(acceptedDocument);
});

describe("DocumentImportWorkspace", () => {
  it("renders source-anchored review evidence without axe violations", async () => {
    const { container } = render(<DocumentImportWorkspace />);

    expect(await screen.findByRole("heading", { name: "regulatory-notice.md" })).toBeTruthy();
    expect(screen.getByText("Possible requirement")).toBeTruthy();
    expect(screen.getByText("The bank must retain records for five years.", { selector: "blockquote" })).toBeTruthy();
    expect(screen.getByText(`Original hash`)).toBeTruthy();

    const result = await axe.run(container, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
      rules: { "color-contrast": { enabled: false } },
    });
    expect(result.violations).toEqual([]);
  });

  it("records an explicit human proposal review", async () => {
    render(<DocumentImportWorkspace />);
    const accept = await screen.findByRole("button", { name: "Accept for governed follow-up" });
    fireEvent.click(accept);

    await waitFor(() => expect(reviewDocumentProposal).toHaveBeenCalledWith(
      documentRecord.id,
      documentRecord.proposals[0]!.id,
      "ACCEPTED",
      1,
    ));
    expect(await screen.findByText("Accepted")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Accept for governed follow-up" })).toBeNull();
  });

  it("renders the bounded empty state", async () => {
    vi.mocked(loadDocumentImports).mockResolvedValue([]);
    render(<DocumentImportWorkspace />);

    expect(await screen.findByRole("heading", { name: "No documents imported" })).toBeTruthy();
    expect(screen.getByText(/PDFs are retained/)).toBeTruthy();
  });

  it("does not show import claims when the service is unavailable", async () => {
    vi.mocked(loadDocumentImports).mockRejectedValue(new Error("Source unavailable"));
    render(<DocumentImportWorkspace />);

    expect(await screen.findByRole("heading", { name: "Imported documents could not be loaded" })).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toContain("Source unavailable");
    expect(screen.queryByText("Possible requirement")).toBeNull();
  });
});
