package bankverticals

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

const vendorDueDiligenceFormCode = "VENDOR-DUE-DILIGENCE"

func (s *Service) ensureVendorDueDiligenceForm(ctx context.Context, config SeedConfig, programID string) error {
	if s.monitoring == nil {
		return nil
	}
	maker := monitoring.Actor{
		TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ActorID,
	}
	checker := monitoring.Actor{
		TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ReviewerPrincipalID,
	}
	if maker.PrincipalID == checker.PrincipalID {
		return monitoring.ErrMakerChecker
	}

	forms, err := s.monitoring.ListForms(ctx, maker, programID, 100)
	if err != nil {
		return fmt.Errorf("list reference vendor due-diligence forms: %w", err)
	}
	var current monitoring.FormTemplate
	for _, form := range forms {
		if !strings.EqualFold(form.Code, vendorDueDiligenceFormCode) {
			continue
		}
		if current.ID == "" || form.Version > current.Version {
			current = form
		}
	}
	if current.ID == "" {
		current, err = s.monitoring.CreateForm(ctx, maker, vendorDueDiligenceFormInput(programID, config.LegalEntityID))
		if err != nil {
			return fmt.Errorf("create reference vendor due-diligence form: %w", err)
		}
	}

	switch current.Status {
	case monitoring.LifecycleActive:
		if !current.IsCurrent {
			return fmt.Errorf("reference vendor due-diligence form is active but not current")
		}
		return nil
	case monitoring.LifecycleDraft:
		current, err = s.monitoring.TransitionForm(ctx, maker, monitoring.TransitionInput{
			ID: current.ID, ProgramID: programID, LegalEntityID: config.LegalEntityID,
			ExpectedVersion: current.Version, To: monitoring.LifecyclePendingApproval,
		})
		if err != nil {
			return fmt.Errorf("submit reference vendor due-diligence form: %w", err)
		}
	case monitoring.LifecyclePendingApproval:
		// Continue an interrupted installation with the configured independent checker.
	case monitoring.LifecyclePaused:
		current, err = s.monitoring.TransitionForm(ctx, checker, monitoring.TransitionInput{
			ID: current.ID, ProgramID: programID, LegalEntityID: config.LegalEntityID,
			ExpectedVersion: current.Version, To: monitoring.LifecycleActive,
		})
		if err != nil {
			return fmt.Errorf("reactivate reference vendor due-diligence form: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("reference vendor due-diligence form cannot be repaired from %s", current.Status)
	}

	_, err = s.monitoring.TransitionForm(ctx, checker, monitoring.TransitionInput{
		ID: current.ID, ProgramID: programID, LegalEntityID: config.LegalEntityID,
		ExpectedVersion: current.Version, To: monitoring.LifecycleActive,
	})
	if err != nil {
		if errors.Is(err, monitoring.ErrMakerChecker) {
			return err
		}
		return fmt.Errorf("activate reference vendor due-diligence form: %w", err)
	}
	return nil
}

func vendorDueDiligenceFormInput(programID, legalEntityID string) monitoring.CreateFormInput {
	minService, maxService := 20, 1200
	minSubprocessors, maxSubprocessors := 10, 1000
	maxEmail := 254
	minSelections, maxSelections := 1, 4
	maxFiles := 1
	maxFileBytes := int64(25_000_000)
	return monitoring.CreateFormInput{
		ProgramID: programID, LegalEntityID: legalEntityID,
		Code:         vendorDueDiligenceFormCode,
		Name:         "Vendor security and privacy review",
		Purpose:      "Collect the vendor information and supporting documents required for onboarding review.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections: []formcontract.Section{
			{ID: "contact", Title: "Company contact", Help: "Confirm who can answer follow-up questions about this submission."},
			{ID: "service", Title: "Service and data", Help: "Describe the service and the bank information it uses."},
			{ID: "controls", Title: "Security controls", Help: "Confirm the controls in operation and provide current supporting documents."},
			{ID: "attestation", Title: "Submission confirmation", Help: "An authorized representative must confirm the response before submission."},
		},
		Fields: []formcontract.Field{
			{ID: "contact_email", SectionID: "contact", Label: "Security contact email", Type: formcontract.TypeEmail, Required: true, Constraints: formcontract.Constraints{MaxLength: &maxEmail}},
			{ID: "service_description", SectionID: "service", Label: "Service description", Type: formcontract.TypeLongText, Required: true, Constraints: formcontract.Constraints{MinLength: &minService, MaxLength: &maxService}},
			{ID: "data_classes", SectionID: "service", Label: "Bank information used", Type: formcontract.TypeMultiSelect, Required: true, Options: []string{"Customer personal data", "Payment data", "Employee data", "Confidential business data", "No bank information"}, Constraints: formcontract.Constraints{MinSelections: &minSelections, MaxSelections: &maxSelections}},
			{ID: "subprocessors", SectionID: "service", Label: "Do subcontractors process bank information?", Type: formcontract.TypeYesNo, Required: true},
			{ID: "subprocessor_details", SectionID: "service", Label: "Subcontractor details", Type: formcontract.TypeLongText, Required: true, Constraints: formcontract.Constraints{MinLength: &minSubprocessors, MaxLength: &maxSubprocessors}, Condition: &formcontract.VisibilityCondition{FieldID: "subprocessors", Operator: formcontract.ConditionEquals, Values: []string{"yes"}}},
			{ID: "security_framework", SectionID: "controls", Label: "Primary security framework", Type: formcontract.TypeSingleSelect, Required: true, Options: []string{"ISO 27001", "SOC 2", "PCI DSS", "NIST CSF", "Other", "None"}},
			{ID: "security_document", SectionID: "controls", Label: "Current independent assurance document", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}, Constraints: formcontract.Constraints{MaxFiles: &maxFiles, MaxFileBytes: &maxFileBytes}},
			{ID: "authorized_attestation", SectionID: "attestation", Label: "Authorized representative confirmation", Type: formcontract.TypeAttestation, Required: true, Attestation: "I confirm that this response is complete and accurate to the best of my knowledge."},
		},
	}
}
