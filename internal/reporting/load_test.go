//go:build load

package reporting

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	reportLoadTenantID       = "tenant-load"
	reportLoadLegalEntityID  = "entity-load"
	reportLoadActivityCount  = 100_000
	reportLoadProgramCount   = 5_000
	reportLoadMatterCount    = 25_000
	reportLoadPageSize       = 50
	reportLoadSampleCount    = 50
	reportLoadPageBudget     = 750 * time.Millisecond
	reportLoadSourceKey      = "processing_activities"
	reportLoadProgramSource  = "programs"
	reportLoadMatterSource   = "matters"
	reportLoadDefinitionBase = 910_000_000
	reportLoadRunBase        = 920_000_000
)

// TestReportPageBudget measures the bounded report page path against the same
// 750ms budget used by the Tranche 1 register list. When TEST_DATABASE_URL is
// reachable, the measured call is PostgresRepository.ListReportRows. If it is
// not reachable, the fallback measures an in-process repository that filters
// and sorts the complete unsorted population on every call; that fallback is
// explicitly not a PostgreSQL query-budget result.
func TestReportPageBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if tryPostgresReportLoad(t, ctx, now) {
		return
	}
	runInProcessReportLoad(t, ctx, now)
}

type reportLoadCase struct {
	dataset            ReportDataset
	code               string
	name               string
	filter             *ReportFilterExpression
	sourcePopulation   int
	selectedPopulation int
	sourceKey          string
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

type reportPageLister interface {
	ListReportRows(context.Context, ReportScope, ReportRun, string, int) (ReportPage, error)
}

func reportLoadCases() []reportLoadCase {
	return []reportLoadCase{
		{
			dataset: DatasetProcessingActivities, code: "LOAD-PROCESSING-ACTIVITIES", name: "Load processing activities",
			filter: reportLoadFilter(ReportFieldStatus, "OPEN"), sourcePopulation: reportLoadActivityCount,
			selectedPopulation: 75_000, sourceKey: reportLoadSourceKey,
		},
		{
			dataset: DatasetProcessingActivityExceptions, code: "LOAD-PROCESSING-EXCEPTIONS", name: "Load processing exceptions",
			filter: reportLoadFilter(ReportFieldStatus, "OPEN"), sourcePopulation: reportLoadActivityCount,
			selectedPopulation: 30_000, sourceKey: reportLoadSourceKey,
		},
		{
			dataset: DatasetPrograms, code: "LOAD-PROGRAMS", name: "Load Program population",
			filter: reportLoadFilter(ReportFieldOverallState, "AT_RISK"), sourcePopulation: reportLoadProgramCount,
			selectedPopulation: 1_250, sourceKey: reportLoadProgramSource,
		},
		{
			dataset: DatasetMatterExceptions, code: "LOAD-MATTERS", name: "Load issue and change population",
			filter: reportLoadFilter(ReportFieldDueCondition, "OVERDUE"), sourcePopulation: reportLoadMatterCount,
			selectedPopulation: 10_000, sourceKey: reportLoadMatterSource,
		},
	}
}

func reportLoadFilter(field ReportFilterField, value string) *ReportFilterExpression {
	return &ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{{
			Kind: "condition", Field: field, Operator: "is", Value: value,
		}},
	}
}

func measureReportLoadPages(t *testing.T, ctx context.Context, lister reportPageLister, scope ReportScope, run ReportRun) reportLoadPageStats {
	t.Helper()

	// Warm up both paths so the reported samples describe steady-state query
	// execution rather than first-use connection and plan setup. The two warm-up
	// calls are excluded from the sample set.
	firstPage, err := lister.ListReportRows(ctx, scope, run, "", reportLoadPageSize)
	if err != nil {
		t.Fatalf("warm first page for %s: %v", run.Dataset, err)
	}
	assertReportLoadPage(t, run.Dataset, "first warm-up", firstPage)
	cursorPage, err := lister.ListReportRows(ctx, scope, run, firstPage.NextCursor, reportLoadPageSize)
	if err != nil {
		t.Fatalf("warm cursor page for %s: %v", run.Dataset, err)
	}
	assertReportLoadPage(t, run.Dataset, "cursor warm-up", cursorPage)

	firstSamples := make([]time.Duration, 0, reportLoadSampleCount)
	cursorSamples := make([]time.Duration, 0, reportLoadSampleCount)
	for sample := 0; sample < reportLoadSampleCount; sample++ {
		start := time.Now()
		page, err := lister.ListReportRows(ctx, scope, run, "", reportLoadPageSize)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("first page sample %d for %s: %v", sample, run.Dataset, err)
		}
		assertReportLoadPage(t, run.Dataset, fmt.Sprintf("first sample %d", sample), page)
		firstSamples = append(firstSamples, elapsed)

		start = time.Now()
		page, err = lister.ListReportRows(ctx, scope, run, firstPage.NextCursor, reportLoadPageSize)
		elapsed = time.Since(start)
		if err != nil {
			t.Fatalf("cursor page sample %d for %s: %v", sample, run.Dataset, err)
		}
		assertReportLoadPage(t, run.Dataset, fmt.Sprintf("cursor sample %d", sample), page)
		cursorSamples = append(cursorSamples, elapsed)
	}

	return reportLoadPageStats{
		first:  summarizeReportLoadDurations(firstSamples),
		cursor: summarizeReportLoadDurations(cursorSamples),
	}
}

func assertReportLoadPage(t *testing.T, dataset ReportDataset, label string, page ReportPage) {
	t.Helper()
	if len(page.Rows) != reportLoadPageSize || page.NextCursor == "" {
		t.Fatalf("%s page for %s returned rows=%d cursor=%q; want %d rows and a cursor", label, dataset, len(page.Rows), page.NextCursor, reportLoadPageSize)
	}
}

func summarizeReportLoadDurations(samples []time.Duration) reportLoadDurationStats {
	if len(samples) == 0 {
		return reportLoadDurationStats{}
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	return reportLoadDurationStats{
		p50: reportLoadQuantile(sorted, 50),
		p95: reportLoadQuantile(sorted, 95),
		max: sorted[len(sorted)-1],
	}
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

func logReportLoadStats(t *testing.T, layer string, testCase reportLoadCase, stats reportLoadPageStats) {
	t.Helper()
	t.Logf("REPORT_PAGE_BUDGET layer=%s dataset=%s source_population=%d selected_population=%d page_size=%d samples=%d warmups=2 first_p50=%s first_p95=%s first_max=%s cursor_p50=%s cursor_p95=%s cursor_max=%s budget=%s",
		layer, testCase.dataset, testCase.sourcePopulation, testCase.selectedPopulation, reportLoadPageSize, reportLoadSampleCount,
		stats.first.p50, stats.first.p95, stats.first.max, stats.cursor.p50, stats.cursor.p95, stats.cursor.max, reportLoadPageBudget)
}

func enforceReportLoadBudget(t *testing.T, layer string, testCase reportLoadCase, stats reportLoadPageStats) {
	t.Helper()
	if stats.first.max > reportLoadPageBudget {
		t.Logf("REPORT_PAGE_BUDGET_MAX_CONCERN layer=%s dataset=%s path=first budget=%s observed=%s over_budget=%s",
			layer, testCase.dataset, reportLoadPageBudget, stats.first.max, stats.first.max-reportLoadPageBudget)
	}
	if stats.cursor.max > reportLoadPageBudget {
		t.Logf("REPORT_PAGE_BUDGET_MAX_CONCERN layer=%s dataset=%s path=cursor budget=%s observed=%s over_budget=%s",
			layer, testCase.dataset, reportLoadPageBudget, stats.cursor.max, stats.cursor.max-reportLoadPageBudget)
	}
	if stats.first.p50 > reportLoadPageBudget || stats.first.p95 > reportLoadPageBudget ||
		stats.cursor.p50 > reportLoadPageBudget || stats.cursor.p95 > reportLoadPageBudget {
		t.Errorf("REPORT_PAGE_BUDGET_CONCERN layer=%s dataset=%s budget=%s first_p50=%s first_p95=%s cursor_p50=%s cursor_p95=%s",
			layer, testCase.dataset, reportLoadPageBudget, stats.first.p50, stats.first.p95, stats.cursor.p50, stats.cursor.p95)
	}
}

func timePtrForLoad(value time.Time) *time.Time { return &value }

func reportLoadUUID(value int) string { return fmt.Sprintf("00000000-0000-7000-8000-%012d", value) }

func newReportLoadID(t *testing.T) string {
	t.Helper()
	value, err := platformid.NewUUIDv7()
	if err != nil {
		t.Fatalf("create load fixture identifier: %v", err)
	}
	return value
}
