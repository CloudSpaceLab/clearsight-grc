//go:build postgres

package ropa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const activityColumns = `id::text,
  tenant_id::text,
  legal_entity_id::text,
  code,
  name,
  description,
  status,
  purpose,
  lawful_basis,
  controller,
  processor,
  automated_decision_making,
  data_subject_categories,
  personal_data_categories,
  security_measures,
  retention_period,
  start_date,
  end_date,
  next_review_date,
  COALESCE(owner_principal_id::text, ''),
  COALESCE(required_authority_principal_id::text, ''),
  COALESCE(program_id::text, ''),
  version,
  created_at,
  updated_at`

func getActivitySQL() string {
	return `SELECT ` + activityColumns + `
FROM ropa_processing_activities
WHERE tenant_id = $1::uuid
  AND id = $2::uuid`
}

func activityByCodeSQL() string {
	return `SELECT ` + activityColumns + `
FROM ropa_processing_activities
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND code = $3`
}

func lockActivitySQL() string {
	return `SELECT ` + activityColumns + `
FROM ropa_processing_activities
WHERE tenant_id = $1::uuid
  AND id = $2::uuid
FOR UPDATE`
}

func activityEventsSQL() string {
	return `
SELECT id::text,
       tenant_id::text,
       legal_entity_id::text,
       aggregate_type,
       aggregate_id::text,
       aggregate_version,
       type,
       payload,
       actor_type,
       COALESCE(actor_id::text, ''),
       occurred_at
FROM ropa_events
WHERE tenant_id = $1::uuid
  AND aggregate_type = 'PROCESSING_ACTIVITY'
  AND aggregate_id = $2::uuid
ORDER BY aggregate_version`
}

func updateActivitySQL() string {
	return `
UPDATE ropa_processing_activities
SET code = $1,
    name = $2,
    description = $3,
    purpose = $4,
    lawful_basis = $5,
    controller = $6,
    processor = $7,
    automated_decision_making = $8,
    data_subject_categories = $9,
    personal_data_categories = $10,
    security_measures = $11,
    retention_period = $12,
    start_date = $13::date,
    end_date = $14::date,
    next_review_date = $15::date,
    owner_principal_id = $16::uuid,
    required_authority_principal_id = $17::uuid,
    program_id = $18::uuid,
    status = $19,
    version = $20,
    updated_at = $21
WHERE tenant_id = $22::uuid
  AND legal_entity_id = $23::uuid
  AND id = $24::uuid
  AND version = $25`
}

type rowScanner interface {
	Scan(...any) error
}

func scanActivity(row rowScanner) (ProcessingActivity, error) {
	var activity ProcessingActivity
	var status string
	var startDate, endDate, nextReviewDate pgtype.Date
	if err := row.Scan(
		&activity.ID,
		&activity.TenantID,
		&activity.LegalEntityID,
		&activity.Code,
		&activity.Name,
		&activity.Description,
		&status,
		&activity.Purpose,
		&activity.LawfulBasis,
		&activity.Controller,
		&activity.Processor,
		&activity.AutomatedDecisionMaking,
		&activity.DataSubjectCategories,
		&activity.PersonalDataCategories,
		&activity.SecurityMeasures,
		&activity.RetentionPeriod,
		&startDate,
		&endDate,
		&nextReviewDate,
		&activity.OwnerPrincipalID,
		&activity.RequiredAuthorityPrincipalID,
		&activity.ProgramID,
		&activity.Version,
		&activity.CreatedAt,
		&activity.UpdatedAt,
	); err != nil {
		return ProcessingActivity{}, err
	}
	activity.Status = Status(status)
	activity.StartDate = nullableDate(startDate)
	activity.EndDate = nullableDate(endDate)
	activity.NextReviewDate = nullableDate(nextReviewDate)
	return activity, nil
}

func nullableDate(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}

// PostgresLister reads only normalized current rows. It applies tenant,
// legal-entity, and every caller filter before requesting limit+1 rows.
type PostgresLister struct {
	pool *pgxpool.Pool
}

func NewPostgresLister(pool *pgxpool.Pool) *PostgresLister {
	return &PostgresLister{pool: pool}
}

func (l *PostgresLister) ListActivities(ctx context.Context, filter ListActivitiesFilter) (ActivityPage, error) {
	if l == nil || l.pool == nil || ctx == nil {
		return ActivityPage{}, ErrInvalid
	}
	if err := ropaContextError(ctx); err != nil {
		return ActivityPage{}, err
	}
	filter.TenantID = strings.TrimSpace(filter.TenantID)
	filter.LegalEntityID = strings.TrimSpace(filter.LegalEntityID)
	filter.Status = Status(strings.ToUpper(strings.TrimSpace(string(filter.Status))))
	filter.LawfulBasis = strings.TrimSpace(filter.LawfulBasis)
	filter.OwnerPrincipalID = strings.TrimSpace(filter.OwnerPrincipalID)
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	if filter.TenantID == "" || filter.LegalEntityID == "" {
		return ActivityPage{}, ErrInvalid
	}
	if filter.Status != "" && !validStatus(filter.Status) {
		return ActivityPage{}, ErrInvalid
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 200 {
		filter.Limit = 200
	}

	hasCursor := filter.Cursor != ""
	cursorRank := 0
	cursorDate := "0001-01-01"
	cursorID := "00000000-0000-0000-0000-000000000000"
	if hasCursor {
		position, err := decodeActivityCursor(filter.Cursor)
		if err != nil {
			return ActivityPage{}, ErrInvalid
		}
		var opaqueID pgtype.UUID
		if err := opaqueID.Scan(position.ID); err != nil || !opaqueID.Valid {
			return ActivityPage{}, ErrInvalid
		}
		cursorRank = activityStatusRank(Status(position.Status))
		cursorDate = position.NextReviewDate.UTC().Format("2006-01-02")
		cursorID = position.ID
	}

	rows, err := l.pool.Query(ctx, ListActivitiesSQL(),
		filter.TenantID,
		filter.LegalEntityID,
		string(filter.Status),
		filter.LawfulBasis,
		filter.OwnerPrincipalID,
		filter.Search,
		filter.IncludeRetired,
		hasCursor,
		cursorRank,
		cursorDate,
		cursorID,
		filter.Limit+1,
	)
	if err != nil {
		return ActivityPage{}, fmt.Errorf("list current processing activities: %w", err)
	}
	defer rows.Close()

	page := ActivityPage{Rows: make([]ProcessingActivity, 0, filter.Limit+1)}
	for rows.Next() {
		activity, err := scanActivity(rows)
		if err != nil {
			return ActivityPage{}, fmt.Errorf("scan current processing activity list: %w", err)
		}
		page.Rows = append(page.Rows, activity)
	}
	if err := rows.Err(); err != nil {
		return ActivityPage{}, fmt.Errorf("list current processing activities: %w", err)
	}
	if len(page.Rows) > filter.Limit {
		page.Rows = page.Rows[:filter.Limit]
		page.NextCursor, err = encodeActivityCursor(page.Rows[len(page.Rows)-1])
		if err != nil {
			return ActivityPage{}, err
		}
		page.HasMore = true
	}
	return page, nil
}

// PostgresSummaryRepository serves the stored dashboard projection. Its write
// is monotonic, but a production worker must additionally lease each
// (tenant_id, legal_entity_id) scope so only one rebuild owns the scope at a
// time.
type PostgresSummaryRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSummaryRepository(pool *pgxpool.Pool) *PostgresSummaryRepository {
	return &PostgresSummaryRepository{pool: pool}
}

func (r *PostgresSummaryRepository) LatestSummary(ctx context.Context, tenantID, legalEntityID string) (RegisterSummary, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return RegisterSummary{}, ErrInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return RegisterSummary{}, ErrInvalid
	}

	var summary RegisterSummary
	var excluded, unknown pgtype.Int4
	var counts []byte
	err := r.pool.QueryRow(ctx, latestSummarySQL(), tenantID, legalEntityID).Scan(
		&summary.GeneratedAt,
		&summary.ProjectionVersion,
		&summary.SourceHighWater,
		&summary.Coverage.Population,
		&excluded,
		&unknown,
		&counts,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RegisterSummary{}, ErrNotFound
	}
	if err != nil {
		return RegisterSummary{}, fmt.Errorf("read latest processing activity summary: %w", err)
	}
	if err := json.Unmarshal(counts, &summary.Counts); err != nil {
		return RegisterSummary{}, fmt.Errorf("decode processing activity summary counts: %w", err)
	}
	if excluded.Valid {
		value := int(excluded.Int32)
		summary.Coverage.Excluded = &value
	}
	if unknown.Valid {
		value := int(unknown.Int32)
		summary.Coverage.Unknown = &value
	}
	summary.TenantID = tenantID
	summary.LegalEntityID = legalEntityID
	summary.GeneratedAt = summary.GeneratedAt.UTC()
	summary.SourceHighWater = summary.SourceHighWater.UTC()
	return summary, nil
}

func (r *PostgresSummaryRepository) ReplaceSummary(ctx context.Context, summary RegisterSummary) error {
	if r == nil || r.pool == nil || ctx == nil {
		return ErrInvalid
	}
	summary.TenantID = strings.TrimSpace(summary.TenantID)
	summary.LegalEntityID = strings.TrimSpace(summary.LegalEntityID)
	if summary.TenantID == "" || summary.LegalEntityID == "" {
		return ErrInvalid
	}
	counts, err := json.Marshal(summary.Counts)
	if err != nil {
		return fmt.Errorf("encode processing activity summary counts: %w", err)
	}
	_, err = r.pool.Exec(ctx, replaceSummarySQL(),
		summary.TenantID,
		summary.LegalEntityID,
		summary.GeneratedAt,
		summary.ProjectionVersion,
		summary.SourceHighWater,
		summary.Coverage.Population,
		nullableInt(summary.Coverage.Excluded),
		nullableInt(summary.Coverage.Unknown),
		counts,
	)
	if err != nil {
		return fmt.Errorf("replace processing activity summary: %w", err)
	}
	return nil
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

var _ ActivityLister = (*PostgresLister)(nil)
var _ SummaryRepository = (*PostgresSummaryRepository)(nil)
