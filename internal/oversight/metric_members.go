package oversight

import "time"

type MetricMember struct {
	MetricID    string
	TargetType  string
	TargetID    string
	Label       string
	SubjectType string
	SubjectID   string
}

type MetricMemberSummary struct {
	MetricID    string `json:"metric_id"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Label       string `json:"label"`
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
}

type MetricMemberPage struct {
	SourceSnapshotID string                `json:"source_snapshot_id"`
	MetricID         string                `json:"metric_id"`
	GeneratedAt      time.Time             `json:"generated_at"`
	Items            []MetricMemberSummary `json:"items"`
	NextCursor       string                `json:"next_cursor,omitempty"`
}
