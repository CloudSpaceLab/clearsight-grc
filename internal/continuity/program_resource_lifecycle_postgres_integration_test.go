//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresProgramResourceLifecycleCommitsProjectionEventAndOutboxTogether(t *testing.T) {
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

	const (
		tenantID = "94222222-2222-7222-8222-222222222221"
		entityID = "94222222-2222-7222-8222-222222222222"
		ownerID  = "94222222-2222-7222-8222-222222222223"
		otherID  = "94222222-2222-7222-8222-222222222224"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'program-resource-lifecycle-test','Program Resource Lifecycle Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'ENTITY-A','Entity A','NG')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$3::uuid,'PERSON','Program owner'),($2::uuid,$3::uuid,'PERSON','Safeguard owner')`, ownerID, otherID, tenantID); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewCurrentPostgresRepository(pool))
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "program-resource-lifecycle-test", LegalEntityID: entityID, Code: "LIFECYCLE", Name: "Lifecycle",
		Type: "ASSURANCE", OwningFunction: "Risk", OwnerPrincipalID: ownerID, EffectiveFrom: now, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, AddControlObjectiveInput{TenantID: "program-resource-lifecycle-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "OBJ", Name: "Objective", Outcome: "The outcome remains controlled.", Status: ObjectiveActive, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlImplementation(ctx, AddControlImplementationInput{TenantID: "program-resource-lifecycle-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ObjectiveID: program.ControlObjectives[0].ID, Name: "Safeguard", Description: "Operate the safeguard.", ImplementationType: "REVIEW", OwnerPrincipalID: ownerID, Scope: json.RawMessage(`{}`), EffectiveFrom: now, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	safeguard := program.ControlImplementations[0]
	program, err = service.AssignControlImplementation(ctx, AssignControlImplementationInput{TenantID: "program-resource-lifecycle-test", ProgramID: program.Program.ID, ImplementationID: safeguard.ID, ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: safeguard.Version, OwnerPrincipalID: otherID, Rationale: "The operating team changed.", ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.TransitionControlImplementation(ctx, TransitionControlImplementationInput{TenantID: "program-resource-lifecycle-test", ProgramID: program.Program.ID, ImplementationID: safeguard.ID, ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: program.ControlImplementations[0].Version, To: ImplementationInProgress, Rationale: "Work has started.", ActorID: otherID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{TenantID: "program-resource-lifecycle-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ControlImplementationID: safeguard.ID, Code: "CHECK", Name: "Evidence check", Claim: "Evidence remains current.", FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	contract := program.EvidenceContracts[0]
	program, err = service.ReviseEvidenceContract(ctx, ReviseEvidenceContractInput{TenantID: "program-resource-lifecycle-test", ProgramID: program.Program.ID, ContractID: contract.ID, ExpectedVersion: program.Program.Version, ExpectedContractVersion: contract.Version, Name: "Current evidence check", Claim: "Evidence remains current and complete.", PopulationScope: json.RawMessage(`{"population":"controls"}`), FreshnessMinutes: 1440, MinimumCoverage: .95, ContradictionPolicy: "FAIL", FailureAction: "MATTER", Rationale: "Align the check with the current process.", ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.TransitionEvidenceContract(ctx, TransitionEvidenceContractInput{TenantID: "program-resource-lifecycle-test", ProgramID: program.Program.ID, ContractID: contract.ID, ExpectedVersion: program.Program.Version, ExpectedContractVersion: program.EvidenceContracts[0].Version, To: EvidenceContractActive, Rationale: "The evidence rules are ready for use.", ActorID: otherID})
	if err != nil {
		t.Fatal(err)
	}

	var safeguardVersion, contractVersion, events, outbox int
	if err = pool.QueryRow(ctx, `SELECT version FROM control_implementations WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, safeguard.ID).Scan(&safeguardVersion); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT version FROM evidence_contracts WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, contract.ID).Scan(&contractVersion); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type IN ('CONTROL_IMPLEMENTATION_OWNER_CHANGED','CONTROL_IMPLEMENTATION_STATUS_CHANGED','EVIDENCE_CONTRACT_REVISED','EVIDENCE_CONTRACT_STATUS_CHANGED')`, tenantID, program.Program.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type IN ('CONTROL_IMPLEMENTATION_OWNER_CHANGED','CONTROL_IMPLEMENTATION_STATUS_CHANGED','EVIDENCE_CONTRACT_REVISED','EVIDENCE_CONTRACT_STATUS_CHANGED')`, tenantID, program.Program.ID).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if safeguardVersion != 3 || contractVersion != 3 || events != 4 || outbox != 4 {
		t.Fatalf("atomic lifecycle rows safeguard_version=%d contract_version=%d events=%d outbox=%d", safeguardVersion, contractVersion, events, outbox)
	}
}
