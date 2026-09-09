package bankverticals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

const operatingDemoMarker = "bank_operating_demo_v1"

type OperatingDemoRecord struct {
	ID        string `json:"id"`
	ProgramID string `json:"program_id,omitempty"`
	Status    string `json:"status"`
	Version   int64  `json:"version"`
	Eligible  bool   `json:"eligible"`
}

type OperatingDemoCatalog struct {
	StartedAt time.Time                      `json:"started_at"`
	Programs  map[string]OperatingDemoRecord `json:"programs"`
	Forms     map[string]OperatingDemoRecord `json:"forms"`
	Matters   map[string]OperatingDemoRecord `json:"matters"`
}

type operatingProgramSpec struct {
	key, code, name, kind, owner string
}

type operatingMatterSpec struct {
	key, title, summary, programKey string
	kind                            continuity.MatterType
	priority                        int
	target                          continuity.MatterStatus
	actions                         []operatingActionSpec
}

type operatingActionSpec struct {
	title, description, owner, origin string
	status                            continuity.ActionStatus
	verify                            bool
}

// EnsureOperatingDemo adds a compact, fictional bank operating population.
// Existing records are found by exact managed identifiers and are never reset.
func (s *Service) EnsureOperatingDemo(ctx context.Context, config SeedConfig) (OperatingDemoCatalog, error) {
	if s == nil || s.continuity == nil || s.monitoring == nil {
		return OperatingDemoCatalog{}, fmt.Errorf("bank operating demo is unavailable")
	}
	config = normalizeSeedConfig(config)
	if err := validateSeedConfig(config); err != nil {
		return OperatingDemoCatalog{}, err
	}
	if config.ActorID == config.ReviewerPrincipalID {
		return OperatingDemoCatalog{}, monitoring.ErrMakerChecker
	}
	ctx = continuity.WithTrustedSystemEntityScope(ctx, config.TenantID, config.LegalEntityID)
	entityID, err := s.continuity.ResolveLegalEntity(ctx, config.TenantID, config.LegalEntityID)
	if err != nil {
		return OperatingDemoCatalog{}, fmt.Errorf("resolve operating demo legal entity: %w", err)
	}
	config.LegalEntityID = entityID
	ctx = continuity.WithTrustedSystemEntityScope(ctx, config.TenantID, entityID)

	catalog := OperatingDemoCatalog{
		Programs: map[string]OperatingDemoRecord{}, Forms: map[string]OperatingDemoRecord{}, Matters: map[string]OperatingDemoRecord{},
	}
	programSpecs := []operatingProgramSpec{
		{"third_party_risk", "DEMO-THIRD-PARTY-RISK", "Third-party risk", "THIRD_PARTY_RISK", "Third-Party Risk"},
		{"it_risk_security", "DEMO-IT-RISK-SECURITY", "IT risk and security", "IT_RISK", "Information Security"},
		{"privacy", programCodeNDPA, "Nigeria data protection", "PRIVACY", "Data Protection Office"},
		{"operational_resilience", "DEMO-OPERATIONAL-RESILIENCE", "Operational resilience", "OPERATIONAL_RISK", "Operational Risk"},
		{"regulatory_compliance", "DEMO-REGULATORY-COMPLIANCE", "Regulatory compliance", "COMPLIANCE", "Compliance"},
	}
	programs := map[string]continuity.ProgramAggregate{}
	for _, spec := range programSpecs {
		program, ensureErr := s.ensureOperatingProgram(ctx, config, spec)
		if ensureErr != nil {
			return OperatingDemoCatalog{}, ensureErr
		}
		programs[spec.key] = program
		catalog.Programs[spec.key] = OperatingDemoRecord{ID: program.Program.ID, Status: string(program.Program.Status), Version: program.Program.Version, Eligible: program.Program.Status == continuity.ProgramActive}
		if catalog.StartedAt.IsZero() || program.Program.CreatedAt.Before(catalog.StartedAt) {
			catalog.StartedAt = program.Program.CreatedAt
		}
	}

	formSpecs := []struct {
		key, programKey string
		input           monitoring.CreateFormInput
	}{
		{"vendor_due_diligence", "third_party_risk", vendorDueDiligenceFormInput(programs["third_party_risk"].Program.ID, entityID)},
		{"vendor_compliance", "third_party_risk", ReferenceThirdPartyRiskComplianceForm(programs["third_party_risk"].Program.ID, entityID)},
		{"certification_refresh", "third_party_risk", vendorCertificationRefreshFormInput(programs["third_party_risk"].Program.ID, entityID)},
		{"vendor_control_attestation", "third_party_risk", vendorControlAttestationForm(programs["third_party_risk"].Program.ID, entityID)},
		{"access_review", "it_risk_security", operatingForm(programs["it_risk_security"].Program.ID, entityID, "IT-ACCESS-REVIEW", "Access review", "Confirm access review results and record unresolved access.", "review", "Review result", []formcontract.Field{
			{ID: "review_completed", SectionID: "review", Label: "Was the access review completed?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "unresolved_access", SectionID: "review", Label: "Unresolved access", Type: formcontract.TypeLongText, Required: true},
		})},
		{"branch_control_return", "operational_resilience", operatingForm(programs["operational_resilience"].Program.ID, entityID, "BRANCH-CONTROL-RETURN", "Branch control return", "Record branch control checks and exceptions for review.", "controls", "Control checks", []formcontract.Field{
			{ID: "cash_count_complete", SectionID: "controls", Label: "Was the cash count completed?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "callover_complete", SectionID: "controls", Label: "Was independent callover completed?", Type: formcontract.TypeYesNo, Required: true},
		})},
		{"continuity_test", "operational_resilience", operatingForm(programs["operational_resilience"].Program.ID, entityID, "SERVICE-CONTINUITY-TEST", "Service continuity test", "Record recovery targets, test results and required retesting.", "test", "Test result", []formcontract.Field{
			{ID: "rto_met", SectionID: "test", Label: "Was the recovery time target met?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "actual_recovery_minutes", SectionID: "test", Label: "Actual recovery time in minutes", Type: formcontract.TypeInteger, Required: true},
		})},
		{"privacy_screening", "privacy", operatingForm(programs["privacy"].Program.ID, entityID, "PRIVACY-CHANGE-SCREENING", "Privacy screening", "Check whether a change needs a DPIA or further privacy review.", "screen", "Change screening", []formcontract.Field{
			{ID: "personal_data", SectionID: "screen", Label: "Does the change use personal data?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "high_risk", SectionID: "screen", Label: "Could the processing create high risk for people?", Type: formcontract.TypeYesNo, Required: true},
		})},
	}
	for _, spec := range formSpecs {
		form, ensureErr := s.ensureOperatingForm(ctx, config, spec.input)
		if ensureErr != nil {
			return OperatingDemoCatalog{}, ensureErr
		}
		catalog.Forms[spec.key] = OperatingDemoRecord{ID: form.ID, ProgramID: form.ProgramID, Status: string(form.Status), Version: form.Version, Eligible: form.Status == monitoring.LifecycleActive && form.IsCurrent}
	}

	matterSpecs := []operatingMatterSpec{
		{key: "access_remediation", title: "Remove access retained after role changes", summary: "The latest access review found active rights that no longer match current roles.", programKey: "it_risk_security", kind: continuity.MatterControlGap, priority: 2, target: continuity.MatterActionsInProgress, actions: []operatingActionSpec{
			{title: "Remove dormant administrator access", description: "Remove the listed dormant administrator accounts and retain the change record.", owner: config.ContributorPrincipalID, origin: operatingDemoMarker + ":access:remove", status: continuity.ActionBlocked},
			{title: "Update role access rules", description: "Apply the approved role access rules to the affected application.", owner: config.OwnerPrincipalID, origin: operatingDemoMarker + ":access:rules", status: continuity.ActionImplemented, verify: true},
		}},
		{key: "branch_control_followup", title: "Recheck missed branch callover", summary: "A branch control return reported a missed independent callover.", programKey: "operational_resilience", kind: continuity.MatterControlGap, priority: 2, target: continuity.MatterVerification, actions: []operatingActionSpec{{title: "Complete branch callover", description: "Complete and retain the independent callover record for the affected day.", owner: config.ContributorPrincipalID, origin: operatingDemoMarker + ":branch:callover", status: continuity.ActionImplemented, verify: true}}},
		{key: "continuity_retest", title: "Retest supplier recovery time", summary: "The latest continuity test exceeded the recovery time target.", programKey: "operational_resilience", kind: continuity.MatterVendorDeficiency, priority: 2, target: continuity.MatterActionsInProgress, actions: []operatingActionSpec{{title: "Run recovery retest", description: "Run the supplier recovery test again after the corrective work is complete.", owner: config.OwnerPrincipalID, origin: operatingDemoMarker + ":continuity:retest", status: continuity.ActionInProgress}}},
		{key: "privacy_screening", title: "Decide whether the mobile change needs a DPIA", summary: "Privacy screening identified personal data and possible high-risk processing.", programKey: "privacy", kind: continuity.MatterRiskSituation, priority: 2, target: continuity.MatterDecisionRequired},
		{key: "source_verification", title: "Confirm the current regulatory source", summary: "The compliance register entry needs confirmation against an official current source.", programKey: "regulatory_compliance", kind: continuity.MatterEvidenceContradiction, priority: 3, target: continuity.MatterAssessment},
		{key: "filing_preparation", title: "Prepare the next compliance filing", summary: "The filing owner is compiling the return and supporting approval record.", programKey: "regulatory_compliance", kind: continuity.MatterOverdueObligation, priority: 3, target: continuity.MatterActionsInProgress, actions: []operatingActionSpec{{title: "Complete filing pack", description: "Complete the return, supporting schedule and approval record.", owner: config.ContributorPrincipalID, origin: operatingDemoMarker + ":filing:pack", status: continuity.ActionInProgress}}},
		{key: "loss_recovery", title: "Recover the outstanding operational loss", summary: "An operational loss has been recorded and the outstanding recovery remains assigned.", programKey: "operational_resilience", kind: continuity.MatterOperationalLoss, priority: 2, target: continuity.MatterActionsInProgress, actions: []operatingActionSpec{{title: "Complete loss recovery", description: "Pursue the recorded recovery and attach evidence of the recovered amount.", owner: config.OwnerPrincipalID, origin: operatingDemoMarker + ":loss:recovery", status: continuity.ActionInProgress}}},
	}
	for _, spec := range matterSpecs {
		matter, ensureErr := s.ensureOperatingMatter(ctx, config, programs[spec.programKey].Program.ID, spec)
		if ensureErr != nil {
			return OperatingDemoCatalog{}, ensureErr
		}
		catalog.Matters[spec.key] = OperatingDemoRecord{ID: matter.Matter.ID, ProgramID: programs[spec.programKey].Program.ID, Status: string(matter.Matter.Status), Version: matter.Matter.Version, Eligible: matter.Matter.Status != continuity.MatterClosed && matter.Matter.Status != continuity.MatterCancelled}
	}
	return catalog, nil
}

func (s *Service) ensureOperatingProgram(ctx context.Context, config SeedConfig, spec operatingProgramSpec) (continuity.ProgramAggregate, error) {
	program, err := s.continuity.ProgramByCode(ctx, config.TenantID, spec.code)
	if err == nil {
		if spec.code != programCodeNDPA && scopeString(program.Program.Scope, "seed_package") != operatingDemoMarker {
			return continuity.ProgramAggregate{}, fmt.Errorf("program code %s is already in use", spec.code)
		}
		return program, nil
	}
	if !errors.Is(err, continuity.ErrNotFound) {
		return continuity.ProgramAggregate{}, err
	}
	program, err = s.continuity.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, Code: spec.code, Name: spec.name, Type: spec.kind,
		OwningFunction: spec.owner, OwnerPrincipalID: config.OwnerPrincipalID, AuthorityPrincipalID: config.SignatoryPrincipalID,
		Jurisdiction: "Nigeria", Scope: mustJSON(map[string]any{"sample": true, "seed_package": operatingDemoMarker, "source": "customer-provided bank operating samples"}),
		EffectiveFrom: config.Now.AddDate(0, -6, 0), ActorID: config.ActorID,
	})
	if err != nil {
		return continuity.ProgramAggregate{}, fmt.Errorf("create %s program: %w", spec.name, err)
	}
	program, err = s.continuity.AddRequirement(ctx, continuity.AddRequirementInput{
		TenantID: config.TenantID, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "DEMO-SCOPE", Title: "Maintain the assigned operating register", Statement: "Maintain the assigned sample register and resolve recorded gaps.",
		Modality: "MUST", Actor: spec.owner, Action: "Maintain", Object: spec.name, Status: continuity.RequirementApproved,
		EffectiveFrom: config.Now.AddDate(0, -6, 0), ActorID: config.ActorID,
	})
	if err != nil {
		return continuity.ProgramAggregate{}, fmt.Errorf("add %s sample requirement: %w", spec.name, err)
	}
	return s.continuity.TransitionProgram(ctx, continuity.ProgramTransitionInput{TenantID: config.TenantID, ID: program.Program.ID, ExpectedVersion: program.Program.Version, To: continuity.ProgramActive, ActorID: config.SignatoryPrincipalID, Rationale: "The sample operating scope and ownership were reviewed."})
}

func vendorControlAttestationForm(programID, entityID string) monitoring.CreateFormInput {
	weightedYes := func(id, fieldID, label string, weight int) formcontract.ScoreContribution {
		return formcontract.ScoreContribution{ID: id, Label: label, Weight: weight, Required: true,
			Predicate:   formcontract.Predicate{FieldID: fieldID, Operator: formcontract.PredicateEquals, Values: []string{"Yes"}},
			MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingIndeterminate}
	}
	return monitoring.CreateFormInput{
		ProgramID: programID, LegalEntityID: entityID, Code: "VENDOR-CONTROL-ATTESTATION", Name: "Vendor control confirmation",
		Purpose:         "Confirm the current controls used to protect the service and identify unresolved weaknesses for review.",
		ResponsibleTeam: "Third-Party Risk", Tags: []string{"sample", operatingDemoMarker}, Jurisdiction: "Nigeria", Industry: "Banking",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		ScoringMode:  formcontract.ScoringCompliance,
		ScoreProfile: &formcontract.ScoreProfile{
			Version: "vendor-control-confirmation-v1", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor,
			Contributions: []formcontract.ScoreContribution{
				weightedYes("encryption", "encryption_enabled", "Service data is encrypted", 35),
				weightedYes("mfa", "admin_mfa", "Administrator access uses MFA", 35),
				weightedYes("testing", "annual_security_test", "Security testing is current", 30),
			},
			Rules: []formcontract.ScoreRule{{
				ID: "critical-weakness", Label: "An unresolved critical weakness is reported",
				Predicate: formcontract.Predicate{FieldID: "critical_weakness", Operator: formcontract.PredicateEquals, Values: []string{"Yes"}},
				Effect:    formcontract.RuleEffect{Kind: formcontract.EffectDisqualify},
			}},
			Bands: formcontract.DefaultConcernBands(),
		},
		Sections: []formcontract.Section{{ID: "controls", Title: "Current controls"}},
		Fields: []formcontract.Field{
			{ID: "encryption_enabled", SectionID: "controls", Label: "Is service data encrypted at rest and in transit?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "admin_mfa", SectionID: "controls", Label: "Is MFA required for administrator access?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "annual_security_test", SectionID: "controls", Label: "Was independent security testing completed in the last 12 months?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "critical_weakness", SectionID: "controls", Label: "Is any critical security weakness unresolved?", Type: formcontract.TypeYesNo, Required: true},
		},
	}
}

func operatingForm(programID, entityID, code, name, purpose, sectionID, sectionTitle string, fields []formcontract.Field) monitoring.CreateFormInput {
	return monitoring.CreateFormInput{ProgramID: programID, LegalEntityID: entityID, Code: code, Name: name, Purpose: purpose,
		ResponsibleTeam: "Risk and Compliance", Tags: []string{"sample", operatingDemoMarker}, Jurisdiction: "Nigeria", Industry: "Banking",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     []formcontract.Section{{ID: sectionID, Title: sectionTitle}}, Fields: fields}
}

func (s *Service) ensureOperatingForm(ctx context.Context, config SeedConfig, input monitoring.CreateFormInput) (monitoring.FormTemplate, error) {
	maker := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ActorID}
	checker := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ReviewerPrincipalID}
	form, err := s.monitoring.LatestFormByCode(ctx, maker, input.ProgramID, input.Code)
	if err == nil {
		return form, nil
	}
	if !errors.Is(err, monitoring.ErrNotFound) {
		return monitoring.FormTemplate{}, err
	}
	if !containsString(input.Tags, operatingDemoMarker) {
		input.Tags = append(input.Tags, "sample", operatingDemoMarker)
	}
	form, err = s.monitoring.CreateForm(ctx, maker, input)
	if err != nil {
		return monitoring.FormTemplate{}, fmt.Errorf("create form %s: %w", input.Code, err)
	}
	form, err = s.monitoring.TransitionForm(ctx, maker, monitoring.TransitionInput{ID: form.ID, ProgramID: input.ProgramID, LegalEntityID: config.LegalEntityID, ExpectedVersion: form.Version, To: monitoring.LifecyclePendingApproval})
	if err != nil {
		return monitoring.FormTemplate{}, fmt.Errorf("submit form %s: %w", input.Code, err)
	}
	form, err = s.monitoring.TransitionForm(ctx, checker, monitoring.TransitionInput{ID: form.ID, ProgramID: input.ProgramID, LegalEntityID: config.LegalEntityID, ExpectedVersion: form.Version, To: monitoring.LifecycleActive})
	if err != nil {
		return monitoring.FormTemplate{}, fmt.Errorf("activate form %s: %w", input.Code, err)
	}
	return form, nil
}

func (s *Service) ensureOperatingMatter(ctx context.Context, config SeedConfig, programID string, spec operatingMatterSpec) (continuity.MatterAggregate, error) {
	triggerKey := operatingDemoMarker + ":" + spec.key
	matter, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerKey)
	if err == nil {
		if scopeString(matter.Matter.Scope, "seed_package") != operatingDemoMarker {
			return continuity.MatterAggregate{}, fmt.Errorf("matter trigger %s is already in use", triggerKey)
		}
		return matter, nil
	}
	if !errors.Is(err, continuity.ErrNotFound) {
		return continuity.MatterAggregate{}, err
	}
	due := config.Now.AddDate(0, 0, 21)
	matter, err = s.continuity.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, Type: spec.kind, Priority: spec.priority, Title: spec.title, Summary: spec.summary,
		Scope: mustJSON(map[string]any{"sample": true, "seed_package": operatingDemoMarker}), TriggerType: "SAMPLE_REGISTER_ENTRY", TriggerID: spec.key, TriggerKey: triggerKey,
		KnownFacts: mustJSON(map[string]any{"source": "customer-provided bank operating samples", "sample": true}), MissingFacts: mustJSON([]any{}), Contradictions: mustJSON([]any{}),
		OwnerPrincipalID: config.OwnerPrincipalID, RequiredAuthority: "RISK_REVIEWER", DueAt: &due, ProgramID: programID, ActorID: config.ActorID,
	})
	if err != nil {
		return continuity.MatterAggregate{}, fmt.Errorf("create work item %s: %w", spec.key, err)
	}
	for matter.Matter.Status != spec.target {
		next := nextOperatingMatterStatus(matter.Matter.Status, spec.target)
		matter, err = s.continuity.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: next, ActorID: config.ActorID, Rationale: "The sample work item was moved to its current operating stage."})
		if err != nil {
			return continuity.MatterAggregate{}, fmt.Errorf("advance work item %s: %w", spec.key, err)
		}
	}
	for _, actionSpec := range spec.actions {
		matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: actionSpec.title, Description: actionSpec.description, OwnerPrincipalID: actionSpec.owner, DueAt: &due, ActorID: config.ActorID, OriginKey: actionSpec.origin})
		if err != nil {
			return continuity.MatterAggregate{}, err
		}
		action := matter.Actions[len(matter.Actions)-1]
		if actionSpec.status == continuity.ActionImplemented {
			matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionInProgress, ActorID: actionSpec.owner})
			if err == nil {
				matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionImplemented, ActorID: actionSpec.owner})
			}
		} else if actionSpec.status != continuity.ActionPlanned {
			matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: actionSpec.status, ActorID: actionSpec.owner, Rationale: "A dependency remains unresolved in the sample work item."})
		}
		if err != nil {
			return continuity.MatterAggregate{}, err
		}
		if actionSpec.verify {
			matter, err = s.continuity.AddVerificationContract(ctx, continuity.AddVerificationContractInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: action.ID, ExpectedOutcome: "The corrective action operates as intended.", Baseline: mustJSON(map[string]any{"sample": true}), Scope: mustJSON(map[string]any{"sample": true}), Threshold: mustJSON(map[string]any{"result": "PASS"}), ObservationPeriodMinutes: 1440, ReviewerCandidateID: config.ReviewerPrincipalID, AuthorityPrincipalID: config.ReviewerPrincipalID, FailureResponse: "BLOCK_CLOSE", ActorID: config.ActorID})
			if err != nil {
				return continuity.MatterAggregate{}, err
			}
		}
	}
	return matter, nil
}

func nextOperatingMatterStatus(current, target continuity.MatterStatus) continuity.MatterStatus {
	if current == continuity.MatterInitialReview {
		return continuity.MatterAssessment
	}
	if current == continuity.MatterAssessment {
		if target == continuity.MatterAssessment {
			return target
		}
		if target == continuity.MatterDecisionRequired {
			return continuity.MatterDecisionRequired
		}
		return continuity.MatterActionsInProgress
	}
	if current == continuity.MatterActionsInProgress && target == continuity.MatterVerification {
		return continuity.MatterVerification
	}
	return target
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}
