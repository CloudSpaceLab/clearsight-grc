package documentimport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// actorReadRepository filters authorization inside the bounded repository query.
// The existing scopedReadRepository is retained for internal workers only.
type actorReadRepository interface {
	ListForActor(context.Context, string, string, string, int, time.Time, time.Time) ([]DocumentSummary, error)
	GetForActor(context.Context, string, string, string, string, time.Time, time.Time) (Document, error)
}

func (s *Service) readActor(ctx context.Context) (identity.Actor, time.Time, error) {
	now := time.Now().UTC()
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || s == nil || s.repo == nil {
		return identity.Actor{}, now, ErrNotFound
	}
	return actor, now, nil
}

func (s *Service) ListVisibleForActor(ctx context.Context, limit int) ([]DocumentSummary, error) {
	actor, now, err := s.readActor(ctx)
	if err != nil {
		return nil, err
	}
	repo, ok := s.repo.(actorReadRepository)
	if !ok {
		return nil, ErrNotFound
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return repo.ListForActor(ctx, actor.TenantID, actor.LegalEntityID, actor.PrincipalID, limit, now, actor.ExpiresAt)
}

func (s *Service) GetVisibleForActor(ctx context.Context, id string) (Document, error) {
	actor, now, err := s.readActor(ctx)
	if err != nil || strings.TrimSpace(id) == "" {
		return Document{}, ErrNotFound
	}
	repo, ok := s.repo.(actorReadRepository)
	if !ok {
		return Document{}, ErrNotFound
	}
	return repo.GetForActor(ctx, actor.TenantID, actor.LegalEntityID, actor.PrincipalID, strings.TrimSpace(id), now, actor.ExpiresAt)
}

type scopedReadRepository interface {
	ListScoped(context.Context, string, string, int) ([]DocumentSummary, error)
	GetScoped(context.Context, string, string, string) (Document, error)
}

func (s *Service) ListVisible(ctx context.Context, tenant, legalEntityID string, limit int) ([]DocumentSummary, error) {
	tenant = strings.TrimSpace(tenant)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenant == "" {
		return nil, fmt.Errorf("tenant is required")
	}
	if legalEntityID == "*" {
		return s.List(ctx, tenant, limit)
	}
	repo, ok := s.repo.(scopedReadRepository)
	if !ok {
		return nil, fmt.Errorf("legal-entity-scoped document reads are unavailable")
	}
	return repo.ListScoped(ctx, tenant, legalEntityID, limit)
}

func (s *Service) GetVisible(ctx context.Context, tenant, legalEntityID, id string) (Document, error) {
	tenant = strings.TrimSpace(tenant)
	legalEntityID = strings.TrimSpace(legalEntityID)
	id = strings.TrimSpace(id)
	if tenant == "" || id == "" {
		return Document{}, ErrNotFound
	}
	if legalEntityID == "*" {
		return s.Get(ctx, tenant, id)
	}
	repo, ok := s.repo.(scopedReadRepository)
	if !ok {
		return Document{}, ErrNotFound
	}
	return repo.GetScoped(ctx, tenant, legalEntityID, id)
}
