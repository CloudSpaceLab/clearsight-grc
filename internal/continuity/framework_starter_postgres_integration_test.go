//go:build postgres && postgresintegration

package continuity_test

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPostgresStarterSeedAdoptionIsolationAndVersionPins(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	uuid := func() string {
		v, e := platformid.NewUUIDv7()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	tenant, entity, owner, checker := uuid(), uuid(), uuid(), uuid()
	suffix := strings.ToUpper(tenant[len(tenant)-8:])
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	count := func(q string, args ...any) int {
		t.Helper()
		var n int
		if e := pool.QueryRow(ctx, q, args...).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	exec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Starter acceptance fixture')`, tenant, "starter-"+strings.ToLower(suffix))
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,$3,'Starter fixture entity','NG')`, entity, tenant, "STARTER-"+suffix)
	if n := count(`SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid`, tenant); n != 8 {
		t.Fatalf("new entity forms=%d, want 8", n)
	}
	if n := count(`SELECT count(*) FROM framework_starter_requirement_links WHERE tenant_id=$1::uuid`, tenant); n != 35 {
		t.Fatalf("starter group links=%d, want 35", n)
	}
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$3::uuid,'PERSON','Starter owner'),($2::uuid,$3::uuid,'PERSON','Independent test checker')`, owner, checker, tenant)
	var unit string
	if err = pool.QueryRow(ctx, `INSERT INTO organization_scopes(tenant_id,legal_entity_id,code,name,kind,department_path,origin) VALUES($1::uuid,$2::uuid,'SECURITY','Information Security','DEPARTMENT',ARRAY['Information Security'],'MANAGED') RETURNING id::text`, tenant, entity).Scan(&unit); err != nil {
		t.Fatal(err)
	}

	// Validate every installed questionnaire against the actual native Forms contract.
	rows, e := pool.Query(ctx, `SELECT code,presentation,sections,fields FROM form_starter_templates WHERE code LIKE 'FRAMEWORK_%' AND enabled`)
	if e != nil {
		t.Fatal(e)
	}
	packs := 0
	for rows.Next() {
		var code string
		var presentation, sections, fields []byte
		if e = rows.Scan(&code, &presentation, &sections, &fields); e != nil {
			t.Fatal(e)
		}
		c := formcontract.Contract{ScoringMode: formcontract.ScoringNone}
		if e = json.Unmarshal(presentation, &c.Presentation); e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(sections, &c.Sections); e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(fields, &c.Fields); e != nil {
			t.Fatal(e)
		}
		if _, e = formcontract.Normalize(c); e != nil {
			t.Fatalf("%s native Forms contract: %v", code, e)
		}
		packs++
	}
	rows.Close()
	if packs != 8 {
		t.Fatalf("native starter packs=%d", packs)
	}
	catalog := frameworkcatalog.NewPostgresRepository(pool)
	source, err := catalog.StarterRequirements(ctx, tenant, entity, "ISO_27001_2022")
	if err != nil || len(source) != 5 {
		t.Fatal("ISO pack", err, len(source))
	}
	formID := source[0].FormID
	// Explicit test-only independent Forms approval fixture. The live demo exercises
	// these transitions through the app; this fixture focuses on atomic adoption.
	exec(`INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,status,is_current,effective_from,version,created_by,submitted_by,approved_by,starter_catalog_code,starter_catalog_version)
 SELECT id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,'ACTIVE',true,clock_timestamp(),3,$2::uuid,$2::uuid,$3::uuid,starter_catalog_code,starter_catalog_version FROM monitoring_form_templates WHERE id=$1::uuid AND version=1`, formID, owner, checker)
	exec(`UPDATE framework_starter_requirement_links SET form_template_version=3 WHERE tenant_id=$1::uuid AND template_code='ISO_27001_2022'`, tenant)
	source, err = catalog.StarterRequirements(ctx, tenant, entity, "ISO_27001_2022")
	if err != nil {
		t.Fatal(err)
	}
	for i := range source {
		source[i].Decision = "KEEP"
		if source[i].Code == "PRIVILEGED_ACCESS" {
			source[i].Decision = "TAILOR"
			source[i].Statement = "Review mobile and internet banking administrators every quarter."
			source[i].Reason = "Digital banking scope"
		}
	}
	c := continuity.StarterConfiguration{OwnerID: owner, RespondentScopeIDs: []string{unit}, Requirements: source}
	sys := continuity.WithTrustedSystemEntityScope(ctx, tenant, entity)
	service := continuity.NewService(continuity.NewCurrentPostgresRepository(pool))
	service.ConfigureFrameworkEditionLookup(catalog)
	create := func(code string) continuity.ProgramAggregate {
		v, e := service.CreateProgram(sys, continuity.CreateProgramInput{TenantID: tenant, LegalEntityID: entity, Code: code, Name: "Digital Banking fixture", Type: "COMPLIANCE", OwningFunction: "Security", OwnerPrincipalID: owner, AuthorityPrincipalID: checker, EffectiveFrom: time.Now().UTC(), ActorID: owner})
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	for _, pack := range []struct {
		code         string
		requirements int
	}{
		{"NIST_CSF_2", 6},
		{"PCI_DSS_4_0_1", 4},
		{"NDPA_2023", 4},
		{"UK_DPA_2018", 4},
		{"DORA_2022_2554", 4},
		{"SOC_2", 4},
	} {
		t.Run(pack.code, func(t *testing.T) {
			requirements, err := catalog.StarterRequirements(ctx, tenant, entity, pack.code)
			if err != nil || len(requirements) != pack.requirements {
				t.Fatalf("starter requirements: got %d, want %d: %v", len(requirements), pack.requirements, err)
			}
			formID := requirements[0].FormID
			if formID == "" {
				t.Fatal("starter has no native questionnaire")
			}
			exec(`INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,status,is_current,effective_from,version,created_by,submitted_by,approved_by,starter_catalog_code,starter_catalog_version)
     SELECT id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,'ACTIVE',true,clock_timestamp(),3,$2::uuid,$2::uuid,$3::uuid,starter_catalog_code,starter_catalog_version
     FROM monitoring_form_templates WHERE id=$1::uuid AND version=1`, formID, owner, checker)
			exec(`UPDATE framework_starter_requirement_links SET form_template_version=3 WHERE tenant_id=$1::uuid AND template_code=$2`, tenant, pack.code)
			requirements, err = catalog.StarterRequirements(ctx, tenant, entity, pack.code)
			if err != nil || len(requirements) != pack.requirements {
				t.Fatalf("load approved questions: %v", err)
			}
			for i := range requirements {
				requirements[i].Decision = "KEEP"
			}
			program := create(pack.code + "-" + suffix)
			input := continuity.ProposeFrameworkAdoptionInput{TenantID: tenant, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ActorID: owner, Rationale: "Verify the scoped bank starter workflow", TemplateCode: pack.code, Configuration: &continuity.StarterConfiguration{OwnerID: owner, RespondentScopeIDs: []string{unit}, Requirements: requirements}}
			draft, err := service.SaveFrameworkAdoptionDraft(sys, input)
			if err != nil {
				t.Fatal("save starter draft", err)
			}
			input.ExpectedVersion = draft.Program.Version
			pending, err := service.ProposeFrameworkAdoption(sys, input)
			if err != nil {
				t.Fatal("submit starter adoption", err)
			}
			adoption := pending.FrameworkAdoptions[0].ID
			approved, err := service.ApproveFrameworkAdoption(sys, continuity.DecideFrameworkAdoptionInput{TenantID: tenant, ProgramID: program.Program.ID, ExpectedVersion: pending.Program.Version, ActorID: checker, AdoptionID: adoption, Rationale: "Independent fixture approval"})
			if err != nil || len(approved.Requirements) != pack.requirements {
				t.Fatalf("approve adoption: %v", err)
			}
			if got := count(`SELECT count(DISTINCT group_id) FROM program_framework_requirement_questions WHERE adoption_id=$1::uuid AND form_template_id=$2::uuid AND form_template_version=3`, adoption, formID); got != pack.requirements {
				t.Fatalf("pinned question groups: got %d, want %d", got, pack.requirements)
			}
		})
	}
	p := create("DB-" + suffix)
	input := continuity.ProposeFrameworkAdoptionInput{TenantID: tenant, ProgramID: p.Program.ID, ExpectedVersion: p.Program.Version, ActorID: owner, Rationale: "Test scoped ISO adoption", TemplateCode: "ISO_27001_2022", Configuration: &c}
	draft, err := service.SaveFrameworkAdoptionDraft(sys, input)
	if err != nil {
		t.Fatal("save draft", err)
	}
	input.ExpectedVersion = draft.Program.Version
	pending, err := service.ProposeFrameworkAdoption(sys, input)
	if err != nil {
		t.Fatal("propose", err)
	}
	adoptionID := pending.FrameworkAdoptions[0].ID
	approved, err := service.ApproveFrameworkAdoption(sys, continuity.DecideFrameworkAdoptionInput{TenantID: tenant, ProgramID: p.Program.ID, ExpectedVersion: pending.Program.Version, ActorID: checker, AdoptionID: adoptionID, Rationale: "Independent test review"})
	if err != nil {
		t.Fatal("atomic approval", err)
	}
	if len(approved.Requirements) != 5 {
		t.Fatalf("native requirements=%d", len(approved.Requirements))
	}
	if n := count(`SELECT count(*) FROM program_framework_requirement_questions WHERE adoption_id=$1::uuid AND form_template_version=3`, adoptionID); n != 5 {
		t.Fatalf("native question links=%d", n)
	}
	for _, r := range approved.Requirements {
		if r.Modality != "MUST" {
			t.Fatalf("unsupported modality %q", r.Modality)
		}
	}
	// Introduce a new native draft without moving configured links.
	exec(`INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,status,is_current,version) SELECT id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,'DRAFT',false,4 FROM monitoring_form_templates WHERE id=$1::uuid AND version=3`, formID)
	source, err = catalog.StarterRequirements(ctx, tenant, entity, "ISO_27001_2022")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range source {
		if r.FormVersion != 3 {
			t.Fatal("new Forms version repointed a link")
		}
		if r.Code == "PRIVILEGED_ACCESS" && strings.Contains(r.Statement, "every quarter") {
			t.Fatal("Program wording leaked into library")
		}
	}
	second := create("DB-SECOND-" + suffix)
	for i := range source {
		source[i].Decision = "KEEP"
	}
	input.ProgramID = second.Program.ID
	input.ExpectedVersion = second.Program.Version
	input.Configuration = &continuity.StarterConfiguration{OwnerID: owner, RespondentScopeIDs: []string{unit}, Requirements: source}
	if _, err = service.SaveFrameworkAdoptionDraft(sys, input); err != nil {
		t.Fatal("second Program", err)
	}

	// Staff can add a bank-defined requirement with a real pinned native group.
	bankRequirement := source[0]
	bankRequirement.Code = "BANK_EXTRA_REVIEW"
	bankRequirement.Version = 0
	bankRequirement.Title = "Review banking service exceptions"
	actorContext := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner})
	added, err := catalog.UpdateStarterRequirement(actorContext, "ISO_27001_2022", bankRequirement, "Test bank-specific starter maintenance")
	if err != nil || added != 1 {
		t.Fatal("add bank requirement", added, err)
	}
	if _, err = catalog.UpdateStarterRequirement(actorContext, "ISO_27001_2022", bankRequirement, "Duplicate create"); err == nil {
		t.Fatal("duplicate requirement accepted")
	}
	exec(`UPDATE framework_starter_requirement_links SET statement='Staff maintained bank wording',version=version+1 WHERE tenant_id=$1::uuid AND template_code='ISO_27001_2022' AND requirement_code='GOV'`, tenant)
	questionnaireCount := count(`SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid`, tenant)
	exec(`SELECT backfill_framework_starter_forms($1::uuid)`, entity)
	exec(`SELECT backfill_framework_starter_forms($1::uuid)`, entity)
	if n := count(`SELECT count(*) FROM framework_starter_requirement_links WHERE tenant_id=$1::uuid`, tenant); n != 36 {
		t.Fatal("rerun duplicated links")
	}
	var wording string
	if err = pool.QueryRow(ctx, `SELECT statement FROM framework_starter_requirement_links WHERE tenant_id=$1::uuid AND template_code='ISO_27001_2022' AND requirement_code='GOV'`, tenant).Scan(&wording); err != nil || wording != "Staff maintained bank wording" {
		t.Fatal("backfill overwrote staff edit", err)
	}
	if n := count(`SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid`, tenant); n != questionnaireCount {
		t.Fatalf("rerun duplicated questionnaires: %d", n)
	}
	if n := count(`SELECT count(*) FROM program_framework_requirement_questions WHERE adoption_id=$1::uuid AND form_template_version=3`, adoptionID); n != 5 {
		t.Fatal("backfill changed adoption")
	}
	hidden, err := catalog.StarterRequirements(ctx, uuid(), entity, "ISO_27001_2022")
	if err != nil || len(hidden) != 0 {
		t.Fatal("cross-tenant read", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE framework_starter_requirement_links SET group_id='not-a-real-group' WHERE tenant_id=$1::uuid AND requirement_code='GOV'`, tenant)
	if err == nil {
		t.Fatal("database accepted a nonexistent group")
	}
}
