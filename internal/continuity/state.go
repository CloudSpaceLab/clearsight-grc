package continuity

import (
	"fmt"
	"sort"
	"time"
)

func deriveProgramState(aggregate ProgramAggregate, openMatters int, now time.Time) ProgramStateSnapshot {
	dimensions := ComplianceDimensions{
		Interpretation:         StateUnknown,
		Applicability:          StateUnknown,
		ControlDesign:          StateUnknown,
		Implementation:         StateUnknown,
		EvidenceSufficiency:    StateUnknown,
		OperatingEffectiveness: StateUnknown,
		Exception:              StateCurrent,
		Assurance:              StateUnknown,
		Deadline:               StateCurrent,
		SourceQuality:          StateUnknown,
	}
	reasons := make([]StateReason, 0, 12)

	approvedRequirements := make(map[string]Requirement)
	for _, requirement := range aggregate.Requirements {
		if requirement.Status == RequirementApproved && (requirement.EffectiveUntil == nil || now.Before(*requirement.EffectiveUntil)) {
			approvedRequirements[requirement.ID] = requirement
		}
	}
	if len(approvedRequirements) == 0 {
		reasons = append(reasons, StateReason{Code: "NO_APPROVED_REQUIREMENTS", Summary: "No approved requirements are in scope."})
	} else {
		dimensions.Interpretation = StateCurrent
	}

	latestApplicability := make(map[string]Applicability)
	for _, item := range aggregate.Applicability {
		current, ok := latestApplicability[item.RequirementID]
		if !ok || item.EffectiveFrom.After(current.EffectiveFrom) {
			latestApplicability[item.RequirementID] = item
		}
	}
	applicableCount := 0
	notApplicableCount := 0
	potentialCount := 0
	for requirementID, requirement := range approvedRequirements {
		item, ok := latestApplicability[requirementID]
		if !ok {
			potentialCount++
			reasons = append(reasons, StateReason{Code: "APPLICABILITY_NOT_RECORDED", Summary: fmt.Sprintf("Applicability has not been confirmed for %s.", requirement.Title), ObjectType: "REQUIREMENT", ObjectID: requirementID})
			continue
		}
		switch item.Status {
		case ApplicabilityApplicable:
			applicableCount++
		case ApplicabilityPartial, ApplicabilityPotential, ApplicabilityLater:
			potentialCount++
			reasons = append(reasons, StateReason{Code: "APPLICABILITY_REVIEW_NEEDED", Summary: fmt.Sprintf("Applicability still needs review for %s.", requirement.Title), ObjectType: "REQUIREMENT", ObjectID: requirementID})
		case ApplicabilityNotApplicable, ApplicabilitySuperseded:
			notApplicableCount++
		}
	}
	switch {
	case len(approvedRequirements) == 0:
		dimensions.Applicability = StateUnknown
	case potentialCount > 0:
		dimensions.Applicability = StateUnderReview
	case applicableCount == 0 && notApplicableCount == len(approvedRequirements):
		dimensions.Applicability = StateNotApplicable
	default:
		dimensions.Applicability = StateCurrent
	}

	linkedRequirements := make(map[string]bool)
	for _, link := range aggregate.RequirementControlLinks {
		linkedRequirements[link.RequirementID] = true
	}
	missingControlLinks := 0
	for requirementID := range approvedRequirements {
		item, ok := latestApplicability[requirementID]
		if !ok || (item.Status != ApplicabilityApplicable && item.Status != ApplicabilityPartial) {
			continue
		}
		if !linkedRequirements[requirementID] {
			missingControlLinks++
			reasons = append(reasons, StateReason{Code: "CONTROL_NOT_MAPPED", Summary: "An applicable requirement has no linked control implementation.", ObjectType: "REQUIREMENT", ObjectID: requirementID})
		}
	}
	switch {
	case applicableCount == 0 && potentialCount == 0:
		dimensions.ControlDesign = StateNotApplicable
	case missingControlLinks > 0:
		dimensions.ControlDesign = StateGapIdentified
	case applicableCount > 0:
		dimensions.ControlDesign = StateCurrent
	default:
		dimensions.ControlDesign = StateUnknown
	}

	implementationStates := make(map[string]ControlImplementationStatus)
	for _, implementation := range aggregate.ControlImplementations {
		implementationStates[implementation.ID] = implementation.Status
	}
	pendingImplementations := 0
	inactiveImplementations := 0
	activeImplementations := 0
	for _, link := range aggregate.RequirementControlLinks {
		state := implementationStates[link.ImplementationID]
		switch state {
		case ImplementationImplemented:
			activeImplementations++
		case ImplementationPlanned, ImplementationInProgress:
			pendingImplementations++
		case ImplementationInactive, ImplementationRetired:
			inactiveImplementations++
		default:
			pendingImplementations++
		}
	}
	switch {
	case missingControlLinks > 0 || inactiveImplementations > 0:
		dimensions.Implementation = StateGapIdentified
	case pendingImplementations > 0:
		dimensions.Implementation = StateImplementationPending
	case activeImplementations > 0:
		dimensions.Implementation = StateCurrent
	case dimensions.ControlDesign == StateNotApplicable:
		dimensions.Implementation = StateNotApplicable
	default:
		dimensions.Implementation = StateUnknown
	}
	if pendingImplementations > 0 {
		reasons = append(reasons, StateReason{Code: "IMPLEMENTATION_PENDING", Summary: fmt.Sprintf("%d control implementation(s) are not yet operating.", pendingImplementations)})
	}

	latestAssessment := make(map[string]EvidenceAssessment)
	for _, assessment := range aggregate.EvidenceAssessments {
		current, ok := latestAssessment[assessment.ContractID]
		if !ok || assessment.AssessedAt.After(current.AssessedAt) {
			latestAssessment[assessment.ContractID] = assessment
		}
	}
	activeContracts := 0
	insufficientContracts := 0
	contradictedContracts := 0
	expiredContracts := 0
	for _, contract := range aggregate.EvidenceContracts {
		if contract.Status != EvidenceContractActive {
			continue
		}
		activeContracts++
		assessment, ok := latestAssessment[contract.ID]
		if !ok {
			insufficientContracts++
			reasons = append(reasons, StateReason{Code: "EVIDENCE_NOT_ASSESSED", Summary: fmt.Sprintf("Evidence has not been assessed for %s.", contract.Name), ObjectType: "EVIDENCE_CONTRACT", ObjectID: contract.ID})
			continue
		}
		if assessment.ValidUntil != nil && !now.Before(*assessment.ValidUntil) {
			expiredContracts++
			reasons = append(reasons, StateReason{Code: "EVIDENCE_EXPIRED", Summary: fmt.Sprintf("Evidence is out of date for %s.", contract.Name), ObjectType: "EVIDENCE_CONTRACT", ObjectID: contract.ID})
			continue
		}
		if assessment.Coverage < contract.MinimumCoverage {
			insufficientContracts++
			reasons = append(reasons, StateReason{Code: "EVIDENCE_COVERAGE_LOW", Summary: fmt.Sprintf("Evidence coverage is below the required level for %s.", contract.Name), ObjectType: "EVIDENCE_CONTRACT", ObjectID: contract.ID})
		}
		switch assessment.Conclusion {
		case EvidenceContradicted:
			contradictedContracts++
			reasons = append(reasons, StateReason{Code: "EVIDENCE_CONTRADICTED", Summary: fmt.Sprintf("Evidence conflicts for %s.", contract.Name), ObjectType: "EVIDENCE_CONTRACT", ObjectID: contract.ID})
		case EvidenceUnsupported, EvidenceIndeterminate, EvidencePartiallySupported:
			insufficientContracts++
		case EvidenceExpired:
			expiredContracts++
		}
	}
	switch {
	case activeContracts == 0 && applicableCount > 0:
		dimensions.EvidenceSufficiency = StateEvidenceInsufficient
		reasons = append(reasons, StateReason{Code: "NO_EVIDENCE_CONTRACTS", Summary: "Applicable requirements do not yet have evidence checks."})
	case contradictedContracts > 0:
		dimensions.EvidenceSufficiency = StateGapIdentified
	case insufficientContracts > 0 || expiredContracts > 0:
		dimensions.EvidenceSufficiency = StateEvidenceInsufficient
	case activeContracts > 0:
		dimensions.EvidenceSufficiency = StateCurrent
	case dimensions.Applicability == StateNotApplicable:
		dimensions.EvidenceSufficiency = StateNotApplicable
	default:
		dimensions.EvidenceSufficiency = StateUnknown
	}

	dimensions.OperatingEffectiveness = StateUnknown
	if dimensions.Implementation == StateCurrent && dimensions.EvidenceSufficiency == StateCurrent {
		dimensions.OperatingEffectiveness = StateCurrent
	}
	if openMatters > 0 {
		dimensions.Exception = StateAtRisk
		reasons = append(reasons, StateReason{Code: "OPEN_MATTERS", Summary: fmt.Sprintf("%d open issue(s) or change(s) affect this program.", openMatters)})
	}
	if aggregate.Program.Status == ProgramPaused {
		dimensions.Assurance = StateUnderReview
	}
	latestSourceTrigger := Trigger{}
	for _, trigger := range aggregate.Triggers {
		if (trigger.Type == "SOURCE_DEGRADED" || trigger.Type == "SOURCE_RECOVERED") && (latestSourceTrigger.ID == "" || trigger.ObservedAt.After(latestSourceTrigger.ObservedAt)) {
			latestSourceTrigger = trigger
		}
	}
	if latestSourceTrigger.Type == "SOURCE_DEGRADED" {
		dimensions.SourceQuality = StateAtRisk
		reasons = append(reasons, StateReason{Code: "SOURCE_QUALITY_ISSUE", Summary: "A source used by this program is stale, degraded or unavailable.", ObjectType: latestSourceTrigger.SubjectType, ObjectID: latestSourceTrigger.SubjectID})
	} else if latestSourceTrigger.Type == "SOURCE_RECOVERED" {
		dimensions.SourceQuality = StateCurrent
	}
	if aggregate.Program.EffectiveUntil != nil && !now.Before(*aggregate.Program.EffectiveUntil) {
		dimensions.Deadline = StateOverdue
		reasons = append(reasons, StateReason{Code: "PROGRAM_PERIOD_ENDED", Summary: "The current program period has ended."})
	}

	overall := chooseOverallState(dimensions)
	sort.SliceStable(reasons, func(i, j int) bool { return reasons[i].Code < reasons[j].Code })
	return ProgramStateSnapshot{
		TenantID:        aggregate.Program.TenantID,
		ProgramID:       aggregate.Program.ID,
		Overall:         overall,
		Dimensions:      dimensions,
		Reasons:         reasons,
		OpenMatterCount: openMatters,
		GeneratedAt:     now.UTC(),
		ProgramVersion:  aggregate.Program.Version,
	}
}

func chooseOverallState(dimensions ComplianceDimensions) ProgramState {
	states := []ProgramState{
		dimensions.Interpretation,
		dimensions.Applicability,
		dimensions.ControlDesign,
		dimensions.Implementation,
		dimensions.EvidenceSufficiency,
		dimensions.OperatingEffectiveness,
		dimensions.Exception,
		dimensions.Assurance,
		dimensions.Deadline,
		dimensions.SourceQuality,
	}
	for _, candidate := range []ProgramState{StateOverdue, StateGapIdentified, StateEvidenceInsufficient, StateImplementationPending, StateAtRisk, StateUnderReview} {
		for _, state := range states {
			if state == candidate {
				return candidate
			}
		}
	}
	core := []ProgramState{dimensions.Interpretation, dimensions.Applicability, dimensions.ControlDesign, dimensions.Implementation, dimensions.EvidenceSufficiency}
	notApplicable := 0
	for _, state := range core {
		if state == StateUnknown {
			return StateUnknown
		}
		if state == StateNotApplicable {
			notApplicable++
		}
	}
	if notApplicable == len(core)-1 || dimensions.Applicability == StateNotApplicable {
		return StateNotApplicable
	}
	return StateCurrent
}

func programStateLabel(state ProgramState) string {
	switch state {
	case StateCurrent:
		return "Up to date"
	case StateAtRisk:
		return "Needs attention"
	case StateGapIdentified:
		return "Gap found"
	case StateEvidenceInsufficient:
		return "Evidence incomplete"
	case StateImplementationPending:
		return "Change in progress"
	case StateOverdue:
		return "Overdue"
	case StateUnderReview:
		return "Under review"
	case StateNotApplicable:
		return "Not applicable"
	default:
		return "Not assessed"
	}
}

func matterStatusLabel(status MatterStatus) string {
	switch status {
	case MatterDraft:
		return "Draft"
	case MatterInitialReview:
		return "Initial review"
	case MatterAssessment:
		return "Reviewing impact"
	case MatterDecisionRequired:
		return "Decision needed"
	case MatterActionsInProgress:
		return "Work in progress"
	case MatterResponsePreparation:
		return "Preparing response"
	case MatterVerification:
		return "Confirming outcome"
	case MatterClosed:
		return "Closed"
	case MatterCancelled:
		return "Cancelled"
	default:
		return "Status unavailable"
	}
}

func matterTypeLabel(value MatterType) string {
	switch value {
	case MatterRegulatoryChange:
		return "Regulatory change"
	case MatterSupervisoryFinding:
		return "Supervisory finding"
	case MatterAuthorityRequest:
		return "Authority request"
	case MatterRiskSituation:
		return "Risk issue"
	case MatterControlGap:
		return "Control gap"
	case MatterAuditFinding:
		return "Audit finding"
	case MatterException:
		return "Exception"
	case MatterIncident:
		return "Incident"
	case MatterOperationalLoss:
		return "Operational loss"
	case MatterDataBreach:
		return "Data breach"
	case MatterVendorDeficiency:
		return "Vendor issue"
	case MatterCustomerConcern:
		return "Customer concern"
	case MatterOverdueObligation:
		return "Overdue obligation"
	case MatterFailedVerification:
		return "Outcome check failed"
	case MatterEvidenceContradiction:
		return "Conflicting evidence"
	case MatterKRIBreach:
		return "Risk indicator breach"
	default:
		return "Matter"
	}
}

func matterNextAction(status MatterStatus) string {
	switch status {
	case MatterDraft:
		return "Start initial review"
	case MatterInitialReview:
		return "Confirm scope and owner"
	case MatterAssessment:
		return "Review impact and options"
	case MatterDecisionRequired:
		return "Record decision"
	case MatterActionsInProgress:
		return "Complete assigned work"
	case MatterResponsePreparation:
		return "Prepare response"
	case MatterVerification:
		return "Confirm whether the outcome was achieved"
	case MatterClosed:
		return "View history"
	case MatterCancelled:
		return "View cancellation reason"
	default:
		return "Review matter"
	}
}

func decorateProgram(aggregate ProgramAggregate) ProgramAggregate {
	switch aggregate.Program.Status {
	case ProgramDraft:
		aggregate.StateLabel = "Setup in progress"
	case ProgramPaused:
		aggregate.StateLabel = "Paused"
	case ProgramRetired:
		aggregate.StateLabel = "Ended"
	default:
		if aggregate.CurrentState == nil {
			aggregate.StateLabel = programStateLabel(StateUnknown)
		} else {
			aggregate.StateLabel = programStateLabel(aggregate.CurrentState.Overall)
		}
	}
	return aggregate
}

func decorateMatter(aggregate MatterAggregate) MatterAggregate {
	aggregate.TypeLabel = matterTypeLabel(aggregate.Matter.Type)
	aggregate.StatusLabel = matterStatusLabel(aggregate.Matter.Status)
	aggregate.NextAction = matterNextAction(aggregate.Matter.Status)
	return aggregate
}
