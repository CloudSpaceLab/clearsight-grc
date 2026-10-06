//go:build postgres

package risk

import (
	"context"
	"fmt"
	"time"
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
	if filter.AsOf.IsZero() {
		filter.AsOf = time.Now().UTC()
	}
	var cursorID any
	if !cursor.UpdatedAt.IsZero() {
		cursorID = cursor.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
			r.id::text,r.tenant_id::text,r.legal_entity_id::text,COALESCE(r.organization_scope_id::text,''),r.code,r.name,r.category,r.statement,r.cause,
			r.event,r.impact,r.scope,COALESCE(r.owner_principal_id::text,''),r.status,r.version,r.created_at,r.updated_at,
			(la.id IS NOT NULL),COALESCE(la.id::text,''),COALESCE(la.risk_version,0),COALESCE(la.assessment_kind,''),COALESCE(la.method_code,''),COALESCE(la.method_version,''),
			COALESCE(la.dimensions,'{}'::jsonb),COALESCE(la.assumptions,'{}'::jsonb),COALESCE(la.evidence_references,'[]'::jsonb),
			(la.confidence IS NOT NULL),COALESCE(la.confidence,0),COALESCE(la.assessed_by::text,''),
			COALESCE(la.appetite_statement_id::text,''),COALESCE(la.appetite_position,''),COALESCE(la.appetite_rationale,''),
			COALESCE(la.assessed_at,'epoch'::timestamptz),COALESCE(la.created_at,'epoch'::timestamptz),
			(ap.id IS NOT NULL),COALESCE(ap.id::text,''),COALESCE(ap.risk_version,0),COALESCE(ap.version,0),COALESCE(ap.statement,''),
			COALESCE(ap.rule,'{}'::jsonb),COALESCE(ap.rationale,''),COALESCE(ap.owner_principal_id::text,''),COALESCE(ap.authority_principal_id::text,''),
			COALESCE(ap.status,''),COALESCE(ap.effective_from,'epoch'::timestamptz),(ap.effective_until IS NOT NULL),
			COALESCE(ap.effective_until,'epoch'::timestamptz),COALESCE(ap.created_at,'epoch'::timestamptz)
		FROM risks r
		JOIN tenants t ON t.id=r.tenant_id
		JOIN legal_entities le ON le.tenant_id=r.tenant_id AND le.id=r.legal_entity_id
		LEFT JOIN LATERAL (
			SELECT a.*
			FROM risk_assessments a
			WHERE a.tenant_id=r.tenant_id AND a.legal_entity_id=r.legal_entity_id AND a.risk_id=r.id
			  AND a.assessment_kind IN ('CURRENT','RESIDUAL')
			ORDER BY a.risk_version DESC,
			  CASE a.assessment_kind WHEN 'CURRENT' THEN 0 ELSE 1 END,
			  a.assessed_at DESC,a.created_at DESC,a.id DESC
			LIMIT 1
		) la ON true
		LEFT JOIN LATERAL (
			SELECT candidate.*
			FROM (
				SELECT a.*
				FROM risk_appetite_statements a
				WHERE a.tenant_id=r.tenant_id AND a.legal_entity_id=r.legal_entity_id AND a.risk_id=r.id
				  AND a.effective_from<=$8
				ORDER BY a.version DESC,a.id DESC
				LIMIT 1
			) candidate
			WHERE candidate.status='ACTIVE'
			  AND (candidate.effective_until IS NULL OR $8<candidate.effective_until)
		) ap ON true
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND ($3='' OR r.status=$3)
		  AND ($4='' OR lower(r.category)=lower($4))
		  AND ($5='' OR r.owner_principal_id::text=$5)
		  AND ($6='' OR strpos(lower(concat_ws(' ',r.code,r.name,r.category,r.statement,r.impact)),lower($6))>0)
		  AND ($7='' OR (
		        CASE
		          WHEN la.risk_version<>r.version OR la.id IS NULL THEN 'UNKNOWN'
		          WHEN la.appetite_statement_id IS NULL THEN 'UNKNOWN'
		          WHEN ap.id IS NULL OR la.appetite_statement_id<>ap.id THEN 'UNKNOWN'
		          ELSE COALESCE(la.appetite_position,'UNKNOWN')
		        END
		      )=$7)
		  AND ($9::boolean=false OR r.updated_at<$10 OR (r.updated_at=$10 AND r.id<$11::uuid))
		  AND (NOT $12::boolean OR r.organization_scope_id=ANY($13::uuid[]))
		ORDER BY r.updated_at DESC,r.id DESC
		LIMIT $14`,
		scope.TenantID, scope.LegalEntityID, string(filter.Status), filter.Category, filter.OwnerPrincipalID,
		filter.Search, string(filter.AppetitePosition), filter.AsOf.UTC(), !cursor.UpdatedAt.IsZero(),
		cursor.UpdatedAt, cursorID, filter.OrganizationScopeID != "", filter.OrganizationScopeIDs, filter.Limit+1,
	)
	if err != nil {
		return Page{}, fmt.Errorf("list risks: %w", err)
	}
	defer rows.Close()

	summaries := make([]Summary, 0, filter.Limit+1)
	for rows.Next() {
		summary, err := scanRiskSummary(rows)
		if err != nil {
			return Page{}, err
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	page := Page{Items: summaries, OrganizationScopeID: filter.OrganizationScopeID}
	if len(summaries) > filter.Limit {
		page.Items = summaries[:filter.Limit]
		last := page.Items[len(page.Items)-1].Risk
		page.NextCursor, err = encodeListCursor(last)
		if err != nil {
			return Page{}, err
		}
	}
	return page, nil
}

type riskSummaryScanner interface{ Scan(...any) error }

func scanRiskSummary(row riskSummaryScanner) (Summary, error) {
	var (
		risk Risk

		hasAssessment bool
		assessment    Assessment
		hasConfidence bool
		confidence    float64

		hasAppetite       bool
		appetite          AppetiteStatement
		hasEffectiveUntil bool
		effectiveUntil    time.Time
	)

	err := row.Scan(
		&risk.ID, &risk.TenantID, &risk.LegalEntityID, &risk.OrganizationScopeID, &risk.Code, &risk.Name, &risk.Category, &risk.Statement, &risk.Cause,
		&risk.Event, &risk.Impact, &risk.Scope, &risk.OwnerPrincipalID, &risk.Status, &risk.Version, &risk.CreatedAt, &risk.UpdatedAt,
		&hasAssessment, &assessment.ID, &assessment.RiskVersion, &assessment.Kind, &assessment.MethodCode, &assessment.MethodVersion,
		&assessment.Dimensions, &assessment.Assumptions, &assessment.EvidenceReferences, &hasConfidence, &confidence, &assessment.AssessedBy,
		&assessment.AppetiteStatementID, &assessment.AppetitePosition, &assessment.AppetiteRationale, &assessment.AssessedAt, &assessment.CreatedAt,
		&hasAppetite, &appetite.ID, &appetite.RiskVersion, &appetite.Version, &appetite.Statement, &appetite.Rule, &appetite.Rationale,
		&appetite.OwnerPrincipalID, &appetite.AuthorityPrincipalID, &appetite.Status, &appetite.EffectiveFrom,
		&hasEffectiveUntil, &effectiveUntil, &appetite.CreatedAt,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("scan risk summary: %w", err)
	}

	summary := Summary{Risk: risk}
	if hasAssessment {
		assessment.RiskID = risk.ID
		assessment.Dimensions = cloneJSON(assessment.Dimensions)
		assessment.Assumptions = cloneJSON(assessment.Assumptions)
		assessment.EvidenceReferences = cloneJSON(assessment.EvidenceReferences)
		if hasConfidence {
			value := confidence
			assessment.Confidence = &value
		}
		summary.LatestAssessment = &assessment
	}
	if hasAppetite {
		appetite.RiskID = risk.ID
		appetite.Rule = cloneJSON(appetite.Rule)
		if hasEffectiveUntil {
			value := effectiveUntil.UTC()
			appetite.EffectiveUntil = &value
		}
		summary.ActiveAppetite = &appetite
	}
	return summary, nil
}

func cloneJSON(value []byte) []byte {
	return append([]byte(nil), value...)
}
