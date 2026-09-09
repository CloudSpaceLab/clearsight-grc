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

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type deniedCollectionGuard struct{}

func (deniedCollectionGuard) Authorize(context.Context, commandauth.Request) (commandauth.Decision, error) {
	return commandauth.Decision{}, thirdparty.ErrAssessmentAuthorityUnavailable
}

func TestPrepareCollectionHTTPUsesVerifiedActorAndReturnsPreparedRequest(t *testing.T) {
	f := newAssessmentHTTPFixture(t, true)
	body := `{"expected_version":` + jsonInt(f.assessment.Version) + `,"audience":"security@vendor.example","deadline":"` + time.Now().UTC().Add(48*time.Hour).Format(time.RFC3339) + `"}`
	path := "/api/v1/vendor-assessments/" + f.assessment.ID + "/prepare-request"
	for _, extra := range []string{`,"tenant_id":"other-bank"`, `,"legal_entity_id":"other-entity"`} {
		response := httptest.NewRecorder()
		f.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(strings.TrimSuffix(body, "}")+extra+"}")))
		if response.Code < 400 {
			t.Fatalf("forged command accepted: %d %s", response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(strings.TrimSuffix(body, "}")+`,"actor_id":"forged-reviewer"}`)))
	if response.Code != 200 {
		t.Fatalf("prepare: %d %s", response.Code, response.Body.String())
	}
	var prepared thirdparty.SendRequestOutcome
	if err := json.Unmarshal(response.Body.Bytes(), &prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.State != "PREPARED" || prepared.Invitation != nil || prepared.Assessment.CurrentRequestID == "" {
		t.Fatalf("invalid preparation: %+v", prepared)
	}
	request, err := f.evidence.GetRequest(context.Background(), "bank", prepared.Request.ID)
	if err != nil || request.CreatedBy != "verified-owner" {
		t.Fatalf("actor not verified: %+v %v", request, err)
	}
	reviews := thirdparty.NewAssessmentReviewService(f.service, f.repository, f.evidence, nil)
	reviews.ConfigureCollectionSources(f.distributions, f.forms)
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory", Identity: identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"), ThirdPartyAssessmentReviews: reviews})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendor-assessments/"+f.assessment.ID+"/collection", nil))
	if response.Code != 200 {
		t.Fatalf("checklist: %d %s", response.Code, response.Body.String())
	}
	var checklist thirdparty.AssessmentCollection
	if err = json.Unmarshal(response.Body.Bytes(), &checklist); err != nil {
		t.Fatal(err)
	}
	if !checklist.Prepared || checklist.RequestID != request.ID || strings.Contains(response.Body.String(), "security@vendor.example") {
		t.Fatalf("checklist exposed recipient or lost preparation: %s", response.Body.String())
	}
	response = httptest.NewRecorder()
	f.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+f.assessment.ID+"/send-request", strings.NewReader(`{"expected_version":`+jsonInt(prepared.Assessment.Version)+`,"audience":"","invitation_ttl_minutes":60}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("resume prepared send: %d %s", response.Code, response.Body.String())
	}
	var sent thirdparty.SendRequestOutcome
	if err = json.Unmarshal(response.Body.Bytes(), &sent); err != nil || sent.Request.ID != request.ID {
		t.Fatalf("resume replaced prepared request: %+v %v", sent, err)
	}
}

func TestCollectionCommandsHTTPFailClosedWithoutIdentityOrAuthority(t *testing.T) {
	f := newAssessmentHTTPFixture(t, true)
	denied := thirdparty.NewAssessmentService(f.repository, deniedCollectionGuard{})
	requests, err := thirdparty.NewAssessmentRequestService(denied, f.repository, f.evidence, f.forms, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	requests.ConfigureDistributionDispatcher(evidence.NewWorkflowDistributionDispatcher(f.distributions, f.access))
	reviews := thirdparty.NewAssessmentReviewService(denied, f.repository, f.evidence, nil)
	reviews.ConfigureCollectionSources(f.distributions, f.forms)
	for _, authenticated := range []bool{false, true} {
		deps := Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "production", ThirdPartyAssessmentRequests: requests, ThirdPartyAssessmentReviews: reviews}
		if authenticated {
			deps.Mode = "test-memory"
			deps.Identity = identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a")
		}
		handler := New(deps)
		for _, suffix := range []string{"/prepare-request", "/collection/assurance_report/reconcile"} {
			body := `{"expected_version":` + jsonInt(f.assessment.Version) + `,"audience":"security@vendor.example","deadline":"` + time.Now().UTC().Add(48*time.Hour).Format(time.RFC3339) + `"}`
			if strings.Contains(suffix, "reconcile") {
				body = `{"expected_version":1,"request_id":"request-1","expected_request_version":1,"source_submission_id":"submission-1","source_field_id":"report","source_artifact_id":"artifact-1","rationale":"The report covers this service."}`
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+f.assessment.ID+suffix, strings.NewReader(body)))
			if response.Code < 400 {
				t.Fatalf("unauthorized command succeeded: %s %d", suffix, response.Code)
			}
		}
	}
	current, err := f.service.GetAssessment(context.Background(), thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner"}, f.assessment.ID)
	if err != nil || current.Version != f.assessment.Version || current.CurrentRequestID != "" {
		t.Fatalf("denied command changed assessment: %+v %v", current, err)
	}
}
