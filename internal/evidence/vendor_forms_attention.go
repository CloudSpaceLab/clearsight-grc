package evidence

import (
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

// Document facts are read with the existing response population; they never
// expose artifacts, answer values or protected review rationale to the browser.
type vendorDocumentFact struct {
	ExpiresOn string `json:"expires_on"`
	Source    string `json:"source"`
	Rejected  bool   `json:"rejected"`
	Expired   bool   `json:"expired"`
	Current   bool   `json:"current"`
}

func populateVendorFormAttention(req Request, answers map[string]formcontract.AnswerValue, known bool, row *VendorFormRow, facts map[string]vendorDocumentFact, now time.Time) {
	row.AttentionItems = []VendorFormAttention{}
	row.Outdated = nil
	add := func(field, rule, label, state, source string) {
		for _, item := range row.AttentionItems {
			if item.FieldID == field && item.RuleID == rule && item.State == state && item.Source == source {
				return
			}
		}
		row.AttentionItems = append(row.AttentionItems, VendorFormAttention{field, rule, label, state, source})
	}
	for _, field := range row.MissingFields {
		add(field.ID, "", field.Label, "MISSING", "RESPONSE")
	}
	if !known || row.ResponseState != "SUBMITTED" {
		return
	}
	if row.ResponseCurrency == "PARTIALLY_REPLACED" {
		value := true
		row.Outdated = &value
	}
	contract, err := requestAnswerContract(req)
	if err != nil {
		return
	}
	visible, err := formcontract.VisibleFields(contract, answers)
	if err != nil {
		return
	}
	fields := map[string]formcontract.Field{}
	for _, field := range visible {
		fields[field.ID] = field
	}
	contributionField := func(id string) string {
		if strings.HasPrefix(id, "assessment:") {
			return strings.TrimPrefix(id, "assessment:")
		}
		if req.ScoreProfile == nil {
			return id
		}
		for _, c := range req.ScoreProfile.Contributions {
			if c.ID == id {
				return c.Predicate.FieldID
			}
		}
		return ""
	}
	reviewed := map[string]bool{}
	if row.AssessedScore != nil && (row.AssessedScore.State == ResponseScoreFinal || row.AssessedScore.State == ResponseScoreProvisional) {
		for _, result := range row.AssessedScore.ContributionResults {
			fieldID := contributionField(result.ID)
			if result.Outcome != formcontract.ScoreIndeterminate && row.reviewedFields[fieldID] && fields[fieldID].Assessment.NeedsReview() {
				reviewed[result.ID] = true
			}
		}
	}
	sourceFields := map[string]Field{}
	for _, field := range req.Fields {
		sourceFields[field.ID] = field
	}
	knownDates, unknownDates, expired := 0, false, false
	for _, field := range visible {
		fields[field.ID] = field
		sourceField := sourceFields[field.ID]
		if field.Type == formcontract.TypeVendorDocument {
			fact, hasFact := facts[field.ID]
			if !hasFact && answers[field.ID].Document != nil {
				fact = vendorDocumentFact{ExpiresOn: answers[field.ID].Document.ExpiresOn, Source: "RESPONSE", Current: true}
				hasFact = true
			}
			if !hasFact && CollectionFieldFulfilled(sourceField, now) {
				r := sourceField.CollectionResolution
				fact = vendorDocumentFact{ExpiresOn: vendorHeldExpiry(r), Source: vendorHeldSource(r), Current: true}
				hasFact = true
			} else if !hasFact && sourceField.CollectionResolution != nil {
				r := sourceField.CollectionResolution
				fact = vendorDocumentFact{ExpiresOn: vendorHeldExpiry(r), Source: vendorHeldSource(r), Current: r.Source.Current && ArtifactUseAllowed(r.Source.ArtifactStatus, r.Source.DemoUnscannedAllowed), Rejected: r.BankReviewState == "REJECTED"}
				if r.Source.Review != nil {
					fact.Rejected = fact.Rejected || r.Source.Review.Status == "REJECTED"
					fact.Expired = r.Source.Review.Status == "EXPIRED"
				}
				hasFact = true
			}
			if hasFact && fact.Current {
				if fact.Expired {
					expired = true
					add(field.ID, "", field.Label, "EXPIRED", fact.Source)
				}
				if fact.Rejected {
					add(field.ID, "", field.Label, "GAP", fact.Source)
				}
				date, err := time.Parse("2006-01-02", fact.ExpiresOn)
				if err != nil {
					unknownDates = true
				} else {
					knownDates++
					if now.UTC().Format("2006-01-02") > date.Format("2006-01-02") {
						expired = true
						add(field.ID, "", field.Label, "EXPIRED", fact.Source)
					}
				}
			} else {
				unknownDates = true
			}
		}
		// Legacy field rules are part of the exact submitted contract. A plain
		// negative answer without configured scoring is never interpreted here.
		if req.ScoreProfile == nil && field.Scoring != nil {
			if text, ok := answers[field.ID].ScalarText(); ok {
				critical := false
				for _, value := range field.Scoring.CriticalAnswers {
					critical = critical || value == text
				}
				if points, ok := field.Scoring.AnswerScores[text]; critical || (ok && !reviewed[field.ID] && vendorContributionAdverse(nil, &ResponseScoreResult{Mode: contract.ScoringMode}, points)) {
					add(field.ID, "", field.Label, "GAP", "RESPONSE")
				}
			}
		}
	}
	if expired || row.ResponseCurrency == "PARTIALLY_REPLACED" {
		value := true
		row.Outdated = &value
	} else if knownDates > 0 && !unknownDates {
		value := false
		row.Outdated = &value
	}
	automatic := row.Score
	// Legacy workflow captures have immutable answers and the pinned profile but
	// no response revision. Derive only requirement attention; never promote this
	// read-time result into a stored score or a completed bank assessment.
	if automatic == nil && req.ScoreProfile != nil {
		if revision, err := buildAutomaticResponseRevision(req, AccessAssurance(""), nil, answers); err == nil {
			automatic = revision.Score
		}
	}
	for _, source := range []struct {
		score *ResponseScoreResult
		name  string
	}{{automatic, "RESPONSE"}, {row.AssessedScore, "REVIEW"}} {
		if source.score == nil || (source.score.State != ResponseScoreFinal && source.score.State != ResponseScoreProvisional) {
			continue
		}
		for _, result := range source.score.ContributionResults {
			if source.name == "REVIEW" && !reviewed[result.ID] {
				continue
			}
			if source.name == "RESPONSE" && reviewed[result.ID] {
				continue
			}
			if result.Outcome == formcontract.ScoreIndeterminate || !vendorContributionAdverse(contract.ScoreProfile, source.score, result.Points) {
				continue
			}
			fieldID, label := "", ""
			if req.ScoreProfile == nil {
				fieldID = result.ID
				label = fields[fieldID].Label
			}
			if strings.HasPrefix(result.ID, "assessment:") {
				fieldID = strings.TrimPrefix(result.ID, "assessment:")
				label = fields[fieldID].Label
			}
			if req.ScoreProfile != nil {
				for _, c := range req.ScoreProfile.Contributions {
					if c.ID == result.ID {
						fieldID = c.Predicate.FieldID
						label = c.Label
						break
					}
				}
			}
			if label != "" && (fieldID == "" || fields[fieldID].ID != "") {
				add(fieldID, result.ID, label, "GAP", source.name)
			}
		}
		if req.ScoreProfile == nil || source.name == "REVIEW" {
			continue
		}
		for _, result := range source.score.RuleResults {
			if !result.Matched {
				continue
			}
			// FLOOR/CAP operate on adverse score after direction conversion.
			// A cap limits concern and cannot itself establish a failed check.
			adverse := result.Effect == formcontract.EffectDisqualify || result.Effect == formcontract.EffectFloor && vendorAdverseBand(contract.ScoreProfile, result.Value) || result.Effect == formcontract.EffectContribution && vendorContributionAdverse(contract.ScoreProfile, source.score, result.Value)
			if !adverse {
				continue
			}
			for _, rule := range req.ScoreProfile.Rules {
				if rule.ID == result.ID {
					add(rule.Predicate.FieldID, rule.ID, rule.Label, "GAP", source.name)
					break
				}
			}
		}
	}
}

func vendorHeldExpiry(r *CollectionResolution) string {
	if r.Source.ExpiresOn == "" {
		return r.Document.ExpiresOn
	}
	if r.Document.ExpiresOn == "" || r.Source.ExpiresOn < r.Document.ExpiresOn {
		return r.Source.ExpiresOn
	}
	return r.Document.ExpiresOn
}

func vendorHeldSource(r *CollectionResolution) string {
	status := r.BankReviewState
	if r.Source.Review != nil {
		status = r.Source.Review.Status
	}
	return vendorDocumentSource(status)
}

func vendorDocumentSource(status string) string {
	if status == "VALIDATED" || status == "REJECTED" || status == "EXPIRED" {
		return "REVIEW"
	}
	return "RESPONSE"
}

func vendorContributionAdverse(profile *formcontract.ScoreProfile, score *ResponseScoreResult, points int) bool {
	if score.Mode != formcontract.ScoringRisk && score.Mode != formcontract.ScoringCompliance && score.Direction == "" {
		return false
	}
	adverse := points
	if score.Direction == formcontract.DirectionLowIsPoor || (score.Direction == "" && score.Mode == formcontract.ScoringCompliance) {
		adverse = 100 - points
	}
	return vendorAdverseBand(profile, adverse)
}

func vendorAdverseBand(profile *formcontract.ScoreProfile, adverse int) bool {
	bands := formcontract.DefaultConcernBands()
	if profile != nil {
		bands = profile.Bands
	}
	for _, band := range bands {
		if adverse >= band.From && adverse <= band.Through {
			return band.Band != formcontract.ConcernLow
		}
	}
	return false
}
