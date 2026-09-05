package formpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

var (
	policyCodePattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	templateVariablePattern = regexp.MustCompile(`\{\{\s*([a-z_]+)\s*\}\}`)
)

func normalizeCreateInput(input *CreateInput, now time.Time) error {
	if input == nil {
		return ErrInvalid
	}
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Purpose = strings.TrimSpace(input.Purpose)
	input.AutomationPolicyID = strings.TrimSpace(input.AutomationPolicyID)
	input.Eligibility.FormTemplateID = strings.TrimSpace(input.Eligibility.FormTemplateID)
	input.Action.Type = strings.ToUpper(strings.TrimSpace(input.Action.Type))
	input.Action.TitleTemplate = strings.TrimSpace(input.Action.TitleTemplate)
	input.Action.SummaryTemplate = strings.TrimSpace(input.Action.SummaryTemplate)
	input.Action.RequestedHandling = strings.TrimSpace(input.Action.RequestedHandling)
	input.Outcome.ExpectedOutcome = strings.TrimSpace(input.Outcome.ExpectedOutcome)
	input.Outcome.FailureResponse = strings.ToUpper(strings.TrimSpace(input.Outcome.FailureResponse))
	if !validSubjectTypes(input.Eligibility.SubjectTypes) || !validBands(input.Eligibility.Bands) {
		return fmt.Errorf("%w: subject type or concern band is invalid", ErrInvalid)
	}
	input.Eligibility.SubjectTypes = normalizedSubjectTypes(input.Eligibility.SubjectTypes)
	input.Eligibility.Bands = normalizedBands(input.Eligibility.Bands)
	if !policyCodePattern.MatchString(input.Code) || !bounded(input.Name, 1, 160) || !bounded(input.Purpose, 1, 1000) || !bounded(input.AutomationPolicyID, 1, 128) || input.AutomationPolicyVersion < 1 {
		return fmt.Errorf("%w: policy identity and automation policy revision are required", ErrInvalid)
	}
	if input.Eligibility.FormTemplateID == "" || input.Eligibility.FormTemplateVersion < 1 || len(input.Eligibility.SubjectTypes) == 0 || len(input.Eligibility.SubjectTypes) > 20 || input.Eligibility.MinimumCoverage < 0 || input.Eligibility.MinimumCoverage > 1 {
		return fmt.Errorf("%w: policy eligibility is invalid", ErrInvalid)
	}
	if len(input.Eligibility.Bands) == 0 && input.Eligibility.RawBelow == nil && input.Eligibility.RawAbove == nil && input.Eligibility.AdverseAtLeast == nil {
		return fmt.Errorf("%w: at least one score condition is required", ErrInvalid)
	}
	if !validScoreThreshold(input.Eligibility.RawBelow) || !validScoreThreshold(input.Eligibility.RawAbove) || !validScoreThreshold(input.Eligibility.AdverseAtLeast) || input.Eligibility.RawBelow != nil && input.Eligibility.RawAbove != nil && *input.Eligibility.RawAbove >= *input.Eligibility.RawBelow {
		return fmt.Errorf("%w: score threshold is invalid", ErrInvalid)
	}
	if !validMatterType(input.Action.Type) || input.Action.Priority < 1 || input.Action.Priority > 5 || !bounded(input.Action.TitleTemplate, 1, 200) || !bounded(input.Action.SummaryTemplate, 1, 2000) || !bounded(input.Action.RequestedHandling, 1, 2000) {
		return fmt.Errorf("%w: matter action is invalid", ErrInvalid)
	}
	if err := validateTemplate(input.Action.TitleTemplate); err != nil {
		return err
	}
	if err := validateTemplate(input.Action.SummaryTemplate); err != nil {
		return err
	}
	if input.BlastRadius.PerRun < 1 || input.BlastRadius.PerRun > 1000 || input.BlastRadius.PerDay < input.BlastRadius.PerRun || input.BlastRadius.PerDay > 10000 {
		return fmt.Errorf("%w: blast radius is invalid", ErrInvalid)
	}
	if !bounded(input.Outcome.ExpectedOutcome, 1, 2000) || input.Outcome.CheckAfterMinutes < 1 || input.Outcome.CheckAfterMinutes > 525600 || !slices.Contains([]string{"ESCALATE", "REOPEN", "CREATE_MATTER", "BLOCK_CLOSE"}, input.Outcome.FailureResponse) {
		return fmt.Errorf("%w: outcome check is invalid", ErrInvalid)
	}
	if input.Rollout != RolloutShadow && input.Rollout != RolloutEnforce {
		return fmt.Errorf("%w: rollout mode is invalid", ErrInvalid)
	}
	if input.EffectiveFrom != nil {
		v := input.EffectiveFrom.UTC()
		input.EffectiveFrom = &v
	}
	if input.EffectiveUntil != nil {
		v := input.EffectiveUntil.UTC()
		input.EffectiveUntil = &v
		if !v.After(now.UTC()) {
			return fmt.Errorf("%w: effective end must be in the future", ErrInvalid)
		}
	}
	if input.EffectiveFrom != nil && input.EffectiveUntil != nil && !input.EffectiveUntil.After(*input.EffectiveFrom) {
		return fmt.Errorf("%w: effective end must follow effective start", ErrInvalid)
	}
	return nil
}

func validActor(actor Actor) bool {
	return strings.TrimSpace(actor.TenantID) != "" && strings.TrimSpace(actor.LegalEntityID) != "" && strings.TrimSpace(actor.PrincipalID) != ""
}

func normalizeActor(actor Actor) Actor {
	return Actor{TenantID: strings.TrimSpace(actor.TenantID), LegalEntityID: strings.TrimSpace(actor.LegalEntityID), PrincipalID: strings.TrimSpace(actor.PrincipalID)}
}

func normalizedSubjectTypes(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" || len(value) > 80 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func validSubjectTypes(values []string) bool {
	if len(values) == 0 || len(values) > 20 {
		return false
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 80 {
			return false
		}
	}
	return true
}

func normalizedBands(values []formcontract.ConcernBand) []formcontract.ConcernBand {
	result := append([]formcontract.ConcernBand(nil), values...)
	slices.Sort(result)
	result = slices.Compact(result)
	for _, value := range result {
		if !slices.Contains([]formcontract.ConcernBand{formcontract.ConcernLow, formcontract.ConcernModerate, formcontract.ConcernHigh, formcontract.ConcernCritical}, value) {
			return nil
		}
	}
	return result
}

func validBands(values []formcontract.ConcernBand) bool {
	for _, value := range values {
		if !slices.Contains([]formcontract.ConcernBand{formcontract.ConcernLow, formcontract.ConcernModerate, formcontract.ConcernHigh, formcontract.ConcernCritical}, value) {
			return false
		}
	}
	return true
}

func validateTemplate(value string) error {
	allowed := map[string]bool{"form_title": true, "subject_type": true, "subject_id": true, "score": true, "concern": true}
	stripped := templateVariablePattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := templateVariablePattern.FindStringSubmatch(match)
		if len(parts) != 2 || !allowed[parts[1]] {
			return "{{invalid}}"
		}
		return ""
	})
	if strings.Contains(stripped, "{{") || strings.Contains(stripped, "}}") || strings.Contains(stripped, "{{invalid}}") {
		return fmt.Errorf("%w: template variable is not approved", ErrInvalid)
	}
	return nil
}

func validMatterType(value string) bool {
	return slices.Contains([]string{
		string(continuity.MatterRiskSituation), string(continuity.MatterControlGap), string(continuity.MatterAuditFinding),
		string(continuity.MatterException), string(continuity.MatterVendorReview), string(continuity.MatterVendorDeficiency),
		string(continuity.MatterFailedVerification), string(continuity.MatterEvidenceContradiction), string(continuity.MatterKRIBreach),
	}, value)
}

func validScoreThreshold(value *float64) bool { return value == nil || *value >= 0 && *value <= 100 }
func bounded(value string, minimum, maximum int) bool {
	return len(value) >= minimum && len(value) <= maximum
}

func policyChecksum(value Policy) string {
	payload, _ := json.Marshal(struct {
		ID                      string
		TenantID                string
		LegalEntityID           string
		Code                    string
		Name                    string
		Purpose                 string
		ActionClass             string
		AutomationPolicyID      string
		AutomationPolicyVersion int64
		Eligibility             Eligibility
		Action                  MatterAction
		BlastRadius             BlastRadius
		Outcome                 OutcomeContract
		Rollout                 RolloutMode
		EffectiveFrom           *time.Time
		EffectiveUntil          *time.Time
		Version                 int64
	}{value.ID, value.TenantID, value.LegalEntityID, value.Code, value.Name, value.Purpose, value.ActionClass, value.AutomationPolicyID, value.AutomationPolicyVersion, value.Eligibility, value.Action, value.BlastRadius, value.Outcome, value.Rollout, value.EffectiveFrom, value.EffectiveUntil, value.Version})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
