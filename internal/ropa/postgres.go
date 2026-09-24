//go:build postgres

package ropa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository stores each material command in one transaction: the
// current row, its normalized child rows, immutable event, immutable revision,
// and transactional outbox row either all commit or all roll back.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateActivity(ctx context.Context, activity ProcessingActivity, event Event) (ProcessingActivity, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	if err := ropaContextError(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	if err := validateActivityWhitespace(activity); err != nil {
		return ProcessingActivity{}, err
	}
	activity = normalizeProcessingActivity(activity)
	if activity.Status != StatusNew || activity.Version != 1 {
		return ProcessingActivity{}, ErrInvalid
	}
	if err := validateActivity(activity); err != nil {
		return ProcessingActivity{}, err
	}
	if activity.ID == "" {
		generated, err := newActivityID()
		if err != nil {
			return ProcessingActivity{}, err
		}
		activity.ID = generated
	}
	event, err := normalizeCreatedEvent(event, activity)
	if err != nil {
		return ProcessingActivity{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ProcessingActivity{}, fmt.Errorf("begin processing activity create: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, buildInsertActivitySQL(),
		activity.ID,
		activity.TenantID,
		activity.LegalEntityID,
		activity.Code,
		activity.Name,
		activity.Description,
		string(activity.Status),
		activity.Purpose,
		activity.LawfulBasis,
		activity.Controller,
		activity.Processor,
		activity.AutomatedDecisionMaking,
		activity.DataSubjectCategories,
		activity.PersonalDataCategories,
		activity.SecurityMeasures,
		activity.RetentionPeriod,
		activity.StartDate,
		activity.EndDate,
		activity.NextReviewDate,
		nullIfEmpty(activity.OwnerPrincipalID),
		nullIfEmpty(activity.RequiredAuthorityPrincipalID),
		nullIfEmpty(activity.ProgramID),
		activity.Version,
		activity.CreatedAt,
		activity.UpdatedAt,
	)
	if err != nil {
		return ProcessingActivity{}, fmt.Errorf("insert current processing activity: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ProcessingActivity{}, ErrDuplicate
	}
	if err := insertActivityChildren(ctx, tx, activity); err != nil {
		return ProcessingActivity{}, err
	}
	if err := appendActivityHistory(ctx, tx, activity, event); err != nil {
		return ProcessingActivity{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProcessingActivity{}, fmt.Errorf("commit processing activity create: %w", err)
	}
	return cloneProcessingActivity(activity), nil
}

func (r *PostgresRepository) GetActivity(ctx context.Context, scope ActivityScope, activityID string) (ProcessingActivity, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" {
		return ProcessingActivity{}, ErrInvalid
	}
	activity, err := scanActivity(r.pool.QueryRow(ctx, getActivitySQL(), scope.TenantID, scope.LegalEntityID, activityID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessingActivity{}, ErrNotFound
	}
	if err != nil {
		return ProcessingActivity{}, fmt.Errorf("read current processing activity: %w", err)
	}
	if err := loadActivityChildren(ctx, r.pool, &activity); err != nil {
		return ProcessingActivity{}, err
	}
	return cloneProcessingActivity(activity), nil
}

func (r *PostgresRepository) ActivityByCode(ctx context.Context, scope ActivityScope, code string) (ProcessingActivity, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	scope, err := normalizeActivityScope(scope)
	code = strings.TrimSpace(code)
	if err != nil || code == "" {
		return ProcessingActivity{}, ErrInvalid
	}
	activity, err := scanActivity(r.pool.QueryRow(ctx, activityByCodeSQL(), scope.TenantID, scope.LegalEntityID, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessingActivity{}, ErrNotFound
	}
	if err != nil {
		return ProcessingActivity{}, fmt.Errorf("read processing activity by code: %w", err)
	}
	return cloneProcessingActivity(activity), nil
}

func (r *PostgresRepository) ApplyActivityEvent(ctx context.Context, scope ActivityScope, activityID string, expectedVersion int64, event Event) (int64, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return 0, ErrInvalid
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" || expectedVersion <= 0 {
		return 0, ErrInvalid
	}
	if strings.TrimSpace(event.ID) == "" {
		generated, generationErr := newEventID()
		if generationErr != nil {
			return 0, generationErr
		}
		event.ID = generated
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin processing activity event: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := scanActivity(tx.QueryRow(ctx, lockActivitySQL(), scope.TenantID, scope.LegalEntityID, activityID))
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock current processing activity: %w", err)
	}
	next, event, err := prepareActivityEvent(current, expectedVersion, event)
	if err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, updateActivitySQL(),
		next.Code,
		next.Name,
		next.Description,
		next.Purpose,
		next.LawfulBasis,
		next.Controller,
		next.Processor,
		next.AutomatedDecisionMaking,
		next.DataSubjectCategories,
		next.PersonalDataCategories,
		next.SecurityMeasures,
		next.RetentionPeriod,
		next.StartDate,
		next.EndDate,
		next.NextReviewDate,
		nullIfEmpty(next.OwnerPrincipalID),
		nullIfEmpty(next.RequiredAuthorityPrincipalID),
		nullIfEmpty(next.ProgramID),
		string(next.Status),
		next.Version,
		next.UpdatedAt,
		current.TenantID,
		current.LegalEntityID,
		current.ID,
		expectedVersion,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicate
		}
		return 0, fmt.Errorf("update current processing activity: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return 0, ErrVersionConflict
	}
	// Event payloads are complete aggregate snapshots, so this is a full
	// replacement rather than a diff. The parent update, child replacement,
	// event, revision, and outbox row commit or roll back together.
	if err := replaceActivityChildren(ctx, tx, next); err != nil {
		return 0, err
	}
	if err := appendActivityHistory(ctx, tx, next, event); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit processing activity event: %w", err)
	}
	return next.Version, nil
}

func (r *PostgresRepository) ActivityEvents(ctx context.Context, scope ActivityScope, activityID string, afterVersion int64, limit int) ([]Event, bool, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return nil, false, ErrInvalid
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" || afterVersion < 0 || limit < 0 {
		return nil, false, ErrInvalid
	}
	if limit == 0 {
		limit = DefaultActivityEventPageSize
	}
	if limit > MaxActivityEventPageSize {
		limit = MaxActivityEventPageSize
	}
	if _, err := scanActivity(r.pool.QueryRow(ctx, getActivitySQL(), scope.TenantID, scope.LegalEntityID, activityID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrNotFound
		}
		return nil, false, fmt.Errorf("verify processing activity for history: %w", err)
	}
	rows, err := r.pool.Query(ctx, activityEventsSQL(), scope.TenantID, scope.LegalEntityID, activityID, afterVersion, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("read processing activity events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0, limit+1)
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID,
			&event.TenantID,
			&event.LegalEntityID,
			&event.AggregateType,
			&event.AggregateID,
			&event.AggregateVersion,
			&event.Type,
			&event.Payload,
			&event.ActorType,
			&event.ActorID,
			&event.OccurredAt,
		); err != nil {
			return nil, false, fmt.Errorf("scan processing activity event: %w", err)
		}
		event.OccurredAt = event.OccurredAt.UTC()
		event.Payload = append(json.RawMessage(nil), event.Payload...)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("read processing activity events: %w", err)
	}
	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}
	return events, hasMore, nil
}

type activityChildInsertStatement struct {
	name   string
	query  string
	values []any
}

// buildActivityChildInsertStatements creates one fixed-arity statement per
// child row. The activity child bound is 1,000 rows per collection, so a
// per-row insert is deliberately simpler and safer than constructing an
// unbounded multi-row VALUES expression. Every statement is executed in the
// caller's transaction, so the parent, children, event, revision and outbox
// row still commit or roll back together.
func buildActivityChildInsertStatements(activity ProcessingActivity) []activityChildInsertStatement {
	statements := make([]activityChildInsertStatement, 0,
		len(activity.DataCategories)+len(activity.Recipients)+len(activity.Systems)+len(activity.Reviews))
	for _, category := range activity.DataCategories {
		statements = append(statements, activityChildInsertStatement{
			name:  "data categories",
			query: buildInsertActivityDataCategoriesSQL(),
			values: []any{
				activity.TenantID,
				activity.LegalEntityID,
				activity.ID,
				category.Category,
				category.Sensitivity,
			},
		})
	}
	for _, recipient := range activity.Recipients {
		statements = append(statements, activityChildInsertStatement{
			name:  "recipients",
			query: buildInsertActivityRecipientsSQL(),
			values: []any{
				activity.TenantID,
				activity.LegalEntityID,
				activity.ID,
				recipient.Recipient,
				recipient.RecipientKind,
				nullIfEmpty(recipient.CountryCode),
				recipient.IsCrossBorder,
				string(recipient.TransferBasis),
			},
		})
	}
	for _, system := range activity.Systems {
		statements = append(statements, activityChildInsertStatement{
			name:  "systems",
			query: buildInsertActivitySystemsSQL(),
			values: []any{
				activity.TenantID,
				activity.LegalEntityID,
				activity.ID,
				system.SystemName,
				system.SystemKind,
			},
		})
	}
	for _, review := range activity.Reviews {
		statements = append(statements, activityChildInsertStatement{
			name:  "reviews",
			query: buildInsertActivityReviewsSQL(),
			values: []any{
				review.ID,
				activity.TenantID,
				activity.LegalEntityID,
				activity.ID,
				review.DueDate,
				review.CompletedAt,
				nullIfEmpty(review.Outcome),
				nullIfEmpty(review.ReviewerPrincipalID),
				review.CreatedAt,
			},
		})
	}
	return statements
}

func insertActivityChildren(ctx context.Context, tx pgx.Tx, activity ProcessingActivity) error {
	for _, statement := range buildActivityChildInsertStatements(activity) {
		if err := execActivityChildInsert(ctx, tx, statement.name, statement.query, statement.values); err != nil {
			return err
		}
	}
	return nil
}

func execActivityChildInsert(ctx context.Context, tx pgx.Tx, name, query string, values []any) error {
	tag, err := tx.Exec(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("insert processing activity %s: %w", name, err)
	}
	if tag.RowsAffected() != 1 {
		return ErrDuplicate
	}
	return nil
}

// replaceActivityChildren performs a full replacement rather than a diff. The
// event payload is the complete aggregate snapshot, so an omitted collection
// represents an empty current collection.
func replaceActivityChildren(ctx context.Context, tx pgx.Tx, activity ProcessingActivity) error {
	for _, statement := range buildDeleteActivityChildrenSQL() {
		if _, err := tx.Exec(ctx, statement, activity.TenantID, activity.LegalEntityID, activity.ID); err != nil {
			return fmt.Errorf("delete current processing activity children: %w", err)
		}
	}
	return insertActivityChildren(ctx, tx, activity)
}

func buildInsertActivityDataCategoriesSQL() string {
	return `
INSERT INTO ropa_processing_activity_data_categories (
  tenant_id,
  legal_entity_id,
  activity_id,
  category,
  sensitivity
)
VALUES (
  $1::uuid,
  $2::uuid,
  $3::uuid,
  $4,
  $5
)
ON CONFLICT DO NOTHING
`
}

func buildInsertActivityRecipientsSQL() string {
	return `
INSERT INTO ropa_processing_activity_recipients (
  tenant_id,
  legal_entity_id,
  activity_id,
  recipient,
  recipient_kind,
  country_code,
  is_cross_border,
  transfer_basis
)
VALUES (
  $1::uuid,
  $2::uuid,
  $3::uuid,
  $4,
  $5,
  NULLIF($6, ''),
  $7,
  $8
)
ON CONFLICT DO NOTHING
`
}

func buildInsertActivitySystemsSQL() string {
	return `
INSERT INTO ropa_processing_activity_systems (
  tenant_id,
  legal_entity_id,
  activity_id,
  system_name,
  system_kind
)
VALUES (
  $1::uuid,
  $2::uuid,
  $3::uuid,
  $4,
  $5
)
ON CONFLICT DO NOTHING
`
}

func buildInsertActivityReviewsSQL() string {
	return `
INSERT INTO ropa_processing_activity_reviews (
  id,
  tenant_id,
  legal_entity_id,
  activity_id,
  due_date,
  completed_at,
  outcome,
  reviewer_principal_id,
  created_at
)
VALUES (
  $1::uuid,
  $2::uuid,
  $3::uuid,
  $4::uuid,
  $5::date,
  $6::timestamptz,
  NULLIF($7, ''),
  NULLIF($8, '')::uuid,
  $9
)
ON CONFLICT DO NOTHING
`
}

func buildDeleteActivityChildrenSQL() []string {
	return []string{
		`DELETE FROM ropa_processing_activity_data_categories
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id = $3::uuid`,
		`DELETE FROM ropa_processing_activity_recipients
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id = $3::uuid`,
		`DELETE FROM ropa_processing_activity_systems
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id = $3::uuid`,
		`DELETE FROM ropa_processing_activity_reviews
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
  AND activity_id = $3::uuid`,
	}
}

func appendActivityHistory(ctx context.Context, tx pgx.Tx, activity ProcessingActivity, event Event) error {
	if _, err := tx.Exec(ctx, buildAppendEventSQL(),
		event.ID,
		event.TenantID,
		event.LegalEntityID,
		event.AggregateType,
		event.AggregateID,
		event.AggregateVersion,
		event.Type,
		event.Payload,
		event.ActorType,
		nullIfEmpty(event.ActorID),
		event.OccurredAt,
	); err != nil {
		if isUniqueViolation(err) {
			return ErrVersionConflict
		}
		return fmt.Errorf("append processing activity event: %w", err)
	}

	snapshot, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("encode processing activity revision: %w", err)
	}
	if _, err := tx.Exec(ctx, buildRevisionSQL(),
		activity.TenantID,
		activity.LegalEntityID,
		activity.ID,
		activity.Version,
		snapshot,
		event.OccurredAt,
	); err != nil {
		if isUniqueViolation(err) {
			return ErrVersionConflict
		}
		return fmt.Errorf("insert processing activity revision: %w", err)
	}

	outboxID, err := newEventID()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, buildOutboxSQL(),
		outboxID,
		event.TenantID,
		event.AggregateType,
		event.AggregateID,
		event.Type,
		event.Payload,
		event.OccurredAt,
	); err != nil {
		return fmt.Errorf("append processing activity outbox row: %w", err)
	}
	return nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

var _ Repository = (*PostgresRepository)(nil)
