package bankverticals

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

const (
	vendorAddressVerificationFormCode  = "VENDOR-ADDRESS-VERIFICATION"
	vendorCertificationRefreshFormCode = "VENDOR-CERTIFICATION-REFRESH"
)

func (s *Service) ensureVendorAcceptanceForms(ctx context.Context, config SeedConfig, programID string) error {
	forms := []struct {
		input   monitoring.CreateFormInput
		purpose string
	}{
		{vendorAddressVerificationFormInput(programID, config.LegalEntityID), "vendor address-verification"},
		{vendorCertificationRefreshFormInput(programID, config.LegalEntityID), "vendor certification-refresh"},
	}
	for _, form := range forms {
		if err := s.ensureGovernedVendorForm(ctx, config, form.input, form.purpose); err != nil {
			return err
		}
	}
	return nil
}

func vendorAddressVerificationFormInput(programID, legalEntityID string) monitoring.CreateFormInput {
	maxFiles := 1
	maxFileBytes := int64(25_000_000)
	return monitoring.CreateFormInput{
		ProgramID: programID, LegalEntityID: legalEntityID,
		Code:         vendorAddressVerificationFormCode,
		Name:         "Verify vendor address",
		Purpose:      "Record how the assigned staff member checked the vendor address and provide the evidence required for compliance review.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections: []formcontract.Section{
			{ID: "result", Title: "Verification result", Help: "Record the address check and the source used."},
			{ID: "evidence", Title: "Supporting evidence", Help: "Upload the current document that supports the verification result."},
			{ID: "attestation", Title: "Staff confirmation", Help: "Confirm that the recorded check and evidence are complete."},
		},
		Fields: []formcontract.Field{
			{ID: "verification_result", SectionID: "result", Label: "Address verification result", Type: formcontract.TypeSingleSelect, Required: true, Options: []string{"Verified", "Could not verify", "Different address found"}},
			{ID: "verification_method", SectionID: "result", Label: "Verification method", Type: formcontract.TypeSingleSelect, Required: true, Options: []string{"Official registry", "Recent utility or bank document", "Site visit", "Independent business directory", "Other documented source"}},
			{ID: "checked_on", SectionID: "result", Label: "Date checked", Type: formcontract.TypeDate, Required: true},
			{ID: "source_contact", SectionID: "result", Label: "Source or contact checked", Type: formcontract.TypeShortText, Required: true},
			{ID: "address_evidence", SectionID: "evidence", Label: "Address verification evidence", Type: formcontract.TypeFile, Required: true, AcceptedFormats: []string{"application/pdf"}, Constraints: formcontract.Constraints{MaxFiles: &maxFiles, MaxFileBytes: &maxFileBytes}},
			{ID: "staff_attestation", SectionID: "attestation", Label: "Staff verification confirmation", Type: formcontract.TypeAttestation, Required: true, Attestation: "I confirm that I performed the address check recorded above and that the attached evidence supports the stated result."},
		},
	}
}

func vendorCertificationRefreshFormInput(programID, legalEntityID string) monitoring.CreateFormInput {
	maxFiles := 1
	maxFileBytes := int64(25_000_000)
	yesCondition := func(fieldID string) *formcontract.VisibilityCondition {
		return &formcontract.VisibilityCondition{FieldID: fieldID, Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}
	}
	return monitoring.CreateFormInput{
		ProgramID: programID, LegalEntityID: legalEntityID,
		Code:         vendorCertificationRefreshFormCode,
		Name:         "Submit current vendor certifications",
		Purpose:      "Confirm whether ISO 27001 and PCI DSS apply to the vendor service and submit the current documents required for bank review.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		ScoringMode:  formcontract.ScoringCompliance,
		ScoreProfile: vendorCertificationScoreProfile(),
		Sections: []formcontract.Section{
			{ID: "iso", Title: "ISO 27001", Help: "Confirm applicability and provide the current certificate when it applies."},
			{ID: "pci", Title: "PCI DSS", Help: "Confirm applicability and provide the current attestation when it applies."},
			{ID: "attestation", Title: "Vendor confirmation", Help: "An authorized vendor representative must confirm the response before submission."},
		},
		Fields: []formcontract.Field{
			{ID: "iso_applicable", SectionID: "iso", Label: "Does ISO 27001 apply to this service?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "iso_current", SectionID: "iso", Label: "Is the ISO 27001 certification current?", Type: formcontract.TypeYesNo, Required: true, Condition: yesCondition("iso_applicable")},
			{ID: "iso_certificate", SectionID: "iso", Label: "Current ISO 27001 certificate", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}, Constraints: formcontract.Constraints{MaxFiles: &maxFiles, MaxFileBytes: &maxFileBytes}, Condition: yesCondition("iso_current")},
			{ID: "pci_applicable", SectionID: "pci", Label: "Does PCI DSS apply to this service?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "pci_current", SectionID: "pci", Label: "Is the PCI DSS attestation current?", Type: formcontract.TypeYesNo, Required: true, Condition: yesCondition("pci_applicable")},
			{ID: "pci_attestation", SectionID: "pci", Label: "Current PCI DSS attestation", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}, Constraints: formcontract.Constraints{MaxFiles: &maxFiles, MaxFileBytes: &maxFileBytes}, Condition: yesCondition("pci_current")},
			{ID: "vendor_attestation", SectionID: "attestation", Label: "Authorized vendor confirmation", Type: formcontract.TypeAttestation, Required: true, Attestation: "I confirm that the applicability answers are accurate and that each submitted document is current and applies to the service provided to the bank."},
		},
	}
}

func vendorCertificationScoreProfile() *formcontract.ScoreProfile {
	applicability := func(id, fieldID, label string) formcontract.ScoreContribution {
		return formcontract.ScoreContribution{
			ID: id, Label: label, Weight: 25, Required: true,
			Predicate:   formcontract.Predicate{FieldID: fieldID, Operator: formcontract.PredicateAnswered},
			MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingIndeterminate,
		}
	}
	current := func(id, fieldID, label string) formcontract.ScoreContribution {
		return formcontract.ScoreContribution{
			ID: id, Label: label, Weight: 25, Required: true,
			Predicate:   formcontract.Predicate{FieldID: fieldID, Operator: formcontract.PredicateEquals, Values: []string{"Yes"}},
			MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingIndeterminate,
		}
	}
	notCurrent := func(applicabilityID, currentID string) formcontract.Predicate {
		return formcontract.Predicate{Operator: formcontract.PredicateAnd, Children: []formcontract.Predicate{
			{FieldID: applicabilityID, Operator: formcontract.PredicateEquals, Values: []string{"Yes"}},
			{FieldID: currentID, Operator: formcontract.PredicateEquals, Values: []string{"No"}},
		}}
	}
	return &formcontract.ScoreProfile{
		Version: "vendor-certification-v1", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor,
		Contributions: []formcontract.ScoreContribution{
			applicability("iso-applicability", "iso_applicable", "ISO 27001 applicability answered"),
			current("iso-current", "iso_current", "ISO 27001 certification current"),
			applicability("pci-applicability", "pci_applicable", "PCI DSS applicability answered"),
			current("pci-current", "pci_current", "PCI DSS attestation current"),
		},
		Rules: []formcontract.ScoreRule{{
			ID: "required-certification-not-current", Label: "An applicable certification is not current",
			Predicate: formcontract.Predicate{Operator: formcontract.PredicateOr, Children: []formcontract.Predicate{
				notCurrent("iso_applicable", "iso_current"), notCurrent("pci_applicable", "pci_current"),
			}},
			Effect: formcontract.RuleEffect{Kind: formcontract.EffectDisqualify},
		}},
		Bands: formcontract.DefaultConcernBands(),
	}
}
