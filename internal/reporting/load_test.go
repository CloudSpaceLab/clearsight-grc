//go:build load

package reporting

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

const (
	reportLoadTenantID       = "tenant-load"
	reportLoadLegalEntityID  = "entity-load"
	reportLoadActivityCount  = 100_000
	reportLoadProgramCount   = 5_000
	reportLoadMatterCount    = 25_000
	reportLoadPageSize       = 50
	reportLoadSampleCount    = 50
	reportLoadPageRepeats    = 20
	reportLoadPageBudget     = 750 * time.Millisecond
	reportLoadSourceKey      = "processing_activities"
	reportLoadProgramSource  = "programs"
	reportLoadMatterSource   = "matters"
	reportLoadDefinitionBase = 910_000_000
	reportLoadRunBase        = 920_000_000
)

// TestReportPageBudget measures the bounded report page path against the same
// 750ms budget used by the Tranche 1 register list. The fixture is deliberately
// local and in-memory: it proves the page and cursor cost for this branch, not
// PostgreSQL capacity or production latency.
func TestReportPageBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}

	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	scope := ReportScope{TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID}

	seedStart := time.Now()
	activityRows, exceptionRows, activityCounts := buildReportLoadActivities(t, ctx, now)
	programRows, programCounts := buildReportLoadPrograms(now)
	matterRows, matterCounts := buildReportLoadMatters(now)
	runtime.GC()
	if activityCounts.complete != 60_000 || activityCounts.exceptions != 40_000 || activityCounts.open != 75_000 {
		t.Fatalf("processing activity mix = %#v, want complete=60000 exceptions=40000 open=75000", activityCounts)
	}
	if programCounts.total != reportLoadProgramCount || programCounts.selected != 1_250 {
		t.Fatalf("Program population = %#v, want total=%d selected=1250", programCounts, reportLoadProgramCount)
	}
	if matterCounts.total != reportLoadMatterCount || matterCounts.selected != 10_000 {
		t.Fatalf("Matter population = %#v, want total=%d selected=10000", matterCounts, reportLoadMatterCount)
	}

	t.Logf("seeded processing activities=%d complete=%d exceptions=%d open=%d; generated programs=%d at_risk=%d; generated matters=%d overdue=%d in %s",
		reportLoadActivityCount, activityCounts.complete, activityCounts.exceptions, activityCounts.open,
		programCounts.total, programCounts.selected, matterCounts.total, matterCounts.selected,
		time.Since(seedStart))

	repository := NewMemoryRepository()
	cases := []struct {
		dataset          ReportDataset
		code             string
		name             string
		filter           *ReportFilterExpression
		rows             []ReportRow
		sourcePopulation int
		pagePopulation   int
		sourceKey        string
	}{
		{
			dataset: DatasetProcessingActivities, code: "LOAD-PROCESSING-ACTIVITIES", name: "Load processing activities",
			filter: reportLoadFilter(ReportFieldStatus, "OPEN"), rows: activityRows, sourcePopulation: reportLoadActivityCount,
			pagePopulation: len(activityRows), sourceKey: reportLoadSourceKey,
		},
		{
			dataset: DatasetProcessingActivityExceptions, code: "LOAD-PROCESSING-EXCEPTIONS", name: "Load processing exceptions",
			filter: reportLoadFilter(ReportFieldStatus, "OPEN"), rows: exceptionRows, sourcePopulation: reportLoadActivityCount,
			pagePopulation: len(exceptionRows), sourceKey: reportLoadSourceKey,
		},
		{
			dataset: DatasetPrograms, code: "LOAD-PROGRAMS", name: "Load Program population",
			filter: reportLoadFilter(ReportFieldOverallState, "AT_RISK"), rows: programRows, sourcePopulation: programCounts.total,
			pagePopulation: len(programRows), sourceKey: reportLoadProgramSource,
		},
		{
			dataset: DatasetMatterExceptions, code: "LOAD-MATTERS", name: "Load issue and change population",
			filter: reportLoadFilter(ReportFieldDueCondition, "OVERDUE"), rows: matterRows, sourcePopulation: matterCounts.total,
			pagePopulation: len(matterRows), sourceKey: reportLoadMatterSource,
		},
	}

	for index, testCase := range cases {
		run := createReportLoadRun(t, ctx, repository, scope, now, index, testCase.dataset, testCase.code, testCase.name, testCase.filter, testCase.rows, testCase.sourceKey)
		stats := measureReportLoadPages(t, ctx, repository, scope, run, testCase.pagePopulation)
		t.Logf("REPORT_PAGE_BUDGET dataset=%s source_population=%d page_population=%d page_size=%d samples=%d first_p50=%s first_p95=%s first_max=%s cursor_p50=%s cursor_p95=%s cursor_max=%s budget=%s",
			testCase.dataset, testCase.sourcePopulation, testCase.pagePopulation, reportLoadPageSize, reportLoadSampleCount,
			stats.first.p50, stats.first.p95, stats.first.max, stats.cursor.p50, stats.cursor.p95, stats.cursor.max, reportLoadPageBudget)
		if stats.first.p50 > reportLoadPageBudget || stats.first.p95 > reportLoadPageBudget ||
			stats.cursor.p50 > reportLoadPageBudget || stats.cursor.p95 > reportLoadPageBudget {
			t.Errorf("dataset %s exceeded the %s page budget: first p50=%s p95=%s, cursor p50=%s p95=%s",
				testCase.dataset, reportLoadPageBudget, stats.first.p50, stats.first.p95, stats.cursor.p50, stats.cursor.p95)
		}
	}
}

type reportLoadActivityCounts struct {
	complete   int
	exceptions int
	open       int
}

type reportLoadPopulationCounts struct {
	total    int
	selected int
}

type reportLoadDurationStats struct {
	p50 time.Duration
	p95 time.Duration
	max time.Duration
}

type reportLoadPageStats struct {
	first  reportLoadDurationStats
	cursor reportLoadDurationStats
}

func buildReportLoadActivities(t *testing.T, ctx context.Context, now time.Time) ([]ReportRow, []ReportRow, reportLoadActivityCounts) {
	t.Helper()
	repository := ropa.NewMemoryRepository()
	service := ropa.NewService(repository, nil)
	service.Now = func() time.Time { return now }
	allOpen := make([]ReportRow, 0, reportLoadActivityCount)
	exceptions := make([]ReportRow, 0, reportLoadActivityCount/2)
	counts := reportLoadActivityCounts{}

	for index := 0; index < reportLoadActivityCount; index++ {
		kind := index % 10
		reviewDate := now.AddDate(0, 0, (index%365)-120)
		startDate := now.AddDate(-1, 0, 0)
		completedAt := now.Add(-30 * 24 * time.Hour)
		input := ropa.CreateActivityInput{
			TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID,
			Code: fmt.Sprintf("PA-LOAD-%06d", index), Name: fmt.Sprintf("Large-bank processing activity %06d", index),
			Description: "Synthetic load fixture for a bank's processing-activity register.", Purpose: "Provide a bank service.",
			LawfulBasis: []string{"Contract", "Consent", "Legal obligation"}[index%3], Controller: "Fidelity Bank Nigeria",
			Processor: "Internal operations", AutomatedDecisionMaking: index%7 == 0,
			DataSubjectCategories: "Customers; Employees", PersonalDataCategories: "Account; contact details",
			SecurityMeasures: "Encryption in transit and at rest", RetentionPeriod: "Seven years after the relationship ends",
			StartDate: &startDate, NextReviewDate: &reviewDate, OwnerPrincipalID: "owner-load",
			RequiredAuthorityPrincipalID: "authority-load", ActorID: "load-seed",
			DataCategories: []ropa.DataCategory{{Category: "Account identifiers", Sensitivity: "DIRECT_PERSONAL"}},
			Recipients:     []ropa.Recipient{loadRecipient(index)}, Systems: []ropa.System{{SystemName: "Core banking platform", SystemKind: "APPLICATION"}},
		}
		if kind == 6 {
			input.LawfulBasis = ""
		}
		if kind == 7 {
			input.OwnerPrincipalID = ""
		}
		if kind == 8 {
			input.DataSubjectCategories = ""
		}
		if kind == 9 {
			input.Reviews = []ropa.Review{{ID: fmt.Sprintf("RV-LOAD-%06d", index), DueDate: now.AddDate(0, 0, 30), Outcome: "CONFIRMED", ReviewerPrincipalID: "reviewer-load"}}
		} else {
			outcome := "CONFIRMED"
			if index%2 == 1 {
				outcome = "REVISED"
			}
			input.Reviews = []ropa.Review{{ID: fmt.Sprintf("RV-LOAD-%06d", index), DueDate: reviewDate, CompletedAt: &completedAt, Outcome: outcome, ReviewerPrincipalID: "reviewer-load"}}
		}

		activity, err := service.CreateActivity(ctx, input)
		if err != nil {
			t.Fatalf("seed processing activity %d: %v", index, err)
		}
		// Keep a realistic lifecycle mix: most records are in progress, a
		// smaller complete population is closed, and a minority remains new.
		if kind < 6 && index%20 == 0 {
			endDate := now.Add(-24 * time.Hour)
			activity, err = service.TransitionActivity(ctx, ropa.TransitionActivityInput{
				TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID, ActivityID: activity.ID,
				ExpectedVersion: activity.Version, To: ropa.StatusClosed, EndDate: &endDate, ActorID: "load-seed",
			})
		} else if index%4 != 0 {
			activity, err = service.TransitionActivity(ctx, ropa.TransitionActivityInput{
				TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID, ActivityID: activity.ID,
				ExpectedVersion: activity.Version, To: ropa.StatusOpen, ActorID: "load-seed",
			})
		}
		if err != nil {
			t.Fatalf("transition processing activity %d: %v", index, err)
		}

		blockers := ExceptionBlockers(activity)
		row := processingActivityReportRow(activity, blockers)
		if len(blockers) == 0 {
			counts.complete++
		} else {
			counts.exceptions++
		}
		if activity.Status == ropa.StatusOpen {
			counts.open++
			allOpen = append(allOpen, row)
			if len(blockers) > 0 {
				exceptions = append(exceptions, row)
			}
		}
	}
	return allOpen, exceptions, counts
}

func loadRecipient(index int) ropa.Recipient {
	if index%5 == 0 {
		return ropa.Recipient{Recipient: "External settlement provider", RecipientKind: "EXTERNAL", CountryCode: "GB", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}
	}
	return ropa.Recipient{Recipient: "Internal operations", RecipientKind: "INTERNAL", IsCrossBorder: false, TransferBasis: ropa.TransferBasisNotApplicable}
}

func processingActivityReportRow(activity ropa.ProcessingActivity, blockers []ExceptionBlocker) ReportRow {
	missing := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		missing = append(missing, blocker.Missing)
	}
	var nextReview any
	if activity.NextReviewDate != nil {
		nextReview = activity.NextReviewDate.UTC()
	}
	return ReportRow{ID: activity.ID, Values: map[string]any{
		"tenant_id": activity.TenantID, "legal_entity_id": activity.LegalEntityID, "code": activity.Code, "name": activity.Name,
		"status": string(activity.Status), "purpose": activity.Purpose, "lawful_basis": activity.LawfulBasis,
		"controller": activity.Controller, "processor": activity.Processor, "automated_decision_making": activity.AutomatedDecisionMaking,
		"data_subject_categories": activity.DataSubjectCategories, "personal_data_categories": activity.PersonalDataCategories,
		"security_measures": activity.SecurityMeasures, "retention_period": activity.RetentionPeriod, "next_review_date": nextReview,
		"owner_principal_id": activity.OwnerPrincipalID, "program_id": activity.ProgramID, "version": activity.Version,
		"updated_at": activity.UpdatedAt.UTC(), "exceptions": missing,
	}}
}

func buildReportLoadPrograms(now time.Time) ([]ReportRow, reportLoadPopulationCounts) {
	rows := make([]ReportRow, 0, reportLoadProgramCount/4)
	counts := reportLoadPopulationCounts{total: reportLoadProgramCount}
	for index := 0; index < reportLoadProgramCount; index++ {
		state := "CURRENT"
		if index%4 == 0 {
			state = "AT_RISK"
		}
		updatedAt := now.Add(-time.Duration(index%365) * time.Hour)
		row := ReportRow{ID: reportLoadUUID(100_000_000 + index), Values: map[string]any{
			"tenant_id": reportLoadTenantID, "legal_entity_id": reportLoadLegalEntityID,
			"code": fmt.Sprintf("PROGRAM-LOAD-%05d", index), "name": fmt.Sprintf("Bank Program %05d", index),
			"program_type": "ONGOING_OBLIGATION", "status": "ACTIVE", "owning_function": "Risk and compliance",
			"owner_principal_id": fmt.Sprintf("owner-program-%03d", index%25), "authority_principal_id": "authority-program",
			"jurisdiction": "NG", "effective_from": now.AddDate(-2, 0, 0), "effective_until": nil,
			"created_at": now.AddDate(-2, 0, 0), "updated_at": updatedAt, "version": int64(1 + index%4),
			"assessed_program_version": int64(1 + index%4), "projection_version": int64(20 + index%7),
			"projection_stale": index%11 == 0, "overall_state": state, "has_open_matters": state == "AT_RISK",
			"state_generated_at": now.Add(-5 * time.Minute), "reasons_total": 2, "reasons": []string{"Current source and owner checks are incomplete."},
			"reasons_omitted": 0,
		}}
		if state == "AT_RISK" {
			rows = append(rows, row)
			counts.selected++
		}
	}
	return rows, counts
}

func buildReportLoadMatters(now time.Time) ([]ReportRow, reportLoadPopulationCounts) {
	rows := make([]ReportRow, 0, reportLoadMatterCount/5*2)
	counts := reportLoadPopulationCounts{total: reportLoadMatterCount}
	for index := 0; index < reportLoadMatterCount; index++ {
		overdue := index%5 < 2
		status := "CLOSED"
		matterType := "REGULATORY_CHANGE"
		var dueAt any
		var closedAt any = now.Add(-24 * time.Hour)
		exceptions := []string{}
		if overdue {
			status = "ACTION_IN_PROGRESS"
			matterType = "OVERDUE_OBLIGATION"
			dueAt = now.Add(-time.Duration(index%180+1) * 24 * time.Hour)
			closedAt = nil
			exceptions = []string{"Obligation is past its recorded due date."}
		}
		row := ReportRow{ID: reportLoadUUID(200_000_000 + index), Values: map[string]any{
			"tenant_id": reportLoadTenantID, "legal_entity_id": reportLoadLegalEntityID,
			"reference": fmt.Sprintf("MATTER-LOAD-%06d", index), "matter_type": matterType, "status": status,
			"priority": 1 + index%5, "title": fmt.Sprintf("Issue or change %06d", index), "summary": "Synthetic load fixture for an overdue privacy obligation.",
			"owner_principal_id": fmt.Sprintf("owner-matter-%03d", index%40), "required_authority": "authority-matter",
			"due_at": dueAt, "closed_at": closedAt, "closure_reason": "", "reopen_count": 0, "created_at": now.AddDate(0, 0, -index%365),
			"updated_at": now.Add(-time.Duration(index%30) * time.Hour), "version": int64(1 + index%3), "scope": map[string]any{"access": "INTERNAL"},
			"latest_verification_result": "INCONCLUSIVE", "latest_verification_at": now.Add(-48 * time.Hour), "open_action_count": 1,
			"outcome_check_count": 1, "reasons_total": len(exceptions), "reasons": exceptions, "reasons_omitted": 0, "exceptions": exceptions,
		}}
		if overdue {
			rows = append(rows, row)
			counts.selected++
		}
	}
	return rows, counts
}

func reportLoadFilter(field ReportFilterField, value string) *ReportFilterExpression {
	return &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{{Kind: "condition", Field: field, Operator: "is", Value: value}}}
}

func createReportLoadRun(t *testing.T, ctx context.Context, repository *MemoryRepository, scope ReportScope, now time.Time, index int, dataset ReportDataset, code, name string, filter *ReportFilterExpression, rows []ReportRow, sourceKey string) ReportRun {
	t.Helper()
	definitionID := reportLoadUUID(reportLoadDefinitionBase + index)
	definition := ReportDefinition{
		ID: definitionID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, Code: code, Name: name,
		Description: "Synthetic load fixture for a governed report definition.", Dataset: dataset, ScopeKind: ScopeLegalEntity,
		Format: FormatCSV, Filter: cloneReportFilter(filter), Status: DefinitionDraft, CurrentVersion: 1, MakerID: "load-maker",
		CreatedAt: now.Add(-5 * time.Hour), UpdatedAt: now.Add(-5 * time.Hour), Version: 1,
	}
	definition.StoredChecksum = definition.Checksum()
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: definition.CurrentVersion, BaseVersion: 0, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind,
		Format: definition.Format, Filter: cloneReportFilter(definition.Filter), Checksum: definition.StoredChecksum,
		MakerID: definition.MakerID, CreatedAt: now.Add(-5 * time.Hour), Decision: "PROPOSED",
	}
	if _, err := repository.CreateDefinition(ctx, scope, definition, revision); err != nil {
		t.Fatalf("create load definition %s: %v", code, err)
	}
	submitted, err := repository.TransitionDefinition(ctx, scope, definition.ID, definition.Version, DefinitionPendingReview, DecisionRecord{
		ActorID: "load-maker", Action: DecisionSubmit, ChecksumSeen: definition.StoredChecksum, Timestamp: now.Add(-4 * time.Hour),
	})
	if err != nil {
		t.Fatalf("submit load definition %s: %v", code, err)
	}
	reviewed, err := repository.TransitionDefinition(ctx, scope, definition.ID, submitted.Version, DefinitionReviewed, DecisionRecord{
		ActorID: "load-reviewer", Action: DecisionReview, ChecksumSeen: submitted.StoredChecksum, Timestamp: now.Add(-3 * time.Hour),
	})
	if err != nil {
		t.Fatalf("review load definition %s: %v", code, err)
	}
	activated, err := repository.TransitionDefinition(ctx, scope, definition.ID, reviewed.Version, DefinitionActive, DecisionRecord{
		ActorID: "load-authorizer", Action: DecisionActivate, ChecksumSeen: reviewed.StoredChecksum, Timestamp: now.Add(-2 * time.Hour),
		EffectiveFrom: timePtrForLoad(now.Add(-time.Hour)),
	})
	if err != nil {
		t.Fatalf("activate load definition %s: %v", code, err)
	}
	run := ReportRun{
		ID: reportLoadUUID(reportLoadRunBase + index), TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		DefinitionID: activated.ID, DefinitionVersion: activated.CurrentVersion, DefinitionCode: activated.Code,
		DefinitionChecksum: activated.StoredChecksum, ScopeKind: activated.ScopeKind, RequestedByRef: "load-performer",
		AsOf: now, Filter: cloneReportFilter(activated.Filter), Dataset: activated.Dataset, Format: activated.Format,
		Status: RunQueued, CreatedAt: now, ExpiresAt: now.Add(ReportRunRetention),
		SourceBoundary: SourceBoundary{CapturedAt: now, ProjectionVersion: "load-report-source.v1", SourceHighWater: map[string]time.Time{sourceKey: now.Add(-time.Minute)}, Population: len(rows), PopulationComplete: true},
	}
	created, err := repository.CreateRun(ctx, scope, run)
	if err != nil {
		t.Fatalf("create load run %s: %v", code, err)
	}
	repository.mu.Lock()
	repository.rows[created.ID] = rows
	repository.mu.Unlock()
	return created
}

func timePtrForLoad(value time.Time) *time.Time { return &value }

func measureReportLoadPages(t *testing.T, ctx context.Context, repository *MemoryRepository, scope ReportScope, run ReportRun, population int) reportLoadPageStats {
	t.Helper()
	if population <= reportLoadPageSize {
		t.Fatalf("population for %s is %d; a first and cursor page require more than %d rows", run.Dataset, population, reportLoadPageSize)
	}
	firstPage, err := repository.ListReportRows(ctx, scope, run, "", reportLoadPageSize)
	if err != nil {
		t.Fatalf("warm first page for %s: %v", run.Dataset, err)
	}
	if len(firstPage.Rows) != reportLoadPageSize || firstPage.NextCursor == "" {
		t.Fatalf("warm first page for %s returned rows=%d cursor=%q; want %d rows and a cursor", run.Dataset, len(firstPage.Rows), firstPage.NextCursor, reportLoadPageSize)
	}
	firstSamples := make([]time.Duration, 0, reportLoadSampleCount)
	cursorSamples := make([]time.Duration, 0, reportLoadSampleCount)
	for index := 0; index < reportLoadSampleCount; index++ {
		// A sample is a short batch divided by its repeat count. This keeps
		// sub-millisecond page reads measurable on the Windows timer while
		// preserving the per-page unit reported to the budget gate.
		start := time.Now()
		for repeat := 0; repeat < reportLoadPageRepeats; repeat++ {
			first, err := repository.ListReportRows(ctx, scope, run, "", reportLoadPageSize)
			if err != nil {
				t.Fatalf("first page sample %d repeat %d for %s: %v", index, repeat, run.Dataset, err)
			}
			if len(first.Rows) != reportLoadPageSize || first.NextCursor == "" {
				t.Fatalf("first page sample %d repeat %d for %s returned rows=%d cursor=%q", index, repeat, run.Dataset, len(first.Rows), first.NextCursor)
			}
		}
		firstSamples = append(firstSamples, time.Since(start)/reportLoadPageRepeats)

		start = time.Now()
		for repeat := 0; repeat < reportLoadPageRepeats; repeat++ {
			cursor, err := repository.ListReportRows(ctx, scope, run, firstPage.NextCursor, reportLoadPageSize)
			if err != nil {
				t.Fatalf("cursor page sample %d repeat %d for %s: %v", index, repeat, run.Dataset, err)
			}
			if len(cursor.Rows) != reportLoadPageSize || cursor.NextCursor == "" {
				t.Fatalf("cursor page sample %d repeat %d for %s returned rows=%d cursor=%q", index, repeat, run.Dataset, len(cursor.Rows), cursor.NextCursor)
			}
		}
		cursorSamples = append(cursorSamples, time.Since(start)/reportLoadPageRepeats)
	}
	return reportLoadPageStats{first: summarizeReportLoadDurations(firstSamples), cursor: summarizeReportLoadDurations(cursorSamples)}
}

func summarizeReportLoadDurations(samples []time.Duration) reportLoadDurationStats {
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	return reportLoadDurationStats{p50: reportLoadQuantile(sorted, 50), p95: reportLoadQuantile(sorted, 95), max: sorted[len(sorted)-1]}
}

func reportLoadQuantile(sorted []time.Duration, percentile int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := (percentile*len(sorted) + 99) / 100
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

func reportLoadUUID(value int) string { return fmt.Sprintf("00000000-0000-7000-8000-%012d", value) }
