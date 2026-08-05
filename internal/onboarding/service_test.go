package onboarding

import (
	"context"
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
}
