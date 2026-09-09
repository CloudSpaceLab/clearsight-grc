package bankverticals

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

const vendorComplianceFormCode = "THIRD-PARTY-RISK-COMPLIANCE"

// ReferenceThirdPartyRiskComplianceForm is a configurable sample contract.
// The source register supplies requirement topics, not vendor facts or ratings.
func ReferenceThirdPartyRiskComplianceForm(programID, legalEntityID string) monitoring.CreateFormInput {
	input := monitoring.CreateFormInput{
		ProgramID: programID, LegalEntityID: legalEntityID, Code: vendorComplianceFormCode,
		Name:         "Third Party Risk Compliance",
		Purpose:      "Sample form: confirm applicable service requirements and provide evidence for independent review. Missing evidence may be declared and submitted.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		ScoringMode:  formcontract.ScoringCompliance,
		ScoreProfile: &formcontract.ScoreProfile{Version: "third-party-risk-compliance-sample-v1", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, Bands: formcontract.DefaultConcernBands()},
	}
	maxFiles, maxBytes := 1, int64(25_000_000)
	condition := func(field string, values ...string) *formcontract.VisibilityCondition {
		return &formcontract.VisibilityCondition{FieldID: field, Operator: formcontract.ConditionIn, Values: values}
	}
	review := func(mode formcontract.AssessmentMode) *formcontract.FieldAssessment {
		return &formcontract.FieldAssessment{Mode: mode, Required: true, Weight: 50, ReviewerRole: "REVIEWER", Rubric: []formcontract.AssessmentOutcome{
			{ID: "supported", Label: "Evidence supports the answer", Points: 100},
			{ID: "not_supported", Label: "Evidence does not support the answer", Points: 0},
		}}
	}
	for _, area := range []struct{ id, title, description, current string }{
		{"iso27001", "ISO 27001", "Confirm whether the service requires information security certification and whether the certificate covers this service.", "Current"},
		{"iso22301", "ISO 22301", "Confirm whether the service requires business continuity certification and whether the certificate covers this service.", "Current"},
		{"vapt", "Vulnerability assessment and penetration testing", "Confirm whether security testing is required for the service and whether a current report is available.", "Current"},
		{"audit_rights", "Contractual audit rights", "Confirm whether the service agreement must give the organization a right to audit and whether the signed terms provide it.", "Present"},
		{"pci_dss", "PCI DSS", "Confirm whether payment card security requirements apply to the service and whether a current attestation covers it.", "Current"},
	} {
		id := area.id
		input.Sections = append(input.Sections, formcontract.Section{ID: id, Title: area.title, Help: area.description, Weight: 20})
		options := []string{area.current, "Missing", "Expired", "Not met"}
		if id == "audit_rights" {
			options = []string{area.current, "Missing", "Not met"}
		}
		input.Fields = append(input.Fields,
			formcontract.Field{ID: id + "_applicable", SectionID: id, Label: "Does " + area.title + " apply to this service?", Type: formcontract.TypeYesNo, Required: true, Description: "The reviewer will check the applicability answer against the service requirements."},
			formcontract.Field{ID: id + "_status", SectionID: id, Label: area.title + " status", Type: formcontract.TypeSingleSelect, Required: true, Options: options, Condition: condition(id+"_applicable", "Yes"), Assessment: review(formcontract.AssessmentAutomaticReview)},
			formcontract.Field{ID: id + "_document", SectionID: id, Label: area.title + " evidence", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}, Constraints: formcontract.Constraints{MaxFiles: &maxFiles, MaxFileBytes: &maxBytes}, Condition: condition(id+"_status", area.current), Description: "Upload the supporting PDF and record its issue and expiry dates where provided. The reviewer must confirm the document's scope, dates and validity."},
			formcontract.Field{ID: id + "_gap", SectionID: id, Label: area.title + " gap and planned action", Type: formcontract.TypeLongText, Required: true, Condition: condition(id+"_status", "Missing", "Expired", "Not met"), Description: "Explain what is missing or not met, who will address it and the planned completion date if known."},
			formcontract.Field{ID: id + "_not_applicable", SectionID: id, Label: "Why " + area.title + " does not apply", Type: formcontract.TypeLongText, Required: true, Condition: condition(id+"_applicable", "No"), Assessment: review(formcontract.AssessmentManual), Description: "Identify the service scope or requirement supporting this answer. The reviewer must confirm it."},
		)
		input.ScoreProfile.Contributions = append(input.ScoreProfile.Contributions, formcontract.ScoreContribution{
			ID: id + "-evidence", Label: area.title + " evidence", Weight: 20, Required: true,
			Predicate:   formcontract.Predicate{FieldID: id + "_status", Operator: formcontract.PredicateEquals, Values: []string{area.current}},
			MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingIndeterminate,
		})
	}
	input.Sections = append(input.Sections, formcontract.Section{ID: "confirmation", Title: "Submission confirmation", Help: "Confirm that the answers describe the service and the evidence available today."})
	input.Fields = append(input.Fields, formcontract.Field{ID: "vendor_confirmation", SectionID: "confirmation", Label: "Authorized representative confirmation", Type: formcontract.TypeAttestation, Required: true, Attestation: "I confirm that these answers describe the service and evidence available to us, including any missing, expired or unmet requirements. The organization will independently review this submission."})
	return input
}
