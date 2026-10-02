//go:build postgres

package risk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, value Risk, event Event) (Risk, error) {
	if r == nil || r.pool == nil {
		return Risk{}, ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Risk{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		INSERT INTO risks(
			id,tenant_id,legal_entity_id,code,name,category,statement,cause,event,impact,scope,
			owner_principal_id,status,version,created_at,updated_at)
		SELECT $3::uuid,t.id,le.id,$4,$5,$6,$7,$8,$9,$10,$11,
		       NULLIF($12,'')::uuid,$13,$14,$15,$16
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND le.valid_from<=$16
		  AND (le.valid_until IS NULL OR $16<le.valid_until)
		RETURNING id::text,tenant_id::text,legal_entity_id::text,code,name,category,statement,cause,event,impact,scope,
		          COALESCE(owner_principal_id::text,''),status,version,created_at,updated_at`,
		value.TenantID, value.LegalEntityID, value.ID, value.Code, value.Name, value.Category,
		value.Statement, value.Cause, value.Event, value.Impact, value.Scope, value.OwnerPrincipalID,
		value.Status, value.Version, value.CreatedAt, value.UpdatedAt,
	)
	created, err := scanRisk(row)
	if err != nil {
		return Risk{}, mapRiskPostgresError(err)
	}
	event.TenantID, event.LegalEntityID, event.RiskID = created.TenantID, created.LegalEntityID, created.ID
	if err := storeRiskHistory(ctx, tx, created, event); err != nil {
		return Risk{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Risk{}, err
	}
	return created, nil
}

func (r *PostgresRepository) Get(ctx context.Context, scope Scope, riskID string) (Risk, error) {
	if r == nil || r.pool == nil {
		return Risk{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	riskID = strings.TrimSpace(riskID)
	if err != nil || !validUUID(riskID) {
		return Risk{}, ErrNotFound
	}
	row := r.pool.QueryRow(ctx, riskReadSQL(false), scope.TenantID, scope.LegalEntityID, riskID)
	value, err := scanRisk(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Risk{}, ErrNotFound
	}
	if err != nil {
		return Risk{}, err
	}
	return value, nil
}

func (r *PostgresRepository) Update(ctx context.Context, scope Scope, next Risk, expectedVersion int64, event Event) (Risk, error) {
	if r == nil || r.pool == nil {
		return Risk{}, ErrInvalid
	}
	tx, current, err := r.lockRisk(ctx, scope, next.ID)
	if err != nil {
		return Risk{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if current.Version != expectedVersion {
		return Risk{}, ErrVersionConflict
	}
	if next.ID != current.ID || next.Code != current.Code || next.Version != expectedVersion+1 {
		return Risk{}, ErrInvalid
	}
	row := tx.QueryRow(ctx, `
		UPDATE risks
		SET name=$5,category=$6,statement=$7,cause=$8,event=$9,impact=$10,scope=$11,
		    owner_principal_id=NULLIF($12,'')::uuid,status=$13,version=$14,updated_at=$15
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$4
		RETURNING id::text,tenant_id::text,legal_entity_id::text,code,name,category,statement,cause,event,impact,scope,
		          COALESCE(owner_principal_id::text,''),status,version,created_at,updated_at`,
		current.TenantID, current.LegalEntityID, current.ID, expectedVersion,
		next.Name, next.Category, next.Statement, next.Cause, next.Event, next.Impact, next.Scope,
		next.OwnerPrincipalID, next.Status, next.Version, next.UpdatedAt,
	)
	updated, err := scanRisk(row)
	if err != nil {
		return Risk{}, mapRiskPostgresError(err)
	}
	event.TenantID, event.LegalEntityID, event.RiskID = updated.TenantID, updated.LegalEntityID, updated.ID
	if err := storeRiskHistory(ctx, tx, updated, event); err != nil {
		return Risk{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Risk{}, err
	}
	return updated, nil
}

func (r *PostgresRepository) AddAssessment(ctx context.Context, scope Scope, riskID string, expectedVersion int64, assessment Assessment, event Event) (Risk, Assessment, error) {
	tx, current, err := r.lockRisk(ctx, scope, riskID)
	if err != nil {
		return Risk{}, Assessment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if current.Version != expectedVersion {
		return Risk{}, Assessment{}, ErrVersionConflict
	}
	if assessment.RiskID != current.ID || assessment.RiskVersion != expectedVersion+1 || event.RiskVersion != expectedVersion+1 {
		return Risk{}, Assessment{}, ErrInvalid
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO risk_assessments(
			id,tenant_id,legal_entity_id,risk_id,risk_version,assessment_kind,method_code,method_version,
			dimensions,assumptions,evidence_references,confidence,assessed_by,appetite_statement_id,
			appetite_position,appetite_rationale,assessed_at,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$12,
		       NULLIF($13,'')::uuid,NULLIF($14,'')::uuid,$15,$16,$17,$18)`,
		assessment.ID, current.TenantID, current.LegalEntityID, current.ID, assessment.RiskVersion,
		assessment.Kind, assessment.MethodCode, assessment.MethodVersion, assessment.Dimensions, assessment.Assumptions,
		assessment.EvidenceReferences, assessment.Confidence, assessment.AssessedBy, assessment.AppetiteStatementID,
		assessment.AppetitePosition, assessment.AppetiteRationale, assessment.AssessedAt, assessment.CreatedAt,
	)
	if err != nil {
		return Risk{}, Assessment{}, mapRiskPostgresError(err)
	}
	updated, err := bumpRiskVersion(ctx, tx, current, expectedVersion, event.OccurredAt)
	if err != nil {
		return Risk{}, Assessment{}, err
	}
	event.TenantID, event.LegalEntityID, event.RiskID = updated.TenantID, updated.LegalEntityID, updated.ID
	if err := storeRiskHistory(ctx, tx, updated, event); err != nil {
		return Risk{}, Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Risk{}, Assessment{}, err
	}
	return updated, assessment, nil
}

func (r *PostgresRepository) AddAppetite(ctx context.Context, scope Scope, riskID string, expectedVersion int64, statement AppetiteStatement, event Event) (Risk, AppetiteStatement, error) {
	tx, current, err := r.lockRisk(ctx, scope, riskID)
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if current.Version != expectedVersion {
		return Risk{}, AppetiteStatement{}, ErrVersionConflict
	}
	if statement.RiskID != current.ID || statement.RiskVersion != expectedVersion+1 || event.RiskVersion != expectedVersion+1 {
		return Risk{}, AppetiteStatement{}, ErrInvalid
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO risk_appetite_statements(
			id,tenant_id,legal_entity_id,risk_id,risk_version,version,statement,rule,rationale,
			owner_principal_id,authority_principal_id,status,effective_from,effective_until,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,
		       NULLIF($10,'')::uuid,NULLIF($11,'')::uuid,$12,$13,$14,$15)`,
		statement.ID, current.TenantID, current.LegalEntityID, current.ID, statement.RiskVersion,
		statement.Version, statement.Statement, statement.Rule, statement.Rationale, statement.OwnerPrincipalID,
		statement.AuthorityPrincipalID, statement.Status, statement.EffectiveFrom, statement.EffectiveUntil, statement.CreatedAt,
	)
	if err != nil {
		return Risk{}, AppetiteStatement{}, mapRiskPostgresError(err)
	}
	updated, err := bumpRiskVersion(ctx, tx, current, expectedVersion, event.OccurredAt)
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	event.TenantID, event.LegalEntityID, event.RiskID = updated.TenantID, updated.LegalEntityID, updated.ID
	if err := storeRiskHistory(ctx, tx, updated, event); err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	return updated, statement, nil
}

func (r *PostgresRepository) Assessments(ctx context.Context, scope Scope, riskID string, limit int) ([]Assessment, error) {
	scope, err := normalizeScope(scope)
	if err != nil || !validUUID(riskID) {
		return nil, ErrNotFound
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT a.id::text,a.risk_id::text,a.risk_version,a.assessment_kind,a.method_code,a.method_version,
		       a.dimensions,a.assumptions,a.evidence_references,a.confidence,COALESCE(a.assessed_by::text,''),
		       COALESCE(a.appetite_statement_id::text,''),a.appetite_position,a.appetite_rationale,a.assessed_at,a.created_at
		FROM risk_assessments a
		JOIN tenants t ON t.id=a.tenant_id
		JOIN legal_entities le ON le.tenant_id=a.tenant_id AND le.id=a.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND a.risk_id=$3::uuid
		ORDER BY a.risk_version DESC,a.id DESC LIMIT $4`, scope.TenantID, scope.LegalEntityID, riskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Assessment, 0, limit)
	for rows.Next() {
		var value Assessment
		if err := rows.Scan(&value.ID,&value.RiskID,&value.RiskVersion,&value.Kind,&value.MethodCode,&value.MethodVersion,
			&value.Dimensions,&value.Assumptions,&value.EvidenceReferences,&value.Confidence,&value.AssessedBy,
			&value.AppetiteStatementID,&value.AppetitePosition,&value.AppetiteRationale,&value.AssessedAt,&value.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(values)==0 {
		if _, err := r.Get(ctx, scope, riskID); err != nil { return nil, err }
	}
	return values, nil
}

func (r *PostgresRepository) AppetiteStatements(ctx context.Context, scope Scope, riskID string, limit int) ([]AppetiteStatement, error) {
	scope, err := normalizeScope(scope)
	if err != nil || !validUUID(riskID) {
		return nil, ErrNotFound
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT a.id::text,a.risk_id::text,a.risk_version,a.version,a.statement,a.rule,a.rationale,
		       COALESCE(a.owner_principal_id::text,''),COALESCE(a.authority_principal_id::text,''),
		       a.status,a.effective_from,a.effective_until,a.created_at
		FROM risk_appetite_statements a
		JOIN tenants t ON t.id=a.tenant_id
		JOIN legal_entities le ON le.tenant_id=a.tenant_id AND le.id=a.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND a.risk_id=$3::uuid
		ORDER BY a.version DESC,a.id DESC LIMIT $4`, scope.TenantID, scope.LegalEntityID, riskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]AppetiteStatement, 0, limit)
	for rows.Next() {
		var value AppetiteStatement
		if err := rows.Scan(&value.ID,&value.RiskID,&value.RiskVersion,&value.Version,&value.Statement,&value.Rule,
			&value.Rationale,&value.OwnerPrincipalID,&value.AuthorityPrincipalID,&value.Status,
			&value.EffectiveFrom,&value.EffectiveUntil,&value.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(values)==0 {
		if _, err := r.Get(ctx, scope, riskID); err != nil { return nil, err }
	}
	return values, nil
}

func (r *PostgresRepository) CurrentAppetite(ctx context.Context, scope Scope, riskID string, at time.Time) (*AppetiteStatement, error) {
	scope, err := normalizeScope(scope)
	if err != nil || !validUUID(riskID) || at.IsZero() {
		return nil, ErrInvalid
	}
	var value AppetiteStatement
	err = r.pool.QueryRow(ctx, `
		SELECT a.id::text,a.risk_id::text,a.risk_version,a.version,a.statement,a.rule,a.rationale,
		       COALESCE(a.owner_principal_id::text,''),COALESCE(a.authority_principal_id::text,''),
		       a.status,a.effective_from,a.effective_until,a.created_at
		FROM risk_appetite_statements a
		JOIN tenants t ON t.id=a.tenant_id
		JOIN legal_entities le ON le.tenant_id=a.tenant_id AND le.id=a.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND a.risk_id=$3::uuid
		  AND a.status='ACTIVE'
		  AND a.effective_from<=$4
		  AND (a.effective_until IS NULL OR $4<a.effective_until)
		ORDER BY a.version DESC,a.id DESC
		LIMIT 1`, scope.TenantID, scope.LegalEntityID, riskID, at.UTC(),
	).Scan(&value.ID,&value.RiskID,&value.RiskVersion,&value.Version,&value.Statement,&value.Rule,
		&value.Rationale,&value.OwnerPrincipalID,&value.AuthorityPrincipalID,&value.Status,
		&value.EffectiveFrom,&value.EffectiveUntil,&value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.Get(ctx, scope, riskID); getErr != nil {
			return nil, getErr
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *PostgresRepository) lockRisk(ctx context.Context, scope Scope, riskID string) (pgx.Tx, Risk, error) {
	if r == nil || r.pool == nil {
		return nil, Risk{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	riskID = strings.TrimSpace(riskID)
	if err != nil || !validUUID(riskID) {
		return nil, Risk{}, ErrNotFound
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, Risk{}, err
	}
	row := tx.QueryRow(ctx, riskReadSQL(true), scope.TenantID, scope.LegalEntityID, riskID)
	current, err := scanRisk(row)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = tx.Rollback(ctx)
		return nil, Risk{}, ErrNotFound
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, Risk{}, err
	}
	return tx, current, nil
}

func riskReadSQL(lock bool) string {
	sql := `
		SELECT r.id::text,r.tenant_id::text,r.legal_entity_id::text,r.code,r.name,r.category,r.statement,r.cause,
		       r.event,r.impact,r.scope,COALESCE(r.owner_principal_id::text,''),r.status,r.version,r.created_at,r.updated_at
		FROM risks r
		JOIN tenants t ON t.id=r.tenant_id
		JOIN legal_entities le ON le.tenant_id=r.tenant_id AND le.id=r.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND r.id=$3::uuid`
	if lock {
		sql += " FOR UPDATE OF r"
	}
	return sql
}

type riskScanner interface{ Scan(...any) error }

func scanRisk(row riskScanner) (Risk, error) {
	var value Risk
	err := row.Scan(&value.ID,&value.TenantID,&value.LegalEntityID,&value.Code,&value.Name,&value.Category,&value.Statement,
		&value.Cause,&value.Event,&value.Impact,&value.Scope,&value.OwnerPrincipalID,&value.Status,&value.Version,
		&value.CreatedAt,&value.UpdatedAt)
	return value, err
}

func bumpRiskVersion(ctx context.Context, tx pgx.Tx, current Risk, expected int64, occurredAt time.Time) (Risk, error) {
	row := tx.QueryRow(ctx, `
		UPDATE risks SET version=version+1,updated_at=$5
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$4
		RETURNING id::text,tenant_id::text,legal_entity_id::text,code,name,category,statement,cause,event,impact,scope,
		          COALESCE(owner_principal_id::text,''),status,version,created_at,updated_at`,
		current.TenantID,current.LegalEntityID,current.ID,expected,occurredAt)
	value, err := scanRisk(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Risk{}, ErrVersionConflict
	}
	return value, err
}

func storeRiskHistory(ctx context.Context, tx pgx.Tx, value Risk, event Event) error {
	snapshot, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if event.RiskVersion != value.Version || event.RiskID != value.ID {
		return ErrInvalid
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO risk_revisions(tenant_id,legal_entity_id,risk_id,risk_version,snapshot,recorded_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6)`,
		value.TenantID, value.LegalEntityID, value.ID, value.Version, snapshot, event.OccurredAt); err != nil {
		return mapRiskPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO risk_events(id,tenant_id,legal_entity_id,risk_id,risk_version,event_type,payload,actor_id,occurred_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,'')::uuid,$9)`,
		event.ID, value.TenantID, value.LegalEntityID, value.ID, value.Version, event.Type, event.Payload, event.ActorID, event.OccurredAt); err != nil {
		return mapRiskPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'RISK',$2::uuid,$3,jsonb_build_object('risk_version',$4,'legal_entity_id',$5::text),$6,$6)`,
		value.TenantID, value.ID, event.Type, value.Version, value.LegalEntityID, event.OccurredAt); err != nil {
		return mapRiskPostgresError(err)
	}
	return nil
}

func mapRiskPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrDuplicate
		case "23503", "22P02", "23514":
			return ErrInvalid
		}
	}
	return fmt.Errorf("risk persistence: %w", err)
}

var _ Repository = (*PostgresRepository)(nil)
