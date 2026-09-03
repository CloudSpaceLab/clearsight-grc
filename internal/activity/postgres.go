//go:build postgres

package activity

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) List(ctx context.Context, query Query) (Page, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(query.TenantID) == "" {
		return Page{}, ErrInvalid
	}
	args := []any{query.TenantID}
	clauses := []string{"TRUE"}
	add := func(value any, clause func(string) string) {
		args = append(args, value)
		clauses = append(clauses, clause("$"+strconv.Itoa(len(args))))
	}
	if query.Cursor != "" {
		add(query.Cursor, func(placeholder string) string {
			return "(oe.occurred_at,oe.id)<(SELECT c.occurred_at,c.id FROM oe c WHERE c.id=" + placeholder + ")"
		})
	}
	if query.From != nil {
		add(*query.From, func(placeholder string) string { return "oe.occurred_at>=" + placeholder })
	}
	if query.To != nil {
		add(*query.To, func(placeholder string) string { return "oe.occurred_at<=" + placeholder })
	}
	if query.Category != "" {
		add(query.Category, func(placeholder string) string { return "(" + categoryExpression + ")=" + placeholder })
	}
	if query.EventType != "" {
		add(query.EventType, func(placeholder string) string { return "upper(oe.event_type)=upper(" + placeholder + ")" })
	}
	if query.ObjectType != "" {
		add(query.ObjectType, func(placeholder string) string { return "upper(oe.aggregate_type)=upper(" + placeholder + ")" })
	}
	if query.ObjectID != "" {
		add(query.ObjectID, func(placeholder string) string { return "oe.aggregate_id=" + placeholder })
	}
	if query.ActorID != "" {
		add(query.ActorID, func(placeholder string) string { return "meta.actor_ref=" + placeholder })
	}
	if query.ActorQuery != "" {
		add(query.ActorQuery, func(placeholder string) string {
			return "(meta.actor_ref=" + placeholder + " OR p.display_name ILIKE '%' || " + placeholder + " || '%')"
		})
	}
	if query.ActorKind != "" {
		add(query.ActorKind, func(placeholder string) string { return "(" + actorKindExpression + ")=" + placeholder })
	}
	if query.LegalEntityID != "" {
		add(query.LegalEntityID, func(placeholder string) string { return "meta.legal_entity_ref=" + placeholder })
	}
	args = append(args, query.Limit+1)
	limitPlaceholder := "$" + strconv.Itoa(len(args))

	sql := activitySourcesCTE + activityProjection +
		"\n\tWHERE " + strings.Join(clauses, " AND ") +
		"\n\tORDER BY oe.occurred_at DESC,oe.id DESC" +
		"\n\tLIMIT " + limitPlaceholder
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()

	items := make([]Event, 0, query.Limit+1)
	for rows.Next() {
		var value Event
		var principalKind string
		if err := rows.Scan(
			&value.ID, &value.OccurredAt, &value.ObjectType, &value.ObjectID, &value.EventType,
			&value.ActorID, &principalKind, &value.ActorDisplayName, &value.LegalEntityID, &value.RequestID, &value.CorrelationID, &value.Source,
		); err != nil {
			return Page{}, err
		}
		value.TenantID = query.TenantID
		value.ActorKind = principalKind
		value.Outcome = OutcomeSucceeded
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	page := Page{Items: items, AsOf: time.Now().UTC()}
	if len(items) > query.Limit {
		page.Items = items[:query.Limit]
		page.NextCursor = page.Items[len(page.Items)-1].ID
	}
	return page, nil
}

func (r *PostgresRepository) Get(ctx context.Context, tenantID, eventID string) (Event, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(eventID) == "" {
		return Event{}, ErrInvalid
	}
	sql := activitySourcesCTE + activityProjection + "\n\tWHERE oe.id=$2"
	row := r.pool.QueryRow(ctx, sql, tenantID, eventID)
	value, err := scanEvent(row, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	return value, err
}

func scanEvent(row pgx.Row, tenantID string) (Event, error) {
	var value Event
	var principalKind string
	if err := row.Scan(
		&value.ID, &value.OccurredAt, &value.ObjectType, &value.ObjectID, &value.EventType,
		&value.ActorID, &principalKind, &value.ActorDisplayName, &value.LegalEntityID, &value.RequestID, &value.CorrelationID, &value.Source,
	); err != nil {
		return Event{}, err
	}
	value.TenantID = tenantID
	value.ActorKind = principalKind
	value.Outcome = OutcomeSucceeded
	return value, nil
}
