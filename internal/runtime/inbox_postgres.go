//go:build postgres

package runtime

import "context"

func (r *PostgresRepository) InboxProcessed(ctx context.Context, tenant, consumer, eventID string) (bool, error) {
	var processed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM inbox_receipts ir
			JOIN tenants t ON t.id=ir.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1) AND ir.consumer=$2 AND ir.event_id=$3
		)`, tenant, consumer, eventID).Scan(&processed)
	return processed, err
}
