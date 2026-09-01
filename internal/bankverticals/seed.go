package bankverticals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type requirementSpec struct {
	code                 string
	title                string
	statement            string
	anchor               string
	objectiveCode        string
	objectiveName        string
	outcome              string
	implementationName   string
	implementationType   string
	implementationDetail string
	evidenceCode         string
	evidenceName         string
	claim                string
	sourceCodes          []string
	population           map[string]any
	freshnessMinutes     int
	minimumCoverage      float64
	coverage             float64
	conclusion           continuity.EvidenceConclusion
	basis                map[string]any
}

func (s *Service) SeedSample(ctx context.Context, config SeedConfig) ([]Journey, error) {
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
	if _, err := s.continuity.ProgramByCode(ctx, config.TenantID, programCodeNDPA); err == nil {
		return s.List(ctx, config.TenantID)
	} else if err != continuity.ErrNotFound {
		return nil, err
	}

	sourceIDs, err := s.seedSources(ctx, config)
	if err != nil {
		return nil, err
	}
	program, err := s.seedNDPAProgram(ctx, config, sourceIDs)
	if err != nil {
		return nil, err
	}
	if _, err = s.seedNDPAEvidenceRequest(ctx, config, program.Program.ID); err != nil {
		return nil, err
	}
	if _, err = s.seedRegulatoryChange(ctx, config, program, sourceIDs["NDPA-GAID-2025"]); err != nil {
		return nil, err
	}
	if _, err = s.seedAuthorityRequest(ctx, config, program, sourceIDs["NDPC-REQUEST-2026"]); err != nil {
		return nil, err
	}
	if _, err = s.seedLegacyFinding(ctx, config, program, sourceIDs["INTERNAL-AUDIT-2024"]); err != nil {
		return nil, err
	}
	return s.List(ctx, config.TenantID)
}

func (s *Service) seedSources(ctx context.Context, config SeedConfig) (map[string]string, error) {
	existing, err := s.evidence.ListSources(ctx, config.TenantID, 200)
	if err != nil {
		return nil, err
	}
	byCode := map[string]evidence.Source{}
	for _, source := range existing {
		byCode[source.Code] = source
	}
	specs := []evidence.CreateSourceInput{
		{Code: "NDPA-ACT-2023", Name: "Nigeria Data Protection Act 2023", Type: evidence.SourceRegulatory, AuthorityClass: "PRIMARY_LAW", Endpoint: "https://www.ndpc.gov.ng/ndp-act-2023/", ExpectedFreshnessMinutes: 43200},
		{Code: "NDPA-GAID-2025", Name: "NDP Act General Application and Implementation Directive 2025", Type: evidence.SourceRegulatory, AuthorityClass: "REGULATORY_DIRECTIVE", Endpoint: "https://ndpc.gov.ng/", ExpectedFreshnessMinutes: 10080},
		{Code: "PRIVACY-PROCESSING-INVENTORY", Name: "Personal-data processing inventory", Type: evidence.SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", Endpoint: "clearsight://bank/privacy/processing-inventory", ExpectedFreshnessMinutes: 1440},
		{Code: "DATA-RIGHTS-REGISTER", Name: "Data rights request register", Type: evidence.SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", Endpoint: "clearsight://bank/privacy/rights-register", ExpectedFreshnessMinutes: 1440},
		{Code: "PRIVACY-INCIDENT-REGISTER", Name: "Privacy incident and breach register", Type: evidence.SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", Endpoint: "clearsight://bank/privacy/incident-register", ExpectedFreshnessMinutes: 720},
		{Code: "CHANGE-PORTFOLIO", Name: "Technology and product change portfolio", Type: evidence.SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", Endpoint: "clearsight://bank/change-portfolio", ExpectedFreshnessMinutes: 1440},
		{Code: "NDPC-REQUEST-2026", Name: "NDPC information request letter", Type: evidence.SourceDocument, AuthorityClass: "OFFICIAL_CORRESPONDENCE", Endpoint: "clearsight://bank/regulatory/ndpc-request-2026", ExpectedFreshnessMinutes: 525600},
		{Code: "INTERNAL-AUDIT-2024", Name: "2024 privacy controls audit report", Type: evidence.SourceDocument, AuthorityClass: "INTERNAL_ASSURANCE", Endpoint: "clearsight://bank/audit/privacy-2024", ExpectedFreshnessMinutes: 525600},
	}
	ids := map[string]string{}
	for _, spec := range specs {
		spec.TenantID = config.TenantID
		spec.LegalEntityID = config.LegalEntityID
		spec.OwnerPrincipalID = config.OwnerPrincipalID
		source, found := byCode[spec.Code]
		if !found {
			source, err = s.evidence.CreateSource(ctx, spec)
			if err != nil {
				return nil, fmt.Errorf("create source %s: %w", spec.Code, err)
			}
		}
		ids[spec.Code] = source.ID
		if source.LastSuccessAt == nil {
			if _, err = s.evidence.RecordSourceObservation(ctx, evidence.SourceObservation{TenantID: config.TenantID, SourceID: source.ID, ObservedAt: config.Now.Add(-30 * time.Minute), Success: true, LatencyMS: 180, Detail: "Sample source check completed.", RecordedBy: config.ActorID}); err != nil {
				return nil, fmt.Errorf("observe source %s: %w", spec.Code, err)
			}
		}
	}
	return ids, nil
}

func (s *Service) seedNDPAProgram(ctx context.Context, config SeedConfig, sourceIDs map[string]string) (continuity.ProgramAggregate, error) {
	program, err := s.continuity.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID:             config.TenantID,
		LegalEntityID:        config.LegalEntityID,
		Code:                 "NDPA-2023",
		Name:                 "Nigeria data protection",
		Type:                 "PRIVACY",
		OwningFunction:       "Data Protection Office",
		OwnerPrincipalID:     config.OwnerPrincipalID,
		AuthorityPrincipalID: config.SignatoryPrincipalID,
		Jurisdiction:         "Nigeria",
		Scope:                mustJSON(map[string]any{"journey_code": JourneyNDPAContinuous, "bank": config.BankName, "population": "customers, employees, vendors and other data subjects", "sample": true}),
		EffectiveFrom:        config.Now.AddDate(0, -6, 0),
		ActorID:              config.ActorID,
	})
	if err != nil {
		return continuity.ProgramAggregate{}, fmt.Errorf("create NDPA Program: %w", err)
	}

	specs := referenceRequirementSpecs()

	for _, spec := range specs {
		program, err = s.addRequirementBundle(ctx, config, program, sourceIDs, spec)
		if err != nil {
			return continuity.ProgramAggregate{}, err
		}
	}
	program, err = s.continuity.TransitionProgram(ctx, continuity.ProgramTransitionInput{TenantID: config.TenantID, ID: program.Program.ID, ExpectedVersion: program.Program.Version, To: continuity.ProgramActive, ActorID: config.SignatoryPrincipalID, Rationale: "The initial obligations, safeguards and evidence checks were reviewed and approved."})
	if err != nil {
		return continuity.ProgramAggregate{}, fmt.Errorf("activate NDPA Program: %w", err)
	}
	return program, nil
}

func (s *Service) addRequirementBundle(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceIDs map[string]string, spec requirementSpec) (continuity.ProgramAggregate, error) {
	legalSourceID := sourceIDs["NDPA-ACT-2023"]
	if strings.HasPrefix(spec.code, "CAR-") || strings.Contains(spec.anchor, "GAID") {
		legalSourceID = sourceIDs["NDPA-GAID-2025"]
	}
	program, err := s.continuity.AddRequirement(ctx, continuity.AddRequirementInput{TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, SourceID: legalSourceID, Code: spec.code, Title: spec.title, Statement: spec.statement, SourceAnchor: spec.anchor, Modality: "MUST", Actor: "Bank", Action: "Maintain", Object: spec.title, Status: continuity.RequirementApproved, EffectiveFrom: config.Now.AddDate(0, -6, 0), ActorID: config.ActorID})
	if err != nil {
		return program, fmt.Errorf("add requirement %s: %w", spec.code, err)
	}
	requirement := program.Requirements[len(program.Requirements)-1]
	program, err = s.continuity.DetermineApplicability(ctx, continuity.DetermineApplicabilityInput{TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: requirement.ID, Status: continuity.ApplicabilityApplicable, Scope: mustJSON(map[string]any{"bank": config.BankName, "legal_entity_id": config.LegalEntityID}), Rationale: "The bank processes customer, employee, vendor and other personal data in Nigeria.", ApprovedBy: config.ReviewerPrincipalID, EffectiveFrom: config.Now.AddDate(0, -6, 0)})
	if err != nil {
		return program, fmt.Errorf("set applicability %s: %w", spec.code, err)
	}
	program, err = s.continuity.AddControlObjective(ctx, continuity.AddControlObjectiveInput{TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: spec.objectiveCode, Name: spec.objectiveName, Outcome: spec.outcome, Status: continuity.ObjectiveActive, ActorID: config.ActorID})
	if err != nil {
		return program, fmt.Errorf("add safeguard objective %s: %w", spec.code, err)
	}
	objective := program.ControlObjectives[len(program.ControlObjectives)-1]
	program, err = s.continuity.AddControlImplementation(ctx, continuity.AddControlImplementationInput{TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ObjectiveID: objective.ID, Name: spec.implementationName, Description: spec.implementationDetail, ImplementationType: spec.implementationType, OwnerPrincipalID: config.OwnerPrincipalID, Scope: mustJSON(map[string]any{"bank": config.BankName}), Status: continuity.ImplementationPlanned, EffectiveFrom: config.Now.AddDate(0, -3, 0), ActorID: config.ActorID})
	if err != nil {
		return program, fmt.Errorf("add safeguard %s: %w", spec.code, err)
	}
	implementation := program.ControlImplementations[len(program.ControlImplementations)-1]
	program, err = implementReferenceSafeguard(ctx, s.continuity, config, program, implementation.ID)
	if err != nil {
		return program, fmt.Errorf("implement safeguard %s: %w", spec.code, err)
	}
	for index := range program.ControlImplementations {
		if program.ControlImplementations[index].ID == implementation.ID {
			implementation = program.ControlImplementations[index]
			break
		}
	}
	program, err = s.continuity.LinkRequirementControl(ctx, continuity.LinkRequirementControlInput{TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: requirement.ID, ImplementationID: implementation.ID, ActorID: config.ActorID})
	if err != nil {
		return program, fmt.Errorf("link safeguard %s: %w", spec.code, err)
	}
	acceptable := make([]string, 0, len(spec.sourceCodes))
	for _, code := range spec.sourceCodes {
		acceptable = append(acceptable, sourceIDs[code])
	}
	program, err = s.continuity.AddEvidenceContract(ctx, continuity.AddEvidenceContractInput{TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ControlImplementationID: implementation.ID, Code: spec.evidenceCode, Name: spec.evidenceName, Claim: spec.claim, AcceptableSourceIDs: acceptable, PopulationScope: mustJSON(spec.population), FreshnessMinutes: spec.freshnessMinutes, MinimumCoverage: spec.minimumCoverage, IndependenceRequired: true, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: continuity.EvidenceContractDraft, ActorID: config.ActorID})
	if err != nil {
		return program, fmt.Errorf("add evidence check %s: %w", spec.code, err)
	}
	contract := program.EvidenceContracts[len(program.EvidenceContracts)-1]
	program, err = activateReferenceEvidenceCheck(ctx, s.continuity, config, program, contract.ID)
	if err != nil {
		return program, fmt.Errorf("activate evidence check %s: %w", spec.code, err)
	}
	for index := range program.EvidenceContracts {
		if program.EvidenceContracts[index].ID == contract.ID {
			contract = program.EvidenceContracts[index]
			break
		}
	}
	validUntil := config.Now.Add(30 * 24 * time.Hour)
	program, err = s.continuity.RecordEvidenceAssessment(ctx, continuity.RecordEvidenceAssessmentInput{TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ContractID: contract.ID, Conclusion: spec.conclusion, Coverage: spec.coverage, Basis: mustJSON(spec.basis), ValidUntil: &validUntil, AssessedBy: config.ReviewerPrincipalID, AssessedAt: config.Now.Add(-2 * time.Hour)})
	if err != nil {
		return program, fmt.Errorf("assess evidence %s: %w", spec.code, err)
	}
	return program, nil
}

func (s *Service) seedNDPAEvidenceRequest(ctx context.Context, config SeedConfig, programID string) (evidence.Request, error) {
	return s.evidence.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID:         config.TenantID,
		LegalEntityID:    config.LegalEntityID,
		SubjectType:      "PROGRAM",
		SubjectID:        programID,
		Title:            "Provide privacy review records for three planned high-risk changes",
		Purpose:          "Prepare the privacy impact assessment evidence for the next release review.",
		WhyYou:           "You own three planned changes that need an approved privacy review record before release.",
		Sensitivity:      "CONFIDENTIAL",
		AudienceType:     "INTERNAL",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: config.OwnerPrincipalID},
		EstimatedMinutes: 12,
		Deadline:         config.Now.Add(5 * 24 * time.Hour),
		KnownFacts:       map[string]string{"released_high_risk_changes": "9", "approved_assessments": "9", "planned_high_risk_changes": "3"},
		Fields: []evidence.Field{
			{ID: "change_references", Label: "Change references", Type: "TEXT", Required: true, Description: "List the three change or release references."},
			{ID: "privacy_review_records", Label: "Approved privacy review records", Type: "FILE", Required: true, Description: "Attach the approved assessment or the recorded approval decision.", AcceptedFormats: []string{"application/pdf", "text/csv", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}},
		},
		CreatedBy: config.ActorID,
	})
}

func (s *Service) seedRegulatoryChange(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) (continuity.MatterAggregate, error) {
	matter, err := s.continuity.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: config.TenantID, Type: continuity.MatterRegulatoryChange, Priority: 4, Title: "Implement GAID 2025 annual return requirements", Summary: "The bank must update its annual Compliance Audit Return process, evidence ownership and filing timetable.", Scope: mustJSON(map[string]any{"journey_code": JourneyRegulatoryChange, "bank": config.BankName, "sample": true, "affected_processes": []string{"privacy governance", "annual compliance audit return", "DPCO review"}}), SourceType: "REGULATORY", SourceID: sourceID, TriggerType: "REQUIREMENT_CHANGED", TriggerKey: "sample:gaid-2025-car", KnownFacts: mustJSON(map[string]any{"filing_deadline": "31 March", "filing_channel": "licensed DPCO", "current_evidence_sections": 8, "required_sections": 10}), MissingFacts: mustJSON([]string{"approved owner for the cross-border transfer section", "final DPCO evidence checklist"}), Contradictions: mustJSON([]string{}), OwnerPrincipalID: config.OwnerPrincipalID, RequiredAuthority: "AUTHORIZER", DueAt: timePointer(config.Now.Add(14 * 24 * time.Hour)), ProgramID: program.Program.ID, ActorID: config.ActorID})
	if err != nil {
		return continuity.MatterAggregate{}, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterAssessment, ActorID: config.ActorID, Rationale: "The official source and affected privacy governance processes were recorded."})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterDecisionRequired, ActorID: config.ActorID, Rationale: "Management approval is required for the revised annual return process and ownership."})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.AddDecision(ctx, continuity.AddDecisionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Type: "IMPLEMENTATION_APPROACH", Status: continuity.DecisionApproved, Options: mustJSON([]string{"UPDATE_CURRENT_PROCESS", "CREATE_SEPARATE_GAID_PROCESS", "NO_CHANGE_REQUIRED"}), SelectedOption: "UPDATE_CURRENT_PROCESS", Rationale: "The existing annual return process will be updated to use source-linked evidence owners and an earlier internal review date.", Conditions: mustJSON([]string{"DPO confirms the final evidence checklist", "DPCO review starts at least 30 days before filing"}), AuthorityPrincipalID: config.SignatoryPrincipalID})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterActionsInProgress, ActorID: config.ActorID, Rationale: "The approved process changes are being implemented."})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: "Update the annual return evidence checklist", Description: "Assign each evidence section, record its authoritative source and move the internal approval date to 1 March.", OwnerPrincipalID: config.OwnerPrincipalID, DueAt: timePointer(config.Now.Add(10 * 24 * time.Hour)), ActorID: config.ActorID})
	if err != nil {
		return matter, err
	}
	actionID := matter.Actions[len(matter.Actions)-1].ID
	matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionInProgress, ActorID: config.OwnerPrincipalID})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.AddVerificationContract(ctx, continuity.AddVerificationContractInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: actionID, ExpectedOutcome: "Every required annual return evidence section has an owner, authoritative source, internal approval date and DPCO review status.", Baseline: mustJSON(map[string]any{"complete_sections": 8, "required_sections": 10}), Scope: mustJSON(map[string]any{"journey_code": JourneyRegulatoryChange, "filing_year": 2027}), Threshold: mustJSON(map[string]any{"complete_sections": 10, "approved": true}), ObservationPeriodMinutes: 0, AuthorityPrincipalID: config.ReviewerPrincipalID, FailureResponse: "BLOCK_CLOSE", ActorID: config.ActorID})
	return matter, err
}

func (s *Service) seedAuthorityRequest(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) (continuity.MatterAggregate, error) {
	matter, err := s.continuity.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: config.TenantID, Type: continuity.MatterAuthorityRequest, Priority: 5, Title: "Respond to NDPC request for privacy incident records", Summary: "The authority requested the incident assessment, customer communication decision and supporting records for a reported event.", Scope: mustJSON(map[string]any{"journey_code": JourneyAuthorityRequest, "bank": config.BankName, "sample": true, "access": "RESTRICTED", "allowed_principal_ids": []string{config.ActorID, config.OwnerPrincipalID, config.ReviewerPrincipalID, config.SignatoryPrincipalID}, "case_reference": "NDPC/ENF/2026/0142"}), SourceType: "AUTHORITY_CORRESPONDENCE", SourceID: sourceID, TriggerType: "AUTHORITY_REQUEST_RECEIVED", TriggerKey: "sample:ndpc-request-2026-0142", KnownFacts: mustJSON(map[string]any{"received_at": config.Now.Add(-72 * time.Hour), "response_due_at": config.Now.Add(7 * 24 * time.Hour), "requested_items": 4}), MissingFacts: mustJSON([]string{"final legal privilege review"}), Contradictions: mustJSON([]string{}), OwnerPrincipalID: config.OwnerPrincipalID, RequiredAuthority: "SIGNATORY", DueAt: timePointer(config.Now.Add(7 * 24 * time.Hour)), ProgramID: program.Program.ID, ActorID: config.ActorID})
	if err != nil {
		return continuity.MatterAggregate{}, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterAssessment, ActorID: config.ActorID, Rationale: "The authority, deadline, requested records and restricted handling group were confirmed."})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterResponsePreparation, ActorID: config.ActorID, Rationale: "The response package and signatory approval are being prepared."})
	if err != nil {
		return matter, err
	}
	request, err := s.evidence.CreateRequest(ctx, evidence.CreateRequestInput{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, SubjectType: "MATTER", SubjectID: matter.Matter.ID, Title: "Provide incident containment and customer communication records", Purpose: "Complete the restricted NDPC response package.", WhyYou: "You own the incident response records requested by Regulatory Affairs.", Sensitivity: "RESTRICTED", AudienceType: "INTERNAL", Recipient: evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: config.OwnerPrincipalID}, EstimatedMinutes: 10, Deadline: config.Now.Add(48 * time.Hour), KnownFacts: map[string]string{"case_reference": "NDPC/ENF/2026/0142", "incident_reference": "PRI-2026-008"}, Fields: []evidence.Field{{ID: "containment_record", Label: "Containment record reference", Type: "TEXT", Required: true, Description: "Enter the incident containment record reference."}, {ID: "communication_decision", Label: "Customer communication decision", Type: "TEXT", Required: true, Description: "State the approved decision and approving authority."}}, CreatedBy: config.ActorID})
	if err != nil {
		return matter, err
	}
	if _, err = s.evidence.Submit(ctx, evidence.Submission{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, RequestID: request.ID, SubmittedBy: config.OwnerPrincipalID, Channel: "INTERNAL", Answers: formcontract.TextAnswers(map[string]string{"containment_record": "Incident containment record PRI-2026-008", "communication_decision": "No direct customer notice was approved after the documented impact assessment."}), ExpectedVersion: request.Version}); err != nil {
		return matter, err
	}
	matter, err = s.continuity.AddResponsePackage(ctx, continuity.AddResponsePackageInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Purpose: "Respond to NDPC request NDPC/ENF/2026/0142", Audience: "Nigeria Data Protection Commission", Manifest: mustJSON([]map[string]any{{"classification": "RESTRICTED", "evidence_request_id": request.ID}, {"document": "incident assessment"}, {"document": "containment record"}, {"document": "notification decision"}, {"document": "customer communication decision"}}), ActorID: config.ActorID})
	if err != nil {
		return matter, err
	}
	responseID := matter.ResponsePackages[len(matter.ResponsePackages)-1].ID
	for _, status := range []continuity.ResponseStatus{continuity.ResponseInReview, continuity.ResponseApproved, continuity.ResponseTransmitted, continuity.ResponseAcknowledged} {
		actor := config.ActorID
		if status == continuity.ResponseApproved {
			actor = config.SignatoryPrincipalID
		}
		matter, err = s.continuity.TransitionResponsePackage(ctx, continuity.TransitionResponseInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ResponseID: responseID, ExpectedVersion: matter.Matter.Version, To: status, ActorID: actor, Rationale: responseRationale(status)})
		if err != nil {
			return matter, err
		}
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterVerification, ActorID: config.ActorID, Rationale: "The response was transmitted and the authority acknowledgement was recorded."})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterClosed, ActorID: config.SignatoryPrincipalID, Rationale: "The authority acknowledged receipt of the approved response package."})
	return matter, err
}

func (s *Service) seedLegacyFinding(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) (continuity.MatterAggregate, error) {
	matter, err := s.continuity.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: config.TenantID, Type: continuity.MatterAuditFinding, Priority: 3, Title: "Close the legacy retention-schedule finding", Summary: "The 2024 privacy audit found that processing records did not consistently include an approved retention period.", Scope: mustJSON(map[string]any{"journey_code": JourneyFindingRemediation, "bank": config.BankName, "sample": true, "finding_reference": "IA-PRIV-2024-07", "affected_records": 14}), SourceType: "INTERNAL_AUDIT", SourceID: sourceID, TriggerType: "LEGACY_FINDING_IMPORT", TriggerKey: "sample:legacy-finding-ia-priv-2024-07", KnownFacts: mustJSON(map[string]any{"affected_records": 14, "business_areas": 4, "original_due_date": config.Now.AddDate(0, -3, 0)}), MissingFacts: mustJSON([]string{}), Contradictions: mustJSON([]string{}), OwnerPrincipalID: config.OwnerPrincipalID, RequiredAuthority: "REVIEWER", DueAt: timePointer(config.Now.Add(-7 * 24 * time.Hour)), ProgramID: program.Program.ID, ActorID: config.ActorID})
	if err != nil {
		return continuity.MatterAggregate{}, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterAssessment, ActorID: config.ActorID, Rationale: "The legacy finding, affected records and current owners were confirmed."})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterActionsInProgress, ActorID: config.ActorID, Rationale: "The remediation plan was assigned to the processing-record owners."})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: "Approve retention periods for the 14 affected records", Description: "Record the approved retention period, approving owner and policy reference for each affected processing activity.", OwnerPrincipalID: config.OwnerPrincipalID, DueAt: timePointer(config.Now.Add(-10 * 24 * time.Hour)), ActorID: config.ActorID})
	if err != nil {
		return matter, err
	}
	actionID := matter.Actions[len(matter.Actions)-1].ID
	for _, status := range []continuity.ActionStatus{continuity.ActionInProgress, continuity.ActionImplemented} {
		matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version, To: status, ActorID: config.OwnerPrincipalID})
		if err != nil {
			return matter, err
		}
	}
	matter, err = s.continuity.AddVerificationContract(ctx, continuity.AddVerificationContractInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: actionID, ExpectedOutcome: "All 14 affected processing records contain an approved retention period, owner and policy reference.", Baseline: mustJSON(map[string]any{"complete_records": 0, "affected_records": 14}), Scope: mustJSON(map[string]any{"finding_reference": "IA-PRIV-2024-07"}), MeasurementSourceID: sourceID, Threshold: mustJSON(map[string]any{"complete_records": 14, "exceptions": 0}), ObservationPeriodMinutes: 0, AuthorityPrincipalID: config.ReviewerPrincipalID, FailureResponse: "REOPEN", ActorID: config.ActorID})
	if err != nil {
		return matter, err
	}
	contract := activeVerificationContract(matter, actionID)
	if contract == nil {
		return matter, fmt.Errorf("seeded outcome check is unavailable")
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterVerification, ActorID: config.ActorID, Rationale: "Implementation is complete and independent checking has started."})
	if err != nil {
		return matter, err
	}
	actions := currentActions(matter.Actions)
	var action *continuity.Action
	for index := range actions {
		if actions[index].ID == actionID {
			value := actions[index]
			action = &value
			break
		}
	}
	if action == nil || action.ImplementedAt == nil {
		return matter, fmt.Errorf("seeded remediation action is not implemented")
	}
	observedAt := *action.ImplementedAt
	if contract.CreatedAt.After(observedAt) {
		observedAt = contract.CreatedAt
	}
	observedAt = observedAt.Add(time.Duration(contract.ObservationPeriodMinutes) * time.Minute)
	matter, err = s.continuity.RecordVerificationResult(ctx, continuity.RecordVerificationResultInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ContractID: contract.ID, Result: continuity.VerificationPassed, Observations: mustJSON(map[string]any{"complete_records": 14, "exceptions": 0}), EvidenceReferences: mustJSON([]string{"processing inventory export", "retention owner approvals", "sample re-performance"}), ReviewerPrincipalID: config.ReviewerPrincipalID, Rationale: "The reviewer re-performed the check and confirmed all 14 records contain the required approved retention information.", ObservedAt: observedAt})
	if err != nil {
		return matter, err
	}
	matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterClosed, ActorID: config.ReviewerPrincipalID, Rationale: "The remediation was implemented and the independent outcome check passed."})
	return matter, err
}

func normalizeSeedConfig(config SeedConfig) SeedConfig {
	if config.Now.IsZero() {
		config.Now = time.Now().UTC()
	} else {
		config.Now = config.Now.UTC()
	}
	if strings.TrimSpace(config.BankName) == "" {
		config.BankName = "Clear Bank Nigeria"
	}
	if config.OwnerPrincipalID == "" {
		config.OwnerPrincipalID = config.ActorID
	}
	if config.ContributorPrincipalID == "" {
		config.ContributorPrincipalID = config.OwnerPrincipalID
	}
	if config.ReviewerPrincipalID == "" {
		config.ReviewerPrincipalID = config.ActorID
	}
	if config.SignatoryPrincipalID == "" {
		config.SignatoryPrincipalID = config.ActorID
	}
	return config
}

func validateSeedConfig(config SeedConfig) error {
	if strings.TrimSpace(config.TenantID) == "" || strings.TrimSpace(config.ActorID) == "" || strings.TrimSpace(config.OwnerPrincipalID) == "" || strings.TrimSpace(config.ContributorPrincipalID) == "" || strings.TrimSpace(config.ReviewerPrincipalID) == "" || strings.TrimSpace(config.SignatoryPrincipalID) == "" {
		return fmt.Errorf("tenant, actor, owner, contributor, reviewer and signatory are required")
	}
	return nil
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func responseRationale(status continuity.ResponseStatus) string {
	switch status {
	case continuity.ResponseInReview:
		return "The response package is complete and ready for legal and signatory review."
	case continuity.ResponseApproved:
		return "The signatory approved the response package for transmission."
	case continuity.ResponseTransmitted:
		return "The approved response was transmitted through the recorded authority channel."
	case continuity.ResponseAcknowledged:
		return "The authority acknowledgement was received and recorded."
	default:
		return ""
	}
}
