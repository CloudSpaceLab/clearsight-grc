package bankverticals

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestVendorAddressVerificationFormContract(t *testing.T) {
	form := vendorAddressVerificationFormInput("program-a", "entity-a")
	if form.Code != "VENDOR-ADDRESS-VERIFICATION" || form.Name != "Verify vendor address" {
		t.Fatalf("unexpected address form identity: %#v", form)
	}
	if len(form.Sections) != 3 || len(form.Fields) != 6 {
		t.Fatalf("address form sections=%d fields=%d", len(form.Sections), len(form.Fields))
	}
	want := map[string]formcontract.Type{
		"verification_result": formcontract.TypeSingleSelect,
		"verification_method": formcontract.TypeSingleSelect,
		"checked_on":          formcontract.TypeDate,
		"source_contact":      formcontract.TypeShortText,
		"address_evidence":    formcontract.TypeFile,
		"staff_attestation":   formcontract.TypeAttestation,
	}
	for _, field := range form.Fields {
		if want[field.ID] != field.Type || !field.Required {
			t.Fatalf("unexpected address field: %#v", field)
		}
		delete(want, field.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing address fields: %#v", want)
	}
	result := form.Fields[0]
	if len(result.Options) != 3 || result.Options[0] != "Verified" || result.Options[1] != "Could not verify" || result.Options[2] != "Different address found" {
		t.Fatalf("verification result options=%#v", result.Options)
	}
	if got := form.Fields[4]; len(got.AcceptedFormats) != 1 || got.AcceptedFormats[0] != "application/pdf" {
		t.Fatalf("address evidence must require one PDF: %#v", got)
	}
}

func TestVendorCertificationRefreshFormContract(t *testing.T) {
	form := vendorCertificationRefreshFormInput("program-a", "entity-a")
	if form.Code != "VENDOR-CERTIFICATION-REFRESH" || form.Name != "Submit current vendor certifications" {
		t.Fatalf("unexpected certification form identity: %#v", form)
	}
	if len(form.Sections) != 3 || len(form.Fields) != 7 {
		t.Fatalf("certification form sections=%d fields=%d", len(form.Sections), len(form.Fields))
	}
	for _, pair := range []struct{ applicability, current, document string }{{"iso_applicable", "iso_current", "iso_certificate"}, {"pci_applicable", "pci_current", "pci_attestation"}} {
		var applicability, current, document *formcontract.Field
		for index := range form.Fields {
			if form.Fields[index].ID == pair.applicability {
				applicability = &form.Fields[index]
			}
			if form.Fields[index].ID == pair.document {
				document = &form.Fields[index]
			}
			if form.Fields[index].ID == pair.current {
				current = &form.Fields[index]
			}
		}
		if applicability == nil || applicability.Type != formcontract.TypeYesNo || !applicability.Required {
			t.Fatalf("invalid applicability field %q: %#v", pair.applicability, applicability)
		}
		if current == nil || current.Type != formcontract.TypeYesNo || !current.Required || current.Condition == nil || current.Condition.FieldID != pair.applicability {
			t.Fatalf("invalid current-status field %q: %#v", pair.current, current)
		}
		if document == nil || document.Type != formcontract.TypeVendorDocument || !document.Required || document.Condition == nil || document.Condition.FieldID != pair.current || document.Condition.Operator != formcontract.ConditionEquals || len(document.Condition.Values) != 1 || document.Condition.Values[0] != "Yes" {
			t.Fatalf("invalid conditional document %q: %#v", pair.document, document)
		}
		if len(document.AcceptedFormats) != 1 || document.AcceptedFormats[0] != "application/pdf" {
			t.Fatalf("document %q does not require PDF: %#v", pair.document, document)
		}
	}
	if form.ScoringMode != formcontract.ScoringCompliance || form.ScoreProfile == nil || form.ScoreProfile.Version != "vendor-certification-v1" {
		t.Fatalf("certification score profile=%#v mode=%q", form.ScoreProfile, form.ScoringMode)
	}
	poor, err := formcontract.EvaluateScoreProfile(*form.ScoreProfile, formcontract.Contract{
		Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: form.Fields,
	}, formcontract.TextAnswers(map[string]string{
		"iso_applicable": "Yes", "iso_current": "No", "pci_applicable": "No", "vendor_attestation": "Confirmed",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !poor.Disqualified || poor.Band != formcontract.ConcernCritical || poor.RawScore == nil || *poor.RawScore < 66 || *poor.RawScore > 67 {
		t.Fatalf("poor certification result=%#v", poor)
	}
	attestation := form.Fields[len(form.Fields)-1]
	if attestation.ID != "vendor_attestation" || attestation.Type != formcontract.TypeAttestation || !attestation.Required {
		t.Fatalf("invalid vendor attestation: %#v", attestation)
	}
}

func TestResponsePolicyAcceptanceFormContract(t *testing.T) {
	form := responsePolicyAcceptanceFormInput("program-a", "entity-a")
	if form.Code != responsePolicyAcceptanceFormCode || form.ScoringMode != formcontract.ScoringCompliance || form.ScoreProfile == nil {
		t.Fatalf("acceptance form identity/scoring=%#v", form)
	}
	if form.ScoreProfile.Version != "response-policy-acceptance-v1" || len(form.ScoreProfile.Contributions) != 3 || len(form.ScoreProfile.Rules) != 1 {
		t.Fatalf("acceptance score profile=%#v", form.ScoreProfile)
	}
	weights := map[string]float64{}
	for _, contribution := range form.ScoreProfile.Contributions {
		weights[contribution.ID] = contribution.Weight
	}
	if weights["control-designed"] != 40 || weights["control-operating"] != 40 || weights["exceptions-resolved"] != 20 {
		t.Fatalf("acceptance weights=%#v", weights)
	}
	if rule := form.ScoreProfile.Rules[0]; rule.ID != "critical-gap-open" || rule.Effect.Kind != formcontract.EffectDisqualify {
		t.Fatalf("acceptance critical rule=%#v", rule)
	}

	cases := []struct {
		name         string
		answers      map[string]string
		wantBand     formcontract.ConcernBand
		disqualified bool
	}{
		{name: "good", answers: map[string]string{"control_designed": "Yes", "control_operating": "Yes", "exceptions_resolved": "Yes", "critical_gap": "No"}, wantBand: formcontract.ConcernLow},
		{name: "borderline", answers: map[string]string{"control_designed": "Yes", "control_operating": "No", "exceptions_resolved": "Yes", "critical_gap": "No"}, wantBand: formcontract.ConcernHigh},
		{name: "poor", answers: map[string]string{"control_designed": "No", "control_operating": "No", "exceptions_resolved": "No", "critical_gap": "Yes"}, wantBand: formcontract.ConcernCritical, disqualified: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := formcontract.EvaluateScoreProfile(*form.ScoreProfile, formcontract.Contract{
				Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: form.Fields,
			}, formcontract.TextAnswers(tc.answers))
			if err != nil {
				t.Fatal(err)
			}
			if result.Band != tc.wantBand || result.Disqualified != tc.disqualified || !result.Final {
				t.Fatalf("%s acceptance score=%#v", tc.name, result)
			}
		})
	}
}
