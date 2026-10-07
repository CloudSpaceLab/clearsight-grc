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

var ErrDefinitionMismatch = errors.New("stored metric definitions do not match the runtime contract")

type ObservationRepository struct {
	pool *pgxpool.Pool
}

func NewObservationRepository(pool *pgxpool.Pool) *ObservationRepository {
	return &ObservationRepository{pool: pool}
}

type ObservationMaintainer struct {
	Repository *ObservationRepository
	Domain     *DomainMaintainer
}

type sourceSnapshot struct {
	ID       string
	TenantID string
	EntityID string
	Value    oversight.Snapshot
}

func (m *ObservationMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.Repository == nil || m.Repository.pool == nil || ctx == nil {
		return 0, ErrInvalidObservation
	}
	if limit <= 0 || limit > 250 {
		limit = 100
	}
	if err := m.Repository.validateDefinitions(ctx); err != nil {
		return 0, err
	}
	completed := 0
	if m.Domain != nil {
		domainCompleted, err := m.Domain.Maintain(ctx, now, limit)
		if err != nil {
			return 0, err
		}
		completed += domainCompleted
	}
	sources, err := m.Repository.pendingOversightSnapshots(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, source := range sources {
		if err := ctx.Err(); err != nil {
			return completed, err
		}
		bundle := FromOversight(source.Value)
		observations, err := ObservationsFromBundle(
			source.TenantID,
			source.EntityID,
			source.ID,
			source.Value.SourceHighWater,
			bundle,
		)
		if err != nil {
			return completed, err
		}
		inserted, err := m.Repository.storeObservations(ctx, observations)
		if err != nil {
			return completed, err
		}
		if inserted {
			completed++
		}
	}
	if err := m.Repository.maintainTrendRetention(ctx, now, limit); err != nil {
		return completed, err
	}
	return completed, nil
}

func (r *ObservationRepository) validateDefinitions(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return ErrInvalidObservation
	}
	if err := r.validateDefinitionRevision(ctx, HomeDefinitionRevision, HomeDefinitionList()); err != nil {
		return err
	}
	return r.validateDefinitionRevision(ctx, DomainDefinitionRevision, DomainDefinitionList())
}

func (r *ObservationRepository) validateDefinitionRevision(ctx context.Context, revision string, expected []Definition) error {
	rows, err := r.pool.Query(ctx, `
		SELECT metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
		       drill_workspace,drill_filter,drill_consistency
		FROM metric_definitions
		WHERE revision=$1
		ORDER BY metric_id`, revision)
	if err != nil {
		return fmt.Errorf("load metric definitions: %w", err)
	}
	defer rows.Close()

	stored := make(map[string]Definition, len(expected))
	for rows.Next() {
		var definition Definition
		var unit, basis, conditionRule, aggregationRule, consistency string
		if err := rows.Scan(
			&definition.ID,
			&definition.Revision,
			&definition.Label,
			&unit,
			&basis,
			&conditionRule,
			&aggregationRule,
			&definition.Drill.Workspace,
			&definition.Drill.Filter,
			&consistency,
		); err != nil {
			return fmt.Errorf("scan metric definition: %w", err)
		}
		definition.Unit = MetricUnit(unit)
		definition.Basis = MetricBasis(basis)
		definition.ConditionRule = ConditionRule(conditionRule)
		definition.AggregationRule = AggregationRule(aggregationRule)
		definition.Drill.Consistency = DrillConsistency(consistency)
		stored[definition.ID] = definition
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate metric definitions: %w", err)
	}
	if len(stored) != len(expected) {
		return ErrDefinitionMismatch
	}
	for _, definition := range expected {
		actual, ok := stored[definition.ID]
		if !ok || actual != definition {
			return ErrDefinitionMismatch
		}
	}
	return nil
}

func (r *ObservationRepository) pendingOversightSnapshots(ctx context.Context, limit int) ([]sourceSnapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT os.id::text,t.id::text,le.id::text,
		       os.generated_at,os.period_start,os.period_end,os.projection_version,
		       os.source_high_water,os.coverage_population,os.coverage_excluded,os.coverage_unknown,os.payload
		FROM oversight_snapshots os
		JOIN tenants t ON t.id=os.tenant_id
		JOIN legal_entities le ON le.tenant_id=os.tenant_id AND le.id=os.legal_entity_id
		WHERE os.metric_membership_revision=$2
		  AND (
			SELECT count(*)
			FROM metric_observations observation
			WHERE observation.source_kind=$1
			  AND observation.source_id=os.id
			  AND observation.definition_revision=$2
		) < $3
		ORDER BY os.generated_at,os.id
		LIMIT $4`,
		ObservationSourceOversightSnapshot,
		HomeDefinitionRevision,
		len(homeDefinitions),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("load pending metric sources: %w", err)
	}
	defer rows.Close()

	values := make([]sourceSnapshot, 0, limit)
	for rows.Next() {
		var source sourceSnapshot
		var highWater, payload []byte
		if err := rows.Scan(
			&source.ID,
			&source.TenantID,
			&source.EntityID,
			&source.Value.GeneratedAt,
			&source.Value.PeriodStart,
			&source.Value.PeriodEnd,
			&source.Value.ProjectionVersion,
			&highWater,
			&source.Value.Coverage.Population,
			&source.Value.Coverage.Excluded,
			&source.Value.Coverage.Unknown,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("scan pending metric source: %w", err)
		}
		source.Value.SnapshotID = source.ID
		source.Value.TenantID = source.TenantID
		source.Value.LegalEntityID = source.EntityID
		source.Value.PostureAsOf = source.Value.GeneratedAt
		source.Value.Freshness = oversight.FreshnessCurrent
		if source.Value.ProjectionVersion != oversight.ProjectionVersion {
			source.Value.Freshness = oversight.FreshnessStale
		}
		source.Value.SourceHighWater = map[string]time.Time{}
		if len(highWater) > 0 {
			if err := json.Unmarshal(highWater, &source.Value.SourceHighWater); err != nil {
				return nil, fmt.Errorf("decode metric source high-water marks: %w", err)
			}
		}
		var metadata struct {
			Counts oversight.Counts `json:"counts"`
		}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &metadata); err != nil {
				return nil, fmt.Errorf("decode metric source snapshot: %w", err)
			}
		}
		source.Value.Counts = metadata.Counts
		values = append(values, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending metric sources: %w", err)
	}
	return values, nil
}

func (r *ObservationRepository) storeObservations(ctx context.Context, values []Observation) (bool, error) {
	if r == nil || r.pool == nil || !validObservationSet(values) {
		return false, ErrInvalidObservation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted, err := storeObservationRows(ctx, tx, values)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return inserted, nil
}

func validObservationSet(values []Observation) bool {
	if len(values) == 0 {
		return false
	}
	revision := values[0].DefinitionRevision
	sourceKind := values[0].SourceKind
	expected := 0
	switch {
	case revision == HomeDefinitionRevision && sourceKind == ObservationSourceOversightSnapshot:
		expected = len(homeDefinitions)
	case revision == DomainDefinitionRevision && sourceKind == ObservationSourceDomainSnapshot:
		expected = len(domainDefinitions)
	default:
		return false
	}
	if len(values) != expected {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.DefinitionRevision != revision || value.SourceKind != sourceKind {
			return false
		}
		definition, ok := metricDefinition(value.MetricID)
		if !ok || definition.Revision != revision || !validObservationMeasure(value, definition) {
			return false
		}
		if _, duplicate := seen[value.MetricID]; duplicate {
			return false
		}
		seen[value.MetricID] = struct{}{}
	}
	return true
}

func validObservationMeasure(value Observation, definition Definition) bool {
	if value.Value < 0 {
		return false
	}
	switch definition.Unit {
	case MetricUnitCount:
		return strings.TrimSpace(value.Currency) == "" && value.MemberCount == nil
	case MetricUnitMoney:
		return validMetricCurrency(value.Currency) && value.MemberCount != nil && *value.MemberCount >= 0
	default:
		return false
	}
}

func validMetricCurrency(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 3 {
		return false
	}
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return true
}

func storeObservationRows(ctx context.Context, tx pgx.Tx, values []Observation) (bool, error) {
	if tx == nil || !validObservationSet(values) {
		return false, ErrInvalidObservation
	}
	inserted := false
	for _, value := range values {
		if strings.TrimSpace(value.TenantID) == "" || strings.TrimSpace(value.LegalEntityID) == "" ||
			strings.TrimSpace(value.MetricID) == "" || strings.TrimSpace(value.SourceID) == "" {
			return false, ErrInvalidObservation
		}
		highWater, err := json.Marshal(value.SourceHighWater)
		if err != nil {
			return false, err
		}
		command, err := tx.Exec(ctx, `
			INSERT INTO metric_observations(
				tenant_id,legal_entity_id,metric_id,definition_revision,
				source_kind,source_id,source_revision,source_high_water,
				generated_at,period_start,period_end,posture_as_of,
				value,condition,currency,member_count,freshness,completeness,population,excluded,unknown
			) VALUES(
				$1::uuid,$2::uuid,$3,$4,
				$5,$6::uuid,$7,$8::jsonb,
				$9,$10,$11,$12,
				$13,$14,NULLIF($15,''),$16,$17,$18,$19,$20,$21
			)
			ON CONFLICT(source_kind,source_id,metric_id,definition_revision) DO NOTHING`,
			value.TenantID,
			value.LegalEntityID,
			value.MetricID,
			value.DefinitionRevision,
			value.SourceKind,
			value.SourceID,
			value.SourceRevision,
			highWater,
			value.GeneratedAt,
			value.PeriodStart,
			value.PeriodEnd,
			value.PostureAsOf,
			value.Value,
			value.Condition,
			value.Currency,
			value.MemberCount,
			value.Freshness,
			value.Completeness,
			value.Population,
			value.Excluded,
			value.Unknown,
		)
		if err != nil {
			return false, err
		}
		inserted = inserted || command.RowsAffected() == 1
	}
	return inserted, nil
}

func (r *ObservationRepository) countObservationsForSource(ctx context.Context, sourceID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM metric_observations
		WHERE source_kind=$1 AND source_id=$2::uuid AND definition_revision=$3`,
		ObservationSourceOversightSnapshot, sourceID, HomeDefinitionRevision,
	).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return count, err
}
