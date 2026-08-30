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
	continuityService := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), evidenceService)
	service := NewService(continuityService, evidenceService)
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
	if len(forms) != 3 {
		t.Fatalf("active reusable forms=%d, want 3: %#v", len(forms), forms)
	}
	want := map[string]int{"VENDOR-DUE-DILIGENCE": 8, "VENDOR-ADDRESS-VERIFICATION": 6, "VENDOR-CERTIFICATION-REFRESH": 5}
	for _, form := range forms {
		fieldCount, exists := want[form.Code]
		if !exists || form.Status != monitoring.LifecycleActive || !form.IsCurrent || len(form.Fields) != fieldCount {
			t.Fatalf("unexpected governed vendor form: %#v", form)
		}
		if form.CreatedBy != config.ActorID || form.SubmittedBy != config.ActorID || form.ApprovedBy != config.ReviewerPrincipalID {
			t.Fatalf("maker-checker history was not preserved: %#v", form.Lifecycle)
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
	if len(revisions) != 9 {
		t.Fatalf("initial form lifecycle revisions=%d, want 9", len(revisions))
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
	continuityService := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), evidenceService)
	service := NewService(continuityService, evidenceService)
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

	err = service.ensureVendorDueDiligenceForm(context.Background(), config, program.Program.ID)
	if !errors.Is(err, monitoring.ErrMakerChecker) {
		t.Fatalf("same maker/checker error=%v, want %v", err, monitoring.ErrMakerChecker)
	}
}

func TestInstallSampleRecoversPartialProgram(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	continuityService := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	service := NewService(continuityService, evidenceService)
	config := DemoSeedConfig()
	config.Now = now
	config = normalizeSeedConfig(config)

	sourceIDs, err := service.seedSources(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	program, err := continuityService.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID:             config.TenantID,
		LegalEntityID:        config.LegalEntityID,
		Code:                 programCodeNDPA,
		Name:                 "Nigeria data protection",
		Type:                 "PRIVACY",
		OwningFunction:       "Data Protection Office",
		OwnerPrincipalID:     config.OwnerPrincipalID,
		AuthorityPrincipalID: config.SignatoryPrincipalID,
		Jurisdiction:         "Nigeria",
		Scope:                mustJSON(map[string]any{"journey_code": JourneyNDPAContinuous, "sample": true}),
		EffectiveFrom:        config.Now.AddDate(0, -6, 0),
		ActorID:              config.ActorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.addRequirementBundle(ctx, config, program, sourceIDs, referenceRequirementSpecs()[0])
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.InstallSample(ctx, config); err != nil {
		t.Fatal(err)
	}
	program, err = continuityService.ProgramByCode(ctx, config.TenantID, programCodeNDPA)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Requirements) != 5 || len(program.EvidenceContracts) != 5 || program.Program.Status != continuity.ProgramActive {
		t.Fatalf("partial program was not reconciled: requirements=%d checks=%d status=%s", len(program.Requirements), len(program.EvidenceContracts), program.Program.Status)
	}
	matters, err := continuityService.ListMatters(ctx, config.TenantID, "", 20)
	if err != nil || len(matters) != 3 {
		t.Fatalf("remaining journeys were not installed: matters=%d err=%v", len(matters), err)
	}
}

func TestInstallSampleResumesPartiallyTransitionedMatter(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	continuityService := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	service := NewService(continuityService, evidenceService)
	config := DemoSeedConfig()
	config.Now = now
	config = normalizeSeedConfig(config)

	sourceIDs, err := service.seedSources(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	program, err := service.ensureNDPAProgram(ctx, config, sourceIDs)
	if err != nil {
		t.Fatal(err)
	}
	partial, err := continuityService.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID:          config.TenantID,
		Type:              continuity.MatterRegulatoryChange,
		Priority:          4,
		Title:             "Implement GAID 2025 annual return requirements",
		Summary:           "Partially installed reference issue.",
		Scope:             mustJSON(map[string]any{"journey_code": JourneyRegulatoryChange, "sample": true}),
		SourceType:        "REGULATORY",
		SourceID:          sourceIDs["NDPA-GAID-2025"],
		TriggerType:       "REQUIREMENT_CHANGED",
		TriggerKey:        triggerRegulatoryChange,
		KnownFacts:        mustJSON(map[string]any{"filing_deadline": "31 March"}),
		MissingFacts:      mustJSON([]string{}),
		Contradictions:    mustJSON([]string{}),
		OwnerPrincipalID:  config.OwnerPrincipalID,
		RequiredAuthority: "AUTHORIZER",
		DueAt:             timePointer(config.Now.Add(14 * 24 * time.Hour)),
		ProgramID:         program.Program.ID,
		ActorID:           config.ActorID,
	})
	if err != nil || partial.Matter.Status != continuity.MatterInitialReview {
		t.Fatalf("could not create partial issue: status=%s err=%v", partial.Matter.Status, err)
	}

	if _, err := service.InstallSample(ctx, config); err != nil {
		t.Fatal(err)
	}
	repaired, err := continuityService.MatterByTriggerKey(ctx, config.TenantID, triggerRegulatoryChange)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.Matter.Status != continuity.MatterActionsInProgress || !currentDecisionApproved(repaired.Decisions) || len(currentActions(repaired.Actions)) == 0 || activeVerificationContract(repaired, "") == nil {
		t.Fatalf("partial issue was not resumed: %#v", repaired)
	}
	before := len(repaired.Actions)
	if _, err := service.InstallSample(ctx, config); err != nil {
		t.Fatal(err)
	}
	repaired, err = continuityService.MatterByTriggerKey(ctx, config.TenantID, triggerRegulatoryChange)
	if err != nil || len(repaired.Actions) != before {
		t.Fatalf("repeat installation duplicated work: before=%d after=%d err=%v", before, len(repaired.Actions), err)
	}
}

func TestInstallSampleUsesEntityBoundedProgramLookup(t *testing.T) {
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	repo := continuity.NewMemoryRepository()
	continuityService := continuity.NewServiceWithClock(repo, func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	service := NewService(continuityService, evidenceService)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now
	other := continuity.WithTrustedSystemEntityScope(context.Background(), config.TenantID, "entity-other")
	foreign, err := continuityService.CreateProgram(other, continuity.CreateProgramInput{TenantID: config.TenantID, Code: programCodeNDPA, Name: "Other entity", Type: "PRIVACY", OwningFunction: "Privacy", Scope: mustJSON(map[string]any{"sample": true}), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.InstallSample(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	targetCtx := continuity.WithTrustedSystemEntityScope(context.Background(), config.TenantID, config.LegalEntityID)
	target, err := continuityService.ProgramByCode(targetCtx, config.TenantID, programCodeNDPA)
	if err != nil {
		t.Fatal(err)
	}
	if target.Program.ID == foreign.Program.ID || target.Program.LegalEntityID != config.LegalEntityID {
		t.Fatalf("installer selected foreign Program: %#v", target.Program)
	}
}

func TestInstallSampleCanonicalizesConfiguredEntityCodeInMemory(t *testing.T) {
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	repo := continuity.NewMemoryRepository()
	const targetID = "11111111-1111-4111-8111-111111111111"
	repo.RegisterLegalEntity("bank-demo", targetID, "bank-ng")
	repo.RegisterLegalEntity("bank-demo", "22222222-2222-4222-8222-222222222222", "other-ng")
	continuityService := continuity.NewServiceWithClock(repo, func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, targetID)
	service := NewService(continuityService, evidenceService)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now
	other := continuity.WithTrustedSystemEntityScope(context.Background(), config.TenantID, "other-ng")
	foreign, err := continuityService.CreateProgram(other, continuity.CreateProgramInput{TenantID: config.TenantID, Code: programCodeNDPA, Name: "Other", Type: "PRIVACY", OwningFunction: "Privacy", Scope: mustJSON(map[string]any{"sample": true}), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.InstallSample(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	target, err := continuityService.ProgramByCode(continuity.WithTrustedSystemEntityScope(context.Background(), config.TenantID, config.LegalEntityID), config.TenantID, programCodeNDPA)
	if err != nil {
		t.Fatal(err)
	}
	if target.Program.ID == foreign.Program.ID || target.Program.LegalEntityID != targetID {
		t.Fatalf("installer Program=%#v", target.Program)
	}
}
