package continuity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProgramTriggerForPrincipalRedactsRestrictedMatterDetails(t *testing.T) {
	trigger := Trigger{
		ID:          "trigger-1",
		TenantID:    "bank",
		ProgramID:   "program-1",
		Type:        "CONTROL_FAILED",
		SubjectType: "CONTROL_IMPLEMENTATION",
		SubjectID:   "control-secret",
		DedupeKey:   "secret-control-failure",
		Payload:     json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-b"],"account":"sensitive"}`),
		ObservedAt:  time.Date(2026, 8, 22, 18, 0, 0, 0, time.UTC),
		Source:      "control-monitor",
		ActorID:     "operator-secret",
	}

	hidden := programTriggerForPrincipal(trigger, "person-a")
	if hidden.Type != trigger.Type || !hidden.ObservedAt.Equal(trigger.ObservedAt) || hidden.Source != trigger.Source {
		t.Fatalf("Program-level trigger fact was lost during redaction: %#v", hidden)
	}
	if hidden.SubjectType != "" || hidden.SubjectID != "" || hidden.DedupeKey != "" || hidden.ActorID != "" || string(hidden.Payload) != `{}` {
		t.Fatalf("restricted trigger details leaked: %#v", hidden)
	}

	visible := programTriggerForPrincipal(trigger, "person-b")
	if visible.SubjectID != trigger.SubjectID || visible.DedupeKey != trigger.DedupeKey || visible.ActorID != trigger.ActorID || string(visible.Payload) != string(trigger.Payload) {
		t.Fatalf("allowed principal lost trigger detail: %#v", visible)
	}
}

func TestProgramReviewEventsForPrincipalUsesSameTriggerRedaction(t *testing.T) {
	trigger := Trigger{
		ID:          "trigger-1",
		TenantID:    "bank",
		ProgramID:   "program-1",
		Type:        "VERIFICATION_FAILED",
		SubjectType: "VERIFICATION",
		SubjectID:   "verification-secret",
		DedupeKey:   "secret-verification",
		Payload:     json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-b"],"result":"sensitive"}`),
		ObservedAt:  time.Date(2026, 8, 22, 18, 0, 0, 0, time.UTC),
		Source:      "verification-worker",
		ActorID:     "operator-secret",
	}
	payload, err := json.Marshal(trigger)
	if err != nil {
		t.Fatal(err)
	}
	events, err := programReviewEventsForPrincipal([]Event{{Type: EventProgramTriggerRecorded, Payload: payload}}, "person-a")
	if err != nil {
		t.Fatal(err)
	}
	var redacted Trigger
	if err := json.Unmarshal(events[0].Payload, &redacted); err != nil {
		t.Fatal(err)
	}
	if redacted.Type != trigger.Type || redacted.SubjectID != "" || redacted.DedupeKey != "" || redacted.ActorID != "" || string(redacted.Payload) != `{}` {
		t.Fatalf("review event did not apply Program trigger redaction: %#v", redacted)
	}

	change, ok := programEventChange(ProgramAggregate{}, events[0])
	if !ok || change.Summary == "" || change.ObjectType != "" || change.ObjectID != "" {
		t.Fatalf("redacted review change leaked subject identity: %#v ok=%v", change, ok)
	}
}
