//go:build !postgres

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type apiReferenceEvidenceRepository struct {
	*evidence.MemoryRepository
}

func (r *apiReferenceEvidenceRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	if tenant != "bank-demo" || (subjectType != "PROGRAM" && subjectType != "MATTER") || strings.TrimSpace(subjectID) == "" {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	return evidence.SubjectScope{TenantID: tenant, LegalEntityID: "bank-ng", SubjectType: subjectType, SubjectID: subjectID}, nil
}

func TestConfigureReferenceVerticalsInstallsActiveVendorDueDiligenceForm(t *testing.T) {
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	evidenceService := evidence.NewService(&apiReferenceEvidenceRepository{MemoryRepository: evidence.NewMemoryRepository(nil, nil)}, evidence.NewMemoryObjectStore())
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), evidenceService)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	configureReferenceVerticals(verticals, monitoringService)

	if _, err := verticals.InstallSample(context.Background(), bankverticals.DemoSeedConfig()); err != nil {
		t.Fatal(err)
	}
	forms, err := monitoringService.ListReusableForms(context.Background(), monitoring.Actor{
		TenantID: "bank-demo", LegalEntityID: "bank-ng", PrincipalID: "owner-demo",
	}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(forms) != 1 || forms[0].Code != "VENDOR-DUE-DILIGENCE" || forms[0].Status != monitoring.LifecycleActive || !forms[0].IsCurrent {
		t.Fatalf("demo vendor due-diligence form=%#v", forms)
	}
}
