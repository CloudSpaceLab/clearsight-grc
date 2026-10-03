//go:build postgres

package oploss

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) List(ctx context.Context, scope Scope, filter ListFilter) (Page, error) {
	if r == nil || r.pool == nil {
		return Page{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Page{}, err
	}
	cursor, err := decodeListCursor(filter.Cursor)
	if err != nil {
		return Page{}, err
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	var cursorID any
	if !cursor.UpdatedAt.IsZero() {
		cursorID = cursor.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT l.id::text,l.tenant_id::text,l.legal_entity_id::text,COALESCE(l.organization_scope_id::text,''),
		       l.code,l.title,l.event_type,l.cause,l.description,l.gross_amount_minor,l.currency,
		       l.occurred_at,l.discovered_at,COALESCE(l.risk_id::text,''),COALESCE(l.matter_id::text,''),
		       l.owner_principal_id::text,l.status,l.version,l.created_at,l.updated_at,
		       COALESCE(recovery.recovered_minor,0)::bigint,
		       CASE
		         WHEN COALESCE(recovery.recovered_minor,0)=0 THEN 'NONE'
		         WHEN COALESCE(recovery.recovered_minor,0)=l.gross_amount_minor THEN 'FULL'
		         ELSE 'PARTIAL'
		       END recovery_status
		FROM operational_losses l
		JOIN tenants t ON t.id=l.tenant_id
		JOIN legal_entities le ON le.tenant_id=l.tenant_id AND le.id=l.legal_entity_id
		LEFT JOIN LATERAL (
			SELECT sum(CASE r.kind WHEN 'RECOVERY' THEN r.amount_minor ELSE -r.amount_minor END)::bigint recovered_minor
			FROM operational_loss_recoveries r
			WHERE r.tenant_id=l.tenant_id AND r.legal_entity_id=l.legal_entity_id AND r.loss_id=l.id
		) recovery ON true
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND ($3::text='' OR l.status=$3::text)
		  AND ($4::text='' OR l.event_type=$4::text)
		  AND ($5::text='' OR l.currency=$5::text)
		  AND ($6::text='' OR l.organization_scope_id=NULLIF($6::text,'')::uuid)
		  AND ($7::text='' OR l.risk_id=NULLIF($7::text,'')::uuid)
		  AND ($8::text='' OR strpos(lower(concat_ws(' ',l.code,l.title,l.cause,l.description)),lower($8::text))>0)
		  AND ($9::text='' OR (
		        CASE
		          WHEN COALESCE(recovery.recovered_minor,0)=0 THEN 'NONE'
		          WHEN COALESCE(recovery.recovered_minor,0)=l.gross_amount_minor THEN 'FULL'
		          ELSE 'PARTIAL'
		        END
		      )=$9::text)
		  AND ($10::boolean=false OR l.updated_at<$11::timestamptz OR (l.updated_at=$11::timestamptz AND l.id<$12::uuid))
		ORDER BY l.updated_at DESC,l.id DESC
		LIMIT $13`,
		scope.TenantID, scope.LegalEntityID, string(filter.Status), string(filter.EventType), filter.Currency,
		filter.OrganizationScopeID, filter.RiskID, filter.Search, filter.RecoveryStatus,
		!cursor.UpdatedAt.IsZero(), cursor.UpdatedAt, cursorID, filter.Limit+1,
	)
	if err != nil {
		return Page{}, fmt.Errorf("list operational losses: %w", err)
	}
	defer rows.Close()

	values := make([]Summary, 0, filter.Limit+1)
	for rows.Next() {
		var (
			loss           Loss
			recoveredMinor int64
			recoveryStatus string
		)
		if err := rows.Scan(
			&loss.ID, &loss.TenantID, &loss.LegalEntityID, &loss.OrganizationScopeID,
			&loss.Code, &loss.Title, &loss.EventType, &loss.Cause, &loss.Description,
			&loss.GrossAmountMinor, &loss.Currency, &loss.OccurredAt, &loss.DiscoveredAt,
			&loss.RiskID, &loss.MatterID, &loss.OwnerPrincipalID, &loss.Status, &loss.Version,
			&loss.CreatedAt, &loss.UpdatedAt, &recoveredMinor, &recoveryStatus,
		); err != nil {
			return Page{}, err
		}
		if recoveredMinor < 0 || recoveredMinor > loss.GrossAmountMinor {
			return Page{}, ErrRecoveryLimit
		}
		values = append(values, Summary{
			Loss: loss,
			Totals: Totals{
				GrossAmountMinor: loss.GrossAmountMinor, RecoveredAmountMinor: recoveredMinor,
				NetLossMinor: loss.GrossAmountMinor - recoveredMinor, Currency: loss.Currency,
				RecoveryStatus: recoveryStatus,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	page := Page{Items: values}
	if len(values) > filter.Limit {
		page.Items = values[:filter.Limit]
		page.NextCursor, err = encodeListCursor(page.Items[len(page.Items)-1].Loss)
		if err != nil {
			return Page{}, err
		}
	}
	return page, nil
}
