package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

type todayAdminStub struct {
	overview access.OperationalStatus
	calls    int
}

func (s *todayAdminStub) OperationalStatus(context.Context, string, int) (access.OperationalStatus, error) {
	s.calls++
	return s.overview, nil
}

type todayJobsStub struct {
	snapshot operations.Snapshot
	calls    int
}

func (s *todayJobsStub) Snapshot(context.Context, string, int) (operations.Snapshot, error) {
	s.calls++
	return s.snapshot, nil
}

func TestActorTodayServiceReturnsOnlyCurrentStoredActorWork(t *testing.T) {
	now := time.Now().UTC()
	due := now.Add(time.Hour)
	tasks := []workflow.Task{
		{
			ID: "assigned", TenantID: "bank", WorkflowID: "request-workflow", StepKey: "provide-evidence",
			Responsibility: "PERFORMER", PrincipalID: "staff-1", Title: "Provide vendor evidence", Status: workflow.StatusReady,
			DueAt: &due, Context: map[string]string{"action_target_type": "EVIDENCE_REQUEST", "action_target_id": "request-1"},
			WorkflowKind: workflow.EvidenceRequestWorkflowKind, EvidenceRequestID: "request-1", EvidenceRecipientID: "staff-1", EvidenceSubjectVisible: true,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "other-person", TenantID: "bank", WorkflowID: "other-workflow", StepKey: "provide-evidence",
			Responsibility: "PERFORMER", PrincipalID: "staff-2", Title: "Other person's task", Status: workflow.StatusReady,
			WorkflowKind: workflow.EvidenceRequestWorkflowKind, EvidenceRequestID: "request-2", EvidenceRecipientID: "staff-2", EvidenceSubjectVisible: true,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "completed", TenantID: "bank", WorkflowID: "completed-workflow", StepKey: "provide-evidence",
			Responsibility: "PERFORMER", PrincipalID: "staff-1", Title: "Completed task", Status: workflow.StatusCompleted,
			WorkflowKind: workflow.EvidenceRequestWorkflowKind, EvidenceRequestID: "request-3", EvidenceRecipientID: "staff-1", EvidenceSubjectVisible: true,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
	service := actorTodayService(workflow.NewService(workflow.NewMemoryRepository(tasks)), nil, nil)
	items, err := service.ListFor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "staff-1"})
	if err != nil {
		t.Fatalf("list Today work: %v", err)
	}
	if len(items) != 1 || items[0].ActionTargetID != "request-1" || items[0].Title != "Provide vendor evidence" {
		t.Fatalf("Today items = %#v, want exact active assignment", items)
	}
}

func TestActorTodayServiceProjectsEveryAssignedOperationalResponsibility(t *testing.T) {
	now := time.Now().UTC()
	responsibilities := []string{"PERFORMER", "ACCOUNTABLE_OWNER", "PROPOSER", "REVIEWER", "INDEPENDENT_CHALLENGER", "AUTHORIZER", "SIGNATORY", "TRANSMITTER", "ACKNOWLEDGEMENT_RECORDER", "ESCALATION_OWNER"}
	tasks := make([]workflow.Task, 0, len(responsibilities))
	for _, responsibility := range responsibilities {
		tasks = append(tasks, workflow.Task{
			ID: "task-" + responsibility, TenantID: "bank", WorkflowID: "workflow-" + responsibility,
			WorkflowKind: workflow.MatterLifecycleWorkflowKind, MatterID: "matter-" + responsibility,
			MatterScope: json.RawMessage(`{"access":"INTERNAL"}`), MatterPriority: 4,
			StepKey: "step-" + responsibility, Responsibility: responsibility, PrincipalID: "actor-1",
			Title: "Complete " + responsibility + " work", Status: workflow.StatusReady,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		})
	}

	items, err := actorTodayService(workflow.NewService(workflow.NewMemoryRepository(tasks)), nil, nil).ListFor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor-1"})
	if err != nil {
		t.Fatalf("list operational work: %v", err)
	}
	if len(items) != len(responsibilities) {
		t.Fatalf("items=%d, want one exact item for each assigned responsibility", len(items))
	}
	seen := map[string]bool{}
	for _, item := range items {
		if item.Authority == nil || item.ActionTargetType != "MATTER" || item.ActionTargetID == "" {
			t.Fatalf("assigned work lost authority or target context: %#v", item)
		}
		seen[item.Authority.Responsibility] = true
	}
	for _, responsibility := range responsibilities {
		if !seen[responsibility] {
			t.Fatalf("assigned responsibility %s did not appear in Today", responsibility)
		}
	}
}

func TestActorTodayServiceAddsOnlyCapabilityScopedAdministratorExceptions(t *testing.T) {
	workflowService := workflow.NewService(workflow.NewMemoryRepository(nil))
	admin := &todayAdminStub{overview: access.OperationalStatus{
		SourceExceptions: []access.SCIMSourceSummary{{ID: "source-1", Code: "BANK_DIRECTORY", Status: "REVOKED"}},
		Escalation:       access.EscalationRuntimeStatus{Unresolved24h: 2, FailedTimers: 1},
	}}
	jobs := &todayJobsStub{snapshot: operations.Snapshot{Queues: []operations.QueueSummary{{Queue: "workflow", Terminal: 3}}}}
	actor := identity.Actor{
		TenantID: "bank", LegalEntityID: "entity", PrincipalID: "system-admin",
		PermissionCodes: []string{identity.PermissionIdentityRead, identity.PermissionPlatformOperationsRead},
	}

	items, err := actorTodayService(workflowService, admin, jobs).ListFor(context.Background(), actor)
	if err != nil {
		t.Fatalf("list administrator Today work: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("administrator items = %#v, want source, unresolved routing, failed escalation and failed jobs", items)
	}
	for _, item := range items {
		if item.ActionTargetType != "CONFIGURE" || item.ActionTargetID == "" {
			t.Fatalf("administrator item has no exact Configure target: %#v", item)
		}
	}
	if admin.calls != 1 || jobs.calls != 1 {
		t.Fatalf("administrator sources called admin=%d jobs=%d, want one each", admin.calls, jobs.calls)
	}

	admin.calls, jobs.calls = 0, 0
	ordinary := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "employee", PermissionCodes: []string{identity.PermissionOversightRead}}
	items, err = actorTodayService(workflowService, admin, jobs).ListFor(context.Background(), ordinary)
	if err != nil {
		t.Fatalf("list ordinary Today work: %v", err)
	}
	if len(items) != 0 || admin.calls != 0 || jobs.calls != 0 {
		t.Fatalf("ordinary actor received administrator work: items=%#v admin_calls=%d job_calls=%d", items, admin.calls, jobs.calls)
	}
}
