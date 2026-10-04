//go:build postgres

package metricview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DomainSnapshotFreshness = 2 * time.Hour

type DomainRepository struct {
	pool *pgxpool.Pool
}

type DomainMaintainer struct {
	Repository *ObservationRepository
}

type domainScope struct {
	TenantID      string
	LegalEntityID string
}

type domainMetricMember struct {
	MemberID   string
	TargetType string
	TargetID   string
	Title      string
	State      string
}

type domainMetricResult struct {
	Definition Definition
	Population int
	Unknown    int
	Members    []domainMetricMember
}

func NewDomainRepository(pool *pgxpool.Pool) *DomainRepository {
	return &DomainRepository{pool: pool}
}

func NewDomainMaintainer(repository *ObservationRepository) *DomainMaintainer {
	return &DomainMaintainer{Repository: repository}
}

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
	bucket := now.UTC().Truncate(time.Hour)
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
		loadLossInterventionMetric,
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
		highWaterJSON, at.UTC().Truncate(time.Hour), at.UTC(),
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
			'monitoring_results',(SELECT max(result.created_at)
				FROM monitoring_results result
				JOIN programs program ON program.tenant_id=result.tenant_id AND program.id=result.program_id
				WHERE result.tenant_id=$1::uuid AND program.legal_entity_id=$2::uuid),
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

func loadOutsideAppetiteMetric(ctx context.Context, tx pgx.Tx, scope domainScope, at time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("risks_outside_appetite")
	rows, err := tx.Query(ctx, `
		WITH selected AS (
			SELECT risk.id,risk.name,risk.version,
			       CASE
			         WHEN assessment.id IS NULL OR assessment.risk_version<>risk.version THEN 'UNKNOWN'
			         WHEN assessment.appetite_statement_id IS NULL THEN 'UNKNOWN'
			         WHEN appetite.id IS NULL OR assessment.appetite_statement_id<>appetite.id THEN 'UNKNOWN'
			         ELSE COALESCE(NULLIF(assessment.appetite_position,''),'UNKNOWN')
			       END state
			FROM risks risk
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM risk_assessments candidate
				WHERE candidate.tenant_id=risk.tenant_id
				  AND candidate.legal_entity_id=risk.legal_entity_id
				  AND candidate.risk_id=risk.id
				ORDER BY candidate.risk_version DESC,candidate.id DESC
				LIMIT 1
			) assessment ON true
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM (
					SELECT statement.*
					FROM risk_appetite_statements statement
					WHERE statement.tenant_id=risk.tenant_id
					  AND statement.legal_entity_id=risk.legal_entity_id
					  AND statement.risk_id=risk.id
					  AND statement.effective_from<=$3
					ORDER BY statement.version DESC,statement.id DESC
					LIMIT 1
				) candidate
				WHERE candidate.status='ACTIVE'
				  AND (candidate.effective_until IS NULL OR $3<candidate.effective_until)
			) appetite ON true
			WHERE risk.tenant_id=$1::uuid
			  AND risk.legal_entity_id=$2::uuid
			  AND risk.status='ACTIVE'
		)
		SELECT id::text,name,state FROM selected ORDER BY id`, scope.TenantID, scope.LegalEntityID, at.UTC())
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load outside-appetite metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var id, title, state string
		if err := rows.Scan(&id, &title, &state); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if state == "UNKNOWN" {
			result.Unknown++
		}
		if state == "BREACHED" {
			result.Members = append(result.Members, domainMetricMember{MemberID: id, TargetType: "RISK", TargetID: id, Title: title, State: state})
		}
	}
	return result, rows.Err()
}

func loadIndicatorBreachMetric(ctx context.Context, tx pgx.Tx, scope domainScope, at time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("indicator_breaches")
	rows, err := tx.Query(ctx, `
		WITH current_links AS (
			SELECT DISTINCT ON (link.risk_id,link.monitoring_check_id)
			       link.id,link.risk_id,link.program_id,link.monitoring_check_id,link.monitoring_check_version,link.kind
			FROM risk_indicator_links link
			JOIN risks risk
			  ON risk.tenant_id=link.tenant_id
			 AND risk.legal_entity_id=link.legal_entity_id
			 AND risk.id=link.risk_id
			 AND risk.status='ACTIVE'
			WHERE link.tenant_id=$1::uuid
			  AND link.legal_entity_id=$2::uuid
			ORDER BY link.risk_id,link.monitoring_check_id,link.risk_version DESC,link.id DESC
		)
		SELECT link.id::text,risk.id::text,risk.name,
		       CASE
		         WHEN check.revision_id IS NULL OR check.status<>'ACTIVE' OR NOT check.is_current THEN 'UNKNOWN'
		         WHEN program.id IS NULL OR program.status<>'ACTIVE' THEN 'UNKNOWN'
		         WHEN result.id IS NULL THEN 'UNKNOWN'
		         WHEN result.evaluated_at < $3-(check.freshness_minutes*interval '1 minute') THEN 'UNKNOWN'
		         WHEN NULLIF(result.evaluation->>'coverage','')::numeric < check.minimum_coverage THEN 'UNKNOWN'
		         WHEN result.evaluation->>'band' IN ('HIGH','CRITICAL') THEN 'BREACH'
		         WHEN result.evaluation->>'band'='MODERATE' THEN 'WATCH'
		         WHEN result.evaluation->>'band'='LOW' THEN 'NORMAL'
		         ELSE 'UNKNOWN'
		       END state
		FROM current_links link
		JOIN risks risk ON risk.id=link.risk_id AND risk.tenant_id=$1::uuid AND risk.legal_entity_id=$2::uuid
		LEFT JOIN monitoring_checks check
		  ON check.tenant_id=$1::uuid
		 AND check.id=link.monitoring_check_id
		 AND check.version=link.monitoring_check_version
		 AND check.program_id=link.program_id
		LEFT JOIN programs program
		  ON program.tenant_id=$1::uuid
		 AND program.id=link.program_id
		 AND program.legal_entity_id=$2::uuid
		LEFT JOIN LATERAL (
			SELECT candidate.*
			FROM monitoring_results candidate
			WHERE candidate.tenant_id=$1::uuid
			  AND candidate.program_id=link.program_id
			  AND candidate.monitoring_check_id=link.monitoring_check_id
			  AND candidate.monitoring_check_version=link.monitoring_check_version
			  AND candidate.evaluated_at<=$3
			ORDER BY candidate.evaluated_at DESC,candidate.id DESC
			LIMIT 1
		) result ON true
		ORDER BY link.id`, scope.TenantID, scope.LegalEntityID, at.UTC())
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load indicator-breach metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var memberID, riskID, title, state string
		if err := rows.Scan(&memberID, &riskID, &title, &state); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if state == "UNKNOWN" {
			result.Unknown++
		}
		if state == "BREACH" {
			result.Members = append(result.Members, domainMetricMember{MemberID: memberID, TargetType: "RISK", TargetID: riskID, Title: title, State: state})
		}
	}
	return result, rows.Err()
}

func loadAssuranceFailureMetric(ctx context.Context, tx pgx.Tx, scope domainScope, at time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("assurance_failures")
	rows, err := tx.Query(ctx, `
		WITH selected_risks AS (
			SELECT risk.id,risk.tenant_id,risk.legal_entity_id,risk.name
			FROM risks risk
			WHERE risk.tenant_id=$1::uuid
			  AND risk.legal_entity_id=$2::uuid
			  AND risk.status='ACTIVE'
		), eligible_links AS (
			SELECT risk_link.risk_id,
			       implementation.tenant_id,
			       implementation.program_id,
			       implementation.id implementation_id,
			       implementation.status implementation_status
			FROM risk_control_links risk_link
			JOIN control_catalog_implementation_links catalog_link
			  ON catalog_link.id=risk_link.catalog_link_id
			 AND catalog_link.tenant_id=risk_link.tenant_id
			 AND catalog_link.legal_entity_id=risk_link.legal_entity_id
			JOIN control_definitions definition
			  ON definition.id=catalog_link.definition_id
			 AND definition.tenant_id=catalog_link.tenant_id
			 AND definition.status='ACTIVE'
			JOIN control_implementations implementation
			  ON implementation.id=catalog_link.implementation_id
			 AND implementation.tenant_id=catalog_link.tenant_id
			 AND implementation.program_id=catalog_link.program_id
			 AND implementation.status NOT IN ('INACTIVE','RETIRED')
			 AND implementation.effective_from<=$3
			 AND (implementation.effective_until IS NULL OR $3<implementation.effective_until)
			WHERE risk_link.tenant_id=$1::uuid
			  AND risk_link.legal_entity_id=$2::uuid
			  AND EXISTS (SELECT 1 FROM selected_risks risk WHERE risk.id=risk_link.risk_id)
		), contract_facts AS (
			SELECT risk.id risk_id,
			       CASE
			         WHEN link.implementation_id IS NULL OR link.implementation_status<>'IMPLEMENTED' THEN 'UNKNOWN'
			         WHEN contract.id IS NULL THEN 'UNKNOWN'
			         WHEN assessment.id IS NULL THEN 'UNKNOWN'
			         WHEN assessment.valid_until IS NOT NULL AND NOT ($3<assessment.valid_until) THEN 'UNKNOWN'
			         WHEN assessment.conclusion IN ('UNSUPPORTED','CONTRADICTED') THEN 'FAILED'
			         WHEN assessment.conclusion='PARTIALLY_SUPPORTED' OR assessment.coverage<contract.minimum_coverage THEN 'PARTIAL'
			         WHEN assessment.conclusion='SUPPORTED' THEN 'SUPPORTED'
			         ELSE 'UNKNOWN'
			       END state
			FROM selected_risks risk
			LEFT JOIN eligible_links link ON link.risk_id=risk.id
			LEFT JOIN evidence_contracts contract
			  ON contract.tenant_id=link.tenant_id
			 AND contract.program_id=link.program_id
			 AND contract.control_implementation_id=link.implementation_id
			 AND contract.status='ACTIVE'
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM evidence_assessments candidate
				WHERE candidate.tenant_id=contract.tenant_id
				  AND candidate.program_id=contract.program_id
				  AND candidate.contract_id=contract.id
				  AND candidate.assessed_at<=$3
				ORDER BY candidate.assessed_at DESC,candidate.id DESC
				LIMIT 1
			) assessment ON true
		), risk_state AS (
			SELECT risk.id,risk.name,
			       CASE
			         WHEN bool_or(fact.state='FAILED') THEN 'FAILED'
			         WHEN bool_or(fact.state='UNKNOWN') AND bool_or(fact.state IN ('SUPPORTED','PARTIAL')) THEN 'PARTIAL'
			         WHEN bool_or(fact.state='UNKNOWN') THEN 'UNKNOWN'
			         WHEN bool_or(fact.state='PARTIAL') THEN 'PARTIAL'
			         WHEN bool_or(fact.state='SUPPORTED') THEN 'SUPPORTED'
			         ELSE 'UNKNOWN'
			       END state
			FROM selected_risks risk
			LEFT JOIN contract_facts fact ON fact.risk_id=risk.id
			GROUP BY risk.id,risk.name
		)
		SELECT id::text,name,state FROM risk_state ORDER BY id`, scope.TenantID, scope.LegalEntityID, at.UTC())
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load assurance-failure metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var id, title, state string
		if err := rows.Scan(&id, &title, &state); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if state == "UNKNOWN" {
			result.Unknown++
		}
		if state == "FAILED" {
			result.Members = append(result.Members, domainMetricMember{MemberID: id, TargetType: "RISK", TargetID: id, Title: title, State: state})
		}
	}
	return result, rows.Err()
}

func loadLossInterventionMetric(ctx context.Context, tx pgx.Tx, scope domainScope, _ time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("losses_without_intervention")
	rows, err := tx.Query(ctx, `
		SELECT loss.id::text,loss.title,loss.matter_id IS NULL
		FROM operational_losses loss
		WHERE loss.tenant_id=$1::uuid
		  AND loss.legal_entity_id=$2::uuid
		  AND loss.status='ACTIVE'
		ORDER BY loss.id`, scope.TenantID, scope.LegalEntityID)
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load loss-intervention metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var id, title string
		var needsIntervention bool
		if err := rows.Scan(&id, &title, &needsIntervention); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if needsIntervention {
			result.Members = append(result.Members, domainMetricMember{MemberID: id, TargetType: "LOSS", TargetID: id, Title: title, State: "NEEDS_INTERVENTION"})
		}
	}
	return result, rows.Err()
}

func (r *DomainRepository) LatestDomainMetrics(ctx context.Context, tenantID, legalEntityID string) (DomainBundle, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return DomainBundle{}, ErrDomainMetricsInvalid
	}
	tenantID, legalEntityID = strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return DomainBundle{}, ErrDomainMetricsInvalid
	}
	var bundle DomainBundle
	err := r.pool.QueryRow(ctx, `
		SELECT source.id::text,source.generated_at,source.source_revision,source.definition_revision,entity.id::text
		FROM domain_metric_snapshots source
		JOIN tenants tenant ON tenant.id=source.tenant_id
		JOIN legal_entities entity ON entity.tenant_id=source.tenant_id AND entity.id=source.legal_entity_id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		  AND source.definition_revision=$3
		ORDER BY source.generated_at DESC,source.id DESC
		LIMIT 1`, tenantID, legalEntityID, DomainDefinitionRevision).Scan(
		&bundle.SourceID, &bundle.GeneratedAt, &bundle.SourceRevision, &bundle.DefinitionRevision, &bundle.ScopeID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DomainBundle{}, ErrDomainMetricsNotFound
	}
	if err != nil {
		return DomainBundle{}, fmt.Errorf("load latest domain metric source: %w", err)
	}
	bundle.PostureAsOf = bundle.GeneratedAt.UTC()
	bundle.ScopeKind = "LEGAL_ENTITY"
	rows, err := r.pool.Query(ctx, `
		SELECT metric_id,value,condition,freshness,completeness,population,excluded,unknown
		FROM metric_observations
		WHERE source_kind=$1
		  AND source_id=$2::uuid
		  AND definition_revision=$3
		ORDER BY metric_id`, ObservationSourceDomainSnapshot, bundle.SourceID, DomainDefinitionRevision)
	if err != nil {
		return DomainBundle{}, fmt.Errorf("load latest domain metrics: %w", err)
	}
	defer rows.Close()
	bundle.Items = make([]Metric, 0, len(domainDefinitions))
	for rows.Next() {
		var item Metric
		if err := rows.Scan(&item.ID, &item.Value, &item.Condition, &item.Freshness, &item.Completeness, &item.Population, &item.Excluded, &item.Unknown); err != nil {
			return DomainBundle{}, fmt.Errorf("scan latest domain metric: %w", err)
		}
		definition, ok := DomainDefinition(item.ID)
		if !ok {
			return DomainBundle{}, ErrDefinitionMismatch
		}
		item.Label = definition.Label
		item.Unit = definition.Unit
		item.Basis = definition.Basis
		item.Drill = definition.Drill
		item.DefinitionRevision = definition.Revision
		item.SourceRevision = bundle.SourceRevision
		item.GeneratedAt = bundle.GeneratedAt.UTC()
		if time.Since(bundle.GeneratedAt.UTC()) > DomainSnapshotFreshness {
			item.Freshness = oversight.FreshnessStale
		}
		bundle.Items = append(bundle.Items, item)
	}
	if err := rows.Err(); err != nil {
		return DomainBundle{}, err
	}
	if len(bundle.Items) != len(domainDefinitions) {
		return DomainBundle{}, ErrDomainMetricsNotFound
	}
	return bundle, nil
}

var _ DomainReader = (*DomainRepository)(nil)
