package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func vendorCustomComplianceForm() DistributionFormRevision {
	form := activeDistributionForm()
	form.ScoringMode = formcontract.ScoringCompliance
	form.Sections[0].Weight = 100
	form.Fields = nil
	form.ScoreProfile = &formcontract.ScoreProfile{Version: "custom-contract-v7", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, Bands: formcontract.DefaultConcernBands()}
	for _, id := range []string{"assurance", "audit"} {
		form.Fields = append(form.Fields, formcontract.Field{ID: id, SectionID: "general", Label: "Contracted " + id, Type: formcontract.TypeYesNo, Required: true, Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentAutomaticReview, Required: true, Weight: 50, ReviewerRole: "Vendor compliance reviewer", Rubric: []formcontract.AssessmentOutcome{{ID: "met", Label: "Evidence accepted", Points: 100}, {ID: "not-met", Label: "Evidence rejected", Points: 0}}}})
		form.ScoreProfile.Contributions = append(form.ScoreProfile.Contributions, formcontract.ScoreContribution{ID: id + "-check", Label: "Required " + id + " condition", Weight: 50, Predicate: formcontract.Predicate{FieldID: id, Operator: formcontract.PredicateEquals, Values: []string{"Yes"}}, MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingIndeterminate})
	}
	form.ScoreProfile.Rules = []formcontract.ScoreRule{{ID: "critical-assurance", Label: "Required independent assurance absent", Predicate: formcontract.Predicate{FieldID: "assurance", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, Effect: formcontract.RuleEffect{Kind: formcontract.EffectFloor, Value: 100}}}
	return form
}

func TestVendorCustomFormDistributionSubmissionShowsFailuresBeforeReview(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	form := vendorCustomComplianceForm()
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": testSecurityKey(0x31)})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMemoryRepository(nil, nil)
	store := NewMemoryDistributionStore(repo, stubDistributionFormReader{form: form}, keyring)
	service := NewDistributionService(store)
	bundle, err := service.Create(ctx, CreateDistributionInput{TenantID: form.TenantID, LegalEntityID: form.LegalEntityID, FormTemplateID: form.ID, FormTemplateVersion: form.Version, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "custom-vendor-relationship", Title: "Contract security review", Purpose: "Provide the contracted assurance status.", AccessPolicy: AccessSharedEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(2 * time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: "bank-owner", Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "Vendor contact"}}})
	if err != nil {
		t.Fatal(err)
	}
	delivery := &recordingOTPDelivery{}
	access, err := NewDistributionAccessService(NewMemoryDistributionAccessStore(store), keyring, delivery, testSecurityKey(0x42), 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := access.IssueDistributionAccessRoutes(ctx, form.TenantID, form.LegalEntityID, bundle.Distribution.ID, "bank-owner")
	if err != nil || len(issued) != 1 {
		t.Fatalf("routes %+v %v", issued, err)
	}
	start, err := access.StartDistributionAccess(ctx, issued[0].Selector)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := access.SendOTP(ctx, issued[0].Selector, start.Recipients[0].SelectorID)
	if err != nil {
		t.Fatal(err)
	}
	redeemed, err := access.VerifyOTP(ctx, issued[0].Selector, challenge.ChallengeID, delivery.values[len(delivery.values)-1].Code)
	if err != nil {
		t.Fatal(err)
	}
	view, err := access.GetResponseWorkspace(ctx, redeemed.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	view, err = access.SaveResponseWorkspace(ctx, redeemed.SessionToken, SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "assurance", Value: formcontract.TextAnswer("No"), BaseSequence: view.FieldSequences["assurance"]}, {FieldID: "audit", Value: formcontract.TextAnswer("Yes"), BaseSequence: view.FieldSequences["audit"]}}})
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := access.SubmitResponseWorkspace(ctx, redeemed.SessionToken, SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version})
	if err != nil {
		t.Fatal(err)
	}
	if submitted.Revision.Score == nil || submitted.Revision.Score.AssessmentReviewCount != 2 {
		t.Fatalf("submission failed to retain configured review %+v", submitted.Revision)
	}
	q := VendorFormsQuery{TenantID: form.TenantID, LegalEntityID: form.LegalEntityID, PrincipalID: "bank-owner", RelationshipIDs: []string{"custom-vendor-relationship"}, Limit: 25}
	page, err := service.ListVendorForms(ctx, q)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page %+v %v", page, err)
	}
	row := page.Items[0]
	if row.ResponseState != "SUBMITTED" || row.AssessmentState != "AWAITING_REVIEW" || len(row.AttentionItems) != 2 {
		t.Fatalf("submitted configured failures missing %+v", row)
	}
	for _, item := range row.AttentionItems {
		if item.Source != "RESPONSE" || item.State != "GAP" {
			t.Fatalf("premature review %+v", item)
		}
	}
	summary, err := service.VendorFormSummaries(ctx, q)
	if err != nil || summary[0].OutstandingForms != 0 || summary[0].SubmittedForms != 1 || summary[0].AwaitingReview != 1 || summary[0].UnassessedForms != 1 {
		t.Fatalf("completion/review mixed %+v %v", summary, err)
	}
}

func TestVendorPartialReviewKeepsUnreviewedAutomaticFailuresAsResponse(t *testing.T) {
	form := vendorCustomComplianceForm()
	req := Request{Status: RequestSubmitted, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections}
	for _, f := range form.Fields {
		req.Fields = append(req.Fields, Field{ID: f.ID, SectionID: f.SectionID, Label: f.Label, Type: string(f.Type), Required: f.Required, Assessment: f.Assessment})
	}
	answers := formcontract.TextAnswers(map[string]string{"assurance": "No", "audit": "Yes"})
	revision, err := buildResponseRevision(req, AccessAssurance(""), nil, answers)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := workspaceScoringContract(req)
	if err != nil {
		t.Fatal(err)
	}
	assessed, err := formcontract.EvaluateAssessment(contract, answers, map[string]string{"audit": "met"})
	if err != nil {
		t.Fatal(err)
	}
	row := vendorFormRow(req, answers, true, &revision, time.Now().UTC())
	row.AssessedScore = &ResponseScoreResult{Mode: form.ScoringMode, Direction: formcontract.DirectionLowIsPoor, State: ResponseScoreProvisional, ContributionResults: assessed.ContributionResults, RuleResults: assessed.RuleResults}
	row.reviewedFields = map[string]bool{"audit": true}
	populateVendorFormAttention(req, answers, true, &row, nil, time.Now().UTC())
	if len(row.AttentionItems) != 2 {
		t.Fatalf("automatic failures disappeared %+v", row.AttentionItems)
	}
	for _, item := range row.AttentionItems {
		if item.Source != "RESPONSE" {
			t.Fatalf("unreviewed finding relabelled %+v", item)
		}
	}
	// Completing the failing field's independent review changes contribution
	// provenance; answer-based rules retain their submitted-response source.
	assessed, err = formcontract.EvaluateAssessment(contract, answers, map[string]string{"audit": "met", "assurance": "met"})
	if err != nil {
		t.Fatal(err)
	}
	row.AssessedScore.ContributionResults = assessed.ContributionResults
	row.reviewedFields["assurance"] = true
	populateVendorFormAttention(req, answers, true, &row, nil, time.Now().UTC())
	if len(row.AttentionItems) != 2 {
		t.Fatalf("critical response or reviewed failure erased %+v", row.AttentionItems)
	}
	for _, item := range row.AttentionItems {
		want := "RESPONSE"
		if item.RuleID == "assurance-check" {
			want = "REVIEW"
		}
		if item.Source != want {
			t.Fatalf("wrong finding source %+v", item)
		}
	}
}

func TestVendorLegacyCustomFormDerivesAttentionWithoutInventingAssessment(t *testing.T) {
	form := vendorCustomComplianceForm()
	now := time.Now().UTC()
	req := Request{ID: "legacy-request", TenantID: "tenant-a", LegalEntityID: "entity-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship", FormTemplateID: form.ID, FormTemplateVersion: form.Version, Status: RequestSubmitted, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections}
	for _, f := range form.Fields {
		req.Fields = append(req.Fields, Field{ID: f.ID, SectionID: f.SectionID, Label: f.Label, Type: string(f.Type), Required: f.Required, Assessment: f.Assessment})
	}
	req.Origin.Type = "THIRD_PARTY_ASSESSMENT"
	req.Origin.ID = "assessment"
	req.Origin.Version = 1
	repo := NewMemoryRepository(nil, []Request{req})
	repo.submissions["legacy-sub"] = Submission{ID: "legacy-sub", TenantID: req.TenantID, RequestID: req.ID, SubmittedAt: now, Answers: formcontract.TextAnswers(map[string]string{"assurance": "No", "audit": "Yes"})}
	store := NewMemoryDistributionStore(repo, nil, nil)
	store.documentContexts = documentContextFunc(func(context.Context, DocumentQuery, Request, Submission) (DocumentContext, error) {
		return DocumentContext{AssessmentID: "assessment", RelationshipID: "relationship"}, nil
	})
	q := VendorFormsQuery{TenantID: req.TenantID, LegalEntityID: req.LegalEntityID, PrincipalID: "owner", RelationshipIDs: []string{"relationship"}, Limit: 25}
	page, err := store.ListVendorForms(context.Background(), q)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page %+v %v", page, err)
	}
	row := page.Items[0]
	if len(row.AttentionItems) != 2 || row.Score != nil || row.AssessedScore != nil || row.ResponseState != "SUBMITTED" {
		t.Fatalf("legacy failure or assessment wrong %+v", row)
	}
	summary, err := store.VendorFormSummaries(context.Background(), q)
	if err != nil || summary[0].UnassessedForms != 1 || summary[0].AssessedForms != 0 {
		t.Fatalf("invented assessment %+v %v", summary, err)
	}
}
