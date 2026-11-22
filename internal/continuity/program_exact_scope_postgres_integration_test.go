//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type revokePostgresBeforeCheckpoint struct {
	*CurrentPostgresRepository
	pool *pgxpool.Pool
}

type revokePostgresDuringDigest struct {
	*CurrentPostgresRepository
	pool  *pgxpool.Pool
	stage string
}

func (r *revokePostgresDuringDigest) ProgramStateVersion(ctx context.Context, tenant, programID string, version int64) (*ProgramStateSnapshot, error) {
	if r.stage == "baseline" {
		if _, err := r.pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, programID); err != nil {
			return nil, err
		}
	}
	return r.CurrentPostgresRepository.ProgramStateVersion(ctx, tenant, programID, version)
}
func (r *revokePostgresDuringDigest) ProgramEventsAfterVersion(ctx context.Context, tenant, programID string, version int64, limit int) ([]Event, bool, error) {
	if r.stage == "events" {
		if _, err := r.pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, programID); err != nil {
			return nil, false, err
		}
	}
	return r.CurrentPostgresRepository.ProgramEventsAfterVersion(ctx, tenant, programID, version, limit)
}

func (r *revokePostgresBeforeCheckpoint) LatestProgramReview(ctx context.Context, tenant, programID, principalID string) (*ProgramReviewCheckpoint, error) {
	if _, err := r.pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, programID); err != nil {
		return nil, err
	}
	return r.CurrentPostgresRepository.LatestProgramReview(ctx, tenant, programID, principalID)
}

func TestPostgresExactProgramReadEnforcesPrincipalScope(t *testing.T) {
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
	var tenant, entity, program, reviewer, other string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &program, &reviewer, &other); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$1,'Scoped Program'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'BANK','Bank','NG',clock_timestamp()-interval '1 hour'); INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,scope,effective_from) VALUES($3::uuid,$1::uuid,$2::uuid,'SCOPE','Restricted Program','COMPLIANCE','ACTIVE','Risk',jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($4::text)),clock_timestamp()-interval '1 hour')`, pgx.QueryExecModeSimpleProtocol, tenant, entity, program, reviewer); err != nil {
		t.Fatal(err)
	}
	store := NewReliablePostgresRepository(pool)
	actor := func(principal string) context.Context {
		return identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: principal, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)})
	}
	if _, err = store.GetProgram(actor(reviewer), tenant, program); err != nil {
		t.Fatalf("named principal denied: %v", err)
	}
	if _, err = store.GetProgram(actor(other), tenant, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted Program disclosed: %v", err)
	}
	if _, err = store.GetProgram(actor(""), tenant, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing principal read Program: %v", err)
	}
	var createdID string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&createdID); err != nil {
		t.Fatal(err)
	}
	scope, err := json.Marshal(map[string]any{"access": "RESTRICTED", "allowed_principal_ids": []string{reviewer}})
	if err != nil {
		t.Fatal(err)
	}
	created := Program{ID: createdID, TenantID: tenant, LegalEntityID: entity, Code: "NEW-PRIVATE", Name: "New private Program", Type: "COMPLIANCE", Status: ProgramDraft, OwningFunction: "Risk", Scope: scope, EffectiveFrom: time.Now().Add(-time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(), Version: 1}
	event, err := newEvent(tenant, "PROGRAM", createdID, 1, EventProgramCreated, created, ActorSystem, "", created.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.CreateProgram(WithTrustedSystemScope(ctx), created, event); err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetProgram(actor(other), tenant, createdID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("post-create fallback disclosed restricted Program: %v", err)
	}
	if events, e := store.ProgramEvents(actor(other), tenant, createdID, nil); !errors.Is(e, ErrNotFound) || len(events) != 0 {
		t.Fatalf("restricted Program events disclosed: %+v %v", events, e)
	}
	until := time.Now().Add(time.Minute)
	if events, e := store.ProgramEvents(actor(reviewer), tenant, createdID, &until); e != nil || len(events) != 1 {
		t.Fatalf("permitted Program events: %+v %v", events, e)
	}
	var visible1, visible2 string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&visible1, &visible2); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET updated_at=clock_timestamp()+interval '1 minute' WHERE id=$1::uuid; INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,scope,effective_from,updated_at) VALUES($2::uuid,$4::uuid,$5::uuid,'VISIBLE-1','Visible one','COMPLIANCE','ACTIVE','Risk','{}',clock_timestamp()-interval '1 hour',clock_timestamp()),($3::uuid,$4::uuid,$5::uuid,'VISIBLE-2','Visible two','COMPLIANCE','ACTIVE','Risk','{}',clock_timestamp()-interval '1 hour',clock_timestamp()-interval '1 minute')`, pgx.QueryExecModeSimpleProtocol, program, visible1, visible2, tenant, entity); err != nil {
		t.Fatal(err)
	}
	listed, err := store.ListPrograms(actor(other), tenant, 1)
	if err != nil || len(listed) != 1 || listed[0].Program.ID != visible1 {
		t.Fatalf("restricted row affected list limit: %+v %v", listed, err)
	}
	first, err := store.ListProgramSummaries(actor(other), tenant, SummaryQuery{Limit: 1})
	if err != nil || len(first.Items) != 1 || first.Items[0].Program.ID != visible1 || first.NextCursor == "" {
		t.Fatalf("first scoped page: %+v %v", first, err)
	}
	second, err := store.ListProgramSummaries(actor(other), tenant, SummaryQuery{Limit: 1, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.Items[0].Program.ID != visible2 || second.NextCursor != "" {
		t.Fatalf("second scoped page: %+v %v", second, err)
	}
	var matter string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&matter); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions) VALUES($1::uuid,$2::uuid,$3::uuid,'SCOPE-MATTER','AUTHORITY_REQUEST','ASSESSMENT',3,'Visible issue','Visible issue','{}','{}','[]','[]'); INSERT INTO matter_links(tenant_id,matter_id,program_id,relationship) VALUES($2::uuid,$1::uuid,$4::uuid,'AFFECTS'),($2::uuid,$1::uuid,$5::uuid,'AFFECTS')`, pgx.QueryExecModeSimpleProtocol, matter, tenant, entity, program, visible1); err != nil {
		t.Fatal(err)
	}
	matterPage, err := store.ListMatterSummaries(actor(other), tenant, SummaryQuery{Limit: 10})
	if err != nil || len(matterPage.Items) != 1 || matterPage.Items[0].ProgramCount != 1 {
		t.Fatalf("restricted Program affected Matter count: %+v %v", matterPage, err)
	}
	reviewerMatterPage, err := store.ListMatterSummaries(actor(reviewer), tenant, SummaryQuery{Limit: 10})
	if err != nil || len(reviewerMatterPage.Items) != 1 || reviewerMatterPage.Items[0].ProgramCount != 2 {
		t.Fatalf("authorized linked Program count: %+v %v", reviewerMatterPage, err)
	}
	filteredMatterPage, err := store.ListMatterSummaries(actor(other), tenant, SummaryQuery{Limit: 10, ProgramID: program})
	if err != nil || len(filteredMatterPage.Items) != 0 {
		t.Fatalf("restricted Program affected Matter filter: %+v %v", filteredMatterPage, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[42]}'::jsonb WHERE id=$1::uuid`, program); err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetProgram(actor(reviewer), tenant, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("malformed scope disclosed: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, program); err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetProgram(actor(other), tenant, program); err != nil {
		t.Fatalf("internal Program denied: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, program); err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetProgram(actor(other), tenant, program); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked Program remained visible: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO program_state_snapshots(tenant_id,program_id,overall_state,dimensions,reasons,generated_at,program_version,projection_version,calculation) VALUES($1::uuid,$2::uuid,'UNKNOWN','{}','[]',clock_timestamp(),1,1,'{"policy_revision":"v1","state":"FINAL","coverage_bps":10000,"categories":[{"code":"REQUIREMENTS","weight_bps":10000}]}'::jsonb)`, tenant, visible1); err != nil {
		t.Fatal(err)
	}
	if snapshot, e := store.ProgramStateVersion(actor(other), tenant, visible1, 1); e != nil || snapshot == nil || snapshot.Calculation == nil || snapshot.Calculation.PolicyRevision != "v1" {
		t.Fatalf("baseline calculation unavailable: %+v %v", snapshot, e)
	}
	service := NewService(&revokePostgresBeforeCheckpoint{CurrentPostgresRepository: NewCurrentPostgresRepository(pool), pool: pool})
	if digest, e := service.ProgramReviewDigest(actor(other), tenant, visible1, other); !errors.Is(e, ErrNotFound) {
		t.Fatalf("revoked scope disclosed Program digest: %+v %v", digest, e)
	}
}
