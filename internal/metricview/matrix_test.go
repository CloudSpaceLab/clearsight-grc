package metricview

import (
	"testing"
	"time"
)

func TestBuildMatrixKeepsStableColumnsAndPopulation(t *testing.T) {
	at := time.Date(2026, 10, 4, 18, 0, 0, 0, time.UTC)
	matrix := buildMatrix(
		MatrixRiskAppetite,
		RiskAppetiteMatrixRevision,
		"entity-1",
		at,
		appetiteMatrixColumns,
		[]matrixBucket{
			{Category: "Operational", State: "BREACHED", Count: 2},
			{Category: "Operational", State: "UNKNOWN", Count: 1},
			{Category: "Cyber", State: "WITHIN", Count: 3},
		},
	)
	if matrix.Population != 6 || len(matrix.Rows) != 2 || len(matrix.Columns) != 4 {
		t.Fatalf("matrix=%#v", matrix)
	}
	if matrix.Rows[0].Label != "Cyber" || matrix.Rows[1].Label != "Operational" {
		t.Fatalf("rows=%#v", matrix.Rows)
	}
	if matrix.Rows[1].Total != 3 || matrix.Rows[1].Cells[2].Count != 2 || matrix.Rows[1].Cells[3].Count != 1 {
		t.Fatalf("operational row=%#v", matrix.Rows[1])
	}
}

func TestBuildMatrixUsesUncategorizedWithoutInventingUnknownPopulation(t *testing.T) {
	matrix := buildMatrix(
		MatrixAssuranceCoverage,
		AssuranceCoverageMatrixRevision,
		"entity-1",
		time.Now(),
		assuranceMatrixColumns,
		[]matrixBucket{{Category: "", State: "UNKNOWN", Count: 4}},
	)
	if matrix.Population != 4 || len(matrix.Rows) != 1 || matrix.Rows[0].Label != "Uncategorized" {
		t.Fatalf("matrix=%#v", matrix)
	}
	if matrix.Rows[0].Cells[3].Count != 4 {
		t.Fatalf("unknown cell=%#v", matrix.Rows[0].Cells)
	}
}
