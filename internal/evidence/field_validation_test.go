package evidence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const testCaptureRecipient = "capture-recipient"

func TestCaptureFieldContractsRejectUnsupportedAndInvalidDefinitions(t *testing.T) {
	service, _, now := testCaptureService()
	base := testRequestInput(now, []Field{{ID: "ok", Label: "Known fact", Type: "text"}})

	unsupported := base
	unsupported.Fields = []Field{{ID: "scan", Label: "Biometric scan", Type: "biometric_scan", Required: true}}
	if _, err := service.CreateRequest(context.Background(), unsupported); err == nil || !strings.Contains(err.Error(), "unsupported field type") {
		t.Fatalf("expected unsupported field rejection, got %v", err)
	}

	badChoice := base
	badChoice.Fields = []Field{{ID: "state", Label: "State", Type: "single_select", Required: true, Options: []string{"Yes"}}}
	if _, err := service.CreateRequest(context.Background(), badChoice); err == nil || !strings.Contains(err.Error(), "2-50 choices") {
		t.Fatalf("expected invalid choice contract rejection, got %v", err)
	}

	badPhotoFormat := base
	badPhotoFormat.Fields = []Field{{ID: "photo", Label: "Site photo", Type: "photo", AcceptedFormats: []string{"application/pdf"}}}
	if _, err := service.CreateRequest(context.Background(), badPhotoFormat); err == nil || !strings.Contains(err.Error(), "non-image photo format") {
		t.Fatalf("expected invalid photo format rejection, got %v", err)
	}
}

func TestCaptureTypedAnswersRejectMalformedAndUnrequestedValues(t *testing.T) {
	service, _, now := testCaptureService()
	request, err := service.CreateRequest(context.Background(), testRequestInput(now, []Field{
		{ID: "review_date", Label: "Review date", Type: "date", Required: true},
		{ID: "amount", Label: "Amount", Type: "number", Required: true},
	}))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		answers map[string]string
		want    string
	}{
		{name: "invalid date", answers: map[string]string{"review_date": "1 March 2027", "amount": "5"}, want: "valid date"},
		{name: "invalid number", answers: map[string]string{"review_date": "2027-03-01", "amount": "five"}, want: "valid number"},
		{name: "unrequested field", answers: map[string]string{"review_date": "2027-03-01", "amount": "5", "hidden": "value"}, want: "unrequested field"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Submit(context.Background(), testSubmission(request, tc.answers))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestValidateAnswersRejectsHiddenAnswerAndIgnoresHiddenRequirement(t *testing.T) {
	service, _, _ := testCaptureService()
	request := Request{TenantID: "bank", Sections: []formcontract.Section{{ID: "vendor", Title: "Vendor"}}, Fields: []Field{
		{ID: "handles_data", SectionID: "vendor", Label: "Handles customer data", Type: "yes_no", Required: true},
		{ID: "data_location", SectionID: "vendor", Label: "Data location", Type: "short_text", Required: true, Condition: &formcontract.VisibilityCondition{FieldID: "handles_data", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}},
	}}
	if err := service.validateAnswers(context.Background(), request, map[string]formcontract.AnswerValue{"handles_data": formcontract.TextAnswer("No")}); err != nil {
		t.Fatalf("hidden required field should not be required: %v", err)
	}
	err := service.validateAnswers(context.Background(), request, map[string]formcontract.AnswerValue{
		"handles_data":  formcontract.TextAnswer("No"),
		"data_location": formcontract.TextAnswer("Lagos"),
	})
	if err == nil || !strings.Contains(err.Error(), "not requested for the current answers") {
		t.Fatalf("unexpected hidden answer error %v", err)
	}
}

func TestValidateAnswersEnforcesSmartTypesAndConstraints(t *testing.T) {
	service, _, _ := testCaptureService()
	minimum, maximum := 1.0, 100.0
	minSelections, maxSelections := 1, 2
	request := Request{TenantID: "bank", Fields: []Field{
		{ID: "email", Label: "Security contact email", Type: "email", Required: true},
		{ID: "website", Label: "Website", Type: "url", Required: true},
		{ID: "employees", Label: "Employees", Type: "integer", Required: true, Constraints: formcontract.Constraints{Minimum: &minimum, Maximum: &maximum}},
		{ID: "regions", Label: "Service regions", Type: "multi_select", Options: []string{"Nigeria", "Ghana", "Kenya"}, Constraints: formcontract.Constraints{MinSelections: &minSelections, MaxSelections: &maxSelections}},
		{ID: "confirm", Label: "Confirmation", Type: "attestation", Required: true, Attestation: "I confirm this response is complete."},
	}}
	tests := []struct {
		name    string
		answers map[string]formcontract.AnswerValue
		want    string
	}{
		{name: "email", answers: typedBaseAnswers("not-an-email", "https://vendor.example", "10", []string{"Nigeria"}, "Yes"), want: "valid email"},
		{name: "url", answers: typedBaseAnswers("security@vendor.example", "vendor.example", "10", []string{"Nigeria"}, "Yes"), want: "valid URL"},
		{name: "integer", answers: typedBaseAnswers("security@vendor.example", "https://vendor.example", "10.5", []string{"Nigeria"}, "Yes"), want: "whole number"},
		{name: "bounds", answers: typedBaseAnswers("security@vendor.example", "https://vendor.example", "101", []string{"Nigeria"}, "Yes"), want: "at most"},
		{name: "selection count", answers: typedBaseAnswers("security@vendor.example", "https://vendor.example", "10", []string{"Nigeria", "Ghana", "Kenya"}, "Yes"), want: "at most 2"},
		{name: "attestation", answers: typedBaseAnswers("security@vendor.example", "https://vendor.example", "10", []string{"Nigeria"}, "No"), want: "must be confirmed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := service.validateAnswers(context.Background(), request, test.answers); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func typedBaseAnswers(email, website, employees string, regions []string, attestation string) map[string]formcontract.AnswerValue {
	return map[string]formcontract.AnswerValue{
		"email":     formcontract.TextAnswer(email),
		"website":   formcontract.TextAnswer(website),
		"employees": formcontract.TextAnswer(employees),
		"regions":   {Values: regions},
		"confirm":   formcontract.TextAnswer(attestation),
	}
}

func TestCaptureSelectHandlesValidatedWhitespaceConsistently(t *testing.T) {
	service, _, now := testCaptureService()
	request, err := service.CreateRequest(context.Background(), testRequestInput(now, []Field{{ID: "present", Label: "ATM present", Type: " SINGLE_SELECT ", Required: true, Options: []string{" Yes ", "No"}}}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"present": " Yes "})); err != nil {
		t.Fatalf("expected server-valid normalized choice to submit, got %v", err)
	}
}

func TestCapturePhotoMustReferenceArtifactFromExactRequest(t *testing.T) {
	service, repo, now := testCaptureService()
	photoField := Field{ID: "photo", Label: "Site photo", Type: "photo", Required: true, AcceptedFormats: []string{"image/jpeg", "image/png"}}
	request, err := service.CreateRequest(context.Background(), testRequestInput(now, []Field{photoField}))
	if err != nil {
		t.Fatal(err)
	}
	otherRequest, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "ASSET", SubjectID: "atm-2", Title: "Other request",
		Purpose: "Verify another site.", WhyYou: "You were assigned the visit.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: testCaptureRecipient}, EstimatedMinutes: 3,
		Deadline: now.Add(time.Hour), Fields: []Field{{ID: "note", Label: "Note", Type: "text"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"photo": "made-up-artifact"})); err == nil || !strings.Contains(err.Error(), "uploaded for this request") {
		t.Fatalf("expected forged artifact rejection, got %v", err)
	}

	otherArtifact := Artifact{ID: "artifact-other", TenantID: "bank", RequestID: otherRequest.ID, FileName: "other.jpg", MediaType: "image/jpeg", SizeBytes: 1200, SHA256: "abc", StorageKey: "other", Status: ArtifactStoredUnscanned, CreatedAt: now}
	if _, err := repo.CreateArtifact(context.Background(), otherArtifact); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"photo": otherArtifact.ID})); err == nil || !strings.Contains(err.Error(), "uploaded for this request") {
		t.Fatalf("expected cross-request artifact rejection, got %v", err)
	}

	wrongMedia := Artifact{ID: "artifact-pdf", TenantID: "bank", RequestID: request.ID, FileName: "site.pdf", MediaType: "application/pdf", SizeBytes: 1200, SHA256: "def", StorageKey: "pdf", Status: ArtifactStoredUnscanned, CreatedAt: now}
	if _, err := repo.CreateArtifact(context.Background(), wrongMedia); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"photo": wrongMedia.ID})); err == nil || !strings.Contains(err.Error(), "JPEG or PNG") {
		t.Fatalf("expected wrong media rejection, got %v", err)
	}

	empty := Artifact{ID: "artifact-empty", TenantID: "bank", RequestID: request.ID, FileName: "empty.jpg", MediaType: "image/jpeg", SizeBytes: 0, SHA256: "empty", StorageKey: "empty", Status: ArtifactStoredUnscanned, CreatedAt: now}
	if _, err := repo.CreateArtifact(context.Background(), empty); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"photo": empty.ID})); err == nil || !strings.Contains(err.Error(), "empty file") {
		t.Fatalf("expected empty artifact rejection, got %v", err)
	}

	unknownState := Artifact{ID: "artifact-unknown", TenantID: "bank", RequestID: request.ID, FileName: "unknown.jpg", MediaType: "image/jpeg", SizeBytes: 1200, SHA256: "unknown", StorageKey: "unknown", Status: ArtifactStatus("UNKNOWN_STATE"), CreatedAt: now}
	if _, err := repo.CreateArtifact(context.Background(), unknownState); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"photo": unknownState.ID})); err == nil || !strings.Contains(err.Error(), "unavailable file") {
		t.Fatalf("expected unknown artifact status rejection, got %v", err)
	}

	photo := Artifact{ID: "artifact-photo", TenantID: "bank", RequestID: request.ID, FileName: "site.jpg", MediaType: "image/jpeg", SizeBytes: 1200, SHA256: "ghi", StorageKey: "photo", Status: ArtifactStoredUnscanned, CreatedAt: now}
	if _, err := repo.CreateArtifact(context.Background(), photo); err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"photo": photo.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != RequestSubmitted {
		t.Fatalf("expected submitted receipt, got %#v", receipt)
	}
}

func TestCaptureSignatureUsesBoundedPNGArtifactNotRawDataURL(t *testing.T) {
	service, repo, now := testCaptureService()
	request, err := service.CreateRequest(context.Background(), testRequestInput(now, []Field{{ID: "signature", Label: "Signature", Type: "signature", Required: true, AcceptedFormats: []string{"image/png"}}}))
	if err != nil {
		t.Fatal(err)
	}

	raw := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"signature": raw})); err == nil || !strings.Contains(err.Error(), "uploaded for this request") {
		t.Fatalf("expected raw signature payload rejection, got %v", err)
	}

	tooLarge := Artifact{ID: "signature-large", TenantID: "bank", RequestID: request.ID, FileName: "signature.png", MediaType: "image/png", SizeBytes: maxSignatureBytes + 1, SHA256: "large", StorageKey: "signature-large", Status: ArtifactStoredUnscanned, CreatedAt: now}
	if _, err := repo.CreateArtifact(context.Background(), tooLarge); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"signature": tooLarge.ID})); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized signature rejection, got %v", err)
	}

	signature := Artifact{ID: "signature-ok", TenantID: "bank", RequestID: request.ID, FileName: "signature.png", MediaType: "image/png", SizeBytes: 5400, SHA256: "ok", StorageKey: "signature", Status: ArtifactStoredUnscanned, CreatedAt: now}
	if _, err := repo.CreateArtifact(context.Background(), signature); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"signature": signature.ID})); err != nil {
		t.Fatalf("expected valid signature artifact, got %v", err)
	}
}

func testCaptureService() (*Service, *MemoryRepository, time.Time) {
	now := time.Date(2026, 8, 7, 18, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	return service, repo, now
}

func testRequestInput(now time.Time, fields []Field) CreateRequestInput {
	return CreateRequestInput{
		TenantID: "bank", SubjectType: "ASSET", SubjectID: "atm-1", Title: "Verify ATM location",
		Purpose: "Confirm the ATM after a physical visit.", WhyYou: "You were assigned the visit.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: testCaptureRecipient}, EstimatedMinutes: 3,
		Deadline: now.Add(time.Hour), KnownFacts: map[string]string{"address": "12 Admiralty Way"}, Fields: fields,
	}
}

func testSubmission(request Request, answers map[string]string) Submission {
	return Submission{
		TenantID: request.TenantID, RequestID: request.ID, SubmittedBy: testCaptureRecipient,
		Channel: "INTERNAL", ExpectedVersion: request.Version, Answers: formcontract.TextAnswers(answers),
	}
}
