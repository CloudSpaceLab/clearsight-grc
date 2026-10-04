package metricview

import (
	"context"
	"errors"
)

var (
	ErrMetricMembershipInvalid   = errors.New("metric membership request is invalid")
	ErrMetricMembershipNotFound  = errors.New("metric membership snapshot is not available")
)

type Member struct {
	MemberID    string `json:"member_id"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	TargetTitle string `json:"target_title"`
	State       string `json:"state"`
}

type MemberPage struct {
	SourceID           string   `json:"source_id"`
	MetricID           string   `json:"metric_id"`
	DefinitionRevision string   `json:"definition_revision"`
	Count              int      `json:"count"`
	Items              []Member `json:"items"`
	NextCursor         string   `json:"next_cursor,omitempty"`
}

type MembershipReader interface {
	ListSnapshotMembers(
		context.Context,
		string,
		string,
		string,
		string,
		string,
		string,
		int,
	) (MemberPage, error)
}
