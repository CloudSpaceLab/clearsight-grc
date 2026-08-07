export type ProposalStatus = "PENDING_REVIEW" | "ACCEPTED" | "REJECTED";
export type ExtractionStatus = "PENDING" | "EXTRACTED" | "UNSUPPORTED" | "FAILED";
export type AnalysisStatus = "PENDING" | "REVIEW_REQUIRED" | "NO_PROPOSALS" | "UNAVAILABLE";
export type DocumentAnchor = { section_id: string; quote: string; page?: number; sheet?: string; row_start?: number; row_end?: number };
export type DocumentSection = { id: string; sequence: number; title: string; text: string; page?: number; sheet?: string; row_start?: number; row_end?: number };
export type DocumentProposal = { id: string; kind: string; title: string; statement: string; confidence: number; anchor: DocumentAnchor; status: ProposalStatus; reviewed_by?: string; reviewed_at?: string; review_note?: string };

export type DocumentImportSummary = {
  id: string;
  tenant_id: string;
  legal_entity_id?: string;
  file_name: string;
  media_type: string;
  purpose: string;
  source_type: string;
  size_bytes: number;
  sha256: string;
  artifact_status: string;
  extraction_status: ExtractionStatus;
  analysis_status: AnalysisStatus;
  sections_total: number;
  sections_omitted: number;
  proposals_total: number;
  proposals_omitted: number;
  pending_proposal_count: number;
  reviewed_proposal_count: number;
  content_truncated: boolean;
  processed_at?: string;
  created_at: string;
  updated_at: string;
  version: number;
};

export type DocumentImport = Omit<DocumentImportSummary, "pending_proposal_count" | "reviewed_proposal_count"> & {
  storage_key: string;
  extraction_method: string;
  analysis_method: string;
  limitations: string[];
  sections: DocumentSection[];
  proposals: DocumentProposal[];
  created_by: string;
};
