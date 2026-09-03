//go:build postgres

package aigovernance

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func (r *PostgresRepository) ActiveGatewayBaseline(ctx context.Context, tenantID string) (Policy, error) {
	return scanPolicy(r.pool.QueryRow(ctx, `SELECT `+policyColumns+`
FROM automation_policies ap
JOIN tenants t ON t.id=ap.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1)
  AND ap.code=$2
  AND ap.action_class=$3
  AND ap.status='ACTIVE'
  AND (ap.effective_from IS NULL OR ap.effective_from<=clock_timestamp())
  AND (ap.effective_until IS NULL OR ap.effective_until>clock_timestamp())
ORDER BY ap.version DESC
LIMIT 1`, tenantID, aigateway.GatewayBaselinePolicyCode, aigateway.GatewayBaselineActionClass))
}
