import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DocumentCoverage, DocumentImport, DocumentImportSummary } from "../documentTypes";
import { applyDocumentCoverageSuggestion, importDocument, loadDocumentCoverage, loadDocumentImport, loadDocumentImports, recompareDocumentCoverage, reviewDocumentCoverage, reviewDocumentProposal } from "../documentApi";
import { ApiError } from "../http";
import { DocumentImportWorkspace } from "./DocumentImportWorkspace";

vi.mock("../documentApi", () => ({
  applyDocumentCoverageSuggestion: vi.fn(), importDocument: vi.fn(), loadDocumentCoverage: vi.fn(), loadDocumentImport: vi.fn(),
  loadDocumentImports: vi.fn(), recompareDocumentCoverage: vi.fn(), reviewDocumentCoverage: vi.fn(), reviewDocumentProposal: vi.fn(),
}));

const documentRecord: DocumentImport = {
  id: "11111111-1111-4111-8111-111111111111", tenant_id: "bank-demo", legal_entity_id: "bank-ng", file_name: "regulatory-notice.md", media_type: "text/markdown", purpose: "Assess a regulatory notice", source_type: "REGULATORY", size_bytes: 128, sha256: "a".repeat(64), storage_key: "document-imports/bank-demo/notice", artifact_status: "STORED_UNSCANNED", extraction_status: "EXTRACTED", extraction_method: "PLAIN_TEXT_V2", analysis_status: "REVIEW_REQUIRED", analysis_method: "DETERMINISTIC_RULES_V2", limitations: ["The artifact has not passed a production malware-scanning service."], sections: [{ id: "22222222-2222-4222-8222-222222222222", sequence: 1, title: "Records", text: "The bank must retain records for five years." }], proposals: [{ id: "33333333-3333-4333-8333-333333333333", kind: "REQUIREMENT_CANDIDATE", title: "Possible requirement", statement: "The bank must retain records for five years.", confidence: 0.86, anchor: { section_id: "22222222-2222-4222-8222-222222222222", quote: "The bank must retain records for five years." }, status: "PENDING_REVIEW" }], sections_total: 1, sections_omitted: 0, proposals_total: 1, proposals_omitted: 0, content_truncated: false, processed_at: "2026-08-06T10:00:01Z", created_by: "reviewer-1", created_at: "2026-08-06T10:00:00Z", updated_at: "2026-08-06T10:00:01Z", version: 2,
};

const documentSummary: DocumentImportSummary = {
  id: documentRecord.id, tenant_id: documentRecord.tenant_id, legal_entity_id: documentRecord.legal_entity_id, file_name: documentRecord.file_name, media_type: documentRecord.media_type, purpose: documentRecord.purpose, source_type: documentRecord.source_type, size_bytes: documentRecord.size_bytes, sha256: documentRecord.sha256, artifact_status: documentRecord.artifact_status, extraction_status: documentRecord.extraction_status, analysis_status: documentRecord.analysis_status, sections_total: 1, sections_omitted: 0, proposals_total: 1, proposals_omitted: 0, pending_proposal_count: 1, reviewed_proposal_count: 0, content_truncated: false, processed_at: documentRecord.processed_at, created_at: documentRecord.created_at, updated_at: documentRecord.updated_at, version: documentRecord.version,
};

const acceptedDocument: DocumentImport = { ...documentRecord, version: 3, proposals: [{ ...documentRecord.proposals[0]!, status: "ACCEPTED", reviewed_by: "reviewer-2", reviewed_at: "2026-08-06T10:05:00Z" }] };
const processingDocument: DocumentImport = { ...documentRecord, extraction_status: "PENDING", extraction_method: "PENDING", analysis_status: "PENDING", sections: [], proposals: [], sections_total: 0, proposals_total: 0, processed_at: undefined, version: 1 };
const processingSummary: DocumentImportSummary = { ...documentSummary, extraction_status: "PENDING", analysis_status: "PENDING", sections_total: 0, proposals_total: 0, pending_proposal_count: 0, processed_at: undefined, version: 1 };
const storedOnlyPDF: DocumentImport = {
  ...documentRecord,
  file_name: "official-regulation.pdf",
  media_type: "application/pdf",
  extraction_status: "UNSUPPORTED",
  extraction_method: "NONE",
  analysis_status: "UNAVAILABLE",
  sections: [],
  proposals: [],
  sections_total: 0,
  proposals_total: 0,
  limitations: ["The original PDF was stored, but this build has no approved PDF text extractor or OCR adapter."],
};

const coverageRecord: DocumentCoverage = {
  id: "coverage-1", tenant_id: "bank-demo", legal_entity_id: "bank-ng", document_id: documentRecord.id,
  document_sha256: documentRecord.sha256, status: "READY", version: 1, limitations: [], next_cursor: "",
  metrics: {
    estimated_verified: { numerator: 1, denominator: 2 }, verified: { numerator: 0, denominator: 2 },
    requirement_mapped: { numerator: 1, denominator: 2 }, control_implemented: { numerator: 1, denominator: 2 }, evidence_supported: { numerator: 1, denominator: 2 },
  },
  candidates: [{
    id: "coverage-candidate-1", fingerprint: "fingerprint-1", eligible: true,
    statement: "The bank must retain records for five years.", anchor: { section_id: "section-records", quote: "The bank must retain records for five years.", page: 7 },
    modality: "MUST", actor: "the bank", action: "retain", object: "records for five years", citations: ["section 41"], dates: [], topics: ["records"], uncertainty: [],
    jurisdiction: "Nigeria", program_type: "PRIVACY", classification: "NEEDS_REVIEW",
    matches: [{
      id: "match-1", program_id: "program-1", program_code: "NDPA-2023", program_name: "Nigeria data protection", program_version: 3,
      requirement_id: "requirement-1", requirement_code: "NDPA-41", requirement_title: "Retain processing records", requirement_version: 2,
      score: .91, band: "STRONG", rationale: "Strong source, topic and obligation-language alignment.", conflicts: [], components: [{ name: "TOPIC", weight: .35, score: 1, reason: "Shared records topic" }],
      coverage: { requirement_id: "requirement-1", applicable: true, applicability: "APPLICABLE", control_implemented: true, evidence_supported: true, complete: true, control_ids: ["control-1"], evidence_contract_ids: ["contract-1"], reasons: [] },
    }],
  }, {
    id: "coverage-candidate-2", fingerprint: "fingerprint-2", eligible: true,
    statement: "The bank must notify the regulator within 72 hours.", anchor: { section_id: "section-notice", quote: "The bank must notify the regulator within 72 hours.", page: 8 },
    modality: "MUST", actor: "the bank", action: "notify", object: "the regulator", citations: [], dates: ["72 hours"], topics: ["notification"], uncertainty: [],
    jurisdiction: "Nigeria", classification: "GAP", matches: [],
  }],
  suggestions: [{ id: "suggestion-1", candidate_id: "coverage-candidate-2", type: "CREATE_PROGRAM", status: "PROPOSED", title: "Create a notification Program", rationale: "No in-scope Program covers this obligation." }],
  matters: [],
};

beforeEach(() => {
  vi.mocked(loadDocumentImports).mockResolvedValue([documentSummary]);
  vi.mocked(loadDocumentImport).mockResolvedValue(documentRecord);
  vi.mocked(importDocument).mockResolvedValue(processingDocument);
  vi.mocked(reviewDocumentProposal).mockResolvedValue(acceptedDocument);
  vi.mocked(loadDocumentCoverage).mockResolvedValue(coverageRecord);
  vi.mocked(reviewDocumentCoverage).mockResolvedValue({ ...coverageRecord, version: 2, metrics: { ...coverageRecord.metrics, verified: { numerator: 1, denominator: 2 } } });
  vi.mocked(recompareDocumentCoverage).mockResolvedValue();
  vi.mocked(applyDocumentCoverageSuggestion).mockResolvedValue({ assessment: { ...coverageRecord, version: 2, suggestions: [{ ...coverageRecord.suggestions[0]!, status: "APPLIED", applied_type: "PROGRAM", applied_id: "program-new" }] }, object_type: "PROGRAM", object_id: "program-new" });
});

describe("DocumentImportWorkspace", () => {
  async function openImport() {
    fireEvent.click(await screen.findByRole("button", { name: "Import document" }));
  }

  it("renders source-anchored review evidence without axe violations", async () => {
    const { container } = render(<DocumentImportWorkspace/>);
    expect(await screen.findByRole("heading", { name: "regulatory-notice.md" })).toBeTruthy();
    expect(screen.getByText("Possible requirement")).toBeTruthy();
    expect(screen.getAllByText("The bank must retain records for five years.", { selector: "blockquote" }).length).toBeGreaterThan(0);
    expect(screen.getByText("Original hash")).toBeTruthy();
    const result = await axe.run(container, { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] }, rules: { "color-contrast": { enabled: false } } });
    expect(result.violations).toEqual([]);
  });

  it("starts with no templated purpose and previews the selected document before import", async () => {
    render(<DocumentImportWorkspace/>);
    await screen.findByRole("heading", { name: "regulatory-notice.md" });
    await openImport();
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
    await openImport();
    const file = new File(["notice"], "new-notice.pdf", { type: "application/pdf" });
    fireEvent.change(screen.getByLabelText("Document"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Import document" }));
    expect((await screen.findByRole("alert")).textContent).toContain("Say what reviewers should look for in this document.");
    expect(importDocument).not.toHaveBeenCalled();
  });

  it("shows a concise, truth-labelled coverage summary and explainable match", async () => {
    render(<DocumentImportWorkspace/>);
    expect(await screen.findByRole("heading", { name: "Coverage assessment" })).toBeTruthy();
    expect(screen.getByText("0%", { selector: ".coverage-primary-value" })).toBeTruthy();
    expect(screen.getByText(/1 of 2 estimated before review/i)).toBeTruthy();
    expect(screen.getByText("Nigeria data protection")).toBeTruthy();
    expect(screen.getByText(/page 7/i)).toBeTruthy();
    expect(screen.getByText(/Strong source, topic and obligation-language alignment/i)).toBeTruthy();
  });

  it("confirms a proposed Program match using the current assessment version", async () => {
    render(<DocumentImportWorkspace/>);
    fireEvent.click(await screen.findByRole("button", { name: "Confirm match" }));
    await waitFor(() => expect(reviewDocumentCoverage).toHaveBeenCalledWith(documentRecord.id, 1, [{ candidate_id: "coverage-candidate-1", decision: "ACCEPT_MATCH", match_id: "match-1" }]));
  });

  it("applies a gap recommendation as a governed draft", async () => {
    render(<DocumentImportWorkspace/>);
    fireEvent.click(await screen.findByRole("button", { name: "Create draft Program" }));
    await waitFor(() => expect(applyDocumentCoverageSuggestion).toHaveBeenCalledWith(documentRecord.id, "suggestion-1", 1));
    expect(await screen.findByText(/Draft Program created/i)).toBeTruthy();
  });

  it("offers a one-step refresh when Program truth changed", async () => {
    vi.mocked(loadDocumentCoverage).mockResolvedValue({ ...coverageRecord, status: "STALE" });
    render(<DocumentImportWorkspace/>);
    fireEvent.click(await screen.findByRole("button", { name: "Compare again" }));
    await waitFor(() => expect(recompareDocumentCoverage).toHaveBeenCalledWith(documentRecord.id));
  });

  it("keeps existing review primary and describes automated searchable-PDF extraction", async () => {
    render(<DocumentImportWorkspace/>);
    await screen.findByRole("heading", { name: "regulatory-notice.md" });
    expect(screen.queryByRole("textbox", { name: "What should reviewers look for?" })).toBeNull();
    await openImport();
    expect(screen.getByRole("textbox", { name: "What should reviewers look for?" })).toBeTruthy();
    expect(screen.getByText(/TXT, Markdown, CSV, DOCX, XLSX or PDF.*20 MB/i)).toBeTruthy();
    expect(screen.getByText(/Searchable PDFs are extracted automatically with page references/i)).toBeTruthy();
    expect(screen.getByText(/Scanned PDFs remain stored and clearly report when OCR is required/i)).toBeTruthy();
  });

  it("preserves the selected document and purpose when upload fails", async () => {
    vi.mocked(importDocument).mockRejectedValue(new Error("Upload interrupted"));
    render(<DocumentImportWorkspace/>);
    await screen.findByRole("heading", { name: "regulatory-notice.md" });
    await openImport();
    const file = new File(["notice"], "retry-notice.pdf", { type: "application/pdf" });
    fireEvent.change(screen.getByLabelText("Document"), { target: { files: [file] } });
    fireEvent.change(screen.getByRole("textbox", { name: "What should reviewers look for?" }), { target: { value: "Review payment requirements" } });
    fireEvent.click(screen.getByRole("button", { name: "Import document" }));
    expect((await screen.findByRole("alert")).textContent).toContain("Upload interrupted");
    expect(screen.getByText(/retry-notice\.pdf/)).toBeTruthy();
    expect((screen.getByRole("textbox", { name: "What should reviewers look for?" }) as HTMLInputElement).value).toBe("Review payment requirements");
  });

  it("rejects a document larger than the stated 20 MB limit before upload", async () => {
    render(<DocumentImportWorkspace/>);
    await screen.findByRole("heading", { name: "regulatory-notice.md" });
    await openImport();
    const file = new File(["oversized"], "oversized.pdf", { type: "application/pdf" });
    Object.defineProperty(file, "size", { value: 20 * 1024 * 1024 + 1 });
    fireEvent.change(screen.getByLabelText("Document"), { target: { files: [file] } });
    expect((await screen.findByRole("alert")).textContent).toContain("no larger than 20 MB");
    fireEvent.change(screen.getByRole("textbox", { name: "What should reviewers look for?" }), { target: { value: "Review requirements" } });
    fireEvent.click(screen.getByRole("button", { name: "Import document" }));
    expect(importDocument).not.toHaveBeenCalled();
  });

  it("describes a PDF as stored only when automated text review is unavailable", async () => {
    vi.mocked(loadDocumentImport).mockResolvedValue(storedOnlyPDF);
    render(<DocumentImportWorkspace/>);
    expect(await screen.findByText("Original stored")).toBeTruthy();
    expect(screen.getByText("Text review unavailable")).toBeTruthy();
    expect(screen.queryByText("Extraction failed")).toBeNull();
  });

  it("records an explicit proposal review", async () => {
    render(<DocumentImportWorkspace/>);
    fireEvent.click(await screen.findByText("Extraction proposals"));
    fireEvent.click(await screen.findByRole("button", { name: "Accept proposal" }));
    await waitFor(() => expect(reviewDocumentProposal).toHaveBeenCalledWith(documentRecord.id, documentRecord.proposals[0]!.id, "ACCEPTED", documentRecord.version));
    expect(await screen.findByText("Accepted")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Accept proposal" })).toBeNull();
  });

  it("serializes proposal review while a write is in flight", async () => {
    let resolveReview!: (value: DocumentImport) => void;
    vi.mocked(reviewDocumentProposal).mockImplementation(() => new Promise((resolve) => { resolveReview = resolve; }));
    render(<DocumentImportWorkspace/>);
    fireEvent.click(await screen.findByText("Extraction proposals"));
    const accept = await screen.findByRole("button", { name: "Accept proposal" });
    const reject = screen.getByRole("button", { name: "Reject" });
    fireEvent.click(accept);
    expect((accept as HTMLButtonElement).disabled).toBe(true);
    expect((reject as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(reject);
    expect(reviewDocumentProposal).toHaveBeenCalledTimes(1);
    resolveReview(acceptedDocument);
    expect(await screen.findByText("Accepted")).toBeTruthy();
  });

  it("surfaces a version conflict and reloads authoritative import state", async () => {
    vi.mocked(reviewDocumentProposal).mockRejectedValue(new ApiError(409, "changed", "version_conflict"));
    render(<DocumentImportWorkspace/>);
    fireEvent.click(await screen.findByText("Extraction proposals"));
    fireEvent.click(await screen.findByRole("button", { name: "Accept proposal" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/changed while you were reviewing/i);
    await waitFor(() => expect(loadDocumentImport).toHaveBeenCalledTimes(2));
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
    expect(screen.getByRole("textbox", { name: "What should reviewers look for?" })).toBeTruthy();
    expect(screen.getByText(/Searchable PDFs are extracted automatically; scanned PDFs remain stored and report when OCR is required/)).toBeTruthy();
  });

  it("does not show import claims when the service is unavailable", async () => {
    vi.mocked(loadDocumentImports).mockRejectedValue(new Error("Source unavailable"));
    render(<DocumentImportWorkspace/>);
    expect(await screen.findByRole("heading", { name: "Imported documents could not be loaded" })).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toContain("Source unavailable");
    expect(screen.queryByText("Possible requirement")).toBeNull();
  });
});
