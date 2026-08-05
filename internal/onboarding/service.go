package onboarding

import (
	"context"
	"fmt"
)

type Service struct {
	repo   Repository
	guides map[string]Guide
}

func NewService(repo Repository) *Service {
	guides := DemoGuides()
	indexed := make(map[string]Guide, len(guides))
	for _, guide := range guides { indexed[guide.Code] = guide }
	return &Service{repo: repo, guides: indexed}
}
func (s *Service) Guide(role, code string) (Guide, error) {
	if code != "" {
		value, ok := s.guides[code]
		if !ok { return Guide{}, fmt.Errorf("guide not found") }
		return value, nil
	}
	for _, guide := range s.guides { if guide.Role == role { return guide, nil } }
	return Guide{}, fmt.Errorf("guide not found")
}
func (s *Service) State(ctx context.Context, tenant, principal, guideCode string) (State, error) {
	value, err := s.repo.Get(ctx, tenant, principal, guideCode)
	if err == ErrStateNotFound {
		guide, guideErr := s.Guide("", guideCode)
		if guideErr != nil { return State{}, guideErr }
		return State{TenantID: tenant, PrincipalID: principal, GuideCode: guideCode, GuideVersion: guide.Version, Version: 0}, nil
	}
	return value, err
}
func (s *Service) Update(ctx context.Context, tenant, principal, guideCode string, input UpdateInput) (State, error) {
	guide, err := s.Guide("", guideCode)
	if err != nil { return State{}, err }
	if input.CurrentStep < 0 || input.CurrentStep > len(guide.Steps) { return State{}, fmt.Errorf("current_step is outside the guide") }
	return s.repo.Upsert(ctx, State{TenantID: tenant, PrincipalID: principal, GuideCode: guideCode, GuideVersion: guide.Version, CurrentStep: input.CurrentStep, Completed: input.Completed, Dismissed: input.Dismissed}, input.ExpectedVersion)
}
func DemoGuides() []Guide {
	return []Guide{{Code: "control-assurance-first-run", Role: "Control Assurance Lead", Version: 1, Title: "Your continuous assurance workspace", Description: "Learn the three actions that keep Programs current without rebuilding registers.", Illustration: "guided-orbit", Steps: []Step{{ID: "today", Title: "Start with what changed", Description: "Today contains only material work requiring your judgment.", Action: "Review Today", Target: "today-brief"}, {ID: "routing", Title: "Inspect who is responsible", Description: "Every review and approval shows the policy and authority that selected the actor.", Action: "Inspect authority", Target: "authority-action"}, {ID: "capture", Title: "Request only missing proof", Description: "Focused capture uses existing evidence first and asks only unresolved facts.", Action: "Open capture wizard", Target: "capture-action"}, {ID: "readiness", Title: "Watch drift, not dashboards", Description: "Continuous readiness separates current, aging, blocked and human-judgment states.", Action: "View readiness", Target: "readiness-panel"}}}}
}
