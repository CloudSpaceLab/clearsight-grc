package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func buildResponseRevision(request Request, assurance AccessAssurance, attestationFieldIDs []string, answers map[string]formcontract.AnswerValue) (ResponseRevision, error) {
	fieldsByID := make(map[string]Field, len(request.Fields))
	for _, field := range request.Fields {
		fieldsByID[field.ID] = field
	}
	for _, fieldID := range attestationFieldIDs {
		field, ok := fieldsByID[fieldID]
		if !ok || formcontract.Type(field.Type) != formcontract.TypeAttestation {
			return ResponseRevision{}, fmt.Errorf("%w: %q is not an attestation field", ErrWorkspaceUnavailable, fieldID)
		}
		if answer, ok := answers[fieldID]; !ok || !answer.Answered() {
			return ResponseRevision{}, fmt.Errorf("%w: attestation %q is not confirmed", ErrWorkspaceUnavailable, fieldID)
		}
	}

	revision := ResponseRevision{
		AchievedAssurance: assurance,
		SignoffSummary: map[string]any{
			"attestation_field_ids": append([]string(nil), attestationFieldIDs...),
			"attestation_count":     len(attestationFieldIDs),
		},
		State:                ResponseRevisionFinal,
		CriticalFieldResults: []map[string]any{},
		ScoringPolicyVersion: "formcontract-v1",
		Current:              true,
		Score:                &ResponseScoreResult{State: ResponseScoreNotConfigured},
	}
	contract, err := workspaceScoringContract(request)
	if err != nil {
		revision.Score = failedResponseScore(request, "SCORE_CONFIGURATION_INVALID")
		return revision, nil
	}
	if contract.ScoreProfile != nil {
		result, err := formcontract.EvaluateScoreProfile(*contract.ScoreProfile, contract, answers)
		if err != nil {
			revision.Score = failedResponseScore(request, "SCORE_EVALUATION_FAILED")
			return revision, nil
		}
		encoded, err := json.Marshal(contract.ScoreProfile)
		if err != nil {
			return ResponseRevision{}, err
		}
		digest := sha256.Sum256(encoded)
		state := ResponseScoreProvisional
		if result.Final {
			state = ResponseScoreFinal
		}
		revision.Score = &ResponseScoreResult{
			Mode: contract.ScoreProfile.Mode, Direction: contract.ScoreProfile.Direction,
			RawScore: result.RawScore, AdverseScore: result.AdverseScore, Band: result.Band,
			Coverage: result.Coverage, Final: result.Final, State: state,
			ProfileVersion: contract.ScoreProfile.Version, ProfileChecksum: hex.EncodeToString(digest[:]),
			EvaluatorVersion: "formcontract-advanced-v1", ContributionResults: result.ContributionResults, RuleResults: result.RuleResults,
		}
	} else if contract.ScoringMode == formcontract.ScoringCompliance {
		result, err := formcontract.ScoreCompliance(contract, answers)
		if err != nil {
			revision.Score = failedResponseScore(request, "SCORE_EVALUATION_FAILED")
			return revision, nil
		}
		revision.ComplianceScore = result.Score
		revision.ScoredWeightCoverage = result.Coverage * 100
		if !result.Final {
			revision.State = ResponseRevisionProvisional
		}
		revision.CriticalFieldResults = scoreRuleMaps(result.CriticalRules)
		revision.Score = legacyComplianceResponseScore(result)
	} else if contract.ScoringMode == formcontract.ScoringRisk {
		fields := make([]formcontract.Scoring, 0)
		for _, field := range contract.Fields {
			if field.Scoring != nil {
				fields = append(fields, *field.Scoring)
			}
		}
		result, err := formcontract.ScoreAnswers(fields, answers)
		if err != nil {
			revision.Score = failedResponseScore(request, "SCORE_EVALUATION_FAILED")
			return revision, nil
		}
		revision.Score = legacyRiskResponseScore(result)
	}
	return revision, nil
}

func failedResponseScore(request Request, code string) *ResponseScoreResult {
	result := &ResponseScoreResult{
		Mode: request.ScoringMode, State: ResponseScoreFailed, FailureCode: code,
		EvaluatorVersion: "formcontract-advanced-v1",
	}
	if request.ScoreProfile == nil {
		return result
	}
	result.Mode = request.ScoreProfile.Mode
	result.Direction = request.ScoreProfile.Direction
	result.ProfileVersion = request.ScoreProfile.Version
	if encoded, err := json.Marshal(request.ScoreProfile); err == nil {
		digest := sha256.Sum256(encoded)
		result.ProfileChecksum = hex.EncodeToString(digest[:])
	}
	return result
}

func workspaceScoringContract(request Request) (formcontract.Contract, error) {
	hasScoring := false
	hasWeightedSection := false
	for _, field := range request.Fields {
		hasScoring = hasScoring || field.Scoring != nil
	}
	for _, section := range request.Sections {
		hasWeightedSection = hasWeightedSection || section.Weight > 0
	}
	mode := request.ScoringMode
	switch {
	case request.ScoringMode != "":
	case hasWeightedSection:
		mode = formcontract.ScoringCompliance
	case hasScoring:
		mode = formcontract.ScoringRisk
	default:
		mode = formcontract.ScoringNone
	}
	return formContractWithScoring(request.Presentation, mode, request.ScoreProfile, request.Sections, request.Fields)
}

func legacyComplianceResponseScore(result formcontract.ComplianceResult) *ResponseScoreResult {
	state := ResponseScoreProvisional
	if result.Final {
		state = ResponseScoreFinal
	}
	var adverse *float64
	if result.Score != nil {
		value := 100 - *result.Score
		adverse = &value
	}
	return &ResponseScoreResult{Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, RawScore: result.Score, AdverseScore: adverse, Band: concernBandForAdverse(adverse), Coverage: result.Coverage, Final: result.Final, State: state, ProfileVersion: "formcontract-v1", EvaluatorVersion: "formcontract-v1"}
}

func legacyRiskResponseScore(result formcontract.ScoreResult) *ResponseScoreResult {
	state := ResponseScoreProvisional
	if result.Score != nil {
		state = ResponseScoreFinal
	}
	return &ResponseScoreResult{Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor, RawScore: result.Score, AdverseScore: result.Score, Band: concernBandForAdverse(result.Score), Coverage: result.Coverage, Final: result.Score != nil, State: state, ProfileVersion: "formcontract-v1", EvaluatorVersion: "formcontract-v1"}
}

func concernBandForAdverse(score *float64) formcontract.ConcernBand {
	if score == nil {
		return ""
	}
	switch {
	case *score <= 24:
		return formcontract.ConcernLow
	case *score <= 49:
		return formcontract.ConcernModerate
	case *score <= 74:
		return formcontract.ConcernHigh
	default:
		return formcontract.ConcernCritical
	}
}

func scoreRuleMaps(values []formcontract.ScoreRuleResult) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]any{
			"field_id": value.FieldID,
			"outcome":  value.Outcome,
			"points":   value.Points,
			"critical": value.Critical,
		})
	}
	return result
}
