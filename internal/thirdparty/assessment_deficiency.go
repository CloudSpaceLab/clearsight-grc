package thirdparty

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const AssessmentDeficiencyCommand = "thirdparty.assessment.deficiency.create"

var assessmentDeficiencyTriggerPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,79}$`)

type CreateAssessmentDeficiencyInput struct {
	ExpectedVersion int64      `json:"expected_version"`
	TriggerKey      string     `json:"trigger_key"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	DueAt           *time.Time `json:"due_at,omitempty"`
}

type AssessmentDeficiencyOutcome struct {
	Assessment Assessment                 `json:"assessment"`
	Matter     continuity.MatterAggregate `json:"matter"`
}

type assessmentDeficiencyMatters interface {
	MatterByTriggerKey(context.Context, string, string) (continuity.MatterAggregate, error)
	CreateMatter(context.Context, continuity.CreateMatterInput) (continuity.MatterAggregate, error)
}

type AssessmentDeficiencyService struct {
	assessments *AssessmentService
	repo        AssessmentRepository
	matters     assessmentDeficiencyMatters
}

func NewAssessmentDeficiencyService(assessments *AssessmentService, repo AssessmentRepository, matters assessmentDeficiencyMatters) *AssessmentDeficiencyService {
	return &AssessmentDeficiencyService{assessments: assessments, repo: repo, matters: matters}
}

func (s *AssessmentDeficiencyService) CreateDeficiency(ctx context.Context, _ Actor, assessmentID string, input CreateAssessmentDeficiencyInput) (AssessmentDeficiencyOutcome, error) {
	assessmentID, input.TriggerKey = strings.TrimSpace(assessmentID), strings.ToLower(strings.TrimSpace(input.TriggerKey))
	input.Title, input.Summary = strings.TrimSpace(input.Title), strings.TrimSpace(input.Summary)
	if s == nil || s.assessments == nil || s.repo == nil || s.matters == nil || !validAssessmentIdentifier(assessmentID) || input.ExpectedVersion < 1 || !assessmentDeficiencyTriggerPattern.MatchString(input.TriggerKey) || input.Title == "" || len(input.Title) > 200 || input.Summary == "" || len(input.Summary) > 2000 {
		return AssessmentDeficiencyOutcome{}, ErrInvalid
	}
	if input.DueAt != nil {
		value := input.DueAt.UTC()
		if !value.After(s.assessments.now().UTC()) {
			return AssessmentDeficiencyOutcome{}, ErrInvalid
		}
		input.DueAt = &value
	}
	verified, err := s.assessments.authorize(ctx, assessmentID, assessmentObjectType, AssessmentDeficiencyCommand, authority.ResponsibilityReviewer)
	if err != nil {
		return AssessmentDeficiencyOutcome{}, err
	}
	scope := scopeFrom(verified)
	assessment, err := s.repo.GetAssessment(ctx, scope, assessmentID)
	if err != nil {
		return AssessmentDeficiencyOutcome{}, err
	}
	if assessment.Status != AssessmentUnderReview || !validAssessmentIdentifier(assessment.ReviewMatterID) {
		return AssessmentDeficiencyOutcome{}, ErrInvalidAssessmentTransition
	}
	relationship, err := s.repo.GetRelationship(ctx, scope, assessment.RelationshipID)
	if err != nil {
		return AssessmentDeficiencyOutcome{}, err
	}
	if relationship.Relationship.Status != RelationshipProposed && relationship.Relationship.Status != RelationshipUnderReview {
		return AssessmentDeficiencyOutcome{}, ErrInvalidAssessmentTransition
	}
	triggerKey := assessmentDeficiencyTriggerKey(scope, assessment.ID, input.TriggerKey)
	matter, err := s.matters.MatterByTriggerKey(ctx, scope.TenantID, triggerKey)
	if assessment.Version != input.ExpectedVersion && errors.Is(err, continuity.ErrNotFound) {
		return AssessmentDeficiencyOutcome{}, ErrVersionConflict
	}
	if errors.Is(err, continuity.ErrNotFound) {
		matter, err = s.matters.CreateMatter(ctx, deficiencyMatterInput(verified, assessment, input, triggerKey))
		if err != nil {
			matter, err = s.matters.MatterByTriggerKey(ctx, scope.TenantID, triggerKey)
		}
	}
	if err != nil {
		return AssessmentDeficiencyOutcome{}, err
	}
	if !validDeficiencyMatter(matter, scope, assessment, triggerKey, verified.PrincipalID) {
		return AssessmentDeficiencyOutcome{}, ErrInvalid
	}
	_, updated, err := s.repo.LinkAssessmentDeficiency(ctx, LinkAssessmentDeficiencyRecord{Scope: scope, AssessmentID: assessment.ID, ExpectedVersion: input.ExpectedVersion, ActorPrincipalID: verified.PrincipalID, MatterID: matter.Matter.ID, MatterTriggerKey: triggerKey, LinkedAt: s.assessments.now().UTC()})
	if err != nil {
		return AssessmentDeficiencyOutcome{}, err
	}
	return AssessmentDeficiencyOutcome{Assessment: updated, Matter: matter}, nil
}

func assessmentDeficiencyTriggerKey(scope Scope, assessmentID, clientKey string) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{scope.TenantID, scope.LegalEntityID, assessmentID, clientKey}, "\x00")))
	return "vendor-deficiency:" + hex.EncodeToString(digest[:])
}

func deficiencyMatterInput(actor Actor, assessment Assessment, input CreateAssessmentDeficiencyInput, triggerKey string) continuity.CreateMatterInput {
	allowed := []string{actor.PrincipalID, assessment.StartedByPrincipalID}
	sort.Strings(allowed)
	if allowed[0] == allowed[1] {
		allowed = allowed[:1]
	}
	scope, _ := json.Marshal(map[string]any{"access": continuity.MatterAccessRestricted, "allowed_principal_ids": allowed, "assessment_id": assessment.ID, "relationship_id": assessment.RelationshipID, "deficiency_key": input.TriggerKey})
	known, _ := json.Marshal(map[string]string{"assessment_id": assessment.ID, "relationship_id": assessment.RelationshipID, "deficiency_key": input.TriggerKey})
	return continuity.CreateMatterInput{TenantID: assessment.TenantID, Type: continuity.MatterVendorDeficiency, Priority: 3, Title: input.Title, Summary: input.Summary, Scope: scope, TriggerType: "VENDOR_ASSESSMENT_DEFICIENCY", TriggerID: assessment.ID, TriggerKey: triggerKey, KnownFacts: known, MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: assessment.StartedByPrincipalID, RequiredAuthority: string(authority.ResponsibilityOwner), DueAt: input.DueAt, ActorID: actor.PrincipalID}
}

func validDeficiencyMatter(value continuity.MatterAggregate, scope Scope, assessment Assessment, triggerKey, principalID string) bool {
	if value.Matter.ID == "" || value.Matter.TenantID != scope.TenantID || value.Matter.Type != continuity.MatterVendorDeficiency || value.Matter.TriggerKey != triggerKey || value.Matter.TriggerID != assessment.ID {
		return false
	}
	policy, valid := continuity.ParseMatterAccessPolicy(value.Matter.Scope)
	if !valid || policy.Access != continuity.MatterAccessRestricted {
		return false
	}
	for _, allowed := range policy.AllowedPrincipalIDs {
		if allowed == principalID {
			return true
		}
	}
	return false
}
