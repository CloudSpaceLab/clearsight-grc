//go:build postgres

package organization

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalid  = errors.New("organization scope is invalid")
	ErrNotFound = errors.New("organization scope was not found")
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ReconcileLegacyPaths(ctx context.Context) (int, error) {
	if r == nil || r.pool == nil {
		return 0, ErrInvalid
	}
	var inserted int
	if err := r.pool.QueryRow(ctx, `SELECT reconcile_organization_scopes()`).Scan(&inserted); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID, legalEntityID string, limit int) (ScopePage, error) {
	tenantID, legalEntityID = strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID)
	if r == nil || r.pool == nil || tenantID == "" || legalEntityID == "" {
		return ScopePage{}, ErrInvalid
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,COALESCE(parent_scope_id::text,''),
		       code,name,kind,department_path,origin,status,valid_from,valid_until,version
		FROM organization_scopes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		  AND status='ACTIVE' AND valid_from<=clock_timestamp()
		  AND (valid_until IS NULL OR clock_timestamp()<valid_until)
		ORDER BY cardinality(department_path),department_path,id
		LIMIT $3`, tenantID, legalEntityID, limit+1)
	if err != nil {
		return ScopePage{}, err
	}
	defer rows.Close()

	page := ScopePage{Items: make([]Scope, 0, limit)}
	for rows.Next() {
		var item Scope
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.LegalEntityID, &item.ParentScopeID,
			&item.Code, &item.Name, &item.Kind, &item.DepartmentPath, &item.Origin,
			&item.Status, &item.ValidFrom, &item.ValidUntil, &item.Version,
		); err != nil {
			return ScopePage{}, err
		}
		if len(page.Items) == limit {
			page.Truncated = true
			continue
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ScopePage{}, err
	}
	return page, nil
}

func (r *PostgresRepository) Get(ctx context.Context, tenantID, legalEntityID, scopeID string) (Scope, error) {
	tenantID, legalEntityID, scopeID = strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), strings.TrimSpace(scopeID)
	if r == nil || r.pool == nil || tenantID == "" || legalEntityID == "" || scopeID == "" {
		return Scope{}, ErrInvalid
	}
	var item Scope
	err := r.pool.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,COALESCE(parent_scope_id::text,''),
		       code,name,kind,department_path,origin,status,valid_from,valid_until,version
		FROM organization_scopes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		  AND status='ACTIVE' AND valid_from<=clock_timestamp()
		  AND (valid_until IS NULL OR clock_timestamp()<valid_until)`,
		tenantID, legalEntityID, scopeID,
	).Scan(
		&item.ID, &item.TenantID, &item.LegalEntityID, &item.ParentScopeID,
		&item.Code, &item.Name, &item.Kind, &item.DepartmentPath, &item.Origin,
		&item.Status, &item.ValidFrom, &item.ValidUntil, &item.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Scope{}, ErrNotFound
	}
	return item, err
}
