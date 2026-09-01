//go:build postgres

package evidence

import (
	"database/sql"
	"encoding/json"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const responseRevisionProjection = `r.id::text,r.tenant_id::text,r.legal_entity_id::text,r.distribution_id::text,r.workspace_id::text,r.submission_id::text,
	r.revision,COALESCE(r.supersedes_revision_id::text,''),r.achieved_assurance,r.signoff_summary,r.compliance_score,
	r.scored_weight_coverage,r.state,r.critical_field_results,r.scoring_policy_version,r.is_current,r.created_at,
	r.score_mode,r.score_direction,r.raw_score,r.adverse_score,r.concern_band,r.score_state,r.score_result,r.score_profile_checksum,r.score_calculated_at`

type responseRevisionScanner interface{ Scan(...any) error }

func scanPostgresResponseRevision(row responseRevisionScanner) (ResponseRevision, error) {
	return scanPostgresResponseRevisionWithExtra(row)
}

func scanPostgresResponseRevisionWithExtra(row responseRevisionScanner, extra ...any) (ResponseRevision, error) {
	var value ResponseRevision
	var signoffJSON, criticalJSON, scoreJSON []byte
	var complianceScore, rawScore, adverseScore sql.NullFloat64
	var mode, direction, band sql.NullString
	var scoreState string
	var scoreChecksum string
	var calculatedAt sql.NullTime
	targets := []any{
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.DistributionID, &value.WorkspaceID, &value.SubmissionID,
		&value.Revision, &value.SupersedesRevisionID, &value.AchievedAssurance, &signoffJSON, &complianceScore,
		&value.ScoredWeightCoverage, &value.State, &criticalJSON, &value.ScoringPolicyVersion, &value.Current, &value.CreatedAt,
		&mode, &direction, &rawScore, &adverseScore, &band, &scoreState, &scoreJSON, &scoreChecksum, &calculatedAt,
	}
	if err := row.Scan(append(targets, extra...)...); err != nil {
		return ResponseRevision{}, err
	}
	if err := json.Unmarshal(signoffJSON, &value.SignoffSummary); err != nil {
		return ResponseRevision{}, err
	}
	if err := json.Unmarshal(criticalJSON, &value.CriticalFieldResults); err != nil {
		return ResponseRevision{}, err
	}
	if complianceScore.Valid {
		value.ComplianceScore = &complianceScore.Float64
	}
	value.Score = &ResponseScoreResult{State: ResponseScoreState(scoreState), ProfileChecksum: scoreChecksum}
	if len(scoreJSON) > 0 {
		if err := json.Unmarshal(scoreJSON, value.Score); err != nil {
			return ResponseRevision{}, err
		}
	}
	value.Score.State = ResponseScoreState(scoreState)
	value.Score.ProfileChecksum = scoreChecksum
	if mode.Valid {
		value.Score.Mode = formcontract.ScoringMode(mode.String)
	}
	if direction.Valid {
		value.Score.Direction = formcontract.ScoreDirection(direction.String)
	}
	if rawScore.Valid {
		value.Score.RawScore = &rawScore.Float64
	}
	if adverseScore.Valid {
		value.Score.AdverseScore = &adverseScore.Float64
	}
	if band.Valid {
		value.Score.Band = formcontract.ConcernBand(band.String)
	}
	if calculatedAt.Valid {
		value.Score.CalculatedAt = calculatedAt.Time.UTC()
	}
	return value, nil
}
