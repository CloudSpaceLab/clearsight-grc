package onboarding

import (
	"context"
	"errors"
	"testing"
)

func TestOnboardingStateVersioning(t *testing.T) {
	service := NewService(NewMemoryRepository())
	initial, err := service.State(context.Background(), "bank-demo", "user-demo", "control-assurance-first-run")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(context.Background(), initial.TenantID, initial.PrincipalID, initial.GuideCode, UpdateInput{CurrentStep: 1, ExpectedVersion: initial.Version})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 1 || updated.CurrentStep != 1 {
		t.Fatalf("unexpected state: %#v", updated)
	}
	_, err = service.Update(context.Background(), initial.TenantID, initial.PrincipalID, initial.GuideCode, UpdateInput{CurrentStep: 2, ExpectedVersion: 0})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestGuideCannotBeCompletedAndDismissed(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.Update(context.Background(), "bank-demo", "user-demo", "control-assurance-first-run", UpdateInput{CurrentStep: 4, Completed: true, Dismissed: true, ExpectedVersion: 0})
	if err == nil {
		t.Fatal("expected invalid terminal state")
	}
}

func TestCompletedGuideMustReachFinalStep(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.Update(context.Background(), "bank-demo", "user-demo", "control-assurance-first-run", UpdateInput{CurrentStep: 2, Completed: true, ExpectedVersion: 0})
	if err == nil {
		t.Fatal("expected incomplete guide completion to fail")
	}
}
