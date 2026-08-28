//go:build postgres

package evidence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func (s *PostgresDistributionStore) ListDistributionResponseRevisions(ctx context.Context, tenantID, legalEntityID, distributionID string, limit int) ([]ResponseRevision, error) {
	if s == nil || s.repo == nil || s.repo.pool == nil {
		return nil, fmt.Errorf("postgres distribution repository is required")
	}
	rows, err := s.repo.pool.Query(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,distribution_id::text,workspace_id::text,submission_id::text,
		       revision,COALESCE(supersedes_revision_id::text,''),achieved_assurance,signoff_summary,compliance_score,
		       scored_weight_coverage,state,critical_field_results,scoring_policy_version,is_current,created_at
		FROM capture_response_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid
		ORDER BY revision DESC,id DESC
		LIMIT $4`, tenantID, legalEntityID, distributionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list distribution response revisions: %w", err)
	}
	defer rows.Close()
	values := make([]ResponseRevision, 0)
	for rows.Next() {
		var value ResponseRevision
		var signoffJSON, criticalJSON []byte
		var score sql.NullFloat64
		if err := rows.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.DistributionID, &value.WorkspaceID, &value.SubmissionID,
			&value.Revision, &value.SupersedesRevisionID, &value.AchievedAssurance, &signoffJSON, &score,
			&value.ScoredWeightCoverage, &value.State, &criticalJSON, &value.ScoringPolicyVersion, &value.Current, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan distribution response revision: %w", err)
		}
		if err := json.Unmarshal(signoffJSON, &value.SignoffSummary); err != nil {
			return nil, fmt.Errorf("decode response signoff summary: %w", err)
		}
		if err := json.Unmarshal(criticalJSON, &value.CriticalFieldResults); err != nil {
			return nil, fmt.Errorf("decode response critical results: %w", err)
		}
		if score.Valid {
			value.ComplianceScore = &score.Float64
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate distribution response revisions: %w", err)
	}
	return values, nil
}
