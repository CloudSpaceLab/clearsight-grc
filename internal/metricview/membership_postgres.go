//go:build postgres

package metricview

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MembershipRepository struct {
	pool *pgxpool.Pool
}

func NewMembershipRepository(pool *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{pool: pool}
}

func (r *MembershipRepository) ListSnapshotMembers(
	ctx context.Context,
	tenantID string,
	legalEntityID string,
	sourceID string,
	metricID string,
	definitionRevision string,
	cursor string,
	limit int,
) (MemberPage, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	sourceID = strings.TrimSpace(sourceID)
	metricID = strings.TrimSpace(metricID)
	definitionRevision = strings.TrimSpace(definitionRevision)
	cursor = strings.TrimSpace(cursor)
	if r == nil || r.pool == nil || tenantID == "" || legalEntityID == "" || sourceID == "" ||
		metricID == "" || definitionRevision == "" || limit < 1 || limit > 100 {
		return MemberPage{}, ErrMetricMembershipInvalid
	}

	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM oversight_snapshots source
		JOIN tenants tenant ON tenant.id=source.tenant_id
		JOIN legal_entities entity ON entity.tenant_id=source.tenant_id AND entity.id=source.legal_entity_id
		JOIN oversight_snapshot_metric_memberships member ON member.oversight_snapshot_id=source.id
		WHERE source.id=$3::uuid
		  AND (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		  AND member.metric_id=$4
		  AND member.definition_revision=$5`,
		tenantID, legalEntityID, sourceID, metricID, definitionRevision,
	).Scan(&count)
	if err != nil {
		return MemberPage{}, fmt.Errorf("count metric snapshot membership: %w", err)
	}

	var sourceExists bool
	if count == 0 {
		err = r.pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM oversight_snapshots source
				JOIN tenants tenant ON tenant.id=source.tenant_id
				JOIN legal_entities entity ON entity.tenant_id=source.tenant_id AND entity.id=source.legal_entity_id
				JOIN metric_definitions definition ON definition.metric_id=$4 AND definition.revision=$5
				WHERE source.id=$3::uuid
				  AND (tenant.id::text=$1 OR tenant.slug=$1)
				  AND (entity.id::text=$2 OR entity.code=$2)
			)`,
			tenantID, legalEntityID, sourceID, metricID, definitionRevision,
		).Scan(&sourceExists)
		if err != nil {
			return MemberPage{}, fmt.Errorf("resolve metric snapshot membership source: %w", err)
		}
		if !sourceExists {
			return MemberPage{}, ErrMetricMembershipNotFound
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT member.member_id::text,member.target_type,member.target_id::text,member.target_title,member.state
		FROM oversight_snapshot_metric_memberships member
		JOIN oversight_snapshots source ON source.id=member.oversight_snapshot_id
		JOIN tenants tenant ON tenant.id=source.tenant_id
		JOIN legal_entities entity ON entity.tenant_id=source.tenant_id AND entity.id=source.legal_entity_id
		WHERE source.id=$3::uuid
		  AND (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		  AND member.metric_id=$4
		  AND member.definition_revision=$5
		  AND ($6='' OR member.member_id>$6::uuid)
		ORDER BY member.member_id
		LIMIT $7`,
		tenantID, legalEntityID, sourceID, metricID, definitionRevision, cursor, limit+1,
	)
	if err != nil {
		return MemberPage{}, fmt.Errorf("list metric snapshot membership: %w", err)
	}
	defer rows.Close()

	page := MemberPage{
		SourceID: sourceID, MetricID: metricID, DefinitionRevision: definitionRevision,
		Count: count, Items: make([]Member, 0, limit),
	}
	for rows.Next() {
		var item Member
		if err := rows.Scan(&item.MemberID, &item.TargetType, &item.TargetID, &item.TargetTitle, &item.State); err != nil {
			return MemberPage{}, fmt.Errorf("scan metric snapshot membership: %w", err)
		}
		if len(page.Items) == limit {
			page.NextCursor = page.Items[len(page.Items)-1].MemberID
			break
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return MemberPage{}, fmt.Errorf("iterate metric snapshot membership: %w", err)
	}
	return page, nil
}

var _ MembershipReader = (*MembershipRepository)(nil)

