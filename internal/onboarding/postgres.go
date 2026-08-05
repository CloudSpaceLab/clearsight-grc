//go:build postgres

package onboarding

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}
func (r *PostgresRepository) Get(ctx context.Context, tenant, principal, guide string) (State, error) {
	var value State
	err := r.pool.QueryRow(ctx, `SELECT t.slug,uos.principal_id::text,uos.guide_code,uos.guide_version,uos.current_step,uos.completed_at IS NOT NULL,uos.dismissed_at IS NOT NULL,uos.updated_at,uos.version FROM user_onboarding_state uos JOIN tenants t ON t.id=uos.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND uos.principal_id=$2::uuid AND uos.guide_code=$3`, tenant, principal, guide).Scan(&value.TenantID, &value.PrincipalID, &value.GuideCode, &value.GuideVersion, &value.CurrentStep, &value.Completed, &value.Dismissed, &value.UpdatedAt, &value.Version)
	if errors.Is(err, pgx.ErrNoRows) { return State{}, ErrStateNotFound }
	if err != nil { return State{}, fmt.Errorf("load onboarding state: %w", err) }
	return value, nil
}
func (r *PostgresRepository) Upsert(ctx context.Context, value State, expected int64) (State, error) {
	var result State
	err := r.pool.QueryRow(ctx, `INSERT INTO user_onboarding_state(tenant_id,principal_id,guide_code,guide_version,current_step,completed_at,dismissed_at,version) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,$5,CASE WHEN $6 THEN clock_timestamp() END,CASE WHEN $7 THEN clock_timestamp() END,1) ON CONFLICT(tenant_id,principal_id,guide_code) DO UPDATE SET guide_version=EXCLUDED.guide_version,current_step=EXCLUDED.current_step,completed_at=EXCLUDED.completed_at,dismissed_at=EXCLUDED.dismissed_at,updated_at=clock_timestamp(),version=user_onboarding_state.version+1 WHERE user_onboarding_state.version=$8 RETURNING (SELECT slug FROM tenants WHERE id=tenant_id),principal_id::text,guide_code,guide_version,current_step,completed_at IS NOT NULL,dismissed_at IS NOT NULL,updated_at,version`, value.TenantID, value.PrincipalID, value.GuideCode, value.GuideVersion, value.CurrentStep, value.Completed, value.Dismissed, expected).Scan(&result.TenantID, &result.PrincipalID, &result.GuideCode, &result.GuideVersion, &result.CurrentStep, &result.Completed, &result.Dismissed, &result.UpdatedAt, &result.Version)
	if errors.Is(err, pgx.ErrNoRows) { return State{}, ErrVersionConflict }
	if err != nil { return State{}, fmt.Errorf("save onboarding state: %w", err) }
	return result, nil
}
