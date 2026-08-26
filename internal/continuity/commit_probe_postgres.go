//go:build postgres

package continuity

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) commitContinuityEvents(ctx context.Context, tx pgx.Tx, events ...Event) error {
	if err := tx.Commit(ctx); err != nil {
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		confirmed, probeErr := r.continuityEventsRecorded(probeCtx, events...)
		return exactCommitResult(err, confirmed, probeErr)
	}
	return nil
}

func (r *PostgresRepository) continuityEventsRecorded(ctx context.Context, events ...Event) (bool, error) {
	for _, event := range events {
		var confirmed bool
		err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM continuity_events ce JOIN tenants t ON t.id=ce.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ce.id::text=$2 AND ce.aggregate_type=$3 AND ce.aggregate_id::text=$4 AND ce.aggregate_version=$5 AND ce.event_type=$6 AND ce.payload=$7::jsonb AND ce.actor_type=$8 AND COALESCE(ce.actor_id::text,'')=$9 AND ce.occurred_at=$10 AND EXISTS(SELECT 1 FROM outbox_events o WHERE o.tenant_id=ce.tenant_id AND o.aggregate_type=ce.aggregate_type AND o.aggregate_id=ce.aggregate_id AND o.event_type=ce.event_type AND o.payload=ce.payload AND o.occurred_at=ce.occurred_at))`, event.TenantID, event.ID, event.AggregateType, event.AggregateID, event.AggregateVersion, event.Type, rawJSON(event.Payload, `{}`), event.ActorType, event.ActorID, event.OccurredAt).Scan(&confirmed)
		if err != nil || !confirmed {
			return false, err
		}
	}
	return len(events) > 0, nil
}
