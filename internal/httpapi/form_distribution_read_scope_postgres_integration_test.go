//go:build postgres && postgresintegration

package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDistributionReadRoutesDoNotExposeOtherOwners(t *testing.T) {
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
	makeID := func() string {
		v, e := id.NewUUIDv7()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	tenant, entity, owner, viewer, form, subject, distribution := makeID(), makeID(), makeID(), makeID(), makeID(), makeID(), makeID()
	now := time.Now().UTC()
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$8,'Scoped read test'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'BANK','Bank','NG',$9); INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($3::uuid,$1::uuid,'PERSON','Owner','ACTIVE',$9),($4::uuid,$1::uuid,'PERSON','Other','ACTIVE',$9); INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,presentation,sections,fields,status,is_current,effective_from,version,created_by,created_at,updated_at) VALUES($5::uuid,$1::uuid,$2::uuid,'SCOPE','Scoped form','Review','{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,'[{"id":"general","title":"General"}]'::jsonb,'[{"id":"q","section_id":"general","label":"Question","type":"short_text","required":true}]'::jsonb,'ACTIVE',true,$9,1,$3::uuid,$9,$9); INSERT INTO capture_form_distributions(id,tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,title,purpose,access_policy,status,deadline,route_expires_at,created_by,version,created_at,updated_at) VALUES($7::uuid,$1::uuid,$2::uuid,$5::uuid,1,'PROGRAM',$6::uuid,'Restricted sample','Review','DIRECT_LINK_EMAIL_OTP','OPEN',$10,$10,$3::uuid,1,$9,$9)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, owner, viewer, form, subject, distribution, "scope-read-"+tenant, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenant)
	service := evidence.NewDistributionService(evidence.NewPostgresDistributionStore(evidence.NewPostgresRepository(pool), nil))
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory", Identity: identity.NewDevelopmentAuthenticator(tenant, viewer, entity, "GRC_ADMIN"), FormDistributions: service})
	for _, path := range []string{"/api/v1/forms/distributions", "/api/v1/forms/distributions/" + distribution, "/api/v1/forms/distributions/" + distribution + "/responses"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "Restricted sample") {
			t.Errorf("%s status=%d body=%s", path, w.Code, w.Body.String())
		}
	}
}
