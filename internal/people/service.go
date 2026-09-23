package people

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) Profile(ctx context.Context, scope Scope) (Profile, error) {
	if err := s.authorize(scope); err != nil {
		return Profile{}, err
	}
	profile, err := s.repository.Profile(ctx, scope)
	if errors.Is(err, ErrNotFound) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	if profile.AsOf.IsZero() {
		profile.AsOf = s.now().UTC()
	}
	return profile, nil
}

func (s *Service) Work(ctx context.Context, query PageQuery) (WorkPage, error) {
	if err := s.authorize(query.Scope); err != nil {
		return WorkPage{}, err
	}
	query, err := normalizePage(query)
	if err != nil {
		return WorkPage{}, err
	}
	page, err := s.repository.Work(ctx, query)
	if err != nil {
		return WorkPage{}, err
	}
	if page.Items == nil {
		page.Items = []WorkItem{}
	}
	if page.AsOf.IsZero() {
		page.AsOf = s.now().UTC()
	}
	return page, nil
}

func (s *Service) Assignments(ctx context.Context, query PageQuery) (AssignmentPage, error) {
	if err := s.authorize(query.Scope); err != nil {
		return AssignmentPage{}, err
	}
	query, err := normalizePage(query)
	if err != nil {
		return AssignmentPage{}, err
	}
	page, err := s.repository.Assignments(ctx, query)
	if err != nil {
		return AssignmentPage{}, err
	}
	if page.Items == nil {
		page.Items = []AssignmentItem{}
	}
	if page.AsOf.IsZero() {
		page.AsOf = s.now().UTC()
	}
	return page, nil
}

func (s *Service) Activity(ctx context.Context, query PageQuery) (ActivityPage, error) {
	if err := s.authorize(query.Scope); err != nil {
		return ActivityPage{}, err
	}
	query, err := normalizePage(query)
	if err != nil {
		return ActivityPage{}, err
	}
	page, err := s.repository.Activity(ctx, query)
	if err != nil {
		return ActivityPage{}, err
	}
	if page.Items == nil {
		page.Items = []ActivityItem{}
	}
	if page.AsOf.IsZero() {
		page.AsOf = s.now().UTC()
	}
	return page, nil
}

func (s *Service) authorize(scope Scope) error {
	if s == nil || s.repository == nil || strings.TrimSpace(scope.PersonID) == "" || strings.TrimSpace(scope.Viewer.TenantID) == "" || strings.TrimSpace(scope.Viewer.LegalEntityID) == "" || strings.TrimSpace(scope.Viewer.PrincipalID) == "" {
		return ErrInvalid
	}
	if scope.Viewer.PrincipalID == strings.TrimSpace(scope.PersonID) || identity.HasPermission(scope.Viewer, identity.PermissionOversightRead) || identity.HasPermission(scope.Viewer, identity.PermissionIdentityRead) {
		return nil
	}
	return ErrNotFound
}

func normalizePage(query PageQuery) (PageQuery, error) {
	if query.Limit <= 0 {
		query.Limit = 25
	}
	if query.Limit > 100 || (query.From != nil && query.To != nil && query.From.After(*query.To)) {
		return PageQuery{}, ErrInvalid
	}
	query.Cursor = strings.TrimSpace(query.Cursor)
	query.Category = strings.ToUpper(strings.TrimSpace(query.Category))
	query.State = strings.ToUpper(strings.TrimSpace(query.State))
	if query.State != "" && query.State != "ACTIVE" && query.State != "COMPLETED" {
		return PageQuery{}, ErrInvalid
	}
	return query, nil
}
