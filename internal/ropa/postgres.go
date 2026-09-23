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
// current row, immutable event, immutable revision, and transactional outbox
// row either all commit or all roll back.
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
	if activity.Status != StatusNew {
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
	if err := appendActivityHistory(ctx, tx, activity, event); err != nil {
		return ProcessingActivity{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProcessingActivity{}, fmt.Errorf("commit processing activity create: %w", err)
	}
	return cloneProcessingActivity(activity), nil
}

func (r *PostgresRepository) GetActivity(ctx context.Context, tenantID, activityID string) (ProcessingActivity, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	activityID = strings.TrimSpace(activityID)
	if tenantID == "" || activityID == "" {
		return ProcessingActivity{}, ErrInvalid
	}
	activity, err := scanActivity(r.pool.QueryRow(ctx, getActivitySQL(), tenantID, activityID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessingActivity{}, ErrNotFound
	}
	if err != nil {
		return ProcessingActivity{}, fmt.Errorf("read current processing activity: %w", err)
	}
	return cloneProcessingActivity(activity), nil
}

func (r *PostgresRepository) ActivityByCode(ctx context.Context, tenantID, legalEntityID, code string) (ProcessingActivity, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	code = strings.TrimSpace(code)
	if tenantID == "" || legalEntityID == "" || code == "" {
		return ProcessingActivity{}, ErrInvalid
	}
	activity, err := scanActivity(r.pool.QueryRow(ctx, activityByCodeSQL(), tenantID, legalEntityID, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessingActivity{}, ErrNotFound
	}
	if err != nil {
		return ProcessingActivity{}, fmt.Errorf("read processing activity by code: %w", err)
	}
	return cloneProcessingActivity(activity), nil
}

func (r *PostgresRepository) ApplyActivityEvent(ctx context.Context, tenantID, activityID string, expectedVersion int64, event Event) (int64, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return 0, ErrInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	activityID = strings.TrimSpace(activityID)
	if tenantID == "" || activityID == "" || expectedVersion <= 0 {
		return 0, ErrInvalid
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin processing activity event: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := scanActivity(tx.QueryRow(ctx, lockActivitySQL(), tenantID, activityID))
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock current processing activity: %w", err)
	}
	if current.Version != expectedVersion || event.AggregateVersion != expectedVersion+1 {
		return 0, ErrVersionConflict
	}

	next, event, err := prepareAppliedActivityEvent(current, expectedVersion, event)
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
	if err := appendActivityHistory(ctx, tx, next, event); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit processing activity event: %w", err)
	}
	return next.Version, nil
}

func (r *PostgresRepository) ActivityEvents(ctx context.Context, tenantID, activityID string) ([]Event, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return nil, ErrInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	activityID = strings.TrimSpace(activityID)
	if tenantID == "" || activityID == "" {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, activityEventsSQL(), tenantID, activityID)
	if err != nil {
		return nil, fmt.Errorf("read processing activity events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)
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
			return nil, fmt.Errorf("scan processing activity event: %w", err)
		}
		event.OccurredAt = event.OccurredAt.UTC()
		event.Payload = append(json.RawMessage(nil), event.Payload...)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read processing activity events: %w", err)
	}
	if len(events) == 0 {
		return nil, ErrNotFound
	}
	return events, nil
}

func prepareAppliedActivityEvent(current ProcessingActivity, expectedVersion int64, event Event) (ProcessingActivity, Event, error) {
	if !validActivityEventType(event.Type) {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if event.TenantID != "" && event.TenantID != current.TenantID {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if event.LegalEntityID != "" && event.LegalEntityID != current.LegalEntityID {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if event.AggregateID != "" && event.AggregateID != current.ID {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if event.AggregateType != "" && event.AggregateType != "PROCESSING_ACTIVITY" {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if strings.TrimSpace(event.ID) == "" {
		generated, err := newEventID()
		if err != nil {
			return ProcessingActivity{}, Event{}, err
		}
		event.ID = generated
	}

	next, err := decodeActivity(event.Payload)
	if err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	if err := validateActivityWhitespace(next); err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	next = normalizeProcessingActivity(next)
	if next.ID != "" && next.ID != current.ID {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if next.TenantID != "" && next.TenantID != current.TenantID {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if next.LegalEntityID != "" && next.LegalEntityID != current.LegalEntityID {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	next.ID = current.ID
	next.TenantID = current.TenantID
	next.LegalEntityID = current.LegalEntityID
	next.Version = expectedVersion + 1
	next.CreatedAt = current.CreatedAt
	if !event.OccurredAt.IsZero() {
		next.UpdatedAt = event.OccurredAt.UTC()
	} else if next.UpdatedAt.IsZero() {
		next.UpdatedAt = current.UpdatedAt
	}
	if err := validateActivity(next); err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	if err := ValidateTransitionForWrite(current, next, event.Type); err != nil {
		return ProcessingActivity{}, Event{}, err
	}

	event.TenantID = current.TenantID
	event.LegalEntityID = current.LegalEntityID
	event.AggregateType = "PROCESSING_ACTIVITY"
	event.AggregateID = current.ID
	event.AggregateVersion = expectedVersion + 1
	event.ActorID = strings.TrimSpace(event.ActorID)
	if event.ActorID == "" {
		event.ActorType = "SERVICE"
	} else {
		event.ActorType = "USER"
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = next.UpdatedAt
	}
	event.OccurredAt = event.OccurredAt.UTC()
	event.Payload = mustMarshalActivity(next)
	return next, event, nil
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
