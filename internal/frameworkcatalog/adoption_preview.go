package frameworkcatalog

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

var ErrInvalidAdoptionPreview = errors.New("framework adoption preview is invalid")

const (
	MaxAdoptionPreviewClauses = 500
	MaxSuggestedRequirements  = 10
)

type ClauseReuseStatus string

const (
	ClauseReuseUnmapped       ClauseReuseStatus = "UNMAPPED"
	ClauseReuseReviewRequired ClauseReuseStatus = "REVIEW_CANDIDATE"
	ClauseReuseAmbiguous      ClauseReuseStatus = "AMBIGUOUS"
)

type ClauseReuseCandidate struct {
	RequirementID   string                         `json:"requirement_id"`
	RequirementCode string                         `json:"requirement_code"`
	Title           string                         `json:"title"`
	Applicability   continuity.ApplicabilityStatus `json:"applicability,omitempty"`
}

type ClauseReuseProposal struct {
	ClauseCode          string                 `json:"clause_code"`
	SourceAnchor        string                 `json:"source_anchor"`
	Status              ClauseReuseStatus      `json:"status"`
	MatchBasis          string                 `json:"match_basis,omitempty"`
	TotalCandidates     int                    `json:"total_candidates"`
	CandidatesTruncated bool                   `json:"candidates_truncated,omitempty"`
	Candidates          []ClauseReuseCandidate `json:"candidates"`
}

// PreviewClauseReuse supplies review-only suggestions for a bounded page of
// clauses. It neither adopts an edition nor concludes that any requirement,
// control, evidence or Program is compliant. The caller must authorize the
// exact actor, tenant, legal entity, Program and edition before invoking it.
//
// Only approved, effective Program Requirements are eligible for suggestion;
// CurrentRequirementCoverage owns the effective-time predicate shared with
// Program scoring. Source-anchor and code equivalence are hints, not proof
// that legal provisions or control implementations are interchangeable.
func PreviewClauseReuse(
	aggregate continuity.ProgramAggregate,
	edition Edition,
	clauses []Clause,
	at time.Time,
) ([]ClauseReuseProposal, error) {
	program := aggregate.Program
	if program.ID == "" || program.TenantID == "" || program.LegalEntityID == "" ||
		edition.ID == "" || edition.ReleaseSHA256 == "" || at.IsZero() ||
		len(clauses) > MaxAdoptionPreviewClauses ||
		(!program.EffectiveFrom.IsZero() && at.Before(program.EffectiveFrom)) ||
		(program.EffectiveUntil != nil && !at.Before(*program.EffectiveUntil)) {
		return nil, ErrInvalidAdoptionPreview
	}

	coverage := continuity.CurrentRequirementCoverage(aggregate, at)
	byAnchor := make(map[string][]ClauseReuseCandidate)
	byCode := make(map[string][]ClauseReuseCandidate)
	seenRequirements := make(map[string]struct{})
	for _, requirement := range aggregate.Requirements {
		if _, eligible := coverage[requirement.ID]; !eligible ||
			requirement.ID == "" || requirement.Code == "" ||
			(requirement.ProgramID != "" && requirement.ProgramID != program.ID) ||
			(requirement.TenantID != "" && requirement.TenantID != program.TenantID) {
			continue
		}
		if _, exists := seenRequirements[requirement.ID]; exists {
			return nil, ErrInvalidAdoptionPreview
		}
		seenRequirements[requirement.ID] = struct{}{}
		item := ClauseReuseCandidate{
			RequirementID: requirement.ID, RequirementCode: requirement.Code,
			Title: requirement.Title, Applicability: coverage[requirement.ID].Applicability,
		}
		if key := normalizeReference(requirement.SourceAnchor); key != "" {
			byAnchor[key] = append(byAnchor[key], item)
		}
		if key := normalizeReference(requirement.Code); key != "" {
			byCode[key] = append(byCode[key], item)
		}
	}

	for _, index := range []map[string][]ClauseReuseCandidate{byAnchor, byCode} {
		for key := range index {
			sort.Slice(index[key], func(i, j int) bool {
				return index[key][i].RequirementID < index[key][j].RequirementID
			})
		}
	}

	result := make([]ClauseReuseProposal, 0, len(clauses))
	seenClauses := make(map[string]struct{}, len(clauses))
	for _, clause := range clauses {
		if strings.TrimSpace(clause.Code) == "" ||
			strings.TrimSpace(clause.SourceAnchor) == "" {
			return nil, ErrInvalidAdoptionPreview
		}
		if _, exists := seenClauses[clause.Code]; exists {
			return nil, ErrInvalidAdoptionPreview
		}
		seenClauses[clause.Code] = struct{}{}
		proposal := ClauseReuseProposal{
			ClauseCode: clause.Code, SourceAnchor: clause.SourceAnchor,
			Status: ClauseReuseUnmapped, Candidates: []ClauseReuseCandidate{},
		}
		// Anchor matches are more specific than matching short clause codes.
		// Do not combine both populations or generate duplicate suggestions.
		matches := byAnchor[normalizeReference(clause.SourceAnchor)]
		if len(matches) > 0 {
			proposal.MatchBasis = "EXACT_SOURCE_ANCHOR"
		} else {
			matches = byCode[normalizeReference(clause.Code)]
			if len(matches) > 0 {
				proposal.MatchBasis = "EXACT_CODE"
			}
		}
		proposal.TotalCandidates = len(matches)
		if len(matches) != 0 {
			proposal.Status = ClauseReuseReviewRequired
			if len(matches) > 1 {
				proposal.Status = ClauseReuseAmbiguous
			}
			limit := len(matches)
			if limit > MaxSuggestedRequirements {
				limit = MaxSuggestedRequirements
				proposal.CandidatesTruncated = true
			}
			proposal.Candidates = append(proposal.Candidates, matches[:limit]...)
		}
		result = append(result, proposal)
	}
	return result, nil
}

func normalizeReference(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
