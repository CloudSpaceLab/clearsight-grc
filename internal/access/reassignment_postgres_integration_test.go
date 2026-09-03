//go:build postgres && postgresintegration

package access

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresReassignmentRequiresCompleteActiveReportingChain(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tests := []struct {
		name       string
		depth      int
		actorIndex int
		mutate     func(*testing.T, *reassignmentFixture)
		basis      string
	}{
		{name: "direct manager", depth: 2, actorIndex: 1, basis: "REPORTING_ANCESTOR"},
		{name: "higher manager", depth: 2, actorIndex: 2, basis: "REPORTING_ANCESTOR"},
		{name: "root at depth twelve", depth: 12, actorIndex: 12, basis: "REPORTING_ANCESTOR"},
		{name: "current owner", depth: 2, actorIndex: 0, basis: "CURRENT_ASSIGNEE"},
		{name: "current owner does not derive authority from hierarchy", depth: 2, actorIndex: 0, basis: "CURRENT_ASSIGNEE", mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET parent_position_id=$1::uuid WHERE id=$2::uuid`, f.positions[0], f.positions[2])
		}},
		{name: "cycle including candidate", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET parent_position_id=$1::uuid WHERE id=$2::uuid`, f.positions[1], f.positions[2])
		}},
		{name: "cycle back to owner", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET parent_position_id=$1::uuid WHERE id=$2::uuid`, f.positions[0], f.positions[2])
		}},
		{name: "valid chain does not conceal another cyclic owner position", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			var position string
			if err := f.pool.QueryRow(f.ctx, `INSERT INTO org_positions(tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,'SECOND-POSITION','Second test position',$3::uuid,clock_timestamp()-interval '1 day') RETURNING id::text`, f.tenant, f.entity, f.principals[0]).Scan(&position); err != nil {
				t.Fatal(err)
			}
			f.exec(t, `UPDATE org_positions SET parent_position_id=id WHERE id=$1::uuid`, position)
		}},
		{name: "chain exceeds depth limit beyond candidate", depth: 13, actorIndex: 1},
		{name: "candidate at truncated depth limit", depth: 13, actorIndex: 12},
		{name: "foreign entity parent beyond candidate", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET legal_entity_id=$1::uuid WHERE id=$2::uuid`, f.otherEntity, f.positions[2])
		}},
		{name: "expired parent beyond candidate", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET valid_until=clock_timestamp()-interval '1 hour' WHERE id=$1::uuid`, f.positions[2])
		}},
		{name: "future parent beyond candidate", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET valid_from=clock_timestamp()+interval '1 hour' WHERE id=$1::uuid`, f.positions[2])
		}},
		{name: "inactive manager", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, f.principals[1])
		}},
		{name: "expired manager", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE principals SET valid_until=clock_timestamp()-interval '1 hour' WHERE id=$1::uuid`, f.principals[1])
		}},
		{name: "revoked manager position", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET valid_until=clock_timestamp()-interval '1 hour' WHERE id=$1::uuid`, f.positions[1])
		}},
		{name: "inactive owner", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, f.principals[0])
		}},
		{name: "expired owner", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE principals SET valid_until=clock_timestamp()-interval '1 hour' WHERE id=$1::uuid`, f.principals[0])
		}},
		{name: "revoked owner position", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET valid_until=clock_timestamp()-interval '1 hour' WHERE id=$1::uuid`, f.positions[0])
		}},
		{name: "inactive current owner", depth: 2, actorIndex: 0, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, f.principals[0])
		}},
		{name: "expired current owner", depth: 2, actorIndex: 0, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE principals SET valid_until=clock_timestamp()-interval '1 hour' WHERE id=$1::uuid`, f.principals[0])
		}},
		{name: "current owner position revoked", depth: 2, actorIndex: 0, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET valid_until=clock_timestamp()-interval '1 hour' WHERE id=$1::uuid`, f.positions[0])
		}},
		{name: "unknown current owner", depth: 2, actorIndex: 0, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.principals[0] = "00000000-0000-0000-0000-000000000000"
		}},
		{name: "current owner outside entity", depth: 2, actorIndex: 0, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET legal_entity_id=$1::uuid WHERE id=$2::uuid`, f.otherEntity, f.positions[0])
		}},
		{name: "unrelated actor", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.exec(t, `UPDATE org_positions SET occupant_principal_id=NULL WHERE id=$1::uuid`, f.positions[1])
		}},
		{name: "unknown actor", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.principals[1] = "00000000-0000-0000-0000-000000000000"
		}},
		{name: "unknown owner", depth: 2, actorIndex: 1, mutate: func(t *testing.T, f *reassignmentFixture) {
			f.principals[0] = "00000000-0000-0000-0000-000000000000"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newReassignmentFixture(t, ctx, pool, tt.depth)
			if tt.mutate != nil {
				tt.mutate(t, f)
			}
			decision, err := NewPostgresResolver(pool).CanReassign(ctx, ReassignmentRequest{
				TenantID: f.tenant, LegalEntityID: f.entity,
				ActorPrincipalID: f.principals[tt.actorIndex], CurrentOwnerPrincipalID: f.principals[0],
			})
			if err != nil {
				t.Fatalf("reassignment returned existence details or error: %v", err)
			}
			if tt.basis == "" {
				if decision != (ReassignmentDecision{}) {
					t.Fatalf("want uniform denial without hierarchy details, got %#v", decision)
				}
				return
			}
			if !decision.Allowed || decision.Basis != tt.basis {
				t.Fatalf("want allowed %s, got %#v", tt.basis, decision)
			}
			if tt.basis == "REPORTING_ANCESTOR" && decision.HierarchyVersion != int64(tt.depth+1) {
				t.Fatalf("want recorded chain version %d, got %d", tt.depth+1, decision.HierarchyVersion)
			}
		})
	}
}

type reassignmentFixture struct {
	ctx                         context.Context
	pool                        *pgxpool.Pool
	tenant, entity, otherEntity string
	principals, positions       []string
}

func newReassignmentFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, depth int) *reassignmentFixture {
	t.Helper()
	f := &reassignmentFixture{ctx: ctx, pool: pool}
	if err := pool.QueryRow(ctx, `INSERT INTO tenants(slug,name) VALUES('reassignment-test-'||uuidv7()::text,'Reassignment integration test') RETURNING id::text`).Scan(&f.tenant); err != nil {
		t.Fatal(err)
	}
	// Each subtest owns a generated tenant. Delete only that tenant's fixtures,
	// including both ends of reporting cycles in one statement.
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, table := range []string{"org_positions", "principals", "legal_entities"} {
			if _, err := pool.Exec(cleanCtx, `DELETE FROM `+table+` WHERE tenant_id=$1::uuid`, f.tenant); err != nil {
				t.Errorf("clean %s fixtures: %v", table, err)
				return
			}
		}
		if _, err := pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, f.tenant); err != nil {
			t.Errorf("clean tenant fixture: %v", err)
		}
	})
	for i, target := range []*string{&f.entity, &f.otherEntity} {
		if err := pool.QueryRow(ctx, `INSERT INTO legal_entities(tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2,'Test bank','NG',clock_timestamp()-interval '1 day') RETURNING id::text`, f.tenant, fmt.Sprintf("BANK-%d", i)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i <= depth; i++ {
		var principal, position string
		if err := pool.QueryRow(ctx, `INSERT INTO principals(tenant_id,kind,display_name,valid_from) VALUES($1::uuid,'PERSON',$2,clock_timestamp()-interval '1 day') RETURNING id::text`, f.tenant, fmt.Sprintf("Test colleague %d", i)).Scan(&principal); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO org_positions(tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from,version) VALUES($1::uuid,$2::uuid,$3,$3,$4::uuid,clock_timestamp()-interval '1 day',$5) RETURNING id::text`, f.tenant, f.entity, fmt.Sprintf("POSITION-%d", i), principal, i+1).Scan(&position); err != nil {
			t.Fatal(err)
		}
		f.principals = append(f.principals, principal)
		f.positions = append(f.positions, position)
		if i > 0 {
			f.exec(t, `UPDATE org_positions SET parent_position_id=$1::uuid WHERE id=$2::uuid`, position, f.positions[i-1])
		}
	}
	return f
}

func (f *reassignmentFixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(f.ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}
