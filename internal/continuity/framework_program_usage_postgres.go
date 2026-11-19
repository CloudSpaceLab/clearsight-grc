//go:build postgres

package continuity

import (
	"context"
	"fmt"
	"time"
)

func (r *PostgresRepository) FrameworkProgramUsage(ctx context.Context, tenant, entity string, at time.Time) ([]FrameworkProgramUsage, error) {
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(a.template_code,''),CASE WHEN COALESCE(a.template_code,'')='' THEN a.edition_id::text ELSE '' END,count(DISTINCT p.id)
 FROM program_framework_adoptions a
 JOIN programs p ON p.id=a.program_id AND p.tenant_id=a.tenant_id AND p.legal_entity_id=a.legal_entity_id
 JOIN tenants t ON t.id=p.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id=$2::uuid
 AND p.status='ACTIVE' AND p.effective_from<=$3 AND (p.effective_until IS NULL OR $3<p.effective_until)
 AND a.status='ACTIVE' AND a.effective_from<=$3 AND (a.effective_until IS NULL OR $3<a.effective_until)
 GROUP BY 1,2 ORDER BY 1,2 LIMIT 1001`, tenant, entity, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []FrameworkProgramUsage{}
	for rows.Next() {
		var value FrameworkProgramUsage
		if err = rows.Scan(&value.TemplateCode, &value.EditionID, &value.ActiveProgramCount); err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(items) > 1000 {
		return nil, fmt.Errorf("framework Program count limit exceeded")
	}
	return items, nil
}
