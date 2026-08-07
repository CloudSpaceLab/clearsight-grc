package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const MatterResponseWorkflowKind = "MATTER_RESPONSE"

// WorkRequirement describes governed work that is safe to route. It does not
// choose a principal; current authority resolution remains the source of truth
// for assignment.
type WorkRequirement struct {
	Key               string
	WorkflowKind      string
	MatterID          string
	SubjectType       string
	SubjectID         string
	CommandName       string
	TargetStatus      string
	Responsibility    authority.Responsibility
	Materiality       int
	Title             string
	WhyNow            string
	InterventionClass string
	DueAt             *time.Time
}

// CompileMatterLifecycleWork returns only lifecycle work whose next governed
// transition is unambiguous from current canonical state. Ambiguous lifecycle
// states intentionally produce no requirement until an explicit policy selects
// a branch.
func CompileMatterLifecycleWork(aggregate continuity.MatterAggregate) []WorkRequirement {
	if aggregate.Matter.ID == "" || aggregate.Matter.Status == continuity.MatterClosed || aggregate.Matter.Status == continuity.MatterCancelled {
		return nil
	}
	requirements := make([]WorkRequirement, 0)
	for _, response := range aggregate.ResponsePackages {
		requirement, ok := responseWorkRequirement(aggregate.Matter, response)
		if ok {
			requirements = append(requirements, requirement)
		}
	}
	return requirements
}

func responseWorkRequirement(matter continuity.Matter, response continuity.ResponsePackage) (WorkRequirement, bool) {
	base := WorkRequirement{
		Key:          "response:" + response.ID,
		WorkflowKind: MatterResponseWorkflowKind,
		MatterID:     matter.ID,
		SubjectType:  "RESPONSE_PACKAGE",
		SubjectID:    response.ID,
		CommandName:  "matter.response.transition",
		DueAt:        matter.DueAt,
	}
	switch response.Status {
	case continuity.ResponseRejected:
		base.TargetStatus = string(continuity.ResponseDraft)
		base.Responsibility = authority.ResponsibilityProposer
		base.Materiality = maxInt(2, matter.Priority)
		base.Title = "Rework response package"
		base.WhyNow = "The response was rejected and must be revised before it can return to review."
		base.InterventionClass = "REVIEW"
		return base, true
	case continuity.ResponseTransmitted:
		base.TargetStatus = string(continuity.ResponseAcknowledged)
		base.Responsibility = authority.ResponsibilityAcknowledger
		base.Materiality = maxInt(3, matter.Priority)
		base.Title = "Record response acknowledgement"
		base.WhyNow = "The approved response was transmitted and receipt acknowledgement is the only remaining response transition."
		base.InterventionClass = "EXTERNAL_REPRESENTATION"
		return base, true
	default:
		return WorkRequirement{}, false
	}
}

func (r WorkRequirement) Validate() error {
	if strings.TrimSpace(r.Key) == "" || strings.TrimSpace(r.WorkflowKind) == "" || strings.TrimSpace(r.MatterID) == "" || strings.TrimSpace(r.SubjectType) == "" || strings.TrimSpace(r.SubjectID) == "" || strings.TrimSpace(r.CommandName) == "" || strings.TrimSpace(r.TargetStatus) == "" || strings.TrimSpace(string(r.Responsibility)) == "" || strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.WhyNow) == "" {
		return fmt.Errorf("work requirement is incomplete")
	}
	if r.Materiality < 0 || r.Materiality > 5 {
		return fmt.Errorf("work requirement materiality must be between 0 and 5")
	}
	return nil
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
