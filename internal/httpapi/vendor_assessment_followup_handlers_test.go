package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type assessmentFollowupHTTPFixture struct {
	handler    http.Handler
	base       assessmentHTTPFixture
	assessment thirdparty.Assessment
	matters    *continuity.Service
}

func newAssessmentFollowupHTTPFixture(t *testing.T) assessmentFollowupHTTPFixture {
	t.Helper()
	review := newReviewHTTPFixture(t)
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-reviewer", Kind: "PERSON", IssuedAt: time.Now().UTC().Add(-time.Minute), ExpiresAt: time.Now().UTC().Add(time.Hour)}
	under, err := review.base.service.StartAssessmentReview(identity.WithActor(context.Background(), actor), thirdparty.Actor{}, review.assessment.ID, review.assessment.Version)
	if err != nil {
		t.Fatal(err)
	}
	requests, err := thirdparty.NewAssessmentRequestService(review.base.service, review.base.repository, review.base.evidence, review.base.forms, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	requests.ConfigureDistributionDispatcher(evidence.NewWorkflowDistributionDispatcher(review.base.distributions, review.base.access))
	matters := continuity.NewService(continuity.NewMemoryRepository())
	deficiencies := thirdparty.NewAssessmentDeficiencyService(review.base.service, review.base.repository, matters)
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "verified-reviewer", "entity-a"), Evidence: review.base.evidence, FormDistributions: review.base.distributions, FormDistributionAccess: review.base.access, ThirdParty: review.base.serviceRepositoryService(), ThirdPartyAssessments: review.base.service, ThirdPartyAssessmentRequests: requests, ThirdPartyAssessmentDeficiencies: deficiencies, Continuity: matters})
	return assessmentFollowupHTTPFixture{handler: handler, base: review.base, assessment: under, matters: matters}
}

func TestVendorAssessmentClarificationRouteUsesStrictProtectedReviewerCommand(t *testing.T) {
	fixture := newAssessmentFollowupHTTPFixture(t)
	deadline := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	body := `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"request_fields":["contact_email"],"message":"Provide the current security contact.","audience":"security@vendor.example","deadline":"` + deadline + `","invitation_ttl_minutes":60,"actor_id":"forged-reviewer"}`
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/clarifications", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("clarification expected 200, got %d: %s", response.Code, response.Body.String())
	}
	raw := append([]byte(nil), response.Body.Bytes()...)
	var outcome thirdparty.AssessmentClarificationOutcome
	if err := json.Unmarshal(raw, &outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.Status != thirdparty.AssessmentCollecting || outcome.State != thirdparty.SendRequestLinkCreatedEmailNotSent || outcome.CaptureURL == "" {
		t.Fatalf("clarification outcome = %#v", outcome)
	}
	if strings.Contains(string(raw), "security@vendor.example") || (outcome.Invitation != nil && outcome.Invitation.Token != "") {
		t.Fatal("clarification response exposed protected recipient or token field")
	}
}

func TestVendorAssessmentDeficiencyRouteCreatesCanonicalMatterAndRejectsInvalidBodies(t *testing.T) {
	fixture := newAssessmentFollowupHTTPFixture(t)
	body := `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"trigger_key":"security-test-report","title":"Provide current security test evidence","summary":"The submitted report is no longer current for this review.","actor_id":"forged-reviewer"}`
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/deficiencies", strings.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("deficiency expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var outcome thirdparty.AssessmentDeficiencyOutcome
	if err := json.NewDecoder(response.Body).Decode(&outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.Status != thirdparty.AssessmentUnderReview || outcome.Matter.Matter.Type != continuity.MatterVendorDeficiency {
		t.Fatalf("deficiency outcome = %#v", outcome)
	}
	for _, test := range []struct {
		name, body string
		want       int
	}{
		{name: "unknown field", body: `{"expected_version":` + jsonInt(outcome.Assessment.Version) + `,"trigger_key":"retention-gap","title":"Record retention gap","summary":"Retention evidence is incomplete.","finding_id":"unsafe"}`, want: http.StatusBadRequest},
		{name: "stale", body: `{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"trigger_key":"retention-gap","title":"Record retention gap","summary":"Retention evidence is incomplete."}`, want: http.StatusConflict},
		{name: "scope", body: `{"expected_version":` + jsonInt(outcome.Assessment.Version) + `,"trigger_key":"retention-gap","title":"Record retention gap","summary":"Retention evidence is incomplete.","legal_entity_id":"other-entity"}`, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			fixture.handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/deficiencies", strings.NewReader(test.body)))
			if recorder.Code != test.want {
				t.Fatalf("want %d got %d: %s", test.want, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestVendorAssessmentFollowupRoutesRequireReviewerMaterialAuthority(t *testing.T) {
	wanted := map[string]string{"/api/v1/vendor-assessments/{id}/clarifications": thirdparty.AssessmentClarificationCommand, "/api/v1/vendor-assessments/{id}/deficiencies": thirdparty.AssessmentDeficiencyCommand}
	for _, route := range (&API{}).routes() {
		command, ok := wanted[route.Path]
		if !ok {
			continue
		}
		if route.Method != http.MethodPost || route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != command || route.Command.Policy.ObjectType != "THIRD_PARTY_ASSESSMENT" || route.Command.Policy.Responsibility != authority.ResponsibilityReviewer {
			t.Fatalf("follow-up route = %#v", route)
		}
		delete(wanted, route.Path)
	}
	if len(wanted) != 0 {
		t.Fatalf("missing follow-up routes: %#v", wanted)
	}
}
