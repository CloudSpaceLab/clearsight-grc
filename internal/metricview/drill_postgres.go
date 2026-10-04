//go:build postgres

package metricview

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *ObservationRepository) ListExactDrill(ctx context.Context, input DrillQuery) (DrillPage, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return DrillPage{}, ErrDrillInvalid
	}
	query, cursor, err := normalizeDrillQuery(input)
	if err != nil {
		return DrillPage{}, err
	}
	var sourceUUID pgtype.UUID
	if err := sourceUUID.Scan(query.SourceID); err != nil || !sourceUUID.Valid {
		return DrillPage{}, ErrDrillInvalid
	}
	if cursor.MemberID != "" {
		var cursorUUID pgtype.UUID
		if err := cursorUUID.Scan(cursor.MemberID); err != nil || !cursorUUID.Valid {
			return DrillPage{}, ErrDrillInvalid
		}
	}

	var (
		generatedAt   time.Time
		expectedTotal int
		tenantID      string
		entityID      string
	)
	err = r.pool.QueryRow(ctx, `
		SELECT observation.generated_at,observation.value,
		       observation.tenant_id::text,observation.legal_entity_id::text
		FROM metric_observations observation
		JOIN tenants tenant ON tenant.id=observation.tenant_id
		JOIN legal_entities entity
		  ON entity.tenant_id=observation.tenant_id
		 AND entity.id=observation.legal_entity_id
		WHERE observation.source_kind=$1
		  AND observation.source_id=$2::uuid
		  AND observation.metric_id=$3
		  AND observation.definition_revision=$4
		  AND (tenant.id::text=$5 OR tenant.slug=$5)
		  AND (entity.id::text=$6 OR entity.code=$6)
		LIMIT 1`,
		ObservationSourceOversightSnapshot,
		query.SourceID,
		query.MetricID,
		HomeDefinitionRevision,
		query.TenantID,
		query.LegalEntityID,
	).Scan(&generatedAt, &expectedTotal, &tenantID, &entityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return DrillPage{}, ErrDrillNotFound
	}
	if err != nil {
		return DrillPage{}, fmt.Errorf("load exact metric observation: %w", err)
	}

	var retainedTotal int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM oversight_metric_members member
		WHERE member.tenant_id=$1::uuid
		  AND member.legal_entity_id=$2::uuid
		  AND member.snapshot_id=$3::uuid
		  AND member.metric_id=$4`,
		tenantID, entityID, query.SourceID, query.MetricID,
	).Scan(&retainedTotal); err != nil {
		return DrillPage{}, fmt.Errorf("count exact metric population: %w", err)
	}
	if retainedTotal != expectedTotal {
		return DrillPage{}, ErrDrillMismatch
	}

	rows, err := r.pool.Query(ctx, `
		SELECT member.metric_id,member.member_type,member.member_id::text,
		       member.subject_type,member.subject_id::text,
		       member.reference,member.title,member.state,member.priority,member.due_at
		FROM oversight_metric_members member
		WHERE member.tenant_id=$1::uuid
		  AND member.legal_entity_id=$2::uuid
		  AND member.snapshot_id=$3::uuid
		  AND member.metric_id=$4
		  AND (
		    $5=''
		    OR (member.member_type,member.member_id) > ($5,NULLIF($6,'')::uuid)
		  )
		ORDER BY member.member_type,member.member_id
		LIMIT $7`,
		tenantID, entityID, query.SourceID, query.MetricID,
		cursor.MemberType, cursor.MemberID, query.Limit+1,
	)
	if err != nil {
		return DrillPage{}, fmt.Errorf("load exact metric population: %w", err)
	}
	defer rows.Close()

	items := make([]oversight.MetricMemberItem, 0, query.Limit+1)
	for rows.Next() {
		var item oversight.MetricMemberItem
		if err := rows.Scan(
			&item.MetricID,
			&item.MemberType,
			&item.MemberID,
			&item.SubjectType,
			&item.SubjectID,
			&item.Reference,
			&item.Title,
			&item.State,
			&item.Priority,
			&item.DueAt,
		); err != nil {
			return DrillPage{}, fmt.Errorf("scan exact metric population: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return DrillPage{}, fmt.Errorf("iterate exact metric population: %w", err)
	}

	page := DrillPage{
		SourceID: query.SourceID, MetricID: query.MetricID,
		DefinitionRevision: HomeDefinitionRevision,
		GeneratedAt: generatedAt.UTC(), Total: expectedTotal,
		Items: items,
	}
	if len(page.Items) > query.Limit {
		last := page.Items[query.Limit-1]
		page.Items = page.Items[:query.Limit]
		page.NextCursor, err = encodeDrillCursor(drillCursor{
			MemberType: last.MemberType,
			MemberID:   last.MemberID,
		})
		if err != nil {
			return DrillPage{}, err
		}
	}
	return page, nil
}
