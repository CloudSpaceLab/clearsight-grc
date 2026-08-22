export type ProposalStatus = "PENDING_REVIEW" | "ACCEPTED" | "REJECTED";
export type HandoffStatus = "AWAITING_REVIEW" | "AWAITING_AUTHORIZATION" | "RETURNED" | "REJECTED" | "APPROVED" | "CONVERSION_FAILED";
export type ConversionTarget = "REQUIREMENT" | "CONTROL_OBJECTIVE";
export type HandoffRoute = {
  responsibility: string;
  status: string;
  principal_id?: string;
  principal_name?: string;
  is_current_actor?: boolean;
  rule_id?: string;
  policy_version?: string;
  explanation?: string;
};
export type ProposalHandoff = {
  id: string;
  status: HandoffStatus;
  intake_principal_id: string;
  reviewer_principal_id?: string;
  authorizer_principal_id?: string;
  draft_title: string;
  draft_statement: string;
  target_type?: ConversionTarget;
  target_program_id?: string;
  target_program_version?: number;
  review_note?: string;
  authorization_note?: string;
  result_object_type?: string;
  result_object_id?: string;
  route?: HandoffRoute;
  updated_at: string;
  version: number;
};
export type ExtractionStatus = "PENDING" | "EXTRACTED" | "UNSUPPORTED" | "FAILED";
export type AnalysisStatus = "PENDING" | "REVIEW_REQUIRED" | "NO_PROPOSALS" | "UNAVAILABLE";
export type DocumentAnchor = { section_id: string; quote: string; page?: number; sheet?: string; row_start?: number; row_end?: number };
export type DocumentSection = { id: string; sequence: number; title: string; text: string; page?: number; sheet?: string; row_start?: number; row_end?: number };
export type DocumentProposal = { id: string; kind: string; title: string; statement: string; confidence: number; anchor: DocumentAnchor; status: ProposalStatus; reviewed_by?: string; reviewed_at?: string; review_note?: string; handoff?: ProposalHandoff };

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

type DocumentDetailBase = Omit<DocumentImportSummary,
  "pending_proposal_count" | "reviewed_proposal_count" |
  "sections_total" | "sections_omitted" | "proposals_total" | "proposals_omitted" | "content_truncated" | "processed_at">;

export type DocumentImport = DocumentDetailBase & {
  storage_key: string;
  extraction_method: string;
  analysis_method: string;
  limitations: string[];
  sections: DocumentSection[];
  proposals: DocumentProposal[];
  sections_total?: number;
  sections_omitted?: number;
  proposals_total?: number;
  proposals_omitted?: number;
  content_truncated?: boolean;
  processed_at?: string;
  created_by: string;
};

export type HandoffReviewAction = "RETURN" | "REJECT" | "SUBMIT_FOR_AUTHORIZATION";
export type HandoffAuthorizationAction = "RETURN" | "REJECT" | "APPROVE";
export type HandoffReviewInput = {
  action: HandoffReviewAction;
  expected_document_version: number;
  expected_handoff_version: number;
  title?: string;
  statement?: string;
  target_type?: ConversionTarget;
  target_program_id?: string;
  target_program_version?: number;
  note?: string;
};
export type HandoffAuthorizationInput = {
  action: HandoffAuthorizationAction;
  expected_document_version: number;
  expected_handoff_version: number;
  note?: string;
};

export type CoverageViewStatus = "PENDING" | "COMPARING" | "READY" | "PARTIAL" | "FAILED" | "STALE";
export type CoverageClassification = "VERIFIED_COVERAGE" | "MAPPED_NO_CURRENT_EVIDENCE" | "MAPPED_CONTROL_GAP" | "PARTIAL_MATCH" | "GAP" | "NEEDS_REVIEW" | "NOT_APPLICABLE";
export type CoverageDecision = "ACCEPT_MATCH" | "REJECT_MATCH" | "NOT_APPLICABLE";
export type CoverageSuggestionType = "LINK_REQUIREMENT" | "ADD_REQUIREMENT" | "CREATE_MATTER" | "CREATE_PROGRAM";
export type CoverageSuggestionStatus = "PROPOSED" | "DISMISSED" | "APPLIED" | "FAILED";

export type CoverageCountMetric = { numerator: number; denominator: number };
export type CoverageMetrics = {
  estimated_verified: CoverageCountMetric;
  verified: CoverageCountMetric;
  requirement_mapped: CoverageCountMetric;
  control_implemented: CoverageCountMetric;
  evidence_supported: CoverageCountMetric;
};

export type CoverageScoreComponent = { name: string; weight: number; score: number; reason: string };
export type CoverageRequirementTruth = {
  requirement_id: string;
  applicable: boolean;
  applicability: string;
  control_implemented: boolean;
  evidence_supported: boolean;
  complete: boolean;
  control_ids: string[];
  evidence_contract_ids: string[];
  reasons: string[];
};
export type CoverageMatch = {
  id: string;
  program_id: string;
  program_code: string;
  program_name: string;
  program_version: number;
  requirement_id: string;
  requirement_code: string;
  requirement_title: string;
  requirement_version: number;
  score: number;
  band: "STRONG" | "POSSIBLE" | "WEAK";
  components: CoverageScoreComponent[];
  rationale: string;
  conflicts: string[];
  coverage: CoverageRequirementTruth;
};
export type CoverageReview = { decision: CoverageDecision; match_id?: string; reason?: string; reviewer_id: string; reviewed_at: string };
export type CoverageCandidate = {
  id: string;
  fingerprint: string;
  eligible: boolean;
  statement: string;
  anchor: DocumentAnchor;
  modality?: string;
  actor?: string;
  action?: string;
  object?: string;
  citations: string[];
  dates: string[];
  topics: string[];
  uncertainty: string[];
  jurisdiction?: string;
  regulator?: string;
  program_type?: string;
  classification: CoverageClassification;
  matches: CoverageMatch[];
  review?: CoverageReview;
};
export type CoverageSuggestion = {
  id: string;
  candidate_id: string;
  type: CoverageSuggestionType;
  status: CoverageSuggestionStatus;
  title: string;
  rationale: string;
  program_id?: string;
  requirement_id?: string;
  applied_type?: string;
  applied_id?: string;
  failure_message?: string;
};
export type CoverageMatter = { candidate_id: string; matter_id: string; reference: string; type: string; status: string; title: string; summary: string; score: number };
export type DocumentCoverage = {
  id?: string;
  tenant_id: string;
  legal_entity_id?: string;
  document_id: string;
  document_sha256: string;
  status: CoverageViewStatus;
  analyzer_version?: string;
  matcher_version?: string;
  scoring_policy_version?: string;
  program_snapshot_hash?: string;
  candidates: CoverageCandidate[];
  suggestions: CoverageSuggestion[];
  matters: CoverageMatter[];
  metrics: CoverageMetrics;
  limitations: string[];
  failure_message?: string;
  assessed_at?: string;
  updated_at?: string;
  version: number;
  next_cursor?: string;
};

export type CoverageReviewInput = { candidate_id: string; decision: CoverageDecision; match_id?: string; reason?: string };
export type CoverageApplyResult = { assessment: DocumentCoverage; object_type?: string; object_id?: string };
