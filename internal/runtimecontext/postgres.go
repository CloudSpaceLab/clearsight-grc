//go:build postgres

package runtimecontext

import (
	"context"
	"errors"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresResolver struct {
	pool *pgxpool.Pool
}

func NewPostgresResolver(pool *pgxpool.Pool) *PostgresResolver {
	return &PostgresResolver{pool: pool}
}

func (r *PostgresResolver) Resolve(ctx context.Context, actor identity.Actor) (Context, error) {
	if r == nil || r.pool == nil {
		return Context{}, ErrContextUnavailable
	}
	var tenantName, legalEntityName, principalName string
	err := r.pool.QueryRow(ctx, `
		SELECT t.name, le.name, p.display_name
		FROM tenants t
		JOIN legal_entities le
		  ON le.id = $2::uuid
		 AND le.tenant_id = t.id
		 AND le.valid_until IS NULL
		JOIN principals p
		  ON p.id = $3::uuid
		 AND p.tenant_id = t.id
		 AND p.valid_until IS NULL
		 AND p.status = 'ACTIVE'
		WHERE t.id = $1::uuid
		LIMIT 1`, actor.TenantID, actor.LegalEntityID, actor.PrincipalID).Scan(&tenantName, &legalEntityName, &principalName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Context{}, ErrContextUnavailable
	}
	if err != nil {
		return Context{}, fmt.Errorf("resolve runtime context: %w", err)
	}
	return contextFromActor(actor, tenantName, legalEntityName, principalName), nil
}
