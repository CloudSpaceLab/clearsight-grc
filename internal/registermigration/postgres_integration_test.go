//go:build postgres && postgresintegration

package registermigration

import (
	"context"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

type pgVendorReader struct {
	repo *thirdparty.PostgresRepository
}

func (v pgVendorReader) GetRelationship(ctx context.Context, a thirdparty.Actor, id string) (thirdparty.Aggregate, error) {
	return v.repo.GetRelationship(ctx, thirdparty.Scope{TenantID: a.TenantID, LegalEntityID: a.LegalEntityID}, id)
}

type pgRoutes struct {
	authority.Service
	actor, owner string
}

func (r pgRoutes) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return authority.Resolution{Principal: authority.Principal{ID: r.actor, Kind: "PERSON"}, CandidatePrincipals: []authority.Principal{{ID: r.owner, Kind: "PERSON"}}}, nil
}

type brokenLinkRepo struct{ Repository }

func (r brokenLinkRepo) Commit(ctx context.Context, c Commit) (Draft, error) {
	c.Links[1].TargetID = stableID("missing-target")
	return r.Repository.Commit(ctx, c)
}

func TestPostgresRegisterMigrationAtomicReceiptAndRollback(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, rollback := range []bool{false, true} {
		t.Run(fmt.Sprintf("rollback=%t", rollback), func(t *testing.T) {
			f := setup(t)
			key := fmt.Sprintf("register-%d", time.Now().UnixNano())
			tenant, entity, actor, owner, source, vendor, payments, hosting := stableID(key+"t"), stableID(key+"e"), stableID(key+"a"), stableID(key+"o"), stableID(key+"s"), stableID(key+"v"), stableID(key+"p"), stableID(key+"h")
			ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: actor, ExpiresAt: time.Now().Add(time.Hour)})
			_, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$1,'Register test');
  INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'TEST','Test entity','NG');
  INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($3::uuid,$1::uuid,'PERSON','Importer'),($4::uuid,$1::uuid,'PERSON','Owner');
  INSERT INTO document_imports(id,tenant_id,legal_entity_id,file_name,purpose,size_bytes,sha256,storage_key,artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,created_by,version) VALUES($5::uuid,$1::uuid,$2::uuid,'register.xlsx','Import findings',100,repeat('a',64),$5,'AVAILABLE','EXTRACTED','XLSX','NO_PROPOSALS','DETERMINISTIC',$3::uuid,2);
  INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($6::uuid,$1::uuid,'Example Ltd','ACTIVE',now(),now(),1);
  INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($7::uuid,$1::uuid,$2::uuid,$6::uuid,'Payments',$4::uuid,'STANDARD','NONE','ACTIVE',now(),now(),1),($8::uuid,$1::uuid,$2::uuid,$6::uuid,'Hosting',$4::uuid,'STANDARD','NONE','ACTIVE',now(),now(),1);`, pgx.QueryExecModeSimpleProtocol, tenant, entity, actor, owner, source, vendor, payments, hosting)
			if err != nil {
				t.Fatal(err)
			}
			f.source.document.ID = source
			f.source.document.TenantID = tenant
			f.source.document.LegalEntityID = entity
			repo := NewPostgresRepository(pool)
			service := New(repo, f.source, pgVendorReader{thirdparty.NewPostgresRepository(pool)}, pgRoutes{actor: actor, owner: owner})
			v, err := service.View(ctx, source)
			if err != nil {
				t.Fatal(err)
			}
			for i := range v.Draft.Selection.Groups {
				v.Draft.Selection.Groups[i].RelationshipID = []string{payments, hosting}[i]
				v.Draft.Selection.Groups[i].RelationshipVersion = 1
			}
			for i := range v.Draft.Selection.Owners {
				v.Draft.Selection.Owners[i].PersonID = owner
			}
			v, err = service.Save(ctx, source, 0, 2, v.Draft.Selection)
			if err != nil {
				t.Fatal(err)
			}
			if rollback {
				service.repo = brokenLinkRepo{repo}
			}
			d, err := service.Import(ctx, source, v.Draft.Version)
			if rollback {
				if err == nil {
					t.Fatal("broken batch committed")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			want := 3
			if rollback {
				want = 0
			}
			for _, table := range []string{"matters", "matter_actions", "third_party_relationship_matter_links"} {
				var count int
				err = pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE tenant_id=$1::uuid", tenant).Scan(&count)
				if err != nil || count != want {
					t.Fatalf("%s: %d want %d (%v)", table, count, want, err)
				}
			}
			var events int
			err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type IN ('MATTER_CREATED','ACTION_ADDED','VendorRelationshipLinked','RiskRegisterImported')`, tenant).Scan(&events)
			if err != nil {
				t.Fatal(err)
			}
			if rollback {
				if events != 0 {
					t.Fatal("failed batch emitted events")
				}
				return
			}
			if events != 10 {
				t.Fatalf("canonical events: got %d want 10", events)
			}
			replay, err := service.Import(ctx, source, v.Draft.Version)
			if err != nil || len(replay.Receipts) != 3 || replay.Receipts[0].MatterID != d.Receipts[0].MatterID {
				t.Fatal("receipt replay failed", err)
			}
			var revisions int
			err = pool.QueryRow(ctx, `SELECT count(*) FROM risk_register_migration_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&revisions)
			if err != nil || revisions != 2 {
				t.Fatal("history missing", err, revisions)
			}
			if _, err = pool.Exec(ctx, `UPDATE risk_register_migration_revisions SET snapshot='{}' WHERE tenant_id=$1::uuid`, tenant); err == nil {
				t.Fatal("history was mutable")
			}
		})
	}
}
