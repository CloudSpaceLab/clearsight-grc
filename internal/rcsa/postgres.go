//go:build postgres

package rcsa

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

func (r *PostgresRepository) CreateCycle(ctx context.Context, cycle Cycle, items []Item, controls []ItemControl) (Aggregate, error) {
	if r == nil || r.pool == nil {
		return Aggregate{}, ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Aggregate{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tenantID, legalEntityID string
	err = tx.QueryRow(ctx, `
		SELECT t.id::text,le.id::text
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND le.valid_from<=$3::timestamptz
		  AND (le.valid_until IS NULL OR $3::timestamptz<le.valid_until)
	`, cycle.TenantID, cycle.LegalEntityID, cycle.CreatedAt).Scan(&tenantID, &legalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Aggregate{}, ErrInvalid
	}
	if err != nil {
		return Aggregate{}, err
	}
	cycle.TenantID, cycle.LegalEntityID = tenantID, legalEntityID
	_, err = tx.Exec(ctx, `
		INSERT INTO rcsa_cycles(
			id,tenant_id,legal_entity_id,code,name,form_template_id,form_template_version,
			period_start,period_end,due_at,challenge_due_at,created_by,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text,$6::uuid,$7::bigint,
		       $8::timestamptz,$9::timestamptz,$10::timestamptz,$11::timestamptz,$12::uuid,$13::timestamptz)
	`, cycle.ID, cycle.TenantID, cycle.LegalEntityID, cycle.Code, cycle.Name, cycle.FormTemplateID, cycle.FormTemplateVersion,
		cycle.PeriodStart, cycle.PeriodEnd, cycle.DueAt, cycle.ChallengeDueAt, cycle.CreatedBy, cycle.CreatedAt)
	if err != nil {
		return Aggregate{}, mapPostgresError(err)
	}

	storedItems := make([]Item, 0, len(items))
	itemByID := make(map[string]Item, len(items))
	for _, item := range items {
		item.TenantID, item.LegalEntityID, item.CycleID = cycle.TenantID, cycle.LegalEntityID, cycle.ID
		_, err = tx.Exec(ctx, `
			INSERT INTO rcsa_cycle_items(
				id,tenant_id,legal_entity_id,cycle_id,risk_id,risk_version,risk_code,risk_name,
				respondent_principal_id,created_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::bigint,$7::text,$8::text,$9::uuid,$10::timestamptz)
		`, item.ID, item.TenantID, item.LegalEntityID, item.CycleID, item.RiskID, item.RiskVersion,
			item.RiskCode, item.RiskName, item.RespondentPrincipalID, item.CreatedAt)
		if err != nil {
			return Aggregate{}, mapPostgresError(err)
		}
		storedItems = append(storedItems, item)
		itemByID[item.ID] = item
	}

	storedControls := make([]ItemControl, 0, len(controls))
	for _, control := range controls {
		item, ok := itemByID[control.ItemID]
		if !ok || item.RiskID != control.RiskID {
			return Aggregate{}, ErrInvalid
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO rcsa_cycle_item_controls(
				tenant_id,legal_entity_id,item_id,risk_id,control_link_id)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid)
		`, cycle.TenantID, cycle.LegalEntityID, control.ItemID, control.RiskID, control.ControlLinkID)
		if err != nil {
			return Aggregate{}, mapPostgresError(err)
		}
		storedControls = append(storedControls, control)
	}
	if err := tx.Commit(ctx); err != nil {
		return Aggregate{}, err
	}
	return aggregateFromRows(cycle, storedItems, storedControls, nil), nil
}

func (r *PostgresRepository) GetCycle(ctx context.Context, scope Scope, cycleID string) (Aggregate, error) {
	scope, err := normalizeScope(scope)
	cycleID = strings.TrimSpace(cycleID)
	if err != nil || cycleID == "" {
		return Aggregate{}, ErrInvalid
	}
	var cycle Cycle
	err = r.pool.QueryRow(ctx, `
		SELECT c.id::text,c.tenant_id::text,c.legal_entity_id::text,c.code,c.name,
		       c.form_template_id::text,c.form_template_version,c.period_start,c.period_end,c.due_at,
		       c.challenge_due_at,c.created_by::text,c.created_at
		FROM rcsa_cycles c
		JOIN tenants t ON t.id=c.tenant_id
		JOIN legal_entities le ON le.tenant_id=c.tenant_id AND le.id=c.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND c.id=$3::uuid
	`, scope.TenantID, scope.LegalEntityID, cycleID).Scan(
		&cycle.ID, &cycle.TenantID, &cycle.LegalEntityID, &cycle.Code, &cycle.Name,
		&cycle.FormTemplateID, &cycle.FormTemplateVersion, &cycle.PeriodStart, &cycle.PeriodEnd,
		&cycle.DueAt, &cycle.ChallengeDueAt, &cycle.CreatedBy, &cycle.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Aggregate{}, ErrNotFound
	}
	if err != nil {
		return Aggregate{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT i.id::text,i.cycle_id::text,i.tenant_id::text,i.legal_entity_id::text,
		       i.risk_id::text,i.risk_version,i.risk_code,i.risk_name,i.respondent_principal_id::text,i.created_at
		FROM rcsa_cycle_items i
		WHERE i.tenant_id=$1::uuid AND i.legal_entity_id=$2::uuid AND i.cycle_id=$3::uuid
		ORDER BY i.risk_code,i.id
	`, cycle.TenantID, cycle.LegalEntityID, cycle.ID)
	if err != nil {
		return Aggregate{}, err
	}
	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.CycleID, &item.TenantID, &item.LegalEntityID,
			&item.RiskID, &item.RiskVersion, &item.RiskCode, &item.RiskName,
			&item.RespondentPrincipalID, &item.CreatedAt); err != nil {
			rows.Close()
			return Aggregate{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Aggregate{}, err
	}
	rows.Close()

	controlRows, err := r.pool.Query(ctx, `
		SELECT x.item_id::text,x.risk_id::text,x.control_link_id::text
		FROM rcsa_cycle_item_controls x
		WHERE x.tenant_id=$1::uuid AND x.legal_entity_id=$2::uuid
		  AND EXISTS (SELECT 1 FROM rcsa_cycle_items i WHERE i.id=x.item_id AND i.cycle_id=$3::uuid)
		ORDER BY x.item_id,x.control_link_id
	`, cycle.TenantID, cycle.LegalEntityID, cycle.ID)
	if err != nil {
		return Aggregate{}, err
	}
	controls := make([]ItemControl, 0)
	for controlRows.Next() {
		var value ItemControl
		if err := controlRows.Scan(&value.ItemID, &value.RiskID, &value.ControlLinkID); err != nil {
			controlRows.Close()
			return Aggregate{}, err
		}
		controls = append(controls, value)
	}
	if err := controlRows.Err(); err != nil {
		controlRows.Close()
		return Aggregate{}, err
	}
	controlRows.Close()

	distributionRows, err := r.pool.Query(ctx, `
		SELECT d.item_id::text,d.distribution_id::text,d.issued_at
		FROM rcsa_cycle_item_distributions d
		WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid
		  AND EXISTS (SELECT 1 FROM rcsa_cycle_items i WHERE i.id=d.item_id AND i.cycle_id=$3::uuid)
		ORDER BY d.item_id
	`, cycle.TenantID, cycle.LegalEntityID, cycle.ID)
	if err != nil {
		return Aggregate{}, err
	}
	distributions := make([]DistributionLink, 0)
	for distributionRows.Next() {
		var value DistributionLink
		if err := distributionRows.Scan(&value.ItemID, &value.DistributionID, &value.IssuedAt); err != nil {
			distributionRows.Close()
			return Aggregate{}, err
		}
		distributions = append(distributions, value)
	}
	if err := distributionRows.Err(); err != nil {
		distributionRows.Close()
		return Aggregate{}, err
	}
	distributionRows.Close()
	return aggregateFromRows(cycle, items, controls, distributions), nil
}

func (r *PostgresRepository) AttachDistribution(ctx context.Context, scope Scope, link DistributionLink) (DistributionLink, error) {
	scope, err := normalizeScope(scope)
	if err != nil || strings.TrimSpace(link.ItemID) == "" || strings.TrimSpace(link.DistributionID) == "" || link.IssuedAt.IsZero() {
		return DistributionLink{}, ErrInvalid
	}
	var stored DistributionLink
	err = r.pool.QueryRow(ctx, `
		INSERT INTO rcsa_cycle_item_distributions(tenant_id,legal_entity_id,item_id,distribution_id,issued_at)
		SELECT i.tenant_id,i.legal_entity_id,i.id,$4::uuid,$5::timestamptz
		FROM rcsa_cycle_items i
		JOIN tenants t ON t.id=i.tenant_id
		JOIN legal_entities le ON le.tenant_id=i.tenant_id AND le.id=i.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND i.id=$3::uuid
		ON CONFLICT(tenant_id,legal_entity_id,item_id) DO NOTHING
		RETURNING item_id::text,distribution_id::text,issued_at
	`, scope.TenantID, scope.LegalEntityID, link.ItemID, link.DistributionID, link.IssuedAt).Scan(
		&stored.ItemID, &stored.DistributionID, &stored.IssuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.pool.QueryRow(ctx, `
			SELECT d.item_id::text,d.distribution_id::text,d.issued_at
			FROM rcsa_cycle_item_distributions d
			JOIN tenants t ON t.id=d.tenant_id
			JOIN legal_entities le ON le.tenant_id=d.tenant_id AND le.id=d.legal_entity_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND d.item_id=$3::uuid
		`, scope.TenantID, scope.LegalEntityID, link.ItemID).Scan(&stored.ItemID, &stored.DistributionID, &stored.IssuedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return DistributionLink{}, ErrNotFound
		}
		if err != nil {
			return DistributionLink{}, err
		}
		if stored.DistributionID != link.DistributionID {
			return DistributionLink{}, ErrConflict
		}
		return stored, nil
	}
	if err != nil {
		return DistributionLink{}, mapPostgresError(err)
	}
	return stored, nil
}

func aggregateFromRows(cycle Cycle, items []Item, controls []ItemControl, distributions []DistributionLink) Aggregate {
	value := Aggregate{Cycle: cycle, Items: items, Controls: map[string][]ItemControl{}, Distributions: map[string]DistributionLink{}}
	for _, control := range controls {
		value.Controls[control.ItemID] = append(value.Controls[control.ItemID], control)
	}
	for _, link := range distributions {
		value.Distributions[link.ItemID] = link
	}
	return value
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

var _ Repository = (*PostgresRepository)(nil)
