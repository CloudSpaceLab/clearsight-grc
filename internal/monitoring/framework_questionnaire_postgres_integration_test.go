//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPostgresFrameworkQuestionnaireSelection(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	uuid := func() string {
		t.Helper()
		value, err := id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	// Own every fixture; never depend on demo seeds or another package's data.
	tenant, entity, program, owner, checker, form1, form2 := uuid(), uuid(), uuid(), uuid(), uuid(), uuid(), uuid()
	edition, adoption := uuid(), uuid()
	const version1, version2 int64 = 1, 1
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'questionnaire-'||$1,'Questionnaire test')`, tenant)
	defer func() {
		for _, table := range []string{"framework_adoption_questionnaire_events", "framework_adoption_questionnaires", "program_framework_adoptions", "framework_starter_requirement_links", "monitoring_form_templates", "programs", "principals", "legal_entities"} {
			if _, err := pool.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id=$1::uuid", tenant); err != nil {
				t.Error(err)
			}
		}
		if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenant); err != nil {
			t.Error(err)
		}
	}()
	mustExec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'TEST','Questionnaire entity','NG')`, entity, tenant)
	for _, person := range []string{owner, checker} {
		mustExec(`INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($1::uuid,$2::uuid,'PERSON','Questionnaire actor','ACTIVE')`, person, tenant)
	}
	mustExec(`INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,scope,effective_from,version,owner_principal_id,authority_principal_id) VALUES($1::uuid,$2::uuid,$3::uuid,'QUESTIONNAIRE','Questionnaire Program','COMPLIANCE','ACTIVE','Compliance','{}',clock_timestamp(),1,$4::uuid,$5::uuid)`, program, tenant, entity, owner, checker)
	for _, form := range []string{form1, form2} {
		mustExec(`INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,fields,status,is_current,effective_from,version,created_by,approved_by) VALUES($1::uuid,$2::uuid,$3::uuid,'QUESTIONNAIRE-'||$1,'Questionnaire Form','Test governed form linkage.','[{"id":"answer","label":"Answer","type":"short_text","required":true}]','ACTIVE',true,clock_timestamp()-interval '1 day',1,$4::uuid,$5::uuid)`, form, tenant, entity, owner, checker)
	}

	code := "TEST-" + strings.ToUpper(edition[len(edition)-8:])
	now := time.Now().UTC()
	_, err = pool.Exec(ctx, `INSERT INTO framework_catalog_editions(id,publisher_code,framework_code,edition_code,title,jurisdiction,source_url,source_sha256,source_published_on,license_basis,release_sha256,approval_reference) VALUES($1::uuid,'EXAMPLE',$2,'1','Test edition','GLOBAL','https://example.org/test',$3,'2026-01-01','CITATIONS_ONLY',$3,'isolated test fixture')`, edition, code, fmt.Sprintf("%x", sha256.Sum256([]byte(edition))))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO program_framework_adoptions(id,tenant_id,legal_entity_id,program_id,edition_id,publisher_code,framework_code,edition_code,status,proposed_by,reviewed_by,rationale,proposed_at,reviewed_at,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,'EXAMPLE',$6,'1','ACTIVE',$7::uuid,$8::uuid,'Isolated test',$9,$9,$9)`, adoption, tenant, entity, program, edition, code, owner, checker, now)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	input := FrameworkQuestionnaire{ID: uuid(), TenantID: tenant, LegalEntityID: entity, ProgramID: program, AdoptionID: adoption, FormID: form1, FormVersion: version1, LinkedBy: owner, LinkedAt: now.Add(time.Second)}
	first, err := repo.SetFrameworkQuestionnaire(ctx, input, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.FormID != form1 || first.Version != 1 {
		t.Fatalf("invalid first link: %+v", first)
	}
	input.ID = uuid()
	input.FormID = form2
	input.FormVersion = version2
	input.LinkedAt = now.Add(2 * time.Second)
	if _, err = repo.SetFrameworkQuestionnaire(ctx, input, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale change: %v", err)
	}
	updated, err := repo.SetFrameworkQuestionnaire(ctx, input, first.Version)
	if err != nil {
		t.Fatal(err)
	}
	if updated.FormID != form2 || updated.Version != 2 {
		t.Fatalf("invalid replacement: %+v", updated)
	}
	if _, err = repo.SetFrameworkQuestionnaire(ctx, input, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("old version accepted: %v", err)
	}
	var eventCount int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM framework_adoption_questionnaire_events WHERE link_id IN ($1::uuid,$2::uuid)`, first.ID, updated.ID).Scan(&eventCount)
	if err != nil || eventCount != 3 {
		t.Fatal(fmt.Sprintf("audit count %d: %v", eventCount, err))
	}
	_, err = repo.GetFrameworkQuestionnaire(ctx, tenant, uuid(), program, adoption)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity read: %v", err)
	}
}
