package monitoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"encoding/json"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"

	"time"
)

type AssessmentRespondent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type AdoptedAssessmentOption struct {
	AdoptionID     string                 `json:"adoption_id"`
	TemplateCode   string                 `json:"template_code"`
	FormID         string                 `json:"form_template_id"`
	FormVersion    int64                  `json:"form_template_version"`
	Name           string                 `json:"name"`
	ProgramVersion int64                  `json:"program_version"`
	ReviewerID     string                 `json:"reviewer_id"`
	ReviewerName   string                 `json:"reviewer_name"`
	Respondents    []AssessmentRespondent `json:"respondents"`
	Ready          bool                   `json:"ready"`
	Blocker        string                 `json:"blocker,omitempty"`
}
type adoptedAssessmentRepository interface {
	AdoptedAssessmentOptions(context.Context, Actor, string) ([]AdoptedAssessmentOption, error)
	ReusableFormRevision(context.Context, string, string, string, int64) (FormTemplate, error)
}

func (s *Service) AdoptedAssessmentOptions(ctx context.Context, actor Actor, program string) ([]AdoptedAssessmentOption, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	repo, ok := s.repo.(adoptedAssessmentRepository)
	if !ok {
		return nil, ErrInactive
	}
	return repo.AdoptedAssessmentOptions(ctx, actor, program)
}
func (s *Service) startAdoptedAssessment(ctx context.Context, actor Actor, input StartCollectionInput) (evidence.Request, error) {
	if err := validateActor(actor); err != nil {
		return evidence.Request{}, err
	}
	if err := validateAssessmentName(input.AssessmentName); err != nil {
		return evidence.Request{}, err
	}
	if !formLinkUUID.MatchString(input.AssessmentKey) || input.PeriodStart.IsZero() || input.PeriodEnd.Before(input.PeriodStart) || input.Deadline.Before(input.PeriodEnd) || !input.Deadline.After(s.now()) {
		return evidence.Request{}, fmt.Errorf("%w: choose a valid assessment period and future response deadline", ErrInvalid)
	}
	repo, ok := s.repo.(adoptedAssessmentRepository)
	if !ok || s.requests == nil {
		return evidence.Request{}, ErrInactive
	}
	options, err := repo.AdoptedAssessmentOptions(ctx, actor, input.ProgramID)
	if err != nil {
		return evidence.Request{}, err
	}
	var option *AdoptedAssessmentOption
	for i := range options {
		v := &options[i]
		if v.AdoptionID == input.AdoptionID && v.FormID == input.FormTemplateID && v.FormVersion == input.FormTemplateVersion {
			option = v
			break
		}
	}
	if option == nil || !option.Ready || option.ProgramVersion != input.ExpectedProgramVersion {
		return evidence.Request{}, fmt.Errorf("%w: reload the Program, resolve requirement reviews and choose its approved questionnaire", ErrConflict)
	}
	eligible := false
	for _, r := range option.Respondents {
		if r.ID == input.RespondentPrincipalID {
			eligible = true
		}
	}
	if !eligible || input.RespondentPrincipalID == option.ReviewerID {
		return evidence.Request{}, fmt.Errorf("%w: choose a respondent from an adopted unit who is independent of the reviewer", ErrInvalid)
	}
	form, err := repo.ReusableFormRevision(ctx, actor.TenantID, actor.LegalEntityID, option.FormID, option.FormVersion)
	if err != nil {
		return evidence.Request{}, err
	}
	if form.Status != LifecycleActive && form.Status != LifecycleRetired {
		return evidence.Request{}, ErrInactive
	}
	fields := make([]evidence.Field, len(form.Fields))
	for i, f := range form.Fields {
		encoded, e := json.Marshal(f)
		if e != nil {
			return evidence.Request{}, e
		}
		if e = json.Unmarshal(encoded, &fields[i]); e != nil {
			return evidence.Request{}, e
		}
	}
	start, end := input.PeriodStart.UTC().Truncate(time.Microsecond), input.PeriodEnd.UTC().Truncate(time.Microsecond)

	facts := map[string]string{"reviewer": option.ReviewerID, "adoption_id": option.AdoptionID, "framework_template": option.TemplateCode, "program_version": fmt.Sprint(option.ProgramVersion), "legal_entity_id": actor.LegalEntityID, "reporting_period_start": input.PeriodStart.Format(time.DateOnly), "reporting_period_end": input.PeriodEnd.Format(time.DateOnly)}
	if input.AssessmentScheduleID != "" {
		facts["assessment_schedule_id"] = input.AssessmentScheduleID
	}
	return s.requests.CreateRequest(evidence.WithRequestOriginAuthority(ctx, "FRAMEWORK_ASSESSMENT"), evidence.CreateRequestInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, SubjectType: "PROGRAM", SubjectID: input.ProgramID,
		Title: assessmentName(input.AssessmentName, form.Name+" · "+input.PeriodStart.Format(time.DateOnly)), Purpose: form.Purpose, WhyYou: "Your department is assigned to provide this Program assessment and supporting records.",
		Sensitivity: string(form.Sensitivity), AudienceType: "INTERNAL", Recipient: evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: input.RespondentPrincipalID},
		EstimatedMinutes: estimateMinutes(len(fields)), Deadline: input.Deadline.UTC().Truncate(time.Microsecond), CreatedBy: actor.PrincipalID,
		KnownFacts:   facts,
		Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: fields,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, CollectionPeriodStart: &start, CollectionPeriodEnd: &end,
		Origin: adoptedAssessmentOrigin(input.AdoptionID, input.AssessmentKey, input.RespondentPrincipalID),
	})
}

func adoptedAssessmentOrigin(adoptionID, key, respondentID string) evidence.RequestOrigin {
	digest := sha256.Sum256([]byte(strings.Join([]string{adoptionID, key, respondentID}, ":")))
	return evidence.RequestOrigin{Type: "FRAMEWORK_ASSESSMENT", ID: hex.EncodeToString(digest[:]), Version: 1}
}

func validateAssessmentName(name string) error {
	if utf8.RuneCountInString(strings.TrimSpace(name)) > 150 {
		return fmt.Errorf("%w: assessment name must contain at most 150 characters", ErrInvalid)
	}
	return nil
}

func assessmentName(name, fallback string) string {
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return trimmed
	}
	return fallback
}
