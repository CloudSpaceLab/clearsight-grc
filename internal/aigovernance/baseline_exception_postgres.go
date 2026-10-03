//go:build postgres

package aigovernance

import (
	"context"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func (r *PostgresRepository) ActiveGatewayBaselineExceptions(ctx context.Context, tenantID, baselineID string, baselineVersion int64, workloadRecordID, environment string, now time.Time) ([]aigateway.BaselineException, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+policyColumns+`
FROM automation_policies ap
JOIN tenants t ON t.id=ap.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1)
  AND ap.code LIKE $2
  AND ap.action_class=$3
  AND ap.status='ACTIVE'
  AND ap.rollout_mode='ENFORCE'
  AND (ap.effective_from IS NULL OR ap.effective_from<=$4)
  AND ap.effective_until>$4
ORDER BY ap.effective_until ASC,ap.version DESC
LIMIT 64`, tenantID, aigateway.GatewayBaselineExceptionCodeRoot+":%", aigateway.GatewayBaselineExceptionActionClass, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]aigateway.BaselineException, 0, 4)
	for rows.Next() {
		policy, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		projected, err := projectGatewayBaselineException(policy, now)
		if err != nil {
			return nil, err
		}
		if projected.TargetBaselineID != baselineID || projected.TargetBaselineVersion != baselineVersion || !containsFold(projected.WorkloadRecordIDs, workloadRecordID) || !containsFold(projected.Environments, environment) {
			continue
		}
		out = append(out, projected)
		if len(out) > 8 {
			return nil, fmt.Errorf("too many applicable gateway baseline exceptions")
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
