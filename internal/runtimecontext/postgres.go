//go:build postgres

package runtimecontext

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresResolver struct {
	pool *pgxpool.Pool
}

func NewPostgresResolver(pool *pgxpool.Pool) *PostgresResolver {
	return &PostgresResolver{pool: pool}
}

func (r *PostgresResolver) Resolve(ctx context.Context, scope Scope) (DisplayContext, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	if r == nil || r.pool == nil || scope.TenantID == "" || scope.LegalEntityID == "" || scope.PrincipalID == "" {
		return DisplayContext{}, ErrInvalid
	}

	var value DisplayContext
	err := r.pool.QueryRow(ctx, `
		SELECT t.name,le.name,p.display_name
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		JOIN principals p ON p.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND p.id::text=$3
		  AND p.status='ACTIVE'
		  AND le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		LIMIT 1`, scope.TenantID, scope.LegalEntityID, scope.PrincipalID).
		Scan(&value.TenantName, &value.LegalEntityName, &value.PrincipalName)
	if errors.Is(err, pgx.ErrNoRows) {
		return DisplayContext{}, ErrNotFound
	}
	if err != nil {
		return DisplayContext{}, err
	}
	return value, nil
}

var _ Resolver = (*PostgresResolver)(nil)
