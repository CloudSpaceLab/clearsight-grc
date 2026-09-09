package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestCollectionRefreshDetectsLaterSourceFieldSubmission(t *testing.T) {
	for _, replacementField := range []string{"certificate", "other"} {
		t.Run(replacementField, func(t *testing.T) {
			repo := NewMemoryRepository(nil, nil)
			NewMemoryDistributionStore(repo, nil, nil)
			now := time.Now().UTC()
			source := Request{ID: "source-request", TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "subject-a", FormTemplateID: "form", FormTemplateVersion: 1, Fields: []Field{{ID: "certificate", Type: "vendor_document"}}}
			source.Origin.Type, source.Origin.ID, source.Origin.Version = "THIRD_PARTY_WORK", "work", 1
			repo.requests[source.ID] = source
			repo.submissions["source-submission"] = Submission{ID: "source-submission", TenantID: "bank", RequestID: source.ID, SubmittedAt: now, Answers: map[string]formcontract.AnswerValue{"certificate": {Document: &formcontract.DocumentAnswer{ArtifactID: "held-artifact"}}}}
			held := captureHeldField(t, now)
			held.CollectionResolution.Source.WorkRequestID = "work"
			target := Request{TenantID: "bank", LegalEntityID: "entity", Fields: []Field{held}}
			before, err := repo.RefreshCollectionRequestReviews(context.Background(), target)
			if err != nil || !CollectionFieldFulfilled(before.Fields[0], now) {
				t.Fatalf("current source not received: %v", err)
			}
			newer := source
			newer.ID, newer.Origin.Version = "replacement-request", 2
			newer.Fields = []Field{{ID: replacementField, Type: "vendor_document"}}
			repo.requests[newer.ID] = newer
			// A newer requested field replaces the source even when its answer is omitted.
			repo.submissions["replacement-submission"] = Submission{ID: "replacement-submission", TenantID: "bank", RequestID: newer.ID, SubmittedAt: now.Add(time.Minute)}
			after, err := repo.RefreshCollectionRequestReviews(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			wantCurrent := replacementField != "certificate"
			if after.Fields[0].CollectionResolution.Source.Current != wantCurrent || CollectionFieldFulfilled(after.Fields[0], now) != wantCurrent {
				t.Fatalf("replacement currency not applied: %+v", after.Fields[0].CollectionResolution)
			}
			if !target.Fields[0].CollectionResolution.Source.Current {
				t.Fatal("refresh changed historical receipt")
			}
		})
	}
}
