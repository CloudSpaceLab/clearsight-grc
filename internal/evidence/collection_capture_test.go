package evidence

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func captureHeldField(t *testing.T, now time.Time) Field {
	t.Helper()
	var field Field
	err := json.Unmarshal([]byte(`{"id":"held","label":"Security certificate","type":"vendor_document","required":true,"collection_resolution":{"id":"receipt-a","version":1,"source_artifact_request_id":"source-request","source":{"id":"source-occurrence","artifact_id":"held-artifact","request_id":"source-request","submission_id":"source-submission","field_id":"certificate","relationship_id":"subject-a","artifact_status":"AVAILABLE","current":true,"sha256":"source-digest","size_bytes":10},"document":{"artifact_id":"held-artifact","document_type":"Security certificate"},"reconciled_by":"bank-reviewer","reconciled_at":"`+now.Format(time.RFC3339)+`","rationale":"This certificate covers the requested service.","bank_review_state":"PENDING"}}`), &field)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

func TestCaptureCollectionReceiptClearsOnlyHeldUpload(t *testing.T) {
	svc, _, now := testCaptureService()
	req := Request{ID: "request", TenantID: "bank", Fields: []Field{captureHeldField(t, now), {ID: "owner", Label: "Service owner", Type: "short_text", Required: true}}}
	answers := map[string]formcontract.AnswerValue{"owner": formcontract.TextAnswer("Ada")}
	if err := svc.validateAnswers(context.Background(), req, answers); err != nil {
		t.Fatalf("held upload was still demanded: %v", err)
	}
	if _, exists := answers["held"]; exists {
		t.Fatal("receipt was inserted into respondent answers")
	}
	if err := svc.validateAnswers(context.Background(), req, nil); err == nil || !strings.Contains(err.Error(), "Service owner is required") {
		t.Fatalf("outstanding scalar was not enforced: %v", err)
	}
	answers["held"] = formcontract.AnswerValue{Document: &formcontract.DocumentAnswer{ArtifactID: "held-artifact", DocumentType: "Security certificate"}}
	if err := svc.validateAnswers(context.Background(), req, answers); err == nil || !strings.Contains(err.Error(), "uploaded for this request") {
		t.Fatalf("receipt bypassed respondent artifact ownership: %v", err)
	}
}

func TestVendorProgressSeparatesHeldEvidenceFromAnswers(t *testing.T) {
	now := time.Now().UTC()
	req := Request{Status: RequestReady, Fields: []Field{captureHeldField(t, now), {ID: "owner", Label: "Service owner", Type: "short_text", Required: true}}}
	row := vendorFormRow(req, nil, true, nil, now)
	if row.RequiredCount == nil || *row.RequiredCount != 2 || row.AnsweredRequired == nil || *row.AnsweredRequired != 0 || len(row.MissingFields) != 1 || row.MissingFields[0].ID != "owner" {
		t.Fatalf("held evidence was counted as a vendor answer or missing upload: %+v", row)
	}
	row = vendorFormRow(req, map[string]formcontract.AnswerValue{"owner": formcontract.TextAnswer("Ada")}, true, nil, now)
	if *row.AnsweredRequired != 1 || len(row.MissingFields) != 0 || row.ResponseState != "READY_TO_SUBMIT" {
		t.Fatalf("mixed collection did not become ready: %+v", row)
	}
}

func TestAllHeldCollectionIsNotOutstandingOrSubmitted(t *testing.T) {
	now := time.Now().UTC()
	req := Request{Status: RequestReady, SubjectID: "vendor", Deadline: now.Add(-time.Hour), Fields: []Field{captureHeldField(t, now)}}
	row := vendorFormRow(req, nil, true, nil, now)
	if row.ResponseState != "NO_VENDOR_ACTION" || vendorFormOutstanding(row) {
		t.Fatalf("all-held evidence still demands a vendor response: %+v", row)
	}
	summary := summarizeVendorForms("vendor", []VendorFormRow{row}, now)
	if summary.OutstandingForms != 0 || summary.OverdueForms != 0 || summary.SubmittedForms != 0 {
		t.Fatalf("collection changed vendor response totals: %+v", summary)
	}
}

func TestAllHeldCollectionDoesNotCreateEmptyVendorSubmission(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()
	_, req, err := fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	installCaptureHeldSource(t, fixture, req.ID)
	repo := fixture.distributions.repo
	repo.mu.Lock()
	r := repo.requests[req.ID]
	r.Fields = []Field{r.Fields[len(r.Fields)-1]}
	repo.requests[r.ID] = r
	repo.mu.Unlock()
	view, err := fixture.access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version}); err == nil {
		t.Fatal("all-held collection manufactured an empty vendor submission")
	}
	if len(repo.submissions) != 1 {
		t.Fatal("all-held response persisted respondent record")
	}
}

func TestVendorProgressUnknownConditionCannotClaimReady(t *testing.T) {
	now := time.Now().UTC()
	held := captureHeldField(t, now)
	held.Condition = &formcontract.VisibilityCondition{FieldID: "data", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}
	req := Request{Status: RequestReady, Fields: []Field{{ID: "data", Label: "Handles customer data", Type: "yes_no"}, held}}
	row := vendorFormRow(req, nil, true, nil, now)
	if row.RequiredCount != nil || row.ResponseState == "READY_TO_SUBMIT" {
		t.Fatalf("unresolved applicability appeared complete: %+v", row)
	}
	svc, _, _ := testCaptureService()
	if err := svc.validateAnswers(context.Background(), req, nil); err == nil {
		t.Fatal("unresolved applicability permitted final submission")
	}
}

func TestCollectionKnownSectionOmissionDoesNotDemandHiddenController(t *testing.T) {
	svc, _, now := testCaptureService()
	held := captureHeldField(t, now)
	held.SectionID = "documents"
	held.Condition = &formcontract.VisibilityCondition{FieldID: "regulated", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}
	req := Request{Sections: []formcontract.Section{{ID: "scope", Title: "Service scope"}, {ID: "documents", Title: "Service documents", Condition: &formcontract.VisibilityCondition{FieldID: "data", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}}}, Fields: []Field{{ID: "data", SectionID: "scope", Label: "Handles customer data", Type: "yes_no"}, {ID: "regulated", SectionID: "documents", Label: "Regulated service", Type: "yes_no"}, held}}
	answers := map[string]formcontract.AnswerValue{"data": formcontract.TextAnswer("No")}
	if err := svc.validateAnswers(context.Background(), req, answers); err != nil {
		t.Fatalf("known section omission demanded hidden applicability answer: %v", err)
	}
	row := vendorFormRow(req, answers, true, nil, now)
	if row.RequiredCount == nil || *row.RequiredCount != 0 {
		t.Fatalf("known non-applicability made the denominator unknown: %+v", row)
	}
}

func TestWorkspaceCollectionReceiptPreservesSourceAndRespondentProvenance(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()
	store := fixture.access.store.(*MemoryDistributionAccessStore)
	session, req, err := fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	installCaptureHeldSource(t, fixture, req.ID)
	view, err := fixture.access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	view, err = fixture.access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes")}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fixture.access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version})
	if err != nil {
		t.Fatalf("mixed response with bank-held upload could not submit: %v", err)
	}
	repo := store.distributions.repo
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	sub := repo.submissions[result.Submission.SubmissionID]
	if len(sub.Answers) != 1 || len(sub.AnswerProvenance) != 1 || sub.Answers["held"].Answered() {
		t.Fatalf("bank receipt was forged as respondent evidence: %+v", sub)
	}
	if repo.artifacts["held-artifact"].SubmissionID != "source-submission" || repo.artifacts["held-artifact"].RequestID == session.RequestID {
		t.Fatal("held artifact was reassigned to the vendor submission")
	}
	versionRecorded := false
	for _, event := range store.distributions.events {
		if event.EventType == "FORM_RESPONSE_SCORED_1" && event.Payload["request_id"] == req.ID && event.Payload["request_version"] == req.Version {
			versionRecorded = true
		}
	}
	if !versionRecorded {
		t.Fatal("submission audit cannot identify the collection request version consumed")
	}
}

func TestWorkspaceSubmissionRechecksReceiptAndRequestUnderLock(t *testing.T) {
	for _, change := range []string{"source quarantined", "request changed"} {
		t.Run(change, func(t *testing.T) {
			fixture, tokens := newTwoRecipientWorkspaceFixture(t)
			ctx := context.Background()
			store := fixture.access.store.(*MemoryDistributionAccessStore)
			session, req, err := fixture.access.SessionRequest(ctx, tokens[0])
			if err != nil {
				t.Fatal(err)
			}
			installCaptureHeldSource(t, fixture, req.ID)
			session, req, err = fixture.access.SessionRequest(ctx, tokens[0])
			if err != nil {
				t.Fatal(err)
			}
			view, err := fixture.access.GetResponseWorkspace(ctx, tokens[0])
			if err != nil {
				t.Fatal(err)
			}
			view, err = fixture.access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes")}}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.SubmitResponseWorkspace(ctx, workspaceSubmitCommand{Session: session, Request: req, Now: *fixture.now, Input: SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version}, Validate: func(map[string]formcontract.AnswerValue) error {
				repo := store.distributions.repo
				repo.mu.Lock()
				defer repo.mu.Unlock()
				if change == "source quarantined" {
					a := repo.artifacts["held-artifact"]
					a.Status = ArtifactQuarantined
					repo.artifacts[a.ID] = a
				} else {
					r := repo.requests[req.ID]
					r.Version++
					repo.requests[r.ID] = r
				}
				return nil
			}, BuildRevision: func(answers map[string]formcontract.AnswerValue) (ResponseRevision, error) {
				return buildResponseRevision(req, session.Assurance, nil, answers)
			}})
			if err == nil {
				t.Fatal("submission committed after collection state changed during validation")
			}
			if len(store.distributions.repo.submissions) != 1 {
				t.Fatal("rejected response persisted a submission")
			}
		})
	}
}

func installCaptureHeldSource(t *testing.T, fixture memoryAccessFixture, requestID string) {
	t.Helper()
	repo := fixture.distributions.repo
	repo.mu.Lock()
	defer repo.mu.Unlock()
	req := repo.requests[requestID]
	req.Fields = append(req.Fields, captureHeldField(t, *fixture.now))
	repo.requests[requestID] = req
	source := Request{ID: "source-request", TenantID: req.TenantID, LegalEntityID: req.LegalEntityID, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "subject-a", Fields: []Field{{ID: "certificate", Type: "vendor_document"}}, Origin: RequestOrigin{Type: "THIRD_PARTY_WORK", ID: "source-work", Version: 1}}
	repo.requests[source.ID] = source
	repo.submissions["source-submission"] = Submission{ID: "source-submission", TenantID: req.TenantID, LegalEntityID: req.LegalEntityID, RequestID: source.ID, SubmittedAt: *fixture.now, Answers: map[string]formcontract.AnswerValue{"certificate": {Document: &formcontract.DocumentAnswer{ArtifactID: "held-artifact"}}}}
	repo.artifacts["held-artifact"] = Artifact{ID: "held-artifact", TenantID: req.TenantID, RequestID: "source-request", SubmissionID: "source-submission", Status: ArtifactAvailable, SHA256: "source-digest", SizeBytes: 10}
}

func TestLegacySubmissionCannotConsumeCollectionReceipt(t *testing.T) {
	svc, repo, now := testCaptureService()
	req, err := svc.CreateRequest(context.Background(), testRequestInput(now, []Field{{ID: "owner", Label: "Service owner", Type: "short_text", Required: true}}))
	if err != nil {
		t.Fatal(err)
	}
	req.Fields = append(req.Fields, captureHeldField(t, now))
	repo.requests[req.ID] = req
	_, err = svc.Submit(context.Background(), testSubmission(req, map[string]string{"owner": "Ada"}))
	if err == nil || !strings.Contains(err.Error(), "secure form") {
		t.Fatalf("legacy submission did not require current secure form: %v", err)
	}
	if len(repo.submissions) != 0 {
		t.Fatal("legacy submission persisted receipt-backed response")
	}
}

func TestInvitedVendorReceivesOnlySafeCurrentCollectionNotice(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()
	_, req, err := fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	installCaptureHeldSource(t, fixture, req.ID)
	_, req, err = fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(req)
	if strings.Contains(string(raw), "source-submission") || strings.Contains(string(raw), "bank-reviewer") || strings.Contains(string(raw), "source-digest") {
		t.Fatalf("bank source details leaked to invited vendor: %s", raw)
	}
	if !strings.Contains(string(raw), `"collection_received":true`) {
		t.Fatal("safe already-held notice missing")
	}
	repo := fixture.distributions.repo
	repo.mu.Lock()
	a := repo.artifacts["held-artifact"]
	a.Status = ArtifactQuarantined
	repo.artifacts[a.ID] = a
	repo.mu.Unlock()
	_, req, err = fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(req)
	if strings.Contains(string(raw), `"collection_received":true`) {
		t.Fatal("unsafe source still displayed as collection received")
	}
}

type captureSourceReviewReader struct{ status string }

func (reader *captureSourceReviewReader) ReadCollectionSourceReview(_ context.Context, tenant, entity, assessment, request, artifact string) (DocumentReview, string, error) {
	if tenant != "tenant-a" || entity != "entity-a" || assessment != "source-assessment" || request != "source-request" || artifact != "held-artifact" {
		return DocumentReview{}, "", ErrNotFound
	}
	return DocumentReview{ID: "source-review", Status: reader.status, Source: "VENDOR_ASSESSMENT"}, "", nil
}

func TestWorkspaceReceiptReflectsLaterSourceReviewRejection(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()
	_, req, err := fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	installCaptureHeldSource(t, fixture, req.ID)
	repo := fixture.distributions.repo
	repo.mu.Lock()
	r := repo.requests[req.ID]
	r.Fields[len(r.Fields)-1].CollectionResolution.Source.AssessmentID = "source-assessment"
	repo.requests[r.ID] = r
	repo.mu.Unlock()
	reader := &captureSourceReviewReader{status: "VALIDATED"}
	NewService(repo, nil).ConfigureCollectionReviewReader(reader)
	_, req, err = fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	if !req.Fields[len(req.Fields)-1].CollectionReceived {
		t.Fatal("valid source review did not retain receipt")
	}
	reader.status = "REJECTED"
	_, req, err = fixture.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	if req.Fields[len(req.Fields)-1].CollectionReceived {
		t.Fatal("later source rejection still displayed collection received")
	}
	view, err := fixture.access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	view, err = fixture.access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes")}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version}); err == nil {
		t.Fatal("rejected source cleared required upload")
	}
}
