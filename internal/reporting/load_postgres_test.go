//go:build load

package reporting

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type reportLoadPostgresFixture struct {
	pool        *pgxpool.Pool
	scope       ReportScope
	tenantID    string
	entityID    string
	ownerID     string
	authorityID string
	makerID     string
	reviewerID  string
	checkerID   string
	performerID string
	now         time.Time
	highWater   time.Time
	mix         reportLoadPostgresMix
}

type reportLoadPostgresMix struct {
	activities         int
	activityOpen       int
	activityExceptions int
	programs           int
	programsAtRisk     int
	matters            int
	matterExceptions   int
}

// tryPostgresReportLoad uses the durable repository when a database is
// configured and reachable. A connection failure is not treated as a passing
// database result: the test logs the fallback layer and measures the complete
// in-process population instead.
func tryPostgresReportLoad(t *testing.T, ctx context.Context, now time.Time) bool {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Logf("REPORT_LOAD_LAYER=in_process_full_population reason=TEST_DATABASE_URL_unset")
		return false
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Logf("REPORT_LOAD_LAYER=in_process_full_population reason=postgres_pool_create_failed error=%v", err)
		return false
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Logf("REPORT_LOAD_LAYER=in_process_full_population reason=postgres_unreachable error=%v", err)
		return false
	}

	t.Cleanup(pool.Close)
	var serverVersion string
	if err := pool.QueryRow(ctx, "SELECT version()").Scan(&serverVersion); err != nil {
		t.Fatalf("read PostgreSQL version for report load test: %v", err)
	}
	t.Logf("REPORT_LOAD_LAYER=postgresql server=%q", serverVersion)

	fixture := seedReportLoadPostgres(t, ctx, pool, now)
	repository := NewPostgresRepository(pool)
	cases := reportLoadCases()
	runs := make(map[ReportDataset]ReportRun, len(cases))
	for _, testCase := range cases {
		runs[testCase.dataset] = fixture.createRun(t, ctx, testCase)
	}

	for _, testCase := range cases {
		stats := measureReportLoadPages(t, ctx, repository, fixture.scope, runs[testCase.dataset])
		logReportLoadStats(t, "postgresql", testCase, stats)
		enforceReportLoadBudget(t, "postgresql", testCase, stats)
	}
	return true
}

func seedReportLoadPostgres(t *testing.T, ctx context.Context, pool *pgxpool.Pool, now time.Time) *reportLoadPostgresFixture {
	t.Helper()
	seedStart := time.Now()
	fixture := &reportLoadPostgresFixture{
		pool: pool, now: now, highWater: now.Add(time.Hour),
		tenantID: newReportLoadID(t), entityID: newReportLoadID(t),
		ownerID: newReportLoadID(t), authorityID: newReportLoadID(t),
		makerID: newReportLoadID(t), reviewerID: newReportLoadID(t),
		checkerID: newReportLoadID(t), performerID: newReportLoadID(t),
	}
	fixture.scope = ReportScope{TenantID: fixture.tenantID, LegalEntityID: fixture.entityID}
	t.Cleanup(fixture.cleanup)

	fixture.exec(t, ctx, `INSERT INTO tenants(id,slug,name) VALUES($1,$2,$3)`,
		fixture.tenantID, "report-load-"+strings.ReplaceAll(fixture.tenantID, "-", "")[:12], "Reporting load fixture")
	fixture.exec(t, ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1,$2,'LOAD_ENTITY','Reporting load entity','NG')`,
		fixture.entityID, fixture.tenantID)
	for _, principal := range []struct {
		id, reference, name string
	}{
		{fixture.ownerID, "load-owner", "ROPA owner"},
		{fixture.authorityID, "load-authority", "ROPA authority"},
		{fixture.makerID, "load-maker", "Report maker"},
		{fixture.reviewerID, "load-reviewer", "Report reviewer"},
		{fixture.checkerID, "load-checker", "Report checker"},
		{fixture.performerID, "load-performer", "Report performer"},
	} {
		fixture.exec(t, ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1,$2,'PERSON',$3,$4)`,
			principal.id, fixture.tenantID, principal.reference, principal.name)
	}

	fixture.insertPrograms(t, ctx)
	fixture.insertMatters(t, ctx)
	fixture.insertMatterChildren(t, ctx)
	fixture.insertActivities(t, ctx)
	fixture.insertReviews(t, ctx)
	fixture.insertSummary(t, ctx)
	fixture.exec(t, ctx, `ANALYZE programs`)
	fixture.exec(t, ctx, `ANALYZE program_state_snapshots`)
	fixture.exec(t, ctx, `ANALYZE matters`)
	fixture.exec(t, ctx, `ANALYZE matter_links`)
	fixture.exec(t, ctx, `ANALYZE matter_actions`)
	fixture.exec(t, ctx, `ANALYZE verification_contracts`)
	fixture.exec(t, ctx, `ANALYZE verification_results`)
	fixture.exec(t, ctx, `ANALYZE ropa_processing_activities`)
	fixture.exec(t, ctx, `ANALYZE ropa_processing_activity_reviews`)
	fixture.exec(t, ctx, `ANALYZE ropa_register_summary`)
	fixture.verifyMix(t, ctx)

	t.Logf("REPORT_LOAD_SEED layer=postgresql activities=%d open=%d exceptions=%d programs=%d at_risk=%d matters=%d exceptions=%d elapsed=%s",
		fixture.mix.activities, fixture.mix.activityOpen, fixture.mix.activityExceptions,
		fixture.mix.programs, fixture.mix.programsAtRisk, fixture.mix.matters, fixture.mix.matterExceptions, time.Since(seedStart))
	return fixture
}

func (f *reportLoadPostgresFixture) exec(t *testing.T, ctx context.Context, query string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(ctx, query, args...); err != nil {
		t.Fatalf("PostgreSQL report load fixture statement failed: %v\n%s", err, query)
	}
}

func (f *reportLoadPostgresFixture) insertPrograms(t *testing.T, ctx context.Context) {
	t.Helper()
	f.exec(t, ctx, `
		INSERT INTO programs
		(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,owner_principal_id,authority_principal_id,
			jurisdiction,scope,effective_from,created_at,updated_at,version)
		SELECT md5('report-load-program-' || gs::text)::uuid,$1::uuid,$2::uuid,
			'PROGRAM-LOAD-' || lpad(gs::text,5,'0'),'Bank Program ' || gs::text,'ONGOING_OBLIGATION','ACTIVE',
			'Privacy Office',$3::uuid,$4::uuid,'NG','{}'::jsonb,
			$5::timestamptz - interval '2 years',$5::timestamptz - interval '2 years',
			$5::timestamptz - (gs % 365) * interval '1 hour',1
		FROM generate_series(0,$6::integer-1) AS gs`, f.tenantID, f.entityID, f.ownerID, f.authorityID, f.now, reportLoadProgramCount)

	f.exec(t, ctx, `
		INSERT INTO program_state_snapshots
		(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,trigger_type,trigger_id,generated_at,program_version,projection_version)
		SELECT md5('report-load-program-snapshot-' || gs::text)::uuid,$1::uuid,
			md5('report-load-program-' || gs::text)::uuid,
			CASE WHEN gs % 4 = 0 THEN 'AT_RISK' ELSE 'CURRENT' END,
			jsonb_build_object(
				'interpretation','CURRENT','applicability','CURRENT','control_design','CURRENT','implementation','CURRENT',
				'evidence_sufficiency','CURRENT','operating_effectiveness','CURRENT',
				'exception',CASE WHEN gs % 4 = 0 THEN 'AT_RISK' ELSE 'CURRENT' END,
				'assurance','CURRENT','deadline','CURRENT','source_quality','CURRENT'),
			CASE WHEN gs % 4 = 0 THEN jsonb_build_array(jsonb_build_object('code','OPEN_EXCEPTION','summary','A current exception needs review.')) ELSE '[]'::jsonb END,
			0,'','report-load',$2::timestamptz - (gs % 30) * interval '1 minute',1,1
		FROM generate_series(0,$3::integer-1) AS gs`, f.tenantID, f.now, reportLoadProgramCount)
}

func (f *reportLoadPostgresFixture) insertMatters(t *testing.T, ctx context.Context) {
	t.Helper()
	f.exec(t, ctx, `
		INSERT INTO matters
		(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,
			owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version)
		SELECT md5('report-load-matter-' || gs::text)::uuid,$1::uuid,$2::uuid,
			'MATTER-LOAD-' || lpad(gs::text,6,'0'),
			CASE WHEN gs % 5 = 0 THEN 'EXCEPTION' WHEN gs % 5 = 1 THEN 'OVERDUE_OBLIGATION' ELSE 'REGULATORY_CHANGE' END,
			CASE WHEN gs % 5 < 2 THEN 'ACTION_IN_PROGRESS' ELSE 'CLOSED' END,
			1 + (gs % 5),'Issue or change ' || gs::text,'Synthetic load fixture for an overdue privacy obligation.',
			'{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,
			CASE WHEN gs % 5 < 2 THEN '["An accountable response is still open."]'::jsonb ELSE '[]'::jsonb END,
			'[]'::jsonb,$3::uuid,'Privacy authority',
			CASE WHEN gs % 5 < 2 THEN $4::timestamptz - ((gs % 180) + 1) * interval '1 day' ELSE NULL END,
			CASE WHEN gs % 5 < 2 THEN NULL ELSE $4::timestamptz - interval '1 day' END,
			CASE WHEN gs % 5 < 2 THEN '' ELSE 'Outcome confirmed and record retained.' END,
			CASE WHEN gs % 5 < 2 THEN 0 ELSE 1 END,
			$4::timestamptz - interval '2 years',$4::timestamptz - (gs % 30) * interval '1 hour',1
		FROM generate_series(0,$5::integer-1) AS gs`, f.tenantID, f.entityID, f.ownerID, f.now, reportLoadMatterCount)
}

func (f *reportLoadPostgresFixture) insertMatterChildren(t *testing.T, ctx context.Context) {
	t.Helper()
	f.exec(t, ctx, `
		INSERT INTO matter_links(id,tenant_id,matter_id,program_id,relationship,created_at)
		SELECT md5('report-load-link-' || gs::text)::uuid,$1::uuid,
			md5('report-load-matter-' || (gs * 5)::text)::uuid,
			md5('report-load-program-' || gs::text)::uuid,'AFFECTS',$2::timestamptz
		FROM generate_series(0,1249) AS gs`, f.tenantID, f.now)

	f.exec(t, ctx, `
		INSERT INTO matter_actions
		(id,tenant_id,matter_id,title,description,owner_principal_id,status,due_at,created_at,updated_at,version)
		SELECT md5('report-load-action-' || gs::text)::uuid,$1::uuid,
			md5('report-load-matter-' || gs::text)::uuid,'Resolve the open privacy obligation',
			'Complete the recorded remediation and retain the outcome.',$2::uuid,'IN_PROGRESS',
			$3::timestamptz - interval '7 days',$3::timestamptz - interval '30 days',$3::timestamptz,1
		FROM generate_series(0,$4::integer-1) AS gs
		WHERE gs % 5 < 2`, f.tenantID, f.ownerID, f.now, reportLoadMatterCount)

	f.exec(t, ctx, `
		INSERT INTO verification_contracts
		(id,tenant_id,matter_id,action_id,expected_outcome,baseline,scope,threshold,observation_period_minutes,authority_principal_id,failure_response,status,created_at,updated_at,version)
		SELECT md5('report-load-contract-' || gs::text)::uuid,$1::uuid,
			md5('report-load-matter-' || gs::text)::uuid,md5('report-load-action-' || gs::text)::uuid,
			'Open obligation is resolved','{}'::jsonb,'{}'::jsonb,'{}'::jsonb,30,$2::uuid,'REOPEN','ACTIVE',
			$3::timestamptz - interval '20 days',$3::timestamptz,1
		FROM generate_series(0,$4::integer-1) AS gs
		WHERE gs % 5 < 2`, f.tenantID, f.authorityID, f.now, reportLoadMatterCount)

	f.exec(t, ctx, `
		INSERT INTO verification_results
		(id,tenant_id,matter_id,contract_id,result,observations,evidence_references,reviewer_principal_id,rationale,observed_at,created_at,reviewer_authority_principal_id)
		SELECT md5('report-load-result-' || gs::text)::uuid,$1::uuid,
			md5('report-load-matter-' || gs::text)::uuid,md5('report-load-contract-' || gs::text)::uuid,
			'INCONCLUSIVE','{}'::jsonb,'[]'::jsonb,$2::uuid,'Outcome check remains open.',
			$3::timestamptz - interval '2 days',$3::timestamptz,$4::uuid
		FROM generate_series(0,$5::integer-1) AS gs
		WHERE gs % 5 < 2`, f.tenantID, f.reviewerID, f.now, f.authorityID, reportLoadMatterCount)
}

func (f *reportLoadPostgresFixture) insertActivities(t *testing.T, ctx context.Context) {
	t.Helper()
	f.exec(t, ctx, `
		INSERT INTO ropa_processing_activities
		(id,tenant_id,legal_entity_id,code,name,description,status,purpose,lawful_basis,controller,processor,automated_decision_making,
			data_subject_categories,personal_data_categories,security_measures,retention_period,start_date,end_date,next_review_date,
			owner_principal_id,required_authority_principal_id,program_id,version,created_at,updated_at,matter_id)
		SELECT md5('report-load-activity-' || gs::text)::uuid,$1::uuid,$2::uuid,
			'PA-LOAD-' || lpad(gs::text,6,'0'),'Large-bank processing activity ' || gs::text,
			'Synthetic load fixture for a bank processing-activity register.',
			CASE WHEN gs % 20 = 0 THEN 'NEW' WHEN gs % 20 BETWEEN 1 AND 4 THEN 'CLOSED' ELSE 'OPEN' END,
			'Provide a bank service.',
			CASE WHEN gs % 20 IN (1,2,5) THEN '' ELSE CASE WHEN gs % 2 = 0 THEN 'CONSENT' ELSE 'LEGAL_OBLIGATION' END END,
			'Fidelity Bank Nigeria','Internal operations',gs % 7 = 0,
			CASE WHEN gs % 20 IN (8,9) THEN '' ELSE 'Customers; Employees' END,
			'Account; contact details','Encryption in transit and at rest','Seven years after the relationship ends',
			$3::timestamptz::date - 730,CASE WHEN gs % 20 BETWEEN 1 AND 4 THEN $3::timestamptz::date - 30 ELSE NULL END,
			($3::timestamptz + ((gs % 365) - 180) * interval '1 day')::date,
			CASE WHEN gs % 20 IN (6,7) THEN NULL ELSE $4::uuid END,$5::uuid,
			CASE WHEN gs % 5 = 0 THEN md5('report-load-program-' || (gs % $6::integer)::text)::uuid ELSE NULL END,
			1,$3::timestamptz - interval '2 years',$3::timestamptz - (gs % 365) * interval '1 hour',
			CASE WHEN gs % 10 = 0 THEN md5('report-load-matter-' || (gs % $7::integer)::text)::uuid ELSE NULL END
		FROM generate_series(0,$8::integer-1) AS gs`, f.tenantID, f.entityID, f.now, f.ownerID, f.authorityID, reportLoadProgramCount, reportLoadMatterCount, reportLoadActivityCount)
}

func (f *reportLoadPostgresFixture) insertReviews(t *testing.T, ctx context.Context) {
	t.Helper()
	f.exec(t, ctx, `
		INSERT INTO ropa_processing_activity_reviews
		(id,tenant_id,legal_entity_id,activity_id,due_date,completed_at,outcome,reviewer_principal_id,created_at)
		SELECT md5('report-load-review-' || gs::text)::uuid,$1::uuid,$2::uuid,
			md5('report-load-activity-' || gs::text)::uuid,$3::timestamptz::date - 30,
			CASE WHEN gs % 20 = 10 THEN NULL ELSE $3::timestamptz - interval '30 days' END,
			CASE WHEN gs % 20 = 10 THEN NULL ELSE 'CONFIRMED' END,$4::uuid,$3::timestamptz - interval '60 days'
		FROM generate_series(0,$5::integer-1) AS gs`, f.tenantID, f.entityID, f.now, f.reviewerID, reportLoadActivityCount)
}

func (f *reportLoadPostgresFixture) insertSummary(t *testing.T, ctx context.Context) {
	t.Helper()
	f.exec(t, ctx, `INSERT INTO ropa_register_summary
		(tenant_id,legal_entity_id,generated_at,projection_version,source_high_water,population,excluded,unknown,counts)
		VALUES($1,$2,$3,'report-load-postgres.v1',$4,$5,0,0,$6::jsonb)`, f.tenantID, f.entityID, f.now, f.highWater, reportLoadActivityCount,
		`{"total":100000,"new":5000,"open":75000,"closed":15000}`)
}

func (f *reportLoadPostgresFixture) verifyMix(t *testing.T, ctx context.Context) {
	t.Helper()
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*)::integer,
			count(*) FILTER (WHERE status='OPEN')::integer,
			count(*) FILTER (WHERE status='OPEN' AND `+ExceptionPredicateSQL+`)::integer
		FROM ropa_processing_activities a
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, f.tenantID, f.entityID).
		Scan(&f.mix.activities, &f.mix.activityOpen, &f.mix.activityExceptions); err != nil {
		t.Fatalf("verify processing activity load mix: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*)::integer,
		count(*) FILTER (WHERE overall_state='AT_RISK')::integer
		FROM program_state_snapshots WHERE tenant_id=$1::uuid`, f.tenantID).
		Scan(&f.mix.programs, &f.mix.programsAtRisk); err != nil {
		t.Fatalf("verify Program load mix: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*)::integer,
		count(*) FILTER (WHERE status NOT IN ('CLOSED','CANCELLED') AND
			(matter_type='EXCEPTION' OR (due_at IS NOT NULL AND due_at<$3::timestamptz)))::integer
		FROM matters WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, f.tenantID, f.entityID, f.highWater).
		Scan(&f.mix.matters, &f.mix.matterExceptions); err != nil {
		t.Fatalf("verify Matter load mix: %v", err)
	}
	if f.mix.activities != reportLoadActivityCount || f.mix.activityOpen != 75_000 ||
		f.mix.activityExceptions != 30_000 || f.mix.programs != reportLoadProgramCount ||
		f.mix.programsAtRisk != 1_250 || f.mix.matters != reportLoadMatterCount || f.mix.matterExceptions != 10_000 {
		t.Fatalf("unexpected PostgreSQL load mix: %#v", f.mix)
	}
}

func (f *reportLoadPostgresFixture) createRun(t *testing.T, ctx context.Context, testCase reportLoadCase) ReportRun {
	t.Helper()
	filterJSON, err := json.Marshal(testCase.filter)
	if err != nil {
		t.Fatalf("encode %s load filter: %v", testCase.dataset, err)
	}
	now := f.now
	definition := ReportDefinition{
		ID: newReportLoadID(t), TenantID: f.tenantID, LegalEntityID: f.entityID,
		Code: testCase.code, Name: testCase.name, Description: "Synthetic PostgreSQL load fixture for a governed report definition.",
		Dataset: testCase.dataset, ScopeKind: ScopeLegalEntity, Format: FormatCSV, Filter: testCase.filter,
		Status: DefinitionActive, CurrentVersion: 1, MakerID: f.makerID, CheckerID: f.checkerID, ReviewerID: f.reviewerID,
		ReviewerNote:  "Scope, filter and source boundary checked for the load fixture.",
		EffectiveFrom: timePtrForLoad(now.Add(-time.Hour)), SubmittedAt: timePtrForLoad(now.Add(-4 * time.Hour)),
		ApprovedAt: timePtrForLoad(now.Add(-3 * time.Hour)), CreatedAt: now.Add(-5 * time.Hour), UpdatedAt: now.Add(-4 * time.Hour), Version: 1,
	}
	definition.StoredChecksum = definition.Checksum()
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO report_definitions
		(id,tenant_id,legal_entity_id,code,name,description,dataset,scope_kind,scope_ref,format,filter,status,current_version,checksum,
			maker_id,checker_id,reviewer_id,reviewer_note,effective_from,submitted_at,approved_at,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,NULL,$9,$10::jsonb,'ACTIVE',1,$11,$12::uuid,$13::uuid,$14::uuid,$15,
			$16::timestamptz,$17::timestamptz,$18::timestamptz,$19::timestamptz,$20::timestamptz,1)`,
		definition.ID, definition.TenantID, definition.LegalEntityID, definition.Code, definition.Name, definition.Description,
		definition.Dataset, definition.ScopeKind, definition.Format, string(filterJSON), definition.StoredChecksum,
		definition.MakerID, definition.CheckerID, definition.ReviewerID, definition.ReviewerNote, definition.EffectiveFrom,
		definition.SubmittedAt, definition.ApprovedAt, definition.CreatedAt, definition.UpdatedAt); err != nil {
		t.Fatalf("insert %s load definition: %v", testCase.dataset, err)
	}

	boundary := SourceBoundary{
		CapturedAt: now, ProjectionVersion: "report-load-postgres.v1",
		SourceHighWater: map[string]time.Time{testCase.sourceKey: f.highWater},
		Population:      testCase.sourcePopulation, PopulationComplete: true,
	}
	boundaryJSON, err := json.Marshal(boundary)
	if err != nil {
		t.Fatalf("encode %s load source boundary: %v", testCase.dataset, err)
	}
	run := ReportRun{
		ID: newReportLoadID(t), TenantID: f.tenantID, LegalEntityID: f.entityID,
		DefinitionID: definition.ID, DefinitionVersion: definition.CurrentVersion, DefinitionCode: definition.Code,
		DefinitionChecksum: definition.StoredChecksum, ScopeKind: ScopeLegalEntity, RequestedByRef: f.performerID,
		AsOf: f.highWater, Filter: testCase.filter, Dataset: testCase.dataset, Format: FormatCSV,
		Status: RunRunning, AttemptCount: 1, CreatedAt: now, ExpiresAt: now.Add(ReportRunRetention), SourceBoundary: boundary,
	}
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO report_runs
		(id,tenant_id,legal_entity_id,definition_id,definition_version,definition_code,definition_checksum,scope_kind,scope_ref,
			requested_by_ref,as_of,source_boundary,filter,dataset,format,status,attempt_count,row_count,created_at,expires_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,NULL,$9,$10::timestamptz,$11::jsonb,$12::jsonb,$13,$14,'RUNNING',1,0,$15::timestamptz,$16::timestamptz)`,
		run.ID, run.TenantID, run.LegalEntityID, run.DefinitionID, run.DefinitionVersion, run.DefinitionCode, run.DefinitionChecksum,
		run.ScopeKind, run.RequestedByRef, run.AsOf, string(boundaryJSON), string(filterJSON), run.Dataset, run.Format, run.CreatedAt, run.ExpiresAt); err != nil {
		t.Fatalf("insert %s load run: %v", testCase.dataset, err)
	}
	return run
}

func (f *reportLoadPostgresFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// The load fixture writes only current report rows; no immutable revision
	// rows need a trigger override during cleanup.
	for _, query := range []string{
		`DELETE FROM report_runs WHERE tenant_id=$1::uuid`,
		`DELETE FROM report_definitions WHERE tenant_id=$1::uuid`,
		`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM verification_results WHERE tenant_id=$1::uuid`,
		`DELETE FROM verification_contracts WHERE tenant_id=$1::uuid`,
		`DELETE FROM matter_actions WHERE tenant_id=$1::uuid`,
		`DELETE FROM matter_links WHERE tenant_id=$1::uuid`,
		`DELETE FROM program_state_snapshots WHERE tenant_id=$1::uuid`,
		`DELETE FROM ropa_processing_activity_reviews WHERE tenant_id=$1::uuid`,
		`DELETE FROM ropa_processing_activities WHERE tenant_id=$1::uuid`,
		`DELETE FROM ropa_register_summary WHERE tenant_id=$1::uuid`,
		`DELETE FROM matters WHERE tenant_id=$1::uuid`,
		`DELETE FROM programs WHERE tenant_id=$1::uuid`,
		`DELETE FROM legal_entities WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		_, _ = f.pool.Exec(ctx, query, f.tenantID)
	}
}
