package continuity

import "testing"

func TestMatterReferenceUsesEntropyBearingUUIDSuffix(t *testing.T) {
	first := matterReference("019fd32b-1234-7000-8000-111111111111")
	second := matterReference("019fd32b-1234-7000-8000-222222222222")
	if first == second {
		t.Fatalf("same-millisecond UUIDv7 values produced the same reference: %s", first)
	}
	if first != "MAT-8000111111111111" || second != "MAT-8000222222222222" {
		t.Fatalf("unexpected references: %s %s", first, second)
	}
}
