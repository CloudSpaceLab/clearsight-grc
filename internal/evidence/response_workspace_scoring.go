package evidence

import (
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
		ScoringPolicyVersion: "formcontract-v1",
		Current:              true,
	}
	contract, err := workspaceScoringContract(request)
	if err != nil {
		return ResponseRevision{}, err
	}
	if contract.ScoringMode == formcontract.ScoringCompliance {
		result, err := formcontract.ScoreCompliance(contract, answers)
		if err != nil {
			return ResponseRevision{}, err
		}
		revision.ComplianceScore = result.Score
		revision.ScoredWeightCoverage = result.Coverage * 100
		if !result.Final {
			revision.State = ResponseRevisionProvisional
		}
		revision.CriticalFieldResults = scoreRuleMaps(result.CriticalRules)
	}
	return revision, nil
}

func workspaceScoringContract(request Request) (formcontract.Contract, error) {
	contract, err := formContract(request.Presentation, request.Sections, request.Fields)
	if err != nil {
		return formcontract.Contract{}, err
	}
	hasScoring := false
	hasWeightedSection := false
	for _, field := range contract.Fields {
		hasScoring = hasScoring || field.Scoring != nil
	}
	for _, section := range contract.Sections {
		hasWeightedSection = hasWeightedSection || section.Weight > 0
	}
	switch {
	case hasWeightedSection:
		contract.ScoringMode = formcontract.ScoringCompliance
	case hasScoring:
		contract.ScoringMode = formcontract.ScoringRisk
	default:
		contract.ScoringMode = formcontract.ScoringNone
	}
	return formcontract.Normalize(contract)
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
