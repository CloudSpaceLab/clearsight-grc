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
  AND legal_entity_id = $2::uuid
  AND id = $3::uuid`
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
  AND legal_entity_id = $2::uuid
  AND id = $3::uuid
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
  AND legal_entity_id = $2::uuid
  AND aggregate_type = 'PROCESSING_ACTIVITY'
  AND aggregate_id = $3::uuid
  AND aggregate_version > $4
ORDER BY aggregate_version
LIMIT $5`
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

type activityChildQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func activityDataCategoriesSQL() string {
	return `
SELECT activity_id::text, category, sensitivity
FROM ropa_processing_activity_data_categories
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id=ANY($3::uuid[])
ORDER BY activity_id, category`
}

func activityRecipientsSQL() string {
	return `
SELECT activity_id::text,
       recipient,
       recipient_kind,
       COALESCE(country_code, ''),
       is_cross_border,
       transfer_basis
FROM ropa_processing_activity_recipients
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id=ANY($3::uuid[])
ORDER BY activity_id, recipient`
}

func activitySystemsSQL() string {
	return `
SELECT activity_id::text, system_name, system_kind
FROM ropa_processing_activity_systems
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id=ANY($3::uuid[])
ORDER BY activity_id, system_name`
}

func activityReviewsSQL() string {
	return `
SELECT activity_id::text,
       id::text,
       due_date,
       completed_at,
       COALESCE(outcome, ''),
       COALESCE(reviewer_principal_id::text, ''),
       created_at
FROM ropa_processing_activity_reviews
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id=ANY($3::uuid[])
ORDER BY activity_id, created_at, id`
}

func activityPrincipalNamesSQL() string {
	return `
SELECT id::text, display_name
FROM principals
WHERE tenant_id = $1::uuid
  AND id=ANY($2::uuid[])`
}

func loadActivityChildren(ctx context.Context, queryer activityChildQueryer, activity *ProcessingActivity) error {
	if activity == nil {
		return ErrInvalid
	}
	return loadActivityChildrenForScopes(ctx, queryer, activity.TenantID, activity.LegalEntityID, []*ProcessingActivity{activity})
}

// loadActivityChildrenForPage uses four set-based reads for the materialized
// page. The child population is bounded by the requested page size and the
// domain's per-collection bounds; it never expands into an unbounded scan.
func loadActivityChildrenForPage(ctx context.Context, queryer activityChildQueryer, activities []ProcessingActivity) error {
	if len(activities) == 0 {
		return nil
	}
	pointers := make([]*ProcessingActivity, len(activities))
	for index := range activities {
		pointers[index] = &activities[index]
	}
	return loadActivityChildrenForScopes(ctx, queryer, activities[0].TenantID, activities[0].LegalEntityID, pointers)
}

func loadActivityChildrenForScopes(ctx context.Context, queryer activityChildQueryer, tenantID, legalEntityID string, activities []*ProcessingActivity) error {
	if queryer == nil || len(activities) == 0 {
		return nil
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return ErrInvalid
	}
	byID := make(map[string]*ProcessingActivity, len(activities))
	activityIDs := make([]string, 0, len(activities))
	for _, activity := range activities {
		if activity == nil || activity.TenantID != tenantID || activity.LegalEntityID != legalEntityID || activity.ID == "" {
			return ErrInvalid
		}
		if _, exists := byID[activity.ID]; exists {
			continue
		}
		byID[activity.ID] = activity
		activityIDs = append(activityIDs, activity.ID)
	}

	categoryRows, err := queryer.Query(ctx, activityDataCategoriesSQL(), tenantID, legalEntityID, activityIDs)
	if err != nil {
		return fmt.Errorf("read processing activity data categories: %w", err)
	}
	for categoryRows.Next() {
		var activityID string
		var category DataCategory
		if err := categoryRows.Scan(&activityID, &category.Category, &category.Sensitivity); err != nil {
			categoryRows.Close()
			return fmt.Errorf("scan processing activity data category: %w", err)
		}
		activity := byID[activityID]
		if activity == nil {
			categoryRows.Close()
			return fmt.Errorf("processing activity data category is outside requested page: %s", activityID)
		}
		activity.DataCategories = append(activity.DataCategories, category)
	}
	if err := categoryRows.Err(); err != nil {
		categoryRows.Close()
		return fmt.Errorf("read processing activity data categories: %w", err)
	}
	categoryRows.Close()

	recipientRows, err := queryer.Query(ctx, activityRecipientsSQL(), tenantID, legalEntityID, activityIDs)
	if err != nil {
		return fmt.Errorf("read processing activity recipients: %w", err)
	}
	for recipientRows.Next() {
		var activityID, countryCode, transferBasis string
		var recipient Recipient
		if err := recipientRows.Scan(
			&activityID,
			&recipient.Recipient,
			&recipient.RecipientKind,
			&countryCode,
			&recipient.IsCrossBorder,
			&transferBasis,
		); err != nil {
			recipientRows.Close()
			return fmt.Errorf("scan processing activity recipient: %w", err)
		}
		activity := byID[activityID]
		if activity == nil {
			recipientRows.Close()
			return fmt.Errorf("processing activity recipient is outside requested page: %s", activityID)
		}
		recipient.CountryCode = countryCode
		recipient.TransferBasis = TransferBasis(transferBasis)
		activity.Recipients = append(activity.Recipients, recipient)
	}
	if err := recipientRows.Err(); err != nil {
		recipientRows.Close()
		return fmt.Errorf("read processing activity recipients: %w", err)
	}
	recipientRows.Close()

	systemRows, err := queryer.Query(ctx, activitySystemsSQL(), tenantID, legalEntityID, activityIDs)
	if err != nil {
		return fmt.Errorf("read processing activity systems: %w", err)
	}
	for systemRows.Next() {
		var activityID string
		var system System
		if err := systemRows.Scan(&activityID, &system.SystemName, &system.SystemKind); err != nil {
			systemRows.Close()
			return fmt.Errorf("scan processing activity system: %w", err)
		}
		activity := byID[activityID]
		if activity == nil {
			systemRows.Close()
			return fmt.Errorf("processing activity system is outside requested page: %s", activityID)
		}
		activity.Systems = append(activity.Systems, system)
	}
	if err := systemRows.Err(); err != nil {
		systemRows.Close()
		return fmt.Errorf("read processing activity systems: %w", err)
	}
	systemRows.Close()

	reviewRows, err := queryer.Query(ctx, activityReviewsSQL(), tenantID, legalEntityID, activityIDs)
	if err != nil {
		return fmt.Errorf("read processing activity reviews: %w", err)
	}
	for reviewRows.Next() {
		var activityID string
		var review Review
		var dueDate pgtype.Date
		var completedAt pgtype.Timestamptz
		if err := reviewRows.Scan(
			&activityID,
			&review.ID,
			&dueDate,
			&completedAt,
			&review.Outcome,
			&review.ReviewerPrincipalID,
			&review.CreatedAt,
		); err != nil {
			reviewRows.Close()
			return fmt.Errorf("scan processing activity review: %w", err)
		}
		if !dueDate.Valid {
			reviewRows.Close()
			return fmt.Errorf("processing activity review has no due date: %s", review.ID)
		}
		review.DueDate = dueDate.Time.UTC()
		review.CompletedAt = nullableTimestamptz(completedAt)
		review.CreatedAt = review.CreatedAt.UTC()
		activity := byID[activityID]
		if activity == nil {
			reviewRows.Close()
			return fmt.Errorf("processing activity review is outside requested page: %s", activityID)
		}
		activity.Reviews = append(activity.Reviews, review)
	}
	if err := reviewRows.Err(); err != nil {
		reviewRows.Close()
		return fmt.Errorf("read processing activity reviews: %w", err)
	}
	reviewRows.Close()

	principalIDs := make([]string, 0, len(activities)*2)
	seenPrincipals := make(map[string]struct{}, len(activities)*2)
	addPrincipal := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, exists := seenPrincipals[id]; exists {
			return
		}
		seenPrincipals[id] = struct{}{}
		principalIDs = append(principalIDs, id)
	}
	for _, activity := range byID {
		addPrincipal(activity.OwnerPrincipalID)
		addPrincipal(activity.RequiredAuthorityPrincipalID)
		for _, review := range activity.Reviews {
			addPrincipal(review.ReviewerPrincipalID)
		}
	}
	if len(principalIDs) > 0 {
		principalRows, err := queryer.Query(ctx, activityPrincipalNamesSQL(), tenantID, principalIDs)
		if err != nil {
			return fmt.Errorf("read processing activity principal names: %w", err)
		}
		names := make(map[string]string, len(principalIDs))
		for principalRows.Next() {
			var id, displayName string
			if err := principalRows.Scan(&id, &displayName); err != nil {
				principalRows.Close()
				return fmt.Errorf("scan processing activity principal name: %w", err)
			}
			names[id] = strings.TrimSpace(displayName)
		}
		if err := principalRows.Err(); err != nil {
			principalRows.Close()
			return fmt.Errorf("read processing activity principal names: %w", err)
		}
		principalRows.Close()

		for _, activity := range byID {
			activity.OwnerDisplayName = names[activity.OwnerPrincipalID]
			activity.RequiredAuthorityDisplayName = names[activity.RequiredAuthorityPrincipalID]
			for index := range activity.Reviews {
				activity.Reviews[index].ReviewerDisplayName = names[activity.Reviews[index].ReviewerPrincipalID]
			}
		}
	}

	for _, activity := range byID {
		*activity = normalizeProcessingActivity(*activity)
	}
	return nil
}

func nullableTimestamptz(value pgtype.Timestamptz) *time.Time {
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

func (l *PostgresLister) ListActivities(ctx context.Context, scope ActivityScope, filter ListActivitiesFilter) (ActivityPage, error) {
	if l == nil || l.pool == nil || ctx == nil {
		return ActivityPage{}, ErrInvalid
	}
	if err := ropaContextError(ctx); err != nil {
		return ActivityPage{}, err
	}
	scope, err := normalizeActivityScope(scope)
	if err != nil {
		return ActivityPage{}, err
	}
	filter.Status = Status(strings.ToUpper(strings.TrimSpace(string(filter.Status))))
	filter.LawfulBasis = strings.TrimSpace(filter.LawfulBasis)
	filter.OwnerPrincipalID = strings.TrimSpace(filter.OwnerPrincipalID)
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Cursor = strings.TrimSpace(filter.Cursor)
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
		scope.TenantID,
		scope.LegalEntityID,
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
	// Child hydration after truncation is bounded by the requested page size.
	if err := loadActivityChildrenForPage(ctx, l.pool, page.Rows); err != nil {
		return ActivityPage{}, err
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
