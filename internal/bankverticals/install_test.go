package bankverticals

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestInstallSampleRecoversPartialProgram(t *testing.T) {
	ctx := context.Background()
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	service := NewService(continuityService, evidenceService)
	config := DemoSeedConfig()
	config.Now = time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	config = normalizeSeedConfig(config)

	sourceIDs, err := service.seedSources(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	program, err := continuityService.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID:             config.TenantID,
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
	ctx := context.Background()
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	service := NewService(continuityService, evidenceService)
	config := DemoSeedConfig()
	config.Now = time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
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
	if err != nil || partial.Matter.Status != continuity.MatterDraft {
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
