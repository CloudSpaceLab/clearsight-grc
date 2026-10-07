package metricview

import (
	"strings"
	"testing"
)

func TestLossPeriodMetricMigrationDefinesFlowMetricsAndRuntimeSnapshotSupport(t *testing.T) {
	up := readMetricMigration(t, "000133_operational_loss_period_metrics.up.sql")
	down := readMetricMigration(t, "000133_operational_loss_period_metrics.down.sql")

	for _, required := range []string{
		"'operational_loss_events','operational-loss-period-v1','Loss events','COUNT','PERIOD_FLOW','NO_CONDITION','SUM_DISJOINT_COUNTS'",
		"'operational_loss_net','operational-loss-period-v1','Net operational loss','MONEY','PERIOD_FLOW','NO_CONDITION','SUM_SAME_CURRENCY'",
		"'operational-loss-period-v1'",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("up migration lacks %q", required)
		}
	}
	if !strings.Contains(down, "DELETE FROM metric_runtime_membership_sets") ||
		!strings.Contains(down, "DELETE FROM metric_definitions") {
		t.Fatal("down migration must clear transient Loss period snapshots and definitions")
	}
}
