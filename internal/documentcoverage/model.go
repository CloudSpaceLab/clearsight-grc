package documentcoverage

import (
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
)

const (
	AnalyzerVersion        = "STRUCTURED_OBLIGATION_V1"
	MatcherVersion         = "EXPLAINABLE_MATCHER_V1"
	ScoringPolicyVersion   = "COVERAGE_POLICY_V1"
	StrongMatchThreshold   = 0.85
	PossibleMatchThreshold = 0.55
)

type AssessmentStatus string
type ViewStatus string
type Classification string
type MatchBand string
type Decision string
type SuggestionType string
type SuggestionStatus string

const (
	AssessmentPending   AssessmentStatus = "PENDING"
	AssessmentComparing AssessmentStatus = "COMPARING"
	AssessmentReady     AssessmentStatus = "READY"
	AssessmentPartial   AssessmentStatus = "PARTIAL"
	AssessmentFailed    AssessmentStatus = "FAILED"

	ViewPending   ViewStatus = "PENDING"
	ViewComparing ViewStatus = "COMPARING"
	ViewReady     ViewStatus = "READY"
	ViewPartial   ViewStatus = "PARTIAL"
	ViewFailed    ViewStatus = "FAILED"
	ViewStale     ViewStatus = "STALE"

	ClassificationVerified      Classification = "VERIFIED_COVERAGE"
	ClassificationNoEvidence    Classification = "MAPPED_NO_CURRENT_EVIDENCE"
	ClassificationControlGap    Classification = "MAPPED_CONTROL_GAP"
	ClassificationPartialMatch  Classification = "PARTIAL_MATCH"
	ClassificationGap           Classification = "GAP"
	ClassificationNeedsReview   Classification = "NEEDS_REVIEW"
	ClassificationNotApplicable Classification = "NOT_APPLICABLE"

	MatchStrong   MatchBand = "STRONG"
	MatchPossible MatchBand = "POSSIBLE"
	MatchWeak     MatchBand = "WEAK"

	DecisionAccept        Decision = "ACCEPT_MATCH"
	DecisionReject        Decision = "REJECT_MATCH"
	DecisionNotApplicable Decision = "NOT_APPLICABLE"

	SuggestionLinkRequirement SuggestionType = "LINK_REQUIREMENT"
	SuggestionAddRequirement  SuggestionType = "ADD_REQUIREMENT"
	SuggestionCreateMatter    SuggestionType = "CREATE_MATTER"
	SuggestionCreateProgram   SuggestionType = "CREATE_PROGRAM"

	SuggestionProposed  SuggestionStatus = "PROPOSED"
	SuggestionDismissed SuggestionStatus = "DISMISSED"
	SuggestionApplied   SuggestionStatus = "APPLIED"
	SuggestionFailed    SuggestionStatus = "FAILED"
)

type Candidate struct {
	ID             string                `json:"id"`
	Fingerprint    string                `json:"fingerprint"`
	Eligible       bool                  `json:"eligible"`
	Statement      string                `json:"statement"`
	Anchor         documentimport.Anchor `json:"anchor"`
	Modality       string                `json:"modality,omitempty"`
	Actor          string                `json:"actor,omitempty"`
	Action         string                `json:"action,omitempty"`
	Object         string                `json:"object,omitempty"`
	Citations      []string              `json:"citations"`
	Dates          []string              `json:"dates"`
	Topics         []string              `json:"topics"`
	Uncertainty    []string              `json:"uncertainty"`
	TenantID       string                `json:"tenant_id,omitempty"`
	LegalEntityID  string                `json:"legal_entity_id,omitempty"`
	Jurisdiction   string                `json:"jurisdiction,omitempty"`
	Regulator      string                `json:"regulator,omitempty"`
	ProgramType    string                `json:"program_type,omitempty"`
	Classification Classification        `json:"classification"`
	Matches        []Match               `json:"matches"`
	Review         *ReviewDecision       `json:"review,omitempty"`
}

type RequirementTarget struct {
	ID            string                         `json:"id"`
	Code          string                         `json:"code"`
	Title         string                         `json:"title"`
	Statement     string                         `json:"statement"`
	SourceAnchor  string                         `json:"source_anchor,omitempty"`
	Status        continuity.RequirementStatus   `json:"status"`
	Version       int64                          `json:"version"`
	Modality      string                         `json:"modality,omitempty"`
	Actor         string                         `json:"actor,omitempty"`
	Action        string                         `json:"action,omitempty"`
	Object        string                         `json:"object,omitempty"`
	Citations     []string                       `json:"citations"`
	Dates         []string                       `json:"dates"`
	Topics        []string                       `json:"topics"`
	Applicability continuity.ApplicabilityStatus `json:"applicability"`
	Coverage      continuity.RequirementCoverage `json:"coverage"`
}

type ProgramSnapshot struct {
	TenantID      string                   `json:"tenant_id"`
	LegalEntityID string                   `json:"legal_entity_id,omitempty"`
	ProgramID     string                   `json:"program_id"`
	Code          string                   `json:"code"`
	Name          string                   `json:"name"`
	Type          string                   `json:"type"`
	Status        continuity.ProgramStatus `json:"status"`
	Jurisdiction  string                   `json:"jurisdiction,omitempty"`
	Regulator     string                   `json:"regulator,omitempty"`
	Version       int64                    `json:"version"`
	Requirements  []RequirementTarget      `json:"requirements"`
}

type ScoreComponent struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type Match struct {
	ID                 string                         `json:"id"`
	ProgramID          string                         `json:"program_id"`
	ProgramCode        string                         `json:"program_code"`
	ProgramName        string                         `json:"program_name"`
	ProgramVersion     int64                          `json:"program_version"`
	RequirementID      string                         `json:"requirement_id"`
	RequirementCode    string                         `json:"requirement_code"`
	RequirementTitle   string                         `json:"requirement_title"`
	RequirementVersion int64                          `json:"requirement_version"`
	Score              float64                        `json:"score"`
	Band               MatchBand                      `json:"band"`
	Components         []ScoreComponent               `json:"components"`
	Rationale          string                         `json:"rationale"`
	Conflicts          []string                       `json:"conflicts"`
	Coverage           continuity.RequirementCoverage `json:"coverage"`
}

type ReviewDecision struct {
	Decision   Decision  `json:"decision"`
	MatchID    string    `json:"match_id,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	ReviewerID string    `json:"reviewer_id"`
	ReviewedAt time.Time `json:"reviewed_at"`
}

type Suggestion struct {
	ID             string           `json:"id"`
	CandidateID    string           `json:"candidate_id"`
	Type           SuggestionType   `json:"type"`
	Status         SuggestionStatus `json:"status"`
	Title          string           `json:"title"`
	Rationale      string           `json:"rationale"`
	ProgramID      string           `json:"program_id,omitempty"`
	RequirementID  string           `json:"requirement_id,omitempty"`
	AppliedType    string           `json:"applied_type,omitempty"`
	AppliedID      string           `json:"applied_id,omitempty"`
	FailureMessage string           `json:"failure_message,omitempty"`
}

type CountMetric struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

type Metrics struct {
	EstimatedVerified  CountMetric `json:"estimated_verified"`
	Verified           CountMetric `json:"verified"`
	RequirementMapped  CountMetric `json:"requirement_mapped"`
	ControlImplemented CountMetric `json:"control_implemented"`
	EvidenceSupported  CountMetric `json:"evidence_supported"`
}

type Evaluation struct {
	Candidates  []Candidate  `json:"candidates"`
	Suggestions []Suggestion `json:"suggestions"`
	Metrics     Metrics      `json:"metrics"`
}

type Assessment struct {
	ID                   string           `json:"id"`
	TenantID             string           `json:"tenant_id"`
	LegalEntityID        string           `json:"legal_entity_id,omitempty"`
	DocumentID           string           `json:"document_id"`
	DocumentSHA256       string           `json:"document_sha256"`
	Status               AssessmentStatus `json:"status"`
	AnalyzerVersion      string           `json:"analyzer_version"`
	MatcherVersion       string           `json:"matcher_version"`
	ScoringPolicyVersion string           `json:"scoring_policy_version"`
	ProgramSnapshotHash  string           `json:"program_snapshot_hash"`
	Candidates           []Candidate      `json:"candidates"`
	Suggestions          []Suggestion     `json:"suggestions"`
	Metrics              Metrics          `json:"metrics"`
	Limitations          []string         `json:"limitations"`
	FailureMessage       string           `json:"failure_message,omitempty"`
	AssessedAt           time.Time        `json:"assessed_at"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
	Version              int64            `json:"version"`
}
