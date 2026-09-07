package thirdparty

import (
	"context"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestDocumentContextUsesExactLegacyAssessmentLinkAndExistingReadAuthority(t *testing.T) {
	service, actor, assessment, reader := assessmentReviewFixture(t)
	adapter := DocumentContextReader{Assessments: service}
	q := evidence.DocumentQuery{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}
	request := reader.requests["request-1"]
	request.LegalEntityID = actor.LegalEntityID
	submission := reader.submissions["submission-1"]
	contextValue, err := adapter.ResolveDocumentContext(context.Background(), q, request, submission)
	if err != nil || contextValue.AssessmentID != assessment.ID || !contextValue.Current {
		t.Fatalf("legacy context %+v %v", contextValue, err)
	}
	q.PrincipalID = "intruder"
	if _, err := adapter.ResolveDocumentContext(context.Background(), q, request, submission); err == nil {
		t.Fatal("intruder read legacy evidence")
	}
	q.PrincipalID = actor.PrincipalID
	request.Origin.Version++
	if _, err := adapter.ResolveDocumentContext(context.Background(), q, request, submission); err == nil {
		t.Fatal("mismatched capture link read")
	}
}

func TestWorkDocumentContextRequiresExactFormRevision(t *testing.T) {
	f := newVendorWorkFixture(t)
	work := VendorWorkRequest{ID: "work", TenantID: f.actor.TenantID, LegalEntityID: f.actor.LegalEntityID, RelationshipID: "relationship-1", OwnerPrincipalID: f.actor.PrincipalID, FormTemplateID: "form-1", FormTemplateVersion: 3}
	f.repository.work[work.ID] = work
	f.repository.captures[work.ID] = []VendorWorkCaptureLink{{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, WorkRequestID: work.ID, RequestID: "request", OriginVersion: 1}}
	q := evidence.DocumentQuery{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, PrincipalID: f.actor.PrincipalID}
	request := evidence.Request{ID: "request", TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: work.RelationshipID, FormTemplateID: work.FormTemplateID, FormTemplateVersion: work.FormTemplateVersion, Origin: evidence.RequestOrigin{Type: VendorWorkOrigin, ID: work.ID, Version: 1}}
	submission := evidence.Submission{ID: "submission", TenantID: work.TenantID, RequestID: request.ID}
	reader := DocumentContextReader{Work: f.service}
	if _, err := reader.ResolveDocumentContext(context.Background(), q, request, submission); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*evidence.Request){func(r *evidence.Request) { r.FormTemplateID = "other" }, func(r *evidence.Request) { r.FormTemplateVersion++ }} {
		probe := request
		change(&probe)
		if _, err := reader.ResolveDocumentContext(context.Background(), q, probe, submission); err == nil {
			t.Fatal("accepted a different work form revision")
		}
	}
}
