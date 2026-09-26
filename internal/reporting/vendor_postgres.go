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

// VendorReportPageSQL returns one bounded vendor-relationship page. Tenant,
// legal-entity, archive, source-boundary, filter and keyset predicates are all
// applied before LIMIT so hidden or out-of-scope relationships cannot consume
// page slots.
func VendorReportPageSQL(filterFragment string, filterArgumentCount int) string {
	if strings.TrimSpace(filterFragment) == "" || filterArgumentCount < 0 || filterArgumentCount > maxReportFilterNodes {
		return ""
	}
	hasCursor := 6 + filterArgumentCount
	cursorDate := hasCursor + 1
	cursorID := hasCursor + 2
	limit := hasCursor + 3

	template := `
WITH vendor_rows AS (
  SELECT r.id::text,
         r.tenant_id::text,
         r.legal_entity_id::text,
         r.vendor_id::text,
         v.legal_name AS vendor_name,
         v.trading_name,
         v.registration_ref,
         v.jurisdiction,
         v.status AS vendor_status,
         r.service_name,
         r.business_owner_principal_id::text AS owner_principal_id,
         r.criticality,
         r.privacy_role,
         r.status,
         r.effective_from,
         r.renewal_at,
         r.created_at,
         GREATEST(r.updated_at,v.updated_at) AS updated_at,
         r.version
  FROM third_party_relationships r
  JOIN third_parties v
    ON v.id=r.vendor_id AND v.tenant_id=r.tenant_id
  WHERE r.tenant_id=$1::uuid
    AND r.legal_entity_id=$2::uuid
    AND $3=''
    AND $4<>''
    AND GREATEST(r.updated_at,v.updated_at)<=$5::timestamptz
    AND NOT EXISTS (
      SELECT 1
      FROM demo_record_archives archive
      WHERE archive.tenant_id=r.tenant_id
        AND archive.legal_entity_id=r.legal_entity_id
        AND archive.record_type='VENDOR_RELATIONSHIP'
        AND archive.record_id=r.id
        AND archive.restored_at IS NULL
    )
), page AS MATERIALIZED (
  SELECT a.*
  FROM vendor_rows a
  WHERE (__REPORT_FILTER__)
    AND (
      $__HAS_CURSOR__=false OR
      a.updated_at<$__CURSOR_DATE__::timestamptz OR
      (a.updated_at=$__CURSOR_DATE__::timestamptz AND a.id<NULLIF($__CURSOR_ID__,'')::text)
    )
  ORDER BY a.updated_at DESC,a.id DESC
  LIMIT $__LIMIT__
)
SELECT * FROM page
ORDER BY page.updated_at DESC,page.id DESC`

	return strings.NewReplacer(
		"__REPORT_FILTER__", filterFragment,
		"$__HAS_CURSOR__", fmt.Sprintf("$%d", hasCursor),
		"$__CURSOR_DATE__", fmt.Sprintf("$%d", cursorDate),
		"$__CURSOR_ID__", fmt.Sprintf("$%d", cursorID),
		"$__LIMIT__", fmt.Sprintf("$%d", limit),
	).Replace(template)
}

func (r *PostgresRepository) captureVendorSourceBoundary(ctx context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error) {
	if definition.ScopeKind != ScopeLegalEntity || strings.TrimSpace(definition.ScopeRef) != "" {
		return SourceBoundary{}, ErrInvalid
	}
	var highWater *time.Time
	var population int
	if err := r.pool.QueryRow(ctx, `SELECT max(GREATEST(r.updated_at,v.updated_at)),count(*)::integer
		FROM third_party_relationships r
		JOIN third_parties v ON v.id=r.vendor_id AND v.tenant_id=r.tenant_id
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid
		  AND NOT EXISTS (
		    SELECT 1 FROM demo_record_archives archive
		    WHERE archive.tenant_id=r.tenant_id
		      AND archive.legal_entity_id=r.legal_entity_id
		      AND archive.record_type='VENDOR_RELATIONSHIP'
		      AND archive.record_id=r.id
		      AND archive.restored_at IS NULL
		  )`, scope.TenantID, scope.LegalEntityID).Scan(&highWater, &population); err != nil {
		return SourceBoundary{}, fmt.Errorf("read vendor report source boundary: %w", err)
	}
	captured := time.Now().UTC()
	if highWater == nil {
		highWater = &captured
	}
	return SourceBoundary{
		CapturedAt:         captured,
		ProjectionVersion:  "vendor-relationship-report.v1",
		SourceHighWater:    map[string]time.Time{"vendor_relationships": highWater.UTC()},
		Population:         population,
		PopulationComplete: false,
	}, nil
}

func (r *PostgresRepository) listVendorReportRows(ctx context.Context, scope ReportScope, run ReportRun, cursor string, limit int) (ReportPage, error) {
	filter, args, err := ReportFilterSQLForDataset(DatasetVendors, run.Filter, 6)
	if err != nil {
		return ReportPage{}, fmt.Errorf("build vendor report filter: %w", err)
	}
	query := VendorReportPageSQL(filter, len(args))
	if query == "" {
		return ReportPage{}, fmt.Errorf("vendor report page SQL was empty for filter %q with %d arguments", filter, len(args))
	}
	position, err := decodeVendorReportCursor(cursor)
	if err != nil {
		return ReportPage{}, fmt.Errorf("decode vendor report cursor %q: %w", cursor, err)
	}
	highWater := run.SourceBoundary.SourceHighWater["vendor_relationships"]
	if highWater.IsZero() {
		highWater = run.AsOf
	}
	hasCursor := position.ID != ""
	updatedAt, id := time.Time{}, ""
	if hasCursor {
		updatedAt, id = position.UpdatedAt, position.ID
	}
	queryArgs := append([]any{scope.TenantID, scope.LegalEntityID, run.ScopeRef, run.RequestedByRef, highWater}, args...)
	queryArgs = append(queryArgs, hasCursor, updatedAt, id, limit+1)
	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return ReportPage{}, fmt.Errorf("read bounded vendor report rows: %w", err)
	}
	defer rows.Close()

	page := ReportPage{Rows: make([]ReportRow, 0, limit+1), Columns: []string{
		"tenant_id", "legal_entity_id", "vendor_id", "vendor_name", "trading_name", "registration_ref",
		"jurisdiction", "vendor_status", "service_name", "owner_principal_id", "criticality", "privacy_role",
		"status", "effective_from", "renewal_at", "created_at", "updated_at", "version",
	}}
	for rows.Next() {
		row, err := scanVendorReportRow(rows)
		if err != nil {
			return ReportPage{}, fmt.Errorf("scan bounded vendor report row: %w", err)
		}
		page.Rows = append(page.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return ReportPage{}, fmt.Errorf("read bounded vendor report rows: %w", err)
	}
	if len(page.Rows) > limit {
		page.Rows = page.Rows[:limit]
		page.NextCursor, err = encodeVendorReportCursor(page.Rows[len(page.Rows)-1])
		if err != nil {
			return ReportPage{}, fmt.Errorf("encode vendor report cursor: %w", err)
		}
	}
	return page, nil
}

func scanVendorReportRow(row reportingRowScanner) (ReportRow, error) {
	var id, tenantID, entityID, vendorID, vendorName, tradingName, registrationRef, jurisdiction, vendorStatus string
	var serviceName, ownerID, criticality, privacyRole, status string
	var effectiveFrom, renewalAt pgtype.Timestamptz
	var createdAt, updatedAt time.Time
	var version int64
	if err := row.Scan(
		&id, &tenantID, &entityID, &vendorID, &vendorName, &tradingName, &registrationRef, &jurisdiction,
		&vendorStatus, &serviceName, &ownerID, &criticality, &privacyRole, &status, &effectiveFrom, &renewalAt,
		&createdAt, &updatedAt, &version,
	); err != nil {
		return ReportRow{}, err
	}
	var effective, renewal any
	if effectiveFrom.Valid {
		effective = effectiveFrom.Time.UTC()
	}
	if renewalAt.Valid {
		renewal = renewalAt.Time.UTC()
	}
	return ReportRow{ID: id, Values: map[string]any{
		"tenant_id":          tenantID,
		"legal_entity_id":    entityID,
		"vendor_id":          vendorID,
		"vendor_name":        vendorName,
		"trading_name":       tradingName,
		"registration_ref":   registrationRef,
		"jurisdiction":       jurisdiction,
		"vendor_status":      vendorStatus,
		"service_name":       serviceName,
		"owner_principal_id": ownerID,
		"criticality":        criticality,
		"privacy_role":       privacyRole,
		"status":             status,
		"effective_from":     effective,
		"renewal_at":         renewal,
		"created_at":         createdAt.UTC(),
		"updated_at":         updatedAt.UTC(),
		"version":            version,
	}}, nil
}

type vendorReportCursor struct {
	UpdatedAt time.Time `json:"u"`
	ID        string    `json:"i"`
}

func encodeVendorReportCursor(row ReportRow) (string, error) {
	updated, ok := row.Values["updated_at"].(time.Time)
	if !ok || updated.IsZero() || !isUUID(row.ID) {
		return "", ErrInvalid
	}
	raw, err := json.Marshal(vendorReportCursor{UpdatedAt: updated.UTC(), ID: row.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeVendorReportCursor(value string) (vendorReportCursor, error) {
	if strings.TrimSpace(value) == "" {
		return vendorReportCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) == 0 || len(raw) > 512 {
		return vendorReportCursor{}, ErrInvalid
	}
	var cursor vendorReportCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.UpdatedAt.IsZero() || !isUUID(cursor.ID) {
		return vendorReportCursor{}, ErrInvalid
	}
	return cursor, nil
}
