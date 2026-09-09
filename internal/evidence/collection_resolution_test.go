package evidence

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"testing"
	"time"
)

func TestBankCollectionMutationRechecksSourceBeforeChangingAssessment(t *testing.T) {
	f, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()
	_, request, err := f.access.SessionRequest(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	installCaptureHeldSource(t, f, request.ID)
	repo := f.distributions.repo
	repo.mu.Lock()
	request = repo.requests[request.ID]
	candidate := *request.Fields[len(request.Fields)-1].CollectionResolution
	artifact := repo.artifacts[candidate.Source.ArtifactID]
	artifact.Status = ArtifactQuarantined
	repo.artifacts[artifact.ID] = artifact
	repo.mu.Unlock()
	changed := false
	_, err = repo.MutateCollectionRequest(ctx, request.TenantID, request.ID, request.Version, &candidate, func(*Request) error {
		if !CollectionFieldFulfilled(Field{Type: "vendor_document", CollectionResolution: &candidate}, time.Now()) {
			return ErrNotFound
		}
		changed = true
		return nil
	})
	if err == nil || changed {
		t.Fatal("source changed but bank mutation accepted stale candidate")
	}
}

func TestCollectionSourceMustMeetRequestedFileConstraints(t *testing.T) {
	limit := int64(100)
	field := Field{Type: "vendor_document", AcceptedFormats: []string{"application/pdf"}, Constraints: formcontract.Constraints{MaxFileBytes: &limit}}
	source := DocumentOccurrence{ArtifactStatus: ArtifactAvailable, MediaType: "application/pdf", SizeBytes: 100}
	if err := ValidateCollectionSourceForField(field, source); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*DocumentOccurrence){func(s *DocumentOccurrence) { s.MediaType = "image/png" }, func(s *DocumentOccurrence) { s.SizeBytes = 101 }, func(s *DocumentOccurrence) { s.SizeBytes = 0 }} {
		probe := source
		change(&probe)
		if ValidateCollectionSourceForField(field, probe) == nil {
			t.Fatalf("incompatible source allowed: %+v", probe)
		}
	}
}

func TestCollectionResolutionClearsCollectionWithoutAcceptingDocument(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	field := Field{ID: "certificate", Type: "vendor_document", Required: true, CollectionResolution: &CollectionResolution{ID: "receipt", Version: 1, Source: DocumentOccurrence{ArtifactStatus: ArtifactAvailable, Current: true}, BankReviewState: "PENDING"}}
	if !CollectionFieldFulfilled(field, now) {
		t.Fatal("bank review pending must not require duplicate vendor upload")
	}
	if field.CollectionResolution.BankReviewState != "PENDING" {
		t.Fatal("collection accepted evidence")
	}
	for _, change := range []func(*CollectionResolution){func(r *CollectionResolution) { r.Source.Current = false }, func(r *CollectionResolution) { r.Source.ExpiresOn = "2026-09-07" }, func(r *CollectionResolution) { r.Source.ArtifactStatus = ArtifactQuarantined }, func(r *CollectionResolution) { r.BankReviewState = "REJECTED" }} {
		copy := *field.CollectionResolution
		change(&copy)
		probe := field
		probe.CollectionResolution = &copy
		if CollectionFieldFulfilled(probe, now) {
			t.Fatalf("unusable source fulfilled request: %#v", copy)
		}
	}
}
