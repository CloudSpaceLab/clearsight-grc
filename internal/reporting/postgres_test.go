//go:build postgres

package reporting

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReportPageSQLKeepsFilterArgumentsBeforeCursorArguments(t *testing.T) {
	filter, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "new"},
			{Kind: "condition", Field: ReportFieldName, Operator: "contains", Value: "customer"},
		},
	})
	if err != nil {
		t.Fatalf("normalize filter: %v", err)
	}
	fragment, args, err := ReportFilterSQL(filter, 6)
	if err != nil {
		t.Fatalf("build filter SQL: %v", err)
	}
	query := ReportPageSQL(DatasetProcessingActivities, fragment, len(args))

	for _, required := range []string{
		"a.status = $6", "a.name ILIKE", "$7 ||",
		"$8 = false", "$9::integer", "$10::date", "$11::uuid", "LIMIT $12",
	} {
		if !strings.Contains(query, required) {
			t.Errorf("ReportPageSQL is missing aligned fragment %q:\n%s", required, query)
		}
	}
	positions := regexp.MustCompile(`\$(\d+)`).FindAllStringSubmatch(query, -1)
	seen := make(map[int]bool, len(positions))
	for _, match := range positions {
		var position int
		if _, err := fmt.Sscanf(match[1], "%d", &position); err != nil {
			t.Fatalf("parse placeholder %q: %v", match[0], err)
		}
		seen[position] = true
	}
	for position := 1; position <= 12; position++ {
		if !seen[position] {
			t.Errorf("ReportPageSQL does not bind an argument at every position $%d", position)
		}
		if position > 12 && seen[position] {
			t.Errorf("ReportPageSQL unexpectedly binds $%d", position)
		}
	}
}

func TestCreateDefinitionWritesCurrentAndRevisionInOneTransaction(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	definition, revision := fixture.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	installFailingTrigger(t, fixture, "report_definition_revisions", "insert", "revision", fixture.tenantID)

	if _, err := fixture.repository.CreateDefinition(context.Background(), fixture.scope, definition, revision); err == nil {
		t.Fatal("CreateDefinition succeeded after its revision insert was forced to fail")
	}
	var definitions, revisions int
	if err := fixture.pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM report_definitions WHERE tenant_id=$1 AND id=$2),
			(SELECT count(*) FROM report_definition_revisions WHERE tenant_id=$1 AND definition_id=$2)`,
		fixture.tenantID, definition.ID).Scan(&definitions, &revisions); err != nil {
		t.Fatal(err)
	}
	if definitions != 0 || revisions != 0 {
		t.Fatalf("failed revision insert left current/revision rows: definitions=%d revisions=%d", definitions, revisions)
	}
}

func TestTransitionDefinitionRejectsAStaleExpectedVersion(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	definition, _ := fixture.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	definition, err := fixture.repository.TransitionDefinition(context.Background(), fixture.scope, definition.ID, definition.Version,
		DefinitionPendingReview, DecisionRecord{ActorID: fixture.makerID, Action: DecisionSubmit, ChecksumSeen: definition.StoredChecksum, Timestamp: fixture.now})
	if err != nil {
		t.Fatalf("submit definition: %v", err)
	}
	before, err := fixture.repository.GetDefinition(context.Background(), fixture.scope, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.repository.TransitionDefinition(context.Background(), fixture.scope, definition.ID, definition.Version,
		DefinitionReviewed, DecisionRecord{ActorID: fixture.reviewerID, Action: DecisionReview, ChecksumSeen: definition.StoredChecksum, Timestamp: fixture.now.Add(time.Second)})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale transition error = %v, want ErrConflict", err)
	}
	after, err := fixture.repository.GetDefinition(context.Background(), fixture.scope, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != before.Status || after.Version != before.Version || after.ReviewerID != "" {
		t.Fatalf("stale transition changed current row: before=%#v after=%#v", before, after)
	}
}

func TestTransitionDefinitionWritesRevisionDecisionAndOutboxTogether(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	definition, _ := fixture.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	definition = fixture.submit(t, definition)
	var outboxBefore int
	if err := fixture.pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_id=$2`, fixture.tenantID, definition.ID).Scan(&outboxBefore); err != nil {
		t.Fatal(err)
	}

	reviewed, err := fixture.repository.TransitionDefinition(context.Background(), fixture.scope, definition.ID, definition.Version,
		DefinitionReviewed, DecisionRecord{ActorID: fixture.reviewerID, Action: DecisionReview, Note: "Scope and filter checked.", ChecksumSeen: definition.StoredChecksum, Timestamp: fixture.now.Add(time.Second)})
	if err != nil {
		t.Fatalf("review definition: %v", err)
	}
	if reviewed.Status != DefinitionReviewed || reviewed.ReviewerID != fixture.reviewerID || reviewed.Version != definition.Version+1 {
		t.Fatalf("reviewed definition = %#v", reviewed)
	}
	history, err := fixture.repository.ListDefinitionHistory(context.Background(), fixture.scope, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Decision != "REVIEWED" || history[0].ReviewedBy != fixture.reviewerID || history[0].DecisionNote != "Scope and filter checked." {
		t.Fatalf("revision decision = %#v", history)
	}
	var outboxAfter int
	if err := fixture.pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_id=$2`, fixture.tenantID, definition.ID).Scan(&outboxAfter); err != nil {
		t.Fatal(err)
	}
	if outboxAfter != outboxBefore+1 {
		t.Fatalf("transition outbox rows before=%d after=%d", outboxBefore, outboxAfter)
	}
}

func TestTransitionDefinitionOutboxFailureRollsBackDefinitionRevisionAndDecision(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	definition, _ := fixture.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	definition = fixture.submit(t, definition)
	var outboxBefore int
	if err := fixture.pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_id=$2`, fixture.tenantID, definition.ID).Scan(&outboxBefore); err != nil {
		t.Fatal(err)
	}
	installFailingTrigger(t, fixture, "outbox_events", "insert", "outbox", fixture.tenantID)

	_, err := fixture.repository.TransitionDefinition(context.Background(), fixture.scope, definition.ID, definition.Version,
		DefinitionReviewed, DecisionRecord{ActorID: fixture.reviewerID, Action: DecisionReview, Note: "This decision must roll back.", ChecksumSeen: definition.StoredChecksum, Timestamp: fixture.now.Add(time.Second)})
	if err == nil {
		t.Fatal("transition succeeded after its outbox insert was forced to fail")
	}
	current, err := fixture.repository.GetDefinition(context.Background(), fixture.scope, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != DefinitionPendingReview || current.Version != definition.Version || current.ReviewerID != "" {
		t.Fatalf("definition update survived failed outbox insert: %#v", current)
	}
	history, err := fixture.repository.ListDefinitionHistory(context.Background(), fixture.scope, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Decision != "PROPOSED" || history[0].ReviewedBy != "" || history[0].DecisionNote == "This decision must roll back." {
		t.Fatalf("revision decision survived failed outbox insert: %#v", history)
	}
	var outboxAfter int
	if err := fixture.pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_id=$2`, fixture.tenantID, definition.ID).Scan(&outboxAfter); err != nil {
		t.Fatal(err)
	}
	if outboxAfter != outboxBefore {
		t.Fatalf("failed transition changed outbox count: before=%d after=%d", outboxBefore, outboxAfter)
	}
}

func TestTransitionDefinitionRejectsACrossTenantDefinitionID(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	other := fixture.otherTenant(t)
	otherDefinition, otherRevision := other.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	if _, err := other.repository.CreateDefinition(context.Background(), other.scope, otherDefinition, otherRevision); err != nil {
		t.Fatal(err)
	}

	if _, err := fixture.repository.GetDefinition(context.Background(), fixture.scope, otherDefinition.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant exact read error = %v, want ErrNotFound", err)
	}
	_, err := fixture.repository.TransitionDefinition(context.Background(), fixture.scope, otherDefinition.ID, 1,
		DefinitionPendingReview, DecisionRecord{ActorID: fixture.makerID, Action: DecisionSubmit, ChecksumSeen: otherDefinition.StoredChecksum, Timestamp: fixture.now})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant transition error = %v, want ErrNotFound", err)
	}
}

func TestDefinitionReadsNeverExposeAnotherLegalEntity(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	otherDefinition, otherRevision := fixture.otherEntityProposal(t)
	if _, err := fixture.repository.CreateDefinition(context.Background(), fixture.otherScope, otherDefinition, otherRevision); err != nil {
		t.Fatal(err)
	}
	definitions, err := fixture.repository.ListDefinitions(context.Background(), fixture.scope, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range definitions {
		if definition.ID == otherDefinition.ID || definition.LegalEntityID != fixture.scope.LegalEntityID {
			t.Fatalf("definition list leaked another legal entity: %#v", definitions)
		}
	}
	if _, err := fixture.repository.GetDefinition(context.Background(), fixture.scope, otherDefinition.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("same-tenant cross-entity read error = %v, want ErrNotFound", err)
	}
}

func TestClaimQueuedRunsLeasesExactlyOnce(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	definition, _ := fixture.proposal(t, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())

	start := make(chan struct{})
	results := make(chan []ReportRun, 2)
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for _, worker := range []string{"report-worker-a", "report-worker-b"} {
		wait.Add(1)
		go func(worker string) {
			defer wait.Done()
			<-start
			claimed, err := fixture.repository.ClaimQueuedRuns(context.Background(), fixture.scope, worker, 1)
			results <- claimed
			errs <- err
		}(worker)
	}
	close(start)
	wait.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("claim queued run: %v", err)
		}
	}
	claimedIDs := make([]string, 0, 1)
	for claimed := range results {
		for _, value := range claimed {
			claimedIDs = append(claimedIDs, value.ID)
			if value.Status != RunRunning || value.AttemptCount != 1 {
				t.Fatalf("claimed run = %#v", value)
			}
		}
	}
	sort.Strings(claimedIDs)
	if len(claimedIDs) != 1 || claimedIDs[0] != run.ID {
		t.Fatalf("two workers claimed IDs %v, want exactly [%s]", claimedIDs, run.ID)
	}
}

func TestClaimQueuedRunsDoesNotClaimATerminalRun(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	definition, _ := fixture.proposal(t, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	failed := fixture.createRun(t, definition, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	failed.Status = RunFailed
	failed.FailureCode = FailureRowLimitExceeded
	if _, err := fixture.pool.Exec(context.Background(), `UPDATE report_runs SET status='FAILED',failure_code=$3,completed_at=$4 WHERE tenant_id=$1 AND id=$2`, fixture.tenantID, failed.ID, failed.FailureCode, fixture.now); err != nil {
		t.Fatal(err)
	}

	claimed, err := fixture.repository.ClaimQueuedRuns(context.Background(), fixture.scope, "report-worker-a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 0 {
		t.Fatalf("terminal run was claimed: %#v", claimed)
	}
}

func TestCompleteRunRefusesWithoutArtefacts(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	definition, _ := fixture.proposal(t, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	claimed, err := fixture.repository.ClaimQueuedRuns(context.Background(), fixture.scope, "report-worker-a", 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim run: claimed=%#v err=%v", claimed, err)
	}
	withoutArtifacts := claimed[0]
	withoutArtifacts.Status = RunReady
	withoutArtifacts.CompletedAt = timePtr(fixture.now.Add(time.Second))
	if _, err := fixture.repository.CompleteRun(context.Background(), fixture.scope, withoutArtifacts); !errors.Is(err, ErrInvalid) {
		t.Fatalf("repository completion without artefacts error = %v, want ErrInvalid", err)
	}
	if _, err := fixture.pool.Exec(context.Background(), `
		UPDATE report_runs SET status='READY',completed_at=$3
		WHERE tenant_id=$1 AND id=$2`, fixture.tenantID, run.ID, fixture.now.Add(time.Second)); err == nil || !strings.Contains(err.Error(), "cannot be READY without artefacts") {
		t.Fatalf("database generation guard error = %v", err)
	}
	persisted, err := fixture.repository.GetRun(context.Background(), fixture.scope, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != RunRunning {
		t.Fatalf("database guard left run in %q", persisted.Status)
	}
}

func TestCaptureSourceBoundaryUsesExactScopeProjection(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	fixture.seedRopaSummary(3)
	definition, _ := fixture.proposal(t, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	boundary, err := fixture.repository.CaptureSourceBoundary(context.Background(), fixture.scope, definition)
	if err != nil {
		t.Fatalf("capture source boundary: %v", err)
	}
	if boundary.ProjectionVersion != "ropa-register-test.v1" || boundary.Population != 3 || boundary.SourceHighWater["processing_activities"] != fixture.now {
		t.Fatalf("source boundary = %#v", boundary)
	}
	if _, err := fixture.repository.CaptureSourceBoundary(context.Background(), fixture.otherScope, definition); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing second-entity projection error = %v, want ErrNotFound", err)
	}
}

func TestReportPageSQLReturnsEachRowOnceAcrossPages(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	ids := fixture.insertActivities(t, fixture.scope, 1200, activitySeedOptions{})
	definition, _ := fixture.proposal(t, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	seen := make(map[string]int, len(ids))
	cursor := ""
	for pageNumber := 0; pageNumber < 10; pageNumber++ {
		page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, cursor, 500)
		if err != nil {
			t.Fatalf("page %d: %v", pageNumber+1, err)
		}
		if len(page.Rows) > 500 {
			t.Fatalf("page %d returned %d rows", pageNumber+1, len(page.Rows))
		}
		for _, row := range page.Rows {
			seen[row.ID]++
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(seen) != len(ids) {
		t.Fatalf("paged IDs distinct=%d want=%d", len(seen), len(ids))
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("ID %s appeared %d times across pages", id, count)
		}
	}
}

func TestReportPageSQLAppliesScopeInsideThePage(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	fixture.insertActivities(t, fixture.scope, 1, activitySeedOptions{})
	otherIDs := fixture.insertActivities(t, fixture.otherScope, 3, activitySeedOptions{})
	definition, _ := fixture.proposal(t, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetProcessingActivities, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].Values["legal_entity_id"] != fixture.scope.LegalEntityID {
		t.Fatalf("scoped page = %#v", page.Rows)
	}
	for _, id := range otherIDs {
		if page.Rows[0].ID == id {
			t.Fatal("same-tenant second legal entity leaked into the report page")
		}
	}
}

func TestReportPageSQLAppliesTheFilterBeforeTheLimit(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	fixture.insertActivities(t, fixture.scope, 12, activitySeedOptions{Status: "NEW"})
	closed := fixture.insertActivities(t, fixture.scope, 1, activitySeedOptions{Status: "CLOSED", Name: "Filtered closed activity"})
	filter := &ReportFilterExpression{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "closed"}
	definition, _ := fixture.proposal(t, DatasetProcessingActivities, ScopeLegalEntity, "", filter)
	run := fixture.createRun(t, definition, DatasetProcessingActivities, ScopeLegalEntity, "", filter)
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != closed[0] {
		t.Fatalf("filter was not applied before limit: %#v", page.Rows)
	}
}

func TestExceptionDatasetReturnsOnlyExceptedActivities(t *testing.T) {
	fixture := newReportingPostgresFixture(t)
	complete := fixture.insertActivities(t, fixture.scope, 1, activitySeedOptions{Name: "Complete", LawfulBasis: "CONSENT", Subjects: "CUSTOMERS", OwnerID: fixture.makerID, ReviewOutcome: "CONFIRMED"})
	excepted := fixture.insertActivities(t, fixture.scope, 1, activitySeedOptions{Name: "Missing owner"})
	definition, _ := fixture.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	run := fixture.createRun(t, definition, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	page, err := fixture.repository.ListReportRows(context.Background(), fixture.scope, run, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != excepted[0] || page.Rows[0].ID == complete[0] {
		t.Fatalf("exception page = %#v", page.Rows)
	}
	exceptions, _ := page.Rows[0].Values["exceptions"].([]string)
	if len(exceptions) != 3 {
		t.Fatalf("exception reasons = %#v", page.Rows[0].Values["exceptions"])
	}
}

type reportingPostgresFixture struct {
	t            *testing.T
	ctx          context.Context
	pool         *pgxpool.Pool
	repository   *PostgresRepository
	scope        ReportScope
	otherScope   ReportScope
	tenantID     string
	entityAID    string
	entityBID    string
	makerID      string
	reviewerID   string
	authorizerID string
	performerID  string
	now          time.Time
}

func newReportingPostgresFixture(t *testing.T) *reportingPostgresFixture {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured; real PostgreSQL reporting repository tests were not run")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &reportingPostgresFixture{
		t: t, ctx: ctx, pool: pool, repository: NewPostgresRepository(pool),
		now: time.Now().UTC().Truncate(time.Microsecond),
	}
	fixture.tenantID = fixture.newID()
	fixture.entityAID = fixture.newID()
	fixture.entityBID = fixture.newID()
	fixture.makerID = fixture.newID()
	fixture.reviewerID = fixture.newID()
	fixture.authorizerID = fixture.newID()
	fixture.performerID = fixture.newID()
	fixture.scope = ReportScope{TenantID: fixture.tenantID, LegalEntityID: fixture.entityAID}
	fixture.otherScope = ReportScope{TenantID: fixture.tenantID, LegalEntityID: fixture.entityBID}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1,$2,$3);
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES
			($4,$1,'ENTITY-A','Entity A','NG'),($5,$1,'ENTITY-B','Entity B','NG');
		INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES
			($6,$1,'PERSON',$10,'Maker'),($7,$1,'PERSON',$11,'Reviewer'),
			($8,$1,'PERSON',$12,'Authorizer'),($9,$1,'PERSON',$13,'Performer')`,
		fixture.tenantID, "report-repo-"+fixture.tenantID[:8], "Reporting Repository Test",
		fixture.entityAID, fixture.entityBID, fixture.makerID, fixture.reviewerID, fixture.authorizerID, fixture.performerID,
		"report-maker-"+fixture.tenantID[:8], "report-reviewer-"+fixture.tenantID[:8], "report-authorizer-"+fixture.tenantID[:8], "report-performer-"+fixture.tenantID[:8]); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupReportingTenant(context.Background(), pool, fixture.tenantID)
		pool.Close()
	})
	return fixture
}

func (f *reportingPostgresFixture) newID() string {
	f.t.Helper()
	value, err := platformid.NewUUIDv7()
	if err != nil {
		f.t.Fatal(err)
	}
	return value
}

func (f *reportingPostgresFixture) proposal(t *testing.T, dataset ReportDataset, kind ReportScopeKind, reference string, filter *ReportFilterExpression) (ReportDefinition, ReportDefinitionRevision) {
	t.Helper()
	definition := ReportDefinition{
		ID: f.newID(), TenantID: f.tenantID, LegalEntityID: f.scope.LegalEntityID,
		Code: "REPORT-" + strings.ToUpper(f.newID()[:8]), Name: "Repository test report",
		Description: "Test fixture", Dataset: dataset, ScopeKind: kind, ScopeRef: reference,
		Format: FormatCSV, Filter: filter, Status: DefinitionDraft, CurrentVersion: 1,
		MakerID: f.makerID, CreatedAt: f.now, UpdatedAt: f.now, Version: 1,
	}
	definition.StoredChecksum = definition.Checksum()
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: 1, BaseVersion: 0, Dataset: dataset, ScopeKind: kind, ScopeRef: reference,
		Format: FormatCSV, Filter: cloneReportFilter(filter), Checksum: definition.StoredChecksum,
		MakerID: f.makerID, CreatedAt: f.now, Decision: "PROPOSED",
	}
	created, err := f.repository.CreateDefinition(f.ctx, f.scope, definition, revision)
	if err != nil {
		t.Fatalf("create definition fixture: %v", err)
	}
	return created, revision
}

func (f *reportingPostgresFixture) otherEntityProposal(t *testing.T) (ReportDefinition, ReportDefinitionRevision) {
	t.Helper()
	definition := ReportDefinition{
		ID: f.newID(), TenantID: f.tenantID, LegalEntityID: f.otherScope.LegalEntityID,
		Code: "REPORT-" + strings.ToUpper(f.newID()[:8]), Name: "Second entity report",
		Dataset: DatasetProcessingActivities, ScopeKind: ScopeLegalEntity, Format: FormatCSV,
		Filter: emptyReportFilter(), Status: DefinitionDraft, CurrentVersion: 1, MakerID: f.makerID,
		CreatedAt: f.now, UpdatedAt: f.now, Version: 1,
	}
	definition.StoredChecksum = definition.Checksum()
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: 1, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind, Format: definition.Format,
		Filter: emptyReportFilter(), Checksum: definition.StoredChecksum, MakerID: f.makerID, CreatedAt: f.now, Decision: "PROPOSED",
	}
	return definition, revision
}

func (f *reportingPostgresFixture) submit(t *testing.T, definition ReportDefinition) ReportDefinition {
	t.Helper()
	submitted, err := f.repository.TransitionDefinition(f.ctx, f.scope, definition.ID, definition.Version,
		DefinitionPendingReview, DecisionRecord{ActorID: f.makerID, Action: DecisionSubmit, ChecksumSeen: definition.StoredChecksum, Timestamp: f.now.Add(time.Second)})
	if err != nil {
		t.Fatalf("submit definition fixture: %v", err)
	}
	return submitted
}

func (f *reportingPostgresFixture) createRun(t *testing.T, definition ReportDefinition, dataset ReportDataset, kind ReportScopeKind, reference string, filter *ReportFilterExpression) ReportRun {
	t.Helper()
	now := f.now.Add(2 * time.Second)
	run := ReportRun{
		ID: f.newID(), TenantID: f.tenantID, LegalEntityID: definition.LegalEntityID,
		DefinitionID: definition.ID, DefinitionVersion: definition.CurrentVersion,
		DefinitionCode: definition.Code, DefinitionChecksum: definition.StoredChecksum,
		ScopeKind: kind, ScopeRef: reference, RequestedByRef: f.performerID, AsOf: now,
		Filter: cloneReportFilter(filter), Dataset: dataset, Format: FormatCSV, Status: RunQueued,
		CreatedAt: now, ExpiresAt: now.Add(ReportRunRetention),
		SourceBoundary: SourceBoundary{CapturedAt: now, ProjectionVersion: "test.v1", SourceHighWater: map[string]time.Time{"processing_activities": now}, PopulationComplete: true},
	}
	created, err := f.repository.CreateRun(f.ctx, ReportScope{TenantID: run.TenantID, LegalEntityID: run.LegalEntityID}, run)
	if err != nil {
		t.Fatalf("create run fixture: %v", err)
	}
	return created
}

func (f *reportingPostgresFixture) insertActivities(t *testing.T, scope ReportScope, count int, options activitySeedOptions) []string {
	t.Helper()
	ids := make([]string, count)
	rows := make([][]any, 0, count)
	for index := 0; index < count; index++ {
		id := f.newID()
		ids[index] = id
		status := options.Status
		if status == "" {
			status = []string{"NEW", "OPEN", "CLOSED"}[index%3]
		}
		name := options.Name
		if name == "" {
			name = fmt.Sprintf("Processing activity %04d", index)
		}
		lawfulBasis := options.LawfulBasis
		subjects := options.Subjects
		ownerID := options.OwnerID
		if options.Complete {
			lawfulBasis = "CONSENT"
			subjects = "CUSTOMERS"
			ownerID = f.makerID
		}
		var owner any
		if ownerID != "" {
			owner = ownerID
		}
		rows = append(rows, []any{scope.TenantID, scope.LegalEntityID, fmt.Sprintf("ACT-%s", strings.ReplaceAll(id[:8], "-", "")), name,
			status, "Privacy operations", lawfulBasis, "Fidelity Bank", "Vendor", false, subjects,
			"CUSTOMERS", "Encryption", "Seven years", f.now, owner, 1, f.now, f.now})
	}
	_, err := f.pool.CopyFrom(f.ctx, pgx.Identifier{"ropa_processing_activities"},
		[]string{"tenant_id", "legal_entity_id", "code", "name", "status", "purpose", "lawful_basis", "controller", "processor",
			"automated_decision_making", "data_subject_categories", "personal_data_categories", "security_measures", "retention_period",
			"start_date", "owner_principal_id", "version", "created_at", "updated_at"}, pgx.CopyFromRows(rows))
	if err != nil {
		t.Fatalf("copy processing activities: %v", err)
	}
	if options.ReviewOutcome != "" {
		if _, err := f.pool.Exec(f.ctx, `INSERT INTO ropa_processing_activity_reviews
			(id,tenant_id,legal_entity_id,activity_id,due_date,completed_at,outcome,created_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, f.newID(), scope.TenantID, scope.LegalEntityID, ids[0], f.now, f.now, options.ReviewOutcome, f.now); err != nil {
			t.Fatalf("insert processing activity review: %v", err)
		}
	}
	return ids
}

type activitySeedOptions struct {
	Status        string
	Name          string
	LawfulBasis   string
	Subjects      string
	OwnerID       string
	ReviewOutcome string
	Complete      bool
}

func (f *reportingPostgresFixture) seedRopaSummary(population int) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO ropa_register_summary
		(tenant_id,legal_entity_id,generated_at,projection_version,source_high_water,population,excluded,unknown,counts)
		VALUES($1,$2,$3,'ropa-register-test.v1',$3,$4,0,0,'{}'::jsonb)`, f.tenantID, f.entityAID, f.now, population); err != nil {
		f.t.Fatal(err)
	}
}

func (f *reportingPostgresFixture) otherTenant(t *testing.T) *reportingPostgresFixture {
	t.Helper()
	other := *f
	other.t = t
	other.pool = f.pool
	other.repository = NewPostgresRepository(f.pool)
	other.tenantID = f.newID()
	other.entityAID = f.newID()
	other.entityBID = f.newID()
	other.makerID = f.newID()
	other.reviewerID = f.newID()
	other.authorizerID = f.newID()
	other.performerID = f.newID()
	other.scope = ReportScope{TenantID: other.tenantID, LegalEntityID: other.entityAID}
	other.otherScope = ReportScope{TenantID: other.tenantID, LegalEntityID: other.entityBID}
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO tenants(id,slug,name) VALUES($1,$2,$3)`, other.tenantID, "report-other-"+other.tenantID[:8], "Other Reporting Test"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1,$2,'ENTITY-A','Other Entity','NG')`, other.entityAID, other.tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES
		($1,$5,'PERSON','other-maker','Other Maker'),($2,$5,'PERSON','other-reviewer','Other Reviewer'),
		($3,$5,'PERSON','other-authorizer','Other Authorizer'),($4,$5,'PERSON','other-performer','Other Performer')`,
		other.makerID, other.reviewerID, other.authorizerID, other.performerID, other.tenantID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupReportingTenant(context.Background(), f.pool, other.tenantID) })
	return &other
}

func installFailingTrigger(t *testing.T, fixture *reportingPostgresFixture, table, operation, suffix, tenantID string) {
	t.Helper()
	functionName := "report_test_fail_" + suffix + "_" + platformSuffix()
	triggerName := functionName
	if _, err := fixture.pool.Exec(context.Background(), fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.tenant_id=$1::uuid THEN RAISE EXCEPTION 'forced reporting repository failure';
  END IF;
  RETURN NEW;
END; $$`, functionName), tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(context.Background(), fmt.Sprintf(`CREATE TRIGGER %s BEFORE %s ON %s FOR EACH ROW EXECUTE FUNCTION %s()`,
		triggerName, operation, table, functionName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = fixture.pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s`, triggerName, table))
		_, _ = fixture.pool.Exec(context.Background(), `DROP FUNCTION IF EXISTS `+functionName+`()`)
	})
}

func platformSuffix() string {
	return strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), " ", "")
}

func cleanupReportingTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM report_runs WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM report_definition_revisions WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM report_definitions WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM ropa_processing_activity_reviews WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM ropa_processing_activities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM ropa_register_summary WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}
