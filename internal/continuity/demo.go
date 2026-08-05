package continuity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func SeedDemo(ctx context.Context, service *Service) error {
	now := time.Now().UTC()
	privacy, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank-demo", Code: "NDPA", Name: "Data protection", Type: "PRIVACY", OwningFunction: "Privacy Office", OwnerPrincipalID: "user-demo", AuthorityPrincipalID: "user-demo", Jurisdiction: "Nigeria", Scope: json.RawMessage(`{"legal_entity":"Demonstration Bank Nigeria"}`), EffectiveFrom: now.AddDate(0, -6, 0), ActorID: "user-demo"})
	if err != nil && err != ErrDuplicate {
		return fmt.Errorf("create privacy demo: %w", err)
	}
	if err == nil {
		privacy, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "bank-demo", ProgramID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, Code: "ROPA-CURRENT", Title: "Keep processing records current", Statement: "The bank must keep an up-to-date record of personal-data processing activities.", SourceAnchor: "NDPA governance requirement", Modality: "MUST", Actor: "Bank", Action: "Maintain", Object: "Processing records", Status: RequirementApproved, EffectiveFrom: now.AddDate(0, -6, 0), ActorID: "user-demo"})
		if err != nil {
			return err
		}
		requirement := privacy.Requirements[0]
		privacy, err = service.DetermineApplicability(ctx, DetermineApplicabilityInput{TenantID: "bank-demo", ProgramID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, RequirementID: requirement.ID, Status: ApplicabilityApplicable, Scope: json.RawMessage(`{"legal_entity":"Demonstration Bank Nigeria"}`), Rationale: "The bank processes customer, employee and vendor personal data.", ApprovedBy: "user-demo", EffectiveFrom: now.AddDate(0, -6, 0)})
		if err != nil {
			return err
		}
		privacy, err = service.AddControlObjective(ctx, AddControlObjectiveInput{TenantID: "bank-demo", ProgramID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, Code: "PROCESSING-RECORDS", Name: "Current processing records", Outcome: "Processing records show active systems, purposes, data, recipients, retention and owners.", Status: ObjectiveActive, ActorID: "user-demo"})
		if err != nil {
			return err
		}
		privacy, err = service.AddControlImplementation(ctx, AddControlImplementationInput{TenantID: "bank-demo", ProgramID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, ObjectiveID: privacy.ControlObjectives[0].ID, Name: "Quarterly processing-owner review", Description: "Processing owners review changed activities and confirm unresolved facts each quarter.", ImplementationType: "OWNER_REVIEW", OwnerPrincipalID: "user-demo", Scope: json.RawMessage(`{"legal_entity":"Demonstration Bank Nigeria"}`), Status: ImplementationImplemented, EffectiveFrom: now.AddDate(0, -3, 0), ActorID: "user-demo"})
		if err != nil {
			return err
		}
		privacy, err = service.LinkRequirementControl(ctx, LinkRequirementControlInput{TenantID: "bank-demo", ProgramID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, RequirementID: requirement.ID, ImplementationID: privacy.ControlImplementations[0].ID, ActorID: "user-demo"})
		if err != nil {
			return err
		}
		privacy, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{TenantID: "bank-demo", ProgramID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, ControlImplementationID: privacy.ControlImplementations[0].ID, Code: "ROPA-COVERAGE", Name: "Processing record coverage", Claim: "Every active processing activity has a current owner-approved record.", PopulationScope: json.RawMessage(`{"population":"active_processing_activities"}`), FreshnessMinutes: 43200, MinimumCoverage: 0.95, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractActive, ActorID: "user-demo"})
		if err != nil {
			return err
		}
		validUntil := now.Add(14 * 24 * time.Hour)
		privacy, err = service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{TenantID: "bank-demo", ProgramID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, ContractID: privacy.EvidenceContracts[0].ID, Conclusion: EvidencePartiallySupported, Coverage: 0.89, Basis: json.RawMessage(`{"current_records":112,"expected_records":126}`), ValidUntil: &validUntil, AssessedBy: "user-demo", AssessedAt: now.Add(-2 * time.Hour)})
		if err != nil {
			return err
		}
		privacy, err = service.TransitionProgram(ctx, ProgramTransitionInput{TenantID: "bank-demo", ID: privacy.Program.ID, ExpectedVersion: privacy.Program.Version, To: ProgramActive, ActorID: "user-demo", Rationale: "Initial requirements, controls and evidence checks approved."})
		if err != nil {
			return err
		}
		_, err = service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank-demo", Type: MatterControlGap, Priority: 3, Title: "Complete missing retention details", Summary: "Fourteen processing records do not have an approved retention period.", Scope: json.RawMessage(`{"legal_entity":"Demonstration Bank Nigeria","population":14}`), KnownFacts: json.RawMessage(`{"missing_retention_records":14}`), MissingFacts: json.RawMessage(`["approved retention period","record owner confirmation"]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: "user-demo", DueAt: timePointer(now.Add(10 * 24 * time.Hour)), ProgramID: privacy.Program.ID, RequirementID: requirement.ID, ActorID: "user-demo"})
		if err != nil {
			return err
		}
	}

	cyber, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank-demo", Code: "CBN-CYBER", Name: "Cybersecurity controls", Type: "CYBERSECURITY", OwningFunction: "Information Security", OwnerPrincipalID: "user-demo", AuthorityPrincipalID: "user-demo", Jurisdiction: "Nigeria", Scope: json.RawMessage(`{"legal_entity":"Demonstration Bank Nigeria"}`), EffectiveFrom: now.AddDate(-1, 0, 0), ActorID: "user-demo"})
	if err != nil && err != ErrDuplicate {
		return err
	}
	if err == nil {
		cyber, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "bank-demo", ProgramID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, Code: "PRIV-ACCESS", Title: "Review privileged access", Statement: "Privileged access must be reviewed and supported by current business need.", SourceAnchor: "CBN cybersecurity controls", Modality: "MUST", Status: RequirementApproved, EffectiveFrom: now.AddDate(-1, 0, 0), ActorID: "user-demo"})
		if err != nil {
			return err
		}
		cyber, err = service.DetermineApplicability(ctx, DetermineApplicabilityInput{TenantID: "bank-demo", ProgramID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, RequirementID: cyber.Requirements[0].ID, Status: ApplicabilityApplicable, Scope: json.RawMessage(`{"systems":"privileged-access population"}`), Rationale: "The bank operates systems with privileged accounts.", ApprovedBy: "user-demo", EffectiveFrom: now.AddDate(-1, 0, 0)})
		if err != nil {
			return err
		}
		cyber, err = service.AddControlObjective(ctx, AddControlObjectiveInput{TenantID: "bank-demo", ProgramID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, Code: "ACCESS-JUSTIFIED", Name: "Justified privileged access", Outcome: "Every active privileged account has a current owner and approved business need.", Status: ObjectiveActive, ActorID: "user-demo"})
		if err != nil {
			return err
		}
		cyber, err = service.AddControlImplementation(ctx, AddControlImplementationInput{TenantID: "bank-demo", ProgramID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, ObjectiveID: cyber.ControlObjectives[0].ID, Name: "Monthly privileged-access review", Description: "IAM population is reconciled with HR and account-owner approvals each month.", ImplementationType: "ACCESS_REVIEW", OwnerPrincipalID: "user-demo", Scope: json.RawMessage(`{"systems":"privileged-access population"}`), Status: ImplementationImplemented, EffectiveFrom: now.AddDate(0, -6, 0), ActorID: "user-demo"})
		if err != nil {
			return err
		}
		cyber, err = service.LinkRequirementControl(ctx, LinkRequirementControlInput{TenantID: "bank-demo", ProgramID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, RequirementID: cyber.Requirements[0].ID, ImplementationID: cyber.ControlImplementations[0].ID, ActorID: "user-demo"})
		if err != nil {
			return err
		}
		cyber, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{TenantID: "bank-demo", ProgramID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, ControlImplementationID: cyber.ControlImplementations[0].ID, Code: "ACCESS-COVERAGE", Name: "Privileged-access review coverage", Claim: "Every privileged account is resolved for the current review period.", PopulationScope: json.RawMessage(`{"population":"privileged_accounts"}`), FreshnessMinutes: 44640, MinimumCoverage: 0.99, ContradictionPolicy: "FAIL", FailureAction: "MATTER", Status: EvidenceContractActive, ActorID: "user-demo"})
		if err != nil {
			return err
		}
		validUntil := now.Add(21 * 24 * time.Hour)
		cyber, err = service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{TenantID: "bank-demo", ProgramID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, ContractID: cyber.EvidenceContracts[0].ID, Conclusion: EvidenceSupported, Coverage: 1, Basis: json.RawMessage(`{"resolved":1250,"population":1250}`), ValidUntil: &validUntil, AssessedBy: "user-demo", AssessedAt: now.Add(-24 * time.Hour)})
		if err != nil {
			return err
		}
		_, err = service.TransitionProgram(ctx, ProgramTransitionInput{TenantID: "bank-demo", ID: cyber.Program.ID, ExpectedVersion: cyber.Program.Version, To: ProgramActive, ActorID: "user-demo", Rationale: "Initial requirements, controls and evidence checks approved."})
		if err != nil {
			return err
		}
	}
	return nil
}

func timePointer(value time.Time) *time.Time { return &value }
