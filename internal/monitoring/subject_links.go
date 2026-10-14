package monitoring

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

const FormSubjectMatter = "MATTER"

var (
	ErrFormSubjectLinksUnavailable = errors.New("form subject links unavailable")
	formLinkUUID                   = regexp.MustCompile(`^[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$`)
)

// FormSubjectLink refers to an exact governed form revision. It is independent
// of the template's immutable origin and of any distribution or response.
type FormSubjectLink struct {
	ID                  string          `json:"id"`
	SubjectType         string          `json:"subject_type"`
	SubjectID           string          `json:"subject_id"`
	FormTemplateID      string          `json:"form_template_id"`
	FormTemplateVersion int64           `json:"form_template_version"`
	FormName            string          `json:"form_name"`
	FormStatus          LifecycleStatus `json:"form_status"`
	LinkedBy            string          `json:"linked_by"`
	LinkedAt            time.Time       `json:"linked_at"`
	EndedAt             *time.Time      `json:"ended_at,omitempty"`
	Version             int64           `json:"version"`
	TenantID            string          `json:"-"`
	LegalEntityID       string          `json:"-"`
}

type FormSubjectLinkPage struct {
	Items      []FormSubjectLink `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// Independent association repository; canonical forms and distributions are
// never mutated by these operations.
type FormSubjectLinkRepository interface {
	CreateFormSubjectLink(context.Context, FormSubjectLink) (FormSubjectLink, error)
	ListFormSubjectLinks(context.Context, string, string, string, string, string, int) (FormSubjectLinkPage, error)
	EndFormSubjectLink(context.Context, string, string, string, string, string, int64, string, time.Time, string) (FormSubjectLink, error)
}

func (s *Service) formSubjectLinkRepository() (FormSubjectLinkRepository, error) {
	repo, ok := s.repo.(FormSubjectLinkRepository)
	if !ok {
		return nil, ErrFormSubjectLinksUnavailable
	}
	return repo, nil
}

func validateFormSubject(subjectType, subjectID string) error {
	if subjectType != FormSubjectMatter || !formLinkUUID.MatchString(subjectID) {
		return errors.Join(ErrInvalid, fmt.Errorf("select a valid issue"))
	}
	return nil
}

// LinkExistingForm is retry-safe. It never creates or dispatches a form.
func (s *Service) LinkExistingForm(ctx context.Context, subjectType, subjectID, formID string, revision int64) (FormSubjectLink, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormSubjectLink{}, err
	}
	if err := validateFormSubject(subjectType, subjectID); err != nil {
		return FormSubjectLink{}, err
	}
	if !formLinkUUID.MatchString(formID) || revision < 1 {
		return FormSubjectLink{}, errors.Join(ErrInvalid, fmt.Errorf("select an approved form revision"))
	}
	repo, err := s.formSubjectLinkRepository()
	if err != nil {
		return FormSubjectLink{}, err
	}
	form, err := s.repo.FormLibraryItem(ctx, actor.TenantID, actor.LegalEntityID, formID)
	if err != nil {
		return FormSubjectLink{}, err
	}
	if form.ActiveStatus != LifecycleActive || form.ActiveVersion != revision {
		return FormSubjectLink{}, ErrInactive
	}
	if err := s.authorizeFormCommand(ctx, actor, "MATTER", subjectID, authority.ResponsibilityOwner, "forms.subject.link", 2); err != nil {
		return FormSubjectLink{}, err
	}
	linkID, err := s.newID()
	if err != nil {
		return FormSubjectLink{}, err
	}
	return repo.CreateFormSubjectLink(ctx, FormSubjectLink{
		ID: linkID, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		SubjectType: subjectType, SubjectID: subjectID,
		FormTemplateID: formID, FormTemplateVersion: revision,
		FormName: form.Template.Name, FormStatus: form.ActiveStatus,
		LinkedBy: actor.PrincipalID, LinkedAt: s.now().UTC(), Version: 1,
	})
}

func (s *Service) ListLinkedForms(ctx context.Context, subjectType, subjectID, cursor string, limit int) (FormSubjectLinkPage, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormSubjectLinkPage{}, err
	}
	if err := validateFormSubject(subjectType, subjectID); err != nil {
		return FormSubjectLinkPage{}, err
	}
	cursor = strings.TrimSpace(cursor)
	if cursor != "" && !formLinkUUID.MatchString(cursor) {
		return FormSubjectLinkPage{}, errors.Join(ErrInvalid, fmt.Errorf("invalid form link cursor"))
	}
	if limit < 1 || limit > 50 {
		return FormSubjectLinkPage{}, errors.Join(ErrInvalid, fmt.Errorf("choose 1 to 50 links"))
	}
	repo, err := s.formSubjectLinkRepository()
	if err != nil {
		return FormSubjectLinkPage{}, err
	}
	return repo.ListFormSubjectLinks(ctx, actor.TenantID, actor.LegalEntityID, subjectType, subjectID, cursor, limit)
}

func (s *Service) UnlinkForm(ctx context.Context, subjectType, subjectID, linkID string, expectedVersion int64, reason string) (FormSubjectLink, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormSubjectLink{}, err
	}
	if err := validateFormSubject(subjectType, subjectID); err != nil {
		return FormSubjectLink{}, err
	}
	if !formLinkUUID.MatchString(linkID) || expectedVersion < 1 || len(strings.TrimSpace(reason)) == 0 || len(reason) > 1000 {
		return FormSubjectLink{}, errors.Join(ErrInvalid, fmt.Errorf("link version and reason are required"))
	}
	repo, err := s.formSubjectLinkRepository()
	if err != nil {
		return FormSubjectLink{}, err
	}
	if err := s.authorizeFormCommand(ctx, actor, "MATTER", subjectID, authority.ResponsibilityOwner, "forms.subject.unlink", 2); err != nil {
		return FormSubjectLink{}, err
	}
	return repo.EndFormSubjectLink(ctx, actor.TenantID, actor.LegalEntityID, subjectType, subjectID, linkID, expectedVersion, actor.PrincipalID, s.now().UTC(), strings.TrimSpace(reason))
}
