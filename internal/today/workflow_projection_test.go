package today

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func TestFromWorkflowTasksProjectsOnlyActiveAssignedWork(t *testing.T) {
	now := time.Now().UTC()
	due := now.Add(time.Hour)
	tasks := []workflow.Task{
		{
			ID: "review-1", TenantID: "bank", Responsibility: "AUTHORIZER", Title: "Approve remediation",
			Status: workflow.StatusReady, DueAt: &due,
			Context: map[string]string{
				"scope": "Retail Payments", "action_target_type": "MATTER", "action_target_id": "matter-1",
				"material_conclusion": "The remediation changes a material control conclusion.",
				"decision_type":       "RISK_ACCEPTANCE", "materiality": "5",
			},
		},
		{ID: "done-1", TenantID: "bank", Responsibility: "REVIEWER", Title: "Already completed", Status: workflow.StatusCompleted},
	}

	items := FromWorkflowTasks(tasks)
	if len(items) != 1 {
		t.Fatalf("expected one active intervention, got %d", len(items))
	}
	item := items[0]
	if item.InterventionClass != InterventionAuthorization {
		t.Fatalf("expected authorization intervention, got %s", item.InterventionClass)
	}
	if item.ActionTargetType != "MATTER" || item.ActionTargetID != "matter-1" {
		t.Fatalf("unexpected target: %s/%s", item.ActionTargetType, item.ActionTargetID)
	}
	if item.Authority == nil || item.Authority.Responsibility != "AUTHORIZER" || item.Authority.DecisionType != "RISK_ACCEPTANCE" || item.Authority.Materiality != 5 {
		t.Fatalf("unexpected authority context: %#v", item.Authority)
	}
	if item.Recommendation != nil || item.PreparedWork != nil {
		t.Fatal("workflow projection must not fabricate operator recommendation or prepared-work receipt")
	}
}

func TestFromWorkflowTasksUsesCanonicalTargetFallbacks(t *testing.T) {
	items := FromWorkflowTasks([]workflow.Task{{
		ID: "task-1", Responsibility: "REVIEWER", Title: "Review issue", Status: workflow.StatusReady,
		Context: map[string]string{"matter_id": "matter-42"},
	}})
	if len(items) != 1 || items[0].ActionTargetType != "MATTER" || items[0].ActionTargetID != "matter-42" {
		t.Fatalf("canonical matter target was not projected: %#v", items)
	}
	if items[0].Authority == nil || items[0].Authority.Responsibility != "REVIEWER" {
		t.Fatalf("canonical authority context was not projected: %#v", items[0].Authority)
	}
}

func TestFromWorkflowTasksRejectsUnknownActionTargets(t *testing.T) {
	items := FromWorkflowTasks([]workflow.Task{{
		ID: "task-1", Responsibility: "REVIEWER", Title: "Review task", Status: workflow.StatusReady,
		Context: map[string]string{"action_target_type": "SECRET_OBJECT", "action_target_id": "hidden"},
	}})
	if len(items) != 1 {
		t.Fatalf("expected one projected task, got %d", len(items))
	}
	if items[0].ActionTargetType != "" || items[0].ActionTargetID != "" || items[0].Authority != nil {
		t.Fatal("unknown action target must not become a navigable or authority-resolvable route")
	}
}

func TestFromWorkflowTasksForActorUsesCanonicalMatterMetadataAndVisibility(t *testing.T) {
	allowedScope := json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["actor-1"]}`)
	blockedScope := json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["someone-else"]}`)
	tasks := []workflow.Task{
		{
			ID: "allowed", TenantID: "bank", WorkflowKind: workflow.MatterActionWorkflowKind,
			MatterID: "matter-canonical", MatterPriority: 5, MatterScope: allowedScope,
			Responsibility: "ACCOUNTABLE_OWNER", PrincipalID: "actor-1", Title: "Handle material issue", Status: workflow.StatusReady,
			Context: map[string]string{"matter_id": "matter-stale", "action_target_type": "MATTER", "action_target_id": "matter-spoofed", "materiality": "1"},
		},
		{
			ID: "blocked", TenantID: "bank", WorkflowKind: workflow.MatterActionWorkflowKind,
			MatterID: "matter-hidden", MatterPriority: 4, MatterScope: blockedScope,
			Responsibility: "ACCOUNTABLE_OWNER", PrincipalID: "actor-1", Title: "Hidden issue", Status: workflow.StatusReady,
		},
		{
			ID: "legacy", TenantID: "bank", WorkflowKind: "REVIEW", MatterID: "matter-canonical", MatterPriority: 5, MatterScope: allowedScope,
			Responsibility: "REVIEWER", PrincipalID: "actor-1", Title: "Legacy task", Status: workflow.StatusReady,
		},
	}
	items := FromWorkflowTasksForActor(tasks, "actor-1")
	if len(items) != 1 {
		t.Fatalf("expected only one visible supported Matter Action, got %#v", items)
	}
	item := items[0]
	if item.ActionTargetType != "MATTER" || item.ActionTargetID != "matter-canonical" {
		t.Fatalf("Task context overrode canonical Matter target: %#v", item)
	}
	if item.Authority == nil || item.Authority.Materiality != 5 {
		t.Fatalf("canonical Matter priority was not used for authority inspection: %#v", item.Authority)
	}
}

func TestFromWorkflowTasksRecognizesLifecycleResponsibilities(t *testing.T) {
	for _, test := range []struct {
		responsibility string
		want           string
	}{
		{"PROPOSER", "PROPOSER"},
		{"TRANSMITTER", "TRANSMITTER"},
		{"ACKNOWLEDGEMENT_RECORDER", "ACKNOWLEDGEMENT_RECORDER"},
	} {
		items := FromWorkflowTasks([]workflow.Task{{
			ID: test.responsibility, Responsibility: test.responsibility, Title: "Lifecycle work", Status: workflow.StatusReady,
			Context: map[string]string{"action_target_type": "MATTER", "action_target_id": "matter-1"},
		}})
		if len(items) != 1 || items[0].Authority == nil || items[0].Authority.Responsibility != test.want {
			t.Fatalf("%s was not retained as a lifecycle responsibility: %#v", test.responsibility, items)
		}
	}
}
