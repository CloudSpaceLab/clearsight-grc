package controlcatalog

import (
	"context"
	"strings"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo Repository
	Now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Promote(ctx context.Context, input PromoteInput) (Definition, ImplementationLink, error) {
	if s == nil || s.repo == nil {
		return Definition{}, ImplementationLink{}, ErrInvalid
	}
	now := s.now()
	definition := Definition{
		TenantID: strings.TrimSpace(input.TenantID),
		Code: strings.ToUpper(strings.TrimSpace(input.Code)),
		Name: strings.TrimSpace(input.Name),
		Objective: strings.TrimSpace(input.Objective),
		Description: strings.TrimSpace(input.Description),
		Category: strings.TrimSpace(input.Category),
		Status: DefinitionActive,
		Version: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := validateDefinition(definition); err != nil {
		return Definition{}, ImplementationLink{}, err
	}
	definition.ID = mustNewID()
	link := canonicalLink(input.TenantID, input.LegalEntityID, definition.ID, input.ProgramID, input.ImplementationID, now)
	if err := validateLink(link); err != nil {
		return Definition{}, ImplementationLink{}, err
	}
	link.ID = mustNewID()
	return s.repo.CreateWithImplementationLink(ctx, definition, link)
}

func (s *Service) LinkImplementation(ctx context.Context, input LinkImplementationInput) (ImplementationLink, error) {
	if s == nil || s.repo == nil {
		return ImplementationLink{}, ErrInvalid
	}
	definition, err := s.repo.GetDefinition(ctx, strings.TrimSpace(input.TenantID), strings.TrimSpace(input.DefinitionID))
	if err != nil {
		return ImplementationLink{}, err
	}
	if definition.Status != DefinitionActive {
		return ImplementationLink{}, ErrInvalid
	}
	link := canonicalLink(input.TenantID, input.LegalEntityID, definition.ID, input.ProgramID, input.ImplementationID, s.now())
	if err := validateLink(link); err != nil {
		return ImplementationLink{}, err
	}
	link.ID = mustNewID()
	return s.repo.LinkImplementation(ctx, link)
}

func (s *Service) GetImplementationLink(ctx context.Context, tenant, entity, id string) (ImplementationLink, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(entity) == "" || strings.TrimSpace(id) == "" {
		return ImplementationLink{}, ErrInvalid
	}
	return s.repo.GetImplementationLink(ctx, strings.TrimSpace(tenant), strings.TrimSpace(entity), strings.TrimSpace(id))
}

func (s *Service) ListImplementationLinks(ctx context.Context, tenant, definitionID string, limit int) ([]ImplementationLink, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(definitionID) == "" {
		return nil, ErrInvalid
	}
	if limit <= 0 {
		limit = 100
	} else if limit > 500 {
		limit = 500
	}
	return s.repo.ListImplementationLinks(ctx, strings.TrimSpace(tenant), strings.TrimSpace(definitionID), limit)
}

func canonicalLink(tenant, entity, definitionID, programID, implementationID string, now time.Time) ImplementationLink {
	return ImplementationLink{
		TenantID: strings.TrimSpace(tenant),
		LegalEntityID: strings.TrimSpace(entity),
		DefinitionID: strings.TrimSpace(definitionID),
		ProgramID: strings.TrimSpace(programID),
		ImplementationID: strings.TrimSpace(implementationID),
		CreatedAt: now,
	}
}

func validateDefinition(value Definition) error {
	if value.TenantID == "" || value.Code == "" || value.Name == "" || value.Objective == "" ||
		value.Status != DefinitionActive || value.Version != 1 || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		return ErrInvalid
	}
	if len(value.Code) > 128 || len(value.Name) > 240 || len(value.Objective) > 2000 ||
		len(value.Description) > 4000 || len(value.Category) > 160 {
		return ErrInvalid
	}
	return nil
}

func validateLink(value ImplementationLink) error {
	if value.TenantID == "" || value.LegalEntityID == "" || value.LegalEntityID == "*" ||
		value.DefinitionID == "" || value.ProgramID == "" || value.ImplementationID == "" || value.CreatedAt.IsZero() {
		return ErrInvalid
	}
	return nil
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func mustNewID() string {
	value, err := platformid.NewUUIDv7()
	if err != nil {
		panic(err)
	}
	return value
}
