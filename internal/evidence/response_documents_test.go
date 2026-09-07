package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestDocumentFileKindUsesValidatedNormalizedMediaType(t *testing.T) {
	for _, tt := range []struct {
		media string
		want  DocumentFileKind
	}{
		{" Application/PDF ; charset=binary ", DocumentPDF}, {"image/png", DocumentImage},
		{"application/msword", DocumentWord}, {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", DocumentWord},
		{"text/csv; charset=utf-8", DocumentSpreadsheet}, {"application/vnd.ms-excel", DocumentSpreadsheet},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", DocumentSpreadsheet},
		{"image/png; invalid", DocumentOther}, {"image/svg+xml", DocumentOther}, {"application/zip", DocumentOther},
	} {
		if got := DocumentKindForMediaType(tt.media); got != tt.want {
			t.Errorf("%q: %s, want %s", tt.media, got, tt.want)
		}
	}
}

func documentMemoryFixture() (*MemoryDistributionStore, DocumentQuery) {
	repo := NewMemoryRepository(nil, nil)
	repo.candidates["reader"] = RecipientCandidate{PrincipalID: "reader", TenantID: "tenant", Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program": true}}
	store := NewMemoryDistributionStore(repo, nil, nil)
	store.distributions["distribution"] = FormDistribution{ID: "distribution", TenantID: "tenant", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", Title: "Annual review", FormTemplateID: "form", FormTemplateVersion: 1}
	repo.requests["request"] = Request{ID: "request", TenantID: "tenant", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", Title: "Annual review", FormTemplateID: "form", FormTemplateVersion: 1, Fields: []Field{{ID: "file", Label: "Policy", Type: "file"}, {ID: "photo", Label: "Site", Type: "photo"}}}
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{"older", "newer"} {
		repo.submissions[id] = Submission{ID: id, TenantID: "tenant", LegalEntityID: "entity", RequestID: "request", SubmittedBy: id, SubmittedAt: now.Add(time.Duration(i) * time.Hour), Answers: map[string]formcontract.AnswerValue{"file": {ArtifactIDs: []string{"artifact"}}}}
		store.responseRevisions["distribution"] = append(store.responseRevisions["distribution"], ResponseRevision{ID: "revision-" + id, TenantID: "tenant", LegalEntityID: "entity", DistributionID: "distribution", SubmissionID: id, Current: i == 1})
	}
	repo.artifacts["artifact"] = Artifact{ID: "artifact", TenantID: "tenant", RequestID: "request", SubmissionID: "older", FileName: "policy.pdf", MediaType: "application/pdf", SizeBytes: 10, Status: ArtifactAvailable, StorageKey: "never-expose", CreatedAt: now.Add(-time.Hour), CreatedBy: "uploader"}
	repo.artifacts["draft"] = Artifact{ID: "draft", TenantID: "tenant", RequestID: "request", FileName: "draft.pdf", MediaType: "application/pdf"}
	return store, DocumentQuery{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reader", Limit: 1}
}

func TestDocumentInventoryPreservesReusedArtifactOccurrencesAndHistory(t *testing.T) {
	store, q := documentMemoryFixture()
	first, err := store.ListDocuments(context.Background(), q)
	if err != nil || len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page: %+v %v", first, err)
	}
	if first.Items[0].SubmissionID != "newer" || first.Items[0].UploadedBy != "uploader" || first.Items[0].SubmittedBy != "newer" {
		t.Fatalf("wrong occurrence: %+v", first.Items[0])
	}
	q.Cursor = first.NextCursor
	second, err := store.ListDocuments(context.Background(), q)
	if err != nil || len(second.Items) != 1 || second.Items[0].SubmissionID != "older" || second.NextCursor != "" {
		t.Fatalf("history: %+v %v", second, err)
	}
	q.Cursor = ""
	q.CurrentOnly = true
	current, err := store.ListDocuments(context.Background(), q)
	if err != nil || len(current.Items) != 1 || current.NextCursor != "" {
		t.Fatalf("current: %+v %v", current, err)
	}
	q.PrincipalID = "intruder"
	denied, err := store.ListDocuments(context.Background(), q)
	if err != nil || len(denied.Items) != 0 {
		t.Fatalf("denied: %+v %v", denied, err)
	}
}

func TestDocumentQueryRejectsMissingScopeAndInvalidFilters(t *testing.T) {
	_, q := documentMemoryFixture()
	for _, change := range []func(*DocumentQuery){func(q *DocumentQuery) { q.PrincipalID = "" }, func(q *DocumentQuery) { q.LegalEntityID = "*" }, func(q *DocumentQuery) { q.Limit = 101 }, func(q *DocumentQuery) { q.FileKind = "VIDEO" }, func(q *DocumentQuery) { q.Cursor = "invalid" }} {
		probe := q
		change(&probe)
		if _, err := normalizeDocumentQuery(&probe); err == nil {
			t.Fatalf("accepted invalid query %+v", probe)
		}
	}
}

func TestCompletedResponseAnswersComeFromExactHistoricalSubmission(t *testing.T) {
	store, q := documentMemoryFixture()
	result, err := NewDistributionService(store).GetCompletedResponseAnswers(context.Background(), q.TenantID, q.LegalEntityID, q.PrincipalID, "revision-older")
	if err != nil || result.SubmissionID != "older" || len(result.Answers) != 2 || len(result.Answers[0].Value.ArtifactIDs) != 1 {
		t.Fatalf("historical answers %+v %v", result, err)
	}
	if _, err := NewDistributionService(store).GetCompletedResponseAnswers(context.Background(), q.TenantID, q.LegalEntityID, "intruder", "revision-older"); err == nil {
		t.Fatal("intruder read answers")
	}
	request := store.repo.requests["request"]
	request.SubjectID = "other-program"
	store.repo.requests[request.ID] = request
	if _, err := NewDistributionService(store).GetCompletedResponseAnswers(context.Background(), q.TenantID, q.LegalEntityID, q.PrincipalID, "revision-older"); err == nil {
		t.Fatal("response exposed answers from mismatched subject")
	}
}

type documentContextFunc func(context.Context, DocumentQuery, Request, Submission) (DocumentContext, error)

func (f documentContextFunc) ResolveDocumentContext(ctx context.Context, q DocumentQuery, r Request, s Submission) (DocumentContext, error) {
	return f(ctx, q, r, s)
}
func TestMemoryLegacyDocumentsRequireOwningWorkflowReadContext(t *testing.T) {
	store, q := documentMemoryFixture()
	store.responseRevisions = map[string][]ResponseRevision{}
	req := store.repo.requests["request"]
	req.Origin = RequestOrigin{Type: "THIRD_PARTY_ASSESSMENT", ID: "assessment", Version: 1}
	store.repo.requests[req.ID] = req
	store.documentContexts = documentContextFunc(func(_ context.Context, q DocumentQuery, r Request, s Submission) (DocumentContext, error) {
		if q.PrincipalID != "reader" {
			return DocumentContext{}, ErrNotFound
		}
		return DocumentContext{RelationshipID: "relationship", AssessmentID: "assessment", Current: s.ID == "newer"}, nil
	})
	q.CurrentOnly = true
	q.RelationshipID = "relationship"
	files, err := store.ListDocuments(context.Background(), q)
	if err != nil || len(files.Items) != 1 || files.Items[0].AssessmentID != "assessment" {
		t.Fatalf("legacy files %+v %v", files, err)
	}
	q.PrincipalID = "intruder"
	files, err = store.ListDocuments(context.Background(), q)
	if err != nil || len(files.Items) != 0 {
		t.Fatalf("unauthorized legacy files %+v %v", files, err)
	}
}

func TestDocumentInventoryIncludesPhotoAndVendorDocumentAcrossContributors(t *testing.T) {
	store, q := documentMemoryFixture()
	q.Limit = 100
	q.CurrentOnly = true
	request := store.repo.requests["request"]
	request.Fields = append(request.Fields, Field{ID: "certificate", Label: "Certificate", Type: "vendor_document"})
	store.repo.requests[request.ID] = request
	submission := store.repo.submissions["newer"]
	submission.Answers["photo"] = formcontract.AnswerValue{ArtifactIDs: []string{"image"}}
	submission.Answers["certificate"] = formcontract.AnswerValue{Document: &formcontract.DocumentAnswer{ArtifactID: "certificate", ExpiresOn: "2025-01-01"}}
	store.repo.submissions[submission.ID] = submission
	store.repo.requests["contributor"] = Request{ID: "contributor", TenantID: q.TenantID, LegalEntityID: q.LegalEntityID}
	store.requestDistribution["contributor"] = "distribution"
	store.repo.artifacts["image"] = Artifact{ID: "image", TenantID: q.TenantID, RequestID: "contributor", FileName: "site.png", MediaType: "image/png", Status: ArtifactQuarantined}
	store.repo.artifacts["certificate"] = Artifact{ID: "certificate", TenantID: q.TenantID, RequestID: "request", FileName: "certificate.pdf", MediaType: "application/pdf", Status: ArtifactAvailable}
	files, err := store.ListDocuments(context.Background(), q)
	if err != nil || len(files.Items) != 3 {
		t.Fatalf("all field types %+v %v", files, err)
	}
	q.FileKind = DocumentImage
	files, err = store.ListDocuments(context.Background(), q)
	if err != nil || len(files.Items) != 1 || files.Items[0].RequestID != "request" || files.Items[0].ArtifactRequestID != "contributor" || files.Items[0].UploadedBy != "" {
		t.Fatalf("contributor occurrence %+v %v", files, err)
	}
	store.requestDistribution["contributor"] = "unrelated"
	files, err = store.ListDocuments(context.Background(), q)
	if err != nil || len(files.Items) != 0 {
		t.Fatalf("other distribution artifact %+v %v", files, err)
	}
}
