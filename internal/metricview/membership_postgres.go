//go:build postgres

package metricview

import (
	"context"
	"fmt"
	"strings"

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
	organizationScopeID string,
	sourceID string,
	metricID string,
	definitionRevision string,
	principalID string,
	cursor string,
	limit int,
) (MemberPage, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	organizationScopeID = strings.TrimSpace(organizationScopeID)
	sourceID = strings.TrimSpace(sourceID)
	metricID = strings.TrimSpace(metricID)
	definitionRevision = strings.TrimSpace(definitionRevision)
	principalID = strings.TrimSpace(principalID)
	cursor = strings.TrimSpace(cursor)
	if r == nil || r.pool == nil || tenantID == "" || legalEntityID == "" || sourceID == "" ||
		metricID == "" || definitionRevision == "" || principalID == "" || limit < 1 || limit > 100 {
		return MemberPage{}, ErrMetricMembershipInvalid
	}

	var count, sourceCount int
	if err := r.pool.QueryRow(ctx, `
		WITH membership_sets AS (
			SELECT membership_set.oversight_snapshot_id AS source_id,
			       membership_set.tenant_id,
			       membership_set.legal_entity_id,
			       NULL::uuid AS organization_scope_id,
			       membership_set.definition_revision
			FROM oversight_snapshot_metric_membership_sets membership_set
			UNION ALL
			SELECT membership_set.source_id,
			       membership_set.tenant_id,
			       membership_set.legal_entity_id,
			       membership_set.organization_scope_id,
			       membership_set.definition_revision
			FROM metric_runtime_membership_sets membership_set
			WHERE membership_set.expires_at>clock_timestamp()
		), members AS (
			SELECT member.oversight_snapshot_id AS source_id,
			       member.metric_id,
			       member.definition_revision,
			       member.member_id
			FROM oversight_snapshot_metric_memberships member
			UNION ALL
			SELECT member.source_id,
			       member.metric_id,
			       member.definition_revision,
			       member.member_id
			FROM metric_runtime_memberships member
		)
		SELECT count(member.member_id),count(DISTINCT membership_set.source_id)
		FROM membership_sets membership_set
		JOIN tenants tenant ON tenant.id=membership_set.tenant_id
		JOIN legal_entities entity
		  ON entity.tenant_id=membership_set.tenant_id
		 AND entity.id=membership_set.legal_entity_id
		JOIN metric_definitions definition
		  ON definition.metric_id=$5
		 AND definition.revision=membership_set.definition_revision
		LEFT JOIN members member
		  ON member.source_id=membership_set.source_id
		 AND member.definition_revision=membership_set.definition_revision
		 AND member.metric_id=$5
		WHERE membership_set.source_id=$4::uuid
		  AND membership_set.definition_revision=$6
		  AND membership_set.organization_scope_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid
		  AND (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)`,
		tenantID, legalEntityID, organizationScopeID, sourceID, metricID, definitionRevision,
	).Scan(&count, &sourceCount); err != nil {
		return MemberPage{}, fmt.Errorf("count metric snapshot membership: %w", err)
	}
	if sourceCount != 1 {
		return MemberPage{}, ErrMetricMembershipNotFound
	}

	rows, err := r.pool.Query(ctx, `
		WITH membership_sets AS (
			SELECT membership_set.oversight_snapshot_id AS source_id,
			       membership_set.tenant_id,
			       membership_set.legal_entity_id,
			       NULL::uuid AS organization_scope_id,
			       membership_set.definition_revision
			FROM oversight_snapshot_metric_membership_sets membership_set
			UNION ALL
			SELECT membership_set.source_id,
			       membership_set.tenant_id,
			       membership_set.legal_entity_id,
			       membership_set.organization_scope_id,
			       membership_set.definition_revision
			FROM metric_runtime_membership_sets membership_set
			WHERE membership_set.expires_at>clock_timestamp()
		), members AS (
			SELECT member.oversight_snapshot_id AS source_id,
			       member.metric_id,
			       member.definition_revision,
			       member.member_id,
			       member.target_type,
			       member.target_id,
			       member.target_title,
			       member.state
			FROM oversight_snapshot_metric_memberships member
			UNION ALL
			SELECT member.source_id,
			       member.metric_id,
			       member.definition_revision,
			       member.member_id,
			       member.target_type,
			       member.target_id,
			       member.target_title,
			       member.state
			FROM metric_runtime_memberships member
		)
		SELECT member.member_id::text,
		       member.target_type,
		       CASE WHEN visibility.allowed THEN member.target_id::text ELSE '' END,
		       CASE WHEN visibility.allowed THEN member.target_title ELSE 'Record access changed' END,
		       CASE WHEN visibility.allowed THEN member.state ELSE 'ACCESS_CHANGED' END,
		       visibility.allowed
		FROM membership_sets membership_set
		JOIN tenants tenant ON tenant.id=membership_set.tenant_id
		JOIN legal_entities entity
		  ON entity.tenant_id=membership_set.tenant_id
		 AND entity.id=membership_set.legal_entity_id
		JOIN members member
		  ON member.source_id=membership_set.source_id
		 AND member.definition_revision=membership_set.definition_revision
		LEFT JOIN matters matter
		  ON member.target_type='MATTER'
		 AND matter.tenant_id=membership_set.tenant_id
		 AND matter.legal_entity_id=membership_set.legal_entity_id
		 AND matter.id=member.target_id
		LEFT JOIN programs program
		  ON member.target_type='PROGRAM'
		 AND program.tenant_id=membership_set.tenant_id
		 AND program.legal_entity_id=membership_set.legal_entity_id
		 AND program.id=member.target_id
		CROSS JOIN LATERAL (
		  SELECT CASE member.target_type
		    WHEN 'MATTER' THEN matter.id IS NOT NULL AND (
		      CASE
		        WHEN NOT (matter.scope ? 'access') THEN true
		        WHEN jsonb_typeof(matter.scope->'access')<>'string' THEN false
		        WHEN upper(btrim(matter.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
		        WHEN upper(btrim(matter.scope->>'access'))='RESTRICTED' THEN
		          CASE
		            WHEN jsonb_typeof(matter.scope->'allowed_principal_ids')<>'array' THEN false
		            ELSE
		              NOT EXISTS (
		                SELECT 1
		                FROM jsonb_array_elements(matter.scope->'allowed_principal_ids') entry(value)
		                WHERE jsonb_typeof(entry.value)<>'string'
		              )
		              AND EXISTS (
		                SELECT 1
		                FROM jsonb_array_elements_text(matter.scope->'allowed_principal_ids') nonblank(value)
		                WHERE btrim(nonblank.value)<>''
		              )
		              AND EXISTS (
		                SELECT 1
		                FROM jsonb_array_elements_text(matter.scope->'allowed_principal_ids') allowed(value)
		                WHERE btrim(allowed.value)=$7
		              )
		          END
		        ELSE false
		      END
		      OR COALESCE(matter.owner_principal_id::text,'')=$7
		      OR EXISTS (
		        SELECT 1
		        FROM matter_actions action
		        WHERE action.tenant_id=matter.tenant_id
		          AND action.matter_id=matter.id
		          AND COALESCE(action.owner_principal_id::text,'')=$7
		      )
		    )
		    WHEN 'PROGRAM' THEN program.id IS NOT NULL AND (
		      CASE
		        WHEN NOT (program.scope ? 'access') THEN true
		        WHEN jsonb_typeof(program.scope->'access')<>'string' THEN false
		        WHEN upper(btrim(program.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
		        WHEN upper(btrim(program.scope->>'access'))='RESTRICTED' THEN
		          CASE
		            WHEN jsonb_typeof(program.scope->'allowed_principal_ids')<>'array' THEN false
		            ELSE
		              NOT EXISTS (
		                SELECT 1
		                FROM jsonb_array_elements(program.scope->'allowed_principal_ids') entry(value)
		                WHERE jsonb_typeof(entry.value)<>'string'
		              )
		              AND EXISTS (
		                SELECT 1
		                FROM jsonb_array_elements_text(program.scope->'allowed_principal_ids') nonblank(value)
		                WHERE btrim(nonblank.value)<>''
		              )
		              AND EXISTS (
		                SELECT 1
		                FROM jsonb_array_elements_text(program.scope->'allowed_principal_ids') allowed(value)
		                WHERE btrim(allowed.value)=$7
		              )
		          END
		        ELSE false
		      END
		    )
		    ELSE false
		  END AS allowed
		) visibility
		WHERE membership_set.source_id=$4::uuid
		  AND membership_set.definition_revision=$6
		  AND membership_set.organization_scope_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid
		  AND (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		  AND member.metric_id=$5
		  AND ($8='' OR member.member_id>$8::uuid)
		ORDER BY member.member_id
		LIMIT $9`,
		tenantID, legalEntityID, organizationScopeID, sourceID, metricID, definitionRevision, principalID, cursor, limit+1,
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
		if err := rows.Scan(&item.MemberID, &item.TargetType, &item.TargetID, &item.TargetTitle, &item.State, &item.Accessible); err != nil {
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
