package today

import (
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

// FromWorkflowTasks projects principal-assigned workflow work into the actor-facing
// Today read model. It does not create recommendations or claim automated work;
// richer operator context must come from separately governed records.
func FromWorkflowTasks(tasks []workflow.Task) []AttentionItem {
	items := make([]AttentionItem, 0, len(tasks))
	for _, task := range tasks {
		if task.Status == workflow.StatusCompleted || task.Status == workflow.StatusCancelled {
			continue
		}
		context := task.Context
		targetType, targetID := workflowTarget(context)
		primaryAction := firstValue(context["primary_action"], task.Title)
		whyNow := firstValue(context["why_now"], context["material_conclusion"], workflowReason(task))
		items = append(items, AttentionItem{
			ID:                 "workflow_" + task.ID,
			Type:               firstValue(context["type"], targetType, "WORKFLOW_TASK"),
			Title:              task.Title,
			WhyNow:             whyNow,
			Scope:              firstValue(context["scope"], context["program"], context["population"], "Connected scope"),
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

func workflowTarget(context map[string]string) (string, string) {
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
		return "The assigned workflow step is blocked and still requires accountable handling."
	case workflow.StatusEscalated:
		return "The assigned workflow step has been escalated to your role."
	case workflow.StatusInProgress:
		return "This workflow step is already in progress and remains assigned to your role."
	default:
		return "This workflow step is ready and assigned to your role."
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
