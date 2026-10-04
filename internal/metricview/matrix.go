package metricview

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrMatrixInvalid = errors.New("metric matrix request is invalid")
	ErrMatrixUnavailable = errors.New("metric matrix is unavailable")
)

type MatrixKind string

const (
	MatrixRiskAppetite MatrixKind = "RISK_APPETITE"
	MatrixAssuranceCoverage MatrixKind = "ASSURANCE_COVERAGE"
)

const (
	RiskAppetiteMatrixRevision = "risk-appetite-matrix-v1"
	AssuranceCoverageMatrixRevision = "assurance-coverage-matrix-v1"
)

var appetiteMatrixColumns = []string{"WITHIN", "APPROACHING", "BREACHED", "UNKNOWN"}
var assuranceMatrixColumns = []string{"SUPPORTED", "PARTIAL", "FAILED", "UNKNOWN"}

type MatrixCell struct {
	State string `json:"state"`
	Count int `json:"count"`
}

type MatrixRow struct {
	Key string `json:"key"`
	Label string `json:"label"`
	Total int `json:"total"`
	Cells []MatrixCell `json:"cells"`
}

type Matrix struct {
	Kind MatrixKind `json:"kind"`
	DefinitionRevision string `json:"definition_revision"`
	ScopeID string `json:"scope_id"`
	ScopeKind string `json:"scope_kind"`
	GeneratedAt time.Time `json:"generated_at"`
	Population int `json:"population"`
	Columns []string `json:"columns"`
	Rows []MatrixRow `json:"rows"`
}

type MatrixReader interface {
	RiskAppetiteMatrix(context.Context, string, string, time.Time) (Matrix, error)
	AssuranceCoverageMatrix(context.Context, string, string, time.Time) (Matrix, error)
}

type matrixBucket struct {
	Category string
	State string
	Count int
}

func buildMatrix(kind MatrixKind, revision, scopeID string, at time.Time, columns []string, buckets []matrixBucket) Matrix {
	rows := map[string]*MatrixRow{}
	population := 0
	for _, bucket := range buckets {
		category := strings.TrimSpace(bucket.Category)
		if category == "" {
			category = "Uncategorized"
		}
		state := strings.TrimSpace(bucket.State)
		if bucket.Count <= 0 || state == "" {
			continue
		}
		row := rows[category]
		if row == nil {
			row = &MatrixRow{Key: category, Label: category, Cells: make([]MatrixCell, len(columns))}
			for i, column := range columns {
				row.Cells[i] = MatrixCell{State: column}
			}
			rows[category] = row
		}
		for i, column := range columns {
			if column == state {
				row.Cells[i].Count += bucket.Count
				row.Total += bucket.Count
				population += bucket.Count
				break
			}
		}
	}
	values := make([]MatrixRow, 0, len(rows))
	for _, row := range rows {
		values = append(values, *row)
	}
	sort.Slice(values, func(i, j int) bool {
		return strings.ToLower(values[i].Label) < strings.ToLower(values[j].Label)
	})
	return Matrix{
		Kind: kind,
		DefinitionRevision: revision,
		ScopeID: strings.TrimSpace(scopeID),
		ScopeKind: "LEGAL_ENTITY",
		GeneratedAt: at.UTC(),
		Population: population,
		Columns: append([]string(nil), columns...),
		Rows: values,
	}
}
