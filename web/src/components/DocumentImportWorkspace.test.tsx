import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DocumentImport, DocumentImportSummary } from "../documentTypes";
import { DocumentImportWorkspace } from "./DocumentImportWorkspace";
import { importDocument, loadDocumentImport, loadDocumentImports, reviewDocumentProposal } from "../documentApi";

vi.mock("../documentApi", () => ({ importDocument: vi.fn(), loadDocumentImport: vi.fn(), loadDocumentImports: vi.fn(), reviewDocumentProposal: vi.fn() }));

const documentRecord: DocumentImport = {
  id: "11111111-1111-4111-8111-111111111111", tenant_id: "bank-demo", legal_entity_id: "bank-ng", file_name: "regulatory-notice.md", media_type: "text/markdown", purpose: "Assess a regulatory notice", source_type: "REGULATORY", size_bytes: 128, sha256: "a".repeat(64), storage_key: "document-imports/bank-demo/notice", artifact_status: "STORED_UNSCANNED", extraction_status: "EXTRACTED", extraction_method: "PLAIN_TEXT_V2", analysis_status: "REVIEW_REQUIRED", analysis_method: "DETERMINISTIC_RULES_V2", limitations: ["The artifact has not passed a production malware-scanning service."], sections: [{ id: "22222222-2222-4222-8222-222222222222", sequence: 1, title: "Records", text: "The bank must retain records for five years." }], proposals: [{ id: "33333333-3333-4333-8333-333333333333", kind: "REQUIREMENT_CANDIDATE", title: "Possible requirement", statement: "The bank must retain records for five years.", confidence: 0.86, anchor: { section_id: "22222222-2222-4222-8222-222222222222", quote: "The bank must retain records for five years." }, status: "PENDING_REVIEW" }], sections_total: 1, sections_omitted: 0, proposals_total: 1, proposals_omitted: 0, content_truncated: false, processed_at: "2026-08-06T10:00:01Z", created_by: "reviewer-1", created_at: "2026-08-06T10:00:00Z", updated_at: "2026-08-06T10:00:01Z", version: 2,
};

const documentSummary: DocumentImportSummary = {
  id: documentRecord.id, tenant_id: documentRecord.tenant_id, legal_entity_id: documentRecord.legal_entity_id, file_name: documentRecord.file_name, media_type: documentRecord.media_type, purpose: documentRecord.purpose, source_type: documentRecord.source_type, size_bytes: documentRecord.size_bytes, sha256: documentRecord.sha256, artifact_status: documentRecord.artifact_status, extraction_status: documentRecord.extraction_status, analysis_status: documentRecord.analysis_status, sections_total: 1, sections_omitted: 0, proposals_total: 1, proposals_omitted: 0, pending_proposal_count: 1, reviewed_proposal_count: 0, content_truncated: false, processed_at: documentRecord.processed_at, created_at: documentRecord.created_at, updated_at: documentRecord.updated_at, version: documentRecord.version,
};

const acceptedDocument: DocumentImport = { ...documentRecord, version: 3, proposals: [{ ...documentRecord.proposals[0]!, status: "ACCEPTED", reviewed_by: "reviewer-2", reviewed_at: "2026-08-06T10:05:00Z" }] };
const processingDocument: DocumentImport = { ...documentRecord, extraction_status: "PENDING", extraction_method: "PENDING", analysis_status: "PENDING", sections: [], proposals: [], sections_total: 0, proposals_total: 0, processed_at: undefined, version: 1 };
const processingSummary: DocumentImportSummary = { ...documentSummary, extraction_status: "PENDING", analysis_status: "PENDING", sections_total: 0, proposals_total: 0, pending_proposal_count: 0, processed_at: undefined, version: 1 };

beforeEach(() => {
  vi.mocked(loadDocumentImports).mockResolvedValue([documentSummary]);
  vi.mocked(loadDocumentImport).mockResolvedValue(documentRecord);
  vi.mocked(importDocument).mockResolvedValue(processingDocument);
  vi.mocked(reviewDocumentProposal).mockResolvedValue(acceptedDocument);
});

describe("DocumentImportWorkspace", () => {
  it("renders source-anchored review evidence without axe violations", async () => {
    const { container } = render(<DocumentImportWorkspace/>);
    expect(await screen.findByRole("heading", { name: "regulatory-notice.md" })).toBeTruthy();
    expect(screen.getByText("Possible requirement")).toBeTruthy();
    expect(screen.getByText("The bank must retain records for five years.", { selector: "blockquote" })).toBeTruthy();
    expect(screen.getByText("Original hash")).toBeTruthy();
    const result = await axe.run(container, { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] }, rules: { "color-contrast": { enabled: false } } });
    expect(result.violations).toEqual([]);
  });

  it("starts with no templated purpose and previews the selected document before import", async () => {
    render(<DocumentImportWorkspace/>);
    await screen.findByRole("heading", { name: "regulatory-notice.md" });
    const purpose = screen.getByRole("textbox", { name: "What should reviewers look for?" }) as HTMLInputElement;
    expect(purpose.value).toBe("");
    const file = new File(["notice"], "new-notice.pdf", { type: "application/pdf" });
    fireEvent.change(screen.getByLabelText("Document"), { target: { files: [file] } });
    expect(screen.getByText(/new-notice\.pdf/)).toBeTruthy();
    fireEvent.change(purpose, { target: { value: "Changes affecting card operations" } });
    fireEvent.click(screen.getByRole("button", { name: "Import document" }));
    await waitFor(() => expect(importDocument).toHaveBeenCalledWith(file, "Changes affecting card operations", "DOCUMENT"));
  });

  it("does not allow a generic blank purpose to be persisted", async () => {
    render(<DocumentImportWorkspace/>);
    await screen.findByRole("heading", { name: "regulatory-notice.md" });
    const file = new File(["notice"], "new-notice.pdf", { type: "application/pdf" });
    fireEvent.change(screen.getByLabelText("Document"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Import document" }));
    expect((await screen.findByRole("alert")).textContent).toContain("Say what reviewers should look for in this document.");
    expect(importDocument).not.toHaveBeenCalled();
  });

  it("records an explicit proposal review", async () => {
    render(<DocumentImportWorkspace/>);
    fireEvent.click(await screen.findByRole("button", { name: "Accept proposal" }));
    await waitFor(() => expect(reviewDocumentProposal).toHaveBeenCalledWith(documentRecord.id, documentRecord.proposals[0]!.id, "ACCEPTED", documentRecord.version));
    expect(await screen.findByText("Accepted")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Accept proposal" })).toBeNull();
  });

  it("renders a durable processing receipt without claiming review completion", async () => {
    vi.mocked(loadDocumentImports).mockResolvedValue([processingSummary]);
    vi.mocked(loadDocumentImport).mockResolvedValue(processingDocument);
    render(<DocumentImportWorkspace/>);
    expect(await screen.findByText("Original stored successfully.")).toBeTruthy();
    expect(screen.getByText("Stored · processing")).toBeTruthy();
    expect(screen.queryByText("Review complete")).toBeNull();
    expect(screen.queryByRole("button", { name: "Accept proposal" })).toBeNull();
  });

  it("keeps list records body-free", () => {
    expect("sections" in documentSummary).toBe(false);
    expect("proposals" in documentSummary).toBe(false);
    expect(documentSummary.pending_proposal_count).toBe(1);
  });

  it("renders the bounded empty state", async () => {
    vi.mocked(loadDocumentImports).mockResolvedValue([]);
    render(<DocumentImportWorkspace/>);
    expect(await screen.findByRole("heading", { name: "No documents imported" })).toBeTruthy();
    expect(screen.getByText(/PDFs are stored even when text extraction is unavailable/)).toBeTruthy();
  });

  it("does not show import claims when the service is unavailable", async () => {
    vi.mocked(loadDocumentImports).mockRejectedValue(new Error("Source unavailable"));
    render(<DocumentImportWorkspace/>);
    expect(await screen.findByRole("heading", { name: "Imported documents could not be loaded" })).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toContain("Source unavailable");
    expect(screen.queryByText("Possible requirement")).toBeNull();
  });
});
