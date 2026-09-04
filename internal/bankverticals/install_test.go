package bankverticals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestInstallSampleCreatesGovernedVendorFormsIdempotently(t *testing.T) {
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	repository := continuity.NewMemoryRepository()
	continuityService := continuity.NewServiceWithClock(repository, func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), evidenceService)
	service := NewService(continuityService, evidenceService)
	configureReferenceTimeline(service, repository)
	service.ConfigureMonitoring(monitoringService)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now

	if _, err := service.InstallSample(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	program, err := continuityService.ProgramByCode(
		continuity.WithTrustedSystemEntityScope(context.Background(), config.TenantID, config.LegalEntityID),
		config.TenantID,
		programCodeNDPA,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: program.Program.LegalEntityID, PrincipalID: config.OwnerPrincipalID}
	forms, err := monitoringService.ListReusableForms(context.Background(), actor, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(forms) != 4 {
		t.Fatalf("active reusable forms=%d, want 4: %#v", len(forms), forms)
	}
	want := map[string]int{"VENDOR-DUE-DILIGENCE": 8, "VENDOR-ADDRESS-VERIFICATION": 6, "VENDOR-CERTIFICATION-REFRESH": 7, "RESPONSE-POLICY-ACCEPTANCE": 4}
	for _, form := range forms {
		fieldCount, exists := want[form.Code]
		if !exists || form.Status != monitoring.LifecycleActive || !form.IsCurrent || len(form.Fields) != fieldCount {
			t.Fatalf("unexpected governed vendor form: %#v", form)
		}
		if form.CreatedBy != config.ActorID || form.SubmittedBy != config.ActorID || form.ApprovedBy != config.ReviewerPrincipalID {
			t.Fatalf("maker-checker history was not preserved: %#v", form.Lifecycle)
		}
		if form.Code == vendorCertificationRefreshFormCode && (form.ScoringMode != "COMPLIANCE" || form.ScoreProfile == nil || form.ScoreProfile.Version != "vendor-certification-v1") {
			t.Fatalf("seeded certification scoring was not preserved: %#v", form)
		}
		if form.Code == responsePolicyAcceptanceFormCode && (form.ScoringMode != "COMPLIANCE" || form.ScoreProfile == nil || form.ScoreProfile.Version != "response-policy-acceptance-v1") {
			t.Fatalf("seeded response-policy acceptance scoring was not preserved: %#v", form)
		}
		delete(want, form.Code)
	}
	if len(want) != 0 {
		t.Fatalf("missing governed forms: %#v", want)
	}

	revisions, err := monitoringService.ListForms(context.Background(), actor, program.Program.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 12 {
		t.Fatalf("initial form lifecycle revisions=%d, want 12", len(revisions))
	}
	if _, err := service.InstallSample(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	revisionsAfterRepeat, err := monitoringService.ListForms(context.Background(), actor, program.Program.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisionsAfterRepeat) != len(revisions) {
		t.Fatalf("repeat installation duplicated form revisions: before=%d after=%d", len(revisions), len(revisionsAfterRepeat))
	}
}

func TestInstallSampleRejectsSameVendorFormMakerAndChecker(t *testing.T) {
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	repository := continuity.NewMemoryRepository()
	continuityService := continuity.NewServiceWithClock(repository, func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), evidenceService)
	service := NewService(continuityService, evidenceService)
	configureReferenceTimeline(service, repository)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now
	sourceIDs, err := service.seedSources(continuity.WithTrustedSystemScope(context.Background()), config)
	if err != nil {
		t.Fatal(err)
	}
	program, err := service.ensureNDPAProgram(continuity.WithTrustedSystemScope(context.Background()), config, sourceIDs)
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureMonitoring(monitoringService)
	config.ReviewerPrincipalID = config.ActorID
	if err := service.ensureVendorForms(context.Background(), config, program.Program.ID); !errors.Is(err, ErrInvalidSeed) {
		t.Fatalf("same maker/checker form install err=%v", err)
	}
}
