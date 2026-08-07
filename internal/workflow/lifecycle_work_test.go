package workflow

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestCompileMatterLifecycleWorkOnlyEmitsUnambiguousResponseSteps(t *testing.T) {
	matter := continuity.Matter{ID: "matter-1", Status: continuity.MatterResponsePreparation, Priority: 5}
	for _, test := range []struct {
		status continuity.ResponseStatus
		want   bool
		resp   authority.Responsibility
		target continuity.ResponseStatus
	}{
		{continuity.ResponseDraft, false, "", ""},
		{continuity.ResponseInReview, false, "", ""},
		{continuity.ResponseApproved, false, "", ""},
		{continuity.ResponseRejected, true, authority.ResponsibilityProposer, continuity.ResponseDraft},
		{continuity.ResponseTransmitted, true, authority.ResponsibilityAcknowledger, continuity.ResponseAcknowledged},
		{continuity.ResponseAcknowledged, false, "", ""},
		{continuity.ResponseWithdrawn, false, "", ""},
	} {
		aggregate := continuity.MatterAggregate{Matter: matter, ResponsePackages: []continuity.ResponsePackage{{ID: "response-1", MatterID: matter.ID, Status: test.status}}}
		got := CompileMatterLifecycleWork(aggregate)
		if !test.want {
			if len(got) != 0 {
				t.Fatalf("%s must remain unprojected because its next transition is not an explicit unique work step: %#v", test.status, got)
			}
			continue
		}
		if len(got) != 1 {
			t.Fatalf("%s: expected one work requirement, got %#v", test.status, got)
		}
		requirement := got[0]
		if requirement.Responsibility != test.resp || requirement.TargetStatus != string(test.target) || requirement.Materiality != 5 {
			t.Fatalf("%s: unexpected requirement %#v", test.status, requirement)
		}
		if err := requirement.Validate(); err != nil {
			t.Fatalf("%s: invalid compiled requirement: %v", test.status, err)
		}
	}
}

func TestCompileMatterLifecycleWorkDoesNotInventDecisionAssignments(t *testing.T) {
	aggregate := continuity.MatterAggregate{
		Matter: continuity.Matter{ID: "matter-1", Status: continuity.MatterDecisionRequired, Priority: 4},
		Decisions: []continuity.Decision{
			{ID: "decision-proposed", Type: "POSITION", Status: continuity.DecisionProposed},
			{ID: "decision-review", Type: "RISK", Status: continuity.DecisionInReview},
			{ID: "decision-challenged", Type: "EXCEPTION", Status: continuity.DecisionChallenged},
		},
	}
	if got := CompileMatterLifecycleWork(aggregate); len(got) != 0 {
		t.Fatalf("decision state alone must not choose reviewer/challenger/authorizer work: %#v", got)
	}
}

func TestCompileMatterLifecycleWorkStopsForClosedMatter(t *testing.T) {
	aggregate := continuity.MatterAggregate{
		Matter: continuity.Matter{ID: "matter-1", Status: continuity.MatterClosed, Priority: 5},
		ResponsePackages: []continuity.ResponsePackage{{ID: "response-1", Status: continuity.ResponseTransmitted}},
	}
	if got := CompileMatterLifecycleWork(aggregate); len(got) != 0 {
		t.Fatalf("closed matter must not emit actor work: %#v", got)
	}
}
