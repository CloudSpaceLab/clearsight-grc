package bankverticals

import (
	"sort"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
)

// TodayItems projects only open journey actions into the personal work queue.
// Completed journeys stay available in Explore but do not create noise.
func TodayItems(journeys []Journey, now time.Time) []today.AttentionItem {
	items := make([]today.AttentionItem, 0, len(journeys))
	for _, journey := range journeys {
		if journey.Status == "NOT_SET_UP" || journey.Status == "CLOSED" || journey.NextAction == "" {
			continue
		}
		due := now.Add(14 * 24 * time.Hour)
		if journey.DueAt != nil {
			due = *journey.DueAt
		}
		evidenceLabel := "Current journey record"
		if len(journey.SourceNames) > 0 {
			evidenceLabel = journey.SourceNames[0]
		}
		items = append(items, today.AttentionItem{
			ID:                 "journey_" + string(journey.Code),
			Type:               string(journey.Code),
			Title:              journey.NextAction,
			WhyNow:             journey.Summary,
			Scope:              journey.Title,
			State:              journey.StatusLabel,
			Evidence:           evidenceLabel,
			Owner:              journey.Owner,
			DueAt:              due,
			PrimaryAction:      journey.ActionLabel,
			ActionTargetType:   journey.ActionTargetType,
			ActionTargetID:     journey.ActionTargetID,
			InterventionClass:  interventionClass(journey),
			MaterialConclusion: journey.Summary,
			ChangeSummary:      journey.StatusLabel,
			Recommendation: &today.GovernedRecommendation{
				ProposedAction: journey.ActionLabel,
				Rationale:      journey.Summary,
			},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DueAt.Before(items[j].DueAt) })
	return items
}

func interventionClass(journey Journey) today.InterventionClass {
	if journey.ActionTargetType == ActionTargetEvidenceRequest {
		return today.InterventionEvidenceException
	}
	if journey.Status == "VERIFICATION" {
		return today.InterventionVerification
	}
	return today.InterventionReview
}
