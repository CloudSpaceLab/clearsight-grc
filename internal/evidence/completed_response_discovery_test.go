package evidence

import (
	"context"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"testing"
	"time"
)

func TestRoutedReviewerDiscoversOnlyPermittedSubmittedVendorResponses(t *testing.T) {
	now := time.Now().UTC()
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, nil, []RecipientCandidate{{PrincipalID: "owner", TenantID: "bank", Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"VENDOR_RELATIONSHIP:rel": true}}})
	store := NewMemoryDistributionStore(repo, nil, nil)
	field := Field{ID: "report", Label: "Test report", Type: "short_text", Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentManual, Weight: 100, ReviewerRole: "Risk", Rubric: []formcontract.AssessmentOutcome{{ID: "poor", Label: "Incomplete", Points: 80}}}}
	for i := 1; i <= 4; i++ {
		key := fmt.Sprint(i)
		req := Request{ID: "req" + key, TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "rel", FormTemplateID: "form", FormTemplateVersion: 1, Status: RequestSubmitted, Fields: []Field{field}, ScoringMode: formcontract.ScoringRisk, UpdatedAt: now.Add(time.Duration(i) * time.Minute)}
		if i == 4 {
			req.Origin.Type = "THIRD_PARTY_WORK"
			req.Origin.ID = "restricted-work"
		}
		repo.requests[req.ID] = req
		sub := Submission{ID: "sub" + key, TenantID: "bank", RequestID: req.ID, SubmittedBy: "supplier", Answers: formcontract.TextAnswers(map[string]string{"report": "supplied"}), SubmittedAt: req.UpdatedAt}
		repo.submissions[sub.ID] = sub
		d := FormDistribution{ID: "dist" + key, TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "rel", FormTemplateID: "form", FormTemplateVersion: 1, Status: DistributionCompleted}
		store.distributions[d.ID] = d
		store.requestDistribution[req.ID] = d.ID
		revision := completedRevision("response"+key, "bank", "entity", d.ID, req.UpdatedAt, 80, formcontract.ConcernHigh)
		revision.SubmissionID = sub.ID
		if i == 2 {
			revision.SubmittedRecipientID = "recipient2"
			revision.SubmittedRecipientType = RecipientExternalAudience
			revision.SubmittedRecipientLabel = "Vendor security lead"
			store.recipients[d.ID] = []memoryDistributionRecipient{{safe: DistributionRecipient{ID: "recipient2", TenantID: "bank", LegalEntityID: "entity", DistributionID: d.ID, RequestID: req.ID, Role: RecipientTo, Type: RecipientExternalAudience}}}
		}
		store.responseRevisions[d.ID] = []ResponseRevision{revision}
	}
	repo.requests["pending"] = Request{ID: "pending", TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "rel", FormTemplateID: "form", Status: RequestInProgress, UpdatedAt: now.Add(time.Hour)}
	revoked := false
	service := NewDistributionService(store).WithResponseDiscoveryAuthorizer(func(_ context.Context, a identity.Actor, r CompletedResponseSummary) error {
		if !revoked && a.PrincipalID == "reviewer" && (r.ID == "response1" || r.ID == "response2" || r.ID == "response4") {
			return nil
		}
		return ErrAssessmentForbidden
	})
	ctx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: now.Add(time.Hour)})
	q := CompletedResponseQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", CurrentOnly: true, Sort: ResponseSortNewest, Limit: 1}
	first, err := service.ListCompletedResponses(ctx, q)
	if err != nil || len(first.Items) != 1 || first.Items[0].ID != "response2" || first.NextCursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if first.Items[0].SubmittedBy != "supplier" {
		t.Fatalf("submitted person lost from authorized list: %+v", first.Items[0])
	}
	if first.Items[0].SubmissionRecipient == nil || first.Items[0].SubmissionRecipient.Label != "Vendor security lead" {
		t.Fatalf("recorded recipient missing from list: %+v", first.Items[0])
	}
	q.Cursor = first.NextCursor
	if detail, _, err := service.GetCompletedResponse(ctx, "bank", "entity", "reviewer", "response2"); err != nil || detail.SubmittedBy != "supplier" {
		t.Fatalf("permitted reviewer exact read failed: %v", err)
	}
	second, err := service.ListCompletedResponses(ctx, q)
	if err != nil || len(second.Items) != 1 || second.Items[0].ID != "response1" || second.NextCursor != "" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	vq := VendorFormsQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", RelationshipIDs: []string{"rel"}, Limit: 1}
	page, err := service.ListVendorForms(ctx, vq)
	if err != nil || len(page.Items) != 1 || page.Items[0].ResponseID != "response2" || page.NextCursor == "" {
		t.Fatalf("vendor=%+v err=%v", page, err)
	}
	summaries, err := service.VendorFormSummaries(ctx, vq)
	if err != nil || summaries[0].SubmittedForms != 2 || summaries[0].OutstandingForms != 0 {
		t.Fatalf("review counts=%+v err=%v", summaries, err)
	}
	ownerCtx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner", ExpiresAt: now.Add(time.Hour)})
	vq.PrincipalID = "owner"
	summaries, err = service.VendorFormSummaries(ownerCtx, vq)
	if err != nil || summaries[0].SubmittedForms != 3 || summaries[0].OutstandingForms != 1 {
		t.Fatalf("owner counts=%+v err=%v", summaries, err)
	}
	assigned := repo.requests["req2"]
	assigned.Recipient = Recipient{Type: RecipientInternalPrincipal, PrincipalID: "reviewer", State: RecipientStateAssigned}
	repo.requests["req2"] = assigned
	submitted := repo.submissions["sub2"]
	submitted.SubmittedBy = ""
	repo.submissions["sub2"] = submitted
	q.Cursor = ""
	conflicted, err := service.ListCompletedResponses(ctx, q)
	if err != nil || len(conflicted.Items) != 1 || conflicted.Items[0].ID != "response1" || conflicted.NextCursor != "" {
		t.Fatalf("assigned respondent discovered their own link response: %+v %v", conflicted, err)
	}
	assigned.Recipient = Recipient{}
	repo.requests["req2"] = assigned
	sub := repo.submissions["sub2"]
	sub.SubmittedBy = "reviewer"
	repo.submissions["sub2"] = sub
	vq.PrincipalID = "reviewer"
	summaries, err = service.VendorFormSummaries(ctx, vq)
	if err != nil || summaries[0].SubmittedForms != 1 {
		t.Fatalf("self-review discovery counts=%+v err=%v", summaries, err)
	}
	revoked = true
	if _, _, err := service.GetCompletedResponse(ctx, "bank", "entity", "reviewer", "response1"); err == nil {
		t.Fatal("revoked reviewer retained exact response access")
	}
	vq.PrincipalID = "reviewer"
	summaries, err = service.VendorFormSummaries(ctx, vq)
	if err != nil || summaries[0].SubmittedForms != 0 || summaries[0].OutstandingForms != 0 {
		t.Fatalf("revoked counts=%+v err=%v", summaries, err)
	}
	q.Cursor = ""
	first, err = service.ListCompletedResponses(ctx, q)
	if err != nil || len(first.Items) != 0 {
		t.Fatalf("revoked=%+v err=%v", first, err)
	}
}

func TestResponseReviewerDiscoversUnscoredVendorResponseWithoutRubric(t *testing.T) {
	now := time.Now().UTC()
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, nil, []RecipientCandidate{{PrincipalID: "owner", TenantID: "bank", Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"VENDOR_RELATIONSHIP:rel": true}}})
	store := NewMemoryDistributionStore(repo, nil, nil)
	req := Request{ID: "request", TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "rel", FormTemplateID: "form", FormTemplateVersion: 1, Status: RequestSubmitted, Fields: []Field{{ID: "answer", Label: "Answer", Type: "short_text"}}, ScoringMode: formcontract.ScoringNone}
	repo.requests[req.ID] = req
	repo.submissions["submission"] = Submission{ID: "submission", TenantID: "bank", RequestID: req.ID, SubmittedBy: "supplier", Answers: formcontract.TextAnswers(map[string]string{"answer": "Supplied"}), SubmittedAt: now}
	d := FormDistribution{ID: "distribution", TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "rel", FormTemplateID: "form", FormTemplateVersion: 1, Status: DistributionCompleted}
	store.distributions[d.ID] = d
	store.requestDistribution[req.ID] = d.ID
	rev := completedRevision("response", "bank", "entity", d.ID, now, 0, formcontract.ConcernLow)
	rev.SubmissionID = "submission"
	rev.Score = &ResponseScoreResult{State: ResponseScoreNotConfigured}
	store.responseRevisions[d.ID] = []ResponseRevision{rev}
	revoked := false
	service := NewDistributionService(store).WithResponseReviewAuthorizer(func(_ context.Context, actor identity.Actor, response CompletedResponseSummary) (string, error) {
		if !revoked && actor.PrincipalID == "reviewer" && response.ID == "response" {
			return "policy:route", nil
		}
		return "", ErrAssessmentForbidden
	})
	ctx := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: now.Add(time.Hour)})
	query := CompletedResponseQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", CurrentOnly: true, Limit: 10}
	page, err := service.ListCompletedResponses(ctx, query)
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != rev.ID {
		t.Fatalf("response discovery=%+v err=%v", page, err)
	}
	if _, err := service.GetCompletedResponseAnswers(ctx, "bank", "entity", "reviewer", rev.ID); err != nil {
		t.Fatalf("reviewer answer read: %v", err)
	}
	revoked = true
	if _, _, err := service.GetCompletedResponse(ctx, "bank", "entity", "reviewer", rev.ID); err == nil {
		t.Fatal("revoked route retained access")
	}
}
