package documentcoverage

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

func Evaluate(candidates []Candidate, programs []ProgramSnapshot) Evaluation {
	result := Evaluation{Candidates: make([]Candidate, 0, len(candidates)), Suggestions: []Suggestion{}}
	denominator := 0
	for _, source := range candidates {
		candidate := cloneCandidate(source)
		if candidate.Eligible && (candidate.Review == nil || candidate.Review.Decision != DecisionNotApplicable) {
			denominator++
		}
		candidate.Matches = MatchCandidate(candidate, programs)
		candidate.Classification = classifyCandidate(candidate)
		if suggestion, ok := suggest(candidate, programs); ok {
			result.Suggestions = append(result.Suggestions, suggestion)
		}
		result.Candidates = append(result.Candidates, candidate)
	}
	setDenominator := func(metric *CountMetric) { metric.Denominator = denominator }
	setDenominator(&result.Metrics.EstimatedVerified)
	setDenominator(&result.Metrics.Verified)
	setDenominator(&result.Metrics.RequirementMapped)
	setDenominator(&result.Metrics.ControlImplemented)
	setDenominator(&result.Metrics.EvidenceSupported)

	for _, candidate := range result.Candidates {
		if !candidate.Eligible || (candidate.Review != nil && candidate.Review.Decision == DecisionNotApplicable) {
			continue
		}
		match, accepted := selectedMatch(candidate)
		if !accepted {
			if len(candidate.Matches) > 0 && candidate.Matches[0].Band == MatchStrong && candidate.Matches[0].Coverage.Complete {
				result.Metrics.EstimatedVerified.Numerator++
			}
			continue
		}
		result.Metrics.RequirementMapped.Numerator++
		if match.Coverage.ControlImplemented {
			result.Metrics.ControlImplemented.Numerator++
		}
		if match.Coverage.EvidenceSupported {
			result.Metrics.EvidenceSupported.Numerator++
		}
		if match.Coverage.Complete {
			result.Metrics.Verified.Numerator++
		}
	}
	return result
}

func classifyCandidate(candidate Candidate) Classification {
	if !candidate.Eligible {
		return ClassificationNeedsReview
	}
	if candidate.Review != nil {
		switch candidate.Review.Decision {
		case DecisionNotApplicable:
			return ClassificationNotApplicable
		case DecisionReject:
			return ClassificationGap
		case DecisionAccept:
			match, ok := selectedMatch(candidate)
			if !ok {
				return ClassificationNeedsReview
			}
			if match.Coverage.Complete {
				return ClassificationVerified
			}
			if !match.Coverage.ControlImplemented {
				return ClassificationControlGap
			}
			return ClassificationNoEvidence
		}
	}
	if len(candidate.Uncertainty) > 1 || len(candidate.Matches) == 0 {
		return ClassificationGap
	}
	return ClassificationPartialMatch
}

func selectedMatch(candidate Candidate) (Match, bool) {
	if candidate.Review == nil || candidate.Review.Decision != DecisionAccept {
		return Match{}, false
	}
	for _, match := range candidate.Matches {
		if match.ID == candidate.Review.MatchID {
			return match, true
		}
	}
	return Match{}, false
}

func suggest(candidate Candidate, programs []ProgramSnapshot) (Suggestion, bool) {
	if !candidate.Eligible || candidate.Classification == ClassificationVerified || candidate.Classification == ClassificationNotApplicable {
		return Suggestion{}, false
	}
	suggestion := Suggestion{CandidateID: candidate.ID, Status: SuggestionProposed}
	if len(candidate.Matches) > 0 {
		best := candidate.Matches[0]
		suggestion.ProgramID = best.ProgramID
		suggestion.RequirementID = best.RequirementID
		if best.Band == MatchStrong && candidate.Review == nil {
			suggestion.Type = SuggestionLinkRequirement
			suggestion.Title = "Review the existing requirement match"
			suggestion.Rationale = best.Rationale
		} else if candidate.Review != nil && candidate.Review.Decision == DecisionAccept {
			suggestion.Type = SuggestionCreateMatter
			suggestion.Title = "Prepare remediation work"
			suggestion.Rationale = "The accepted requirement chain is incomplete."
		} else {
			suggestion.Type = SuggestionAddRequirement
			suggestion.Title = "Consider a draft requirement"
			suggestion.Rationale = "A related Program exists, but no strong current requirement match was found."
		}
	} else if len(programs) > 0 {
		suggestion.Type = SuggestionAddRequirement
		suggestion.ProgramID = bestScopedProgram(candidate, programs)
		suggestion.Title = "Consider a draft requirement"
		suggestion.Rationale = "No current requirement covers this obligation."
	} else {
		suggestion.Type = SuggestionCreateProgram
		suggestion.Title = "Consider a new draft Program"
		suggestion.Rationale = "No current Program fits this document scope."
	}
	suggestion.ID = stableSuggestionID(candidate.ID, string(suggestion.Type), suggestion.ProgramID, suggestion.RequirementID)
	return suggestion, true
}

func bestScopedProgram(candidate Candidate, programs []ProgramSnapshot) string {
	values := append([]ProgramSnapshot(nil), programs...)
	sort.Slice(values, func(i, j int) bool { return values[i].Code < values[j].Code })
	for _, program := range values {
		if !hardScopeConflict(candidate, program) {
			return program.ProgramID
		}
	}
	return ""
}

func stableSuggestionID(parts ...string) string {
	joined := ""
	for _, part := range parts {
		joined += part + "\x00"
	}
	digest := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(digest[:16])
}

func cloneCandidate(value Candidate) Candidate {
	value.Citations = append([]string(nil), value.Citations...)
	value.Dates = append([]string(nil), value.Dates...)
	value.Topics = append([]string(nil), value.Topics...)
	value.Uncertainty = append([]string(nil), value.Uncertainty...)
	value.Matches = append([]Match(nil), value.Matches...)
	if value.Review != nil {
		review := *value.Review
		value.Review = &review
	}
	return value
}
