//go:build postgres && postgresintegration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMatterExactReadReceiptResolvesTenantAndEntityAliases(t *testing.T) {
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
	var tenant, entity, otherEntity, principal string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &otherEntity, &principal); err != nil {
		t.Fatal(err)
	}
	slug := "matter-scope-" + strings.ReplaceAll(tenant, "-", "")
	code := "MATTER-SCOPE-" + strings.ReplaceAll(entity, "-", "")
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Matter test')`, tenant, slug); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$3::uuid,$4,'Matter entity','NG',clock_timestamp()-interval '1 hour'),($2::uuid,$3::uuid,'OTHER','Other entity','NG',clock_timestamp()-interval '1 hour')`, entity, otherEntity, tenant, code); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Reviewer','ACTIVE',clock_timestamp()-interval '1 hour')`, principal, tenant); err != nil {
		t.Fatal(err)
	}
	service := continuity.NewService(continuity.NewReliablePostgresRepository(pool))
	creator := identity.Actor{TenantID: slug, LegalEntityID: code, PrincipalID: principal, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	created, err := service.CreateMatter(identity.WithActor(ctx, creator), continuity.CreateMatterInput{TenantID: slug, LegalEntityID: code, Type: continuity.MatterControlGap, Priority: 3, Title: "Access review", Summary: "Review access", Scope: json.RawMessage(`{"access":"INTERNAL"}`), ActorID: principal})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service}}
	for _, tc := range []struct {
		name, tenant, entity string
		status               int
	}{
		{"canonical", tenant, entity, http.StatusOK},
		{"entity code", tenant, code, http.StatusOK},
		{"tenant slug and entity code", slug, code, http.StatusOK},
		{"other entity", tenant, otherEntity, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actor := identity.Actor{TenantID: tc.tenant, LegalEntityID: tc.entity, PrincipalID: principal, Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
			r := httptest.NewRequest(http.MethodGet, "/api/v1/matters/"+created.Matter.ID+"?tenant_id="+tc.tenant, nil)
			r.SetPathValue("id", created.Matter.ID)
			r = r.WithContext(identity.WithActor(r.Context(), actor))
			w := httptest.NewRecorder()
			api.getMatter(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if tc.status != http.StatusOK {
				return
			}
			var result struct {
				Matter continuity.Matter `json:"matter"`
				Scope  struct {
					TenantID       string `json:"tenant_id"`
					LegalEntityID  string `json:"legal_entity_id"`
					PrincipalID    string `json:"principal_id"`
					RecordTenantID string `json:"record_tenant_id"`
					RecordEntityID string `json:"record_legal_entity_id"`
				} `json:"scope"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Matter.TenantID != tenant || result.Matter.LegalEntityID != entity || result.Scope.TenantID != actor.TenantID || result.Scope.LegalEntityID != actor.LegalEntityID || result.Scope.PrincipalID != actor.PrincipalID || result.Scope.RecordTenantID != tenant || result.Scope.RecordEntityID != entity {
				t.Fatalf("unexpected canonical record or verified scope: %+v", result)
			}
		})
	}
}
