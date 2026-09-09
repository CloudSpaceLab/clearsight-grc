//go:build postgres

package main

import "testing"

func TestOperatingFormSampleSpecsCoverVisibleFormStates(t *testing.T) {
	specs := operatingFormSampleSpecs()
	if len(specs) != 4 {
		t.Fatalf("samples=%d, want 4", len(specs))
	}
	want := map[string]bool{"OUTSTANDING": false, "IN_PROGRESS": false, "COMPLETED_HIGH": false, "COMPLETED_GAP": false}
	for _, spec := range specs {
		if spec.key == "" || spec.vendorRef == "" || spec.formCode == "" || spec.title == "" {
			t.Fatalf("sample lacks stable business identity: %+v", spec)
		}
		if _, ok := want[spec.state]; !ok || want[spec.state] {
			t.Fatalf("unexpected or duplicate state %q", spec.state)
		}
		want[spec.state] = true
		if (spec.state == "COMPLETED_HIGH" || spec.state == "COMPLETED_GAP") && len(spec.answers) == 0 {
			t.Fatalf("completed sample %q has no answers", spec.key)
		}
		if spec.state == "IN_PROGRESS" && (len(spec.answers) == 0 || len(spec.answers) >= spec.totalFields) {
			t.Fatalf("in-progress sample %q is not partial", spec.key)
		}
	}
	for state, found := range want {
		if !found {
			t.Fatalf("missing %s sample", state)
		}
	}
}
