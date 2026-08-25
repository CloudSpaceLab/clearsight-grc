//go:build postgres

package continuity

import (
	"encoding/json"
	"testing"
)

func TestMatterEditProjectionPayloadsExposeCurrentRows(t *testing.T) {
	matter := Matter{ID: "matter-1", TenantID: "bank", Title: "Updated title", Version: 2}
	detailPayload, err := json.Marshal(matterDetailsUpdatedEvent{Matter: matter, Previous: Matter{Title: "Old title"}, Rationale: "Clarify the record."})
	if err != nil {
		t.Fatal(err)
	}
	projectedMatter, ok, err := matterProjectionMatter(Event{Type: EventMatterDetailsUpdated, Payload: detailPayload})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || projectedMatter.Title != "Updated title" {
		t.Fatalf("detail event did not expose its current Matter row: %#v ok=%v", projectedMatter, ok)
	}

	action := Action{ID: "action-1", MatterID: "matter-1", TenantID: "bank", OwnerPrincipalID: "performer-2", Version: 3}
	actionPayload, err := json.Marshal(actionAssignedEvent{Action: action, PreviousOwnerID: "performer-1", OwnerPrincipalID: "performer-2", Rationale: "Assign the current owner."})
	if err != nil {
		t.Fatal(err)
	}
	projectedAction, ok, err := matterProjectionAction(Event{Type: EventActionAssigned, Payload: actionPayload})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || projectedAction.OwnerPrincipalID != "performer-2" || projectedAction.Version != 3 {
		t.Fatalf("assignment event did not expose its current Action row: %#v ok=%v", projectedAction, ok)
	}
}
