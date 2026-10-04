package oversight

import "time"

const (
	MetricMembershipVersion = "home-membership-v1"

	MetricCriticalHighOpen = "critical_high_open"
	MetricOverdueOpen      = "overdue_open"
	MetricRoutingGaps      = "routing_gaps"
	MetricOutcomeFailures  = "outcome_failures"
)

type MetricMember struct {
	MetricID     string
	MemberType   string
	MemberID     string
	SubjectType  string
	SubjectID    string
	Reference    string
	Title        string
	State        string
	Priority     *int
	DueAt        *time.Time
}

type MetricMemberItem struct {
	MetricID    string     `json:"metric_id"`
	MemberType  string     `json:"member_type"`
	MemberID    string     `json:"member_id"`
	SubjectType string     `json:"subject_type"`
	SubjectID   string     `json:"subject_id"`
	Reference   string     `json:"reference,omitempty"`
	Title       string     `json:"title"`
	State       string     `json:"state"`
	Priority    *int       `json:"priority,omitempty"`
	DueAt       *time.Time `json:"due_at,omitempty"`
}

type MetricMemberPage struct {
	SourceID      string             `json:"source_id"`
	MetricID      string             `json:"metric_id"`
	GeneratedAt   time.Time          `json:"generated_at"`
	Total         int                `json:"total"`
	Items         []MetricMemberItem `json:"items"`
	NextCursor    string             `json:"next_cursor,omitempty"`
}
