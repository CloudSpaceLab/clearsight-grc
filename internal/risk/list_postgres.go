//go:build postgres

package risk

import (
	"context"
	"fmt"
	"strings"
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
			r.id::text,r.tenant_id::text,r.legal_entity_id::text,r.code,r.name,r.category,r.statement,r.cause,
			r.event,r.impact,r.scope,COALESCE(r.owner_principal_id::text,''),r.status,r.version,r.created_at,r.updated_at,
			la.id::text,la.risk_version,la.assessment_kind,la.method_code,la.method_version,
			la.dimensions,la.assumptions,la.evidence_references,la.confidence,COALESCE(la.assessed_by::text,''),
			COALESCE(la.appetite_statement_id::text,''),la.appetite_position,la.appetite_rationale,la.assessed_at,la.created_at,
			ap.id::text,ap.risk_version,ap.version,ap.statement,ap.rule,ap.rationale,
			COALESCE(ap.owner_principal_id::text,''),COALESCE(ap.authority_principal_id::text,''),
			ap.status,ap.effective_from,ap.effective_until,ap.created_at
		FROM risks r
		JOIN tenants t ON t.id=r.tenant_id
		JOIN legal_entities le ON le.tenant_id=r.tenant_id AND le.id=r.legal_entity_id
		LEFT JOIN LATERAL (
			SELECT a.*
			FROM risk_assessments a
			WHERE a.tenant_id=r.tenant_id AND a.legal_entity_id=r.legal_entity_id AND a.risk_id=r.id
			ORDER BY a.risk_version DESC,a.id DESC
			LIMIT 1
		) la ON true
		LEFT JOIN LATERAL (
			SELECT a.*
			FROM risk_appetite_statements a
			WHERE a.tenant_id=r.tenant_id AND a.legal_entity_id=r.legal_entity_id AND a.risk_id=r.id
			  AND a.status='ACTIVE'
			  AND a.effective_from<=$8
			  AND (a.effective_until IS NULL OR $8<a.effective_until)
			ORDER BY a.version DESC,a.id DESC
			LIMIT 1
		) ap ON true
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND ($3='' OR r.status=$3)
		  AND ($4='' OR lower(r.category)=lower($4))
		  AND ($5='' OR r.owner_principal_id::text=$5)
		  AND ($6='' OR strpos(lower(concat_ws(' ',r.code,r.name,r.category,r.statement,r.impact)),lower($6))>0)
		  AND ($7='' OR COALESCE(la.appetite_position,'')=$7)
		  AND ($9::boolean=false OR r.updated_at<$10 OR (r.updated_at=$10 AND r.id<$11::uuid))
		ORDER BY r.updated_at DESC,r.id DESC
		LIMIT $12`,
		scope.TenantID, scope.LegalEntityID, string(filter.Status), filter.Category, filter.OwnerPrincipalID,
		filter.Search, string(filter.AppetitePosition), filter.AsOf.UTC(), !cursor.UpdatedAt.IsZero(),
		cursor.UpdatedAt, cursorID, filter.Limit+1,
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
	page := Page{Items: summaries}
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

		assessmentID          *string
		assessmentRiskVersion *int64
		assessmentKind        *string
		methodCode            *string
		methodVersion         *string
		dimensions            []byte
		assumptions           []byte
		evidenceRefs          []byte
		confidence            *float64
		assessedBy            *string
		appetiteStatementID   *string
		appetitePosition      *string
		appetiteRationale     *string
		assessedAt            *time.Time
		assessmentCreatedAt   *time.Time

		appetiteID          *string
		appetiteRiskVersion *int64
		appetiteVersion     *int64
		appetiteStatement   *string
		appetiteRule        []byte
		appetiteRationaleV  *string
		appetiteOwner       *string
		appetiteAuthority   *string
		appetiteStatus      *string
		effectiveFrom       *time.Time
		effectiveUntil      *time.Time
		appetiteCreatedAt   *time.Time
	)

	err := row.Scan(
		&risk.ID,&risk.TenantID,&risk.LegalEntityID,&risk.Code,&risk.Name,&risk.Category,&risk.Statement,&risk.Cause,
		&risk.Event,&risk.Impact,&risk.Scope,&risk.OwnerPrincipalID,&risk.Status,&risk.Version,&risk.CreatedAt,&risk.UpdatedAt,
		&assessmentID,&assessmentRiskVersion,&assessmentKind,&methodCode,&methodVersion,
		&dimensions,&assumptions,&evidenceRefs,&confidence,&assessedBy,&appetiteStatementID,
		&appetitePosition,&appetiteRationale,&assessedAt,&assessmentCreatedAt,
		&appetiteID,&appetiteRiskVersion,&appetiteVersion,&appetiteStatement,&appetiteRule,&appetiteRationaleV,
		&appetiteOwner,&appetiteAuthority,&appetiteStatus,&effectiveFrom,&effectiveUntil,&appetiteCreatedAt,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("scan risk summary: %w", err)
	}

	summary := Summary{Risk: risk}
	if assessmentID != nil {
		value := Assessment{
			ID: *assessmentID, RiskID: risk.ID, RiskVersion: derefInt64(assessmentRiskVersion),
			Kind: AssessmentKind(derefString(assessmentKind)), MethodCode: derefString(methodCode), MethodVersion: derefString(methodVersion),
			Dimensions: cloneJSON(dimensions), Assumptions: cloneJSON(assumptions), EvidenceReferences: cloneJSON(evidenceRefs),
			Confidence: confidence, AssessedBy: derefString(assessedBy), AppetiteStatementID: derefString(appetiteStatementID),
			AppetitePosition: AppetitePosition(derefString(appetitePosition)), AppetiteRationale: derefString(appetiteRationale),
			AssessedAt: derefTime(assessedAt), CreatedAt: derefTime(assessmentCreatedAt),
		}
		summary.LatestAssessment = &value
	}
	if appetiteID != nil {
		value := AppetiteStatement{
			ID: *appetiteID, RiskID: risk.ID, RiskVersion: derefInt64(appetiteRiskVersion), Version: derefInt64(appetiteVersion),
			Statement: derefString(appetiteStatement), Rule: cloneJSON(appetiteRule), Rationale: derefString(appetiteRationaleV),
			OwnerPrincipalID: derefString(appetiteOwner), AuthorityPrincipalID: derefString(appetiteAuthority),
			Status: AppetiteStatus(derefString(appetiteStatus)), EffectiveFrom: derefTime(effectiveFrom),
			EffectiveUntil: effectiveUntil, CreatedAt: derefTime(appetiteCreatedAt),
		}
		summary.ActiveAppetite = &value
	}
	return summary, nil
}

func cloneJSON(value []byte) []byte {
	return append([]byte(nil), value...)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}
