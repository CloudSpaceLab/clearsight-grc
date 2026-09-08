package evidence

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestVendorFormProgressKeepsSubmissionSeparateAndHidesDraftValues(t *testing.T) {
	req := Request{ID: "request", Title: "Security review", Status: RequestInProgress, Fields: []Field{
		{ID: "tested", Label: "Testing completed", Type: "yes_no", Required: true},
		{ID: "report", Label: "Testing report", Type: "short_text", Required: true, Condition: &formcontract.VisibilityCondition{FieldID: "tested", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}},
	}}
	answers := formcontract.TextAnswers(map[string]string{"tested": "Yes"})
	row := vendorFormRow(req, answers, true, nil, time.Now())
	if row.ResponseState != "IN_PROGRESS" || row.RequiredCount == nil || *row.RequiredCount != 2 || *row.AnsweredRequired != 1 || len(row.MissingFields) != 1 || row.MissingFields[0].ID != "report" {
		t.Fatalf("progress: %+v", row)
	}
	answers["report"] = formcontract.TextAnswer("private draft value")
	row = vendorFormRow(req, answers, true, nil, time.Now())
	if row.ResponseState != "READY_TO_SUBMIT" {
		t.Fatalf("saved answers must not mean submitted: %+v", row)
	}
	raw, _ := json.Marshal(row)
	if strings.Contains(string(raw), "private draft value") {
		t.Fatal("draft value leaked")
	}
	row = vendorFormRow(req, formcontract.TextAnswers(map[string]string{"tested": "No"}), true, nil, time.Now())
	if *row.RequiredCount != 1 || len(row.MissingFields) != 0 {
		t.Fatalf("hidden field counted: %+v", row)
	}
	row = vendorFormRow(req, nil, false, nil, time.Now())
	if row.RequiredCount != nil || row.AnsweredRequired != nil {
		t.Fatal("unknown progress must stay unknown")
	}
}

func TestVendorFormSummaryDoesNotTreatMissingOrProvisionalScoresAsLow(t *testing.T) {
	now := time.Now().UTC()
	due := now.Add(-time.Hour)
	high := 95.0
	rows := []VendorFormRow{
		{RequestID: "open", RelationshipID: "rel", ResponseState: "AWAITING_RESPONSE", Deadline: due},
		{RequestID: "poor", RelationshipID: "rel", ResponseState: "SUBMITTED", Score: &ResponseScoreResult{State: ResponseScoreFinal, Band: formcontract.ConcernHigh, AdverseScore: &high}},
		{RequestID: "missing", RelationshipID: "rel", ResponseState: "SUBMITTED"},
		{RequestID: "provisional", RelationshipID: "rel", ResponseState: "SUBMITTED", Score: &ResponseScoreResult{State: ResponseScoreProvisional, Band: formcontract.ConcernCritical, AdverseScore: &high}},
	}
	for i := range rows {
		rows[i].Current = true
	}
	summary := summarizeVendorForms("rel", rows, now)
	if summary.OutstandingForms != 1 || summary.OverdueForms != 1 || summary.SubmittedForms != 3 || summary.UnassessedForms != 2 || summary.HighestConcern != formcontract.ConcernHigh {
		t.Fatalf("summary: %+v", summary)
	}
}

func TestVendorFormsRejectUnboundedOrUnscopedQueries(t *testing.T) {
	service := NewDistributionService(nil)
	_, err := service.ListVendorForms(context.Background(), VendorFormsQuery{RelationshipIDs: []string{"rel"}, Limit: 25})
	if err == nil {
		t.Fatal("unscoped query accepted")
	}
	q := VendorFormsQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner", RelationshipIDs: make([]string, 51), Limit: 25}
	if normalizeVendorFormsQuery(&q) == nil {
		t.Fatal("unbounded vendor population accepted")
	}
}

func TestVendorFormsApplyAccessBeforePaginationAndCounts(t *testing.T) {
	now := time.Now().UTC()
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, nil, []RecipientCandidate{{PrincipalID: "owner", TenantID: "bank", Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"VENDOR_RELATIONSHIP:allowed": true}}})
	store := NewMemoryDistributionStore(repo, nil, nil)
	for _, v := range []struct {
		id, rel string
		delta   time.Duration
	}{{"hidden", "restricted", time.Hour}, {"latest", "allowed", 0}, {"older", "allowed", -time.Hour}} {
		repo.requests[v.id] = Request{ID: v.id, TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: v.rel, FormTemplateID: "form", Status: RequestInProgress, UpdatedAt: now.Add(v.delta), Deadline: now.Add(time.Hour), Fields: []Field{{ID: "report", Label: "Security testing report", Type: "short_text", Required: true}}}
	}
	repo.drafts["draft"] = ResponseDraft{ID: "draft", TenantID: "bank", RequestID: "latest", UpdatedAt: now, Answers: formcontract.TextAnswers(map[string]string{"report": "Confidential draft value"})}
	q := VendorFormsQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner", RelationshipIDs: []string{"allowed", "restricted"}, Limit: 1}
	page, err := store.ListVendorForms(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].RequestID != "latest" || page.NextCursor == "" {
		t.Fatalf("page=%+v", page)
	}
	if page.Items[0].ResponseState != "READY_TO_SUBMIT" {
		t.Fatalf("saved answer wrongly submitted: %+v", page.Items[0])
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "Confidential") {
		t.Fatal("draft leaked")
	}
	q.Cursor = page.NextCursor
	next, err := store.ListVendorForms(context.Background(), q)
	if err != nil || len(next.Items) != 1 || next.Items[0].RequestID != "older" {
		t.Fatalf("next=%+v err=%v", next, err)
	}
	summaries, err := store.VendorFormSummaries(context.Background(), q)
	if err != nil || summaries[0].OutstandingForms != 2 || summaries[1].OutstandingForms != 0 {
		t.Fatalf("counts=%+v err=%v", summaries, err)
	}
}

func TestVendorHighRiskFilterRequiresCurrentFinalAssessment(t *testing.T) {
	row := VendorFormRow{Current: true, ResponseState: "SUBMITTED", Score: &ResponseScoreResult{State: ResponseScoreFinal, Band: formcontract.ConcernHigh}, AssessmentState: "AWAITING_REVIEW"}
	q := VendorFormsQuery{Filter: "HIGH_RISK"}
	if vendorFormMatches(row, q, time.Now()) {
		t.Fatal("automatic result masked pending bank review")
	}
	row.AssessmentState = "ASSESSED"
	row.AssessedScore = &ResponseScoreResult{State: ResponseScoreFinal, Band: formcontract.ConcernHigh}
	if !vendorFormMatches(row, q, time.Now()) {
		t.Fatal("final bank risk missing")
	}
	row.Current = false
	if vendorFormMatches(row, q, time.Now()) {
		t.Fatal("historical risk presented as current")
	}
}
