package workflow

import (
	"context"
	"fmt"
	"strings"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, input CreateInput) (Task, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.WorkflowID) == "" || strings.TrimSpace(input.StepKey) == "" || strings.TrimSpace(input.Responsibility) == "" || strings.TrimSpace(input.Title) == "" {
		return Task{}, fmt.Errorf("tenant, workflow, step, responsibility and title are required")
	}
	return s.repo.Create(ctx, input)
}

func (s *Service) Get(ctx context.Context, tenantID, id string) (Task, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(id) == "" {
		return Task{}, fmt.Errorf("tenant_id and task id are required")
	}
	return s.repo.Get(ctx, tenantID, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	if strings.TrimSpace(filter.TenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	return s.repo.List(ctx, filter)
}

func (s *Service) Transition(ctx context.Context, id string, input TransitionInput) (Task, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(id) == "" {
		return Task{}, fmt.Errorf("tenant_id and task id are required")
	}
	if input.ExpectedVersion <= 0 {
		return Task{}, fmt.Errorf("expected_version must be positive")
	}
	current, err := s.repo.Get(ctx, input.TenantID, id)
	if err != nil {
		return Task{}, err
	}
	if !allowed(current.Status, input.Status) {
		return Task{}, ErrInvalidTransition
	}
	return s.repo.Transition(ctx, id, input)
}

func allowed(from, to Status) bool {
	switch from {
	case StatusReady:
		return to == StatusInProgress || to == StatusCancelled || to == StatusEscalated
	case StatusInProgress:
		return to == StatusBlocked || to == StatusCompleted || to == StatusEscalated || to == StatusCancelled
	case StatusBlocked:
		return to == StatusInProgress || to == StatusEscalated || to == StatusCancelled
	case StatusEscalated:
		return to == StatusInProgress || to == StatusCompleted || to == StatusCancelled
	default:
		return false
	}
}
