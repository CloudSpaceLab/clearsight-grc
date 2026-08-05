//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCommandIntegrityAndProgramStatusOperations(t *testing.T) {
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
	const tenantID = "66666666-6666-7666-8666-666666666661"
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'integrity-bank','Integrity Bank')`, tenantID); err != nil {
		t.Fatal(err)
	}

	repository := continuity.NewPostgresRepository(pool)
	service := continuity.NewService(repository)
	now := time.Now().UTC().Truncate(time.Second)
	program, err := service.CreateProgram(ctx, continuity.CreateProgramInput{TenantID: "integrity-bank", Code: "RESILIENCE", Name: "Operational resilience", Type: "RESILIENCE", OwningFunction: "Operational Risk", Scope: json.RawMessage(`{"entity":"Integrity Bank"}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Version != 1 || program.CurrentState != nil {
		t.Fatalf("command returned an unexpected synchronous status: %#v", program)
	}
	var readyJobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_projection_jobs WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND status='READY'`, tenantID, program.Program.ID).Scan(&readyJobs); err != nil {
		t.Fatal(err)
	}
	if readyJobs != 1 {
		t.Fatalf("expected one deduplicated status update job, got %d", readyJobs)
	}

	maintainer := &continuity.ProjectionMaintainer{Service: service, Repo: repository, WorkerID: "projection-test-worker"}
	completed, err := maintainer.Maintain(ctx, time.Now().UTC().Add(time.Second), 20)
	if err != nil || completed != 1 {
		t.Fatalf("initial status maintenance failed completed=%d err=%v", completed, err)
	}
	program, err = service.GetProgram(ctx, "integrity-bank", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Version != 1 || program.CurrentState == nil || program.CurrentState.ProgramVersion != 1 {
		t.Fatalf("calculated status changed the command version: %#v", program)
	}
	var commandEvents, stateEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE aggregate_type='PROGRAM'),count(*) FILTER (WHERE aggregate_type='PROGRAM_STATE') FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid`, tenantID, program.Program.ID).Scan(&commandEvents, &stateEvents); err != nil {
		t.Fatal(err)
	}
	if commandEvents != 1 || stateEvents != 1 {
		t.Fatalf("unexpected event streams command=%d state=%d", commandEvents, stateEvents)
	}

	program, err = service.AddRequirement(ctx, continuity.AddRequirementInput{TenantID: "integrity-bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "TEST-RECOVERY", Title: "Test recovery arrangements", Statement: "Recovery arrangements must be tested.", Modality: "MUST", Status: continuity.RequirementApproved, EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Version != 2 || program.CurrentState == nil || program.CurrentState.ProgramVersion != 1 {
		t.Fatalf("latest status should remain visible but stale until maintained: %#v", program)
	}
	health, err := service.ProjectionHealth(ctx, "integrity-bank")
	if err != nil || len(health) != 1 || health[0].Pending != 1 {
		t.Fatalf("unexpected pending health %#v err=%v", health, err)
	}
	pendingReconcile, err := service.ReconcileProgramState(ctx, "integrity-bank", 20)
	if err != nil || pendingReconcile.Queued != 0 || pendingReconcile.AlreadyQueued != 1 {
		t.Fatalf("reconciliation did not report the existing job %#v err=%v", pendingReconcile, err)
	}
	completed, err = maintainer.Maintain(ctx, time.Now().UTC().Add(2*time.Second), 20)
	if err != nil || completed != 1 {
		t.Fatalf("requirement status maintenance failed completed=%d err=%v", completed, err)
	}
	program, err = service.GetProgram(ctx, "integrity-bank", program.Program.ID)
	if err != nil || program.CurrentState == nil || program.CurrentState.ProgramVersion != 2 || program.Program.Version != 2 {
		t.Fatalf("unexpected maintained Program %#v err=%v", program, err)
	}

	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: "integrity-bank", Type: continuity.MatterControlGap, Priority: 3, Title: "Complete the recovery exercise", Summary: "The current recovery exercise has not been completed.", Scope: json.RawMessage(`{"service":"payments"}`), ProgramID: program.Program.ID})
	if err != nil {
		t.Fatal(err)
	}
	if matter.Matter.Version != 2 || len(matter.Links) != 1 {
		t.Fatalf("issue and first Program link were not committed together: %#v", matter)
	}
	var matterEvents, matterLinks int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_type='MATTER' AND aggregate_id=$2::uuid`, tenantID, matter.Matter.ID).Scan(&matterEvents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matter_links WHERE tenant_id=$1::uuid AND matter_id=$2::uuid`, tenantID, matter.Matter.ID).Scan(&matterLinks); err != nil {
		t.Fatal(err)
	}
	if matterEvents != 2 || matterLinks != 1 {
		t.Fatalf("unexpected atomic issue records events=%d links=%d", matterEvents, matterLinks)
	}

	trigger := continuity.Trigger{TenantID: "integrity-bank", ProgramID: program.Program.ID, Type: "CONTROL_FAILED", SubjectType: "PROGRAM", SubjectID: program.Program.ID, DedupeKey: "resilience-test-failed", Payload: json.RawMessage(`{"service":"payments"}`), ObservedAt: now.Add(time.Minute), Source: "resilience-monitor"}
	_, triggered, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil || !inserted || triggered == nil {
		t.Fatalf("trigger bundle failed inserted=%v matter=%#v err=%v", inserted, triggered, err)
	}
	_, duplicate, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil || inserted || duplicate == nil || duplicate.ID != triggered.ID {
		t.Fatalf("duplicate trigger contract failed inserted=%v matter=%#v err=%v", inserted, duplicate, err)
	}
	var triggerRows, triggerMatterRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM program_trigger_events WHERE tenant_id=$1::uuid AND dedupe_key='resilience-test-failed'`, tenantID).Scan(&triggerRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND trigger_key='resilience-test-failed'`, tenantID).Scan(&triggerMatterRows); err != nil {
		t.Fatal(err)
	}
	if triggerRows != 1 || triggerMatterRows != 1 {
		t.Fatalf("trigger bundle was not idempotent triggers=%d matters=%d", triggerRows, triggerMatterRows)
	}

	completed, err = maintainer.Maintain(ctx, time.Now().UTC().Add(3*time.Second), 20)
	if err != nil || completed != 1 {
		t.Fatalf("linked issue status maintenance failed completed=%d err=%v", completed, err)
	}
	reconciled, err := service.ReconcileProgramState(ctx, "integrity-bank", 20)
	if err != nil || reconciled.Checked != 1 || reconciled.Queued != 0 || reconciled.Current != 1 {
		t.Fatalf("unexpected reconciliation %#v err=%v", reconciled, err)
	}
}
