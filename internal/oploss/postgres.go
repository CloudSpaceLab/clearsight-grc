//go:build postgres

package oploss

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, value Loss, event Event) (Loss, error) {
	if r == nil || r.pool == nil {
		return Loss{}, ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Loss{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := scanLoss(tx.QueryRow(ctx, `
		INSERT INTO operational_losses(
			id,tenant_id,legal_entity_id,organization_scope_id,code,title,event_type,cause,description,
			gross_amount_minor,currency,occurred_at,discovered_at,risk_id,matter_id,owner_principal_id,
			status,version,created_at,updated_at)
		SELECT $3::uuid,t.id,le.id,NULLIF($4::text,'')::uuid,$5::text,$6::text,$7::text,$8::text,$9::text,
		       $10::bigint,$11::text,$12::timestamptz,$13::timestamptz,NULLIF($14::text,'')::uuid,
		       NULLIF($15::text,'')::uuid,$16::uuid,$17::text,$18::bigint,$19::timestamptz,$20::timestamptz
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND le.valid_from<=$20::timestamptz
		  AND (le.valid_until IS NULL OR $20::timestamptz<le.valid_until)
		RETURNING id::text,tenant_id::text,legal_entity_id::text,COALESCE(organization_scope_id::text,''),
		          code,title,event_type,cause,description,gross_amount_minor,currency,occurred_at,discovered_at,
		          COALESCE(risk_id::text,''),COALESCE(matter_id::text,''),owner_principal_id::text,
		          status,version,created_at,updated_at`,
		value.TenantID, value.LegalEntityID, value.ID, value.OrganizationScopeID, value.Code, value.Title,
		value.EventType, value.Cause, value.Description, value.GrossAmountMinor, value.Currency,
		value.OccurredAt, value.DiscoveredAt, value.RiskID, value.MatterID, value.OwnerPrincipalID,
		value.Status, value.Version, value.CreatedAt, value.UpdatedAt,
	))
	if err != nil {
		return Loss{}, mapPostgresError(err)
	}
	event.TenantID, event.LegalEntityID, event.LossID = created.TenantID, created.LegalEntityID, created.ID
	if err := storeHistory(ctx, tx, created, event); err != nil {
		return Loss{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Loss{}, err
	}
	return created, nil
}

func (r *PostgresRepository) Update(ctx context.Context, scope Scope, next Loss, expected int64, event Event) (Loss, error) {
	tx, current, err := r.lockLoss(ctx, scope, next.ID)
	if err != nil {
		return Loss{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if current.Version != expected {
		return Loss{}, ErrVersionConflict
	}
	if next.ID != current.ID || next.TenantID != current.TenantID || next.LegalEntityID != current.LegalEntityID ||
		next.Code != current.Code || next.OwnerPrincipalID != current.OwnerPrincipalID ||
		next.Currency != current.Currency || next.Version != expected+1 || !next.CreatedAt.Equal(current.CreatedAt) {
		return Loss{}, ErrInvalid
	}
	recovered, err := signedRecoveredMinor(ctx, tx, current)
	if err != nil {
		return Loss{}, err
	}
	if recovered < 0 || recovered > next.GrossAmountMinor {
		return Loss{}, ErrRecoveryLimit
	}

	updated, err := scanLoss(tx.QueryRow(ctx, `
		UPDATE operational_losses
		SET organization_scope_id=NULLIF($5::text,'')::uuid,title=$6::text,event_type=$7::text,
		    cause=$8::text,description=$9::text,gross_amount_minor=$10::bigint,
		    occurred_at=$11::timestamptz,discovered_at=$12::timestamptz,
		    risk_id=NULLIF($13::text,'')::uuid,matter_id=NULLIF($14::text,'')::uuid,
		    status=$15::text,version=$16::bigint,updated_at=$17::timestamptz
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$4::bigint
		RETURNING id::text,tenant_id::text,legal_entity_id::text,COALESCE(organization_scope_id::text,''),
		          code,title,event_type,cause,description,gross_amount_minor,currency,occurred_at,discovered_at,
		          COALESCE(risk_id::text,''),COALESCE(matter_id::text,''),owner_principal_id::text,
		          status,version,created_at,updated_at`,
		current.TenantID, current.LegalEntityID, current.ID, expected,
		next.OrganizationScopeID, next.Title, next.EventType, next.Cause, next.Description, next.GrossAmountMinor,
		next.OccurredAt, next.DiscoveredAt, next.RiskID, next.MatterID, next.Status, next.Version, next.UpdatedAt,
	))
	if err != nil {
		return Loss{}, mapPostgresError(err)
	}
	event.TenantID, event.LegalEntityID, event.LossID = updated.TenantID, updated.LegalEntityID, updated.ID
	if err := storeHistory(ctx, tx, updated, event); err != nil {
		return Loss{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Loss{}, err
	}
	return updated, nil
}

func (r *PostgresRepository) AddRecovery(ctx context.Context, scope Scope, id string, expected int64, recovery Recovery, event Event) (Loss, Recovery, error) {
	tx, current, err := r.lockLoss(ctx, scope, id)
	if err != nil {
		return Loss{}, Recovery{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if current.Version != expected {
		return Loss{}, Recovery{}, ErrVersionConflict
	}
	if current.Status != StatusActive || recovery.LossID != current.ID || recovery.LossVersion != expected+1 ||
		recovery.Currency != current.Currency || recovery.AmountMinor <= 0 || !validRecoveryKind(recovery.Kind) ||
		event.LossID != current.ID || event.LossVersion != expected+1 || event.Type != EventLossRecoveryRecorded {
		return Loss{}, Recovery{}, ErrInvalid
	}
	recovered, err := signedRecoveredMinor(ctx, tx, current)
	if err != nil {
		return Loss{}, Recovery{}, err
	}
	nextRecovered := recovered
	if recovery.Kind == RecoveryCash {
		nextRecovered += recovery.AmountMinor
	} else {
		nextRecovered -= recovery.AmountMinor
	}
	if nextRecovered < 0 || nextRecovered > current.GrossAmountMinor {
		return Loss{}, Recovery{}, ErrRecoveryLimit
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO operational_loss_recoveries(
			id,tenant_id,legal_entity_id,loss_id,loss_version,kind,amount_minor,currency,reference,recovered_at,actor_id,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::bigint,$6::text,$7::bigint,$8::text,$9::text,
		       $10::timestamptz,NULLIF($11::text,'')::uuid,$12::timestamptz)`,
		recovery.ID, current.TenantID, current.LegalEntityID, current.ID, recovery.LossVersion,
		recovery.Kind, recovery.AmountMinor, recovery.Currency, recovery.Reference, recovery.RecoveredAt,
		recovery.ActorID, recovery.CreatedAt,
	); err != nil {
		return Loss{}, Recovery{}, mapPostgresError(err)
	}

	updated, err := scanLoss(tx.QueryRow(ctx, `
		UPDATE operational_losses
		SET version=$4::bigint,updated_at=$5::timestamptz
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$6::bigint
		RETURNING id::text,tenant_id::text,legal_entity_id::text,COALESCE(organization_scope_id::text,''),
		          code,title,event_type,cause,description,gross_amount_minor,currency,occurred_at,discovered_at,
		          COALESCE(risk_id::text,''),COALESCE(matter_id::text,''),owner_principal_id::text,
		          status,version,created_at,updated_at`,
		current.TenantID, current.LegalEntityID, current.ID, expected+1, event.OccurredAt, expected,
	))
	if err != nil {
		return Loss{}, Recovery{}, mapPostgresError(err)
	}
	event.TenantID, event.LegalEntityID, event.LossID = updated.TenantID, updated.LegalEntityID, updated.ID
	if err := storeHistory(ctx, tx, updated, event); err != nil {
		return Loss{}, Recovery{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Loss{}, Recovery{}, err
	}
	return updated, recovery, nil
}

func (r *PostgresRepository) Get(ctx context.Context, scope Scope, id string) (Aggregate, error) {
	if r == nil || r.pool == nil {
		return Aggregate{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	id = strings.TrimSpace(id)
	if err != nil || !uuidPattern.MatchString(id) {
		return Aggregate{}, ErrNotFound
	}
	loss, err := scanLoss(r.pool.QueryRow(ctx, lossReadSQL(false), scope.TenantID, scope.LegalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Aggregate{}, ErrNotFound
	}
	if err != nil {
		return Aggregate{}, err
	}
	recoveries, err := r.recoveries(ctx, loss)
	if err != nil {
		return Aggregate{}, err
	}
	calculated, err := totals(loss, recoveries)
	if err != nil {
		return Aggregate{}, err
	}
	return Aggregate{Loss: loss, Recoveries: recoveries, Totals: calculated}, nil
}

func (r *PostgresRepository) ResolveLegalEntity(ctx context.Context, tenant, id string) (string, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" || !uuidPattern.MatchString(strings.TrimSpace(id)) {
		return "", ErrNotFound
	}
	var entity string
	err := r.pool.QueryRow(ctx, `
		SELECT l.legal_entity_id::text
		FROM operational_losses l
		JOIN tenants t ON t.id=l.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND l.id=$2::uuid`,
		strings.TrimSpace(tenant), strings.TrimSpace(id)).Scan(&entity)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return entity, nil
}

func (r *PostgresRepository) lockLoss(ctx context.Context, scope Scope, id string) (pgx.Tx, Loss, error) {
	if r == nil || r.pool == nil {
		return nil, Loss{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	id = strings.TrimSpace(id)
	if err != nil || !uuidPattern.MatchString(id) {
		return nil, Loss{}, ErrNotFound
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, Loss{}, err
	}
	value, err := scanLoss(tx.QueryRow(ctx, lossReadSQL(true), scope.TenantID, scope.LegalEntityID, id))
	if err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, Loss{}, ErrNotFound
		}
		return nil, Loss{}, err
	}
	return tx, value, nil
}

func (r *PostgresRepository) recoveries(ctx context.Context, loss Loss) ([]Recovery, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text,loss_id::text,loss_version,kind,amount_minor,currency,reference,recovered_at,
		       COALESCE(actor_id::text,''),created_at
		FROM operational_loss_recoveries
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND loss_id=$3::uuid
		ORDER BY recovered_at DESC,id DESC`,
		loss.TenantID, loss.LegalEntityID, loss.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Recovery, 0)
	for rows.Next() {
		var value Recovery
		if err := rows.Scan(&value.ID, &value.LossID, &value.LossVersion, &value.Kind, &value.AmountMinor,
			&value.Currency, &value.Reference, &value.RecoveredAt, &value.ActorID, &value.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func signedRecoveredMinor(ctx context.Context, tx pgx.Tx, loss Loss) (int64, error) {
	var recovered int64
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(sum(CASE kind WHEN 'RECOVERY' THEN amount_minor ELSE -amount_minor END),0)::bigint
		FROM operational_loss_recoveries
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND loss_id=$3::uuid`,
		loss.TenantID, loss.LegalEntityID, loss.ID).Scan(&recovered)
	return recovered, err
}

func storeHistory(ctx context.Context, tx pgx.Tx, loss Loss, event Event) error {
	snapshot, err := json.Marshal(loss)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO operational_loss_revisions(tenant_id,legal_entity_id,loss_id,loss_version,snapshot,recorded_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::bigint,$5::jsonb,$6::timestamptz)`,
		loss.TenantID, loss.LegalEntityID, loss.ID, loss.Version, snapshot, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO operational_loss_events(id,tenant_id,legal_entity_id,loss_id,loss_version,event_type,actor_id,occurred_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::bigint,$6::text,NULLIF($7::text,'')::uuid,$8::timestamptz)`,
		event.ID, loss.TenantID, loss.LegalEntityID, loss.ID, loss.Version, event.Type, event.ActorID, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	payload, err := json.Marshal(map[string]any{
		"loss_version": loss.Version, "legal_entity_id": loss.LegalEntityID,
		"status": loss.Status, "currency": loss.Currency,
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES($1::uuid,'OPERATIONAL_LOSS',$2::uuid,$3::text,$4::jsonb,$5::timestamptz,$5::timestamptz,$5::timestamptz)`,
		loss.TenantID, loss.ID, event.Type, payload, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	return nil
}

func lossReadSQL(forUpdate bool) string {
	query := `
		SELECT l.id::text,l.tenant_id::text,l.legal_entity_id::text,COALESCE(l.organization_scope_id::text,''),
		       l.code,l.title,l.event_type,l.cause,l.description,l.gross_amount_minor,l.currency,
		       l.occurred_at,l.discovered_at,COALESCE(l.risk_id::text,''),COALESCE(l.matter_id::text,''),
		       l.owner_principal_id::text,l.status,l.version,l.created_at,l.updated_at
		FROM operational_losses l
		JOIN tenants t ON t.id=l.tenant_id
		JOIN legal_entities le ON le.tenant_id=l.tenant_id AND le.id=l.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND l.id=$3::uuid`
	if forUpdate {
		query += " FOR UPDATE OF l"
	}
	return query
}

type rowScanner interface{ Scan(...any) error }

func scanLoss(row rowScanner) (Loss, error) {
	var value Loss
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.OrganizationScopeID,
		&value.Code, &value.Title, &value.EventType, &value.Cause, &value.Description,
		&value.GrossAmountMinor, &value.Currency, &value.OccurredAt, &value.DiscoveredAt,
		&value.RiskID, &value.MatterID, &value.OwnerPrincipalID, &value.Status, &value.Version,
		&value.CreatedAt, &value.UpdatedAt,
	)
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
		case "23503", "23514", "22P02", "22001", "22003":
			return ErrInvalid
		}
	}
	return err
}
