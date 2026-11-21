//go:build postgres && postgresintegration

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresQuestionTestHTTPAcceptsTenantAliases(t *testing.T) {
	dburl := os.Getenv("TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dburl)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	unique, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "question_http_" + strings.ReplaceAll(unique, "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, e := admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`); e != nil {
			t.Error(e)
		}
	}()
	config, err := pgxpool.ParseConfig(dburl)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `CREATE TABLE tenants(id uuid PRIMARY KEY,slug text UNIQUE NOT NULL);
 CREATE TABLE legal_entities(id uuid PRIMARY KEY,tenant_id uuid NOT NULL REFERENCES tenants(id));
 CREATE TABLE principals(id uuid PRIMARY KEY,tenant_id uuid NOT NULL REFERENCES tenants(id));
 CREATE TABLE monitoring_form_templates (id uuid, tenant_id uuid, legal_entity_id uuid, program_id uuid, origin_type text, origin_id uuid, code text, name text, purpose text, owner_principal_id uuid, responsible_team text, approved_uses text[], tags text[], jurisdiction text, industry text, sensitivity text, scoring_mode text, score_profile jsonb, next_review_at timestamptz, starter_catalog_code text, starter_catalog_version bigint, presentation jsonb, sections jsonb, fields jsonb, status text, is_current boolean, effective_from timestamptz, effective_until timestamptz, version bigint, created_by uuid, submitted_by uuid, approved_by uuid, rejected_by uuid, created_at timestamptz, updated_at timestamptz);`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/000142_api_question_test_audit.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	const tenant = "11111111-1111-7111-8111-111111111111"
	const entity = "22222222-2222-7222-8222-222222222222"
	const person = "33333333-3333-7333-8333-333333333333"
	const form = "44444444-4444-7444-8444-444444444444"
	_, err = pool.Exec(ctx, `INSERT INTO tenants VALUES($1,'bank'); INSERT INTO legal_entities VALUES($2,$1); INSERT INTO principals VALUES($3,$1); INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,jurisdiction,industry,sensitivity,scoring_mode,score_profile,presentation,sections,fields,status,is_current,version,created_at,updated_at) VALUES($4,$1,$2,'F','Form','','','{}','{}','','','','','{}','{}','[]','[{"id":"q","type":"yes_no"}]','DRAFT',false,1,now(),now())`, pgx.QueryExecModeSimpleProtocol, tenant, entity, person, form)
	if err != nil {
		t.Fatal(err)
	}
	service := monitoring.NewService(monitoring.NewPostgresRepository(pool), nil)
	handler := validQuestionHandler(&questionAuditStub{}, &questionTesterStub{})
	handler.forms = service
	handler.audit = sourceaccess.NewPostgresQuestionTestAudit(pool)
	handler.tester = sourceaccess.NewQuestionTester()
	for _, tenantRef := range []string{"bank", tenant} {
		actor := identity.Actor{TenantID: tenantRef, LegalEntityID: entity, PrincipalID: person, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"mode":"SAMPLE","result_type":"BOOLEAN","sample":{"result":true}}`))
		req.SetPathValue("id", form)
		req.SetPathValue("version", "1")
		req.SetPathValue("field_id", "q")
		req = req.WithContext(identity.WithActor(req.Context(), actor))
		out := httptest.NewRecorder()
		handler.serve(out, req)
		if out.Code != http.StatusOK || !strings.Contains(out.Body.String(), `"outcome":"VALID"`) {
			t.Fatalf("tenant %s: status=%d body=%s", tenantRef, out.Code, out.Body.String())
		}
	}
	var attempts, outcomes int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM api_question_test_attempts), (SELECT count(*) FROM api_question_test_outcomes)`).Scan(&attempts, &outcomes); err != nil || attempts != 2 || outcomes != 2 {
		t.Fatalf("attempts=%d outcomes=%d err=%v", attempts, outcomes, err)
	}
}
