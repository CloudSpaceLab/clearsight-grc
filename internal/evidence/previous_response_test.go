package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestBuildPreviousResponsePrefillCopiesOnlyCompatibleScalars(t *testing.T) {
	submittedAt := time.Date(2026, 8, 14, 10, 32, 0, 0, time.UTC)
	previous := Request{ID: "request-1", Fields: []Field{
		{ID: "name", Type: "text"},
		{ID: "reviewed", Type: "date"},
		{ID: "rating", Type: "single_select", Options: []string{"Low", "High"}},
		{ID: "removed", Type: "text"},
		{ID: "certificate", Type: "file"},
		{ID: "site_photo", Type: "photo"},
		{ID: "signature", Type: "signature"},
	}}
	submission := Submission{ID: "submission-1", RequestID: previous.ID, SubmittedAt: submittedAt, Answers: map[string]string{
		"name": "Acme Processing Limited", "reviewed": "2026-08-14", "rating": "High", "removed": "old", "certificate": "artifact-1", "site_photo": "artifact-2", "signature": "artifact-3",
	}}
	next := []Field{
		{ID: "name", Type: "text"},
		{ID: "reviewed", Type: "date"},
		{ID: "rating", Type: "single_select", Options: []string{"Low", "High"}},
		{ID: "certificate", Type: "file"},
		{ID: "site_photo", Type: "photo"},
		{ID: "signature", Type: "signature"},
		{ID: "new_owner", Type: "text", Required: true},
	}

	got := BuildPreviousResponsePrefill(previous, submission, next)
	if len(got) != 3 {
		t.Fatalf("prefill = %#v", got)
	}
	if got["name"].Value != "Acme Processing Limited" || got["reviewed"].PreviousSubmissionID != submission.ID || !got["rating"].PreviousSubmittedAt.Equal(submittedAt) {
		t.Fatalf("scalar prefill provenance = %#v", got)
	}
	if got["name"].PreviousRequestID != previous.ID {
		t.Fatalf("previous request = %q", got["name"].PreviousRequestID)
	}
	for _, excluded := range []string{"removed", "certificate", "site_photo", "signature", "new_owner"} {
		if _, exists := got[excluded]; exists {
			t.Fatalf("%s was copied into successor prefill", excluded)
		}
	}
}

func TestBuildPreviousResponsePrefillRejectsChangedTypeAndOptions(t *testing.T) {
	previous := Request{ID: "request-1", Fields: []Field{
		{ID: "renamed_type", Type: "text"},
		{ID: "retired_choice", Type: "single_select", Options: []string{"Current", "Retired"}},
	}}
	submission := Submission{ID: "submission-1", RequestID: previous.ID, SubmittedAt: time.Date(2026, 8, 14, 10, 32, 0, 0, time.UTC), Answers: map[string]string{
		"renamed_type": "42", "retired_choice": "Retired",
	}}
	next := []Field{
		{ID: "renamed_type", Type: "number"},
		{ID: "retired_choice", Type: "single_select", Options: []string{"Current", "Pending"}},
	}

	if got := BuildPreviousResponsePrefill(previous, submission, next); len(got) != 0 {
		t.Fatalf("incompatible prefill = %#v", got)
	}
}

func TestPreviousResponseProvenanceTracksConfirmationAndCorrection(t *testing.T) {
	submittedAt := time.Date(2026, 8, 14, 10, 32, 0, 0, time.UTC)
	previous := PreviousResponseValue{Value: "Acme Processing Limited", PreviousRequestID: "request-1", PreviousSubmissionID: "submission-1", PreviousSubmittedAt: submittedAt}
	request := Request{TenantID: "tenant-a", Fields: []Field{{ID: "same", Type: "text"}, {ID: "changed", Type: "text"}}, PreviousResponses: map[string]PreviousResponseValue{"same": previous, "changed": previous}}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())

	got := service.deriveAnswerProvenance(context.Background(), request, map[string]string{"same": previous.Value, "changed": "Acme Payments Limited"})
	if got["same"].Origin != AnswerPriorResponsePrefilled {
		t.Fatalf("same origin = %s", got["same"].Origin)
	}
	changed := got["changed"]
	if changed.Origin != AnswerRespondentCorrected || changed.PreviousValue != previous.Value || changed.PreviousRequestID != previous.PreviousRequestID || changed.PreviousSubmissionID != previous.PreviousSubmissionID || changed.PreviousSubmittedAt == nil || !changed.PreviousSubmittedAt.Equal(submittedAt) {
		t.Fatalf("changed provenance = %#v", changed)
	}
}

func TestPreviousResponseProvenanceDefersToCurrentGovernedSource(t *testing.T) {
	submittedAt := time.Date(2026, 8, 14, 10, 32, 0, 0, time.UTC)
	previous := PreviousResponseValue{Value: "Previous branch", PreviousRequestID: "request-1", PreviousSubmissionID: "submission-1", PreviousSubmittedAt: submittedAt}
	sourceValue := sourceaccess.StringValue("Current branch")
	request := Request{TenantID: "tenant-a", Fields: []Field{{
		ID: "branch", Type: "text",
		Bindings:          []FieldBindingReference{{BindingID: "branch-register", BindingVersion: 2, Mode: BindingUsePrefill}},
		SourceResolutions: []SourceResolution{{Mode: BindingUsePrefill, BindingID: "branch-register", BindingVersion: 2, State: SourceResolutionCurrent, Value: &sourceValue}},
	}}, PreviousResponses: map[string]PreviousResponseValue{"branch": previous}}
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())

	got := service.deriveAnswerProvenance(context.Background(), request, map[string]string{"branch": "Current branch"})["branch"]
	if got.Origin != AnswerSourcePrefilled || got.PreviousRequestID != "" || got.PreviousValue != "" {
		t.Fatalf("source precedence provenance = %#v", got)
	}
}

func TestPreviousResponseLineageMustMatchImmutableSubmission(t *testing.T) {
	service := newOriginService()
	firstInput := validOriginRequestInput("bank-a")
	firstInput.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: "11111111-1111-7111-8111-111111111111", Sequence: 1}
	previous, err := service.CreateRequest(context.Background(), firstInput)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Submit(context.Background(), Submission{
		TenantID: previous.TenantID, RequestID: previous.ID, SubmittedBy: "respondent", Channel: "INTERNAL",
		ExpectedVersion: previous.Version, Answers: map[string]string{"answer": "Confirmed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	submission, err := service.GetSubmission(context.Background(), previous.TenantID, receipt.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}

	nextInput := validOriginRequestInput("bank-a")
	nextInput.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: firstInput.Origin.ID, Sequence: 2}
	nextInput.PredecessorRequestID = previous.ID
	nextInput.PreviousResponses = BuildPreviousResponsePrefill(previous, submission, nextInput.Fields)
	tampered := nextInput
	tampered.PreviousResponses = clonePreviousResponses(nextInput.PreviousResponses)
	tamperedValue := tampered.PreviousResponses["answer"]
	tamperedValue.Value = "Changed outside the submission"
	tampered.PreviousResponses["answer"] = tamperedValue
	if _, err := service.CreateRequest(context.Background(), tampered); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("tampered lineage error = %v", err)
	}

	next, err := service.CreateRequest(context.Background(), nextInput)
	if err != nil {
		t.Fatal(err)
	}
	if next.PreviousResponses["answer"].PreviousSubmissionID != submission.ID {
		t.Fatalf("stored previous response = %#v", next.PreviousResponses)
	}
}
