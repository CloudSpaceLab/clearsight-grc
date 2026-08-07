//go:build postgres

package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

type lifecycleAuthorityStub struct {
	resolution authority.Resolution
	err        error
}

func (s lifecycleAuthorityStub) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return s.resolution, s.err
}
func (s lifecycleAuthorityStub) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (s lifecycleAuthorityStub) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (s lifecycleAuthorityStub) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestLifecycleAssignmentFailsClosedForUnresolvedAuthorityShapes(t *testing.T) {
	matter := continuity.Matter{
		TenantID: "bank", ID: "matter-1", Priority: 5,
		Scope: []byte(`{"access":"RESTRICTED","allowed_principal_ids":["allowed"]}`),
	}
	requirement := WorkRequirement{
		Responsibility: authority.ResponsibilityAcknowledger,
		CommandName:    "matter.response.transition",
		Materiality:    5,
	}
	now := time.Date(2026, 8, 7, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		service    authority.Service
		wantState  string
		wantStatus Status
	}{
		{
			name:       "no route",
			service:    lifecycleAuthorityStub{err: authority.ErrNoRoute},
			wantState:  "NO_ELIGIBLE_ROUTE",
			wantStatus: StatusBlocked,
		},
		{
			name:       "ambiguous route",
			service:    lifecycleAuthorityStub{err: authority.ErrAmbiguousRoute},
			wantState:  "AMBIGUOUS_ROUTE",
			wantStatus: StatusBlocked,
		},
		{
			name: "candidate set",
			service: lifecycleAuthorityStub{resolution: authority.Resolution{
				Principal: authority.Principal{ID: "allowed", Kind: "PERSON"},
				CandidatePrincipals: []authority.Principal{
					{ID: "allowed", Kind: "PERSON"}, {ID: "other", Kind: "PERSON"},
				},
				Strategy: "CANDIDATE_SET", RuleID: "rule-1", PolicyVersion: "v1",
			}},
			wantState:  "CANDIDATE_SET",
			wantStatus: StatusBlocked,
		},
		{
			name: "non-person route",
			service: lifecycleAuthorityStub{resolution: authority.Resolution{
				Principal: authority.Principal{ID: "queue-1", Kind: "QUEUE"},
				Strategy: "DIRECT", RuleID: "rule-2", PolicyVersion: "v2",
			}},
			wantState:  "NON_PERSON_ROUTE",
			wantStatus: StatusBlocked,
		},
		{
			name: "principal cannot read Matter",
			service: lifecycleAuthorityStub{resolution: authority.Resolution{
				Principal: authority.Principal{ID: "hidden", Kind: "PERSON"},
				Strategy: "DIRECT", RuleID: "rule-3", PolicyVersion: "v3",
			}},
			wantState:  "ROUTE_NOT_VISIBLE",
			wantStatus: StatusBlocked,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			projector := &MatterLifecycleProjector{Authority: test.service}
			assignment, err := projector.resolveAssignment(context.Background(), matter, requirement, "entity-1", now)
			if err != nil {
				t.Fatal(err)
			}
			if assignment.Status != test.wantStatus || assignment.RoutingState != test.wantState || assignment.PrincipalID != "" {
				t.Fatalf("unexpected assignment: %#v", assignment)
			}
		})
	}
}

func TestLifecycleAssignmentAcceptsOneVisibleDirectPerson(t *testing.T) {
	matter := continuity.Matter{
		TenantID: "bank", ID: "matter-1", Priority: 4,
		Scope: []byte(`{"access":"RESTRICTED","allowed_principal_ids":["allowed"]}`),
	}
	requirement := WorkRequirement{
		Responsibility: authority.ResponsibilityProposer,
		CommandName:    "matter.response.transition",
		Materiality:    4,
	}
	projector := &MatterLifecycleProjector{Authority: lifecycleAuthorityStub{resolution: authority.Resolution{
		Principal:     authority.Principal{ID: "allowed", Kind: "PERSON"},
		Strategy:      "DIRECT",
		RuleID:        "rule-direct",
		PolicyVersion: "v4",
	}}}
	assignment, err := projector.resolveAssignment(context.Background(), matter, requirement, "entity-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if assignment.Status != StatusReady || assignment.PrincipalID != "allowed" || assignment.RoutingState != "DIRECT" {
		t.Fatalf("unexpected direct assignment: %#v", assignment)
	}
}

func TestLifecycleAssignmentPropagatesUnexpectedAuthorityFailure(t *testing.T) {
	boom := errors.New("authority backend failed")
	projector := &MatterLifecycleProjector{Authority: lifecycleAuthorityStub{err: boom}}
	_, err := projector.resolveAssignment(context.Background(), continuity.Matter{TenantID: "bank", ID: "matter-1", Scope: []byte(`{}`)}, WorkRequirement{
		Responsibility: authority.ResponsibilityProposer,
		CommandName:    "matter.response.transition",
	}, "entity-1", time.Now().UTC())
	if !errors.Is(err, boom) {
		t.Fatalf("expected authority failure to propagate, got %v", err)
	}
}
