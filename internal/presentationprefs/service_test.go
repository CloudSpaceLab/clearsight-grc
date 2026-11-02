package presentationprefs

import (
	"context"
	"testing"
)

func TestRoleDefaultsRemainPresentationOnlyAndOverridable(t *testing.T) {
	service := NewService(NewMemoryRepository())
	cro, err := service.Get(context.Background(), "tenant", "cro", []string{"CRO"})
	if err != nil {
		t.Fatal(err)
	}
	if cro.HomeFocus != HomeFocusAuto || cro.EffectiveHomeFocus != HomeFocusPosture || cro.EffectivePortfolioLens != PortfolioLensRisks || cro.Version != 0 {
		t.Fatalf("cro defaults=%#v", cro)
	}
	updated, err := service.Update(context.Background(), "tenant", "cro", []string{"CRO"}, UpdateInput{
		HomeFocus:       HomeFocusMyWork,
		PortfolioLens:   PortfolioLensPrograms,
		ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.EffectiveHomeFocus != HomeFocusMyWork || updated.EffectivePortfolioLens != PortfolioLensPrograms || updated.Version != 1 {
		t.Fatalf("updated=%#v", updated)
	}
}

func TestRoleDefaultsSeparateOversightFromAssignedWork(t *testing.T) {
	tests := []struct {
		name      string
		roles     []string
		home      HomeFocus
		portfolio PortfolioLens
	}{
		{name: "grc administrator", roles: []string{"GRC_ADMIN"}, home: HomeFocusPosture, portfolio: PortfolioLensRisks},
		{name: "risk manager", roles: []string{"RISK_MANAGER"}, home: HomeFocusPosture, portfolio: PortfolioLensRisks},
		{name: "risk owner", roles: []string{"RISK_OWNER"}, home: HomeFocusMyWork, portfolio: PortfolioLensRisks},
		{name: "control owner", roles: []string{"CONTROL_OWNER"}, home: HomeFocusMyWork, portfolio: PortfolioLensPrograms},
		{name: "program owner", roles: []string{"PROGRAM_OWNER"}, home: HomeFocusMyWork, portfolio: PortfolioLensPrograms},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home, portfolio := roleDefaults(tt.roles)
			if home != tt.home || portfolio != tt.portfolio {
				t.Fatalf("roles=%v home=%q portfolio=%q", tt.roles, home, portfolio)
			}
		})
	}
}

func TestPresentationPreferenceVersionAndEnumValidation(t *testing.T) {
	service := NewService(NewMemoryRepository())
	if _, err := service.Update(context.Background(), "tenant", "actor", nil, UpdateInput{
		HomeFocus:       "QUERY",
		PortfolioLens:   PortfolioLensAuto,
		ExpectedVersion: 0,
	}); err != ErrInvalid {
		t.Fatalf("invalid enum error=%v", err)
	}
	if _, err := service.Update(context.Background(), "tenant", "actor", nil, UpdateInput{
		HomeFocus:       HomeFocusAuto,
		PortfolioLens:   PortfolioLensAuto,
		ExpectedVersion: 0,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(context.Background(), "tenant", "actor", nil, UpdateInput{
		HomeFocus:       HomeFocusPosture,
		PortfolioLens:   PortfolioLensRisks,
		ExpectedVersion: 0,
	}); err != ErrVersionConflict {
		t.Fatalf("stale update error=%v", err)
	}
}
