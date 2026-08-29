package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type proposalFormPreparer interface {
	PrepareLibraryForm(context.Context, CreateFormInput) (FormTemplate, error)
	PrepareFormRevision(context.Context, string, CreateFormRevisionInput) (FormTemplate, error)
}

// PrepareLibraryForm performs the same authority and contract checks as
// CreateLibraryForm but does not persist anything. It exists so a caller that
// owns a wider PostgreSQL transaction can compose form creation atomically with
// another material state change.
func (s *Service) PrepareLibraryForm(ctx context.Context, input CreateFormInput) (FormTemplate, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := s.authorizeFormCommand(ctx, actor, "LEGAL_ENTITY", actor.LegalEntityID, authority.ResponsibilityOwner, "forms.template.create", 2); err != nil {
		return FormTemplate{}, err
	}
	valueID, err := s.newID()
	if err != nil {
		return FormTemplate{}, err
	}
	return s.prepareLibraryRevision(actor, valueID, 1, input, FormTemplate{})
}

// PrepareFormRevision validates and authorizes an exact base revision without
// writing it. The returned draft is safe to pass to a transaction-capable form
// repository writer.
func (s *Service) PrepareFormRevision(ctx context.Context, formID string, input CreateFormRevisionInput) (FormTemplate, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplate{}, err
	}
	formID = strings.TrimSpace(formID)
	if formID == "" || input.ExpectedVersion < 1 {
		return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("form and expected version are required"))
	}
	base, err := s.repo.ReusableFormRevision(ctx, actor.TenantID, actor.LegalEntityID, formID, input.ExpectedVersion)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := s.authorizeFormCommand(ctx, actor, "FORM_TEMPLATE", base.ID, authority.ResponsibilityOwner, "forms.template.revise", 2); err != nil {
		return FormTemplate{}, err
	}
	return s.prepareLibraryRevision(actor, base.ID, base.Version+1, input.Form, base)
}

func (s *Service) prepareLibraryRevision(actor identity.Actor, id string, version int64, input CreateFormInput, base FormTemplate) (FormTemplate, error) {
	contract, err := normalizeLibraryDraft(input)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := validateTextFields(input.Code, input.Name, input.Purpose); err != nil {
		return FormTemplate{}, err
	}
	ownerID := strings.TrimSpace(input.OwnerPrincipalID)
	if ownerID == "" {
		ownerID = actor.PrincipalID
	}
	now := s.now().UTC()
	value := FormTemplate{
		ID: id, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ProgramID: strings.TrimSpace(base.ProgramID),
		Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name), Purpose: strings.TrimSpace(input.Purpose),
		OwnerPrincipalID: ownerID, ResponsibleTeam: strings.TrimSpace(input.ResponsibleTeam),
		ApprovedUses: append([]string(nil), input.ApprovedUses...), Tags: append([]string(nil), input.Tags...),
		Jurisdiction: strings.TrimSpace(input.Jurisdiction), Industry: strings.TrimSpace(input.Industry), Sensitivity: strings.TrimSpace(input.Sensitivity),
		ScoringMode: contract.ScoringMode, NextReviewAt: input.NextReviewAt,
		StarterCatalogCode: base.StarterCatalogCode, StarterCatalogVersion: base.StarterCatalogVersion,
		Presentation: contract.Presentation, Sections: contract.Sections, Fields: contract.Fields,
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: version, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now},
	}
	if base.ID == "" {
		value.ProgramID = strings.TrimSpace(input.ProgramID)
	}
	return value, nil
}
