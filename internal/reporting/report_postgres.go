//go:build postgres || load

package reporting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxReportPageLimit  = 500
	reportCursorVersion = 1
)

// ReportPageSQL returns one bounded processing-activity page. The filter
// fragment must have been built by ReportFilterSQL starting at position six;
// filterArgumentCount is the number of values in that fragment. Scope, as-of,
// dataset, filter, and keyset predicates all sit inside the materialized page
// CTE before its limit.
//
// The keyset expressions intentionally match migration 000092's
// ropa_register_keyset_idx: the same status rank, null review-date expression,
// and id ordering are used by the cursor and ORDER BY.
func ReportPageSQL(dataset ReportDataset, filterFragment string, filterArgumentCount int) string {
	columnSQL, predicateSQL, ok := reportDatasetFragments(dataset)
	if !ok || strings.TrimSpace(filterFragment) == "" || filterArgumentCount < 0 || filterArgumentCount > maxReportFilterNodes {
		return ""
	}
	hasCursor := 6 + filterArgumentCount
	cursorRank := hasCursor + 1
	cursorDate := hasCursor + 2
	cursorID := hasCursor + 3
	limit := hasCursor + 4
	formatted := fmt.Sprintf(`
WITH page AS MATERIALIZED (
  SELECT a.id::text,
         a.tenant_id::text,
         a.legal_entity_id::text,
         a.code,
         a.name,
         a.status,
         a.purpose,
         a.lawful_basis,
         a.controller,
         a.processor,
         a.automated_decision_making,
         a.data_subject_categories,
         a.personal_data_categories,
         a.security_measures,
         a.retention_period,
         a.next_review_date,
         COALESCE(a.owner_principal_id::text, ''),
         COALESCE(a.program_id::text, ''),
         COALESCE(a.matter_id::text, ''),
         a.version,
         a.updated_at,
         %s AS exceptions
  FROM ropa_processing_activities a
  WHERE a.tenant_id = $1::uuid
    AND a.legal_entity_id = $2::uuid
    AND ($3 = '' OR a.status::text = $3)
    AND ($4 = '' OR a.program_id = NULLIF($4, '')::uuid OR a.matter_id = NULLIF($4, '')::uuid)
    AND a.updated_at <= $5::timestamptz
    AND %s
    AND (__REPORT_FILTER__)
    AND (
      $%d = false OR
      (
        CASE a.status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END,
        COALESCE(a.next_review_date, '0001-01-01'::date),
        a.id
      ) > ($%d::integer, $%d::date, $%d::uuid)
    )
  ORDER BY
    CASE a.status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END,
    COALESCE(a.next_review_date, '0001-01-01'::date),
    a.id
  LIMIT $%d
)
SELECT * FROM page
ORDER BY
  CASE page.status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END,
  COALESCE(page.next_review_date, '0001-01-01'::date),
  page.id`, columnSQL, predicateSQL, hasCursor, cursorRank, cursorDate, cursorID, limit)
	return strings.ReplaceAll(formatted, "__REPORT_FILTER__", filterFragment)
}

func reportDatasetFragments(dataset ReportDataset) (string, string, bool) {
	switch dataset {
	case DatasetProcessingActivities:
		return `ARRAY[]::text[]`, `TRUE`, true
	case DatasetProcessingActivityExceptions:
		return ExceptionColumnSQL, ExceptionPredicateSQL, true
	default:
		return "", "", false
	}
}

type reportPageCursor struct {
	Version int    `json:"v"`
	Rank    int    `json:"r"`
	Status  string `json:"s"`
	Date    string `json:"d"`
	ID      string `json:"i"`
}

func (r *PostgresRepository) ListReportRows(ctx context.Context, scope ReportScope, requested ReportRun, cursor string, limit int) (ReportPage, error) {
	if err := r.validateInput(ctx, scope, requested.ID); err != nil {
		return ReportPage{}, err
	}
	limit = boundedReportPageLimit(limit)
	persisted, err := r.GetRun(ctx, scope, requested.ID)
	if err != nil {
		return ReportPage{}, err
	}
	if persisted.Status != RunQueued && persisted.Status != RunRunning {
		return ReportPage{}, ErrConflict
	}
	if persisted.Dataset == DatasetPrograms {
		return r.listProgramReportRows(ctx, scope, persisted, cursor, limit)
	}
	if persisted.Dataset == DatasetMatterExceptions {
		return r.listMatterReportRows(ctx, scope, persisted, cursor, limit)
	}
	_, _, ok := reportDatasetFragments(persisted.Dataset)
	if !ok {
		return ReportPage{}, ErrInvalid
	}
	position, err := decodeReportPageCursor(cursor)
	if err != nil {
		return ReportPage{}, err
	}
	filterFragment, filterArgs, err := ReportFilterSQLForDataset(persisted.Dataset, persisted.Filter, 6)
	if err != nil {
		return ReportPage{}, err
	}
	query := ReportPageSQL(persisted.Dataset, filterFragment, len(filterArgs))
	if query == "" {
		return ReportPage{}, ErrInvalid
	}
	highWater, ok := persisted.SourceBoundary.SourceHighWater["processing_activities"]
	if !ok || highWater.IsZero() {
		return ReportPage{}, ErrInvalid
	}
	hasCursor := position.ID != ""
	rank := 1
	date := "0001-01-01"
	id := "00000000-0000-0000-0000-000000000000"
	if hasCursor {
		rank = position.Rank
		date = position.Date
		id = position.ID
	}
	arguments := make([]any, 0, 10+len(filterArgs))
	arguments = append(arguments, scope.TenantID, scope.LegalEntityID, "", persisted.ScopeRef, highWater.UTC())
	arguments = append(arguments, filterArgs...)
	arguments = append(arguments, hasCursor, rank, date, id, limit+1)

	rows, err := r.pool.Query(ctx, query, arguments...)
	if err != nil {
		return ReportPage{}, fmt.Errorf("read bounded report rows: %w", err)
	}
	defer rows.Close()
	page := ReportPage{
		Rows: make([]ReportRow, 0, limit+1),
		Columns: []string{"tenant_id", "legal_entity_id", "code", "name", "status", "purpose", "lawful_basis",
			"controller", "processor", "automated_decision_making", "data_subject_categories", "personal_data_categories",
			"security_measures", "retention_period", "next_review_date", "owner_principal_id", "program_id", "matter_id",
			"version", "updated_at", "exceptions"},
	}
	for rows.Next() {
		row, err := scanProcessingActivityReportRow(rows)
		if err != nil {
			return ReportPage{}, fmt.Errorf("scan bounded report row: %w", err)
		}
		page.Rows = append(page.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return ReportPage{}, fmt.Errorf("read bounded report rows: %w", err)
	}
	if len(page.Rows) > limit {
		page.Rows = page.Rows[:limit]
		last := page.Rows[len(page.Rows)-1]
		page.NextCursor, err = encodeReportPageCursor(last)
		if err != nil {
			return ReportPage{}, err
		}
	}
	return page, nil
}

func scanProcessingActivityReportRow(row reportingRowScanner) (ReportRow, error) {
	var id, tenantID, legalEntityID, code, name, status, purpose, lawfulBasis, controller, processor string
	var automated bool
	var subjects, personalData, security, retention string
	var nextReview pgtype.Date
	var ownerID, programID, matterID string
	var version int64
	var updatedAt time.Time
	var exceptions []string
	if err := row.Scan(&id, &tenantID, &legalEntityID, &code, &name, &status, &purpose, &lawfulBasis,
		&controller, &processor, &automated, &subjects, &personalData, &security, &retention, &nextReview,
		&ownerID, &programID, &matterID, &version, &updatedAt, &exceptions); err != nil {
		return ReportRow{}, err
	}
	var review any
	if nextReview.Valid {
		review = nextReview.Time.UTC()
	}
	return ReportRow{ID: id, Values: map[string]any{
		"tenant_id": tenantID, "legal_entity_id": legalEntityID, "code": code, "name": name, "status": status,
		"purpose": purpose, "lawful_basis": lawfulBasis, "controller": controller, "processor": processor,
		"automated_decision_making": automated, "data_subject_categories": subjects,
		"personal_data_categories": personalData, "security_measures": security, "retention_period": retention,
		"next_review_date": review, "owner_principal_id": ownerID, "program_id": programID, "matter_id": matterID,
		"version": version, "updated_at": updatedAt.UTC(), "exceptions": exceptions,
	}}, nil
}

func encodeReportPageCursor(row ReportRow) (string, error) {
	status, _ := row.Values["status"].(string)
	if !validReportActivityStatus(status) || strings.TrimSpace(row.ID) == "" {
		return "", ErrInvalid
	}
	date := "0001-01-01"
	if value, ok := row.Values["next_review_date"].(time.Time); ok && !value.IsZero() {
		date = value.UTC().Format("2006-01-02")
	}
	body, err := json.Marshal(reportPageCursor{Version: reportCursorVersion, Rank: reportActivityStatusRank(status), Status: status, Date: date, ID: row.ID})
	if err != nil {
		return "", fmt.Errorf("encode report page cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func decodeReportPageCursor(value string) (reportPageCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return reportPageCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) > 512 {
		return reportPageCursor{}, ErrInvalid
	}
	var cursor reportPageCursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return reportPageCursor{}, ErrInvalid
	}
	if cursor.Version != reportCursorVersion || !validReportActivityStatus(cursor.Status) ||
		cursor.Rank != reportActivityStatusRank(cursor.Status) || !isUUID(cursor.ID) || !validReportCursorDate(cursor.Date) {
		return reportPageCursor{}, ErrInvalid
	}
	return cursor, nil
}

func reportActivityStatusRank(status string) int {
	switch status {
	case "NEW":
		return 1
	case "OPEN":
		return 2
	case "CLOSED":
		return 3
	default:
		return 0
	}
}

func validReportActivityStatus(status string) bool { return reportActivityStatusRank(status) > 0 }

func validReportCursorDate(value string) bool {
	if value == "0001-01-01" {
		return true
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func boundedReportPageLimit(value int) int {
	if value <= 0 {
		return ReportRunPageSize
	}
	if value > maxReportPageLimit {
		return maxReportPageLimit
	}
	return value
}
