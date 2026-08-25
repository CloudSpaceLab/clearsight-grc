//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetDraft(ctx context.Context, tenant, requestID, sessionID string) (ResponseDraft, error) {
	value, err := scanResponseDraft(r.pool.QueryRow(ctx, `
		SELECT d.id::text,d.tenant_id::text,d.request_id::text,d.session_id::text,d.answers,d.presentation_mode,d.version,d.created_at,d.updated_at
		FROM capture_response_drafts d
		JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND d.request_id=$2::uuid AND d.session_id=$3::uuid`, tenant, requestID, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseDraft{}, ErrNotFound
	}
	if err != nil {
		return ResponseDraft{}, fmt.Errorf("get response draft: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) SaveDraft(ctx context.Context, record SaveDraftRecord) (ResponseDraft, error) {
	answers, err := json.Marshal(record.Answers)
	if err != nil {
		return ResponseDraft{}, err
	}
	if record.ExpectedVersion == 0 {
		value, insertErr := scanResponseDraft(r.pool.QueryRow(ctx, `
			INSERT INTO capture_response_drafts(id,tenant_id,request_id,session_id,answers,presentation_mode,version,created_at,updated_at)
			VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5::jsonb,$6,1,$7,$7)
			ON CONFLICT (tenant_id,request_id,session_id) DO NOTHING
			RETURNING id::text,tenant_id::text,request_id::text,session_id::text,answers,presentation_mode,version,created_at,updated_at`,
			record.ID, record.TenantID, record.RequestID, record.SessionID, string(answers), record.PresentationMode, record.UpdatedAt))
		if errors.Is(insertErr, pgx.ErrNoRows) {
			return ResponseDraft{}, ErrVersionConflict
		}
		if insertErr != nil {
			return ResponseDraft{}, fmt.Errorf("create response draft: %w", insertErr)
		}
		return value, nil
	}

	value, updateErr := scanResponseDraft(r.pool.QueryRow(ctx, `
		UPDATE capture_response_drafts d
		SET answers=$5::jsonb,presentation_mode=$6,version=d.version+1,updated_at=$7
		FROM tenants t
		WHERE t.id=d.tenant_id AND (t.id::text=$1 OR t.slug=$1)
		  AND d.request_id=$2::uuid AND d.session_id=$3::uuid AND d.version=$4
		RETURNING d.id::text,d.tenant_id::text,d.request_id::text,d.session_id::text,d.answers,d.presentation_mode,d.version,d.created_at,d.updated_at`,
		record.TenantID, record.RequestID, record.SessionID, record.ExpectedVersion, string(answers), record.PresentationMode, record.UpdatedAt))
	if updateErr == nil {
		return value, nil
	}
	if !errors.Is(updateErr, pgx.ErrNoRows) {
		return ResponseDraft{}, fmt.Errorf("update response draft: %w", updateErr)
	}
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM capture_response_drafts d JOIN tenants t ON t.id=d.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1) AND d.request_id=$2::uuid AND d.session_id=$3::uuid
		)`, record.TenantID, record.RequestID, record.SessionID).Scan(&exists); err != nil {
		return ResponseDraft{}, fmt.Errorf("resolve response draft conflict: %w", err)
	}
	if exists {
		return ResponseDraft{}, ErrVersionConflict
	}
	return ResponseDraft{}, ErrNotFound
}

func (r *PostgresRepository) DeleteDraft(ctx context.Context, tenant, requestID, sessionID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM capture_response_drafts d USING tenants t
		WHERE t.id=d.tenant_id AND (t.id::text=$1 OR t.slug=$1)
		  AND d.request_id=$2::uuid AND d.session_id=$3::uuid`, tenant, requestID, sessionID)
	if err != nil {
		return fmt.Errorf("delete response draft: %w", err)
	}
	return nil
}

func scanResponseDraft(row scanner) (ResponseDraft, error) {
	var value ResponseDraft
	var answers []byte
	if err := row.Scan(
		&value.ID, &value.TenantID, &value.RequestID, &value.SessionID, &answers,
		&value.PresentationMode, &value.Version, &value.CreatedAt, &value.UpdatedAt,
	); err != nil {
		return ResponseDraft{}, err
	}
	if err := json.Unmarshal(answers, &value.Answers); err != nil {
		return ResponseDraft{}, err
	}
	return value, nil
}

var _ DraftStore = (*PostgresRepository)(nil)
