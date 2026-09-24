//go:build !postgres

package main

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type apiReferenceEvidenceRepository struct {
	*evidence.MemoryRepository
}

func TestMemoryTodayIsDerivedFromInstalledMatterAssignments(t *testing.T) {
	services, err := buildServices(t.Context(), config.Config{
		Environment: "development", DemoMode: true, DemoTenantID: "bank-demo", DemoLegalEntityID: "bank-ng",
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	items, err := services.Today.ListFor(t.Context(), identity.Actor{TenantID: "bank-demo", LegalEntityID: "bank-ng", PrincipalID: "owner-demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("installed Matter assignments did not produce Today work")
	}
	for _, item := range items {
		if item.ID == "workflow_task_review_cbn" || item.ID == "workflow_task_access_evidence" || item.ActionTargetType != "MATTER" || item.ActionTargetID == "" {
			t.Fatalf("Today item was not derived from an installed Matter: %#v", item)
		}
	}
}

func TestMemoryCompositionInstallsRopaDemoOnlyInDemoMode(t *testing.T) {
	tests := []struct {
		name     string
		demoMode bool
		wantRows int
	}{
		{name: "demo mode", demoMode: true, wantRows: 3},
		{name: "non-demo mode", demoMode: false, wantRows: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			services, err := buildServices(t.Context(), config.Config{
				Environment:       "development",
				DemoMode:          test.demoMode,
				DemoTenantID:      ropa.DemoTenant,
				DemoLegalEntityID: ropa.DemoLegalEntity,
			}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err != nil {
				t.Fatal(err)
			}
			defer services.Close()

			page, err := services.Ropa.ListActivities(t.Context(), ropa.ActivityScope{
				TenantID:      ropa.DemoTenant,
				LegalEntityID: ropa.DemoLegalEntity,
			}, ropa.ListActivitiesFilter{Limit: 50})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(page.Rows); got != test.wantRows {
				t.Fatalf("ROPA rows = %d, want %d", got, test.wantRows)
			}
		})
	}
}

func TestMemoryServicesExposeOnlyVerifiedRuntimeIdentifiers(t *testing.T) {
	services, err := buildServices(t.Context(), config.Config{Environment: "development"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	if services.RuntimeContext == nil {
		t.Fatal("runtime context resolver is absent")
	}
	value, err := services.RuntimeContext.Resolve(t.Context(), runtimecontext.Scope{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a"})
	if err != nil {
		t.Fatal(err)
	}
	if value.TenantName != "tenant-a" || value.LegalEntityName != "entity-a" || value.PrincipalName != "principal-a" {
		t.Fatalf("memory runtime context = %#v", value)
	}
}

func (r *apiReferenceEvidenceRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	if tenant != "bank-demo" || (subjectType != "PROGRAM" && subjectType != "MATTER") || strings.TrimSpace(subjectID) == "" {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	return evidence.SubjectScope{TenantID: tenant, LegalEntityID: "bank-ng", SubjectType: subjectType, SubjectID: subjectID}, nil
}

func TestConfigureReferenceVerticalsInstallsActiveVendorForms(t *testing.T) {
	repository := continuity.NewMemoryRepository()
	continuityService := continuity.NewService(repository)
	evidenceService := evidence.NewService(&apiReferenceEvidenceRepository{MemoryRepository: evidence.NewMemoryRepository(nil, nil)}, evidence.NewMemoryObjectStore())
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), evidenceService)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	configureReferenceVerticals(verticals, monitoringService)
	verticals.ConfigureReferenceTimeline(func(at time.Time) *continuity.Service {
		return continuity.NewServiceWithClock(repository, func() time.Time { return at })
	})

	if _, err := verticals.InstallSample(context.Background(), bankverticals.DemoSeedConfig()); err != nil {
		t.Fatal(err)
	}
	forms, err := monitoringService.ListReusableForms(context.Background(), monitoring.Actor{
		TenantID: "bank-demo", LegalEntityID: "bank-ng", PrincipalID: "owner-demo",
	}, 100)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"VENDOR-DUE-DILIGENCE": true, "VENDOR-ADDRESS-VERIFICATION": true, "VENDOR-CERTIFICATION-REFRESH": true, "THIRD-PARTY-RISK-COMPLIANCE": true}
	if len(forms) != len(want) {
		t.Fatalf("demo vendor forms=%#v", forms)
	}
	for _, form := range forms {
		if !want[form.Code] || form.Status != monitoring.LifecycleActive || !form.IsCurrent {
			t.Fatalf("unexpected demo vendor form=%#v", form)
		}
		delete(want, form.Code)
	}
	if len(want) != 0 {
		t.Fatalf("missing demo vendor forms=%#v", want)
	}
}
