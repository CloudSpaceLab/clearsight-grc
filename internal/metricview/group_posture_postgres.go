//go:build postgres

package metricview

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupPostureRepository struct {
	pool *pgxpool.Pool
}

func NewGroupPostureRepository(pool *pgxpool.Pool) *GroupPostureRepository {
	return &GroupPostureRepository{pool: pool}
}

func (r *GroupPostureRepository) ActiveGroupEntities(
	ctx context.Context,
	tenantID string,
	at time.Time,
) ([]GroupPostureEntity, error) {
	tenantID = strings.TrimSpace(tenantID)
	at = at.UTC()
	if r == nil || r.pool == nil || ctx == nil || tenantID == "" || at.IsZero() {
		return nil, ErrGroupPostureInvalid
	}
	rows, err := r.pool.Query(ctx, `
		SELECT entity.id::text,entity.code,entity.name,COALESCE(entity.jurisdiction,'')
		FROM tenants tenant
		JOIN legal_entities entity ON entity.tenant_id=tenant.id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND entity.valid_from<=$2
		  AND (entity.valid_until IS NULL OR $2<entity.valid_until)
		ORDER BY lower(entity.name),entity.id`, tenantID, at)
	if err != nil {
		return nil, fmt.Errorf("list Group legal entities: %w", err)
	}
	defer rows.Close()

	values := make([]GroupPostureEntity, 0, 16)
	for rows.Next() {
		var value GroupPostureEntity
		if err := rows.Scan(
			&value.LegalEntityID,
			&value.LegalEntityCode,
			&value.LegalEntityName,
			&value.Jurisdiction,
		); err != nil {
			return nil, fmt.Errorf("scan Group legal entity: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Group legal entities: %w", err)
	}
	if len(values) == 0 {
		return nil, ErrGroupPostureMissing
	}
	return values, nil
}

func (r *GroupPostureRepository) GroupPosture(
	ctx context.Context,
	tenantID string,
	legalEntityIDs []string,
	periodStart time.Time,
	periodEnd time.Time,
	generatedAt time.Time,
) (GroupPostureBundle, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityIDs = normalizeGroupLegalEntityIDs(legalEntityIDs)
	periodStart = periodStart.UTC()
	periodEnd = periodEnd.UTC()
	generatedAt = generatedAt.UTC()
	if r == nil || r.pool == nil || ctx == nil || tenantID == "" || len(legalEntityIDs) < 2 ||
		periodStart.IsZero() || periodEnd.IsZero() || generatedAt.IsZero() ||
		periodEnd.Before(periodStart) || generatedAt.Before(periodEnd) {
		return GroupPostureBundle{}, ErrGroupPostureInvalid
	}

	rows, err := r.pool.Query(ctx, `
		WITH selected_tenant AS (
			SELECT id
			FROM tenants
			WHERE id::text=$1 OR slug=$1
			LIMIT 1
		), entities AS (
			SELECT entity.id,entity.code,entity.name,COALESCE(entity.jurisdiction,'') AS jurisdiction
			FROM legal_entities entity
			JOIN selected_tenant tenant ON tenant.id=entity.tenant_id
			WHERE entity.id=ANY($2::uuid[])
			  AND entity.valid_from<=$5
			  AND (entity.valid_until IS NULL OR $5<entity.valid_until)
		), latest AS (
			SELECT entity.*,
			       source.id AS source_id,
			       source.generated_at AS source_generated_at,
			       source.source_revision,
			       source.definition_revision
			FROM entities entity
			LEFT JOIN LATERAL (
				SELECT snapshot.id,snapshot.generated_at,snapshot.source_revision,snapshot.definition_revision
				FROM domain_metric_snapshots snapshot
				JOIN selected_tenant tenant ON tenant.id=snapshot.tenant_id
				WHERE snapshot.legal_entity_id=entity.id
				  AND snapshot.definition_revision=$6
				  AND snapshot.generated_at<=$5
				ORDER BY snapshot.generated_at DESC,snapshot.id DESC
				LIMIT 1
			) source ON true
		), metric_rows AS (
			SELECT latest.id,
			       latest.code,
			       latest.name,
			       latest.jurisdiction,
			       latest.source_id,
			       latest.source_generated_at,
			       latest.source_revision,
			       latest.definition_revision,
			       count(observation.metric_id) FILTER (
			         WHERE observation.metric_id=ANY($7::text[])
			       ) AS metric_count,
			       max(observation.value) FILTER (
			         WHERE observation.metric_id='risks_outside_appetite'
			       ) AS outside_appetite,
			       max(observation.value) FILTER (
			         WHERE observation.metric_id='indicator_breaches'
			       ) AS indicator_breaches,
			       max(observation.value) FILTER (
			         WHERE observation.metric_id='assurance_failures'
			       ) AS assurance_failures,
			       COALESCE(sum(COALESCE(observation.unknown,0)) FILTER (
			         WHERE observation.metric_id=ANY($7::text[])
			       ),0)::bigint AS unknown_count,
			       COALESCE(sum(COALESCE(observation.excluded,0)) FILTER (
			         WHERE observation.metric_id=ANY($7::text[])
			       ),0)::bigint AS excluded_count,
			       COALESCE(bool_and(
			         observation.completeness='COMPLETE'
			         AND COALESCE(observation.unknown,0)=0
			         AND COALESCE(observation.excluded,0)=0
			       ) FILTER (WHERE observation.metric_id=ANY($7::text[])),false) AS complete_metrics
			FROM latest
			LEFT JOIN metric_observations observation
			  ON observation.source_kind=$8
			 AND observation.source_id=latest.source_id
			 AND observation.definition_revision=$6
			 AND observation.metric_id=ANY($7::text[])
			GROUP BY latest.id,latest.code,latest.name,latest.jurisdiction,
			         latest.source_id,latest.source_generated_at,latest.source_revision,latest.definition_revision
		), loss_counts AS (
			SELECT entity.id,
			       count(loss.id)::bigint AS event_count
			FROM entities entity
			LEFT JOIN operational_losses loss
			  ON loss.legal_entity_id=entity.id
			 AND loss.status='ACTIVE'
			 AND loss.occurred_at>=$3
			 AND loss.occurred_at<=$4
			GROUP BY entity.id
		)
		SELECT metric.id::text,
		       metric.code,
		       metric.name,
		       metric.jurisdiction,
		       COALESCE(metric.source_id::text,''),
		       metric.source_generated_at,
		       COALESCE(metric.source_revision,''),
		       COALESCE(metric.definition_revision,''),
		       metric.metric_count,
		       COALESCE(metric.outside_appetite,0),
		       COALESCE(metric.indicator_breaches,0),
		       COALESCE(metric.assurance_failures,0),
		       metric.unknown_count,
		       metric.excluded_count,
		       metric.complete_metrics,
		       loss.event_count
		FROM metric_rows metric
		JOIN loss_counts loss ON loss.id=metric.id
		ORDER BY lower(metric.name),metric.id`,
		tenantID,
		legalEntityIDs,
		periodStart,
		periodEnd,
		generatedAt,
		DomainDefinitionRevision,
		[]string{"risks_outside_appetite", "indicator_breaches", "assurance_failures"},
		ObservationSourceDomainSnapshot,
	)
	if err != nil {
		return GroupPostureBundle{}, fmt.Errorf("load Group CRO posture: %w", err)
	}
	defer rows.Close()

	value := GroupPostureBundle{
		GeneratedAt: generatedAt,
		PeriodStart: periodStart,
		PeriodEnd: periodEnd,
		DefinitionRevision: DomainDefinitionRevision,
		Children: make([]GroupPostureChild, 0, len(legalEntityIDs)),
	}
	value.RiskCoverage.AuthorizedChildren = len(legalEntityIDs)
	value.LossCoverage.AuthorizedChildren = len(legalEntityIDs)

	for rows.Next() {
		var child GroupPostureChild
		var sourceGeneratedAt *time.Time
		var metricCount int
		var completeMetrics bool
		if err := rows.Scan(
			&child.LegalEntityID,
			&child.LegalEntityCode,
			&child.LegalEntityName,
			&child.Jurisdiction,
			&child.SourceID,
			&sourceGeneratedAt,
			&child.SourceRevision,
			&child.DefinitionRevision,
			&metricCount,
			&child.Counts.RisksOutsideAppetite,
			&child.Counts.IndicatorBreaches,
			&child.Counts.AssuranceFailures,
			&child.Unknown,
			&child.Excluded,
			&completeMetrics,
			&child.Counts.LossEvents,
		); err != nil {
			return GroupPostureBundle{}, fmt.Errorf("scan Group CRO posture: %w", err)
		}
		child.SourceGeneratedAt = sourceGeneratedAt
		child.Freshness = oversight.FreshnessStale
		child.Completeness = CompletenessUnknown
		child.RiskState = "MISSING"

		if metricCount == 3 && sourceGeneratedAt != nil {
			child.RiskState = "AVAILABLE"
			child.Freshness = oversight.FreshnessCurrent
			child.Completeness = CompletenessPartial
			if completeMetrics {
				child.Completeness = CompletenessComplete
			}
			if generatedAt.Sub(sourceGeneratedAt.UTC()) > DomainSnapshotFreshness {
				child.RiskState = "STALE"
				child.Freshness = oversight.FreshnessStale
			}
			value.RiskCoverage.IncludedChildren++
			if child.RiskState == "STALE" {
				value.RiskCoverage.StaleChildren++
			}
			if child.Completeness != CompletenessComplete {
				value.RiskCoverage.PartialChildren++
			}
			value.Counts.RisksOutsideAppetite += child.Counts.RisksOutsideAppetite
			value.Counts.IndicatorBreaches += child.Counts.IndicatorBreaches
			value.Counts.AssuranceFailures += child.Counts.AssuranceFailures
		} else {
			value.RiskCoverage.MissingChildren++
		}

		value.LossCoverage.IncludedChildren++
		value.Counts.LossEvents += child.Counts.LossEvents
		value.Children = append(value.Children, child)
	}
	if err := rows.Err(); err != nil {
		return GroupPostureBundle{}, fmt.Errorf("iterate Group CRO posture: %w", err)
	}
	if len(value.Children) != len(legalEntityIDs) {
		return GroupPostureBundle{}, ErrGroupPostureMissing
	}

	value.RiskCoverage.Complete = value.RiskCoverage.MissingChildren == 0 &&
		value.RiskCoverage.StaleChildren == 0 &&
		value.RiskCoverage.PartialChildren == 0
	value.LossCoverage.Complete = value.LossCoverage.IncludedChildren == value.LossCoverage.AuthorizedChildren
	return value, nil
}

func normalizeGroupLegalEntityIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

var _ GroupPostureReader = (*GroupPostureRepository)(nil)
