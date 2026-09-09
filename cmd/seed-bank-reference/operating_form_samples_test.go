//go:build postgres

package main

import "testing"

func TestOperatingFormSampleSpecsCoverVisibleFormStates(t *testing.T) {
	specs := operatingFormSampleSpecs()
	if len(specs) != 5 {
		t.Fatalf("samples=%d, want 5", len(specs))
	}
	want := map[string]bool{"OUTSTANDING": false, "IN_PROGRESS": false, "COMPLETED_HIGH": false, "COMPLETED_GAP": false, "COMPLETED_UNREVIEWED": false}
	var cloudspace *operatingFormSampleSpec
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
		if spec.key == "cloudspace-oem-risk-register" {
			copy := spec
			cloudspace = &copy
		}
	}
	for state, found := range want {
		if !found {
			t.Fatalf("missing %s sample", state)
		}
	}
	if cloudspace == nil || cloudspace.vendorRef != "vendor:cloudspace-oem" || cloudspace.formCode != "VENDOR-DUE-DILIGENCE" || cloudspace.state != "COMPLETED_UNREVIEWED" {
		t.Fatalf("Cloudspace sample=%+v", cloudspace)
	}
	if answer := cloudspace.answers["data_classes"]; len(answer.Values) != 1 || answer.Values[0] != "Payment data" {
		t.Fatalf("Cloudspace payment-data answer=%+v", answer)
	}
	if answer, ok := cloudspace.answers["assurance_gap"].ScalarText(); !ok || answer == "" {
		t.Fatalf("Cloudspace assurance gap=%q, present=%t", answer, ok)
	}
}
