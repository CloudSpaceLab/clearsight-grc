package bankverticals

import (
	"context"
	"errors"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func (s *Service) InstallSample(ctx context.Context, config SeedConfig) ([]Journey, error) {
	if s == nil || s.continuity == nil || s.evidence == nil {
		return nil, fmt.Errorf("bank journeys are unavailable")
	}
	config = normalizeSeedConfig(config)
	if err := validateSeedConfig(config); err != nil {
		return nil, err
	}
	ctx = continuity.WithTrustedSystemEntityScope(ctx, config.TenantID, config.LegalEntityID)
	canonicalEntityID, err := s.continuity.ResolveLegalEntity(ctx, config.TenantID, config.LegalEntityID)
	if err != nil {
		return nil, fmt.Errorf("resolve reference-data legal entity: %w", err)
	}
	config.LegalEntityID = canonicalEntityID
	ctx = continuity.WithTrustedSystemEntityScope(ctx, config.TenantID, config.LegalEntityID)

	sourceIDs, err := s.seedSources(ctx, config)
	if err != nil {
		return nil, err
	}
	program, err := s.ensureNDPAProgram(ctx, config, sourceIDs)
	if err != nil {
		return nil, err
	}
	if err := s.ensureVendorDueDiligenceForm(ctx, config, program.Program.ID); err != nil {
		return nil, err
	}
	if err := s.ensureProgramEvidenceRequest(ctx, config, program.Program.ID); err != nil {
		return nil, err
	}
	if err := s.ensureRegulatoryChange(ctx, config, program, sourceIDs["NDPA-GAID-2025"]); err != nil {
		return nil, err
	}
	if err := s.ensureAuthorityRequest(ctx, config, program, sourceIDs["NDPC-REQUEST-2026"]); err != nil {
		return nil, err
	}
	if err := s.ensureLegacyFinding(ctx, config, program, sourceIDs["INTERNAL-AUDIT-2024"]); err != nil {
		return nil, err
	}
	return s.List(ctx, config.TenantID)
}

func (s *Service) ensureNDPAProgram(ctx context.Context, config SeedConfig, sourceIDs map[string]string) (continuity.ProgramAggregate, error) {
	program, err := s.continuity.ProgramByCode(ctx, config.TenantID, programCodeNDPA)
	if errors.Is(err, continuity.ErrNotFound) {
		return s.seedNDPAProgram(ctx, config, sourceIDs)
	}
	if err != nil {
		return continuity.ProgramAggregate{}, err
	}
	if scopeString(program.Program.Scope, "sample") != "true" || scopeString(program.Program.Scope, "journey_code") != string(JourneyNDPAContinuous) {
		return continuity.ProgramAggregate{}, fmt.Errorf("program code %s already exists without the ClearSight reference-data marker", programCodeNDPA)
	}
	if program.Program.Status == continuity.ProgramRetired {
		return continuity.ProgramAggregate{}, fmt.Errorf("reference program %s is retired and cannot be repaired", programCodeNDPA)
	}

	for _, spec := range referenceRequirementSpecs() {
		program, err = s.ensureRequirementBundle(ctx, config, program, sourceIDs, spec)
		if err != nil {
			return continuity.ProgramAggregate{}, err
		}
	}
	if program.Program.Status == continuity.ProgramDraft {
		program, err = s.continuity.TransitionProgram(ctx, continuity.ProgramTransitionInput{
			TenantID:        config.TenantID,
			ID:              program.Program.ID,
			ExpectedVersion: program.Program.Version,
			To:              continuity.ProgramActive,
			ActorID:         config.SignatoryPrincipalID,
			Rationale:       "The reconciled reference obligations, safeguards and evidence checks were reviewed and approved.",
		})
	}
	return program, err
}

func (s *Service) ensureProgramEvidenceRequest(ctx context.Context, config SeedConfig, programID string) error {
	_, err := s.evidence.LatestRequestForSubject(ctx, config.TenantID, "PROGRAM", programID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, evidence.ErrNotFound) {
		return err
	}
	_, err = s.seedNDPAEvidenceRequest(ctx, config, programID)
	return err
}
