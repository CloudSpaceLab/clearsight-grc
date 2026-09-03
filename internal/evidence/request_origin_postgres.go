//go:build postgres

package evidence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const requestSelectColumns = `er.id::text,t.id::text,er.subject_type,er.subject_id,er.title,er.purpose,er.why_you,er.sensitivity,er.audience_type,
	er.estimated_minutes,er.deadline,er.known_facts,er.fields,er.source_bindings,COALESCE(er.form_template_id::text,''),COALESCE(er.form_template_version,0),
	er.collection_period_start,er.collection_period_end,er.status,COALESCE(er.created_by::text,''),er.version,er.created_at,er.updated_at,
	COALESCE(er.origin_type,''),COALESCE(er.origin_id::text,''),COALESCE(er.origin_sequence,0),COALESCE(er.predecessor_request_id::text,'')`

const requestReturningColumns = `id::text,(SELECT slug FROM tenants WHERE id=tenant_id),subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
	estimated_minutes,deadline,known_facts,fields,source_bindings,COALESCE(form_template_id::text,''),COALESCE(form_template_version,0),
	collection_period_start,collection_period_end,status,COALESCE(created_by::text,''),version,created_at,updated_at,
	COALESCE(origin_type,''),COALESCE(origin_id::text,''),COALESCE(origin_sequence,0),COALESCE(predecessor_request_id::text,'')`

func (r *PostgresRepository) GetRequestByOrigin(ctx context.Context, tenant string, origin RequestOrigin) (Request, error) {
	if err := validateRequestOriginIdentity(origin); err != nil {
		return Request{}, err
	}
	value, err := scanRequest(r.pool.QueryRow(ctx, `SELECT `+requestSelectColumns+`
		FROM capture_requests er JOIN tenants t ON t.id=er.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND er.origin_type=$2 AND er.origin_id=$3::uuid AND er.origin_sequence=$4`,
		tenant, origin.Type, origin.ID, origin.Sequence))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	return value, err
}

func (r *PostgresRepository) resolveOriginCreate(ctx context.Context, value Request, created Request, createErr error) (Request, error) {
	if createErr == nil {
		return created, nil
	}
	var databaseError *pgconn.PgError
	if value.Origin == nil || !errors.As(createErr, &databaseError) || databaseError.Code != "23505" || databaseError.ConstraintName != "capture_requests_origin_idx" {
		return Request{}, createErr
	}
	existing, err := r.GetRequestByOrigin(ctx, value.TenantID, *value.Origin)
	if err != nil {
		return Request{}, err
	}
	if value.Recipient.Type != "" {
		recipient, recipientErr := r.GetRequestRecipient(ctx, value.TenantID, existing.ID)
		if recipientErr != nil {
			return Request{}, recipientErr
		}
		existing.Recipient = recipient
	}
	if !sameImmutableRequest(existing, value) {
		return Request{}, ErrVersionConflict
	}
	return existing, nil
}

func originDatabaseValues(origin *RequestOrigin) (any, any, any) {
	if origin == nil {
		return nil, nil, nil
	}
	return origin.Type, origin.ID, origin.Sequence
}

func requestOriginFromDatabase(originType, originID string, sequence int64) (*RequestOrigin, error) {
	if originType == "" && originID == "" && sequence == 0 {
		return nil, nil
	}
	origin := &RequestOrigin{Type: RequestOriginType(originType), ID: originID, Sequence: sequence}
	if err := validateRequestOriginIdentity(*origin); err != nil {
		return nil, fmt.Errorf("stored request origin: %w", err)
	}
	return origin, nil
}

var _ Repository = (*PostgresRepository)(nil)
