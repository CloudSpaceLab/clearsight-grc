package metricview

import (
	"context"
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

var (
	ErrMetricMembershipInvalid  = errors.New("metric membership request is invalid")
	ErrMetricMembershipNotFound = errors.New("metric membership snapshot is not available")
)

type Member struct {
	MemberID    string `json:"member_id"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id,omitempty"`
	TargetTitle string `json:"target_title"`
	State       string `json:"state"`
	Accessible  bool   `json:"accessible"`
}

type MemberPage struct {
	SourceID           string   `json:"source_id"`
	MetricID           string   `json:"metric_id"`
	DefinitionRevision string   `json:"definition_revision"`
	Count              int      `json:"count"`
	Items              []Member `json:"items"`
	NextCursor         string   `json:"next_cursor,omitempty"`
}

type OrganizationMemberCount struct {
	OrganizationScopeID string `json:"organization_scope_id,omitempty"`
	Count               int    `json:"count"`
}

type OrganizationMemberCounts struct {
	SourceID           string                    `json:"source_id"`
	MetricID           string                    `json:"metric_id"`
	DefinitionRevision string                    `json:"definition_revision"`
	Count              int                       `json:"count"`
	Items              []OrganizationMemberCount `json:"items"`
}

type OrganizationMembershipReader interface {
	CountSnapshotMembersByOrganization(
		context.Context,
		string,
		string,
		string,
		string,
		string,
		string,
	) (OrganizationMemberCounts, error)
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
		string,
		string,
		int,
	) (MemberPage, error)
}

type MembershipWriter interface {
	RetainRuntimeSnapshot(context.Context, oversight.Snapshot) (string, error)
}
