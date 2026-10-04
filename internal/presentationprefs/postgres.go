//go:build postgres

package presentationprefs

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
		SELECT t.id::text,pref.principal_id::text,pref.home_focus,pref.portfolio_lens,pref.updated_at,pref.version
		FROM user_presentation_preferences pref
		JOIN tenants t ON t.id=pref.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND pref.principal_id=$2::uuid`,
		tenantID, principalID,
	).Scan(&value.TenantID, &value.PrincipalID, &value.HomeFocus, &value.PortfolioLens, &value.UpdatedAt, &value.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Stored{}, ErrNotFound
	}
	if err != nil {
		return Stored{}, fmt.Errorf("load presentation preferences: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) Upsert(ctx context.Context, value Stored, expected int64) (Stored, error) {
	if r == nil || r.pool == nil {
		return Stored{}, ErrInvalid
	}
	var result Stored
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_presentation_preferences(
			tenant_id,principal_id,home_focus,portfolio_lens,version
		)
		SELECT t.id,$2::uuid,$3,$4,1
		FROM tenants t
		WHERE (t.id::text=$1 OR t.slug=$1) AND $5=0
		ON CONFLICT(tenant_id,principal_id) DO UPDATE
		SET home_focus=EXCLUDED.home_focus,
		    portfolio_lens=EXCLUDED.portfolio_lens,
		    updated_at=clock_timestamp(),
		    version=user_presentation_preferences.version+1
		WHERE user_presentation_preferences.version=$5
		RETURNING tenant_id::text,principal_id::text,home_focus,portfolio_lens,updated_at,version`,
		value.TenantID, value.PrincipalID, value.HomeFocus, value.PortfolioLens, expected,
	).Scan(&result.TenantID, &result.PrincipalID, &result.HomeFocus, &result.PortfolioLens, &result.UpdatedAt, &result.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Stored{}, ErrVersionConflict
	}
	if err != nil {
		return Stored{}, fmt.Errorf("save presentation preferences: %w", err)
	}
	return result, nil
}

var _ Repository = (*PostgresRepository)(nil)
