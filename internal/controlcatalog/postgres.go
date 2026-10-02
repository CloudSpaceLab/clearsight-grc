//go:build postgres

package controlcatalog

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateWithImplementationLink(ctx context.Context, definition Definition, link ImplementationLink) (Definition, ImplementationLink, error) {
	if r == nil || r.pool == nil {
		return Definition{}, ImplementationLink{}, ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Definition{}, ImplementationLink{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := scanDefinition(tx.QueryRow(ctx, `
		INSERT INTO control_definitions(
			id,tenant_id,code,name,objective,description,category,status,version,created_at,updated_at)
		SELECT $2::uuid,t.id,$3::text,$4::text,$5::text,$6::text,$7::text,$8::text,$9::bigint,$10::timestamptz,$11::timestamptz
		FROM tenants t
		WHERE t.id::text=$1 OR t.slug=$1
		RETURNING id::text,tenant_id::text,code,name,objective,description,category,status,version,created_at,updated_at`,
		definition.TenantID, definition.ID, definition.Code, definition.Name, definition.Objective,
		definition.Description, definition.Category, definition.Status, definition.Version,
		definition.CreatedAt, definition.UpdatedAt,
	))
	if err != nil {
		return Definition{}, ImplementationLink{}, mapPostgresError(err)
	}
	link.TenantID = created.TenantID
	linked, err := insertImplementationLink(ctx, tx, link)
	if err != nil {
		return Definition{}, ImplementationLink{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Definition{}, ImplementationLink{}, err
	}
	return created, linked, nil
}

func (r *PostgresRepository) GetDefinition(ctx context.Context, tenant, id string) (Definition, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return Definition{}, ErrInvalid
	}
	value, err := scanDefinition(r.pool.QueryRow(ctx, `
		SELECT d.id::text,d.tenant_id::text,d.code,d.name,d.objective,d.description,d.category,d.status,d.version,d.created_at,d.updated_at
		FROM control_definitions d
		JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND d.id=$2::uuid`,
		strings.TrimSpace(tenant), strings.TrimSpace(id),
	))
	if err != nil {
		return Definition{}, mapPostgresError(err)
	}
	return value, nil
}

func (r *PostgresRepository) LinkImplementation(ctx context.Context, link ImplementationLink) (ImplementationLink, error) {
	if r == nil || r.pool == nil {
		return ImplementationLink{}, ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ImplementationLink{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	linked, err := insertImplementationLink(ctx, tx, link)
	if err != nil {
		return ImplementationLink{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ImplementationLink{}, err
	}
	return linked, nil
}

func (r *PostgresRepository) GetImplementationLink(ctx context.Context, tenant, entity, id string) (ImplementationLink, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(entity) == "" || strings.TrimSpace(id) == "" {
		return ImplementationLink{}, ErrInvalid
	}
	value, err := scanImplementationLink(r.pool.QueryRow(ctx, `
		SELECT l.id::text,l.tenant_id::text,l.legal_entity_id::text,l.definition_id::text,l.program_id::text,l.implementation_id::text,l.created_at
		FROM control_catalog_implementation_links l
		JOIN tenants t ON t.id=l.tenant_id
		JOIN legal_entities le ON le.tenant_id=l.tenant_id AND le.id=l.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND l.id=$3::uuid`,
		strings.TrimSpace(tenant), strings.TrimSpace(entity), strings.TrimSpace(id),
	))
	if err != nil {
		return ImplementationLink{}, mapPostgresError(err)
	}
	return value, nil
}

func (r *PostgresRepository) ListImplementationLinks(ctx context.Context, tenant, definitionID string, limit int) ([]ImplementationLink, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(definitionID) == "" {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `
		SELECT l.id::text,l.tenant_id::text,l.legal_entity_id::text,l.definition_id::text,l.program_id::text,l.implementation_id::text,l.created_at
		FROM control_catalog_implementation_links l
		JOIN tenants t ON t.id=l.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND l.definition_id=$2::uuid
		ORDER BY l.created_at,l.id
		LIMIT $3`,
		strings.TrimSpace(tenant), strings.TrimSpace(definitionID), limit,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	values := make([]ImplementationLink, 0)
	for rows.Next() {
		value, scanErr := scanImplementationLink(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (r *PostgresRepository) ListEntityImplementationLinks(ctx context.Context, tenant, entity, programID string, limit int) ([]ImplementationLink, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(entity) == "" {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `
		SELECT l.id::text,l.tenant_id::text,l.legal_entity_id::text,l.definition_id::text,l.program_id::text,l.implementation_id::text,l.created_at
		FROM control_catalog_implementation_links l
		JOIN tenants t ON t.id=l.tenant_id
		JOIN legal_entities le ON le.tenant_id=l.tenant_id AND le.id=l.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND ($3::text='' OR l.program_id=NULLIF($3::text,'')::uuid)
		ORDER BY l.created_at,l.id
		LIMIT $4`,
		strings.TrimSpace(tenant), strings.TrimSpace(entity), strings.TrimSpace(programID), limit,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	values := make([]ImplementationLink, 0)
	for rows.Next() {
		value, scanErr := scanImplementationLink(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

type rowScanner interface{ Scan(...any) error }

type txExecutor interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func insertImplementationLink(ctx context.Context, tx txExecutor, link ImplementationLink) (ImplementationLink, error) {
	value, err := scanImplementationLink(tx.QueryRow(ctx, `
		INSERT INTO control_catalog_implementation_links(
			id,tenant_id,legal_entity_id,definition_id,program_id,implementation_id,created_at)
		SELECT $3::uuid,t.id,le.id,$4::uuid,p.id,ci.id,$7::timestamptz
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		JOIN programs p ON p.tenant_id=t.id AND p.legal_entity_id=le.id
		JOIN control_implementations ci ON ci.tenant_id=t.id AND ci.program_id=p.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND p.id=$5::uuid
		  AND ci.id=$6::uuid
		  AND EXISTS (
		      SELECT 1 FROM control_definitions d
		      WHERE d.id=$4::uuid AND d.tenant_id=t.id AND d.status='ACTIVE')
		RETURNING id::text,tenant_id::text,legal_entity_id::text,definition_id::text,program_id::text,implementation_id::text,created_at`,
		link.TenantID, link.LegalEntityID, link.ID, link.DefinitionID, link.ProgramID, link.ImplementationID, link.CreatedAt,
	))
	if err != nil {
		return ImplementationLink{}, mapPostgresError(err)
	}
	return value, nil
}

func scanDefinition(row rowScanner) (Definition, error) {
	var value Definition
	err := row.Scan(&value.ID, &value.TenantID, &value.Code, &value.Name, &value.Objective, &value.Description,
		&value.Category, &value.Status, &value.Version, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func scanImplementationLink(row rowScanner) (ImplementationLink, error) {
	var value ImplementationLink
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.DefinitionID,
		&value.ProgramID, &value.ImplementationID, &value.CreatedAt)
	return value, err
}

func mapPostgresError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrDuplicate
		case "23503", "23514", "22P02", "22001":
			return ErrInvalid
		}
	}
	return err
}
