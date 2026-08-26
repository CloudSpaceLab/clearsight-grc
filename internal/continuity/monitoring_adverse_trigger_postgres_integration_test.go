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

func TestPostgresMonitoringAdverseTriggerIsAtomicAndIdempotent(t *testing.T) {
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
		tenantID       = "96666666-6666-7666-8666-666666666661"
		entityID       = "96666666-6666-7666-8666-666666666662"
		ownerID        = "96666666-6666-7666-8666-666666666663"
		reviewerID     = "96666666-6666-7666-8666-666666666664"
		checkID        = "96666666-6666-7666-8666-666666666665"
		triggerID      = "96666666-6666-7666-8666-666666666666"
		rollbackID     = "96666666-6666-7666-8666-666666666667"
		rollbackMatter = "96666666-6666-7666-8666-666666666668"
		rollbackLink   = "96666666-6666-7666-8666-666666666669"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'monitoring-adverse-trigger-test','Monitoring Adverse Trigger Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'ENTITY-A','Entity A','NG')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$3::uuid,'PERSON','Program owner'),($2::uuid,$3::uuid,'PERSON','Control assurance reviewer')`, ownerID, reviewerID, tenantID); err != nil {
		t.Fatal(err)
	}

	repo := NewCurrentPostgresRepository(pool)
	service := NewService(repo)
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "monitoring-adverse-trigger-test", LegalEntityID: entityID, Code: "ACCESS", Name: "Access monitoring",
		Type: "COMPLIANCE", OwningFunction: "Information Security", OwnerPrincipalID: ownerID,
		Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	trigger := Trigger{
		ID: triggerID, TenantID: "monitoring-adverse-trigger-test", ProgramID: program.Program.ID,
		Type: "MONITORING_RESULT_ADVERSE", SubjectType: "MONITORING_CHECK", SubjectID: checkID,
		DedupeKey: "monitoring-adverse:check-1:period-2026-08", Payload: json.RawMessage(`{"risk_band":"HIGH","score":72}`),
		ObservedAt: now.Add(time.Minute), Source: "monitoring", ActorID: reviewerID,
	}
	updated, matter, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted || matter == nil || matter.LegalEntityID != entityID || matter.OwnerPrincipalID != ownerID || matter.RequiredAuthority != "CONTROL_ASSURANCE" || matter.Type != MatterControlGap || matter.Priority != 4 {
		t.Fatalf("adverse monitoring result = inserted=%v matter=%#v", inserted, matter)
	}
	replayed, duplicate, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil {
		t.Fatal(err)
	}
	if inserted || duplicate == nil || duplicate.ID != matter.ID || replayed.Program.Version != updated.Program.Version {
		t.Fatalf("retry was not idempotent: inserted=%v matter=%#v program=%#v", inserted, duplicate, replayed)
	}

	var triggers, programEvents, matters, links, matterEvents, outboxEvents, projectionJobs int
	checks := []struct {
		query  string
		args   []any
		target *int
	}{
		{`SELECT count(*) FROM program_trigger_events WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND dedupe_key=$3`, []any{tenantID, program.Program.ID, trigger.DedupeKey}, &triggers},
		{`SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND event_type='PROGRAM_TRIGGER_RECORDED' AND payload->>'dedupe_key'=$3`, []any{tenantID, program.Program.ID, trigger.DedupeKey}, &programEvents},
		{`SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND id=$2::uuid AND legal_entity_id=$3::uuid AND owner_principal_id=$4::uuid AND required_authority='CONTROL_ASSURANCE'`, []any{tenantID, matter.ID, entityID, ownerID}, &matters},
		{`SELECT count(*) FROM matter_links WHERE tenant_id=$1::uuid AND matter_id=$2::uuid AND program_id=$3::uuid`, []any{tenantID, matter.ID, program.Program.ID}, &links},
		{`SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_type='MATTER' AND aggregate_id=$2::uuid`, []any{tenantID, matter.ID}, &matterEvents},
		{`SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND ((aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND event_type='PROGRAM_TRIGGER_RECORDED' AND payload->>'dedupe_key'=$3) OR (aggregate_type='MATTER' AND aggregate_id=$4::uuid))`, []any{tenantID, program.Program.ID, trigger.DedupeKey, matter.ID}, &outboxEvents},
		{`SELECT count(*) FROM continuity_projection_jobs WHERE tenant_id=$1::uuid AND aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND source_aggregate_version=$3`, []any{tenantID, program.Program.ID, updated.Program.Version}, &projectionJobs},
	}
	for _, check := range checks {
		if err := pool.QueryRow(ctx, check.query, check.args...).Scan(check.target); err != nil {
			t.Fatal(err)
		}
	}
	if triggers != 1 || programEvents != 1 || matters != 1 || links != 1 || matterEvents != 2 || outboxEvents != 3 || projectionJobs != 1 {
		t.Fatalf("atomic/idempotent rows triggers=%d program_events=%d matters=%d links=%d matter_events=%d outbox=%d jobs=%d", triggers, programEvents, matters, links, matterEvents, outboxEvents, projectionJobs)
	}

	current, err := repo.GetProgram(ctx, "monitoring-adverse-trigger-test", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	rollbackTrigger := Trigger{ID: rollbackID, TenantID: "monitoring-adverse-trigger-test", ProgramID: program.Program.ID, Type: "MONITORING_RESULT_ADVERSE", SubjectType: "MONITORING_CHECK", SubjectID: checkID, DedupeKey: "monitoring-adverse:rollback", Payload: json.RawMessage(`{"risk_band":"CRITICAL"}`), ObservedAt: now.Add(2 * time.Minute), Source: "monitoring", ActorID: reviewerID}
	programEvent, err := newEvent(rollbackTrigger.TenantID, "PROGRAM", rollbackTrigger.ProgramID, current.Program.Version+1, EventProgramTriggerRecorded, rollbackTrigger, ActorPerson, reviewerID, rollbackTrigger.ObservedAt)
	if err != nil {
		t.Fatal(err)
	}
	badMatter := Matter{ID: rollbackMatter, TenantID: rollbackTrigger.TenantID, LegalEntityID: entityID, Reference: matterReference(rollbackMatter), Type: MatterControlGap, Status: MatterInitialReview, Priority: 4, Title: "Rollback test", Summary: "The bundle must roll back.", Scope: json.RawMessage(`{}`), TriggerType: rollbackTrigger.Type, TriggerID: rollbackTrigger.ID, TriggerKey: rollbackTrigger.DedupeKey, KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: "not-a-uuid", RequiredAuthority: "CONTROL_ASSURANCE", CreatedAt: rollbackTrigger.ObservedAt, UpdatedAt: rollbackTrigger.ObservedAt, Version: 1}
	matterEvent, err := newEvent(rollbackTrigger.TenantID, "MATTER", badMatter.ID, 1, EventMatterCreated, badMatter, ActorPerson, reviewerID, rollbackTrigger.ObservedAt)
	if err != nil {
		t.Fatal(err)
	}
	link := MatterLink{ID: rollbackLink, TenantID: rollbackTrigger.TenantID, MatterID: badMatter.ID, ProgramID: rollbackTrigger.ProgramID, Relationship: "AFFECTS", CreatedAt: rollbackTrigger.ObservedAt}
	linkEvent, err := newEvent(rollbackTrigger.TenantID, "MATTER", badMatter.ID, 2, EventMatterLinked, link, ActorPerson, reviewerID, rollbackTrigger.ObservedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.ApplyTriggerBundle(ctx, TriggerBundle{Trigger: rollbackTrigger, ProgramEvent: programEvent, Matter: &badMatter, MatterEvent: &matterEvent, Link: &link, LinkEvent: &linkEvent}); err == nil {
		t.Fatal("invalid Matter owner did not fail the trigger bundle")
	}
	var rollbackTriggers, rollbackEvents, rollbackOutbox, rollbackMatters, rollbackLinks, rollbackJobs int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM program_trigger_events WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND dedupe_key=$3`, tenantID, program.Program.ID, rollbackTrigger.DedupeKey).Scan(&rollbackTriggers); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND id IN ($2::uuid,$3::uuid,$4::uuid)`, tenantID, programEvent.ID, matterEvent.ID, linkEvent.ID).Scan(&rollbackEvents); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND ((aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND payload->>'dedupe_key'=$3) OR (aggregate_type='MATTER' AND aggregate_id=$4::uuid))`, tenantID, program.Program.ID, rollbackTrigger.DedupeKey, rollbackMatter).Scan(&rollbackOutbox); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, rollbackMatter).Scan(&rollbackMatters); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM matter_links WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, rollbackLink).Scan(&rollbackLinks); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM continuity_projection_jobs WHERE tenant_id=$1::uuid AND aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND source_aggregate_version=$3`, tenantID, program.Program.ID, current.Program.Version+1).Scan(&rollbackJobs); err != nil {
		t.Fatal(err)
	}
	afterRollback, err := repo.GetProgram(ctx, "monitoring-adverse-trigger-test", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rollbackTriggers != 0 || rollbackEvents != 0 || rollbackOutbox != 0 || rollbackMatters != 0 || rollbackLinks != 0 || rollbackJobs != 0 || afterRollback.Program.Version != current.Program.Version {
		t.Fatalf("failed bundle left partial state triggers=%d events=%d outbox=%d matters=%d links=%d jobs=%d version=%d want=%d", rollbackTriggers, rollbackEvents, rollbackOutbox, rollbackMatters, rollbackLinks, rollbackJobs, afterRollback.Program.Version, current.Program.Version)
	}
}
