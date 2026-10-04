package metricid

const (
	CriticalHighOpen = "critical_high_open"
	OverdueOpen      = "overdue_open"
	RoutingGaps      = "routing_gaps"
	OutcomeFailures  = "outcome_failures"
)

func Home() []string {
	return []string{CriticalHighOpen, OverdueOpen, RoutingGaps, OutcomeFailures}
}

func IsHome(value string) bool {
	switch value {
	case CriticalHighOpen, OverdueOpen, RoutingGaps, OutcomeFailures:
		return true
	default:
		return false
	}
}
