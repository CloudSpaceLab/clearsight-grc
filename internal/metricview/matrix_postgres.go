//go:build postgres

package metricview

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MatrixRepository struct {
	pool *pgxpool.Pool
}

func NewMatrixRepository(pool *pgxpool.Pool) *MatrixRepository {
	return &MatrixRepository{pool: pool}
}

func (r *MatrixRepository) RiskAppetiteMatrix(ctx context.Context, tenantID, legalEntityID string, at time.Time) (Matrix, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return Matrix{}, ErrMatrixUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return Matrix{}, ErrMatrixInvalid
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	rows, err := r.pool.Query(ctx, `
		WITH selected AS (
			SELECT r.id,r.category,r.version,
			       CASE
			         WHEN la.id IS NULL OR la.risk_version<>r.version THEN 'UNKNOWN'
			         WHEN la.appetite_statement_id IS NULL THEN 'UNKNOWN'
			         WHEN ap.id IS NULL OR la.appetite_statement_id<>ap.id THEN 'UNKNOWN'
			         ELSE COALESCE(NULLIF(la.appetite_position,''),'UNKNOWN')
			       END state
			FROM risks r
			JOIN tenants t ON t.id=r.tenant_id
			JOIN legal_entities le ON le.tenant_id=r.tenant_id AND le.id=r.legal_entity_id
			LEFT JOIN LATERAL (
				SELECT a.*
				FROM risk_assessments a
				WHERE a.tenant_id=r.tenant_id
				  AND a.legal_entity_id=r.legal_entity_id
				  AND a.risk_id=r.id
				ORDER BY a.risk_version DESC,a.id DESC
				LIMIT 1
			) la ON true
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM (
					SELECT a.*
					FROM risk_appetite_statements a
					WHERE a.tenant_id=r.tenant_id
					  AND a.legal_entity_id=r.legal_entity_id
					  AND a.risk_id=r.id
					  AND a.effective_from<=$3
					ORDER BY a.version DESC,a.id DESC
					LIMIT 1
				) candidate
				WHERE candidate.status='ACTIVE'
				  AND (candidate.effective_until IS NULL OR $3<candidate.effective_until)
			) ap ON true
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND r.status='ACTIVE'
		)
		SELECT COALESCE(NULLIF(btrim(category),''),'Uncategorized'),state,count(*)
		FROM selected
		GROUP BY 1,2
		ORDER BY 1,2`, tenantID, legalEntityID, at.UTC())
	if err != nil {
		return Matrix{}, fmt.Errorf("load appetite matrix: %w", err)
	}
	defer rows.Close()
	buckets := []matrixBucket{}
	for rows.Next() {
		var bucket matrixBucket
		if err := rows.Scan(&bucket.Category, &bucket.State, &bucket.Count); err != nil {
			return Matrix{}, fmt.Errorf("scan appetite matrix: %w", err)
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		return Matrix{}, fmt.Errorf("iterate appetite matrix: %w", err)
	}
	return buildMatrix(MatrixRiskAppetite, RiskAppetiteMatrixRevision, legalEntityID, at, appetiteMatrixColumns, buckets), nil
}

func (r *MatrixRepository) AssuranceCoverageMatrix(ctx context.Context, tenantID, legalEntityID string, at time.Time) (Matrix, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return Matrix{}, ErrMatrixUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return Matrix{}, ErrMatrixInvalid
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	rows, err := r.pool.Query(ctx, `
		WITH selected_risks AS (
			SELECT r.id,r.tenant_id,r.legal_entity_id,r.category
			FROM risks r
			JOIN tenants t ON t.id=r.tenant_id
			JOIN legal_entities le ON le.tenant_id=r.tenant_id AND le.id=r.legal_entity_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND r.status='ACTIVE'
		), eligible_links AS (
			SELECT risk_link.risk_id,
			       implementation.tenant_id,
			       implementation.program_id,
			       implementation.id implementation_id,
			       implementation.status implementation_status
			FROM risk_control_links risk_link
			JOIN control_catalog_implementation_links catalog_link
			  ON catalog_link.id=risk_link.catalog_link_id
			 AND catalog_link.tenant_id=risk_link.tenant_id
			 AND catalog_link.legal_entity_id=risk_link.legal_entity_id
			JOIN control_definitions definition
			  ON definition.id=catalog_link.definition_id
			 AND definition.tenant_id=catalog_link.tenant_id
			 AND definition.status='ACTIVE'
			JOIN control_implementations implementation
			  ON implementation.id=catalog_link.implementation_id
			 AND implementation.tenant_id=catalog_link.tenant_id
			 AND implementation.program_id=catalog_link.program_id
			 AND implementation.status NOT IN ('INACTIVE','RETIRED')
			 AND implementation.effective_from<=$3
			 AND (implementation.effective_until IS NULL OR $3<implementation.effective_until)
			WHERE EXISTS (
				SELECT 1
				FROM selected_risks risk
				WHERE risk.id=risk_link.risk_id
				  AND risk.tenant_id=risk_link.tenant_id
				  AND risk.legal_entity_id=risk_link.legal_entity_id
			)
		), contract_facts AS (
			SELECT risk.id risk_id,
			       CASE
			         WHEN link.implementation_id IS NULL OR link.implementation_status<>'IMPLEMENTED' THEN 'UNKNOWN'
			         WHEN contract.id IS NULL THEN 'UNKNOWN'
			         WHEN assessment.id IS NULL THEN 'UNKNOWN'
			         WHEN assessment.valid_until IS NOT NULL AND NOT ($3<assessment.valid_until) THEN 'UNKNOWN'
			         WHEN assessment.conclusion IN ('UNSUPPORTED','CONTRADICTED') THEN 'FAILED'
			         WHEN assessment.conclusion='PARTIALLY_SUPPORTED' OR assessment.coverage<contract.minimum_coverage THEN 'PARTIAL'
			         WHEN assessment.conclusion='SUPPORTED' THEN 'SUPPORTED'
			         ELSE 'UNKNOWN'
			       END state
			FROM selected_risks risk
			LEFT JOIN eligible_links link ON link.risk_id=risk.id
			LEFT JOIN evidence_contracts contract
			  ON contract.tenant_id=link.tenant_id
			 AND contract.program_id=link.program_id
			 AND contract.control_implementation_id=link.implementation_id
			 AND contract.status='ACTIVE'
			LEFT JOIN LATERAL (
				SELECT assessment.*
				FROM evidence_assessments assessment
				WHERE assessment.tenant_id=contract.tenant_id
				  AND assessment.program_id=contract.program_id
				  AND assessment.contract_id=contract.id
				  AND assessment.assessed_at<=$3
				ORDER BY assessment.assessed_at DESC,assessment.id DESC
				LIMIT 1
			) assessment ON true
		), risk_state AS (
			SELECT risk.id,risk.category,
			       CASE
			         WHEN bool_or(fact.state='FAILED') THEN 'FAILED'
			         WHEN bool_or(fact.state='UNKNOWN') AND bool_or(fact.state IN ('SUPPORTED','PARTIAL')) THEN 'PARTIAL'
			         WHEN bool_or(fact.state='UNKNOWN') THEN 'UNKNOWN'
			         WHEN bool_or(fact.state='PARTIAL') THEN 'PARTIAL'
			         WHEN bool_or(fact.state='SUPPORTED') THEN 'SUPPORTED'
			         ELSE 'UNKNOWN'
			       END state
			FROM selected_risks risk
			LEFT JOIN contract_facts fact ON fact.risk_id=risk.id
			GROUP BY risk.id,risk.category
		)
		SELECT COALESCE(NULLIF(btrim(category),''),'Uncategorized'),state,count(*)
		FROM risk_state
		GROUP BY 1,2
		ORDER BY 1,2`, tenantID, legalEntityID, at.UTC())
	if err != nil {
		return Matrix{}, fmt.Errorf("load assurance matrix: %w", err)
	}
	defer rows.Close()
	buckets := []matrixBucket{}
	for rows.Next() {
		var bucket matrixBucket
		if err := rows.Scan(&bucket.Category, &bucket.State, &bucket.Count); err != nil {
			return Matrix{}, fmt.Errorf("scan assurance matrix: %w", err)
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		return Matrix{}, fmt.Errorf("iterate assurance matrix: %w", err)
	}
	return buildMatrix(MatrixAssuranceCoverage, AssuranceCoverageMatrixRevision, legalEntityID, at, assuranceMatrixColumns, buckets), nil
}

var _ MatrixReader = (*MatrixRepository)(nil)
