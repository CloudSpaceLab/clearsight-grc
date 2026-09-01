//go:build postgres

package evidence

import (
	"database/sql"
	"encoding/json"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const responseRevisionProjection = `id::text,tenant_id::text,legal_entity_id::text,distribution_id::text,workspace_id::text,submission_id::text,
	revision,COALESCE(supersedes_revision_id::text,''),achieved_assurance,signoff_summary,compliance_score,
	scored_weight_coverage,state,critical_field_results,scoring_policy_version,is_current,created_at,
	score_mode,score_direction,raw_score,adverse_score,concern_band,score_state,score_result,score_profile_checksum,score_calculated_at`

type responseRevisionScanner interface{ Scan(...any) error }

func scanPostgresResponseRevision(row responseRevisionScanner) (ResponseRevision, error) {
	var value ResponseRevision
	var signoffJSON, criticalJSON, scoreJSON []byte
	var complianceScore, rawScore, adverseScore sql.NullFloat64
	var mode, direction, band sql.NullString
	var scoreState string
	var scoreChecksum string
	var calculatedAt sql.NullTime
	if err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.DistributionID, &value.WorkspaceID, &value.SubmissionID,
		&value.Revision, &value.SupersedesRevisionID, &value.AchievedAssurance, &signoffJSON, &complianceScore,
		&value.ScoredWeightCoverage, &value.State, &criticalJSON, &value.ScoringPolicyVersion, &value.Current, &value.CreatedAt,
		&mode, &direction, &rawScore, &adverseScore, &band, &scoreState, &scoreJSON, &scoreChecksum, &calculatedAt,
	); err != nil {
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
