package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"strings"
	"time"
)

var (
	ErrAssessmentConflict  = errors.New("response assessment has changed")
	ErrAssessmentForbidden = errors.New("response assessment authority is unavailable")
	ErrAssessmentInvalid   = errors.New("response assessment is invalid")
)

type FieldAssessmentInput struct {
	FieldID   string `json:"field_id"`
	OutcomeID string `json:"outcome_id"`
	Rationale string `json:"rationale"`
}
type RecordResponseAssessmentInput struct {
	ExpectedVersion int64                  `json:"expected_version"`
	Decisions       []FieldAssessmentInput `json:"decisions"`
}
type FieldAssessmentDecision struct {
	ID                  string    `json:"id"`
	ResponseID          string    `json:"response_id"`
	ResponseRevision    int64     `json:"response_revision"`
	FormTemplateID      string    `json:"form_template_id"`
	FormTemplateVersion int64     `json:"form_template_version"`
	FieldID             string    `json:"field_id"`
	FieldChecksum       string    `json:"field_checksum"`
	OutcomeID           string    `json:"outcome_id"`
	Points              int       `json:"points"`
	Rationale           string    `json:"rationale"`
	ReviewerID          string    `json:"reviewer_id"`
	AuthorityRoute      string    `json:"authority_route"`
	AssessedAt          time.Time `json:"assessed_at"`
	SupersedesID        string    `json:"supersedes_id,omitempty"`
	EvidenceReferences  []string  `json:"evidence_references,omitempty"`
}
type ResponseAssessmentField struct {
	MayReview bool                     `json:"may_review"`
	Field     formcontract.Field       `json:"field"`
	Answer    formcontract.AnswerValue `json:"answer"`
	Decision  *FieldAssessmentDecision `json:"decision,omitempty"`
}
type ResponseAssessmentSummary struct {
	Version               int64                `json:"version"`
	State                 string               `json:"state"`
	RequiredCount         int                  `json:"required_count"`
	ReviewedRequiredCount int                  `json:"reviewed_required_count"`
	ReviewedCount         int                  `json:"reviewed_count"`
	Score                 *ResponseScoreResult `json:"score"`
}
type ResponseAssessment struct {
	MayReview             bool                       `json:"may_review"`
	ScoreProfile          *formcontract.ScoreProfile `json:"score_profile,omitempty"`
	ResponseID            string                     `json:"response_id"`
	FormTemplateID        string                     `json:"form_template_id"`
	FormTemplateVersion   int64                      `json:"form_template_version"`
	Version               int64                      `json:"version"`
	Current               bool                       `json:"current"`
	State                 string                     `json:"state"`
	RequiredCount         int                        `json:"required_count"`
	ReviewedRequiredCount int                        `json:"reviewed_required_count"`
	ReviewedCount         int                        `json:"reviewed_count"`
	Fields                []ResponseAssessmentField  `json:"fields"`
	AutomaticScore        *ResponseScoreResult       `json:"automatic_score"`
	AssessedScore         *ResponseScoreResult       `json:"assessed_score"`
}

func (v ResponseAssessment) Summary() *ResponseAssessmentSummary {
	return &ResponseAssessmentSummary{Version: v.Version, State: v.State, RequiredCount: v.RequiredCount, ReviewedRequiredCount: v.ReviewedRequiredCount, ReviewedCount: v.ReviewedCount, Score: v.AssessedScore}
}

type ResponseAssessmentAuthorizer func(context.Context, identity.Actor, CompletedResponseSummary, formcontract.Field) (string, error)
type assessmentSnapshot struct {
	Version   int64
	Decisions map[string]FieldAssessmentDecision
	Result    ResponseAssessmentSummary
}
type assessmentMaterial struct {
	Summary    CompletedResponseSummary
	Revision   ResponseRevision
	Request    Request
	Submission Submission
	Snapshot   assessmentSnapshot
}
type assessmentReadAuthorizer func(context.Context, assessmentMaterial) bool

type responseAssessmentStore interface {
	ReadResponseAssessment(context.Context, string, string, string, string, assessmentReadAuthorizer) (assessmentMaterial, error)
	WriteResponseAssessment(context.Context, string, string, string, string, assessmentReadAuthorizer, func(context.Context, assessmentMaterial) (assessmentSnapshot, []FieldAssessmentDecision, error)) (assessmentMaterial, error)
	ReadAssessmentSummary(context.Context, string, string) (*ResponseAssessmentSummary, error)
}

func (s *DistributionService) WithAssessmentAuthorizer(a ResponseAssessmentAuthorizer) *DistributionService {
	s.assessmentAuthorizer = a
	return s
}
func (s *DistributionService) GetResponseAssessment(ctx context.Context, tenant, entity, principal, response string) (ResponseAssessment, error) {
	if s == nil || s.store == nil || tenant == "" || entity == "" || principal == "" || response == "" {
		return ResponseAssessment{}, ErrAssessmentInvalid
	}
	store, ok := s.store.(responseAssessmentStore)
	if !ok {
		return ResponseAssessment{}, ErrAssessmentInvalid
	}
	m, err := store.ReadResponseAssessment(ctx, tenant, entity, principal, response, s.assessmentReadAuthority(ctx, tenant, entity, principal))
	if err != nil {
		return ResponseAssessment{}, err
	}
	v, err := buildAssessment(m)
	if err == nil {
		s.applyAssessmentPermissions(ctx, m, &v)
	}
	return v, err
}
func (s *DistributionService) RecordResponseAssessment(ctx context.Context, tenant, entity, response string, input RecordResponseAssessmentInput) (ResponseAssessment, error) {
	actor, ok := identity.FromContext(ctx)
	if s == nil || !ok || actor.Valid(s.currentTime()) != nil || actor.TenantID != tenant || (actor.LegalEntityID != entity && actor.LegalEntityID != "*") || s.assessmentAuthorizer == nil {
		return ResponseAssessment{}, ErrAssessmentForbidden
	}
	if input.ExpectedVersion < 0 || len(input.Decisions) < 1 || len(input.Decisions) > formcontract.MaxFields {
		return ResponseAssessment{}, ErrAssessmentInvalid
	}
	store, ok := s.store.(responseAssessmentStore)
	if !ok {
		return ResponseAssessment{}, ErrAssessmentInvalid
	}
	var committed ResponseAssessment
	_, err := store.WriteResponseAssessment(ctx, tenant, entity, actor.PrincipalID, response, s.assessmentReadAuthority(ctx, tenant, entity, actor.PrincipalID), func(ctx context.Context, m assessmentMaterial) (assessmentSnapshot, []FieldAssessmentDecision, error) {
		if m.Submission.SubmittedBy == actor.PrincipalID {
			return assessmentSnapshot{}, nil, ErrAssessmentForbidden
		}
		if !m.Revision.Current || m.Snapshot.Version != input.ExpectedVersion {
			return assessmentSnapshot{}, nil, ErrAssessmentConflict
		}
		v, err := buildAssessment(m)
		if err != nil {
			return assessmentSnapshot{}, nil, err
		}
		fields := map[string]ResponseAssessmentField{}
		for _, f := range v.Fields {
			fields[f.Field.ID] = f
		}
		next := assessmentSnapshot{Version: m.Snapshot.Version + 1, Decisions: map[string]FieldAssessmentDecision{}}
		for k, d := range m.Snapshot.Decisions {
			next.Decisions[k] = d
		}
		changed := []FieldAssessmentDecision{}
		seen := map[string]bool{}
		for _, i := range input.Decisions {
			f, found := fields[i.FieldID]
			if !found || !f.Field.Assessment.NeedsReview() || seen[i.FieldID] || strings.TrimSpace(i.Rationale) == "" || len(i.Rationale) > 4000 {
				return assessmentSnapshot{}, nil, ErrAssessmentInvalid
			}
			seen[i.FieldID] = true
			var points int
			valid := false
			for _, r := range f.Field.Assessment.Rubric {
				if r.ID == i.OutcomeID {
					valid = true
					points = r.Points
					break
				}
			}
			if !valid {
				return assessmentSnapshot{}, nil, ErrAssessmentInvalid
			}
			route, e := s.assessmentAuthorizer(ctx, actor, m.Summary, f.Field)
			if e != nil || strings.TrimSpace(route) == "" {
				return assessmentSnapshot{}, nil, ErrAssessmentForbidden
			}
			decisionID, e := id.NewUUIDv7()
			if e != nil {
				return assessmentSnapshot{}, nil, e
			}
			encoded, e := json.Marshal(f.Field)
			if e != nil {
				return assessmentSnapshot{}, nil, e
			}
			digest := sha256.Sum256(encoded)
			d := FieldAssessmentDecision{ID: decisionID, ResponseID: response, ResponseRevision: m.Revision.Revision, FormTemplateID: m.Summary.FormTemplateID, FormTemplateVersion: m.Summary.FormTemplateVersion, FieldID: i.FieldID, FieldChecksum: hex.EncodeToString(digest[:]), OutcomeID: i.OutcomeID, Points: points, Rationale: strings.TrimSpace(i.Rationale), ReviewerID: actor.PrincipalID, AuthorityRoute: route, AssessedAt: s.currentTime(), EvidenceReferences: append([]string(nil), f.Answer.ArtifactIDs...)}
			if f.Answer.Document != nil {
				d.EvidenceReferences = append(d.EvidenceReferences, f.Answer.Document.ArtifactID)
			}
			if old, ok := next.Decisions[i.FieldID]; ok {
				d.SupersedesID = old.ID
			}
			next.Decisions[i.FieldID] = d
			changed = append(changed, d)
		}
		m.Snapshot = next
		result, e := buildAssessment(m)
		if e != nil {
			return assessmentSnapshot{}, nil, e
		}
		next.Result = *result.Summary()
		next.Result.Score.CalculatedAt = s.currentTime()
		s.applyAssessmentPermissions(ctx, m, &result)
		committed = result
		return next, changed, nil
	})
	if err != nil {
		return ResponseAssessment{}, err
	}
	return committed, nil
}
func buildAssessment(m assessmentMaterial) (ResponseAssessment, error) {
	contract, err := workspaceScoringContract(m.Request)
	if err != nil {
		return ResponseAssessment{}, err
	}
	visible, err := formcontract.VisibleFields(contract, m.Submission.Answers)
	if err != nil {
		return ResponseAssessment{}, err
	}
	v := ResponseAssessment{ResponseID: m.Revision.ID, FormTemplateID: m.Summary.FormTemplateID, FormTemplateVersion: m.Summary.FormTemplateVersion, Current: m.Revision.Current, Version: m.Snapshot.Version, State: "NOT_REQUIRED", Fields: []ResponseAssessmentField{}, AutomaticScore: cloneResponseRevision(m.Revision).Score}
	v.ScoreProfile = contract.ScoreProfile
	outcomes := map[string]string{}
	reviewFields := 0
	for _, f := range visible {
		field := ResponseAssessmentField{Field: f, Answer: m.Submission.Answers[f.ID]}
		if d, ok := m.Snapshot.Decisions[f.ID]; ok {
			copy := d
			field.Decision = &copy
			outcomes[f.ID] = d.OutcomeID
			v.ReviewedCount++
		}
		if f.Assessment.NeedsReview() {
			reviewFields++
			if f.Assessment.Required {
				v.RequiredCount++
				if field.Decision != nil {
					v.ReviewedRequiredCount++
				}
			}
		}
		v.Fields = append(v.Fields, field)
	}
	if reviewFields == 0 {
		v.AssessedScore = cloneResponseRevision(m.Revision).Score
		return v, nil
	}
	v.State = "AWAITING_REVIEW"
	if v.ReviewedCount > 0 {
		v.State = "IN_REVIEW"
	}
	if v.RequiredCount == v.ReviewedRequiredCount && (v.RequiredCount > 0 || v.ReviewedCount > 0) {
		v.State = "ASSESSED"
	}
	result, err := formcontract.EvaluateAssessment(contract, m.Submission.Answers, outcomes)
	if err != nil {
		return ResponseAssessment{}, err
	}
	mode := contract.ScoringMode
	if mode == formcontract.ScoringNone {
		mode = formcontract.ScoringRisk
	}
	direction := formcontract.DirectionHighIsPoor
	if mode == formcontract.ScoringCompliance {
		direction = formcontract.DirectionLowIsPoor
	}
	state := ResponseScoreProvisional
	if result.Final && v.State == "ASSESSED" {
		state = ResponseScoreFinal
	}
	v.AssessedScore = &ResponseScoreResult{Mode: mode, Direction: direction, RawScore: result.RawScore, AdverseScore: result.AdverseScore, Band: result.Band, Coverage: result.Coverage, Final: state == ResponseScoreFinal, State: state, ProfileVersion: "field-assessment-v1", EvaluatorVersion: "formcontract-assessment-v1", ContributionResults: result.ContributionResults, RuleResults: result.RuleResults}
	encoded, err := json.Marshal(contract)
	if err != nil {
		return ResponseAssessment{}, err
	}
	digest := sha256.Sum256(encoded)
	v.AssessedScore.ProfileChecksum = hex.EncodeToString(digest[:])
	if contract.ScoreProfile != nil {
		v.AssessedScore.ProfileVersion = contract.ScoreProfile.Version
	}
	if m.Snapshot.Version > 0 && m.Snapshot.Result.Score != nil {
		v.AssessedScore = m.Snapshot.Result.Score
	}
	if v.AutomaticScore != nil && v.AutomaticScore.Band == formcontract.ConcernCritical {
		v.AssessedScore.Band = formcontract.ConcernCritical
	}
	return v, nil
}
func (s *DistributionService) GetAssessedResponseForExecution(ctx context.Context, tenant, response string) (CompletedResponseSummary, error) {
	summary, err := s.GetCompletedResponseForExecution(ctx, tenant, response)
	if err != nil {
		return CompletedResponseSummary{}, err
	}
	store, ok := s.store.(responseAssessmentStore)
	if !ok {
		return CompletedResponseSummary{}, ErrNotFound
	}
	assessment, err := store.ReadAssessmentSummary(ctx, summary.TenantID, response)
	if err != nil {
		return CompletedResponseSummary{}, err
	}
	if assessment == nil || assessment.State != "ASSESSED" || assessment.Score == nil || !assessment.Score.Final || !summary.Current {
		return CompletedResponseSummary{}, ErrNotFound
	}
	summary.BankAssessment = assessment
	summary.Score = assessment.Score
	return summary, nil
}
func assessmentEvent(m assessmentMaterial, actor string, now time.Time) distributionEvent {
	return distributionEvent{DistributionID: m.Revision.DistributionID, Version: m.Snapshot.Version, EventType: "FORM_RESPONSE_ASSESSED", ActorID: actor, OccurredAt: now, Payload: map[string]any{"version": m.Snapshot.Version, "response_revision_id": m.Revision.ID, "form_template_id": m.Summary.FormTemplateID, "form_template_version": m.Summary.FormTemplateVersion, "assessment_version": m.Snapshot.Version, "result_basis": "BANK_ASSESSED", "score_state": string(m.Snapshot.Result.Score.State)}}
}

// WithAssessedResponseForExecution coordinates an in-memory policy commit with
// assessment corrections. Durable policy repositories must perform the same
// version check under the response row lock in their own material transaction.
// The callback must not re-enter the distribution store.
func (s *DistributionService) WithAssessedResponseForExecution(ctx context.Context, tenant, response string, version int64, apply func() error) error {
	if s == nil || apply == nil || version < 1 {
		return ErrAssessmentInvalid
	}
	guard, ok := s.store.(interface {
		WithAssessedResponseForExecution(context.Context, string, string, int64, func() error) error
	})
	if !ok {
		return ErrAssessmentForbidden
	}
	return guard.WithAssessedResponseForExecution(ctx, tenant, response, version, apply)
}

func (s *DistributionService) assessmentReadAuthority(ctx context.Context, tenant, entity, principal string) assessmentReadAuthorizer {
	return func(ctx context.Context, m assessmentMaterial) bool {
		actor, ok := identity.FromContext(ctx)
		if !ok || actor.Valid(s.currentTime()) != nil || actor.TenantID != tenant || actor.PrincipalID != principal || (actor.LegalEntityID != entity && actor.LegalEntityID != "*") || (s.assessmentAuthorizer == nil && s.responseDiscoveryAuthorizer == nil) || m.Submission.SubmittedBy == principal {
			return false
		}
		if s.responseDiscoveryAuthorizer != nil {
			reviewable := false
			for _, field := range m.Request.Fields {
				if field.Assessment.NeedsReview() {
					reviewable = true
					break
				}
			}
			return reviewable && s.responseDiscoveryAuthorizer(ctx, actor, m.Summary) == nil
		}
		v, err := buildAssessment(m)
		if err != nil {
			return false
		}
		for _, f := range v.Fields {
			if !f.Field.Assessment.NeedsReview() {
				continue
			}
			route, err := s.assessmentAuthorizer(ctx, actor, m.Summary, f.Field)
			if err == nil && strings.TrimSpace(route) != "" {
				return true
			}
		}
		return false
	}
}
func (s *DistributionService) applyAssessmentPermissions(ctx context.Context, m assessmentMaterial, v *ResponseAssessment) {
	actor, ok := identity.FromContext(ctx)
	if !ok || actor.Valid(s.currentTime()) != nil || actor.TenantID != m.Summary.TenantID || (actor.LegalEntityID != m.Summary.LegalEntityID && actor.LegalEntityID != "*") || s.assessmentAuthorizer == nil || m.Submission.SubmittedBy == actor.PrincipalID || !v.Current {
		return
	}
	for i := range v.Fields {
		f := &v.Fields[i]
		if !f.Field.Assessment.NeedsReview() {
			continue
		}
		route, err := s.assessmentAuthorizer(ctx, actor, m.Summary, f.Field)
		f.MayReview = err == nil && strings.TrimSpace(route) != ""
		v.MayReview = v.MayReview || f.MayReview
	}
}
