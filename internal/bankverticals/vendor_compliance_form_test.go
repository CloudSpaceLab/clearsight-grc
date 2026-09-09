package bankverticals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func installedComplianceForm(t *testing.T) (*Service, *monitoring.Service, SeedConfig, monitoring.FormTemplate) {
	t.Helper()
	config := normalizeSeedConfig(DemoSeedConfig())
	forms := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	service := NewService(nil, nil)
	service.ConfigureMonitoring(forms)
	if err := service.ensureVendorAcceptanceForms(context.Background(), config, "program"); err != nil {
		t.Fatal(err)
	}
	actor := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ActorID}
	items, err := forms.ListReusableForms(context.Background(), actor, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range items {
		if form.Code == "THIRD-PARTY-RISK-COMPLIANCE" {
			return service, forms, config, form
		}
	}
	t.Fatal("default Third Party Risk Compliance form was not installed")
	return nil, nil, config, monitoring.FormTemplate{}
}

func TestDefaultComplianceFormAllowsSubmittedGapsAndRequiresReview(t *testing.T) {
	_, _, _, form := installedComplianceForm(t)
	contract, err := formcontract.Normalize(formcontract.Contract{Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: form.Fields})
	if err != nil {
		t.Fatal(err)
	}
	answers := formcontract.TextAnswers(map[string]string{"vendor_confirmation": "true"})
	for _, area := range []string{"iso27001", "iso22301", "vapt", "audit_rights", "pci_dss"} {
		answers[area+"_applicable"] = formcontract.TextAnswer("Yes")
		answers[area+"_status"] = formcontract.TextAnswer("Missing")
		answers[area+"_gap"] = formcontract.TextAnswer("Evidence is unavailable; the service owner will confirm a completion date.")
	}
	visible, err := formcontract.VisibleFields(contract, answers)
	if err != nil {
		t.Fatal(err)
	}
	reviews := 0
	for _, field := range visible {
		if field.Required && !answers[field.ID].Answered() {
			t.Fatalf("negative response cannot be submitted: %s still requires evidence", field.ID)
		}
		if field.Assessment != nil && field.Assessment.NeedsReview() {
			reviews++
			if !field.Assessment.Required || field.Assessment.ReviewerRole == "" {
				t.Fatalf("missing independent review contract: %+v", field)
			}
		}
	}
	if reviews != 5 {
		t.Fatalf("expected five independent requirement reviews, got %d", reviews)
	}
	automatic, err := formcontract.EvaluateScoreProfile(*contract.ScoreProfile, contract, answers)
	if err != nil {
		t.Fatal(err)
	}
	if len(automatic.ContributionResults) != 5 || automatic.RawScore == nil || *automatic.RawScore != 0 {
		t.Fatalf("submitted missing evidence must expose five adverse results: %+v", automatic)
	}
	assessed, err := formcontract.EvaluateAssessment(contract, answers, nil)
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Final {
		t.Fatal("vendor submission cannot complete independent evidence review")
	}
	outcomes := map[string]string{}
	for _, area := range []string{"iso27001", "iso22301", "vapt", "audit_rights", "pci_dss"} {
		outcomes[area+"_status"] = "supported"
	}
	reviewed, err := formcontract.EvaluateAssessment(contract, answers, outcomes)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.RawScore == nil || *reviewed.RawScore != 0 {
		t.Fatalf("review cannot remove a declared automatic gap: %+v", reviewed)
	}
	// Submit the negative answers through the actual capture validation boundary.
	data, err := json.Marshal(form.Fields)
	if err != nil {
		t.Fatal(err)
	}
	var fields []evidence.Field
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	capture := evidence.NewServiceWithClock(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore(), func() time.Time { return now })
	request, err := capture.CreateRequest(context.Background(), evidence.CreateRequestInput{
		TenantID: form.TenantID, SubjectType: "PROGRAM", SubjectID: form.ProgramID,
		Title: form.Name, Purpose: form.Purpose, WhyYou: "You provide the service evidence.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: "vendor-respondent"}, EstimatedMinutes: 10, Deadline: now.Add(time.Hour),
		Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: fields,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	answers["pci_dss_status"] = formcontract.TextAnswer("Expired")
	if _, err := capture.Submit(context.Background(), evidence.Submission{TenantID: form.TenantID, RequestID: request.ID, SubmittedBy: "vendor-respondent", Channel: "INTERNAL", ExpectedVersion: request.Version, Answers: answers}); err != nil {
		t.Fatalf("missing and expired evidence must remain submittable: %v", err)
	}
}

func TestDefaultComplianceNotApplicableStillNeedsEvidenceReview(t *testing.T) {
	input := ReferenceThirdPartyRiskComplianceForm("program", "entity")
	contract := formcontract.Contract{Presentation: input.Presentation, ScoringMode: input.ScoringMode, ScoreProfile: input.ScoreProfile, Sections: input.Sections, Fields: input.Fields}
	answers := formcontract.TextAnswers(map[string]string{"vendor_confirmation": "true"})
	for _, area := range []string{"iso27001", "iso22301", "vapt", "audit_rights", "pci_dss"} {
		answers[area+"_applicable"] = formcontract.TextAnswer("No")
		answers[area+"_not_applicable"] = formcontract.TextAnswer("The service owner must confirm the applicable contractual scope.")
	}
	result, err := formcontract.EvaluateAssessment(contract, answers, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Final || result.Coverage != 0 {
		t.Fatalf("unreviewed applicability cannot establish compliance: %+v", result)
	}
	for _, contribution := range result.ContributionResults {
		if strings.HasSuffix(contribution.ID, "-evidence") && (contribution.Weight != 0 || contribution.Outcome != formcontract.ScoreIndeterminate) {
			t.Fatalf("non-applicable status became a missing or adverse evidence contribution: %+v", contribution)
		}
	}
	visible, err := formcontract.VisibleFields(contract, answers)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range visible {
		if field.Type == formcontract.TypeVendorDocument || strings.HasSuffix(field.ID, "_status") {
			t.Fatalf("non-applicable requirement requests missing evidence: %s", field.ID)
		}
		if field.Required && !answers[field.ID].Answered() {
			t.Fatalf("non-applicable submission is incomplete: %s", field.ID)
		}
	}
}

func TestDefaultComplianceDocumentQuestionsRetainConditionalMetadataCapture(t *testing.T) {
	input := ReferenceThirdPartyRiskComplianceForm("program", "entity")
	count := 0
	for _, field := range input.Fields {
		if field.Type != formcontract.TypeVendorDocument {
			continue
		}
		count++
		if !field.Required || field.Condition == nil || len(field.AcceptedFormats) != 1 || field.AcceptedFormats[0] != "application/pdf" {
			t.Fatalf("evidence contract lost: %+v", field)
		}
	}
	if count != 5 {
		t.Fatalf("expected five native vendor documents with issue/expiry metadata, got %d", count)
	}
}

func TestDefaultComplianceInstallerPreservesCustomizedDraft(t *testing.T) {
	config := normalizeSeedConfig(DemoSeedConfig())
	forms := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	service := NewService(nil, nil)
	service.ConfigureMonitoring(forms)
	actor := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ActorID}
	input := ReferenceThirdPartyRiskComplianceForm("program", config.LegalEntityID)
	input.Fields[0].Label = "Our contracted security certification requirement"
	custom, err := forms.CreateForm(context.Background(), actor, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ensureVendorAcceptanceForms(context.Background(), config, "program"); err != nil {
		t.Fatal(err)
	}
	items, err := forms.ListForms(context.Background(), actor, "program", 100)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, f := range items {
		if f.Code == vendorComplianceFormCode {
			count++
			if !reflect.DeepEqual(f, custom) {
				t.Fatal("customized draft was modified or approved")
			}
		}
	}
	if count != 1 {
		t.Fatalf("customized draft duplicated: %d", count)
	}
}

func TestDefaultComplianceInstallerPreservesExistingLifecycle(t *testing.T) {
	service, forms, config, form := installedComplianceForm(t)
	actor := monitoring.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.ActorID}
	if _, err := forms.TransitionForm(context.Background(), actor, monitoring.TransitionInput{ID: form.ID, ProgramID: "program", LegalEntityID: config.LegalEntityID, ExpectedVersion: form.Version, To: monitoring.LifecyclePaused}); err != nil {
		t.Fatal(err)
	}
	for n := range 110 {
		_, err := forms.CreateForm(context.Background(), actor, monitoring.CreateFormInput{ProgramID: "program", LegalEntityID: config.LegalEntityID, Code: fmt.Sprintf("A-FORM-%03d", n), Name: "Other form", Purpose: "Collect a separate service answer.", Fields: []formcontract.Field{{ID: "answer", Label: "Service answer", Type: formcontract.TypeShortText}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	before, _ := forms.ListForms(context.Background(), actor, "program", 100)
	for range 2 {
		if err := service.ensureGovernedVendorForm(context.Background(), config, ReferenceThirdPartyRiskComplianceForm("program", config.LegalEntityID), "third-party risk compliance"); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := forms.ListForms(context.Background(), actor, "program", 100)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("repeat install changed an existing form")
	}
	reusable, err := forms.ListReusableForms(context.Background(), actor, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range reusable {
		if item.Code == vendorComplianceFormCode {
			t.Fatal("form outside the first 100 revisions was duplicated or reactivated")
		}
	}
}

func TestDefaultComplianceInstallerRejectsSelfApproval(t *testing.T) {
	config := normalizeSeedConfig(DemoSeedConfig())
	config.ReviewerPrincipalID = config.ActorID
	service := NewService(nil, nil)
	service.ConfigureMonitoring(monitoring.NewService(monitoring.NewMemoryRepository(), nil))
	if err := service.ensureVendorAcceptanceForms(context.Background(), config, "program"); !errors.Is(err, monitoring.ErrMakerChecker) {
		t.Fatalf("same maker/checker accepted: %v", err)
	}
}
