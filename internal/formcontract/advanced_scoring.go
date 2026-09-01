package formcontract

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	MaxScoreContributions     = 100
	MaxScoreRules             = 100
	MaxScorePredicateDepth    = 8
	MaxScorePredicateChildren = 20
)

type ContributionResult struct {
	ID      string       `json:"id"`
	Outcome ScoreOutcome `json:"outcome"`
	Points  int          `json:"points"`
	Weight  int          `json:"weight"`
}
type AdvancedRuleResult struct {
	ID      string         `json:"id"`
	Matched bool           `json:"matched"`
	Outcome ScoreOutcome   `json:"outcome"`
	Effect  RuleEffectKind `json:"effect"`
	Value   int            `json:"value,omitempty"`
	Weight  int            `json:"weight,omitempty"`
}
type AdvancedScoreResult struct {
	RawScore            *float64             `json:"raw_score,omitempty"`
	AdverseScore        *float64             `json:"adverse_score,omitempty"`
	Coverage            float64              `json:"coverage"`
	Final               bool                 `json:"final"`
	Band                ConcernBand          `json:"band"`
	Disqualified        bool                 `json:"disqualified,omitempty"`
	ContributionResults []ContributionResult `json:"contribution_results"`
	RuleResults         []AdvancedRuleResult `json:"rule_results"`
}

func DefaultConcernBands() []ScoreBandRange {
	return []ScoreBandRange{{ConcernLow, 0, 24}, {ConcernModerate, 25, 49}, {ConcernHigh, 50, 74}, {ConcernCritical, 75, 100}}
}

func normalizeScoreProfile(contract *Contract) error {
	if contract.ScoreProfile == nil {
		return nil
	}
	p := *contract.ScoreProfile
	p.Version = strings.TrimSpace(p.Version)
	p.Mode = ScoringMode(strings.ToUpper(strings.TrimSpace(string(p.Mode))))
	expected := DirectionHighIsPoor
	if p.Mode == ScoringCompliance {
		expected = DirectionLowIsPoor
	}
	if p.Direction == "" {
		p.Direction = expected
	}
	if p.Version == "" || len(p.Version) > 128 || p.Mode == ScoringNone || p.Mode != contract.ScoringMode || p.Direction != expected {
		return invalid("score profile version, mode or direction is invalid")
	}
	if len(p.Contributions) < 1 || len(p.Contributions) > MaxScoreContributions || len(p.Rules) > MaxScoreRules {
		return invalid("score profile contribution or rule limit is invalid")
	}
	fields := map[string]Field{}
	for _, f := range contract.Fields {
		fields[f.ID] = f
	}
	ids := map[string]bool{}
	for i := range p.Contributions {
		c := &p.Contributions[i]
		c.ID = strings.TrimSpace(c.ID)
		if c.ID == "" || ids[c.ID] || c.Weight < 1 || c.Weight > 100 || c.MatchPoints < 0 || c.MatchPoints > 100 || c.NonMatchPoints < 0 || c.NonMatchPoints > 100 {
			return invalid("score contribution is invalid")
		}
		ids[c.ID] = true
		if !slices.Contains([]MissingScoreBehaviour{MissingIndeterminate, MissingExclude, MissingZero}, c.Missing) {
			return invalid("score contribution missing handling is invalid")
		}
		if err := normalizePredicate(&c.Predicate, fields, 1); err != nil {
			return err
		}
	}
	ruleIDs := map[string]bool{}
	maxFloor, minCap := -1, 101
	for i := range p.Rules {
		r := &p.Rules[i]
		r.ID = strings.TrimSpace(r.ID)
		if r.ID == "" || ruleIDs[r.ID] {
			return invalid("advanced score rule id is invalid")
		}
		ruleIDs[r.ID] = true
		if err := normalizePredicate(&r.Predicate, fields, 1); err != nil {
			return err
		}
		switch r.Effect.Kind {
		case EffectContribution:
			if r.Effect.Weight < 1 || r.Effect.Weight > 100 || r.Effect.Value < 0 || r.Effect.Value > 100 {
				return invalid("contribution effect is invalid")
			}
		case EffectFloor:
			if r.Effect.Value < 0 || r.Effect.Value > 100 {
				return invalid("floor is invalid")
			}
			maxFloor = max(maxFloor, r.Effect.Value)
		case EffectCap:
			if r.Effect.Value < 0 || r.Effect.Value > 100 {
				return invalid("cap is invalid")
			}
			minCap = min(minCap, r.Effect.Value)
		case EffectDisqualify:
		default:
			return invalid("rule effect is invalid")
		}
	}
	if maxFloor > minCap {
		return invalid("score profile floor cannot exceed cap")
	}
	if len(p.Bands) == 0 {
		p.Bands = DefaultConcernBands()
	}
	if err := validateBands(p.Bands); err != nil {
		return err
	}
	contract.ScoreProfile = &p
	return nil
}

func normalizePredicate(p *Predicate, fields map[string]Field, depth int) error {
	if depth > MaxScorePredicateDepth || len(p.Children) > MaxScorePredicateChildren || len(p.Values) > MaxScorePredicateChildren {
		return invalid("score predicate limit exceeded")
	}
	switch p.Operator {
	case PredicateAnd, PredicateOr:
		if len(p.Children) < 2 {
			return invalid("logical predicate requires two children")
		}
	case PredicateNot:
		if len(p.Children) != 1 {
			return invalid("NOT requires one child")
		}
	default:
		f, ok := fields[p.FieldID]
		if !ok {
			return invalid("score predicate field is unknown")
		}
		if err := validateLeaf(*p, f); err != nil {
			return err
		}
	}
	for i := range p.Children {
		if err := normalizePredicate(&p.Children[i], fields, depth+1); err != nil {
			return err
		}
	}
	return nil
}
func validateLeaf(p Predicate, f Field) error {
	one := slices.Contains([]PredicateOperator{PredicateEquals, PredicateNotEquals, PredicateContains, PredicateGreaterThan, PredicateGreaterEqual, PredicateLessThan, PredicateLessEqual, PredicateDateBefore, PredicateDateOnOrAfter}, p.Operator)
	many := slices.Contains([]PredicateOperator{PredicateIn, PredicateNotIn, PredicateContainsAny, PredicateContainsAll}, p.Operator)
	if one && len(p.Values) != 1 || many && len(p.Values) < 1 || (p.Operator == PredicateNumberBetween || p.Operator == PredicateDateBetween) && len(p.Values) != 2 || (p.Operator == PredicateAnswered || p.Operator == PredicateUnanswered) && len(p.Values) != 0 {
		return invalid("predicate comparison values are invalid")
	}
	supported := one || many || p.Operator == PredicateNumberBetween || p.Operator == PredicateDateBetween || p.Operator == PredicateAnswered || p.Operator == PredicateUnanswered
	if !supported {
		return invalid("predicate operator %s is not supported", p.Operator)
	}
	numeric := slices.Contains([]Type{TypeInteger, TypeDecimal, TypePercentage, TypeCurrency}, f.Type)
	date := f.Type == TypeDate
	multi := f.Type == TypeMultiSelect
	if slices.Contains([]PredicateOperator{PredicateGreaterThan, PredicateGreaterEqual, PredicateLessThan, PredicateLessEqual, PredicateNumberBetween}, p.Operator) && !numeric {
		return invalid("numeric predicate requires numeric field")
	}
	if slices.Contains([]PredicateOperator{PredicateDateBefore, PredicateDateOnOrAfter, PredicateDateBetween}, p.Operator) && !date {
		return invalid("date predicate requires date field")
	}
	if slices.Contains([]PredicateOperator{PredicateContains, PredicateContainsAny, PredicateContainsAll}, p.Operator) && !multi {
		return invalid("contains predicate requires multi-select field")
	}
	for _, v := range p.Values {
		if v == "" {
			return invalid("predicate values cannot be empty")
		}
		if numeric {
			if n, e := strconv.ParseFloat(v, 64); e != nil || math.IsNaN(n) || math.IsInf(n, 0) {
				return invalid("numeric predicate value is invalid")
			}
		}
		if date {
			if _, e := time.Parse("2006-01-02", v); e != nil {
				return invalid("date predicate value is invalid")
			}
		}
	}
	return nil
}
func validateBands(b []ScoreBandRange) error {
	if len(b) != 4 {
		return invalid("four concern bands are required")
	}
	covered := [101]bool{}
	seen := map[ConcernBand]bool{}
	for _, r := range b {
		if !slices.Contains([]ConcernBand{ConcernLow, ConcernModerate, ConcernHigh, ConcernCritical}, r.Band) || seen[r.Band] || r.From < 0 || r.Through > 100 || r.From > r.Through {
			return invalid("concern bands are invalid")
		}
		seen[r.Band] = true
		for i := r.From; i <= r.Through; i++ {
			if covered[i] {
				return invalid("concern bands overlap")
			}
			covered[i] = true
		}
	}
	if slices.Contains(covered[:], false) {
		return invalid("concern bands must cover 0-100")
	}
	return nil
}

func EvaluateScoreProfile(profile ScoreProfile, contract Contract, answers map[string]AnswerValue) (AdvancedScoreResult, error) {
	contract.ScoreProfile = &profile
	if contract.ScoringMode == "" {
		contract.ScoringMode = profile.Mode
	}
	n, err := Normalize(contract)
	if err != nil {
		return AdvancedScoreResult{}, err
	}
	profile = *n.ScoreProfile
	visible, err := VisibleFields(n, answers)
	if err != nil {
		return AdvancedScoreResult{}, err
	}
	vis := map[string]bool{}
	for _, f := range visible {
		vis[f.ID] = true
	}
	out := AdvancedScoreResult{ContributionResults: []ContributionResult{}, RuleResults: []AdvancedRuleResult{}}
	achieved, covered, total := 0, 0, 0
	for _, c := range profile.Contributions {
		if hidden(c.Predicate, vis) {
			out.ContributionResults = append(out.ContributionResults, ContributionResult{ID: c.ID, Outcome: ScoreIndeterminate})
			continue
		}
		m, d, e := evalPredicate(c.Predicate, answers)
		if e != nil {
			return AdvancedScoreResult{}, e
		}
		if !d {
			if c.Missing == MissingExclude {
				continue
			}
			total += c.Weight
			if c.Missing == MissingZero {
				covered += c.Weight
			}
			out.ContributionResults = append(out.ContributionResults, ContributionResult{ID: c.ID, Outcome: ScoreIndeterminate, Weight: c.Weight})
			continue
		}
		points := c.NonMatchPoints
		if m {
			points = c.MatchPoints
		}
		total += c.Weight
		covered += c.Weight
		achieved += points * c.Weight
		out.ContributionResults = append(out.ContributionResults, ContributionResult{ID: c.ID, Points: points, Weight: c.Weight})
	}
	floor, cap := -1, 101
	for _, r := range profile.Rules {
		m, d, e := evalPredicate(r.Predicate, answers)
		if e != nil {
			return AdvancedScoreResult{}, e
		}
		out.RuleResults = append(out.RuleResults, AdvancedRuleResult{ID: r.ID, Matched: m && d, Effect: r.Effect.Kind, Value: r.Effect.Value, Weight: r.Effect.Weight})
		if !d {
			if r.Effect.Kind == EffectContribution {
				total += r.Effect.Weight
			}
			continue
		}
		if r.Effect.Kind == EffectContribution {
			total += r.Effect.Weight
			covered += r.Effect.Weight
			if m {
				achieved += r.Effect.Value * r.Effect.Weight
			}
		}
		if m {
			switch r.Effect.Kind {
			case EffectFloor:
				floor = max(floor, r.Effect.Value)
			case EffectCap:
				cap = min(cap, r.Effect.Value)
			case EffectDisqualify:
				out.Disqualified = true
			}
		}
	}
	if total == 0 {
		return AdvancedScoreResult{}, invalid("no applicable score contribution")
	}
	out.Coverage = math.Round(float64(covered)/float64(total)*10000) / 10000
	out.Final = out.Coverage == 1
	if !out.Final {
		return out, nil
	}
	raw := math.Round(float64(achieved)/float64(total)*100) / 100
	adverse := raw
	if profile.Mode == ScoringCompliance {
		adverse = 100 - raw
	}
	if floor >= 0 && adverse < float64(floor) {
		adverse = float64(floor)
	}
	if cap <= 100 && adverse > float64(cap) {
		adverse = float64(cap)
	}
	out.RawScore = &raw
	out.AdverseScore = &adverse
	out.Band = bandForScore(profile.Bands, adverse)
	if out.Disqualified {
		out.Band = ConcernCritical
	}
	return out, nil
}
func hidden(p Predicate, v map[string]bool) bool {
	if p.FieldID != "" {
		return !v[p.FieldID]
	}
	for _, c := range p.Children {
		if hidden(c, v) {
			return true
		}
	}
	return false
}
func evalPredicate(p Predicate, a map[string]AnswerValue) (bool, bool, error) {
	switch p.Operator {
	case PredicateAnd:
		for _, c := range p.Children {
			m, d, e := evalPredicate(c, a)
			if e != nil {
				return false, false, e
			}
			if d && !m {
				return false, true, nil
			}
			if !d {
				return false, false, nil
			}
		}
		return true, true, nil
	case PredicateOr:
		anyMissing := false
		for _, c := range p.Children {
			m, d, e := evalPredicate(c, a)
			if e != nil {
				return false, false, e
			}
			if d && m {
				return true, true, nil
			}
			anyMissing = anyMissing || !d
		}
		return false, !anyMissing, nil
	case PredicateNot:
		m, d, e := evalPredicate(p.Children[0], a)
		return !m, d, e
	}
	av, exists := a[p.FieldID]
	if p.Operator == PredicateAnswered {
		return exists && av.Answered(), true, nil
	}
	if p.Operator == PredicateUnanswered {
		return !exists || !av.Answered(), true, nil
	}
	if !exists || !av.Answered() {
		return false, false, nil
	}
	vals := answerComparableValues(av)
	switch p.Operator {
	case PredicateEquals, PredicateIn:
		return intersects(vals, p.Values), true, nil
	case PredicateNotEquals, PredicateNotIn:
		return !intersects(vals, p.Values), true, nil
	case PredicateContains:
		return slices.Contains(vals, p.Values[0]), true, nil
	case PredicateContainsAny:
		return intersects(vals, p.Values), true, nil
	case PredicateContainsAll:
		for _, x := range p.Values {
			if !slices.Contains(vals, x) {
				return false, true, nil
			}
		}
		return true, true, nil
	}
	text, ok := av.ScalarText()
	if !ok {
		return false, true, invalid("scalar answer required")
	}
	if slices.Contains([]PredicateOperator{PredicateGreaterThan, PredicateGreaterEqual, PredicateLessThan, PredicateLessEqual, PredicateNumberBetween}, p.Operator) {
		return compareNum(text, p.Operator, p.Values)
	}
	return compareDate(text, p.Operator, p.Values)
}
func intersects(a, b []string) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}
	return false
}
func compareNum(s string, o PredicateOperator, v []string) (bool, bool, error) {
	n, e := strconv.ParseFloat(s, 64)
	if e != nil {
		return false, true, invalid("numeric answer is invalid")
	}
	x, _ := strconv.ParseFloat(v[0], 64)
	switch o {
	case PredicateGreaterThan:
		return n > x, true, nil
	case PredicateGreaterEqual:
		return n >= x, true, nil
	case PredicateLessThan:
		return n < x, true, nil
	case PredicateLessEqual:
		return n <= x, true, nil
	default:
		y, _ := strconv.ParseFloat(v[1], 64)
		return n >= x && n <= y, true, nil
	}
}
func compareDate(s string, o PredicateOperator, v []string) (bool, bool, error) {
	n, e := time.Parse("2006-01-02", s)
	if e != nil {
		return false, true, invalid("date answer is invalid")
	}
	x, _ := time.Parse("2006-01-02", v[0])
	switch o {
	case PredicateDateBefore:
		return n.Before(x), true, nil
	case PredicateDateOnOrAfter:
		return !n.Before(x), true, nil
	default:
		y, _ := time.Parse("2006-01-02", v[1])
		return !n.Before(x) && !n.After(y), true, nil
	}
}
func bandForScore(b []ScoreBandRange, s float64) ConcernBand {
	for _, r := range b {
		if s >= float64(r.From) && (s < float64(r.Through+1) || r.Through == 100 && s == 100) {
			return r.Band
		}
	}
	return ConcernCritical
}
