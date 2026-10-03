//go:build postgres

package rcsa

import (
	"context"
	"encoding/json"
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

func (r *PostgresRepository) Create(ctx context.Context, cycle Cycle, risks []RiskSnapshot, controls []ControlSnapshot, event Event) (Aggregate, error) {
	if r == nil || r.pool == nil {
		return Aggregate{}, ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Aggregate{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := scanCycle(tx.QueryRow(ctx, `
		INSERT INTO rcsa_cycles(
			id,tenant_id,legal_entity_id,code,name,trigger_kind,first_line_owner_principal_id,status,
			population_checksum,version,created_at,updated_at)
		SELECT $3::uuid,t.id,le.id,$4::text,$5::text,$6::text,$7::uuid,$8::text,$9::text,$10::bigint,$11::timestamptz,$12::timestamptz
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		RETURNING id::text,tenant_id::text,legal_entity_id::text,code,name,trigger_kind,
		          first_line_owner_principal_id::text,status,population_checksum,
		          COALESCE(first_line_distribution_id::text,''),COALESCE(challenge_matter_id::text,''),
		          version,created_at,updated_at`,
		cycle.TenantID, cycle.LegalEntityID, cycle.ID, cycle.Code, cycle.Name, cycle.TriggerKind,
		cycle.FirstLineOwnerID, cycle.Status, cycle.PopulationChecksum, cycle.Version, cycle.CreatedAt, cycle.UpdatedAt,
	))
	if err != nil {
		return Aggregate{}, mapPostgresError(err)
	}

	for _, item := range risks {
		if _, err = tx.Exec(ctx, `
			INSERT INTO rcsa_cycle_risks(
				tenant_id,legal_entity_id,cycle_id,risk_id,risk_version,code,name,category)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::bigint,$6::text,$7::text,$8::text)`,
			created.TenantID, created.LegalEntityID, created.ID, item.RiskID, item.RiskVersion,
			item.Code, item.Name, item.Category); err != nil {
			return Aggregate{}, mapPostgresError(err)
		}
	}
	for _, item := range controls {
		if _, err = tx.Exec(ctx, `
			INSERT INTO rcsa_cycle_controls(
				tenant_id,legal_entity_id,cycle_id,risk_id,risk_version,risk_control_link_id,
				catalog_link_id,definition_id,definition_code,definition_name,program_id,
				implementation_id,implementation_version,implementation_name,implementation_status,
				implementation_effective_from,implementation_effective_until)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::bigint,$6::uuid,
			       $7::uuid,$8::uuid,$9::text,$10::text,$11::uuid,$12::uuid,$13::bigint,$14::text,$15::text,
			       $16::timestamptz,$17::timestamptz)`,
			created.TenantID, created.LegalEntityID, created.ID, item.RiskID, item.RiskVersion,
			item.RiskControlLinkID, item.CatalogLinkID, item.DefinitionID, item.DefinitionCode,
			item.DefinitionName, item.ProgramID, item.ImplementationID, item.ImplementationVersion, item.ImplementationName,
			item.ImplementationStatus, item.ImplementationEffectiveFrom, item.ImplementationEffectiveUntil); err != nil {
			return Aggregate{}, mapPostgresError(err)
		}
	}
	if err := storeHistory(ctx, tx, created, event); err != nil {
		return Aggregate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Aggregate{}, err
	}
	for i := range risks {
		risks[i].CycleID = created.ID
	}
	for i := range controls {
		controls[i].CycleID = created.ID
	}
	return Aggregate{Cycle: created, Risks: risks, Controls: controls}, nil
}

func (r *PostgresRepository) Get(ctx context.Context, scope Scope, id string) (Aggregate, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(id) == "" {
		return Aggregate{}, ErrInvalid
	}
	cycle, err := scanCycle(r.pool.QueryRow(ctx, `
		SELECT c.id::text,c.tenant_id::text,c.legal_entity_id::text,c.code,c.name,c.trigger_kind,
		       c.first_line_owner_principal_id::text,c.status,c.population_checksum,
		       COALESCE(c.first_line_distribution_id::text,''),COALESCE(c.challenge_matter_id::text,''),
		       c.version,c.created_at,c.updated_at
		FROM rcsa_cycles c
		JOIN tenants t ON t.id=c.tenant_id
		JOIN legal_entities le ON le.tenant_id=c.tenant_id AND le.id=c.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND c.id=$3::uuid`,
		scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id),
	))
	if err != nil {
		return Aggregate{}, mapPostgresError(err)
	}
	risks, err := r.risks(ctx, cycle)
	if err != nil {
		return Aggregate{}, err
	}
	controls, err := r.controls(ctx, cycle)
	if err != nil {
		return Aggregate{}, err
	}
	return Aggregate{Cycle: cycle, Risks: risks, Controls: controls}, nil
}

func (r *PostgresRepository) ResolveLegalEntity(ctx context.Context, tenant, id string) (string, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return "", ErrInvalid
	}
	var entity string
	err := r.pool.QueryRow(ctx, `
		SELECT c.legal_entity_id::text
		FROM rcsa_cycles c
		JOIN tenants t ON t.id=c.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND c.id=$2::uuid`,
		strings.TrimSpace(tenant), strings.TrimSpace(id)).Scan(&entity)
	if err != nil {
		return "", mapPostgresError(err)
	}
	return entity, nil
}

func (r *PostgresRepository) risks(ctx context.Context, cycle Cycle) ([]RiskSnapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cycle_id::text,risk_id::text,risk_version,code,name,category
		FROM rcsa_cycle_risks
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND cycle_id=$3::uuid
		ORDER BY code,risk_id`, cycle.TenantID, cycle.LegalEntityID, cycle.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]RiskSnapshot, 0)
	for rows.Next() {
		var value RiskSnapshot
		if err := rows.Scan(&value.CycleID, &value.RiskID, &value.RiskVersion, &value.Code, &value.Name, &value.Category); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) controls(ctx context.Context, cycle Cycle) ([]ControlSnapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cycle_id::text,risk_id::text,risk_version,risk_control_link_id::text,catalog_link_id::text,
		       definition_id::text,definition_code,definition_name,program_id::text,implementation_id::text,
		       implementation_version,implementation_name,implementation_status,implementation_effective_from,implementation_effective_until
		FROM rcsa_cycle_controls
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND cycle_id=$3::uuid
		ORDER BY definition_code,risk_control_link_id`,
		cycle.TenantID, cycle.LegalEntityID, cycle.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]ControlSnapshot, 0)
	for rows.Next() {
		var value ControlSnapshot
		if err := rows.Scan(&value.CycleID, &value.RiskID, &value.RiskVersion, &value.RiskControlLinkID,
			&value.CatalogLinkID, &value.DefinitionID, &value.DefinitionCode, &value.DefinitionName,
			&value.ProgramID, &value.ImplementationID, &value.ImplementationVersion, &value.ImplementationName,
			&value.ImplementationStatus, &value.ImplementationEffectiveFrom, &value.ImplementationEffectiveUntil); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanCycle(row scanner) (Cycle, error) {
	var value Cycle
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.Code, &value.Name, &value.TriggerKind,
		&value.FirstLineOwnerID, &value.Status, &value.PopulationChecksum, &value.FirstLineDistributionID,
		&value.ChallengeMatterID, &value.Version, &value.CreatedAt, &value.UpdatedAt,
	)
	return value, err
}

func storeHistory(ctx context.Context, tx pgx.Tx, cycle Cycle, event Event) error {
	snapshot, err := json.Marshal(cycle)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO rcsa_cycle_revisions(tenant_id,legal_entity_id,cycle_id,cycle_version,snapshot,recorded_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::bigint,$5::jsonb,$6::timestamptz)`,
		cycle.TenantID, cycle.LegalEntityID, cycle.ID, cycle.Version, snapshot, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO rcsa_cycle_events(id,tenant_id,legal_entity_id,cycle_id,cycle_version,event_type,actor_id,occurred_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::bigint,$6::text,NULLIF($7::text,'')::uuid,$8::timestamptz)`,
		event.ID, cycle.TenantID, cycle.LegalEntityID, cycle.ID, cycle.Version,
		event.Type, event.ActorID, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	payload, err := json.Marshal(map[string]any{
		"cycle_version": cycle.Version, "legal_entity_id": cycle.LegalEntityID,
		"population_checksum": cycle.PopulationChecksum,
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES($1::uuid,'RCSA_CYCLE',$2::uuid,$3::text,$4::jsonb,$5::timestamptz,$5::timestamptz,$5::timestamptz)`,
		cycle.TenantID, cycle.ID, event.Type, payload, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	return nil
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
