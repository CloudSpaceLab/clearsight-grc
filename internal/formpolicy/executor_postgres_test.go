//go:build postgres && postgresintegration

package formpolicy

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresExecutionAppliesMatterReceiptEpisodeEventAndInboxAtomically(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cleanupPolicyFixture(t, pool)
	t.Cleanup(func() { cleanupPolicyFixture(t, pool) })
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedPolicyFixture(t, ctx, pool, now)
	repo := NewPostgresRepository(pool)
	policy := postgresPolicyFixture(now)
	policy.Status, policy.Rollout = PolicyActive, RolloutEnforce
	policy.Checksum = policyChecksum(policy)
	if _, err := repo.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}

	failed := postgresExecutionCommand(now, policy)
	failed.Route.OwnerPrincipalID = "00000000-0000-0000-0000-000000000099"
	if _, err := repo.ApplyExecution(ctx, failed); err == nil {
		t.Fatal("expected Matter owner foreign-key failure")
	}
	assertExecutionBundleCounts(t, ctx, pool, 0, 0, 0, 0, 0, 0, 0, 0)

	command := postgresExecutionCommand(now, policy)
	first, err := repo.ApplyExecution(ctx, command)
	if err != nil || first.State != ExecutionApplied || !first.CreatedMatter || first.MatterID != command.Matter.ID {
		t.Fatalf("first execution = %#v err=%v", first, err)
	}
	assertExecutionBundleCounts(t, ctx, pool, 1, 1, 1, 2, 2, 1, 1, 1)

	replay := command
	replay.Receipt.ID = "9f650000-0000-7650-8650-000000000021"
	replay.Episode.ID = "9f650000-0000-7650-8650-000000000022"
	replay.Matter.ID = "9f650000-0000-7650-8650-000000000023"
	replay.Outcome.ID = "9f650000-0000-7650-8650-000000000024"
	replay.Episode.MatterID = replay.Matter.ID
	replay.Outcome.MatterID = replay.Matter.ID
	replay.Matter.TriggerID = replay.Receipt.ID
	stored, err := repo.ApplyExecution(ctx, replay)
	if err != nil || stored.ID != command.Receipt.ID || stored.MatterID != command.Matter.ID {
		t.Fatalf("replayed execution = %#v err=%v", stored, err)
	}
	assertExecutionBundleCounts(t, ctx, pool, 1, 1, 1, 2, 2, 1, 1, 1)
}

func TestPostgresRollbackCompensationIsAtomicIdempotentAndDoesNotCloseMatter(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cleanupPolicyFixture(t, pool)
	t.Cleanup(func() { cleanupPolicyFixture(t, pool) })
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedPolicyFixture(t, ctx, pool, now)
	repo := NewPostgresRepository(pool)
	policy := postgresPolicyFixture(now)
	policy.Status, policy.Rollout = PolicyActive, RolloutEnforce
	policy.BlastRadius.PerRun, policy.BlastRadius.PerDay = 10, 10
	policy.Checksum = policyChecksum(policy)
	if _, err := repo.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	executionCommand := postgresExecutionCommand(now, policy)
	applied, err := repo.ApplyExecution(ctx, executionCommand)
	if err != nil || !applied.CreatedMatter {
		t.Fatalf("applied=%#v err=%v", applied, err)
	}
	seedDistinctSubjectPolicyResponse(t, ctx, pool, now.Add(time.Second))
	secondApplied, err := repo.ApplyExecution(ctx, postgresDistinctSubjectExecutionCommand(now.Add(time.Second), policy))
	if err != nil || !secondApplied.CreatedMatter || secondApplied.MatterID == applied.MatterID {
		t.Fatalf("second applied=%#v err=%v", secondApplied, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE form_response_policy_definitions SET status='SUSPENDED',suspended_at=$2,record_version=record_version+1,updated_at=$2 WHERE id=$1::uuid`, policy.ID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	rollback := policy
	rollback.ID, rollback.Version, rollback.RecordVersion = "9f650000-0000-7650-8650-000000000030", 2, 1
	rollback.RollbackOfPolicyID, rollback.SupersedesPolicyID = policy.ID, policy.ID
	rollback.Status, rollback.ActivatedAt, rollback.SuspendedAt = PolicyActive, ptrTime(now.Add(time.Minute)), nil
	rollback.CreatedAt, rollback.UpdatedAt = now.Add(time.Minute), now.Add(time.Minute)
	rollback.Checksum = policyChecksum(rollback)
	if _, err := repo.CreatePolicy(ctx, rollback); err != nil {
		t.Fatal(err)
	}
	candidates, err := repo.ListPendingCompensations(ctx, now.Add(2*time.Minute), 10)
	if err != nil || len(candidates) != 2 || candidates[0].OriginalExecution.ID != applied.ID || candidates[1].OriginalExecution.ID != secondApplied.ID {
		t.Fatalf("candidates=%#v err=%v", candidates, err)
	}
	maintenanceNow := time.Now().UTC()
	if maintenanceNow.Before(now.Add(3 * time.Minute)) {
		maintenanceNow = now.Add(3 * time.Minute)
	}
	if seeded, seedErr := repo.SeedCompensations(ctx, maintenanceNow, 1); seedErr != nil || seeded != 1 {
		t.Fatalf("first seed=%d err=%v", seeded, seedErr)
	}
	claimed, err := repo.ClaimCompensations(ctx, "compensation-worker", maintenanceNow, time.Minute, 1)
	if err != nil || len(claimed) != 1 || claimed[0].OriginalExecution.ID != applied.ID {
		t.Fatalf("claimed=%#v err=%v", claimed, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs SET attempts=5 WHERE id=$1::uuid`, claimed[0].JobID); err != nil {
		t.Fatal(err)
	}
	if err := repo.RetryCompensation(ctx, claimed[0].JobID, "compensation-worker", maintenanceNow, ErrAuthorityUnavailable.Error()); err != nil {
		t.Fatal(err)
	}
	var retryState string
	var retryDueAt time.Time
	if err := pool.QueryRow(ctx, `SELECT state,due_at FROM form_response_policy_maintenance_jobs WHERE id=$1::uuid`, claimed[0].JobID).Scan(&retryState, &retryDueAt); err != nil {
		t.Fatal(err)
	}
	if retryState != "READY" || !retryDueAt.After(maintenanceNow) {
		t.Fatalf("retry state=%s due=%s now=%s", retryState, retryDueAt, maintenanceNow)
	}
	if seeded, seedErr := repo.SeedCompensations(ctx, maintenanceNow, 1); seedErr != nil || seeded != 1 {
		t.Fatalf("second seed behind existing job=%d err=%v", seeded, seedErr)
	}
	var compensationJobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM form_response_policy_maintenance_jobs WHERE tenant_id=$1::uuid AND job_type='COMPENSATION'`, policyTenantID).Scan(&compensationJobs); err != nil || compensationJobs != 2 {
		t.Fatalf("compensation jobs=%d err=%v", compensationJobs, err)
	}
	compensation := CompensationCommand{
		Candidate: candidates[0], Response: executionCommand.Response, Route: executionCommand.Route,
		Receipt: CompensationReceipt{ID: "9f650000-0000-7650-8650-000000000031", TenantID: policyTenantID, LegalEntityID: policyEntityID,
			RollbackPolicyID: rollback.ID, RollbackPolicyVersion: rollback.Version, OriginalExecutionID: applied.ID,
			MatterID: applied.MatterID, ReviewMatterID: "9f650000-0000-7650-8650-000000000033", ActorID: policyCheckerID, ReviewerPrincipalID: policyCheckerID,
			State: CompensationReviewRequired, ReasonCode: "ROLLBACK_POLICY_ACTIVE", CreatedAt: now.Add(2 * time.Minute)},
		ReviewMatter: continuity.Matter{ID: "9f650000-0000-7650-8650-000000000033", TenantID: policyTenantID, LegalEntityID: policyEntityID, Reference: "MAT-ROLLBACK000033", Type: continuity.MatterException, Status: continuity.MatterInitialReview, Priority: 4, Title: "Review form policy rollback impact", Summary: "Confirm whether the original policy-created issue remains valid after the active rollback revision.", Scope: mustExecutionJSON(map[string]any{"original_matter_id": applied.MatterID}), SourceType: "FORM_RESPONSE_POLICY_COMPENSATION", SourceID: applied.ID, TriggerType: "FORM_RESPONSE_POLICY_COMPENSATION_REVIEW_REQUIRED", TriggerID: "9f650000-0000-7650-8650-000000000031", TriggerKey: "form-response-policy-compensation:" + rollback.ID + ":" + applied.ID, KnownFacts: mustExecutionJSON(map[string]any{"original_execution_id": applied.ID}), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: policyMakerID, RequiredAuthority: "ACCOUNTABLE_OWNER", CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute), Version: 1},
		ReviewAction: continuity.Action{ID: "9f650000-0000-7650-8650-000000000034", TenantID: policyTenantID, MatterID: "9f650000-0000-7650-8650-000000000033", OriginKey: "form-response-policy-compensation-review", Title: "Review rollback impact", Description: "Review the original execution receipt and Matter, then record whether the issue remains valid, needs correction or can proceed unchanged.", OwnerPrincipalID: policyCheckerID, RequiredResponsibility: "REVIEWER", Status: continuity.ActionPlanned, CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute), Version: 1},
	}
	first, err := repo.ApplyCompensation(ctx, compensation)
	if err != nil || first.ID != compensation.Receipt.ID {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	var count, version, compensationOutbox, reviewVersion, reviewActions int
	var status continuity.MatterStatus
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM form_response_policy_compensations WHERE rollback_policy_id=$1::uuid),
		(SELECT version FROM matters WHERE id=$2::uuid),
		(SELECT status FROM matters WHERE id=$2::uuid),
		(SELECT count(*) FROM outbox_events WHERE aggregate_type='FORM_RESPONSE_POLICY_COMPENSATION' AND aggregate_id=$3::uuid),
		(SELECT version FROM matters WHERE id=$4::uuid),
		(SELECT count(*) FROM matter_actions WHERE matter_id=$4::uuid AND owner_principal_id=$5::uuid AND required_responsibility='REVIEWER')`, rollback.ID, applied.MatterID, first.ID, first.ReviewMatterID, policyCheckerID).Scan(&count, &version, &status, &compensationOutbox, &reviewVersion, &reviewActions); err != nil {
		t.Fatal(err)
	}
	if count != 1 || version != 3 || status != continuity.MatterInitialReview || compensationOutbox != 1 || reviewVersion != 2 || reviewActions != 1 || first.ReviewerPrincipalID != policyCheckerID {
		t.Fatalf("count=%d version=%d status=%s outbox=%d reviewVersion=%d reviewActions=%d receipt=%#v", count, version, status, compensationOutbox, reviewVersion, reviewActions, first)
	}
	compensation.Receipt.ID = "9f650000-0000-7650-8650-000000000032"
	replayed, err := repo.ApplyCompensation(ctx, compensation)
	if err != nil || replayed.ID != first.ID {
		t.Fatalf("replayed=%#v err=%v", replayed, err)
	}
}

func TestPostgresConcurrentAdverseResponsesCreateOneMatterAndReuseItsEpisode(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cleanupPolicyFixture(t, pool)
	t.Cleanup(func() { cleanupPolicyFixture(t, pool) })
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedPolicyFixture(t, ctx, pool, now)
	seedSecondPolicyResponseRevision(t, ctx, pool, now.Add(time.Second))
	repo := NewPostgresRepository(pool)
	policy := postgresPolicyFixture(now)
	policy.Status, policy.Rollout, policy.ActivatedAt = PolicyActive, RolloutEnforce, ptrTime(now.Add(-time.Minute))
	policy.Checksum = policyChecksum(policy)
	if _, err := repo.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	first := postgresExecutionCommand(now, policy)
	second := postgresExecutionCommand(now.Add(time.Second), policy)
	second.Response.ID, second.Receipt.ResponseRevisionID, second.Episode.LastResponseRevisionID, second.Matter.SourceID = "ddb4fe49-9070-3aa1-4335-58dc7bdaeed3", "ddb4fe49-9070-3aa1-4335-58dc7bdaeed3", "ddb4fe49-9070-3aa1-4335-58dc7bdaeed3", "ddb4fe49-9070-3aa1-4335-58dc7bdaeed3"
	second.Receipt.ID, second.Episode.ID, second.Matter.ID, second.Outcome.ID = "9f650000-0000-7650-8650-000000000040", "9f650000-0000-7650-8650-000000000041", "9f650000-0000-7650-8650-000000000042", "9f650000-0000-7650-8650-000000000043"
	second.Matter.TriggerID, second.Outcome.MatterID, second.Episode.MatterID = second.Receipt.ID, second.Matter.ID, second.Matter.ID
	second.Matter.TriggerKey = "form-response-policy:poor-vendor-response:vendor:" + second.Route.CanonicalSubjectID + ":" + second.Episode.ID
	commands := []ExecutionCommand{first, second}
	receipts := make([]ExecutionReceipt, 2)
	errs := make([]error, 2)
	var group sync.WaitGroup
	for index := range commands {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			receipts[index], errs[index] = repo.ApplyExecution(ctx, commands[index])
		}(index)
	}
	group.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("errors=%v receipts=%#v", errs, receipts)
	}
	created, reused := 0, 0
	for _, receipt := range receipts {
		if receipt.CreatedMatter {
			created++
		} else if receipt.State == ExecutionReused {
			reused++
		}
	}
	if created != 1 || reused != 1 || receipts[0].MatterID != receipts[1].MatterID {
		t.Fatalf("receipts=%#v", receipts)
	}
	var matters, episodes int
	var matterSourceID, episodeResponseID string
	var matterUpdatedAt, episodeUpdatedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_type='FORM_RESPONSE'),
		(SELECT count(*) FROM form_response_policy_adverse_episodes WHERE tenant_id=$1::uuid AND state='OPEN'),
		m.source_id::text,m.updated_at,e.last_response_revision_id::text,e.updated_at
		FROM matters m JOIN form_response_policy_adverse_episodes e ON e.matter_id=m.id AND e.tenant_id=m.tenant_id
		WHERE m.tenant_id=$1::uuid AND m.source_type='FORM_RESPONSE' AND e.state='OPEN'`, policyTenantID).Scan(&matters, &episodes, &matterSourceID, &matterUpdatedAt, &episodeResponseID, &episodeUpdatedAt); err != nil || matters != 1 || episodes != 1 {
		t.Fatalf("matters=%d episodes=%d err=%v", matters, episodes, err)
	}
	if matterSourceID != second.Response.ID || episodeResponseID != second.Response.ID || !matterUpdatedAt.Equal(second.Receipt.CreatedAt) || !episodeUpdatedAt.Equal(second.Receipt.CreatedAt) {
		t.Fatalf("latest response regressed: matter=%s/%s episode=%s/%s want=%s/%s", matterSourceID, matterUpdatedAt, episodeResponseID, episodeUpdatedAt, second.Response.ID, second.Receipt.CreatedAt)
	}
}

func TestPostgresConcurrentDistinctSubjectsRespectPerRunBlastRadius(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cleanupPolicyFixture(t, pool)
	t.Cleanup(func() { cleanupPolicyFixture(t, pool) })
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedPolicyFixture(t, ctx, pool, now)
	seedDistinctSubjectPolicyResponse(t, ctx, pool, now)
	repo := NewPostgresRepository(pool)
	policy := postgresPolicyFixture(now)
	policy.Status, policy.Rollout, policy.ActivatedAt = PolicyActive, RolloutEnforce, ptrTime(now.Add(-time.Minute))
	policy.BlastRadius.PerRun, policy.BlastRadius.PerDay = 1, 10
	policy.Checksum = policyChecksum(policy)
	if _, err := repo.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	first := postgresExecutionCommand(now, policy)
	second := postgresDistinctSubjectExecutionCommand(now, policy)
	commands := []ExecutionCommand{first, second}
	receipts := make([]ExecutionReceipt, len(commands))
	errs := make([]error, len(commands))
	var group sync.WaitGroup
	for index := range commands {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			receipts[index], errs[index] = repo.ApplyExecution(ctx, commands[index])
		}(index)
	}
	group.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("errors=%v receipts=%#v", errs, receipts)
	}
	created, suppressed := 0, 0
	for _, receipt := range receipts {
		if receipt.CreatedMatter && receipt.State == ExecutionApplied {
			created++
		}
		if !receipt.CreatedMatter && receipt.State == ExecutionBlastSuppressed && receipt.ReasonCode == "PER_RUN_LIMIT" {
			suppressed++
		}
	}
	if created != 1 || suppressed != 1 {
		t.Fatalf("distinct-subject receipts=%#v", receipts)
	}
	var executions, matters, episodes int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM form_response_policy_executions WHERE tenant_id=$1::uuid AND policy_id=$2::uuid),
		(SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_type='FORM_RESPONSE'),
		(SELECT count(*) FROM form_response_policy_adverse_episodes WHERE tenant_id=$1::uuid AND state='OPEN')`, policyTenantID, policy.ID).Scan(&executions, &matters, &episodes); err != nil {
		t.Fatal(err)
	}
	if executions != 2 || matters != 1 || episodes != 1 {
		t.Fatalf("executions=%d matters=%d episodes=%d", executions, matters, episodes)
	}
}

func TestPostgresAuthorityFailureKeepsAttemptHistoryCreatesExceptionAndCanRecover(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cleanupPolicyFixture(t, pool)
	t.Cleanup(func() { cleanupPolicyFixture(t, pool) })
	now := time.Now().UTC()
	seedPolicyFixture(t, ctx, pool, now)
	repo := NewPostgresRepository(pool)
	policy := postgresPolicyFixture(now)
	policy.Status, policy.Rollout, policy.ActivatedAt = PolicyActive, RolloutEnforce, ptrTime(now)
	policy.Checksum = policyChecksum(policy)
	if _, err := repo.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	failed := postgresExecutionCommand(now, policy)
	failed.Receipt.ID, failed.Receipt.State, failed.Receipt.ReasonCode = "9f650000-0000-7650-8650-000000000048", ExecutionFailed, "AUTHORITY_ROUTE_INVALID"
	failed.Episode, failed.Matter, failed.Outcome = AdverseEpisode{}, continuity.Matter{}, continuity.VerificationContract{}
	failed.FailureMatter = &continuity.Matter{ID: "9f650000-0000-7650-8650-000000000049", TenantID: policyTenantID, LegalEntityID: policyEntityID, Reference: "MAT-ROUTING000049", Type: continuity.MatterException, Status: continuity.MatterInitialReview, Priority: 4, Title: "Repair form response policy routing", Summary: "The current route is invalid.", Scope: json.RawMessage(`{"policy_id":"9f650000-0000-7650-8650-000000000009"}`), SourceType: "FORM_RESPONSE_POLICY_EXECUTION", SourceID: failed.Receipt.ID, TriggerType: "FORM_RESPONSE_POLICY_EXECUTION_FAILED", TriggerID: failed.Receipt.ID, TriggerKey: "form-response-policy-execution-failure:" + policy.ID + ":" + failed.Response.ID, KnownFacts: json.RawMessage(`{"reason_code":"AUTHORITY_ROUTE_INVALID"}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: policyCheckerID, RequiredAuthority: "ESCALATION_OWNER", CreatedAt: now, UpdatedAt: now, Version: 1}
	failed.FailureAction = &continuity.Action{ID: "9f650000-0000-7650-8650-000000000050", TenantID: policyTenantID, MatterID: failed.FailureMatter.ID, OriginKey: "form-response-policy-execution-recovery", Title: "Restore the authority route", Description: "Repair the route and retry the response.", OwnerPrincipalID: policyCheckerID, RequiredResponsibility: "ESCALATION_OWNER", Status: continuity.ActionPlanned, CreatedAt: now, UpdatedAt: now, Version: 1}
	receipt, err := repo.ApplyExecution(ctx, failed)
	if err != nil || receipt.State != ExecutionFailed {
		t.Fatalf("failure receipt=%#v err=%v", receipt, err)
	}
	var failures, executions, exceptions, actions, jobs int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM form_response_policy_execution_failures WHERE tenant_id=$1::uuid),
		(SELECT count(*) FROM form_response_policy_executions WHERE tenant_id=$1::uuid),
		(SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_type='FORM_RESPONSE_POLICY_EXECUTION'),
		(SELECT count(*) FROM matter_actions WHERE tenant_id=$1::uuid AND origin_key='form-response-policy-execution-recovery'),
		(SELECT count(*) FROM form_response_policy_maintenance_jobs WHERE tenant_id=$1::uuid AND job_type='RECONCILE')`, policyTenantID).Scan(&failures, &executions, &exceptions, &actions, &jobs); err != nil {
		t.Fatal(err)
	}
	if failures != 1 || executions != 0 || exceptions != 1 || actions != 1 || jobs != 1 {
		t.Fatalf("failures=%d executions=%d exceptions=%d actions=%d jobs=%d", failures, executions, exceptions, actions, jobs)
	}
	applied, err := repo.ApplyExecution(ctx, postgresExecutionCommand(now.Add(time.Minute), policy))
	if err != nil || applied.State != ExecutionApplied || !applied.CreatedMatter {
		t.Fatalf("recovered=%#v err=%v", applied, err)
	}
	var recoveryMatterStatus continuity.MatterStatus
	var recoveryMatterVersion, recoveryActionVersion, recoveryEvents int
	var recoveryExecutionID, recoveryActionStatus, recoveryActionDescription string
	var recoveryImplementedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT m.status,m.version,COALESCE(m.known_facts->>'route_recovery_execution_id',''),a.status,a.version,a.implemented_at,a.description,
		(SELECT count(*) FROM continuity_events ce WHERE ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id)
		FROM matters m JOIN matter_actions a ON a.tenant_id=m.tenant_id AND a.matter_id=m.id AND a.origin_key='form-response-policy-execution-recovery'
		WHERE m.tenant_id=$1::uuid AND m.source_type='FORM_RESPONSE_POLICY_EXECUTION'`, policyTenantID).Scan(&recoveryMatterStatus, &recoveryMatterVersion, &recoveryExecutionID, &recoveryActionStatus, &recoveryActionVersion, &recoveryImplementedAt, &recoveryActionDescription, &recoveryEvents); err != nil {
		t.Fatal(err)
	}
	if recoveryMatterStatus != continuity.MatterInitialReview || recoveryMatterVersion != 5 || recoveryExecutionID != applied.ID || recoveryActionStatus != string(continuity.ActionImplemented) || recoveryActionVersion != 3 || recoveryImplementedAt == nil || !strings.Contains(recoveryActionDescription, applied.ID) || recoveryEvents != 5 {
		t.Fatalf("recovery matter status=%s version=%d execution=%s action=%s/%d implemented=%v description=%q events=%d", recoveryMatterStatus, recoveryMatterVersion, recoveryExecutionID, recoveryActionStatus, recoveryActionVersion, recoveryImplementedAt, recoveryActionDescription, recoveryEvents)
	}
}

func seedSecondPolicyResponseRevision(t *testing.T, ctx context.Context, pool *pgxpool.Pool, createdAt time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		UPDATE capture_response_revisions SET is_current=false WHERE id=md5('form-policy:response:1')::uuid;
		INSERT INTO capture_response_revisions(id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,supersedes_revision_id,achieved_assurance,signoff_summary,compliance_score,scored_weight_coverage,state,critical_field_results,scoring_policy_version,is_current,created_at,score_mode,score_direction,raw_score,adverse_score,concern_band,score_state,score_result,score_profile_checksum,score_calculated_at)
		SELECT 'ddb4fe49-9070-3aa1-4335-58dc7bdaeed3'::uuid,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,2,id,achieved_assurance,signoff_summary,18,100,state,critical_field_results,scoring_policy_version,true,$1,score_mode,score_direction,18,82,'HIGH','FINAL',score_result,score_profile_checksum,$1
		FROM capture_response_revisions WHERE id=md5('form-policy:response:1')::uuid`, pgx.QueryExecModeSimpleProtocol, createdAt); err != nil {
		t.Fatal(err)
	}
}

func seedDistinctSubjectPolicyResponse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, createdAt time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_form_distributions(id,tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,title,purpose,access_policy,status,deadline,route_expires_at,created_by,version,created_at,updated_at)
		VALUES('9f650000-0000-7650-8650-000000000052'::uuid,$1::uuid,$2::uuid,$4::uuid,1,'VENDOR','9f650000-0000-7650-8650-000000000051'::uuid,'Second vendor response','Review the second completed response.','DIRECT_MAGIC_LINK','COMPLETED',$5::timestamptz+interval '30 days',$5::timestamptz+interval '7 days',$3::uuid,1,$5,$5);
		INSERT INTO capture_requests(id,tenant_id,legal_entity_id,distribution_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,known_facts,presentation,scoring_mode,sections,fields,source_bindings,form_template_id,form_template_version,status,created_by,version,created_at,updated_at)
		VALUES('9f650000-0000-7650-8650-000000000053'::uuid,$1::uuid,$2::uuid,'9f650000-0000-7650-8650-000000000052'::uuid,'VENDOR','9f650000-0000-7650-8650-000000000051'::uuid,'Second vendor request','Review the response.','Review the response.','INTERNAL','INTERNAL',5,$5::timestamptz+interval '30 days','{}','{"default_mode":"CLASSIC","allow_mode_switch":false}','COMPLIANCE','[{"id":"general","title":"Questions"}]','[{"id":"q1","section_id":"general","label":"Question","type":"short_text","required":true}]','[]',$4::uuid,1,'SUBMITTED',$3::uuid,1,$5,$5);
		INSERT INTO capture_response_workspaces(id,tenant_id,legal_entity_id,distribution_id,status,version,created_at,updated_at)
		VALUES('9f650000-0000-7650-8650-000000000054'::uuid,$1::uuid,$2::uuid,'9f650000-0000-7650-8650-000000000052'::uuid,'COMPLETED',1,$5,$5);
		INSERT INTO capture_submissions(id,tenant_id,request_id,submitted_by,channel,answers,submitted_at,created_at,distribution_id)
		VALUES('9f650000-0000-7650-8650-000000000055'::uuid,$1::uuid,'9f650000-0000-7650-8650-000000000053'::uuid,$3::uuid,'INTERNAL','{"q1":"no"}',$5,$5,'9f650000-0000-7650-8650-000000000052'::uuid);
		INSERT INTO capture_response_revisions(id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,achieved_assurance,signoff_summary,compliance_score,scored_weight_coverage,state,critical_field_results,scoring_policy_version,is_current,created_at,score_mode,score_direction,raw_score,adverse_score,concern_band,score_state,score_result,score_profile_checksum,score_calculated_at)
		VALUES('9f650000-0000-7650-8650-000000000056'::uuid,$1::uuid,$2::uuid,'9f650000-0000-7650-8650-000000000052'::uuid,'9f650000-0000-7650-8650-000000000054'::uuid,'9f650000-0000-7650-8650-000000000055'::uuid,1,'EMAIL_VERIFIED','{}',15,100,'FINAL','[]','v1',true,$5,'COMPLIANCE','LOW_IS_POOR',15,85,'HIGH','FINAL','{"profile_version":"v1"}','profile-v1',$5)`, pgx.QueryExecModeSimpleProtocol, policyTenantID, policyEntityID, policyMakerID, policyFormID, createdAt); err != nil {
		t.Fatal(err)
	}
}

func postgresExecutionCommand(now time.Time, policy Policy) ExecutionCommand {
	responseID := "ddb4fe49-9070-3aa1-4335-58dc7bdaeed2" // md5('form-policy:response:1')::uuid
	subjectID := "65408db6-fda1-8860-4712-7d1e610808f2"  // md5('form-policy:subject:1')::uuid
	receiptID := "9f650000-0000-7650-8650-000000000018"
	episodeID := "9f650000-0000-7650-8650-000000000019"
	matterID := "9f650000-0000-7650-8650-000000000020"
	outcomeID := "9f650000-0000-7650-8650-000000000024"
	raw, adverse := 20.0, 80.0
	response := evidence.CompletedResponseSummary{ID: responseID, TenantID: policyTenantID, LegalEntityID: policyEntityID, DistributionID: "ed7af73d-cc8d-c42f-969f-7faf6a4b56c2", FormTemplateID: policyFormID, FormTemplateVersion: 1, Title: "Vendor response", SubjectType: "VENDOR", SubjectID: subjectID, Revision: 1, Current: true, State: evidence.ResponseRevisionFinal, Score: &evidence.ResponseScoreResult{RawScore: &raw, AdverseScore: &adverse, State: evidence.ResponseScoreFinal, Coverage: 1}, CompletedAt: now}
	matter := continuity.Matter{ID: matterID, TenantID: policyTenantID, LegalEntityID: policyEntityID, Reference: "MAT-EXECUTION000020", Type: continuity.MatterVendorDeficiency, Status: continuity.MatterInitialReview, Priority: 4, Title: "Review vendor response", Summary: "The latest response requires review.", Scope: json.RawMessage(`{"policy_id":"9f650000-0000-7650-8650-000000000009"}`), SourceType: "FORM_RESPONSE", SourceID: responseID, TriggerType: "FORM_RESPONSE_POLICY_MATCHED", TriggerID: receiptID, TriggerKey: "form-response-policy:poor-vendor-response:vendor:" + subjectID + ":" + episodeID, KnownFacts: json.RawMessage(`{"adverse_score":80}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: policyMakerID, RequiredAuthority: "ACCOUNTABLE_OWNER", CreatedAt: now, UpdatedAt: now, Version: 1}
	contract := continuity.VerificationContract{ID: outcomeID, TenantID: policyTenantID, MatterID: matterID, ExpectedOutcome: policy.Outcome.ExpectedOutcome, Baseline: json.RawMessage(`{"adverse_score":80}`), Scope: json.RawMessage(`{"policy_id":"9f650000-0000-7650-8650-000000000009"}`), Threshold: json.RawMessage(`{"result":"PASS"}`), ObservationPeriodMinutes: policy.Outcome.CheckAfterMinutes, AuthorityPrincipalID: policyCheckerID, FailureResponse: policy.Outcome.FailureResponse, Status: continuity.VerificationActive, CreatedAt: now, UpdatedAt: now, Version: 1}
	return ExecutionCommand{EventID: "scored-event-1", Policy: policy, Response: response, Route: ExecutionRoute{TenantID: policyTenantID, LegalEntityID: policyEntityID, CanonicalSubjectType: "VENDOR", CanonicalSubjectID: subjectID, ServicePrincipalID: policyCheckerID, OwnerPrincipalID: policyMakerID, ReviewerPrincipalID: policyCheckerID}, Receipt: ExecutionReceipt{ID: receiptID, TenantID: policyTenantID, LegalEntityID: policyEntityID, PolicyID: policy.ID, PolicyVersion: policy.Version, AutomationPolicyID: policy.AutomationPolicyID, AutomationPolicyVersion: policy.AutomationPolicyVersion, ResponseRevisionID: responseID, State: ExecutionApplied, ReasonCode: "POLICY_MATCHED", CreatedAt: now}, Episode: AdverseEpisode{ID: episodeID, TenantID: policyTenantID, LegalEntityID: policyEntityID, PolicyCode: policy.Code, PolicyID: policy.ID, PolicyVersion: policy.Version, SubjectType: "VENDOR", SubjectID: subjectID, State: EpisodeOpen, MatterID: matterID, LastResponseRevisionID: responseID, OpenedAt: now, UpdatedAt: now, RecordVersion: 1}, Matter: matter, Outcome: contract}
}

func postgresDistinctSubjectExecutionCommand(now time.Time, policy Policy) ExecutionCommand {
	command := postgresExecutionCommand(now, policy)
	command.EventID = "scored-event-distinct-subject-2"
	command.Response.ID, command.Response.DistributionID, command.Response.SubjectID = "9f650000-0000-7650-8650-000000000056", "9f650000-0000-7650-8650-000000000052", "9f650000-0000-7650-8650-000000000051"
	command.Route.CanonicalSubjectID = command.Response.SubjectID
	command.Receipt.ID, command.Receipt.ResponseRevisionID = "9f650000-0000-7650-8650-000000000057", command.Response.ID
	command.Episode.ID, command.Episode.SubjectID, command.Episode.MatterID, command.Episode.LastResponseRevisionID = "9f650000-0000-7650-8650-000000000058", command.Response.SubjectID, "9f650000-0000-7650-8650-000000000059", command.Response.ID
	command.Matter.ID, command.Matter.Reference, command.Matter.SourceID, command.Matter.TriggerID, command.Matter.TriggerKey = command.Episode.MatterID, "MAT-EXECUTION000059", command.Response.ID, command.Receipt.ID, "form-response-policy:poor-vendor-response:vendor:"+command.Response.SubjectID+":"+command.Episode.ID
	command.Outcome.ID, command.Outcome.MatterID = "9f650000-0000-7650-8650-000000000060", command.Matter.ID
	return command
}

func assertExecutionBundleCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool, executions, episodes, matters, events, outbox, inbox, maintenance, contracts int) {
	t.Helper()
	var gotExecutions, gotEpisodes, gotMatters, gotEvents, gotOutbox, gotInbox, gotMaintenance, gotContracts int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM form_response_policy_executions WHERE tenant_id=$1::uuid),
		(SELECT count(*) FROM form_response_policy_adverse_episodes WHERE tenant_id=$1::uuid),
		(SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND source_type='FORM_RESPONSE'),
		(SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_type='MATTER' AND aggregate_id='9f650000-0000-7650-8650-000000000020'::uuid),
		(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='MATTER' AND aggregate_id='9f650000-0000-7650-8650-000000000020'::uuid),
		(SELECT count(*) FROM inbox_receipts WHERE tenant_id=$1::uuid AND consumer LIKE 'form-response-policy-executor:%'),
		(SELECT count(*) FROM form_response_policy_maintenance_jobs WHERE tenant_id=$1::uuid),
		(SELECT count(*) FROM verification_contracts WHERE tenant_id=$1::uuid)`, policyTenantID).Scan(&gotExecutions, &gotEpisodes, &gotMatters, &gotEvents, &gotOutbox, &gotInbox, &gotMaintenance, &gotContracts); err != nil {
		t.Fatal(err)
	}
	if gotExecutions != executions || gotEpisodes != episodes || gotMatters != matters || gotEvents != events || gotOutbox != outbox || gotInbox != inbox || gotMaintenance != maintenance || gotContracts != contracts {
		t.Fatalf("bundle counts executions=%d/%d episodes=%d/%d matters=%d/%d events=%d/%d outbox=%d/%d inbox=%d/%d maintenance=%d/%d contracts=%d/%d", gotExecutions, executions, gotEpisodes, episodes, gotMatters, matters, gotEvents, events, gotOutbox, outbox, gotInbox, inbox, gotMaintenance, maintenance, gotContracts, contracts)
	}
}
