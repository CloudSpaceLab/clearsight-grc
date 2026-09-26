//go:build postgres

package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestProgramReportPageSQLUsesTheTrancheStyleKeyset(t *testing.T) {
	query := ProgramReportPageSQL("TRUE", 0)
	for _, required := range []string{
		"CASE a.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END",
		"a.updated_at", "a.id", "LIMIT $10", "p.tenant_id = $1::uuid", "p.legal_entity_id = $2::uuid",
	} {
		if !strings.Contains(query, required) {
			t.Errorf("Program page SQL is missing %q", required)
		}
	}
	pageSQL := query[strings.Index(query, "page AS MATERIALIZED"):]
	if strings.Index(pageSQL, "a.overall_state") > strings.Index(pageSQL, "LIMIT") {
		t.Fatal("Program filter was placed after the page limit")
	}
}

func TestMatterReportPageSQLPlacesVisibilityBeforeLimit(t *testing.T) {
	query := MatterReportPageSQL("TRUE", 0)
	pageSQL := query[strings.Index(query, "page AS MATERIALIZED"):]
	if end := strings.Index(pageSQL, ")\nSELECT * FROM page"); end >= 0 {
		pageSQL = pageSQL[:end]
	}
	visibility := strings.Index(pageSQL, "WHEN NOT (a.scope ? 'access') THEN true")
	order := strings.LastIndex(pageSQL, "ORDER BY")
	limit := strings.LastIndex(pageSQL, "LIMIT")
	if visibility < 0 || order < 0 || limit < 0 || !(visibility < order && order < limit) {
		t.Fatalf("Matter visibility is not before ORDER BY/LIMIT: visibility=%d order=%d limit=%d\n%s", visibility, order, limit, query)
	}
	for _, required := range []string{
		"jsonb_typeof(a.scope->'access')<>'string'",
		"jsonb_typeof(a.scope->'allowed_principal_ids')<>'array'",
		"jsonb_array_elements_text(a.scope->'allowed_principal_ids') AS nonblank",
	} {
		if !strings.Contains(query, required) {
			t.Errorf("Matter visibility SQL is missing %q", required)
		}
	}
}

func TestProgramReportPageAppliesScopeBeforeTheLimit(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	visible := fixture.insertProgram(t, fixture.entityAID, "PROGRAM-VISIBLE", 3, fixture.now, "NG", `[]`)
	fixture.insertProgram(t, fixture.entityBID, "PROGRAM-HIDDEN", 3, fixture.now.Add(time.Minute), "NG", `[]`)
	definition, _ := fixture.proposal(t, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != visible || page.Rows[0].Values["legal_entity_id"] != fixture.entityAID {
		t.Fatalf("Program page escaped its legal entity: %#v", page.Rows)
	}
}

func TestMatterReportPageAppliesVisibilityBeforeTheLimit(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	hiddenUnsupported := fixture.insertMatter(t, fixture.entityAID, "MATTER-UNSUPPORTED", 5, "REGULATORY_CHANGE", `{"access":"PARTNER"}`, time.Now().UTC().Add(-time.Hour))
	hiddenNonArray := fixture.insertMatter(t, fixture.entityAID, "MATTER-NON-ARRAY", 4, "REGULATORY_CHANGE", `{"access":"RESTRICTED","allowed_principal_ids":"not-an-array"}`, time.Now().UTC().Add(-time.Hour))
	hiddenBlank := fixture.insertMatter(t, fixture.entityAID, "MATTER-BLANK-LIST", 3, "REGULATORY_CHANGE", `{"access":"RESTRICTED","allowed_principal_ids":["  ",""]}`, time.Now().UTC().Add(-time.Hour))
	visible := fixture.insertMatter(t, fixture.entityAID, "MATTER-VISIBLE", 1, "REGULATORY_CHANGE", `{"access":"INTERNAL"}`, time.Now().UTC().Add(-time.Hour))
	definition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != visible {
		t.Fatalf("malformed/restricted Matters consumed a page slot: %#v hidden=%v,%v,%v", page.Rows, hiddenUnsupported, hiddenNonArray, hiddenBlank)
	}
}

func TestMatterReportPageHidesUnsupportedAccessBeforeLimit(t *testing.T) {
	assertMalformedMatterHiddenBeforeLimit(t, `{"access":"PARTNER"}`)
}

func TestMatterReportPageHidesNonArrayAllowListBeforeLimit(t *testing.T) {
	assertMalformedMatterHiddenBeforeLimit(t, `{"access":"RESTRICTED","allowed_principal_ids":"not-an-array"}`)
}

func TestMatterReportPageHidesBlankPrincipalListBeforeLimit(t *testing.T) {
	assertMalformedMatterHiddenBeforeLimit(t, `{"access":"RESTRICTED","allowed_principal_ids":["  ",""]}`)
}

func assertMalformedMatterHiddenBeforeLimit(t *testing.T, scope string) {
	t.Helper()
	fixture := newReportingPostgresFixture(t)
	hidden := fixture.insertMatter(t, fixture.entityAID, "MALFORMED-HIDDEN", 5, "EXCEPTION", scope, fixture.now)
	visible := fixture.insertMatter(t, fixture.entityAID, "MALFORMED-VISIBLE", 1, "EXCEPTION", `{"access":"INTERNAL"}`, fixture.now)
	definition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != visible || page.Rows[0].ID == hidden {
		t.Fatalf("malformed access policy consumed a page slot: rows=%#v hidden=%s visible=%s", page.Rows, hidden, visible)
	}
}

func TestMatterReportPageNeverReturnsAnotherLegalEntitysMatter(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	fixture.insertMatter(t, fixture.entityBID, "MATTER-OTHER-ENTITY", 5, "REGULATORY_CHANGE", `{"access":"INTERNAL"}`, time.Now().UTC())
	visible := fixture.insertMatter(t, fixture.entityAID, "MATTER-VISIBLE-ENTITY", 1, "REGULATORY_CHANGE", `{"access":"INTERNAL"}`, time.Now().UTC())
	definition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != visible {
		t.Fatalf("Matter page escaped its legal entity: %#v", page.Rows)
	}
}

func TestProgramReportCarriesItsCalculatedStateAndVersion(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	reasons := make([]map[string]any, 8)
	for index := range reasons {
		reasons[index] = map[string]any{"code": fmt.Sprintf("REASON-%d", index), "summary": fmt.Sprintf("Reason %d", index)}
	}
	programID := fixture.insertProgramWithSnapshot(t, fixture.entityAID, "PROGRAM-STATE", 11, 2, 1, "AT_RISK", reasons)
	definition, _ := fixture.proposal(t, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != programID {
		t.Fatalf("Program state row = %#v", page.Rows)
	}
	values := page.Rows[0].Values
	if values["overall_state"] != "AT_RISK" || values["assessed_program_version"] != int64(1) || values["projection_version"] != int64(11) || values["projection_stale"] != true {
		t.Fatalf("Program version/state fields = %#v", values)
	}
	if values["reasons_omitted"] != 2 || values["reasons_total"] != 8 {
		t.Fatalf("Program reasons omitted/total = %#v", values)
	}
}

func TestProgramReportIgnoresRetiredMatterLinks(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	programID := fixture.insertProgram(t, fixture.entityAID, "PROGRAM-RETIRED-LINK", 1, fixture.now, "NG", `[]`)
	matterID := fixture.insertMatter(t, fixture.entityAID, "MATTER-RETIRED-LINK", 1, "EXCEPTION", `{"access":"INTERNAL"}`, fixture.now.Add(time.Hour))
	if _, err := fixture.pool.Exec(fixture.ctx, `INSERT INTO matter_links
		(tenant_id,matter_id,program_id,relationship,retired_at,retired_by,retirement_reason)
		VALUES($1,$2,$3,'AFFECTS',$4,$5,'relationship retired')`, fixture.tenantID, matterID, programID, fixture.now, fixture.makerID); err != nil {
		t.Fatal(err)
	}
	definition, _ := fixture.proposal(t, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].Values["has_open_matters"] != false {
		t.Fatalf("retired Matter link changed Program open-Matter state: %#v", page.Rows)
	}
}

func TestProgramReportDoesNotClaimStateGenerationWithoutSnapshot(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	programID := fixture.insertProgram(t, fixture.entityAID, "PROGRAM-NO-SNAPSHOT", 2, fixture.now, "NG", `[]`)
	definition, _ := fixture.proposal(t, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != programID {
		t.Fatalf("Program without snapshot rows=%#v", page.Rows)
	}
	if generated := page.Rows[0].Values["state_generated_at"]; generated != nil {
		t.Fatalf("Program without a state snapshot claimed a generated time: %#v", generated)
	}
}

func TestProgramReportReasonsOmitIsNotSilentlyZero(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	reasons := make([]map[string]any, 8)
	for index := range reasons {
		reasons[index] = map[string]any{"code": fmt.Sprintf("REASON-%d", index), "summary": "A stored reason"}
	}
	fixture.insertProgramWithSnapshot(t, fixture.entityAID, "PROGRAM-REASONS", 3, 1, 1, "CURRENT", reasons)
	definition, _ := fixture.proposal(t, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	values := page.Rows[0].Values
	storedReasons, _ := values["reasons"].([]any)
	if values["reasons_omitted"] != 2 || len(storedReasons) != 6 {
		t.Fatalf("Program truncated reasons were presented as complete: %#v", values)
	}
}

func TestMatterReportReasonsOmittedIsNotSilentlyZero(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	fixture.insertMatterWithFacts(t, fixture.entityAID, "MATTER-REASONS", 1, `{"access":"INTERNAL"}`, []string{"one", "two", "three", "four", "five", "six", "seven", "eight"})
	definition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	values := page.Rows[0].Values
	if values["reasons_omitted"] != 4 || values["reasons_total"] != 10 {
		t.Fatalf("Matter truncated reasons were presented as complete: %#v", values)
	}
}

func TestMatterExceptionDatasetSelectsOpenExceptionsAndOverdueObligations(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	now := time.Now().UTC()
	openException := fixture.insertMatter(t, fixture.entityAID, "MATTER-OPEN-EXCEPTION", 5, "EXCEPTION", `{"access":"INTERNAL"}`, now.Add(time.Hour))
	overdue := fixture.insertMatter(t, fixture.entityAID, "MATTER-OVERDUE", 4, "OVERDUE_OBLIGATION", `{"access":"INTERNAL"}`, now.Add(-time.Hour))
	fixture.insertMatter(t, fixture.entityAID, "MATTER-CLOSED", 3, "EXCEPTION", `{"access":"INTERNAL"}`, now.Add(time.Hour), "CLOSED")
	fixture.insertMatter(t, fixture.entityAID, "MATTER-NOT-DUE", 2, "REGULATORY_CHANGE", `{"access":"INTERNAL"}`, now.Add(time.Hour))
	definition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(page.Rows))
	for _, row := range page.Rows {
		got[row.ID] = true
	}
	if !got[openException] || !got[overdue] || len(got) != 2 {
		t.Fatalf("Matter exception dataset = %#v, want open exception and overdue obligation", page.Rows)
	}
}

func TestProgramAndMatterPagesAreKeysetStableAcrossPageBoundaries(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	programIDs := make([]string, 0, 5)
	for index := 0; index < 5; index++ {
		programIDs = append(programIDs, fixture.insertProgram(t, fixture.entityAID, fmt.Sprintf("PROGRAM-KEY-%d", index), int64(5-index), fixture.now.Add(-time.Duration(5-index)*time.Second), "NG", `[]`))
	}
	matterIDs := make([]string, 0, 5)
	for index := 0; index < 5; index++ {
		matterIDs = append(matterIDs, fixture.insertMatter(t, fixture.entityAID, fmt.Sprintf("MATTER-KEY-%d", index), 5-index, "EXCEPTION", `{"access":"INTERNAL"}`, fixture.now.Add(time.Duration(index)*time.Minute)))
	}
	programDefinition, _ := fixture.proposal(t, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	programRun := fixture.createRun(t, programDefinition, DatasetPrograms, ScopeLegalEntity, "", emptyReportFilter())
	matterDefinition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	matterRun := fixture.createRun(t, matterDefinition, DatasetMatterExceptions, ScopeLegalEntity, "", emptyReportFilter())
	assertStableProgramIDs(t, fixture, programRun, programIDs)
	assertStableMatterIDs(t, fixture, matterRun, matterIDs)
}

func assertStableProgramIDs(t *testing.T, fixture *reportingPostgresFixture, run ReportRun, want []string) {
	t.Helper()
	got := make(map[string]bool)
	cursor := ""
	for {
		page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range page.Rows {
			if got[row.ID] {
				t.Fatalf("Program ID %s repeated across keyset pages", row.ID)
			}
			got[row.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(got) != len(want) {
		t.Fatalf("Program keyset IDs=%d want=%d (%v)", len(got), len(want), got)
	}
}

func assertStableMatterIDs(t *testing.T, fixture *reportingPostgresFixture, run ReportRun, want []string) {
	t.Helper()
	got := make(map[string]bool)
	cursor := ""
	for {
		page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range page.Rows {
			if got[row.ID] {
				t.Fatalf("Matter ID %s repeated across keyset pages; cursor=%q rows=%#v", row.ID, cursor, page.Rows)
			}
			got[row.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(got) != len(want) {
		t.Fatalf("Matter keyset IDs=%d want=%d (%v)", len(got), len(want), got)
	}
}

func TestProgramAndMatterOwnerFiltersUseUUIDColumns(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	programOwned := fixture.insertProgram(t, fixture.entityAID, "PROGRAM-OWNER-MAKER", 1, fixture.now, "NG", `[]`)
	programOther := fixture.insertProgram(t, fixture.entityAID, "PROGRAM-OWNER-REVIEWER", 1, fixture.now.Add(-time.Second), "NG", `[]`)
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE programs SET owner_principal_id=$1::uuid WHERE id=$2::uuid`, fixture.reviewerID, programOther); err != nil {
		t.Fatal(err)
	}
	ownerFilter := &ReportFilterExpression{Kind: "condition", Field: ReportFieldOwner, Operator: "is", Value: fixture.makerID}
	programDefinition, _ := fixture.proposal(t, DatasetPrograms, ScopeLegalEntity, "", ownerFilter)
	programRun := fixture.createRun(t, programDefinition, DatasetPrograms, ScopeLegalEntity, "", ownerFilter)
	programPage, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, programRun, "", 10)
	if err != nil {
		t.Fatalf("Program owner filter: %v", err)
	}
	if len(programPage.Rows) != 1 || programPage.Rows[0].ID != programOwned {
		t.Fatalf("Program owner filter rows=%#v, want %s", programPage.Rows, programOwned)
	}

	matterOwned := fixture.insertMatter(t, fixture.entityAID, "MATTER-OWNER-MAKER", 1, "EXCEPTION", `{"access":"INTERNAL"}`, fixture.now.Add(time.Hour))
	matterOther := fixture.insertMatter(t, fixture.entityAID, "MATTER-OWNER-REVIEWER", 1, "EXCEPTION", `{"access":"INTERNAL"}`, fixture.now.Add(time.Hour))
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE matters SET owner_principal_id=$1::uuid WHERE id=$2::uuid`, fixture.makerID, matterOwned); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE matters SET owner_principal_id=$1::uuid WHERE id=$2::uuid`, fixture.reviewerID, matterOther); err != nil {
		t.Fatal(err)
	}
	matterDefinition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", ownerFilter)
	matterRun := fixture.createRun(t, matterDefinition, DatasetMatterExceptions, ScopeLegalEntity, "", ownerFilter)
	matterPage, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, matterRun, "", 10)
	if err != nil {
		t.Fatalf("Matter owner filter: %v", err)
	}
	if len(matterPage.Rows) != 1 || matterPage.Rows[0].ID != matterOwned {
		t.Fatalf("Matter owner filter rows=%#v, want %s", matterPage.Rows, matterOwned)
	}
}

func TestMatterReportProgramFilterUsesCurrentMatterLink(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	program := fixture.insertProgram(t, fixture.entityAID, "PROGRAM-LINKED", 1, fixture.now, "NG", `[]`)
	matterLinked := fixture.insertMatter(t, fixture.entityAID, "MATTER-LINKED", 1, "EXCEPTION", `{"access":"INTERNAL"}`, fixture.now.Add(time.Hour))
	matterUnlinked := fixture.insertMatter(t, fixture.entityAID, "MATTER-UNLINKED", 1, "EXCEPTION", `{"access":"INTERNAL"}`, fixture.now.Add(time.Hour))
	if _, err := fixture.pool.Exec(fixture.ctx, `INSERT INTO matter_links (tenant_id,matter_id,program_id,relationship) VALUES($1,$2,$3,'AFFECTS')`, fixture.tenantID, matterLinked, program); err != nil {
		t.Fatal(err)
	}
	filter := &ReportFilterExpression{Kind: "condition", Field: ReportFieldMatterProgram, Operator: "is", Value: program}
	definition, _ := fixture.proposal(t, DatasetMatterExceptions, ScopeLegalEntity, "", filter)
	run := fixture.createRun(t, definition, DatasetMatterExceptions, ScopeLegalEntity, "", filter)
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != matterLinked {
		t.Fatalf("Matter program filter rows=%#v, want %s and not %s", page.Rows, matterLinked, matterUnlinked)
	}
}

func TestProgramAndMatterSourceBoundaryRejectsMismatchedScope(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	cases := []struct {
		name    string
		dataset ReportDataset
		scope   ReportScopeKind
	}{
		{name: "Program Matter scope", dataset: DatasetPrograms, scope: ScopeMatter},
		{name: "Matter Program scope", dataset: DatasetMatterExceptions, scope: ScopeProgram},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			definition, _ := fixture.proposal(t, testCase.dataset, ScopeLegalEntity, "", emptyReportFilter())
			definition.ScopeKind = testCase.scope
			definition.ScopeRef = fixture.entityBID
			if _, err := fixture.repository.CaptureSourceBoundary(context.Background(), fixture.scope, definition); !errors.Is(err, ErrInvalid) {
				t.Fatalf("mismatched source-boundary scope error = %v, want ErrInvalid", err)
			}
		})
	}
}

func (f *reportingPostgresFixture) insertProgram(t *testing.T, entityID, code string, version int64, updatedAt time.Time, jurisdiction string, reasonsJSON string) string {
	t.Helper()
	id := f.newID()
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO programs
		(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,owner_principal_id,authority_principal_id,
			jurisdiction,scope,effective_from,created_at,updated_at,version)
		VALUES($1,$2,$3,$4,$5,'PRIVACY','ACTIVE','Privacy Office',$6,$7,$8,'{}'::jsonb,$9,$9,$9,$10)`,
		id, f.tenantID, entityID, code, code, f.makerID, f.authorizerID, jurisdiction, updatedAt, version); err != nil {
		t.Fatalf("insert Program fixture: %v", err)
	}
	if reasonsJSON != "" && reasonsJSON != "[]" {
		var reasons []map[string]any
		if err := json.Unmarshal([]byte(reasonsJSON), &reasons); err != nil {
			t.Fatal(err)
		}
		if _, err := f.pool.Exec(f.ctx, `INSERT INTO program_state_snapshots
			(id,tenant_id,program_id,overall_state,dimensions,reasons,generated_at,program_version,projection_version)
			VALUES($1,$2,$3,'CURRENT',$4,$5,$6,$7,$8)`, f.newID(), f.tenantID, id,
			allCurrentDimensionsJSON(), reasons, updatedAt, version, version); err != nil {
			t.Fatalf("insert Program state fixture: %v", err)
		}
	}
	return id
}

func (f *reportingPostgresFixture) insertProgramWithSnapshot(t *testing.T, entityID, code string, projectionVersion, programVersion, assessedVersion int64, state string, reasons []map[string]any) string {
	t.Helper()
	id := f.newID()
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO programs
		(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,owner_principal_id,authority_principal_id,
			jurisdiction,scope,effective_from,created_at,updated_at,version)
		VALUES($1,$2,$3,$4,$5,'PRIVACY','ACTIVE','Privacy Office',$6,$7,'NG','{}'::jsonb,$8,$8,$8,$9)`,
		id, f.tenantID, entityID, code, code, f.makerID, f.authorizerID, f.now.Add(-time.Hour), programVersion); err != nil {
		t.Fatalf("insert Program state fixture: %v", err)
	}
	reasonJSON, _ := json.Marshal(reasons)
	dimensions := allCurrentDimensionsJSON()
	if state == "AT_RISK" {
		dimensions = []byte(`{"interpretation":"CURRENT","applicability":"CURRENT","control_design":"CURRENT","implementation":"CURRENT","evidence_sufficiency":"CURRENT","operating_effectiveness":"CURRENT","exception":"AT_RISK","assurance":"CURRENT","deadline":"CURRENT","source_quality":"CURRENT"}`)
	}
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO program_state_snapshots
		(id,tenant_id,program_id,overall_state,dimensions,reasons,generated_at,program_version,projection_version)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, f.newID(), f.tenantID, id, state, dimensions, reasonJSON, f.now, assessedVersion, projectionVersion); err != nil {
		t.Fatalf("insert Program snapshot fixture: %v", err)
	}
	return id
}

func allCurrentDimensionsJSON() []byte {
	return []byte(`{"interpretation":"CURRENT","applicability":"CURRENT","control_design":"CURRENT","implementation":"CURRENT","evidence_sufficiency":"CURRENT","operating_effectiveness":"CURRENT","exception":"CURRENT","assurance":"CURRENT","deadline":"CURRENT","source_quality":"CURRENT"}`)
}

func (f *reportingPostgresFixture) insertMatter(t *testing.T, entityID, reference string, priority int, matterType, scope string, dueAt time.Time, status ...string) string {
	t.Helper()
	id := f.newID()
	matterStatus := "ASSESSMENT"
	if len(status) > 0 {
		matterStatus = status[0]
	}
	closedAt := any(nil)
	if matterStatus == "CLOSED" {
		closedAt = f.now
	}
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO matters
		(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,
			contradictions,owner_principal_id,due_at,closed_at,created_at,updated_at,version)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,NULL,$11,$12,$13,$13,1)`,
		id, f.tenantID, entityID, reference, matterType, matterStatus, priority, reference, "fixture", scope, dueAt, closedAt, f.now); err != nil {
		t.Fatalf("insert Matter fixture: %v", err)
	}
	return id
}

func (f *reportingPostgresFixture) insertMatterWithFacts(t *testing.T, entityID, reference string, priority int, scope string, reasons []string) string {
	t.Helper()
	id := f.newID()
	missingFacts, _ := json.Marshal(reasons)
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO matters
		(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,
			contradictions,created_at,updated_at,version)
		VALUES($1,$2,$3,$4,'EXCEPTION','ASSESSMENT',$5,$4,'fixture',$6::jsonb,'{}'::jsonb,$7::jsonb,'[]'::jsonb,$8,$8,1)`,
		id, f.tenantID, entityID, reference, priority, scope, missingFacts, f.now); err != nil {
		t.Fatalf("insert Matter facts fixture: %v", err)
	}
	return id
}

func TestWorkReportPageSQLKeepsVisibleWorkBeyondExceptions(t *testing.T) {
	query := WorkReportPageSQL("TRUE", 0)
	if query == "" {
		t.Fatal("full Work report query was empty")
	}
	if strings.Contains(query, MatterReportExceptionPredicateSQL) {
		t.Fatal("full Work report silently inherited the exception-only population predicate")
	}
	if !strings.Contains(query, MatterReportVisibilitySQL) {
		t.Fatal("full Work report must retain the canonical fail-closed Matter visibility predicate")
	}
	if !strings.Contains(query, "AND TRUE") {
		t.Fatal("full Work report did not replace the exception-only predicate with an unrestricted governed population")
	}
}
