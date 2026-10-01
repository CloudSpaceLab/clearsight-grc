package oversight

import "github.com/CloudSpaceLab/clearsight-grc/internal/metric"

type headlineMetricSpec struct {
	code       string
	label      string
	drillKey   string
	value      func(Counts) int
	active     metric.State
	activeText string
	reason     string
}

var headlineMetricSpecs = []headlineMetricSpec{
	{
		code:       "critical_high_open",
		label:      "Critical and high",
		drillKey:   "critical-high",
		value:      func(counts Counts) int { return counts.CriticalHigh },
		active:     metric.StateCritical,
		activeText: "Needs attention",
		reason:     "Open priority 4–5 issues",
	},
	{
		code:       "overdue_open",
		label:      "Overdue",
		drillKey:   "overdue",
		value:      func(counts Counts) int { return counts.Overdue },
		active:     metric.StateWarning,
		activeText: "Past due",
		reason:     "Open issues past their due date",
	},
	{
		code:       "routing_gaps",
		label:      "Routing gaps",
		drillKey:   "routing-gaps",
		value:      func(counts Counts) int { return counts.RoutingFailures },
		active:     metric.StateWarning,
		activeText: "Routing blocked",
		reason:     "Active work without a resolved recipient",
	},
	{
		code:       "outcome_failures",
		label:      "Outcome failures",
		drillKey:   "outcome-failures",
		value:      func(counts Counts) int { return counts.OutcomeFailures },
		active:     metric.StateCritical,
		activeText: "Outcome not confirmed",
		reason:     "Latest outcome check failed or was inconclusive",
	},
}

func headlineMetrics(snapshot Snapshot) []metric.Snapshot {
	complete := snapshot.Freshness == FreshnessCurrent &&
		snapshot.Coverage.Unknown != nil &&
		*snapshot.Coverage.Unknown == 0

	items := make([]metric.Snapshot, 0, len(headlineMetricSpecs))
	for _, spec := range headlineMetricSpecs {
		value := spec.value(snapshot.Counts)
		state := metric.StateClear
		stateLabel := "Current"
		if value > 0 {
			state = spec.active
			stateLabel = spec.activeText
		} else if !complete {
			state = metric.StateUnknown
			stateLabel = "Coverage incomplete"
		}
		items = append(items, metric.Snapshot{
			Code:              spec.code,
			Label:             spec.label,
			Value:             int64(value),
			Unit:              "COUNT",
			State:             state,
			StateLabel:        stateLabel,
			Reason:            spec.reason,
			Population:        snapshot.Coverage.Population,
			Excluded:          snapshot.Coverage.Excluded,
			Unknown:           snapshot.Coverage.Unknown,
			Complete:          complete,
			GeneratedAt:       snapshot.GeneratedAt,
			ProjectionVersion: snapshot.ProjectionVersion,
			Direction:         metric.DirectionUnknown,
			DrillKey:          spec.drillKey,
		})
	}
	return items
}
