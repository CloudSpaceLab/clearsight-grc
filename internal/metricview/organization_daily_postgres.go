//go:build postgres

package metricview

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var organizationDailyPostureMetricIDs = []string{
	"risks_outside_appetite",
}

type organizationDailySourceCandidate struct {
	TenantID           string
	LegalEntityID      string
	DefinitionRevision string
	BucketDate         time.Time
	SourceID           string
	SourceRevision     string
	SourceGeneratedAt  time.Time
	SourceHighWater    []byte
}

func maintainOrganizationDailyHistory(
	ctx context.Context,
	pool *pgxpool.Pool,
	now time.Time,
	limit int,
) (int, error) {
	if ctx == nil || pool == nil {
		return 0, ErrDomainMetricsInvalid
	}
	if limit <= 0 || limit > 250 {
		limit = 100
	}
	now = now.UTC()

	candidates, err := pendingOrganizationDailySources(ctx, pool, now, limit)
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return completed, err
		}
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if err != nil {
			return completed, err
		}
		inserted, finalizeErr := finalizeOrganizationMetricDay(ctx, tx, candidate)
		if finalizeErr != nil {
			_ = tx.Rollback(ctx)
			return completed, finalizeErr
		}
		if err := tx.Commit(ctx); err != nil {
			return completed, err
		}
		if inserted {
			completed++
		}
	}
	return completed, nil
}

func pendingOrganizationDailySources(
	ctx context.Context,
	pool *pgxpool.Pool,
	now time.Time,
	limit int,
) ([]organizationDailySourceCandidate, error) {
	rows, err := pool.Query(ctx, `
		WITH ranked AS (
			SELECT source.tenant_id,
			       source.legal_entity_id,
			       source.definition_revision,
			       (source.generated_at AT TIME ZONE 'UTC')::date AS bucket_date,
			       source.id,
			       source.source_revision,
			       source.generated_at,
			       source.source_high_water,
			       row_number() OVER (
			         PARTITION BY source.tenant_id,source.legal_entity_id,source.definition_revision,
			                      (source.generated_at AT TIME ZONE 'UTC')::date
			         ORDER BY source.generated_at DESC,source.id DESC
			       ) AS source_rank
			FROM domain_metric_snapshots source
			WHERE source.definition_revision=$1
			  AND source.generated_at<
			      (date_trunc('day',$2::timestamptz AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')
		), latest AS (
			SELECT *
			FROM ranked
			WHERE source_rank=1
		)
		SELECT tenant.id::text,
		       entity.id::text,
		       latest.definition_revision,
		       latest.bucket_date,
		       latest.id::text,
		       latest.source_revision,
		       latest.generated_at,
		       latest.source_high_water
		FROM latest
		JOIN tenants tenant ON tenant.id=latest.tenant_id
		JOIN legal_entities entity
		  ON entity.tenant_id=latest.tenant_id
		 AND entity.id=latest.legal_entity_id
		WHERE NOT EXISTS (
			SELECT 1
			FROM organization_metric_daily_sources daily
			WHERE daily.tenant_id=latest.tenant_id
			  AND daily.legal_entity_id=latest.legal_entity_id
			  AND daily.definition_revision=latest.definition_revision
			  AND daily.bucket_date=latest.bucket_date
		)
		  AND NOT EXISTS (
			SELECT 1
			FROM domain_metric_snapshot_memberships member
			WHERE member.source_id=latest.id
			  AND member.metric_id=ANY($3::text[])
			  AND member.organization_scope_id IS NOT NULL
			  AND NOT EXISTS (
				SELECT 1
				FROM organization_scope_lineage_events lineage
				WHERE lineage.tenant_id=latest.tenant_id
				  AND lineage.legal_entity_id=latest.legal_entity_id
				  AND lineage.scope_id=member.organization_scope_id
				  AND lineage.effective_at<=latest.generated_at
			  )
		  )
		ORDER BY latest.bucket_date,latest.legal_entity_id
		LIMIT $4`,
		DomainDefinitionRevision,
		now,
		organizationDailyPostureMetricIDs,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("load pending organization daily metric sources: %w", err)
	}
	defer rows.Close()

	values := make([]organizationDailySourceCandidate, 0, limit)
	for rows.Next() {
		var value organizationDailySourceCandidate
		if err := rows.Scan(
			&value.TenantID,
			&value.LegalEntityID,
			&value.DefinitionRevision,
			&value.BucketDate,
			&value.SourceID,
			&value.SourceRevision,
			&value.SourceGeneratedAt,
			&value.SourceHighWater,
		); err != nil {
			return nil, fmt.Errorf("scan pending organization daily metric source: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending organization daily metric sources: %w", err)
	}
	return values, nil
}

func finalizeOrganizationMetricDay(
	ctx context.Context,
	tx pgx.Tx,
	source organizationDailySourceCandidate,
) (bool, error) {
	if tx == nil || source.TenantID == "" || source.LegalEntityID == "" ||
		source.DefinitionRevision != DomainDefinitionRevision || source.SourceID == "" ||
		source.SourceRevision == "" || source.BucketDate.IsZero() || source.SourceGeneratedAt.IsZero() {
		return false, ErrDomainMetricsInvalid
	}

	command, err := tx.Exec(ctx, `
		INSERT INTO organization_metric_daily_sources(
			tenant_id,legal_entity_id,definition_revision,bucket_date,
			source_id,source_revision,source_generated_at,source_high_water
		)
		VALUES($1::uuid,$2::uuid,$3,$4::date,$5::uuid,$6,$7,$8::jsonb)
		ON CONFLICT(tenant_id,legal_entity_id,definition_revision,bucket_date) DO NOTHING`,
		source.TenantID,
		source.LegalEntityID,
		source.DefinitionRevision,
		source.BucketDate,
		source.SourceID,
		source.SourceRevision,
		source.SourceGeneratedAt.UTC(),
		source.SourceHighWater,
	)
	if err != nil {
		return false, fmt.Errorf("store organization daily metric source: %w", err)
	}
	if command.RowsAffected() == 0 {
		return false, nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_metric_daily_buckets(
			tenant_id,legal_entity_id,definition_revision,bucket_date,
			metric_id,organization_scope_id,lineage_event_id,value
		)
		SELECT $1::uuid,
		       $2::uuid,
		       $3,
		       $4::date,
		       member.metric_id,
		       member.organization_scope_id,
		       lineage.id,
		       count(*)::bigint
		FROM domain_metric_snapshot_memberships member
		LEFT JOIN LATERAL (
			SELECT event.id
			FROM organization_scope_lineage_events event
			WHERE member.organization_scope_id IS NOT NULL
			  AND event.tenant_id=$1::uuid
			  AND event.legal_entity_id=$2::uuid
			  AND event.scope_id=member.organization_scope_id
			  AND event.effective_at<=$6
			ORDER BY event.effective_at DESC,event.id DESC
			LIMIT 1
		) lineage ON true
		WHERE member.source_id=$5::uuid
		  AND member.definition_revision=$3
		  AND member.metric_id=ANY($7::text[])
		GROUP BY member.metric_id,member.organization_scope_id,lineage.id`,
		source.TenantID,
		source.LegalEntityID,
		source.DefinitionRevision,
		source.BucketDate,
		source.SourceID,
		source.SourceGeneratedAt.UTC(),
		organizationDailyPostureMetricIDs,
	); err != nil {
		return false, fmt.Errorf("store organization daily metric buckets: %w", err)
	}

	return true, nil
}
