//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresRepository) resolveOriginCreate(ctx context.Context, value Request, created Request, createErr error) (Request, error) {
	if createErr == nil {
		return created, nil
	}
	var databaseError *pgconn.PgError
	if value.Origin.empty() || !errors.As(createErr, &databaseError) || databaseError.Code != "23505" || databaseError.ConstraintName != "capture_requests_origin_idx" {
		return Request{}, createErr
	}
	existing, err := r.GetRequestByOrigin(ctx, value.TenantID, value.Origin)
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

var _ Repository = (*PostgresRepository)(nil)
