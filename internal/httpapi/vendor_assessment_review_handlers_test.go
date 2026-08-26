package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type reviewHTTPFixture struct {
	handler    http.Handler
	base       assessmentHTTPFixture
	assessment thirdparty.Assessment
}

func newReviewHTTPFixture(t *testing.T, includeDocument ...bool) reviewHTTPFixture {
	t.Helper()
	base := newAssessmentHTTPFixture(t, true)
	sent := sendHTTPVendorAssessmentRequest(t, base)
	parsed, err := url.Parse(sent.CaptureURL)
	if err != nil {
		t.Fatal(err)
	}
	invitation := parsed.Query().Get("capture_invite")
	if invitation == "" {
		t.Fatalf("send outcome did not contain an immediate fallback invitation: %#v", sent)
	}
	redeemed, err := base.evidence.RedeemInvitation(context.Background(), invitation, "security@vendor.example")
	if err != nil {
		t.Fatal(err)
	}
	answers := map[string]formcontract.AnswerValue{
		"contact_email": formcontract.TextAnswer("vendor-contact-response@example.test"),
	}
	if len(includeDocument) > 0 && includeDocument[0] {
		artifact, storeErr := base.evidence.StoreArtifact(context.Background(), evidence.ArtifactInput{
			TenantID: "bank", RequestID: sent.Request.ID, FileName: "assurance-report.pdf", MediaType: "application/pdf", SessionToken: redeemed.SessionToken,
		}, strings.NewReader("review evidence"))
		if storeErr != nil {
			t.Fatal(storeErr)
		}
		answers["assurance_report"] = formcontract.AnswerValue{Document: &formcontract.DocumentAnswer{
			ArtifactID: artifact.ID, DocumentType: "SOC_2_TYPE_II", Reference: "SOC2-2026", IssuedBy: "Independent auditor", IssuedOn: "2026-06-01", ExpiresOn: "2027-05-31",
		}}
	}
	receipt, err := base.evidence.SubmitSession(context.Background(), redeemed.SessionToken, answers, sent.Request.Version)
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := base.service.RecordAssessmentSubmitted(context.Background(), thirdparty.AssessmentSubmittedInput{
		Scope: thirdparty.Scope{TenantID: "bank", LegalEntityID: "entity-a"}, AssessmentID: sent.Assessment.ID,
		ExpectedVersion: sent.Assessment.Version, CausationID: "submission-event-1", EventID: "submission-event-1",
		RequestID: sent.Request.ID, SubmissionID: receipt.SubmissionID,
	})
	if err != nil {
		t.Fatal(err)
	}
	reviews := thirdparty.NewAssessmentReviewService(base.service, base.repository, base.evidence, nil)
	reviews.ConfigureAuthority(authority.NewResolver("review-test", []authority.Rule{{
		ID: "assessment-reviewer", TenantID: "bank", LegalEntityID: "entity-a", ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: submitted.ID,
		Responsibility: authority.ResponsibilityReviewer, DecisionType: thirdparty.AssessmentReviewCommand, MinMateriality: 3,
		Principal: authority.Principal{ID: "verified-reviewer", Kind: "PERSON"}, Priority: 1,
	}}))
	base.service.ConfigureCompletionReadiness(reviews)
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory",
		Identity:   identity.NewDevelopmentAuthenticator("bank", "verified-reviewer", "entity-a", "REVIEWER"),
		ThirdParty: base.serviceRepositoryService(), ThirdPartyAssessments: base.service, ThirdPartyAssessmentReviews: reviews,
	})
	return reviewHTTPFixture{handler: handler, base: base, assessment: submitted}
}

func TestReviewVendorAssessmentDocumentReturnsRefreshedViewAndEnforcesVersion(t *testing.T) {
	fixture := newReviewHTTPFixture(t, true)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/review/start", strings.NewReader(`{"expected_version":`+jsonInt(fixture.assessment.Version)+`}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("start review expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var underReview thirdparty.Assessment
	if err := json.NewDecoder(response.Body).Decode(&underReview); err != nil {
		t.Fatal(err)
	}
	viewResponse := httptest.NewRecorder()
	body := `{"expected_version":` + jsonInt(underReview.Version) + `,"decision":"REJECT","document_type":"SOC_2_TYPE_II","evidence_class":"VENDOR_SUPPLIED","valid_until":"2027-05-31","actor_id":"forged-reviewer"}`
	fixture.handler.ServeHTTP(viewResponse, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/documents/"+reviewArtifactID(t, fixture)+"/validate", strings.NewReader(body)))
	if viewResponse.Code != http.StatusOK {
		t.Fatalf("document review expected 200, got %d: %s", viewResponse.Code, viewResponse.Body.String())
	}
	var view thirdparty.AssessmentReviewView
	if err := json.NewDecoder(viewResponse.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Assessment.Version != underReview.Version+1 || len(view.Documents) != 1 || view.Documents[0].Status != "REJECTED" || view.Documents[0].Reference != "SOC2-2026" {
		t.Fatalf("unexpected refreshed document review %#v", view)
	}

	stale := httptest.NewRecorder()
	fixture.handler.ServeHTTP(stale, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/documents/"+view.Documents[0].ArtifactID+"/validate", strings.NewReader(body)))
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale document review expected 409, got %d: %s", stale.Code, stale.Body.String())
	}
}

func reviewArtifactID(t *testing.T, fixture reviewHTTPFixture) string {
	t.Helper()
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendor-assessments/"+fixture.assessment.ID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("load document review expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var view thirdparty.AssessmentReviewView
	if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if len(view.Documents) != 1 {
		t.Fatalf("expected one submitted document, got %#v", view.Documents)
	}
	return view.Documents[0].ArtifactID
}

func TestGetVendorAssessmentReviewUsesVerifiedScopeAndOmitsProtectedCaptureFields(t *testing.T) {
	fixture := newReviewHTTPFixture(t)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendor-assessments/"+fixture.assessment.ID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	raw := append([]byte(nil), response.Body.Bytes()...)
	var view thirdparty.AssessmentReviewView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatal(err)
	}
	if view.Assessment.ID != fixture.assessment.ID || view.Response == nil || view.Response.RequestID != fixture.assessment.CurrentRequestID || len(view.Requests) != 1 {
		t.Fatalf("unexpected exact review view %#v", view)
	}
	if !strings.Contains(string(raw), `"matters":[]`) || strings.Contains(string(raw), `"findings"`) {
		t.Fatalf("review response did not use the rich reviewer matter contract: %s", raw)
	}
	for _, protected := range []string{"security@vendor.example", "capture_invite", "invitation_id", "session_id", "storage_key", "submitted_by"} {
		if strings.Contains(string(raw), protected) {
			t.Fatalf("review response exposed protected capture field %q: %s", protected, raw)
		}
	}

	wrongScope := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "verified-reviewer", "entity-b", "REVIEWER"),
		ThirdPartyAssessments:       fixture.base.service,
		ThirdPartyAssessmentReviews: thirdparty.NewAssessmentReviewService(fixture.base.service, fixture.base.repository, fixture.base.evidence, nil),
	})
	response = httptest.NewRecorder()
	wrongScope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendor-assessments/"+fixture.assessment.ID, nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-entity review read expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestGetVendorAssessmentReviewRejectsUnrelatedSameScopePrincipal(t *testing.T) {
	fixture := newReviewHTTPFixture(t)
	reviews := thirdparty.NewAssessmentReviewService(fixture.base.service, fixture.base.repository, fixture.base.evidence, nil)
	reviews.ConfigureAuthority(authority.NewResolver("review-test", []authority.Rule{{
		ID: "assessment-reviewer", TenantID: "bank", LegalEntityID: "entity-a", ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: fixture.assessment.ID,
		Responsibility: authority.ResponsibilityReviewer, DecisionType: thirdparty.AssessmentReviewCommand, MinMateriality: 3,
		Principal: authority.Principal{ID: "verified-reviewer", Kind: "PERSON"}, Priority: 1,
	}}))
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "unrelated-principal", "entity-a", "REVIEWER"),
		ThirdPartyAssessments:       fixture.base.service,
		ThirdPartyAssessmentReviews: reviews,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendor-assessments/"+fixture.assessment.ID, nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unrelated same-scope principal expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestStartAndCompleteVendorAssessmentReviewUseVerifiedReviewerAndDoNotActivateRelationship(t *testing.T) {
	fixture := newReviewHTTPFixture(t)
	before, err := fixture.base.serviceRepositoryService().GetRelationship(context.Background(), thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-reviewer"}, fixture.base.vendor.Relationship.ID)
	if err != nil {
		t.Fatal(err)
	}
	startBody := []byte(`{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"tenant_id":"bank","legal_entity_id":"entity-a","actor_id":"forged-reviewer"}`)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/review/start", bytes.NewReader(startBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("start review expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var underReview thirdparty.Assessment
	if err := json.NewDecoder(response.Body).Decode(&underReview); err != nil {
		t.Fatal(err)
	}
	if underReview.Status != thirdparty.AssessmentUnderReview || underReview.ReviewerPrincipalID != "verified-reviewer" {
		t.Fatalf("review did not use verified reviewer: %#v", underReview)
	}
	response = httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/review/start", strings.NewReader(`{"expected_version":`+jsonInt(underReview.Version)+`}`)))
	if response.Code != http.StatusConflict {
		t.Fatalf("second start outside SUBMITTED expected 409, got %d: %s", response.Code, response.Body.String())
	}

	nextReview := time.Now().UTC().Add(90 * 24 * time.Hour).Format(time.RFC3339)
	completeBody := []byte(`{"expected_version":` + jsonInt(underReview.Version) + `,"conclusion":"SATISFACTORY_WITH_CONDITIONS","rationale":"Proceed only after the documented access-control actions are complete.","uncertainty":"The latest resilience exercise remains under review.","next_review_recommended_at":"` + nextReview + `","actor_id":"forged-reviewer"}`)
	response = httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/complete", strings.NewReader(`{"expected_version":`+jsonInt(underReview.Version-1)+`,"conclusion":"SATISFACTORY","rationale":"Current evidence supports completion."}`)))
	if response.Code != http.StatusConflict {
		t.Fatalf("stale completion expected 409, got %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/complete", bytes.NewReader(completeBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("complete review expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var completed thirdparty.Assessment
	if err := json.NewDecoder(response.Body).Decode(&completed); err != nil {
		t.Fatal(err)
	}
	if completed.Status != thirdparty.AssessmentCompleted || completed.Conclusion != thirdparty.AssessmentSatisfactoryWithConditions || completed.ReviewerPrincipalID != "verified-reviewer" {
		t.Fatalf("unexpected completed assessment %#v", completed)
	}
	after, err := fixture.base.serviceRepositoryService().GetRelationship(context.Background(), thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-reviewer"}, fixture.base.vendor.Relationship.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Relationship.Status != before.Relationship.Status || after.Relationship.Version != before.Relationship.Version || after.Relationship.Status == thirdparty.RelationshipActive {
		t.Fatalf("assessment completion changed relationship activation state: before=%#v after=%#v", before.Relationship, after.Relationship)
	}
}

func TestVendorAssessmentReviewRejectsStaleWrongStateScopeAndInvalidBodies(t *testing.T) {
	fixture := newReviewHTTPFixture(t)
	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{name: "stale start", path: "/review/start", body: `{"expected_version":1}`, want: http.StatusConflict},
		{name: "forged tenant", path: "/review/start", body: `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"tenant_id":"other-bank"}`, want: http.StatusForbidden},
		{name: "forged legal entity", path: "/review/start", body: `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"legal_entity_id":"entity-b"}`, want: http.StatusForbidden},
		{name: "complete before review", path: "/complete", body: `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"conclusion":"SATISFACTORY","rationale":"Current evidence supports completion."}`, want: http.StatusConflict},
		{name: "unknown field", path: "/review/start", body: `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"reviewer":"forged"}`, want: http.StatusBadRequest},
		{name: "invalid conclusion", path: "/complete", body: `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"conclusion":"APPROVED","rationale":"This must not activate the relationship."}`, want: http.StatusUnprocessableEntity},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+test.path, strings.NewReader(test.body)))
			if response.Code != test.want {
				t.Fatalf("expected %d, got %d: %s", test.want, response.Code, response.Body.String())
			}
		})
	}
}

func TestVendorAssessmentReviewMutationsRequireReviewerAuthority(t *testing.T) {
	fixture := newReviewHTTPFixture(t)
	denied := thirdparty.NewAssessmentService(fixture.base.repository, &deniedAssessmentGuard{})
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "verified-reviewer", "entity-a"),
		ThirdPartyAssessments:       denied,
		ThirdPartyAssessmentReviews: thirdparty.NewAssessmentReviewService(denied, fixture.base.repository, fixture.base.evidence, nil),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/review/start", strings.NewReader(`{"expected_version":`+jsonInt(fixture.assessment.Version)+`}`)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("review without reviewer authority expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestVendorAssessmentReviewRoutesDeclareScopedReadAndReviewerCommands(t *testing.T) {
	routes := (&API{}).routes()
	wanted := map[string]bool{
		http.MethodGet + " /api/v1/vendor-assessments/{id}":                                   false,
		http.MethodPost + " /api/v1/vendor-assessments/{id}/review/start":                     false,
		http.MethodPost + " /api/v1/vendor-assessments/{id}/documents/{artifact_id}/validate": false,
		http.MethodPost + " /api/v1/vendor-assessments/{id}/complete":                         false,
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := wanted[key]; !ok {
			continue
		}
		wanted[key] = true
		if route.Method == http.MethodGet {
			if route.Class != routeAuthenticatedRead || route.Command != nil {
				t.Fatalf("review read route is not a scoped authenticated read: %#v", route)
			}
			continue
		}
		if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Policy.Responsibility != "REVIEWER" || route.Command.Policy.ObjectType != "THIRD_PARTY_ASSESSMENT" {
			t.Fatalf("review mutation does not require assessment reviewer authority: %#v", route)
		}
	}
	for route, found := range wanted {
		if !found {
			t.Fatalf("review route is missing: %s", route)
		}
	}
}
