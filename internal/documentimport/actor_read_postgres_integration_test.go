//go:build postgres && postgresintegration

package documentimport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresImportReadFiltersActorBeforeLimit(t *testing.T) {
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
	var tenant, entity, actor, other, allowed, denied string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &actor, &other, &allowed, &denied); err != nil {
		t.Fatal(err)
	}
	slug := "import-read-" + tenant
	now := time.Now().UTC()
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Import test')`, tenant, slug); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM segregation_rules WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM delegations WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM document_imports WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenant)
	})
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'ENTITY','Entity','NG')`, entity, tenant); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{actor, other} {
		if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Person','ACTIVE',$3)`, id, tenant, now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewPostgresRepository(pool)
	for i, id := range []string{allowed, denied} {
		at := now.Add(time.Duration(i) * time.Minute)
		_, err = repo.Create(ctx, Document{ID: id, TenantID: tenant, LegalEntityID: entity, FileName: fmt.Sprintf("source-%d.txt", i), MediaType: "text/plain", Purpose: "Review", SourceType: "DOCUMENT", SizeBytes: 1, SHA256: strings.Repeat("a", 64), StorageKey: "test-" + id, ArtifactStatus: "AVAILABLE", ExtractionStatus: ExtractionPending, ExtractionMethod: "PENDING", AnalysisStatus: AnalysisPending, AnalysisMethod: "PENDING", Limitations: []string{}, Sections: []Section{}, Elements: []ExtractedElement{}, Degradations: []Degradation{}, Proposals: []Proposal{}, CreatedBy: actor, CreatedAt: at, UpdatedAt: at, Version: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type) VALUES($1::uuid,$2::uuid,$3::uuid,'PERFORMER','DOCUMENT_IMPORT',$4::uuid,100,$5,'read:v1','DOCUMENT.IMPORT.READ')`, tenant, entity, actor, allowed, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	person := identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: actor, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}
	actorCtx := identity.WithActor(ctx, person)
	assertRoute := func(documentID, principal string, want bool) {
		t.Helper()
		result, routeErr := authority.NewPostgresService(pool).Resolve(ctx, authority.ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "DOCUMENT_IMPORT", ObjectID: documentID, Responsibility: authority.ResponsibilityPerformer, DecisionType: "DOCUMENT.IMPORT.READ", Materiality: 3, At: now})
		allowedByResolver := routeErr == nil && result.AllowsPrincipal(principal)
		actorForRead := person
		actorForRead.PrincipalID = principal
		_, readErr := service.GetVisibleForActor(identity.WithActor(ctx, actorForRead), documentID)
		if allowedByResolver != want || (readErr == nil) != want {
			t.Fatalf("route mismatch resolver=%v (%v), read=%v, want=%v", allowedByResolver, routeErr, readErr, want)
		}
	}
	assertRoute(allowed, actor, true)
	assertRoute(allowed, other, false)
	page, err := service.ListVisibleForActor(actorCtx, 1)
	if err != nil || len(page) != 1 || page[0].ID != allowed {
		t.Fatalf("authorized page: %#v %v", page, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=$3 WHERE tenant_id=$1::uuid AND object_id=$2::uuid`, tenant, allowed, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetForActor(ctx, tenant, entity, actor, allowed, now.Add(-2*time.Minute), now.Add(time.Hour)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired route with stale request time: %v", err)
	}
	page, err = repo.ListForActor(ctx, tenant, entity, actor, 1, now.Add(-2*time.Minute), now.Add(time.Hour))
	if err != nil || len(page) != 0 {
		t.Fatalf("expired route in list with stale request time: %#v %v", page, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=NULL WHERE tenant_id=$1::uuid AND object_id=$2::uuid`, tenant, allowed); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetForActor(ctx, tenant, entity, actor, allowed, now.Add(-2*time.Minute), now.Add(-time.Minute)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired identity in exact read: %v", err)
	}
	page, err = repo.ListForActor(ctx, tenant, entity, actor, 1, now.Add(-2*time.Minute), now.Add(-time.Minute))
	if err != nil || len(page) != 0 {
		t.Fatalf("expired identity in list: %#v %v", page, err)
	}
	if _, err = service.GetVisibleForActor(actorCtx, denied); !errors.Is(err, ErrNotFound) {
		t.Fatalf("denied exact: %v", err)
	}
	person.PrincipalID = other
	if _, err = service.GetVisibleForActor(identity.WithActor(ctx, person), allowed); !errors.Is(err, ErrNotFound) {
		t.Fatalf("different actor: %v", err)
	}
	var otherEntity, delegationID string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&otherEntity, &delegationID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'OTHER','Other','NG')`, otherEntity, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO delegations(id,tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,'PERFORMER',jsonb_build_object('legal_entity_id',$3::text),$6,$7,'ACTIVE',$4::uuid)`, delegationID, tenant, otherEntity, actor, other, now.Add(-time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetVisibleForActor(identity.WithActor(ctx, person), allowed); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity delegation: %v", err)
	}
	assertRoute(allowed, other, false)
	if _, err = pool.Exec(ctx, `DELETE FROM delegations WHERE id=$1::uuid`, delegationID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type) VALUES($1::uuid,$2::uuid,$3::uuid,'PERFORMER','DOCUMENT_IMPORT',$4::uuid,100,$5,'read:conflict','DOCUMENT.IMPORT.READ')`, tenant, entity, other, allowed, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	assertRoute(allowed, actor, false)
	assertRoute(allowed, other, false)
	if _, err = pool.Exec(ctx, `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid AND policy_version='read:conflict'`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO delegations(id,tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,'PERFORMER',jsonb_build_object('legal_entity_id',$3::text),$6,$7,'ACTIVE',$4::uuid)`, delegationID, tenant, entity, actor, other, now.Add(-time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetVisibleForActor(identity.WithActor(ctx, person), allowed); err != nil {
		t.Fatalf("valid delegation: %v", err)
	}
	assertRoute(allowed, other, true)
	if _, err = pool.Exec(ctx, `INSERT INTO authority_grants(tenant_id,legal_entity_id,principal_id,decision_type,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'DOCUMENT.IMPORT.READ',$4,'grant:v1')`, tenant, entity, other, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	assertRoute(allowed, other, true)
	assertRoute(allowed, actor, false)
	if _, err = pool.Exec(ctx, `DELETE FROM authority_grants WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM delegations WHERE id=$1::uuid`, delegationID); err != nil {
		t.Fatal(err)
	}
	var roleID, positionID, bindingID string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&roleID, &positionID, &bindingID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,valid_from) VALUES($1::uuid,$2::uuid,'DOC_READER','Reader',$3)`, roleID, tenant, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,'READER','Reader',$4::uuid,$5)`, positionID, tenant, entity, other, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,scope,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,jsonb_build_object('legal_entity_id',$5::text),$6)`, bindingID, tenant, positionID, roleID, otherEntity, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,role_template_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type) VALUES($1::uuid,$2::uuid,$3::uuid,'PERFORMER','DOCUMENT_IMPORT',$4::uuid,100,$5,'read:role','DOCUMENT.IMPORT.READ')`, tenant, entity, roleID, denied, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetVisibleForActor(identity.WithActor(ctx, person), denied); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity role scope: %v", err)
	}
	assertRoute(denied, other, false)
	if _, err = pool.Exec(ctx, `UPDATE position_role_bindings SET scope=jsonb_build_object('legal_entity_id',$2::text) WHERE id=$1::uuid`, bindingID, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetVisibleForActor(identity.WithActor(ctx, person), denied); err != nil {
		t.Fatalf("eligible role: %v", err)
	}
	assertRoute(denied, other, true)
	if _, err = pool.Exec(ctx, `INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code,status,valid_from) VALUES($1::uuid,'NO_READER','PERFORMER','DOC_READER','ACTIVE',$2)`, tenant, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	assertRoute(denied, other, false)
	if _, err = pool.Exec(ctx, `DELETE FROM segregation_rules WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE org_positions SET legal_entity_id=NULL WHERE id=$1::uuid`, positionID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetVisibleForActor(identity.WithActor(ctx, person), denied); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unscoped role position: %v", err)
	}
	assertRoute(denied, other, false)
	if _, err = pool.Exec(ctx, `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetVisibleForActor(actorCtx, allowed); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked route: %v", err)
	}
}
