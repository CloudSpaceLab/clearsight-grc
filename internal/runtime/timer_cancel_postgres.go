//go:build postgres

package runtime

import (
	"context"
	"fmt"
	"strings"
)

func (r *PostgresRepository) CancelPendingTaskTimers(ctx context.Context, tenant, taskID, timerType string) (int, error) {
	tenant = strings.TrimSpace(tenant)
	taskID = strings.TrimSpace(taskID)
	timerType = strings.TrimSpace(timerType)
	if tenant == "" || taskID == "" || timerType == "" {
		return 0, fmt.Errorf("tenant, task_id and timer_type are required")
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE workflow_timers wt
		SET state='CANCELLED',locked_by=NULL,lease_until=NULL,last_error=NULL
		WHERE wt.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND wt.task_id=$2::uuid
		  AND wt.timer_type=$3
		  AND wt.state='READY'`, tenant, taskID, timerType)
	if err != nil {
		return 0, fmt.Errorf("cancel pending task timers: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
