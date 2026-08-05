package bankverticals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

// InstallSample is the recoverable, explicit reference-data installation path.
// It reconciles every stable reference object independently so an interrupted
// installation can be safely re-run without duplicating completed stages.
func (s *Service) InstallSample(ctx context.Context, config SeedConfig) ([]Journey, error) {
	if s == nil || s.continuity == nil || s.evidence == nil {
		return nil, fmt.Errorf("bank journeys are unavailable")
	}
	config = normalizeSeedConfig(config)
	if err := validateSeedConfig(config); err != nil {
		return nil, err
	}

	sourceIDs, err := s.seedSources(ctx, config)
	if err != nil {
		return nil, err
	}
	program, err := s.ensureNDPAProgram(ctx, config, sourceIDs)
	if err != nil {
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

func (s *Service) ensureRequirementBundle(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceIDs map[string]string, spec requirementSpec) (continuity.ProgramAggregate, error) {
	var requirement *continuity.Requirement
	for index := range program.Requirements {
		if strings.EqualFold(program.Requirements[index].Code, spec.code) && program.Requirements[index].Status == continuity.RequirementApproved {
			value := program.Requirements[index]
			requirement = &value
			break
		}
	}
	if requirement == nil {
		return s.addRequirementBundle(ctx, config, program, sourceIDs, spec)
	}

	applicable := false
	for _, value := range program.Applicability {
		if value.RequirementID == requirement.ID && value.Status == continuity.ApplicabilityApplicable {
			applicable = true
			break
		}
	}
	var err error
	if !applicable {
		program, err = s.continuity.DetermineApplicability(ctx, continuity.DetermineApplicabilityInput{
			TenantID:        config.TenantID,
			ProgramID:       program.Program.ID,
			ExpectedVersion: program.Program.Version,
			RequirementID:   requirement.ID,
			Status:          continuity.ApplicabilityApplicable,
			Scope:           mustJSON(map[string]any{"bank": config.BankName, "legal_entity_id": config.LegalEntityID}),
			Rationale:       "The bank processes customer, employee, vendor and other personal data in Nigeria.",
			ApprovedBy:      config.ReviewerPrincipalID,
			EffectiveFrom:   config.Now.AddDate(0, -6, 0),
		})
		if err != nil {
			return program, fmt.Errorf("repair applicability %s: %w", spec.code, err)
		}
	}

	var objective *continuity.ControlObjective
	for index := range program.ControlObjectives {
		if strings.EqualFold(program.ControlObjectives[index].Code, spec.objectiveCode) && program.ControlObjectives[index].Status == continuity.ObjectiveActive {
			value := program.ControlObjectives[index]
			objective = &value
			break
		}
	}
	if objective == nil {
		program, err = s.continuity.AddControlObjective(ctx, continuity.AddControlObjectiveInput{
			TenantID:        config.TenantID,
			ProgramID:       program.Program.ID,
			ExpectedVersion: program.Program.Version,
			Code:            spec.objectiveCode,
			Name:            spec.objectiveName,
			Outcome:         spec.outcome,
			Status:          continuity.ObjectiveActive,
			ActorID:         config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair safeguard objective %s: %w", spec.code, err)
		}
		value := program.ControlObjectives[len(program.ControlObjectives)-1]
		objective = &value
	}

	var implementation *continuity.ControlImplementation
	for index := range program.ControlImplementations {
		value := program.ControlImplementations[index]
		if value.ObjectiveID == objective.ID && strings.EqualFold(value.Name, spec.implementationName) && value.Status == continuity.ImplementationImplemented {
			copy := value
			implementation = &copy
			break
		}
	}
	if implementation == nil {
		program, err = s.continuity.AddControlImplementation(ctx, continuity.AddControlImplementationInput{
			TenantID:           config.TenantID,
			ProgramID:          program.Program.ID,
			ExpectedVersion:    program.Program.Version,
			ObjectiveID:        objective.ID,
			Name:               spec.implementationName,
			Description:        spec.implementationDetail,
			ImplementationType: spec.implementationType,
			OwnerPrincipalID:   config.OwnerPrincipalID,
			Scope:              mustJSON(map[string]any{"bank": config.BankName}),
			Status:             continuity.ImplementationImplemented,
			EffectiveFrom:      config.Now.AddDate(0, -3, 0),
			ActorID:            config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair safeguard %s: %w", spec.code, err)
		}
		value := program.ControlImplementations[len(program.ControlImplementations)-1]
		implementation = &value
	}

	linked := false
	for _, value := range program.RequirementControlLinks {
		if value.RequirementID == requirement.ID && value.ImplementationID == implementation.ID {
			linked = true
			break
		}
	}
	if !linked {
		program, err = s.continuity.LinkRequirementControl(ctx, continuity.LinkRequirementControlInput{
			TenantID:        config.TenantID,
			ProgramID:       program.Program.ID,
			ExpectedVersion: program.Program.Version,
			RequirementID:   requirement.ID,
			ImplementationID: implementation.ID,
			ActorID:         config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair safeguard link %s: %w", spec.code, err)
		}
	}

	var contract *continuity.EvidenceContract
	for index := range program.EvidenceContracts {
		if strings.EqualFold(program.EvidenceContracts[index].Code, spec.evidenceCode) && program.EvidenceContracts[index].Status == continuity.EvidenceContractActive {
			value := program.EvidenceContracts[index]
			contract = &value
			break
		}
	}
	if contract == nil {
		acceptable := make([]string, 0, len(spec.sourceCodes))
		for _, code := range spec.sourceCodes {
			acceptable = append(acceptable, sourceIDs[code])
		}
		program, err = s.continuity.AddEvidenceContract(ctx, continuity.AddEvidenceContractInput{
			TenantID:               config.TenantID,
			ProgramID:              program.Program.ID,
			ExpectedVersion:        program.Program.Version,
			ControlImplementationID: implementation.ID,
			Code:                   spec.evidenceCode,
			Name:                   spec.evidenceName,
			Claim:                  spec.claim,
			AcceptableSourceIDs:    acceptable,
			PopulationScope:        mustJSON(spec.population),
			FreshnessMinutes:       spec.freshnessMinutes,
			MinimumCoverage:        spec.minimumCoverage,
			IndependenceRequired:   true,
			ContradictionPolicy:    "REVIEW",
			FailureAction:          "MATTER",
			Status:                 continuity.EvidenceContractActive,
			ActorID:                config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair evidence check %s: %w", spec.code, err)
		}
		value := program.EvidenceContracts[len(program.EvidenceContracts)-1]
		contract = &value
	}

	current := false
	for _, assessment := range program.EvidenceAssessments {
		if assessment.ContractID == contract.ID && (assessment.ValidUntil == nil || assessment.ValidUntil.After(config.Now)) {
			current = true
			break
		}
	}
	if !current {
		validUntil := config.Now.Add(30 * 24 * time.Hour)
		program, err = s.continuity.RecordEvidenceAssessment(ctx, continuity.RecordEvidenceAssessmentInput{
			TenantID:        config.TenantID,
			ProgramID:       program.Program.ID,
			ExpectedVersion: program.Program.Version,
			ContractID:      contract.ID,
			Conclusion:      spec.conclusion,
			Coverage:        spec.coverage,
			Basis:           mustJSON(spec.basis),
			ValidUntil:      &validUntil,
			AssessedBy:      config.ReviewerPrincipalID,
			AssessedAt:      config.Now.Add(-2 * time.Hour),
		})
		if err != nil {
			return program, fmt.Errorf("repair evidence assessment %s: %w", spec.code, err)
		}
	}
	return program, nil
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

func (s *Service) ensureRegulatoryChange(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) error {
	matter, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerRegulatoryChange)
	if errors.Is(err, continuity.ErrNotFound) {
		_, err = s.seedRegulatoryChange(ctx, config, program, sourceID)
		return err
	}
	if err != nil {
		return err
	}
	if !referenceMatter(matter.Matter, JourneyRegulatoryChange) {
		return fmt.Errorf("trigger key %s is already used by a non-reference issue", triggerRegulatoryChange)
	}
	if matter.Matter.Status == continuity.MatterClosed || matter.Matter.Status == continuity.MatterCancelled {
		return fmt.Errorf("regulatory-change reference issue cannot be repaired from %s", matter.Matter.Status)
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterAssessment, "The official source and affected processes were reconciled.")
	if err != nil {
		return err
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterDecisionRequired, "The reconciled change requires an approved bank position.")
	if err != nil {
		return err
	}
	if !currentDecisionApproved(matter.Decisions) {
		matter, err = s.continuity.AddDecision(ctx, continuity.AddDecisionInput{
			TenantID:             config.TenantID,
			MatterID:             matter.Matter.ID,
			ExpectedVersion:      matter.Matter.Version,
			Type:                 "IMPLEMENTATION_APPROACH",
			Status:               continuity.DecisionApproved,
			Options:              mustJSON([]string{"UPDATE_CURRENT_PROCESS", "CREATE_SEPARATE_GAID_PROCESS", "NO_CHANGE_REQUIRED"}),
			SelectedOption:       "UPDATE_CURRENT_PROCESS",
			Rationale:            "The existing annual return process will use source-linked evidence owners and an earlier internal review date.",
			Conditions:           mustJSON([]string{"DPO confirms the final evidence checklist", "DPCO review starts at least 30 days before filing"}),
			AuthorityPrincipalID: config.SignatoryPrincipalID,
		})
		if err != nil {
			return err
		}
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterActionsInProgress, "The approved process changes are being implemented.")
	if err != nil {
		return err
	}
	actions := currentActions(matter.Actions)
	if len(actions) == 0 {
		matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{
			TenantID:        config.TenantID,
			MatterID:        matter.Matter.ID,
			ExpectedVersion: matter.Matter.Version,
			Title:           "Update the annual return evidence checklist",
			Description:     "Assign each evidence section, record its authoritative source and move the internal approval date to 1 March.",
			OwnerPrincipalID: config.OwnerPrincipalID,
			DueAt:           timePointer(config.Now.Add(10 * 24 * time.Hour)),
			ActorID:         config.ActorID,
		})
		if err != nil {
			return err
		}
		actions = currentActions(matter.Actions)
	}
	action := actions[len(actions)-1]
	if action.Status == continuity.ActionPlanned {
		matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionInProgress, ActorID: config.OwnerPrincipalID})
		if err != nil {
			return err
		}
	}
	if !hasActiveVerificationContract(matter, action.ID) {
		_, err = s.continuity.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
			TenantID:                  config.TenantID,
			MatterID:                 matter.Matter.ID,
			ExpectedVersion:          matter.Matter.Version,
			ActionID:                 action.ID,
			ExpectedOutcome:          "Every required annual return evidence section has an owner, authoritative source, internal approval date and DPCO review status.",
			Baseline:                 mustJSON(map[string]any{"complete_sections": 8, "required_sections": 10}),
			Scope:                    mustJSON(map[string]any{"journey_code": JourneyRegulatoryChange, "filing_year": 2027}),
			Threshold:                mustJSON(map[string]any{"complete_sections": 10, "approved": true}),
			ObservationPeriodMinutes: 0,
			AuthorityPrincipalID:      config.ReviewerPrincipalID,
			FailureResponse:           "BLOCK_CLOSE",
			ActorID:                   config.ActorID,
		})
	}
	return err
}

func (s *Service) ensureAuthorityRequest(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) error {
	matter, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerAuthorityRequest)
	if errors.Is(err, continuity.ErrNotFound) {
		_, err = s.seedAuthorityRequest(ctx, config, program, sourceID)
		return err
	}
	if err != nil {
		return err
	}
	if !referenceMatter(matter.Matter, JourneyAuthorityRequest) || !restrictedPolicyComplete(matter.Matter) {
		return fmt.Errorf("authority-request reference issue has invalid provenance or restricted-access metadata")
	}
	if matter.Matter.Status == continuity.MatterCancelled {
		return fmt.Errorf("authority-request reference issue is cancelled and cannot be repaired")
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterAssessment, "The authority request and restricted handling group were reconciled.")
	if err != nil {
		return err
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterResponsePreparation, "The response package and signatory approval are being prepared.")
	if err != nil {
		return err
	}
	request, requestErr := s.evidence.LatestRequestForSubject(ctx, config.TenantID, "MATTER", matter.Matter.ID)
	if errors.Is(requestErr, evidence.ErrNotFound) {
		request, requestErr = s.evidence.CreateRequest(ctx, authorityEvidenceRequest(config, matter.Matter.ID))
	}
	if requestErr != nil {
		return requestErr
	}
	if requestIsActionable(request) {
		_, err = s.evidence.Submit(ctx, evidence.Submission{
			TenantID:         config.TenantID,
			RequestID:        request.ID,
			SubmittedBy:      config.OwnerPrincipalID,
			Channel:          "INTERNAL",
			Answers:          map[string]string{"containment_record": "Incident containment record PRI-2026-008", "communication_decision": "No direct customer notice was approved after the documented impact assessment."},
			ExpectedVersion: request.Version,
		})
		if err != nil {
			return err
		}
	}
	response := currentResponse(matter.ResponsePackages)
	if response == nil || response.Status == continuity.ResponseRejected || response.Status == continuity.ResponseWithdrawn {
		matter, err = s.continuity.AddResponsePackage(ctx, continuity.AddResponsePackageInput{
			TenantID:        config.TenantID,
			MatterID:        matter.Matter.ID,
			ExpectedVersion: matter.Matter.Version,
			Purpose:         "Respond to NDPC request NDPC/ENF/2026/0142",
			Audience:        "Nigeria Data Protection Commission",
			Manifest:        mustJSON([]map[string]any{{"classification": "RESTRICTED", "evidence_request_id": request.ID}, {"document": "incident assessment"}, {"document": "containment record"}, {"document": "notification decision"}, {"document": "customer communication decision"}}),
			ActorID:         config.ActorID,
		})
		if err != nil {
			return err
		}
		response = currentResponse(matter.ResponsePackages)
	}
	for _, target := range []continuity.ResponseStatus{continuity.ResponseInReview, continuity.ResponseApproved, continuity.ResponseTransmitted, continuity.ResponseAcknowledged} {
		if responseRank(response.Status) >= responseRank(target) {
			continue
		}
		actorID := config.ActorID
		if target == continuity.ResponseApproved {
			actorID = config.SignatoryPrincipalID
		}
		matter, err = s.continuity.TransitionResponsePackage(ctx, continuity.TransitionResponseInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ResponseID: response.ID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: actorID, Rationale: responseRationale(target)})
		if err != nil {
			return err
		}
		response = currentResponse(matter.ResponsePackages)
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterVerification, "The response was transmitted and acknowledgement was recorded.")
	if err != nil {
		return err
	}
	_, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterClosed, "The authority acknowledged receipt of the approved response package.")
	return err
}

func (s *Service) ensureLegacyFinding(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) error {
	matter, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerFindingRemediation)
	if errors.Is(err, continuity.ErrNotFound) {
		_, err = s.seedLegacyFinding(ctx, config, program, sourceID)
		return err
	}
	if err != nil {
		return err
	}
	if !referenceMatter(matter.Matter, JourneyFindingRemediation) {
		return fmt.Errorf("finding reference trigger is already used by a non-reference issue")
	}
	if matter.Matter.Status == continuity.MatterCancelled {
		return fmt.Errorf("finding reference issue is cancelled and cannot be repaired")
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterAssessment, "The legacy finding and affected scope were reconciled.")
	if err != nil {
		return err
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterActionsInProgress, "The remediation was assigned to the processing-record owners.")
	if err != nil {
		return err
	}
	actions := currentActions(matter.Actions)
	if len(actions) == 0 {
		matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{
			TenantID:         config.TenantID,
			MatterID:         matter.Matter.ID,
			ExpectedVersion:  matter.Matter.Version,
			Title:            "Approve retention periods for the 14 affected records",
			Description:      "Record the approved retention period, approving owner and policy reference for each affected processing activity.",
			OwnerPrincipalID: config.OwnerPrincipalID,
			DueAt:            timePointer(config.Now.Add(-10 * 24 * time.Hour)),
			ActorID:          config.ActorID,
		})
		if err != nil {
			return err
		}
		actions = currentActions(matter.Actions)
	}
	action := actions[len(actions)-1]
	for _, target := range []continuity.ActionStatus{continuity.ActionInProgress, continuity.ActionImplemented} {
		if actionRank(action.Status) >= actionRank(target) {
			continue
		}
		matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: config.OwnerPrincipalID})
		if err != nil {
			return err
		}
		actions = currentActions(matter.Actions)
		action = actions[len(actions)-1]
	}
	contract := activeVerificationContract(matter, action.ID)
	if contract == nil {
		matter, err = s.continuity.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
			TenantID:                  config.TenantID,
			MatterID:                 matter.Matter.ID,
			ExpectedVersion:          matter.Matter.Version,
			ActionID:                 action.ID,
			ExpectedOutcome:          "All 14 affected processing records contain an approved retention period, owner and policy reference.",
			Baseline:                 mustJSON(map[string]any{"complete_records": 0, "affected_records": 14}),
			Scope:                    mustJSON(map[string]any{"finding_reference": "IA-PRIV-2024-07"}),
			MeasurementSourceID:      sourceID,
			Threshold:                mustJSON(map[string]any{"complete_records": 14, "exceptions": 0}),
			ObservationPeriodMinutes: 0,
			AuthorityPrincipalID:      config.ReviewerPrincipalID,
			FailureResponse:           "REOPEN",
			ActorID:                   config.ActorID,
		})
		if err != nil {
			return err
		}
		contract = activeVerificationContract(matter, action.ID)
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterVerification, "Implementation is complete and independent checking has started.")
	if err != nil {
		return err
	}
	if !contractHasIndependentPass(matter, *contract) {
		matter, err = s.continuity.RecordVerificationResult(ctx, continuity.RecordVerificationResultInput{
			TenantID:            config.TenantID,
			MatterID:           matter.Matter.ID,
			ExpectedVersion:    matter.Matter.Version,
			ContractID:         contract.ID,
			Result:             continuity.VerificationPassed,
			Observations:       mustJSON(map[string]any{"complete_records": 14, "exceptions": 0}),
			EvidenceReferences: mustJSON([]string{"processing inventory export", "retention owner approvals", "sample re-performance"}),
			ReviewerPrincipalID: config.ReviewerPrincipalID,
			Rationale:           "The reviewer re-performed the check and confirmed all 14 records contain the required approved retention information.",
			ObservedAt:          config.Now.Add(-24 * time.Hour),
		})
		if err != nil {
			return err
		}
	}
	_, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterClosed, "The remediation was implemented and the independent outcome check passed.")
	return err
}

func referenceRequirementSpecs() []requirementSpec {
	return []requirementSpec{
		{code: "PROCESSING-ACCOUNTABILITY", title: "Keep personal-data processing accountable", statement: "The bank must process personal data fairly, lawfully and accountably for defined purposes.", anchor: "NDP Act 2023, section 24", objectiveCode: "PROCESSING-RECORDS", objectiveName: "Current processing records", outcome: "Every active processing activity has a current purpose, lawful basis, owner, data set, recipient and retention record.", implementationName: "Quarterly processing-owner review", implementationType: "OWNER_REVIEW", implementationDetail: "Processing owners confirm new, changed and retired processing activities each quarter.", evidenceCode: "PROCESSING-COVERAGE", evidenceName: "Processing inventory coverage", claim: "Every active processing activity has a current owner-approved record.", sourceCodes: []string{"PRIVACY-PROCESSING-INVENTORY"}, population: map[string]any{"population": "active_processing_activities", "expected": 126}, freshnessMinutes: 43200, minimumCoverage: .95, coverage: .92, conclusion: continuity.EvidencePartiallySupported, basis: map[string]any{"current_records": 116, "expected_records": 126}},
		{code: "DATA-SUBJECT-RIGHTS", title: "Respond to data rights requests", statement: "The bank must receive, assess and complete valid data-subject requests through a controlled process.", anchor: "NDP Act 2023, data-subject rights provisions", objectiveCode: "RIGHTS-TIMELINESS", objectiveName: "Timely rights handling", outcome: "Each request has identity checks, scope, response decision, completion date and supporting evidence.", implementationName: "Data rights case workflow", implementationType: "CASE_MANAGEMENT", implementationDetail: "The Privacy Office monitors open requests, ageing and approved extensions each business day.", evidenceCode: "RIGHTS-REGISTER", evidenceName: "Rights request completion", claim: "Every closed rights request has an approved response and completion evidence.", sourceCodes: []string{"DATA-RIGHTS-REGISTER"}, population: map[string]any{"population": "closed_rights_requests", "period": "rolling_90_days"}, freshnessMinutes: 1440, minimumCoverage: 1, coverage: 1, conclusion: continuity.EvidenceSupported, basis: map[string]any{"closed_requests": 38, "complete_records": 38}},
		{code: "PRIVACY-INCIDENTS", title: "Assess personal-data incidents", statement: "The bank must identify, assess, contain and document personal-data incidents and make required notifications.", anchor: "NDP Act 2023, personal-data breach provisions", objectiveCode: "INCIDENT-ASSESSMENT", objectiveName: "Complete privacy incident assessment", outcome: "Every suspected personal-data incident has severity, affected data, containment, notification decision and closure evidence.", implementationName: "Privacy incident assessment", implementationType: "INCIDENT_RESPONSE", implementationDetail: "Cybersecurity and the Privacy Office jointly assess incidents with possible personal-data impact.", evidenceCode: "INCIDENT-COVERAGE", evidenceName: "Privacy incident record coverage", claim: "Every security incident tagged for privacy review has a completed privacy assessment.", sourceCodes: []string{"PRIVACY-INCIDENT-REGISTER"}, population: map[string]any{"population": "privacy_review_incidents", "period": "rolling_12_months"}, freshnessMinutes: 720, minimumCoverage: 1, coverage: 1, conclusion: continuity.EvidenceSupported, basis: map[string]any{"privacy_review_incidents": 12, "completed_assessments": 12}},
		{code: "DPIA-HIGH-RISK", title: "Review high-risk processing before release", statement: "The bank must assess high-risk processing and record privacy safeguards before the change goes live.", anchor: "NDP Act 2023 and GAID 2025 DPIA requirements", objectiveCode: "DPIA-GATE", objectiveName: "Privacy review before release", outcome: "High-risk product and technology changes have an approved privacy impact assessment before production release.", implementationName: "High-risk change privacy gate", implementationType: "CHANGE_GATE", implementationDetail: "The change portfolio routes high-risk processing to the Privacy Office before release approval.", evidenceCode: "DPIA-COVERAGE", evidenceName: "High-risk change review coverage", claim: "Every high-risk change has an approved privacy impact assessment before production release.", sourceCodes: []string{"CHANGE-PORTFOLIO", "PRIVACY-PROCESSING-INVENTORY"}, population: map[string]any{"population": "high_risk_changes", "period": "current_quarter"}, freshnessMinutes: 1440, minimumCoverage: 1, coverage: .75, conclusion: continuity.EvidencePartiallySupported, basis: map[string]any{"high_risk_changes": 12, "approved_assessments": 9, "awaiting_evidence": 3}},
		{code: "CAR-ANNUAL", title: "Prepare the annual compliance audit return", statement: "The bank must maintain the records and independent review needed for its annual Compliance Audit Return.", anchor: "GAID 2025, Articles 10.7 and 10.8; filing before 31 March", objectiveCode: "CAR-READINESS", objectiveName: "Annual audit return ready for filing", outcome: "The DPCO receives a complete, approved and traceable evidence pack before the filing deadline.", implementationName: "Annual CAR readiness review", implementationType: "ANNUAL_CERTIFICATION", implementationDetail: "The Privacy Office and licensed DPCO review the evidence pack, unresolved findings and management approval before filing.", evidenceCode: "CAR-EVIDENCE", evidenceName: "Annual return evidence pack", claim: "The annual audit return evidence pack is complete, reviewed and approved before filing.", sourceCodes: []string{"NDPA-GAID-2025", "PRIVACY-PROCESSING-INVENTORY", "DATA-RIGHTS-REGISTER", "PRIVACY-INCIDENT-REGISTER"}, population: map[string]any{"population": "car_evidence_sections", "filing_deadline": "31 March"}, freshnessMinutes: 43200, minimumCoverage: 1, coverage: .8, conclusion: continuity.EvidencePartiallySupported, basis: map[string]any{"complete_sections": 8, "required_sections": 10}},
	}
}

func authorityEvidenceRequest(config SeedConfig, matterID string) evidence.CreateRequestInput {
	return evidence.CreateRequestInput{
		TenantID:         config.TenantID,
		SubjectType:      "MATTER",
		SubjectID:        matterID,
		Title:            "Provide incident containment and customer communication records",
		Purpose:          "Complete the restricted NDPC response package.",
		WhyYou:           "You own the incident response records requested by Regulatory Affairs.",
		Sensitivity:      "RESTRICTED",
		AudienceType:     "INTERNAL",
		EstimatedMinutes: 10,
		Deadline:         config.Now.Add(48 * time.Hour),
		KnownFacts:       map[string]string{"case_reference": "NDPC/ENF/2026/0142", "incident_reference": "PRI-2026-008"},
		Fields: []evidence.Field{
			{ID: "containment_record", Label: "Containment record", Type: "FILE", Required: true, AcceptedFormats: []string{"application/pdf", "text/csv"}},
			{ID: "communication_decision", Label: "Customer communication decision", Type: "TEXT", Required: true, Description: "State the approved decision and approving authority."},
		},
		CreatedBy: config.ActorID,
	}
}

func referenceMatter(matter continuity.Matter, code Code) bool {
	return scopeString(matter.Scope, "sample") == "true" && scopeString(matter.Scope, "journey_code") == string(code)
}

func advanceMatter(ctx context.Context, service *continuity.Service, config SeedConfig, matter continuity.MatterAggregate, target continuity.MatterStatus, rationale string) (continuity.MatterAggregate, error) {
	if statusAtLeast(matter.Matter.Status, target) {
		return matter, nil
	}
	return service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: config.ActorID, Rationale: rationale})
}

func hasActiveVerificationContract(matter continuity.MatterAggregate, actionID string) bool {
	return activeVerificationContract(matter, actionID) != nil
}

func activeVerificationContract(matter continuity.MatterAggregate, actionID string) *continuity.VerificationContract {
	for index := range matter.VerificationContracts {
		value := matter.VerificationContracts[index]
		if value.Status == continuity.VerificationActive && (actionID == "" || value.ActionID == actionID) {
			copy := value
			return &copy
		}
	}
	return nil
}

func contractHasIndependentPass(matter continuity.MatterAggregate, contract continuity.VerificationContract) bool {
	for _, result := range matter.VerificationResults {
		if result.ContractID == contract.ID && result.Result == continuity.VerificationPassed && result.ReviewerPrincipalID == contract.AuthorityPrincipalID {
			return true
		}
	}
	return false
}

func responseRank(status continuity.ResponseStatus) int {
	switch status {
	case continuity.ResponseDraft:
		return 0
	case continuity.ResponseInReview:
		return 1
	case continuity.ResponseApproved:
		return 2
	case continuity.ResponseTransmitted:
		return 3
	case continuity.ResponseAcknowledged:
		return 4
	default:
		return -1
	}
}

func actionRank(status continuity.ActionStatus) int {
	switch status {
	case continuity.ActionPlanned:
		return 0
	case continuity.ActionInProgress:
		return 1
	case continuity.ActionImplemented:
		return 2
	default:
		return -1
	}
}
