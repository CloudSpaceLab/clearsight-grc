import type { DocumentImport, DocumentImportSummary, ProposalStatus } from "./documentTypes";
import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export async function loadDocumentImports(): Promise<DocumentImportSummary[]> {
  const body = await requestJSON<{ items: Array<DocumentImportSummary | DocumentImport> }>(apiBase, "/api/v1/document-imports?limit=50");
  return body.items.map(normalizeSummary);
}

export async function loadDocumentImport(id: string): Promise<DocumentImport> {
  return normalizeDetail(await requestJSON<DocumentImport>(apiBase, `/api/v1/document-imports/${encodeURIComponent(id)}`));
}

export async function importDocument(file: File, purpose: string, sourceType: string): Promise<DocumentImport> {
  const body = new FormData();
  body.set("file", file);
  body.set("purpose", purpose);
  body.set("source_type", sourceType);
  return normalizeDetail(await requestJSON<DocumentImport>(apiBase, "/api/v1/document-imports", { method: "POST", body }));
}

export async function reviewDocumentProposal(documentID: string, proposalID: string, status: ProposalStatus, expectedVersion: number, note = ""): Promise<DocumentImport> {
  return normalizeDetail(await requestJSON<DocumentImport>(apiBase, `/api/v1/document-imports/${encodeURIComponent(documentID)}/proposals/${encodeURIComponent(proposalID)}/review`, {
    method: "POST",
    body: JSON.stringify({ status, note, expected_version: expectedVersion }),
  }));
}

function normalizeDetail(value: DocumentImport): DocumentImport {
  return {
    ...value,
    sections: Array.isArray(value.sections) ? value.sections : [],
    proposals: Array.isArray(value.proposals) ? value.proposals : [],
    limitations: Array.isArray(value.limitations) ? value.limitations : [],
    sections_total: value.sections_total ?? value.sections?.length ?? 0,
    sections_omitted: value.sections_omitted ?? 0,
    proposals_total: value.proposals_total ?? value.proposals?.length ?? 0,
    proposals_omitted: value.proposals_omitted ?? 0,
    content_truncated: value.content_truncated ?? false,
  };
}

function normalizeSummary(value: DocumentImportSummary | DocumentImport): DocumentImportSummary {
  const detail = "proposals" in value ? value : null;
  const proposals = detail?.proposals ?? [];
  const pending = "pending_proposal_count" in value ? value.pending_proposal_count : proposals.filter((proposal) => proposal.status === "PENDING_REVIEW").length;
  const reviewed = "reviewed_proposal_count" in value ? value.reviewed_proposal_count : proposals.length - pending;
  return {
    id: value.id,
    tenant_id: value.tenant_id,
    legal_entity_id: value.legal_entity_id,
    file_name: value.file_name,
    media_type: value.media_type,
    purpose: value.purpose,
    source_type: value.source_type,
    size_bytes: value.size_bytes,
    sha256: value.sha256,
    artifact_status: value.artifact_status,
    extraction_status: value.extraction_status,
    analysis_status: value.analysis_status,
    sections_total: value.sections_total ?? (detail?.sections.length ?? 0),
    sections_omitted: value.sections_omitted ?? 0,
    proposals_total: value.proposals_total ?? proposals.length,
    proposals_omitted: value.proposals_omitted ?? 0,
    pending_proposal_count: pending,
    reviewed_proposal_count: reviewed,
    content_truncated: value.content_truncated ?? false,
    processed_at: value.processed_at,
    created_at: value.created_at,
    updated_at: value.updated_at,
    version: value.version,
  };
}
