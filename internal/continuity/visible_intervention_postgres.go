//go:build postgres

package continuity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) VisibleIntervention(ctx context.Context, tenant, entity, principal, trigger, program string) (VisibleIntervention, error) {
	var result VisibleIntervention
	var status string
	err := r.pool.QueryRow(ctx, `SELECT m.id::text,m.reference,m.status,m.priority,m.created_at
 FROM matters m JOIN tenants t ON t.id=m.tenant_id
 JOIN legal_entities le ON le.tenant_id=m.tenant_id AND le.id=m.legal_entity_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
 AND (SELECT count(*) FROM tenants matched_t WHERE matched_t.id::text=$1 OR matched_t.slug=$1)=1
 AND (SELECT count(*) FROM legal_entities matched_le WHERE matched_le.tenant_id=t.id AND (matched_le.id::text=$2 OR matched_le.code=$2) AND matched_le.valid_from<=clock_timestamp() AND (matched_le.valid_until IS NULL OR clock_timestamp()<matched_le.valid_until))=1
 AND EXISTS (SELECT 1 FROM principals p WHERE p.tenant_id=t.id AND p.id::text=$3 AND p.status='ACTIVE' AND p.valid_from<=clock_timestamp() AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until))
 AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
 AND m.trigger_key=$4 AND m.status NOT IN ('CLOSED','CANCELLED')
 AND EXISTS (SELECT 1 FROM matter_links ml WHERE ml.tenant_id=m.tenant_id AND ml.matter_id=m.id AND ml.program_id::text=$5 AND ml.retired_at IS NULL)
 AND NOT EXISTS (SELECT 1 FROM demo_record_archives archive WHERE archive.tenant_id=m.tenant_id AND archive.legal_entity_id=m.legal_entity_id AND archive.record_type='MATTER' AND archive.record_id=m.id AND archive.restored_at IS NULL)
 AND ((CASE WHEN jsonb_typeof(m.scope)<>'object' THEN false WHEN NOT (m.scope ? 'access') THEN true WHEN jsonb_typeof(m.scope->'access')<>'string' THEN false WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN CASE WHEN jsonb_typeof(m.scope->'allowed_principal_ids')<>'array' THEN false ELSE NOT EXISTS (SELECT 1 FROM jsonb_array_elements(m.scope->'allowed_principal_ids') e(value) WHERE jsonb_typeof(e.value)<>'string') AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') a(value) WHERE btrim(a.value)=$3 AND btrim(a.value)<>'') END ELSE false END)
 OR COALESCE(m.owner_principal_id::text,'')=$3 OR EXISTS (SELECT 1 FROM matter_actions ma WHERE ma.tenant_id=m.tenant_id AND ma.matter_id=m.id AND ma.owner_principal_id::text=$3))
 LIMIT 1`, tenant, entity, principal, trigger, program).Scan(&result.ID, &result.Reference, &status, &result.Priority, &result.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return VisibleIntervention{}, ErrNotFound
	}
	if err != nil {
		return VisibleIntervention{}, fmt.Errorf("read visible intervention: %w", err)
	}
	result.Status = MatterStatus(status)
	return result, nil
}
