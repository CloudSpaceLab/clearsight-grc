//go:build postgres

package continuity

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// WriteImportedFindingTx joins the import receipt's transaction. The caller
// owns commit; canonical creation/action events also feed ordinary work workers.
func WriteImportedFindingTx(ctx context.Context, tx pgx.Tx, item ImportedFinding) error {
	m := item.Matter
	_, err := tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,created_at,updated_at,version,due_at)
 VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12::uuid,$13,$14,$15,$16,$17,$18::uuid,$19,$19,2,$20)`, m.ID, m.TenantID, m.LegalEntityID, m.Reference, m.Type, m.Status, m.Priority, m.Title, m.Summary, m.Scope, m.SourceType, m.SourceID, m.TriggerType, m.TriggerKey, m.KnownFacts, m.MissingFacts, m.Contradictions, m.OwnerPrincipalID, m.CreatedAt, m.DueAt)
	if err != nil {
		return err
	}
	for _, event := range item.Events {
		if event.Type == EventActionAdded {
			if err = applyMatterProjection(ctx, tx, event); err != nil {
				return err
			}
		}
		if err = insertContinuityEvent(ctx, tx, event); err != nil {
			return err
		}
		if err = insertOutbox(ctx, tx, event); err != nil {
			return err
		}
	}
	return nil
}
