//go:build postgres

package main

import (
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"os"
	"testing"
)

// This is the original 05ceac62 shipped contract, before f0ee6b27 added
// collection-intent and browser-cache defaults. Hosted v3 was checked read-only.
func historicalVendorForm(t *testing.T) monitoring.FormTemplate {
	t.Helper()
	raw, err := os.ReadFile("testdata/vendor-due-diligence-20260827.json")
	if err != nil {
		t.Fatal(err)
	}
	var form monitoring.FormTemplate
	if err = json.Unmarshal(raw, &form); err != nil {
		t.Fatal(err)
	}
	return form
}

func TestKnownLegacyVendorFormPreservesHistoricalMeaning(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*monitoring.FormTemplate)
		want   bool
	}{
		{name: "historical omitted defaults", want: true},
		{name: "explicit defaults", want: true, change: func(f *monitoring.FormTemplate) {
			for n := range f.Fields {
				f.Fields[n].CollectionIntent = formcontract.IntentCapture
				f.Fields[n].BrowserCachePolicy = formcontract.BrowserCacheAllowed
			}
		}},
		{name: "changed section guidance", change: func(f *monitoring.FormTemplate) {
			f.Sections[1].Help = "Describe the new controls required by our reviewers."
		}},
		{name: "browser cache prohibited", change: func(f *monitoring.FormTemplate) { f.Fields[0].BrowserCachePolicy = formcontract.BrowserCacheDenied }},
		{name: "different collection task", change: func(f *monitoring.FormTemplate) {
			f.Fields[0].CollectionIntent = formcontract.IntentConfirmOrCorrect
			f.Fields[0].RecordTarget = &formcontract.RecordTarget{Key: "VENDOR.IDENTITY.SECURITY_CONTACT", RequiredSubjectType: "VENDOR_RELATIONSHIP"}
		}},
		{name: "changed field label", change: func(f *monitoring.FormTemplate) { f.Fields[0].Label = "Our designated security officer" }},
		{name: "changed choices", change: func(f *monitoring.FormTemplate) { f.Fields[2].Options[0] = "Only employee data" }},
		{name: "changed requirement", change: func(f *monitoring.FormTemplate) { f.Fields[6].Required = false }},
		{name: "invalid contract", change: func(f *monitoring.FormTemplate) { f.Fields[0].BrowserCachePolicy = "UNKNOWN" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			form := historicalVendorForm(t)
			if test.change != nil {
				test.change(&form)
			}
			desired := bankverticals.ReferenceVendorDueDiligenceForm(form.ProgramID, form.LegalEntityID)
			before, _ := json.Marshal(form)
			desiredBefore, _ := json.Marshal(desired)
			if got := knownLegacyVendorForm(form, desired); got != test.want {
				t.Fatalf("historical match=%v, want %v", got, test.want)
			}
			after, _ := json.Marshal(form)
			desiredAfter, _ := json.Marshal(desired)
			if string(before) != string(after) || string(desiredBefore) != string(desiredAfter) {
				t.Fatal("comparison mutated a form contract")
			}
		})
	}
}
