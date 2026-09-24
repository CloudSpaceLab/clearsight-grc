package ropa

import (
	"encoding/json"
	"testing"
)

func TestActivityPageUsesStableSnakeCaseJSONFields(t *testing.T) {
	page := ActivityPage{Rows: []ProcessingActivity{}, NextCursor: "cursor-2", HasMore: true}
	raw, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"rows", "next_cursor", "has_more"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("activity page JSON missing %q: %s", field, raw)
		}
	}
	for _, field := range []string{"Rows", "NextCursor", "HasMore"} {
		if _, ok := body[field]; ok {
			t.Fatalf("activity page JSON retained exported field %q: %s", field, raw)
		}
	}
}
