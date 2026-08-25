//go:build postgres

package workflow

import (
	"encoding/json"
	"testing"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestDecodeMatterActionProjectionAcceptsEditAndAssignmentEnvelopes(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"action": map[string]any{
			"id": "action-1", "tenant_id": "bank", "matter_id": "matter-1",
			"title": "Update checklist", "owner_principal_id": "performer-2", "status": "PLANNED", "version": 3,
		},
		"previous_owner_principal_id": "performer-1",
		"owner_principal_id":          "performer-2",
		"rationale":                   "Assign the current process owner.",
	})
	if err != nil {
		t.Fatal(err)
	}
	action, err := decodeMatterActionProjection(workflowruntime.OutboxEvent{EventType: "ACTION_ASSIGNED", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if action.ID != "action-1" || action.OwnerPrincipalID != "performer-2" || action.Status != "PLANNED" {
		t.Fatalf("assignment envelope was not decoded: %#v", action)
	}
}
