package metricview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeMoneyMigrationExtendsMetricContractWithoutWeakeningCounts(t *testing.T) {
	up := readMetricMigration(t, "000132_metric_native_money.up.sql")
	down := readMetricMigration(t, "000132_metric_native_money.down.sql")

	for _, required := range []string{
		"unit IN ('COUNT','MONEY')",
		"basis IN ('CURRENT_POSTURE','PERIOD_FLOW')",
		"aggregation_rule IN ('SUM_DISJOINT_COUNTS','SUM_SAME_CURRENCY')",
		"ADD COLUMN currency text",
		"ADD COLUMN member_count bigint",
		"condition IN ('CLEAR','ATTENTION','NEUTRAL')",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("up migration lacks %q", required)
		}
	}
	if !strings.Contains(down, "Native metric measure history exists; refusing to erase it") {
		t.Fatal("down migration must refuse to erase native measure history")
	}
}

func readMetricMigration(t *testing.T, name string) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(payload)
}
