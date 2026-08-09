//go:build postgres

package continuity

import "context"

func (r *PostgresRepository) ProgramEventsAfterVersion(ctx context.Context, tenant, programID string, afterVersion int64, limit int) ([]Event, bool, error) {
	if limit <= 0 {
		return []Event{}, false, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ce.id::text,t.slug,ce.aggregate_type,ce.aggregate_id::text,
		       ce.aggregate_version,ce.event_type,ce.payload,ce.actor_type,
		       COALESCE(ce.actor_id::text,''),ce.occurred_at
		FROM continuity_events ce
		JOIN tenants t ON t.id=ce.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND ce.aggregate_type='PROGRAM'
		  AND ce.aggregate_id=$2::uuid
		  AND ce.aggregate_version>$3
		ORDER BY ce.aggregate_version DESC
		LIMIT $4`, tenant, programID, afterVersion, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	values := make([]Event, 0, limit+1)
	for rows.Next() {
		var value Event
		if err := rows.Scan(
			&value.ID,
			&value.TenantID,
			&value.AggregateType,
			&value.AggregateID,
			&value.AggregateVersion,
			&value.Type,
			&value.Payload,
			&value.ActorType,
			&value.ActorID,
			&value.OccurredAt,
		); err != nil {
			return nil, false, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(values) > limit {
		return values[:limit], true, nil
	}
	return values, false, nil
}
