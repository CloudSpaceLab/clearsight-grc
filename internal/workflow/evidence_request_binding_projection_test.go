//go:build postgres

package workflow

import "testing"

func TestProjectedBindingReferencesDeduplicatesAndSortsOwningDomainReferences(t *testing.T) {
	fields := []byte(`[{"bindings":[{"binding_id":"z","binding_version":2},{"binding_id":"a","binding_version":4}]},{"bindings":[{"binding_id":"z","binding_version":2}]}]`)
	request := []byte(`[{"binding_id":"a","binding_version":4},{"binding_id":"evidence","binding_version":1}]`)
	values, err := projectedBindingReferences(fields, request)
	if err != nil {
		t.Fatalf("project references: %v", err)
	}
	if len(values) != 3 || values[0].BindingID != "a" || values[1].BindingID != "evidence" || values[2].BindingID != "z" {
		t.Fatalf("unexpected projection: %#v", values)
	}
}

func TestProjectedBindingReferencesRejectsMalformedOrIncompleteOwningRecord(t *testing.T) {
	cases := []struct {
		name     string
		fields   []byte
		requests []byte
	}{
		{name: "malformed fields", fields: []byte(`{"not":"an array"}`), requests: []byte(`[]`)},
		{name: "missing field binding id", fields: []byte(`[{"bindings":[{"binding_version":2}]}]`), requests: []byte(`[]`)},
		{name: "missing request binding version", fields: []byte(`[]`), requests: []byte(`[{"binding_id":"evidence"}]`)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := projectedBindingReferences(test.fields, test.requests); err == nil {
				t.Fatal("expected owning record to fail closed")
			}
		})
	}
}
