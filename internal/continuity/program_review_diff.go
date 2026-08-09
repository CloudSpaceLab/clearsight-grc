package continuity

import (
	"encoding/json"
	"fmt"
	"strings"
)

func deriveProgramReviewChanges(aggregate ProgramAggregate, baseline, current ProgramStateSnapshot, events []Event, newReasons []StateReason) []ProgramReviewChange {
	changes := make([]ProgramReviewChange, 0, 16)
	seen := make(map[string]struct{})
	appendChange := func(change ProgramReviewChange) {
		key := change.Kind + "\x00" + change.Summary + "\x00" + change.ObjectType + "\x00" + change.ObjectID
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		changes = append(changes, change)
	}
	if baseline.Overall != current.Overall {
		appendChange(ProgramReviewChange{Kind: "STATE", Summary: fmt.Sprintf("Overall status changed from %s to %s.", humanProgramState(baseline.Overall), humanProgramState(current.Overall))})
	}
	for _, change := range dimensionChanges(baseline.Dimensions, current.Dimensions) {
		appendChange(change)
	}
	if baseline.OpenMatterCount != current.OpenMatterCount {
		delta := current.OpenMatterCount - baseline.OpenMatterCount
		if delta > 0 {
			appendChange(ProgramReviewChange{Kind: "ISSUE", Summary: fmt.Sprintf("%d additional open issue(s) now affect this Program.", delta)})
		} else {
			appendChange(ProgramReviewChange{Kind: "ISSUE", Summary: fmt.Sprintf("%d open issue(s) affecting this Program were resolved or removed.", -delta)})
		}
	}
	for _, reason := range newReasons {
		appendChange(ProgramReviewChange{Kind: "EXCEPTION", Summary: reason.Summary, ObjectType: reason.ObjectType, ObjectID: reason.ObjectID})
	}
	for _, event := range events {
		if change, ok := programEventChange(aggregate, event); ok {
			appendChange(change)
		}
	}
	return changes
}

func programEventChange(aggregate ProgramAggregate, event Event) (ProgramReviewChange, bool) {
	switch event.Type {
	case EventProgramStatusChanged:
		var value Program
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "PROGRAM", Summary: "Operating status changed to " + humanToken(string(value.Status)) + ".", ObjectType: "PROGRAM", ObjectID: value.ID}, true
		}
	case EventRequirementAdded:
		var value Requirement
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "REQUIREMENT", Summary: "Requirement changed: " + value.Title + ".", ObjectType: "REQUIREMENT", ObjectID: value.ID}, true
		}
	case EventApplicabilityDetermined:
		var value Applicability
		if json.Unmarshal(event.Payload, &value) == nil {
			label := requirementTitle(aggregate.Requirements, value.RequirementID)
			return ProgramReviewChange{Kind: "APPLICABILITY", Summary: fmt.Sprintf("Applicability for %s is now %s.", label, humanToken(string(value.Status))), ObjectType: "REQUIREMENT", ObjectID: value.RequirementID}, true
		}
	case EventControlObjectiveAdded:
		var value ControlObjective
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "SAFEGUARD", Summary: "Control objective changed: " + value.Name + ".", ObjectType: "CONTROL_OBJECTIVE", ObjectID: value.ID}, true
		}
	case EventControlImplementationAdded:
		var value ControlImplementation
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "SAFEGUARD", Summary: "Safeguard changed: " + value.Name + ".", ObjectType: "CONTROL_IMPLEMENTATION", ObjectID: value.ID}, true
		}
	case EventRequirementControlLinked:
		var value RequirementControlLink
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "MAPPING", Summary: "A requirement-to-safeguard mapping changed.", ObjectType: "REQUIREMENT", ObjectID: value.RequirementID}, true
		}
	case EventEvidenceContractAdded:
		var value EvidenceContract
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "EVIDENCE", Summary: "Evidence check changed: " + value.Name + ".", ObjectType: "EVIDENCE_CONTRACT", ObjectID: value.ID}, true
		}
	case EventEvidenceAssessmentRecorded:
		var value EvidenceAssessment
		if json.Unmarshal(event.Payload, &value) == nil {
			label := evidenceContractName(aggregate.EvidenceContracts, value.ContractID)
			return ProgramReviewChange{Kind: "EVIDENCE", Summary: fmt.Sprintf("Evidence for %s was assessed as %s.", label, humanToken(string(value.Conclusion))), ObjectType: "EVIDENCE_CONTRACT", ObjectID: value.ContractID}, true
		}
	case EventProgramTriggerRecorded:
		var value Trigger
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "CHANGE", Summary: "Observed change: " + humanToken(value.Type) + ".", ObjectType: value.SubjectType, ObjectID: value.SubjectID}, true
		}
	}
	return ProgramReviewChange{}, false
}

func dimensionChanges(before, after ComplianceDimensions) []ProgramReviewChange {
	values := []struct {
		name string
		old  ProgramState
		new  ProgramState
	}{
		{"Interpretation", before.Interpretation, after.Interpretation},
		{"Applicability", before.Applicability, after.Applicability},
		{"Control design", before.ControlDesign, after.ControlDesign},
		{"Implementation", before.Implementation, after.Implementation},
		{"Evidence sufficiency", before.EvidenceSufficiency, after.EvidenceSufficiency},
		{"Operating effectiveness", before.OperatingEffectiveness, after.OperatingEffectiveness},
		{"Exception", before.Exception, after.Exception},
		{"Assurance", before.Assurance, after.Assurance},
		{"Deadline", before.Deadline, after.Deadline},
		{"Source quality", before.SourceQuality, after.SourceQuality},
	}
	result := make([]ProgramReviewChange, 0, 4)
	for _, value := range values {
		if value.old == value.new {
			continue
		}
		result = append(result, ProgramReviewChange{Kind: "STATE", Summary: fmt.Sprintf("%s changed from %s to %s.", value.name, humanProgramState(value.old), humanProgramState(value.new))})
	}
	return result
}

func diffStateReasons(before, after []StateReason) ([]StateReason, []StateReason) {
	beforeByKey := make(map[string]StateReason, len(before))
	afterByKey := make(map[string]StateReason, len(after))
	for _, reason := range before {
		beforeByKey[stateReasonKey(reason)] = reason
	}
	for _, reason := range after {
		afterByKey[stateReasonKey(reason)] = reason
	}
	added := make([]StateReason, 0)
	resolved := make([]StateReason, 0)
	for key, reason := range afterByKey {
		if _, exists := beforeByKey[key]; !exists {
			added = append(added, reason)
		}
	}
	for key, reason := range beforeByKey {
		if _, exists := afterByKey[key]; !exists {
			resolved = append(resolved, reason)
		}
	}
	sortStateReasons(added)
	sortStateReasons(resolved)
	return added, resolved
}

func sortStateReasons(values []StateReason) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && stateReasonKey(values[j]) < stateReasonKey(values[j-1]); j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func stateReasonKey(reason StateReason) string {
	return strings.ToUpper(strings.TrimSpace(reason.Code)) + "\x00" + strings.ToUpper(strings.TrimSpace(reason.ObjectType)) + "\x00" + strings.TrimSpace(reason.ObjectID)
}

func limitReasons(values []StateReason, limit int) []StateReason {
	if len(values) <= limit {
		return append([]StateReason(nil), values...)
	}
	return append([]StateReason(nil), values[:limit]...)
}

func requirementTitle(values []Requirement, id string) string {
	for _, value := range values {
		if value.ID == id && strings.TrimSpace(value.Title) != "" {
			return value.Title
		}
	}
	return "the requirement"
}

func evidenceContractName(values []EvidenceContract, id string) string {
	for _, value := range values {
		if value.ID == id && strings.TrimSpace(value.Name) != "" {
			return value.Name
		}
	}
	return "the evidence check"
}

func humanProgramState(value ProgramState) string {
	return strings.ToLower(strings.ReplaceAll(string(value), "_", " "))
}

func humanToken(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", " "))
}
