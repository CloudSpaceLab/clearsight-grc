package formcontract

import "strings"

type AssessmentMode string

const (
	AssessmentNone            AssessmentMode = "NONE"
	AssessmentManual          AssessmentMode = "MANUAL"
	AssessmentAutomatic       AssessmentMode = "AUTOMATIC"
	AssessmentAutomaticReview AssessmentMode = "AUTOMATIC_REVIEW"
)

type AssessmentOutcome struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Points int    `json:"points"`
}
type FieldAssessment struct {
	Mode         AssessmentMode      `json:"mode"`
	Required     bool                `json:"required"`
	Weight       int                 `json:"weight"`
	ReviewerRole string              `json:"reviewer_role,omitempty"`
	Rubric       []AssessmentOutcome `json:"rubric,omitempty"`
}

func (a *FieldAssessment) NeedsReview() bool {
	return a != nil && (a.Mode == AssessmentManual || a.Mode == AssessmentAutomaticReview)
}
func normalizeAssessment(f *Field) error {
	if f.Assessment == nil {
		return nil
	}
	a := *f.Assessment
	a.Rubric = append([]AssessmentOutcome(nil), a.Rubric...)
	a.ReviewerRole = strings.TrimSpace(a.ReviewerRole)
	switch a.Mode {
	case AssessmentNone, AssessmentManual, AssessmentAutomatic, AssessmentAutomaticReview:
	default:
		return invalid("%s assessment mode is invalid", f.Label)
	}
	if a.Mode != AssessmentNone && (a.Weight < 1 || a.Weight > 100) {
		return invalid("%s assessment weight must be from 1 to 100", f.Label)
	}
	if len(a.ReviewerRole) > 128 {
		return invalid("%s reviewer role is too long", f.Label)
	}
	if a.NeedsReview() && (a.ReviewerRole == "" || len(a.Rubric) < 1 || len(a.Rubric) > 50) {
		return invalid("%s requires a reviewer role and approved rubric", f.Label)
	}
	if !a.NeedsReview() && (a.Required || len(a.Rubric) > 0) {
		return invalid("%s bank review must be enabled to require a rubric or review", f.Label)
	}
	seen := map[string]bool{}
	for i := range a.Rubric {
		r := &a.Rubric[i]
		r.ID = strings.TrimSpace(r.ID)
		r.Label = strings.TrimSpace(r.Label)
		if r.ID == "" || len(r.ID) > 80 || r.Label == "" || len(r.Label) > 200 || seen[r.ID] || r.Points < 0 || r.Points > 100 {
			return invalid("%s rubric outcome is invalid", f.Label)
		}
		seen[r.ID] = true
	}
	if a.Mode == AssessmentManual && f.Scoring != nil {
		return invalid("%s bank review cannot also have automatic field scores", f.Label)
	}
	if a.Mode == AssessmentNone && f.Scoring != nil {
		return invalid("%s unscored field cannot retain automatic field scores", f.Label)
	}
	f.Assessment = &a
	if f.Scoring != nil && (a.Mode == AssessmentAutomatic || a.Mode == AssessmentAutomaticReview) {
		copy := *f.Scoring
		copy.Weight = a.Weight
		f.Scoring = &copy
	}
	return nil
}

// EvaluateAssessment reuses the automatic evaluator with separately supplied bank
// outcomes. Answers remain immutable and never encode a bank judgement.
func EvaluateAssessment(contract Contract, answers map[string]AnswerValue, outcomes map[string]string) (AdvancedScoreResult, error) {
	n, err := Normalize(contract)
	if err != nil {
		return AdvancedScoreResult{}, err
	}
	visible, err := VisibleFields(n, answers)
	if err != nil {
		return AdvancedScoreResult{}, err
	}
	extra := []ContributionResult{}
	critical := false
	weightFor := func(f Field, w int) int {
		return w
	}
	overrides := map[string]int{}
	for _, f := range visible {
		a := f.Assessment
		if a != nil && a.Mode == AssessmentNone {
			continue
		}
		if n.ScoreProfile == nil && f.Scoring != nil {
			text, ok := answers[f.ID].ScalarText()
			points, found := f.Scoring.AnswerScores[text]
			weight := f.Scoring.Weight
			if a != nil {
				weight = a.Weight
			}
			r := ContributionResult{ID: f.ID, Weight: weightFor(f, weight), Outcome: ScoreIndeterminate}
			if ok && found {
				r.Points = points
				r.Outcome = ScorePassed
				critical = critical || slicesContains(f.Scoring.CriticalAnswers, text)
			}
			extra = append(extra, r)
		}
		if !a.NeedsReview() {
			continue
		}
		outcome, exists := outcomes[f.ID]
		if !exists {
			if a.Required {
				if n.ScoreProfile == nil && f.Scoring != nil {
					extra[len(extra)-1].Outcome = ScoreIndeterminate
				} else {
					extra = append(extra, ContributionResult{ID: "assessment:" + f.ID, Weight: weightFor(f, a.Weight), Outcome: ScoreIndeterminate})
				}
			}
			continue
		}
		found := false
		for _, r := range a.Rubric {
			if r.ID == outcome {
				if a.Mode == AssessmentAutomaticReview && f.Scoring != nil && n.ScoreProfile == nil {
					prior := &extra[len(extra)-1]
					if prior.Outcome != ScoreIndeterminate {
						if n.ScoringMode == ScoringCompliance {
							prior.Points = min(prior.Points, r.Points)
						} else {
							prior.Points = max(prior.Points, r.Points)
						}
					}
				} else if a.Mode == AssessmentAutomaticReview && n.ScoreProfile != nil {
					overrides[f.ID] = r.Points
				} else {
					extra = append(extra, ContributionResult{ID: "assessment:" + f.ID, Weight: weightFor(f, a.Weight), Points: r.Points, Outcome: ScorePassed})
				}
				found = true
				break
			}
		}
		if !found {
			return AdvancedScoreResult{}, invalid("%s assessment outcome is not in its approved rubric", f.Label)
		}
	}
	p := ScoreProfile{Version: "field-assessment-v1", Mode: n.ScoringMode, Direction: DirectionHighIsPoor, Bands: DefaultConcernBands()}
	if p.Mode == ScoringNone {
		p.Mode = ScoringRisk
	}
	if p.Mode == ScoringCompliance {
		p.Direction = DirectionLowIsPoor
	}
	if n.ScoreProfile != nil {
		p = *n.ScoreProfile
		p.Contributions = append([]ScoreContribution(nil), p.Contributions...)
		for i := range p.Contributions {
			c := &p.Contributions[i]
			for _, f := range visible {
				if c.Predicate.FieldID == f.ID && f.Assessment != nil {
					c.Weight = f.Assessment.Weight
					if points, ok := overrides[f.ID]; ok {
						if p.Mode == ScoringCompliance {
							c.MatchPoints = min(c.MatchPoints, points)
							c.NonMatchPoints = min(c.NonMatchPoints, points)
						} else {
							c.MatchPoints = max(c.MatchPoints, points)
							c.NonMatchPoints = max(c.NonMatchPoints, points)
						}
					}
				}
			}
		}
	}
	// Preserve the approved section share even when conditions hide some fields
	// or optional reviews are absent. Field weights divide their own section.
	effectiveWeights := map[string]float64{}
	if n.ScoreProfile == nil && n.ScoringMode == ScoringCompliance {
		sections := map[string]string{}
		sectionTotals := map[string]int{}
		sectionWeights := map[string]int{}
		for _, section := range n.Sections {
			sectionWeights[section.ID] = section.Weight
		}
		for _, f := range visible {
			sections[f.ID] = f.SectionID
			sections["assessment:"+f.ID] = f.SectionID
		}
		for _, c := range extra {
			sectionTotals[sections[c.ID]] += c.Weight
		}
		for _, c := range extra {
			section := sections[c.ID]
			if sectionTotals[section] > 0 {
				effectiveWeights[c.ID] = float64(c.Weight) * float64(sectionWeights[section]) / float64(sectionTotals[section])
			}
		}
	}
	result, err := evaluateScoreProfile(p, n, answers, extra, false, effectiveWeights)
	if err != nil {
		return result, err
	}
	if critical {
		result.Disqualified = true
		result.Band = ConcernCritical
	}
	return result, nil
}

func validateAssessmentProfile(c Contract) error {
	for _, f := range c.Fields {
		if f.Assessment == nil || (f.Assessment.Mode != AssessmentAutomatic && f.Assessment.Mode != AssessmentAutomaticReview) {
			continue
		}
		if c.ScoreProfile == nil {
			if f.Scoring == nil {
				return invalid("%s requires automatic scoring rules", f.Label)
			}
			continue
		}
		direct := 0
		for _, contribution := range c.ScoreProfile.Contributions {
			if contribution.Predicate.FieldID == f.ID {
				direct++
			}
		}
		if f.Assessment.Mode == AssessmentAutomaticReview && direct != 1 {
			return invalid("%s bank confirmation requires one automatic contribution for this field", f.Label)
		}
		if direct > 1 {
			return invalid("%s assessment weight requires one automatic contribution for this field", f.Label)
		}
	}
	if c.ScoreProfile == nil {
		return nil
	}
	for _, f := range c.Fields {
		if f.Assessment == nil || (f.Assessment.Mode != AssessmentNone && f.Assessment.Mode != AssessmentManual) {
			continue
		}
		for _, contribution := range c.ScoreProfile.Contributions {
			if predicateUsesField(contribution.Predicate, f.ID) {
				return invalid("%s cannot retain automatic contributions in its assessment mode", f.Label)
			}
		}
		for _, rule := range c.ScoreProfile.Rules {
			if rule.Effect.Kind == EffectContribution && predicateUsesField(rule.Predicate, f.ID) {
				return invalid("%s cannot retain automatic contribution rules in its assessment mode", f.Label)
			}
		}
	}
	return nil
}
func predicateUsesField(p Predicate, id string) bool {
	if p.FieldID == id {
		return true
	}
	for _, child := range p.Children {
		if predicateUsesField(child, id) {
			return true
		}
	}
	return false
}
