package workflow

import (
	"encoding/json"
	"testing"
)

func TestAssignedActorCanSeeTheirRestrictedMatterWork(t *testing.T) {
	task := Task{
		TenantID: "bank", PrincipalID: "assignee", WorkflowKind: MatterActionWorkflowKind, MatterID: "matter-1",
		MatterScope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["cro"]}`),
	}
	if !ActorWorkVisibleTo(task, "assignee") {
		t.Fatal("canonical assignment did not grant visibility to the assigned actor")
	}
	if ActorWorkVisibleTo(task, "other") {
		t.Fatal("unassigned actor gained restricted Matter work visibility")
	}
}

func TestMemoryActorWorkReadAppliesExactEntityAndActorVisibility(t *testing.T) {
	repo := NewMemoryRepository([]Task{
		{ID: "expected", TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "assignee", WorkflowKind: MatterActionWorkflowKind, MatterID: "matter-a", MatterScope: json.RawMessage(`{"access":"RESTRICTED"}`), Status: StatusReady},
		{ID: "other-entity", TenantID: "bank", LegalEntityID: "entity-b", PrincipalID: "assignee", WorkflowKind: MatterActionWorkflowKind, MatterID: "matter-b", MatterScope: json.RawMessage(`{"access":"INTERNAL"}`), Status: StatusReady},
		{ID: "unsupported", TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "assignee", WorkflowKind: "UNSUPPORTED", Status: StatusReady},
	})
	values, err := repo.List(t.Context(), ListFilter{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "assignee", ActiveOnly: true, VisibleActorWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != "expected" {
		t.Fatalf("actor work = %#v", values)
	}
}
