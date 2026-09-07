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
