package monitoring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

// FrameworkQuestionnaire identifies an exact approved form revision selected
// for one adoption; selecting it does not dispatch an assessment.
type FrameworkQuestionnaire struct {
	ID            string    `json:"id"`
	ProgramID     string    `json:"program_id"`
	AdoptionID    string    `json:"adoption_id"`
	EditionID     string    `json:"edition_id"`
	FormID        string    `json:"form_template_id"`
	FormVersion   int64     `json:"form_template_version"`
	FormName      string    `json:"form_name"`
	LinkedBy      string    `json:"linked_by"`
	LinkedAt      time.Time `json:"linked_at"`
	Version       int64     `json:"version"`
	TenantID      string    `json:"-"`
	LegalEntityID string    `json:"-"`
}

type FrameworkQuestionnaireRepository interface {
	GetFrameworkQuestionnaire(context.Context, string, string, string, string) (FrameworkQuestionnaire, error)
	SetFrameworkQuestionnaire(context.Context, FrameworkQuestionnaire, int64) (FrameworkQuestionnaire, error)
}

func (s *Service) ReadFrameworkQuestionnaire(ctx context.Context, programID, adoptionID string) (FrameworkQuestionnaire, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FrameworkQuestionnaire{}, err
	}
	if !formLinkUUID.MatchString(programID) || !formLinkUUID.MatchString(adoptionID) {
		return FrameworkQuestionnaire{}, ErrInvalid
	}
	repo, ok := s.repo.(FrameworkQuestionnaireRepository)
	if !ok {
		return FrameworkQuestionnaire{}, ErrFormSubjectLinksUnavailable
	}
	return repo.GetFrameworkQuestionnaire(ctx, actor.TenantID, actor.LegalEntityID, programID, adoptionID)
}

func (s *Service) SelectFrameworkQuestionnaire(ctx context.Context, programID, adoptionID, formID string, formVersion, expectedLinkVersion int64) (FrameworkQuestionnaire, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FrameworkQuestionnaire{}, err
	}
	if !formLinkUUID.MatchString(programID) || !formLinkUUID.MatchString(adoptionID) || !formLinkUUID.MatchString(formID) || formVersion < 1 || expectedLinkVersion < 0 {
		return FrameworkQuestionnaire{}, errors.Join(ErrInvalid, fmt.Errorf("select an approved form revision"))
	}
	repo, ok := s.repo.(FrameworkQuestionnaireRepository)
	if !ok {
		return FrameworkQuestionnaire{}, ErrFormSubjectLinksUnavailable
	}
	form, err := s.repo.FormLibraryItem(ctx, actor.TenantID, actor.LegalEntityID, formID)
	if err != nil {
		return FrameworkQuestionnaire{}, err
	}
	if form.ActiveStatus != LifecycleActive || form.ActiveVersion != formVersion {
		return FrameworkQuestionnaire{}, ErrInactive
	}
	if err = s.authorizeFormCommand(ctx, actor, "PROGRAM", programID, authority.ResponsibilityOwner, "program.framework.questionnaire.select", 3); err != nil {
		return FrameworkQuestionnaire{}, err
	}
	id, err := s.newID()
	if err != nil {
		return FrameworkQuestionnaire{}, err
	}
	return repo.SetFrameworkQuestionnaire(ctx, FrameworkQuestionnaire{ID: id, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ProgramID: programID, AdoptionID: adoptionID, FormID: formID, FormVersion: formVersion, LinkedBy: actor.PrincipalID, LinkedAt: s.now().UTC()}, expectedLinkVersion)
}
