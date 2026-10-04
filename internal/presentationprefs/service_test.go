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
