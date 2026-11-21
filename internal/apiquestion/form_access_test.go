package apiquestion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type formLibraryFixture struct {
	form monitoring.FormTemplate
	err  error
}

func (f formLibraryFixture) GetLibraryForm(context.Context, string, int64) (monitoring.FormTemplate, error) {
	return f.form, f.err
}

func TestAdoptionFormAccessRequiresExactScopedActiveForm(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	ctx := identity.WithActor(context.Background(), actor)
	form := monitoring.FormTemplate{ID: "form", LegalEntityID: "entity", Lifecycle: monitoring.Lifecycle{Version: 2, Status: monitoring.LifecycleActive, IsCurrent: true}}
	access := NewAdoptionFormAccess(formLibraryFixture{form: form})
	if err := access.AuthorizeAdoptionForm(ctx, "bank", "entity", "program", "form", 2); err != nil {
		t.Fatalf("valid form: %v", err)
	}
	for _, test := range []struct {
		name                    string
		ctx                     context.Context
		form                    monitoring.FormTemplate
		err                     error
		tenant, entity, program string
	}{
		{"anonymous", context.Background(), form, nil, "bank", "entity", "program"},
		{"tenant mismatch", ctx, form, nil, "other", "entity", "program"},
		{"entity mismatch", ctx, form, nil, "bank", "other", "program"},
		{"other form", ctx, func() monitoring.FormTemplate { f := form; f.ID = "other"; return f }(), nil, "bank", "entity", "program"},
		{"retired", ctx, func() monitoring.FormTemplate { f := form; f.Status = monitoring.LifecycleRetired; return f }(), nil, "bank", "entity", "program"},
		{"different program", ctx, func() monitoring.FormTemplate { f := form; f.ProgramID = "other"; return f }(), nil, "bank", "entity", "program"},
		{"lookup error", ctx, form, errors.New("unavailable"), "bank", "entity", "program"},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := NewAdoptionFormAccess(formLibraryFixture{form: test.form, err: test.err})
			if err := candidate.AuthorizeAdoptionForm(test.ctx, test.tenant, test.entity, test.program, "form", 2); err == nil {
				t.Fatal("expected denial")
			}
		})
	}
}
