package today

import (
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

// FromWorkflowTasks projects workflow work into the actor-facing Today read
// model without inventing recommendations or automated-work receipts.
func FromWorkflowTasks(tasks []workflow.Task) []AttentionItem {
	return projectWorkflowTasks(tasks, nil)
}

// FromWorkflowTasksForActor is the production-safe projection. It accepts only
// canonical actor-visible work and rechecks its source-domain visibility before
// exposing the item to the actor.
func FromWorkflowTasksForActor(tasks []workflow.Task, principalID string) []AttentionItem {
	return projectWorkflowTasks(tasks, func(task workflow.Task) bool {
		return workflow.ActorWorkVisibleTo(task, principalID)
	})
}

func projectWorkflowTasks(tasks []workflow.Task, include func(workflow.Task) bool) []AttentionItem {
	items := make([]AttentionItem, 0, len(tasks))
	for _, task := range tasks {
		if task.Status == workflow.StatusCompleted || task.Status == workflow.StatusCancelled {
			continue
		}
		if include != nil && !include(task) {
			continue
		}
		context := task.Context
		targetType, targetID := workflowTarget(task)
		primaryAction := firstValue(context["primary_action"], task.Title)
		whyNow := firstValue(context["why_now"], context["material_conclusion"], workflowReason(task))
		items = append(items, AttentionItem{
			ID:                 "workflow_" + task.ID,
			Type:               firstValue(context["type"], targetType, "WORKFLOW_TASK"),
			Title:              task.Title,
			WhyNow:             whyNow,
			Scope:              firstValue(context["scope"], context["program"], context["population"], "Scope not provided"),
			State:              humanize(string(task.Status)),
			Evidence:           firstValue(context["evidence"], context["evidence_state"], "Workflow assignment"),
			Owner:              firstValue(context["owner"], humanize(task.Responsibility)),
			DueAt:              workflowDueAt(task.DueAt),
			PrimaryAction:      primaryAction,
			ActionTargetType:   targetType,
			ActionTargetID:     targetID,
			InterventionClass:  workflowIntervention(task, targetType),
			MaterialConclusion: context["material_conclusion"],
			ChangeSummary:      context["change_summary"],
			Authority:          workflowAuthorityContext(task, targetType, targetID),
			Verification:       workflowVerificationPlan(task),
		})
	}
	return items
}

func workflowDueAt(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func workflowTarget(task workflow.Task) (string, string) {
	if (task.WorkflowKind == workflow.MatterActionWorkflowKind || task.WorkflowKind == workflow.MatterLifecycleWorkflowKind) && strings.TrimSpace(task.MatterID) != "" {
		return "MATTER", strings.TrimSpace(task.MatterID)
	}
	context := task.Context
	targetType := strings.ToUpper(strings.TrimSpace(context["action_target_type"]))
	targetID := strings.TrimSpace(context["action_target_id"])
	if targetID != "" {
		switch targetType {
		case "PROGRAM", "MATTER", "EVIDENCE_REQUEST":
			return targetType, targetID
		}
	}
	if value := strings.TrimSpace(context["evidence_request_id"]); value != "" {
		return "EVIDENCE_REQUEST", value
	}
	if value := strings.TrimSpace(context["matter_id"]); value != "" {
		return "MATTER", value
	}
	if value := strings.TrimSpace(context["program_id"]); value != "" {
		return "PROGRAM", value
	}
	return "", ""
}

func workflowAuthorityContext(task workflow.Task, targetType, targetID string) *AuthorityContext {
	if targetType == "" || targetID == "" {
		return nil
	}
	responsibility := canonicalResponsibility(task.Responsibility)
	if responsibility == "" {
		return nil
	}
	materiality := task.MatterPriority
	if materiality <= 0 {
		if raw := strings.TrimSpace(task.Context["materiality"]); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
				materiality = parsed
			}
		}
	}
	return &AuthorityContext{
		Responsibility: responsibility,
		DecisionType:   strings.TrimSpace(task.Context["decision_type"]),
		Materiality:    materiality,
	}
}

func workflowVerificationPlan(task workflow.Task) *VerificationPlan {
	context := task.Context
	if strings.TrimSpace(context["verification_contract_id"]) == "" {
		return nil
	}
	method := "Outcome review"
	if strings.EqualFold(strings.TrimSpace(context["verification_independent"]), "true") {
		method = "Independent outcome review"
	}
	var nextCheck *time.Time
	if task.DueAt != nil {
		value := task.DueAt.UTC()
		nextCheck = &value
	}
	return &VerificationPlan{
		State:           firstValue(context["verification_evidence_state"], "Outcome check ready"),
		ExpectedOutcome: strings.TrimSpace(context["verification_expected_outcome"]),
		Method:          method,
		NextCheckAt:     nextCheck,
	}
}

func canonicalResponsibility(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PERFORMER":
		return "PERFORMER"
	case "OWNER", "ACCOUNTABLE_OWNER":
		return "ACCOUNTABLE_OWNER"
	case "PROPOSER":
		return "PROPOSER"
	case "REVIEWER":
		return "REVIEWER"
	case "CHALLENGER", "INDEPENDENT_CHALLENGER":
		return "INDEPENDENT_CHALLENGER"
	case "AUTHORIZER", "APPROVER", "DECIDER", "DECISION_OWNER":
		return "AUTHORIZER"
	case "SIGNATORY":
		return "SIGNATORY"
	case "TRANSMITTER":
		return "TRANSMITTER"
	case "ACKNOWLEDGER", "ACKNOWLEDGEMENT_RECORDER":
		return "ACKNOWLEDGEMENT_RECORDER"
	case "ESCALATION_OWNER":
		return "ESCALATION_OWNER"
	default:
		return ""
	}
}

func workflowIntervention(task workflow.Task, targetType string) InterventionClass {
	if value := parseIntervention(task.Context["intervention_class"]); value != "" {
		return value
	}
	if task.Status == workflow.StatusEscalated {
		return InterventionEscalation
	}
	if targetType == "EVIDENCE_REQUEST" {
		return InterventionEvidenceException
	}
	switch strings.ToUpper(task.Responsibility) {
	case "AUTHORIZER", "APPROVER", "SIGNATORY":
		return InterventionAuthorization
	case "DECIDER", "DECISION_OWNER":
		return InterventionDecision
	case "TRANSMITTER", "ACKNOWLEDGER", "ACKNOWLEDGEMENT_RECORDER":
		return InterventionExternalRepresentation
	default:
		return InterventionReview
	}
}

func parseIntervention(value string) InterventionClass {
	candidate := InterventionClass(strings.ToUpper(strings.TrimSpace(value)))
	switch candidate {
	case InterventionReview, InterventionDecision, InterventionAuthorization, InterventionEvidenceException, InterventionEscalation, InterventionVerification, InterventionExternalRepresentation:
		return candidate
	default:
		return ""
	}
}

func workflowReason(task workflow.Task) string {
	switch task.Status {
	case workflow.StatusBlocked:
		return "This assigned step is blocked and needs action."
	case workflow.StatusEscalated:
		return "This step was escalated to your role."
	case workflow.StatusInProgress:
		return "This step is in progress and remains assigned to you."
	default:
		return "This step is ready and assigned to you."
	}
}

func firstValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func humanize(value string) string {
	parts := strings.Fields(strings.ReplaceAll(strings.ToLower(value), "_", " "))
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
