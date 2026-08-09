package continuity

import "testing"

func TestReasonExplanationChangeIsReviewableWithoutBecomingNewException(t *testing.T) {
	before := []StateReason{{Code: "EVIDENCE_EXPIRED", Summary: "Evidence is old.", ObjectType: "EVIDENCE_CONTRACT", ObjectID: "contract-1"}}
	after := []StateReason{{Code: "EVIDENCE_EXPIRED", Summary: "Annual evidence is outside its freshness window.", ObjectType: "EVIDENCE_CONTRACT", ObjectID: "contract-1"}}

	added, resolved := diffStateReasons(before, after)
	if len(added) != 0 || len(resolved) != 0 {
		t.Fatalf("wording change manufactured exception identity: added=%#v resolved=%#v", added, resolved)
	}
	changes := reasonExplanationChanges(before, after)
	if len(changes) != 1 {
		t.Fatalf("expected one explanation change, got %#v", changes)
	}
	if changes[0].Kind != "EXPLANATION" || changes[0].ObjectType != "EVIDENCE_CONTRACT" || changes[0].ObjectID != "contract-1" {
		t.Fatalf("explanation change lost canonical reason identity: %#v", changes[0])
	}
	if changes[0].Summary != "Status explanation updated: Annual evidence is outside its freshness window." {
		t.Fatalf("unexpected explanation copy: %q", changes[0].Summary)
	}
}
