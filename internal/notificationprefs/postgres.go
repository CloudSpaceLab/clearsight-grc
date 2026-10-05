//go:build postgres

package notificationprefs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Get(ctx context.Context, tenantID, principalID string) (Stored, error) {
	if r == nil || r.pool == nil {
		return Stored{}, ErrInvalid
	}
	var value Stored
	err := r.pool.QueryRow(ctx, `
		SELECT t.id::text,pref.principal_id::text,pref.daily_digest_enabled,pref.digest_minute,
		       pref.time_zone,pref.quiet_hours_enabled,pref.quiet_start_minute,pref.quiet_end_minute,
		       pref.updated_at,pref.version
		FROM user_notification_preferences pref
		JOIN tenants t ON t.id=pref.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND pref.principal_id=$2::uuid`,
		tenantID, principalID,
	).Scan(
		&value.TenantID, &value.PrincipalID, &value.DailyDigestEnabled, &value.DigestMinute,
		&value.TimeZone, &value.QuietHoursEnabled, &value.QuietStartMinute, &value.QuietEndMinute,
		&value.UpdatedAt, &value.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Stored{}, ErrNotFound
	}
	if err != nil {
		return Stored{}, fmt.Errorf("load notification preferences: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) Upsert(ctx context.Context, value Stored, expected int64) (Stored, error) {
	if r == nil || r.pool == nil || !validate(value) {
		return Stored{}, ErrInvalid
	}
	var result Stored
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_notification_preferences(
			tenant_id,principal_id,daily_digest_enabled,digest_minute,time_zone,
			quiet_hours_enabled,quiet_start_minute,quiet_end_minute,version
		)
		SELECT t.id,p.id,$3,$4,$5,$6,$7,$8,1
		FROM tenants t
		JOIN principals p ON p.tenant_id=t.id AND p.id=$2::uuid
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (
		    $9=0
		    OR EXISTS (
		      SELECT 1 FROM user_notification_preferences current
		      WHERE current.tenant_id=t.id AND current.principal_id=p.id AND current.version=$9
		    )
		  )
		ON CONFLICT(tenant_id,principal_id) DO UPDATE
		SET daily_digest_enabled=EXCLUDED.daily_digest_enabled,
		    digest_minute=EXCLUDED.digest_minute,
		    time_zone=EXCLUDED.time_zone,
		    quiet_hours_enabled=EXCLUDED.quiet_hours_enabled,
		    quiet_start_minute=EXCLUDED.quiet_start_minute,
		    quiet_end_minute=EXCLUDED.quiet_end_minute,
		    updated_at=clock_timestamp(),
		    version=user_notification_preferences.version+1
		WHERE user_notification_preferences.version=$9
		RETURNING tenant_id::text,principal_id::text,daily_digest_enabled,digest_minute,time_zone,
		          quiet_hours_enabled,quiet_start_minute,quiet_end_minute,updated_at,version`,
		value.TenantID, value.PrincipalID, value.DailyDigestEnabled, value.DigestMinute, value.TimeZone,
		value.QuietHoursEnabled, value.QuietStartMinute, value.QuietEndMinute, expected,
	).Scan(
		&result.TenantID, &result.PrincipalID, &result.DailyDigestEnabled, &result.DigestMinute,
		&result.TimeZone, &result.QuietHoursEnabled, &result.QuietStartMinute, &result.QuietEndMinute,
		&result.UpdatedAt, &result.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Stored{}, ErrVersionConflict
	}
	if err != nil {
		return Stored{}, fmt.Errorf("save notification preferences: %w", err)
	}
	return result, nil
}

var _ Repository = (*PostgresRepository)(nil)
