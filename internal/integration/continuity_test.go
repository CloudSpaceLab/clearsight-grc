//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	continuityTenantID      = "44444444-4444-7444-8444-444444444441"
	continuityOtherTenantID = "44444444-4444-7444-8444-444444444442"
	continuityActorID       = "44444444-4444-7444-8444-444444444443"
	continuityReviewerID    = "44444444-4444-7444-8444-444444444444"
	continuitySourceID      = "44444444-4444-7444-8444-444444444445"
)

func TestProgramMatterPostgresContracts(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'continuity-bank','Continuity Bank'),($2::uuid,'other-bank','Other Bank')`, continuityTenantID, continuityOtherTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($2::uuid,$1::uuid,'PERSON','owner','Program Owner'),($3::uuid,$1::uuid,'PERSON','reviewer','Independent Reviewer')`, continuityTenantID, continuityActorID, continuityReviewerID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,expected_freshness_minutes,health) VALUES($2::uuid,$1::uuid,'IAM','Identity directory','SYSTEM','SYSTEM_OF_RECORD',60,'CURRENT')`, continuityTenantID, continuitySourceID); err != nil {
		t.Fatal(err)
	}

	service := continuity.NewService(continuity.NewPostgresRepository(pool))
	now := time.Now().UTC().Truncate(time.Second)

	program, err := service.CreateProgram(ctx, continuity.CreateProgramInput{TenantID: "continuity-bank", Code: "ACCESS", Name: "Privileged access", Type: "CYBERSECURITY", OwningFunction: "Information Security", OwnerPrincipalID: continuityActorID, AuthorityPrincipalID: continuityActorID, Scope: json.RawMessage(`{"population":"privileged_accounts"}`), EffectiveFrom: now.AddDate(0, -6, 0), ActorID: continuityActorID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, continuity.AddRequirementInput{TenantID: "continuity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, SourceID: continuitySourceID, Code: "OWNER-APPROVAL", Title: "Keep privileged access approved", Statement: "Every privileged account must have a current owner and business need.", Modality: "MUST", Status: continuity.RequirementApproved, EffectiveFrom: now.AddDate(0, -6, 0), ActorID: continuityActorID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.DetermineApplicability(ctx, continuity.DetermineApplicabilityInput{TenantID: "continuity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: program.Requirements[0].ID, Status: continuity.ApplicabilityApplicable, Scope: json.RawMessage(`{"population":"privileged_accounts"}`), Rationale: "The bank operates privileged accounts.", ApprovedBy: continuityActorID, EffectiveFrom: now.AddDate(0, -6, 0)})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, continuity.AddControlObjectiveInput{TenantID: "continuity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "ACCESS-JUSTIFIED", Name: "Justified access", Outcome: "Every active privileged account is approved and attributable.", Status: continuity.ObjectiveActive, ActorID: continuityActorID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlImplementation(ctx, continuity.AddControlImplementationInput{TenantID: "continuity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ObjectiveID: program.ControlObjectives[0].ID, Name: "Monthly account review", Description: "IAM accounts are reconciled with HR and owner approvals.", ImplementationType: "ACCESS_REVIEW", OwnerPrincipalID: continuityActorID, Scope: json.RawMessage(`{"population":"privileged_accounts"}`), Status: continuity.ImplementationImplemented, EffectiveFrom: now.AddDate(0, -3, 0), ActorID: continuityActorID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.LinkRequirementControl(ctx, continuity.LinkRequirementControlInput{TenantID: "continuity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: program.Requirements[0].ID, ImplementationID: program.ControlImplementations[0].ID, ActorID: continuityActorID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, continuity.AddEvidenceContractInput{TenantID: "continuity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ControlImplementationID: program.ControlImplementations[0].ID, Code: "ACCESS-COVERAGE", Name: "Access review coverage", Claim: "Every privileged account is resolved for the current period.", AcceptableSourceIDs: []string{continuitySourceID}, PopulationScope: json.RawMessage(`{"population":"privileged_accounts"}`), FreshnessMinutes: 44640, MinimumCoverage: .99, ContradictionPolicy: "FAIL", FailureAction: "MATTER", Status: continuity.EvidenceContractActive, ActorID: continuityActorID})
	if err != nil {
		t.Fatal(err)
	}
	validUntil := now.Add(30 * 24 * time.Hour)
	program, err = service.RecordEvidenceAssessment(ctx, continuity.RecordEvidenceAssessmentInput{TenantID: "continuity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ContractID: program.EvidenceContracts[0].ID, Conclusion: continuity.EvidenceSupported, Coverage: 1, Basis: json.RawMessage(`{"resolved":1250,"population":1250}`), ValidUntil: &validUntil, AssessedBy: continuityReviewerID, AssessedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.TransitionProgram(ctx, continuity.ProgramTransitionInput{TenantID: "continuity-bank", ID: program.Program.ID, ExpectedVersion: program.Program.Version, To: continuity.ProgramActive, ActorID: continuityReviewerID, Rationale: "Initial setup approved."})
	if err != nil {
		t.Fatal(err)
	}
	maintainer := &continuity.ProjectionMaintainer{Service: service, Repo: continuity.NewPostgresRepository(pool), WorkerID: "continuity-test-worker"}
	if completed, err := maintainer.Maintain(ctx, time.Now().UTC().Add(time.Second), 20); err != nil || completed != 1 {
		t.Fatalf("Program status update failed completed=%d err=%v", completed, err)
	}
	program, err = service.GetProgram(ctx, "continuity-bank", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if program.CurrentState == nil || program.CurrentState.Overall != continuity.StateCurrent || program.StateLabel != "Up to date" || program.CurrentState.ProgramVersion != program.Program.Version {
		t.Fatalf("unexpected current state %#v", program.CurrentState)
	}

	checkpoint := time.Now().UTC()
	trigger := continuity.Trigger{TenantID: "continuity-bank", ProgramID: program.Program.ID, Type: "CONTROL_FAILED", SubjectType: "CONTROL_IMPLEMENTATION", SubjectID: program.ControlImplementations[0].ID, DedupeKey: "access-review-failure-2026-08", Payload: json.RawMessage(`{"unresolved_accounts":4}`), ObservedAt: now.Add(2 * time.Minute), Source: "access-review"}
	program, created, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted || created == nil || created.Type != continuity.MatterControlGap {
		t.Fatalf("unexpected trigger result inserted=%v matter=%#v", inserted, created)
	}
	if completed, err := maintainer.Maintain(ctx, time.Now().UTC().Add(2*time.Second), 20); err != nil || completed != 1 {
		t.Fatalf("trigger status update failed completed=%d err=%v", completed, err)
	}
	program, err = service.GetProgram(ctx, "continuity-bank", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if program.CurrentState == nil || program.CurrentState.OpenMatterCount != 1 || program.CurrentState.ProgramVersion != program.Program.Version {
		t.Fatalf("unexpected trigger status %#v", program.CurrentState)
	}
	_, duplicate, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil || inserted || duplicate == nil || duplicate.ID != created.ID {
		t.Fatalf("trigger idempotency failed inserted=%v duplicate=%#v err=%v", inserted, duplicate, err)
	}
	var matterCount, triggerCount, eventCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid`, continuityTenantID).Scan(&matterCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM program_trigger_events WHERE tenant_id=$1::uuid`, continuityTenantID).Scan(&triggerCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid`, continuityTenantID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if matterCount != 1 || triggerCount != 1 || eventCount < 10 {
		t.Fatalf("unexpected durable counts matters=%d triggers=%d events=%d", matterCount, triggerCount, eventCount)
	}
	historical, err := service.ProgramAt(ctx, "continuity-bank", program.Program.ID, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if historical.CurrentState == nil || historical.CurrentState.OpenMatterCount != 0 || len(historical.Triggers) != 0 {
		t.Fatalf("historical program includes later trigger %#v", historical)
	}

	matter, err := service.GetMatter(ctx, "continuity-bank", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: "continuity-bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterAssessment, ActorID: continuityActorID, Rationale: "Scope and affected accounts confirmed."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: "continuity-bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterActionsInProgress, ActorID: continuityActorID, Rationale: "Remediation is required."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(ctx, continuity.AddActionInput{TenantID: "continuity-bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: "Confirm the four account owners", Description: "Obtain current owner approval or remove access.", OwnerPrincipalID: continuityActorID, ActorID: continuityActorID})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID
	matter, err = service.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: "continuity-bank", MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionInProgress, ActorID: continuityActorID, Rationale: "Owner review started."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: "continuity-bank", MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionImplemented, ActorID: continuityActorID, Rationale: "Access removed or approved."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, continuity.AddVerificationContractInput{TenantID: "continuity-bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: actionID, ExpectedOutcome: "No privileged account lacks current approval.", Baseline: json.RawMessage(`{"unresolved":4}`), Scope: json.RawMessage(`{"accounts":4}`), MeasurementSourceID: continuitySourceID, Threshold: json.RawMessage(`{"unresolved":0}`), FailureResponse: "REOPEN", AuthorityPrincipalID: continuityReviewerID, ActorID: continuityReviewerID})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: "continuity-bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterVerification, ActorID: continuityReviewerID, Rationale: "Check the account population after remediation."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: "continuity-bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterClosed, ActorID: continuityReviewerID, Rationale: "Close."}); !errors.Is(err, continuity.ErrClosureBlocked) {
		t.Fatalf("expected closure block, got %v", err)
	}
	matter, err = service.RecordVerificationResult(ctx, continuity.RecordVerificationResultInput{TenantID: "continuity-bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ContractID: matter.VerificationContracts[0].ID, Result: continuity.VerificationPassed, Observations: json.RawMessage(`{"unresolved":0}`), EvidenceReferences: json.RawMessage(`[]`), ReviewerPrincipalID: continuityReviewerID, Rationale: "The current IAM population has no unresolved accounts."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: "continuity-bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: continuity.MatterClosed, ActorID: continuityReviewerID, Rationale: "The defined outcome passed."})
	if err != nil || matter.Matter.Status != continuity.MatterClosed {
		t.Fatalf("close failed matter=%#v err=%v", matter.Matter, err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO matter_links(tenant_id,matter_id,program_id,relationship) VALUES($1::uuid,$2::uuid,$3::uuid,'AFFECTS')`, continuityOtherTenantID, matter.Matter.ID, program.Program.ID); err == nil {
		t.Fatal("expected cross-tenant matter link to be rejected")
	}
}
