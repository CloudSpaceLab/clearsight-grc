package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestQuestionQualitySurvivesRequestAndSubmission(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	for _, state := range []string{"match", "missing", "stale", "unavailable"} {
		t.Run(state, func(t *testing.T) {
			service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
			service.now = func() time.Time { return now }
			service.ConfigureSourceBindings(fakeBindingReader{
				revisions: map[string]sourceaccess.BindingRevision{"validate:3": bindingRevision("validate", "Approved staff register", sourceaccess.OperationLookup, "employee_id")},
				lookup: func(id string, _ int64, query sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
					if state == "unavailable" {
						return sourceaccess.LookupResult{}, sourceaccess.ErrConnection
					}
					observed := now
					if state == "stale" {
						observed = now.Add(-2 * time.Hour)
					}
					records := []sourceaccess.Record{}
					if state != "missing" {
						records = append(records, sourceaccess.Record{"employee_id": sourceaccess.StringValue("E-17")})
					}
					return sourceaccess.LookupResult{Records: records, Receipt: sourceReceipt(id, sourceaccess.OperationLookup, int64(len(records)), observed)}, nil
				},
			})
			request, err := service.CreateRequest(context.Background(), CreateRequestInput{
				TenantID: "tenant-a", SubjectType: "PROGRAM", SubjectID: "program-a",
				Title: "Confirm staff identifier", Purpose: "Review access evidence", WhyYou: "You own this access record",
				Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
				Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: testCaptureRecipient},
				Deadline:  now.Add(time.Hour), EstimatedMinutes: 2,
				Fields: []Field{
					{ID: "employee", Label: "Staff ID", Type: "short_text", Required: true, AnswerValidityDays: 7, ValidationBinding: &formcontract.ValidationBinding{BindingID: "validate", BindingVersion: 3, ValueField: "employee_id"}},
					{ID: "note", Label: "Access review note", Type: "short_text", Required: true, AnswerValidityDays: 30},
					{ID: "legacy", Label: "Additional note", Type: "short_text", Required: true},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if request.Fields[0].AnswerValidityDays != 7 || len(request.Fields[0].Bindings) != 1 {
				t.Fatal("question quality lost on request creation")
			}
			visible := RespondentRequest(request)
			if visible.Fields[0].ValidationBinding != nil || len(visible.Fields[0].Bindings) != 0 || visible.Fields[0].AnswerValidityDays != 7 {
				t.Fatal("respondent view exposed source configuration or lost validity")
			}
			receipt, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"employee": "E-17", "note": "Reviewed", "legacy": "Retained"}))
			if err != nil {
				t.Fatal(err)
			}
			submission, err := service.GetSubmissionForRequest(context.Background(), request.TenantID, request.ID)
			if err != nil || submission.ID != receipt.SubmissionID {
				t.Fatal("saved submission unavailable", err)
			}
			first, second := submission.AnswerProvenance["employee"], submission.AnswerProvenance["note"]
			if first.ExpiresAt == nil || !first.ExpiresAt.Equal(now.AddDate(0, 0, 7)) || second.ExpiresAt == nil || !second.ExpiresAt.Equal(now.AddDate(0, 0, 30)) {
				t.Fatal("independent expiry not retained")
			}
			if submission.AnswerProvenance["legacy"].ExpiresAt != nil {
				t.Fatal("legacy answer acquired an expiry")
			}
			expected := map[string]SourceResolutionState{"match": SourceResolutionCurrent, "missing": SourceResolutionNotFound, "stale": SourceResolutionStale, "unavailable": SourceResolutionUnavailable}[state]
			if len(first.Validations) != 1 || first.Validations[0].State != expected {
				t.Fatalf("source result: %+v", first.Validations)
			}
			if state == "match" && (first.Validations[0].Receipt == nil || first.Validations[0].Receipt.SourceID != "source-a") {
				t.Fatal("source receipt missing")
			}
			copy := cloneAnswerProvenance(submission.AnswerProvenance)
			*copy["employee"].ExpiresAt = now
			if first.ExpiresAt.Equal(now) {
				t.Fatal("mutable expiry escaped clone")
			}
		})
	}
}
