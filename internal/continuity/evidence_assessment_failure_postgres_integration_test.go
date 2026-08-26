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

func TestPostgresEvidenceFailureCommitsAssessmentMatterEventsAndOutboxTogether(t *testing.T) {
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
		tenantID   = "93333333-3333-7333-8333-333333333331"
		entityID   = "93333333-3333-7333-8333-333333333332"
		ownerID    = "93333333-3333-7333-8333-333333333333"
		reviewerID = "93333333-3333-7333-8333-333333333334"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'evidence-failure-test','Evidence Failure Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'ENTITY-A','Entity A','NG')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$3::uuid,'PERSON','Program owner'),($2::uuid,$3::uuid,'PERSON','Evidence reviewer')`, ownerID, reviewerID, tenantID); err != nil {
		t.Fatal(err)
	}

	repo := NewCurrentPostgresRepository(pool)
	service := NewService(repo)
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "evidence-failure-test", LegalEntityID: entityID, Code: "EVIDENCE", Name: "Evidence oversight",
		Type: "COMPLIANCE", OwningFunction: "Compliance", OwnerPrincipalID: ownerID, Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "evidence-failure-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "REQ", Title: "Retain evidence", Statement: "Evidence must be retained.", Status: RequirementApproved, EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{
		TenantID: "evidence-failure-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		RequirementID: program.Requirements[0].ID, Code: "CHECK", Name: "Retention evidence", Claim: "Required evidence is retained.",
		FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeVersion := program.Program.Version
	program, err = service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{
		TenantID: "evidence-failure-test", ProgramID: program.Program.ID, ExpectedVersion: beforeVersion,
		ContractID: program.EvidenceContracts[0].ID, Conclusion: EvidenceUnsupported, Coverage: .5,
		Basis: json.RawMessage(`{"missing":1}`), AssessedBy: reviewerID, AssessedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Version != beforeVersion+1 || len(program.EvidenceAssessments) != 1 {
		t.Fatalf("program after failed assessment = %#v", program)
	}
	var matters, links, assessmentEvents, matterEvents, outboxEvents, projectionJobs int
	queries := []struct {
		query  string
		args   []any
		target *int
	}{
		{`SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND legal_entity_id=$3::uuid`, []any{tenantID, program.EvidenceContracts[0].ID, entityID}, &matters},
		{`SELECT count(*) FROM matter_links ml JOIN matters m ON m.tenant_id=ml.tenant_id AND m.id=ml.matter_id WHERE ml.tenant_id=$1::uuid AND m.source_id=$2::uuid AND ml.program_id=$3::uuid`, []any{tenantID, program.EvidenceContracts[0].ID, program.Program.ID}, &links},
		{`SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND event_type='EVIDENCE_ASSESSMENT_RECORDED'`, []any{tenantID, program.Program.ID}, &assessmentEvents},
		{`SELECT count(*) FROM continuity_events ce JOIN matters m ON m.tenant_id=ce.tenant_id AND m.id=ce.aggregate_id WHERE ce.tenant_id=$1::uuid AND m.source_id=$2::uuid`, []any{tenantID, program.EvidenceContracts[0].ID}, &matterEvents},
		{`SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND ((aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND event_type='EVIDENCE_ASSESSMENT_RECORDED') OR (aggregate_type='MATTER' AND aggregate_id IN (SELECT id FROM matters WHERE tenant_id=$1::uuid AND source_id=$3::uuid)))`, []any{tenantID, program.Program.ID, program.EvidenceContracts[0].ID}, &outboxEvents},
		{`SELECT count(*) FROM continuity_projection_jobs WHERE tenant_id=$1::uuid AND aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND source_aggregate_version=$3`, []any{tenantID, program.Program.ID, program.Program.Version}, &projectionJobs},
	}
	for _, check := range queries {
		if err := pool.QueryRow(ctx, check.query, check.args...).Scan(check.target); err != nil {
			t.Fatal(err)
		}
	}
	if matters != 1 || links != 1 || assessmentEvents != 1 || matterEvents != 2 || outboxEvents != 3 || projectionJobs != 1 {
		t.Fatalf("atomic rows matters=%d links=%d assessment_events=%d matter_events=%d outbox=%d jobs=%d", matters, links, assessmentEvents, matterEvents, outboxEvents, projectionJobs)
	}
}

func TestPostgresLegacyEvidenceFailureActionsRemainExecutable(t *testing.T) {
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
		tenantID   = "94444444-4444-7444-8444-444444444441"
		entityID   = "94444444-4444-7444-8444-444444444442"
		ownerID    = "94444444-4444-7444-8444-444444444443"
		reviewerID = "94444444-4444-7444-8444-444444444444"
		requestID  = "94444444-4444-7444-8444-444444444445"
		flagID     = "94444444-4444-7444-8444-444444444446"
		blockID    = "94444444-4444-7444-8444-444444444447"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'legacy-evidence-failure-test','Legacy Evidence Failure Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'ENTITY-A','Entity A','NG')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$3::uuid,'PERSON','Program owner'),($2::uuid,$3::uuid,'PERSON','Evidence reviewer')`, ownerID, reviewerID, tenantID); err != nil {
		t.Fatal(err)
	}

	repo := NewCurrentPostgresRepository(pool)
	service := NewService(repo)
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "legacy-evidence-failure-test", LegalEntityID: entityID, Code: "LEGACY", Name: "Legacy evidence oversight",
		Type: "COMPLIANCE", OwningFunction: "Compliance", OwnerPrincipalID: ownerID, Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "legacy-evidence-failure-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "REQ", Title: "Retain evidence", Statement: "Evidence must be retained.", Status: RequirementApproved, EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	addLegacyContract := func(contractID, action string) {
		t.Helper()
		contract := EvidenceContract{
			ID: contractID, TenantID: "legacy-evidence-failure-test", ProgramID: program.Program.ID, RequirementID: program.Requirements[0].ID,
			Code: action, Name: action + " evidence", Claim: "Required evidence is retained.", PopulationScope: json.RawMessage(`{}`),
			FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW", FailureAction: action,
			Status: EvidenceContractActive, CreatedAt: now, UpdatedAt: now, Version: 1,
		}
		if err := service.applyProgramValue(ctx, "legacy-evidence-failure-test", program.Program.ID, program.Program.Version, EventEvidenceContractAdded, contract, ownerID); err != nil {
			t.Fatal(err)
		}
		program, err = service.GetProgram(ctx, "legacy-evidence-failure-test", program.Program.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	recordFailure := func(contractID string) {
		t.Helper()
		program, err = service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{
			TenantID: "legacy-evidence-failure-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
			ContractID: contractID, Conclusion: EvidenceUnsupported, Coverage: .4,
			Basis: json.RawMessage(`{"missing":1}`), AssessedBy: reviewerID, AssessedAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	addLegacyContract(requestID, "REQUEST")
	recordFailure(requestID)
	addLegacyContract(flagID, "FLAG")
	recordFailure(flagID)
	addLegacyContract(blockID, "BLOCK")
	recordFailure(blockID)

	var requests, flagMatters, blockMatters, legacyAssessments int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND legal_entity_id=$3::uuid AND matter_type='AUTHORITY_REQUEST' AND trigger_type='EVIDENCE_REQUEST_REQUIRED'`, tenantID, requestID, entityID).Scan(&requests); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_id=$2::uuid`, tenantID, flagID).Scan(&flagMatters); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_id=$2::uuid`, tenantID, blockID).Scan(&blockMatters); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM evidence_assessments WHERE tenant_id=$1::uuid AND contract_id IN ($2::uuid,$3::uuid,$4::uuid)`, tenantID, requestID, flagID, blockID).Scan(&legacyAssessments); err != nil {
		t.Fatal(err)
	}
	if requests != 1 || flagMatters != 0 || blockMatters != 0 || legacyAssessments != 3 {
		t.Fatalf("legacy actions requests=%d flag_matters=%d block_matters=%d assessments=%d", requests, flagMatters, blockMatters, legacyAssessments)
	}
}
