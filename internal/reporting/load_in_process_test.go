//go:build load

package reporting

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func runInProcessReportLoad(t *testing.T, ctx context.Context, now time.Time) {
	t.Helper()
	activityRows, activityCounts := buildInProcessReportActivities(t, ctx, now)
	programRows, programCounts := buildInProcessReportPrograms(now)
	matterRows, matterCounts := buildInProcessReportMatters(now)
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
	t.Logf("REPORT_LOAD_SEED layer=in_process_full_population activities=%d open=%d exceptions=%d programs=%d at_risk=%d matters=%d exceptions=%d",
		reportLoadActivityCount, activityCounts.open, activityCounts.exceptions, reportLoadProgramCount, programCounts.selected,
		reportLoadMatterCount, matterCounts.selected)

	scope := ReportScope{TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID}
	base := NewMemoryRepository()
	repository := &reportLoadInProcessRepository{MemoryRepository: base, rows: make(map[string][]ReportRow)}
	for index, testCase := range reportLoadCases() {
		rows := activityRows
		switch testCase.dataset {
		case DatasetPrograms:
			rows = programRows
		case DatasetMatterExceptions:
			rows = matterRows
		}
		run := createInProcessReportRun(t, ctx, base, repository, scope, now, index, testCase, rows)
		stats := measureReportLoadPages(t, ctx, repository, scope, run)
		logReportLoadStats(t, "in_process_full_population", testCase, stats)
		enforceReportLoadBudget(t, "in_process_full_population", testCase, stats)
	}
}

type reportLoadInProcessRepository struct {
	*MemoryRepository
	rows map[string][]ReportRow
}

func (r *reportLoadInProcessRepository) ListReportRows(ctx context.Context, scope ReportScope, requested ReportRun, cursor string, limit int) (ReportPage, error) {
	if r == nil || r.MemoryRepository == nil || ctx == nil {
		return ReportPage{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return ReportPage{}, err
	}
	if requested.TenantID != scope.TenantID || requested.LegalEntityID != scope.LegalEntityID {
		return ReportPage{}, ErrNotFound
	}
	if limit <= 0 || limit > maxMemoryReportPageSize {
		limit = maxMemoryReportPageSize
	}
	current, err := r.GetRun(ctx, scope, requested.ID)
	if err != nil {
		return ReportPage{}, err
	}
	rows, ok := r.rows[current.ID]
	if !ok {
		return ReportPage{}, ErrNotFound
	}
	filtered := make([]ReportRow, 0, len(rows))
	for _, row := range rows {
		if reportLoadInProcessMatches(row, current) {
			filtered = append(filtered, row)
		}
	}
	sort.SliceStable(filtered, func(left, right int) bool {
		return reportLoadInProcessLess(filtered[left], filtered[right], current.Dataset)
	})

	start := 0
	if cursor != "" {
		start, err = strconv.Atoi(cursor)
		if err != nil || start < 0 || start > len(filtered) {
			return ReportPage{}, ErrInvalid
		}
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := ReportPage{Rows: make([]ReportRow, 0, end-start)}
	for _, row := range filtered[start:end] {
		values := make(map[string]any, len(row.Values))
		for key, value := range row.Values {
			values[key] = value
		}
		page.Rows = append(page.Rows, ReportRow{ID: row.ID, Values: values})
	}
	if end < len(filtered) {
		page.NextCursor = strconv.Itoa(end)
	}
	for _, row := range page.Rows {
		for field := range row.Values {
			page.Columns = append(page.Columns, field)
		}
	}
	sort.Strings(page.Columns)
	return page, nil
}

func reportLoadInProcessMatches(row ReportRow, run ReportRun) bool {
	filterValue := reportLoadInProcessFilterValue(run.Filter)
	switch run.Dataset {
	case DatasetProcessingActivities:
		return reportLoadString(row.Values["status"]) == "OPEN" && matchesReportLoadFilter(row, ReportFieldStatus, filterValue)
	case DatasetProcessingActivityExceptions:
		return reportLoadString(row.Values["status"]) == "OPEN" && reportLoadHasExceptions(row) && matchesReportLoadFilter(row, ReportFieldStatus, filterValue)
	case DatasetPrograms:
		return reportLoadString(row.Values["overall_state"]) == "AT_RISK" && matchesReportLoadFilter(row, ReportFieldOverallState, filterValue)
	case DatasetMatterExceptions:
		if !reportLoadMatterIsException(row, run.AsOf) || !matchesReportLoadFilter(row, ReportFieldDueCondition, filterValue) {
			return false
		}
		due, ok := row.Values["due_at"].(time.Time)
		return ok && !due.IsZero() && due.Before(run.AsOf)
	default:
		return false
	}
}

func reportLoadInProcessLess(left, right ReportRow, dataset ReportDataset) bool {
	switch dataset {
	case DatasetProcessingActivities, DatasetProcessingActivityExceptions:
		leftRank := reportLoadActivityStatusRank(reportLoadString(left.Values["status"]))
		rightRank := reportLoadActivityStatusRank(reportLoadString(right.Values["status"]))
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftDate, rightDate := reportLoadTime(left.Values["next_review_date"]), reportLoadTime(right.Values["next_review_date"])
		if !leftDate.Equal(rightDate) {
			return leftDate.Before(rightDate)
		}
		return left.ID < right.ID
	case DatasetPrograms:
		leftRank := reportLoadProgramStatusRank(reportLoadString(left.Values["status"]))
		rightRank := reportLoadProgramStatusRank(reportLoadString(right.Values["status"]))
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftUpdated, rightUpdated := reportLoadTime(left.Values["updated_at"]), reportLoadTime(right.Values["updated_at"])
		if !leftUpdated.Equal(rightUpdated) {
			return leftUpdated.After(rightUpdated)
		}
		return left.ID > right.ID
	case DatasetMatterExceptions:
		leftPriority, rightPriority := reportLoadInt(left.Values["priority"]), reportLoadInt(right.Values["priority"])
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		leftUpdated, rightUpdated := reportLoadTime(left.Values["updated_at"]), reportLoadTime(right.Values["updated_at"])
		if !leftUpdated.Equal(rightUpdated) {
			return leftUpdated.After(rightUpdated)
		}
		return left.ID > right.ID
	default:
		return left.ID < right.ID
	}
}

func reportLoadInProcessFilterValue(filter *ReportFilterExpression) string {
	if filter == nil {
		return ""
	}
	if filter.Kind == "condition" {
		return filter.Value
	}
	for _, child := range filter.Children {
		if child.Kind == "condition" {
			return child.Value
		}
	}
	return ""
}

func matchesReportLoadFilter(row ReportRow, field ReportFilterField, value string) bool {
	switch field {
	case ReportFieldStatus:
		return reportLoadString(row.Values[string(field)]) == value
	case ReportFieldOverallState:
		return reportLoadString(row.Values[string(field)]) == value
	case ReportFieldDueCondition:
		if value != "OVERDUE" {
			return false
		}
		due, ok := row.Values["due_at"].(time.Time)
		return ok && !due.IsZero() && due.Before(time.Now().UTC())
	default:
		return false
	}
}

func reportLoadMatterIsException(row ReportRow, asOf time.Time) bool {
	status := reportLoadString(row.Values["status"])
	if status == "CLOSED" || status == "CANCELLED" {
		return false
	}
	if reportLoadString(row.Values["matter_type"]) == "EXCEPTION" {
		return true
	}
	due, ok := row.Values["due_at"].(time.Time)
	return ok && !due.IsZero() && due.Before(asOf)
}

func reportLoadHasExceptions(row ReportRow) bool {
	switch values := row.Values["exceptions"].(type) {
	case []string:
		return len(values) > 0
	case []any:
		return len(values) > 0
	default:
		return false
	}
}

func reportLoadString(value any) string {
	text, _ := value.(string)
	return text
}

func reportLoadTime(value any) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case *time.Time:
		if typed != nil {
			return *typed
		}
	}
	return time.Time{}
}

func reportLoadInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case int32:
		return int(typed)
	default:
		return 0
	}
}

func reportLoadActivityStatusRank(status string) int {
	switch status {
	case "NEW":
		return 1
	case "OPEN":
		return 2
	case "CLOSED":
		return 3
	default:
		return 4
	}
}

func reportLoadProgramStatusRank(status string) int {
	switch status {
	case "ACTIVE":
		return 0
	case "PAUSED":
		return 1
	case "DRAFT":
		return 2
	default:
		return 3
	}
}

func createInProcessReportRun(t *testing.T, ctx context.Context, base *MemoryRepository, repository *reportLoadInProcessRepository, scope ReportScope, now time.Time, index int, testCase reportLoadCase, rows []ReportRow) ReportRun {
	t.Helper()
	definition := ReportDefinition{
		ID: reportLoadUUID(reportLoadDefinitionBase + index), TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		Code: testCase.code, Name: testCase.name, Description: "Synthetic in-process load fixture for a governed report definition.",
		Dataset: testCase.dataset, ScopeKind: ScopeLegalEntity, Format: FormatCSV, Filter: cloneReportFilter(testCase.filter),
		Status: DefinitionDraft, CurrentVersion: 1, MakerID: "load-maker", CreatedAt: now.Add(-5 * time.Hour), UpdatedAt: now.Add(-5 * time.Hour), Version: 1,
	}
	definition.StoredChecksum = definition.Checksum()
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID, Version: 1, BaseVersion: 0,
		Dataset: definition.Dataset, ScopeKind: definition.ScopeKind, Format: definition.Format, Filter: cloneReportFilter(definition.Filter),
		Checksum: definition.StoredChecksum, MakerID: definition.MakerID, CreatedAt: now.Add(-5 * time.Hour), Decision: "PROPOSED",
	}
	if _, err := base.CreateDefinition(ctx, scope, definition, revision); err != nil {
		t.Fatalf("create in-process load definition %s: %v", testCase.dataset, err)
	}
	submitted, err := base.TransitionDefinition(ctx, scope, definition.ID, definition.Version, DefinitionPendingReview, DecisionRecord{
		ActorID: "load-maker", Action: DecisionSubmit, ChecksumSeen: definition.StoredChecksum, Timestamp: now.Add(-4 * time.Hour),
	})
	if err != nil {
		t.Fatalf("submit in-process load definition %s: %v", testCase.dataset, err)
	}
	reviewed, err := base.TransitionDefinition(ctx, scope, definition.ID, submitted.Version, DefinitionReviewed, DecisionRecord{
		ActorID: "load-reviewer", Action: DecisionReview, ChecksumSeen: submitted.StoredChecksum, Timestamp: now.Add(-3 * time.Hour),
	})
	if err != nil {
		t.Fatalf("review in-process load definition %s: %v", testCase.dataset, err)
	}
	activated, err := base.TransitionDefinition(ctx, scope, definition.ID, reviewed.Version, DefinitionActive, DecisionRecord{
		ActorID: "load-authorizer", Action: DecisionActivate, ChecksumSeen: reviewed.StoredChecksum, Timestamp: now.Add(-2 * time.Hour), EffectiveFrom: timePtrForLoad(now.Add(-time.Hour)),
	})
	if err != nil {
		t.Fatalf("activate in-process load definition %s: %v", testCase.dataset, err)
	}
	run := ReportRun{
		ID: reportLoadUUID(reportLoadRunBase + index), TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		DefinitionID: activated.ID, DefinitionVersion: activated.CurrentVersion, DefinitionCode: activated.Code, DefinitionChecksum: activated.StoredChecksum,
		ScopeKind: activated.ScopeKind, RequestedByRef: "load-performer", AsOf: now, Filter: cloneReportFilter(activated.Filter), Dataset: activated.Dataset,
		Format: activated.Format, Status: RunQueued, CreatedAt: now, ExpiresAt: now.Add(ReportRunRetention),
		SourceBoundary: SourceBoundary{CapturedAt: now, ProjectionVersion: "in-process-report-source.v1", SourceHighWater: map[string]time.Time{testCase.sourceKey: now}, Population: len(rows), PopulationComplete: true},
	}
	created, err := base.CreateRun(ctx, scope, run)
	if err != nil {
		t.Fatalf("create in-process load run %s: %v", testCase.dataset, err)
	}
	repository.rows[created.ID] = rows
	return created
}

type inProcessActivityCounts struct {
	complete   int
	exceptions int
	open       int
}

type inProcessPopulationCounts struct {
	total    int
	selected int
}

func buildInProcessReportActivities(t *testing.T, ctx context.Context, now time.Time) ([]ReportRow, inProcessActivityCounts) {
	t.Helper()
	repository := ropa.NewMemoryRepository()
	service := ropa.NewService(repository, nil)
	service.Now = func() time.Time { return now }
	rows := make([]ReportRow, 0, reportLoadActivityCount)
	counts := inProcessActivityCounts{}
	for index := 0; index < reportLoadActivityCount; index++ {
		kind := index % 10
		reviewDate := now.AddDate(0, 0, (index%365)-120)
		startDate := now.AddDate(-1, 0, 0)
		completedAt := now.Add(-30 * 24 * time.Hour)
		input := ropa.CreateActivityInput{
			TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID, Code: fmt.Sprintf("PA-LOAD-%06d", index),
			Name: fmt.Sprintf("Large-bank processing activity %06d", index), Description: "Synthetic load fixture for a bank's processing-activity register.",
			Purpose: "Provide a bank service.", LawfulBasis: []string{"Contract", "Consent", "Legal obligation"}[index%3], Controller: "Fidelity Bank Nigeria",
			Processor: "Internal operations", AutomatedDecisionMaking: index%7 == 0, DataSubjectCategories: "Customers; Employees", PersonalDataCategories: "Account; contact details",
			SecurityMeasures: "Encryption in transit and at rest", RetentionPeriod: "Seven years after the relationship ends", StartDate: &startDate, NextReviewDate: &reviewDate,
			OwnerPrincipalID: "owner-load", RequiredAuthorityPrincipalID: "authority-load", ActorID: "load-seed",
			DataCategories: []ropa.DataCategory{{Category: "Account identifiers", Sensitivity: "DIRECT_PERSONAL"}}, Recipients: []ropa.Recipient{loadRecipient(index)},
			Systems: []ropa.System{{SystemName: "Core banking platform", SystemKind: "APPLICATION"}},
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
			input.Reviews = []ropa.Review{{ID: fmt.Sprintf("RV-LOAD-%06d", index), DueDate: reviewDate, CompletedAt: &completedAt, Outcome: "CONFIRMED", ReviewerPrincipalID: "reviewer-load"}}
		}
		activity, err := service.CreateActivity(ctx, input)
		if err != nil {
			t.Fatalf("seed in-process processing activity %d: %v", index, err)
		}
		if kind < 6 && index%20 == 0 {
			endDate := now.Add(-24 * time.Hour)
			activity, err = service.TransitionActivity(ctx, ropa.TransitionActivityInput{TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID, ActivityID: activity.ID, ExpectedVersion: activity.Version, To: ropa.StatusClosed, EndDate: &endDate, ActorID: "load-seed"})
		} else if index%4 != 0 {
			activity, err = service.TransitionActivity(ctx, ropa.TransitionActivityInput{TenantID: reportLoadTenantID, LegalEntityID: reportLoadLegalEntityID, ActivityID: activity.ID, ExpectedVersion: activity.Version, To: ropa.StatusOpen, ActorID: "load-seed"})
		}
		if err != nil {
			t.Fatalf("transition in-process processing activity %d: %v", index, err)
		}
		blockers := ExceptionBlockers(activity)
		if len(blockers) == 0 {
			counts.complete++
		} else {
			counts.exceptions++
		}
		if activity.Status == ropa.StatusOpen {
			counts.open++
		}
		rows = append(rows, processingActivityReportRow(activity, blockers))
	}
	return rows, counts
}

func buildInProcessReportPrograms(now time.Time) ([]ReportRow, inProcessPopulationCounts) {
	rows := make([]ReportRow, 0, reportLoadProgramCount)
	counts := inProcessPopulationCounts{total: reportLoadProgramCount}
	for index := 0; index < reportLoadProgramCount; index++ {
		state := "CURRENT"
		if index%4 == 0 {
			state = "AT_RISK"
			counts.selected++
		}
		updatedAt := now.Add(-time.Duration(index%365) * time.Hour)
		rows = append(rows, ReportRow{ID: reportLoadUUID(100_000_000 + index), Values: map[string]any{
			"tenant_id": reportLoadTenantID, "legal_entity_id": reportLoadLegalEntityID, "code": fmt.Sprintf("PROGRAM-LOAD-%05d", index),
			"name": fmt.Sprintf("Bank Program %05d", index), "program_type": "ONGOING_OBLIGATION", "status": "ACTIVE", "owning_function": "Risk and compliance",
			"owner_principal_id": fmt.Sprintf("owner-program-%03d", index%25), "authority_principal_id": "authority-program", "jurisdiction": "NG",
			"effective_from": now.AddDate(-2, 0, 0), "effective_until": nil, "created_at": now.AddDate(-2, 0, 0), "updated_at": updatedAt,
			"version": int64(1 + index%4), "assessed_program_version": int64(1 + index%4), "projection_version": int64(20 + index%7), "projection_stale": index%11 == 0,
			"overall_state": state, "has_open_matters": state == "AT_RISK", "state_generated_at": now.Add(-5 * time.Minute), "reasons_total": 2,
			"reasons": []string{"Current source and owner checks are incomplete."}, "reasons_omitted": 0,
		}})
	}
	return rows, counts
}

func buildInProcessReportMatters(now time.Time) ([]ReportRow, inProcessPopulationCounts) {
	rows := make([]ReportRow, 0, reportLoadMatterCount)
	counts := inProcessPopulationCounts{total: reportLoadMatterCount}
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
			counts.selected++
		}
		rows = append(rows, ReportRow{ID: reportLoadUUID(200_000_000 + index), Values: map[string]any{
			"tenant_id": reportLoadTenantID, "legal_entity_id": reportLoadLegalEntityID, "reference": fmt.Sprintf("MATTER-LOAD-%06d", index),
			"matter_type": matterType, "status": status, "priority": 1 + index%5, "title": fmt.Sprintf("Issue or change %06d", index),
			"summary": "Synthetic load fixture for an overdue privacy obligation.", "owner_principal_id": fmt.Sprintf("owner-matter-%03d", index%40),
			"required_authority": "authority-matter", "due_at": dueAt, "closed_at": closedAt, "closure_reason": "", "reopen_count": 0,
			"created_at": now.AddDate(0, 0, -index%365), "updated_at": now.Add(-time.Duration(index%30) * time.Hour), "version": int64(1 + index%3),
			"scope": map[string]any{"access": "INTERNAL"}, "latest_verification_result": "INCONCLUSIVE", "latest_verification_at": now.Add(-48 * time.Hour),
			"open_action_count": 1, "outcome_check_count": 1, "reasons_total": len(exceptions), "reasons": exceptions, "reasons_omitted": 0, "exceptions": exceptions,
		}})
	}
	return rows, counts
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
		"status": string(activity.Status), "purpose": activity.Purpose, "lawful_basis": activity.LawfulBasis, "controller": activity.Controller,
		"processor": activity.Processor, "automated_decision_making": activity.AutomatedDecisionMaking, "data_subject_categories": activity.DataSubjectCategories,
		"personal_data_categories": activity.PersonalDataCategories, "security_measures": activity.SecurityMeasures, "retention_period": activity.RetentionPeriod,
		"next_review_date": nextReview, "owner_principal_id": activity.OwnerPrincipalID, "program_id": activity.ProgramID, "version": activity.Version,
		"updated_at": activity.UpdatedAt.UTC(), "exceptions": missing,
	}}
}
