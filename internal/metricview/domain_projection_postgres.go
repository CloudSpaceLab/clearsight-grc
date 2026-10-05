//go:build postgres

package metricview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (m *DomainMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.Repository == nil || m.Repository.pool == nil || ctx == nil {
		return 0, ErrDomainMetricsInvalid
	}
	if limit <= 0 || limit > 250 {
		limit = 100
	}
	now = now.UTC()
	scopes, err := pendingDomainScopes(ctx, m.Repository.pool, now, limit)
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, scope := range scopes {
		if err := ctx.Err(); err != nil {
			return completed, err
		}
		inserted, err := maintainDomainScope(ctx, m.Repository.pool, scope, now)
		if err != nil {
			return completed, err
		}
		if inserted {
			completed++
		}
	}
	if _, err := m.Repository.pool.Exec(ctx, `
		DELETE FROM domain_metric_snapshots
		WHERE id IN (
			SELECT id
			FROM domain_metric_snapshots
			WHERE generated_at<$1
			ORDER BY generated_at,id
			LIMIT $2
		)`, now.Add(-RawObservationRetention-24*time.Hour), limit*len(domainDefinitions)); err != nil {
		return completed, fmt.Errorf("prune domain metric sources: %w", err)
	}
	return completed, nil
}

func pendingDomainScopes(ctx context.Context, pool *pgxpool.Pool, now time.Time, limit int) ([]domainScope, error) {
	bucket := now.UTC().Truncate(DomainSnapshotInterval)
	rows, err := pool.Query(ctx, `
		SELECT tenant.id::text,entity.id::text
		FROM legal_entities entity
		JOIN tenants tenant ON tenant.id=entity.tenant_id
		WHERE entity.valid_from<=$1
		  AND (entity.valid_until IS NULL OR $1<entity.valid_until)
		  AND NOT EXISTS (
			SELECT 1
			FROM domain_metric_snapshots source
			WHERE source.tenant_id=entity.tenant_id
			  AND source.legal_entity_id=entity.id
			  AND source.definition_revision=$2
			  AND source.bucket_start=$3
		  )
		ORDER BY entity.id
		LIMIT $4`, now, DomainDefinitionRevision, bucket, limit)
	if err != nil {
		return nil, fmt.Errorf("load pending domain metric scopes: %w", err)
	}
	defer rows.Close()
	values := make([]domainScope, 0, limit)
	for rows.Next() {
		var value domainScope
		if err := rows.Scan(&value.TenantID, &value.LegalEntityID); err != nil {
			return nil, fmt.Errorf("scan pending domain metric scope: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending domain metric scopes: %w", err)
	}
	return values, nil
}

func maintainDomainScope(ctx context.Context, pool *pgxpool.Pool, scope domainScope, at time.Time) (bool, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	highWater, err := domainSourceHighWater(ctx, tx, scope)
	if err != nil {
		return false, err
	}
	results := make([]domainMetricResult, 0, len(domainDefinitions))
	loaders := []func(context.Context, pgx.Tx, domainScope, time.Time) (domainMetricResult, error){
		loadOutsideAppetiteMetric,
		loadIndicatorBreachMetric,
		loadAssuranceFailureMetric,
		loadLossWithoutIssueMetric,
	}
	for _, load := range loaders {
		result, err := load(ctx, tx, scope, at)
		if err != nil {
			return false, err
		}
		results = append(results, result)
	}

	highWaterJSON, err := json.Marshal(highWater)
	if err != nil {
		return false, err
	}
	var sourceID string
	err = tx.QueryRow(ctx, `
		INSERT INTO domain_metric_snapshots(
			tenant_id,legal_entity_id,definition_revision,source_revision,source_high_water,bucket_start,generated_at
		) VALUES($1::uuid,$2::uuid,$3,$4,$5::jsonb,$6,$7)
		ON CONFLICT(tenant_id,legal_entity_id,definition_revision,bucket_start) DO NOTHING
		RETURNING id::text`,
		scope.TenantID, scope.LegalEntityID, DomainDefinitionRevision, DomainSourceRevision,
		highWaterJSON, at.UTC().Truncate(DomainSnapshotInterval), at.UTC(),
	).Scan(&sourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store domain metric source: %w", err)
	}

	zero := 0
	observations := make([]Observation, 0, len(results))
	for _, result := range results {
		for _, member := range result.Members {
			if _, err := tx.Exec(ctx, `
				INSERT INTO domain_metric_snapshot_memberships(
					source_id,metric_id,definition_revision,member_id,target_type,target_id,target_title,state
				) VALUES($1::uuid,$2,$3,$4::uuid,$5,$6::uuid,$7,$8)`,
				sourceID, result.Definition.ID, DomainDefinitionRevision, member.MemberID,
				member.TargetType, member.TargetID, member.Title, member.State,
			); err != nil {
				return false, fmt.Errorf("store domain metric member: %w", err)
			}
		}
		condition := ConditionClear
		if len(result.Members) > 0 {
			condition = ConditionAttention
		}
		completeness := CompletenessComplete
		if result.Unknown > 0 {
			completeness = CompletenessPartial
		}
		unknown := result.Unknown
		observations = append(observations, Observation{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
			MetricID: result.Definition.ID, DefinitionRevision: DomainDefinitionRevision,
			SourceKind: ObservationSourceDomainSnapshot, SourceID: sourceID, SourceRevision: DomainSourceRevision,
			SourceHighWater: cloneHighWater(highWater), GeneratedAt: at.UTC(),
			PeriodStart: at.UTC(), PeriodEnd: at.UTC(), PostureAsOf: at.UTC(),
			Value: len(result.Members), Condition: condition, Freshness: oversight.FreshnessCurrent,
			Completeness: completeness, Population: result.Population, Excluded: &zero, Unknown: &unknown,
		})
	}
	if _, err := storeObservationRows(ctx, tx, observations); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func domainSourceHighWater(ctx context.Context, tx pgx.Tx, scope domainScope) (map[string]time.Time, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `
		SELECT jsonb_strip_nulls(jsonb_build_object(
			'risks',(SELECT max(r.updated_at) FROM risks r WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid),
			'risk_assessments',(SELECT max(a.created_at) FROM risk_assessments a WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid),
			'risk_appetite',(SELECT max(a.created_at) FROM risk_appetite_statements a WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid),
			'risk_indicators',(SELECT max(l.created_at) FROM risk_indicator_links l WHERE l.tenant_id=$1::uuid AND l.legal_entity_id=$2::uuid),
			'programs',(SELECT max(program.updated_at) FROM programs program WHERE program.tenant_id=$1::uuid AND program.legal_entity_id=$2::uuid),
			'monitoring_checks',(SELECT max(check_config.updated_at)
				FROM monitoring_checks check_config
				JOIN programs program ON program.tenant_id=check_config.tenant_id AND program.id=check_config.program_id
				WHERE check_config.tenant_id=$1::uuid AND program.legal_entity_id=$2::uuid),
			'monitoring_results',(SELECT max(result.created_at)
				FROM monitoring_results result
				JOIN programs program ON program.tenant_id=result.tenant_id AND program.id=result.program_id
				WHERE result.tenant_id=$1::uuid AND program.legal_entity_id=$2::uuid),
			'control_definitions',(SELECT max(definition.updated_at)
				FROM control_definitions definition
				JOIN control_catalog_implementation_links catalog_link
				  ON catalog_link.tenant_id=definition.tenant_id AND catalog_link.definition_id=definition.id
				WHERE definition.tenant_id=$1::uuid AND catalog_link.legal_entity_id=$2::uuid),
			'control_implementations',(SELECT max(implementation.updated_at)
				FROM control_implementations implementation
				JOIN programs program ON program.tenant_id=implementation.tenant_id AND program.id=implementation.program_id
				WHERE implementation.tenant_id=$1::uuid AND program.legal_entity_id=$2::uuid),
			'risk_controls',(SELECT max(link.created_at) FROM risk_control_links link WHERE link.tenant_id=$1::uuid AND link.legal_entity_id=$2::uuid),
			'evidence_contracts',(SELECT max(contract.updated_at)
				FROM evidence_contracts contract
				JOIN programs program ON program.tenant_id=contract.tenant_id AND program.id=contract.program_id
				WHERE contract.tenant_id=$1::uuid AND program.legal_entity_id=$2::uuid),
			'assurance_assessments',(SELECT max(assessment.created_at)
				FROM evidence_assessments assessment
				JOIN programs program ON program.tenant_id=assessment.tenant_id AND program.id=assessment.program_id
				WHERE assessment.tenant_id=$1::uuid AND program.legal_entity_id=$2::uuid),
			'losses',(SELECT max(loss.updated_at) FROM operational_losses loss WHERE loss.tenant_id=$1::uuid AND loss.legal_entity_id=$2::uuid)
		))`, scope.TenantID, scope.LegalEntityID).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("load domain metric high-water marks: %w", err)
	}
	value := map[string]time.Time{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode domain metric high-water marks: %w", err)
		}
	}
	return value, nil
}
