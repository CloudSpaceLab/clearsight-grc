package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type httpAssessmentGuard struct{}

func (httpAssessmentGuard) Authorize(ctx context.Context, _ commandauth.Request) (commandauth.Decision, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return commandauth.Decision{}, err
	}
	return commandauth.Decision{Allowed: true, Enforced: true, Actor: actor}, nil
}

type assessmentHTTPFixture struct {
	handler    http.Handler
	repository *thirdparty.MemoryAssessmentRepository
	service    *thirdparty.AssessmentService
	evidence   *evidence.Service
	forms      *monitoring.MemoryRepository
	vendor     thirdparty.Aggregate
	assessment thirdparty.Assessment
}

func newAssessmentHTTPFixture(t *testing.T, ready bool, inlineSetup ...bool) assessmentHTTPFixture {
	t.Helper()
	repo := thirdparty.NewMemoryAssessmentRepository()
	vendorService := thirdparty.NewService(repo)
	vendor, err := vendorService.CreateRelationship(context.Background(), thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner"}, thirdparty.CreateRelationshipInput{
		LegalName: "Acme Processing Limited", TradingName: "Acme Processing", RegistrationRef: "RC-10001", Jurisdiction: "Nigeria",
		ServiceName: "Card transaction processing", Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor,
	})
	if err != nil {
		t.Fatal(err)
	}
	assessmentService := thirdparty.NewAssessmentService(repo, httpAssessmentGuard{})
	formRepo := monitoring.NewMemoryRepository()
	form, err := formRepo.CreateFormRevision(context.Background(), monitoring.FormTemplate{
		ID: "form-1", TenantID: "bank", Name: "Vendor due diligence", Purpose: "Provide the information required for this vendor review.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     []formcontract.Section{{ID: "organisation", Title: "Organisation"}},
		Fields: []monitoring.TemplateField{
			{ID: "contact_email", SectionID: "organisation", Label: "Contact email", Type: formcontract.TypeEmail, Required: true},
			{ID: "assurance_report", SectionID: "organisation", Label: "Assurance report", Type: formcontract.TypeVendorDocument, AcceptedFormats: []string{"application/pdf"}},
		},
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 3, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	})
	if err != nil {
		t.Fatal(err)
	}
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	assessmentService.ConfigureCancellationRevoker(evidenceService)
	requestService, err := thirdparty.NewAssessmentRequestService(assessmentService, repo, evidenceService, formRepo, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	var setup *thirdparty.AssessmentProvisioner
	if len(inlineSetup) > 0 && inlineSetup[0] {
		setup = thirdparty.NewAssessmentProvisioner(repo, continuity.NewService(continuity.NewMemoryRepository()), "memory-api-test")
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory",
		Identity:   identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"),
		ThirdParty: vendorService, ThirdPartyAssessments: assessmentService, ThirdPartyAssessmentRequests: requestService, ThirdPartyAssessmentSetup: setup,
	})
	fixture := assessmentHTTPFixture{handler: handler, repository: repo, service: assessmentService, evidence: evidenceService, forms: formRepo, vendor: vendor}
	if ready {
		ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner", Kind: "PERSON", IssuedAt: time.Now().UTC().Add(-time.Minute), ExpiresAt: time.Now().UTC().Add(time.Hour)})
		assessment, startErr := assessmentService.StartAssessment(ctx, thirdparty.Actor{}, vendor.Relationship.ID, thirdparty.StartAssessmentInput{
			RelationshipVersion: vendor.Relationship.Version, FormTemplateID: form.ID, FormTemplateVersion: form.Version, ReviewDueAt: time.Now().UTC().Add(14 * 24 * time.Hour),
		})
		if startErr != nil {
			t.Fatal(startErr)
		}
		assessment, startErr = assessmentService.RecordAssessmentSetupCompleted(context.Background(), thirdparty.AssessmentSetupCompletedInput{
			Scope: thirdparty.Scope{TenantID: "bank", LegalEntityID: "entity-a"}, AssessmentID: assessment.ID,
			ExpectedVersion: assessment.Version, CausationID: "setup-event", SetupJobID: "setup-job", ReviewMatterID: "review-matter",
		})
		if startErr != nil {
			t.Fatal(startErr)
		}
		fixture.assessment = assessment
	}
	return fixture
}

func TestMemoryAPIProvisionerCompletesAssessmentSetupWithoutASeparateRepository(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, false, true)
	body := []byte(`{"relationship_version":1,"form_template_id":"form-1","form_template_version":3,"review_due_at":"2030-09-09T10:00:00Z"}`)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+fixture.vendor.Relationship.ID+"/assessments", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
	}
	var assessment thirdparty.Assessment
	if err := json.NewDecoder(response.Body).Decode(&assessment); err != nil {
		t.Fatal(err)
	}
	if assessment.Status != thirdparty.AssessmentReadyToSend || assessment.ReviewMatterID == "" {
		t.Fatalf("inline memory setup did not return the operable assessment: %#v", assessment)
	}
	status, err := fixture.service.GetAssessmentSetupStatus(context.Background(), thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner"}, assessment.ID)
	if err != nil || status.State != thirdparty.AssessmentJobCompleted {
		t.Fatalf("inline memory setup job = %#v, %v", status, err)
	}
}

func TestStartAndGetCurrentVendorAssessmentUseVerifiedRouteScope(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, false)
	body := []byte(`{"relationship_version":1,"form_template_id":"form-1","form_template_version":3,"review_due_at":"2030-09-09T10:00:00Z","tenant_id":"forged","legal_entity_id":"forged","actor_id":"forged"}`)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+fixture.vendor.Relationship.ID+"/assessments", bytes.NewReader(body)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected forged scope rejection, got %d: %s", response.Code, response.Body.String())
	}

	body = []byte(`{"relationship_version":1,"form_template_id":"form-1","form_template_version":3,"review_due_at":"2030-09-09T10:00:00Z","actor_id":"forged-owner"}`)
	response = httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+fixture.vendor.Relationship.ID+"/assessments", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
	}
	var started thirdparty.Assessment
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendors/"+fixture.vendor.Relationship.ID+"/assessments/current", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var current vendorAssessmentCurrentResponse
	if err := json.NewDecoder(response.Body).Decode(&current); err != nil {
		t.Fatal(err)
	}
	if current.Assessment.ID != started.ID || current.Assessment.TenantID != "bank" || current.Assessment.LegalEntityID != "entity-a" || current.Assessment.StartedByPrincipalID != "verified-owner" {
		t.Fatalf("assessment did not use verified identity: %#v", current)
	}
	if current.Setup == nil || current.Setup.State != thirdparty.AssessmentJobReady {
		t.Fatalf("setup recovery state was not included: %#v", current.Setup)
	}
}

func TestCancelVendorAssessmentUsesVerifiedOwnerAndCurrentVersion(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, true)
	body := []byte(`{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"reason":"The proposed service is no longer being procured.","actor_id":"forged-owner"}`)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/cancel", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("cancel expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var cancelled thirdparty.Assessment
	if err := json.NewDecoder(response.Body).Decode(&cancelled); err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != thirdparty.AssessmentCancelled || cancelled.CancellationReason == "" || cancelled.Version != fixture.assessment.Version+1 {
		t.Fatalf("unexpected cancelled assessment %#v", cancelled)
	}

	stale := httptest.NewRecorder()
	fixture.handler.ServeHTTP(stale, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/cancel", bytes.NewReader(body)))
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale cancel expected 409, got %d: %s", stale.Code, stale.Body.String())
	}
}

func TestCancelVendorAssessmentRevokesInvitationAndRedeemedSession(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, true)
	issued := sendHTTPVendorAssessmentRequest(t, fixture)
	if issued.Invitation == nil {
		t.Fatal("assessment request did not issue an invitation")
	}
	captureURL, err := url.Parse(issued.CaptureURL)
	if err != nil {
		t.Fatal(err)
	}
	token := captureURL.Query().Get("capture_invite")
	session, err := fixture.evidence.RedeemInvitation(context.Background(), token, "security@vendor.example")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"expected_version":` + jsonInt(issued.Assessment.Version) + `,"reason":"The proposed service is no longer being procured."}`)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/cancel", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("cancel expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if _, _, err := fixture.evidence.SessionRequest(context.Background(), session.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
		t.Fatalf("cancelled assessment session remained usable: %v", err)
	}
	if _, err := fixture.evidence.RedeemInvitation(context.Background(), token, "security@vendor.example"); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("cancelled assessment invitation remained usable: %v", err)
	}
}

func TestRetryVendorAssessmentSetupRequeuesSameTerminalJobAndIsReplaySafe(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, false)
	assessment, terminal := terminalHTTPAssessmentSetup(t, fixture)
	body := `{"expected_version":` + jsonInt(assessment.Version) + `,"actor_id":"forged-owner"}`

	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+assessment.ID+"/setup/retry", strings.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("retry setup expected 202, got %d: %s", response.Code, response.Body.String())
	}
	var outcome thirdparty.AssessmentSetupRetryOutcome
	if err := json.NewDecoder(response.Body).Decode(&outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.ID != assessment.ID || outcome.Assessment.Version != assessment.Version+1 || outcome.Assessment.Status != thirdparty.AssessmentSetupPending || outcome.Setup.State != thirdparty.AssessmentJobReady || outcome.Setup.Attempts != 0 {
		t.Fatalf("retry setup outcome = %#v", outcome)
	}
	jobs, err := fixture.repository.ListAssessmentSetupJobs(context.Background(), thirdparty.Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID}, assessment.ID)
	if err != nil || len(jobs) != 1 || jobs[0].ID != terminal.ID {
		t.Fatalf("retry setup jobs = (%#v, %v)", jobs, err)
	}

	replay := httptest.NewRecorder()
	fixture.handler.ServeHTTP(replay, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+assessment.ID+"/setup/retry", strings.NewReader(body)))
	if replay.Code != http.StatusAccepted {
		t.Fatalf("retry setup replay expected 202, got %d: %s", replay.Code, replay.Body.String())
	}
	var replayed thirdparty.AssessmentSetupRetryOutcome
	if err := json.NewDecoder(replay.Body).Decode(&replayed); err != nil {
		t.Fatal(err)
	}
	if replayed.Assessment.Version != outcome.Assessment.Version || replayed.Setup.State != outcome.Setup.State {
		t.Fatalf("retry setup replay = %#v", replayed)
	}
}

func TestRetryVendorAssessmentSetupRejectsStaleScopeUnknownFieldAndNonOwner(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, false)
	assessment, _ := terminalHTTPAssessmentSetup(t, fixture)
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "stale version", body: `{"expected_version":99}`, want: http.StatusConflict},
		{name: "forged tenant", body: `{"expected_version":` + jsonInt(assessment.Version) + `,"tenant_id":"other-bank"}`, want: http.StatusForbidden},
		{name: "forged legal entity", body: `{"expected_version":` + jsonInt(assessment.Version) + `,"legal_entity_id":"other-entity"}`, want: http.StatusForbidden},
		{name: "unknown field", body: `{"expected_version":` + jsonInt(assessment.Version) + `,"retry_job":true}`, want: http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+assessment.ID+"/setup/retry", strings.NewReader(testCase.body)))
			if response.Code != testCase.want {
				t.Fatalf("expected %d, got %d: %s", testCase.want, response.Code, response.Body.String())
			}
		})
	}

	deniedService := thirdparty.NewAssessmentService(fixture.repository, &deniedAssessmentGuard{})
	deniedHandler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"),
		ThirdParty: fixture.serviceRepositoryService(), ThirdPartyAssessments: deniedService,
	})
	response := httptest.NewRecorder()
	deniedHandler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+assessment.ID+"/setup/retry", strings.NewReader(`{"expected_version":`+jsonInt(assessment.Version)+`}`)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-owner setup retry expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRetryVendorAssessmentSetupRouteRequiresOwnerMaterialAuthority(t *testing.T) {
	routes := (&API{}).routes()
	for _, route := range routes {
		if route.Method == http.MethodPost && route.Path == "/api/v1/vendor-assessments/{id}/setup/retry" {
			if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != thirdparty.AssessmentSetupRetryCommand || route.Command.Policy.ObjectType != "THIRD_PARTY_ASSESSMENT" || route.Command.Policy.Responsibility != authority.ResponsibilityOwner {
				t.Fatalf("setup retry route = %#v", route)
			}
			return
		}
	}
	t.Fatal("setup retry route is missing")
}

func TestSendVendorAssessmentRequestBuildsLinkFromConfiguredBaseNotHost(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, true)
	deadline := time.Now().UTC().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	body := []byte(`{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"audience":"security@vendor.example","deadline":"` + deadline + `","invitation_ttl_minutes":60}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/send-request", bytes.NewReader(body))
	request.Host = "attacker.example"
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var outcome thirdparty.SendRequestOutcome
	if err := json.NewDecoder(response.Body).Decode(&outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.State != thirdparty.SendRequestLinkCreatedEmailNotSent || !strings.HasPrefix(outcome.CaptureURL, "https://capture.example.test/respond?capture_invite=") || strings.Contains(outcome.CaptureURL, "attacker.example") {
		t.Fatalf("unexpected send response: %#v", outcome)
	}
}

func TestSendVendorAssessmentErrorDoesNotEchoRecipientOrToken(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, true)
	body := []byte(`{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"audience":"secret-recipient@vendor.example","deadline":"2020-01-01T00:00:00Z","invitation_ttl_minutes":60,"token":"secret-token"}`)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/send-request", bytes.NewReader(body)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret-recipient") || strings.Contains(response.Body.String(), "secret-token") {
		t.Fatalf("protected value echoed in error: %s", response.Body.String())
	}
}

func TestReissueVendorAssessmentRequestRecoversAfterReloadUsingVerifiedActor(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, true)
	initial := sendHTTPVendorAssessmentRequest(t, fixture)
	body := []byte(`{"expected_version":` + jsonInt(initial.Assessment.Version) + `,"audience":"security@vendor.example","invitation_ttl_minutes":60,"actor_id":"forged-owner"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+initial.Assessment.ID+"/reissue-request", bytes.NewReader(body))
	request.Host = "attacker.example"
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var outcome thirdparty.SendRequestOutcome
	if err := json.NewDecoder(response.Body).Decode(&outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.Status != thirdparty.AssessmentCollecting || outcome.Assessment.Version != initial.Assessment.Version+2 || outcome.Request.ID != initial.Request.ID {
		t.Fatalf("reissue changed the collection lifecycle or request: %#v", outcome)
	}
	if outcome.State != thirdparty.SendRequestLinkCreatedEmailNotSent || !strings.HasPrefix(outcome.CaptureURL, "https://capture.example.test/respond?capture_invite=") || strings.Contains(outcome.CaptureURL, "attacker.example") {
		t.Fatalf("reissue did not return the immediate configured fallback link: %#v", outcome)
	}
	if outcome.Invitation == nil || outcome.Invitation.Token != "" {
		t.Fatalf("reissue response exposed the raw token outside the fallback URL: %#v", outcome.Invitation)
	}
}

func TestReissueVendorAssessmentRequestRejectsStaleScopeAndWrongState(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, true)
	initial := sendHTTPVendorAssessmentRequest(t, fixture)
	for _, test := range []struct {
		name       string
		assessment thirdparty.Assessment
		body       string
		want       int
	}{
		{name: "stale version", assessment: initial.Assessment, body: `{"expected_version":` + jsonInt(initial.Assessment.Version-1) + `,"audience":"security@vendor.example","invitation_ttl_minutes":60}`, want: http.StatusConflict},
		{name: "wrong tenant", assessment: initial.Assessment, body: `{"expected_version":` + jsonInt(initial.Assessment.Version) + `,"audience":"security@vendor.example","invitation_ttl_minutes":60,"tenant_id":"other-bank"}`, want: http.StatusForbidden},
		{name: "wrong legal entity", assessment: initial.Assessment, body: `{"expected_version":` + jsonInt(initial.Assessment.Version) + `,"audience":"security@vendor.example","invitation_ttl_minutes":60,"legal_entity_id":"other-entity"}`, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+test.assessment.ID+"/reissue-request", strings.NewReader(test.body)))
			if response.Code != test.want {
				t.Fatalf("expected %d, got %d: %s", test.want, response.Code, response.Body.String())
			}
		})
	}

	readyFixture := newAssessmentHTTPFixture(t, true)
	body := `{"expected_version":` + jsonInt(readyFixture.assessment.Version) + `,"audience":"security@vendor.example","invitation_ttl_minutes":60}`
	response := httptest.NewRecorder()
	readyFixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+readyFixture.assessment.ID+"/reissue-request", strings.NewReader(body)))
	if response.Code != http.StatusConflict {
		t.Fatalf("wrong-state reissue expected 409, got %d: %s", response.Code, response.Body.String())
	}
}

func TestReissueVendorAssessmentRequestRejectsNonOwner(t *testing.T) {
	fixture := newAssessmentHTTPFixture(t, true)
	initial := sendHTTPVendorAssessmentRequest(t, fixture)
	deniedService := thirdparty.NewAssessmentService(fixture.repository, &deniedAssessmentGuard{})
	requestService, err := thirdparty.NewAssessmentRequestService(deniedService, fixture.repository, fixture.evidence, fixture.forms, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory",
		Identity:   identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"),
		ThirdParty: fixture.serviceRepositoryService(), ThirdPartyAssessments: deniedService, ThirdPartyAssessmentRequests: requestService,
	})
	body := `{"expected_version":` + jsonInt(initial.Assessment.Version) + `,"audience":"security@vendor.example","invitation_ttl_minutes":60}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+initial.Assessment.ID+"/reissue-request", strings.NewReader(body)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-owner reissue expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

type deniedAssessmentGuard struct{}

func (*deniedAssessmentGuard) Authorize(context.Context, commandauth.Request) (commandauth.Decision, error) {
	return commandauth.Decision{}, commandauth.ErrNotAuthorized
}

func (fixture assessmentHTTPFixture) serviceRepositoryService() *thirdparty.Service {
	return thirdparty.NewService(fixture.repository)
}

func terminalHTTPAssessmentSetup(t *testing.T, fixture assessmentHTTPFixture) (thirdparty.Assessment, thirdparty.AssessmentSetupJob) {
	t.Helper()
	now := time.Now().UTC()
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner", Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)})
	form, err := fixture.forms.AssessmentFormRevision(context.Background(), "bank", "entity-a", "form-1", 3)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := fixture.service.StartAssessment(ctx, thirdparty.Actor{}, fixture.vendor.Relationship.ID, thirdparty.StartAssessmentInput{
		RelationshipVersion: fixture.vendor.Relationship.Version, FormTemplateID: form.ID, FormTemplateVersion: form.Version, ReviewDueAt: now.Add(14 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := fixture.repository.ClaimAssessmentSetupJobs(context.Background(), "worker-terminal", assessment.CreatedAt, time.Minute, 1, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim terminal setup = (%#v, %v)", claimed, err)
	}
	terminal, err := fixture.repository.FailAssessmentSetupJob(context.Background(), claimed[0], 1, thirdparty.AssessmentSetupFailureMatter, assessment.CreatedAt.Add(time.Second), assessment.CreatedAt.Add(time.Minute))
	if err != nil || terminal.State != thirdparty.AssessmentJobFailed {
		t.Fatalf("terminal setup = (%#v, %v)", terminal, err)
	}
	return assessment, terminal
}

func sendHTTPVendorAssessmentRequest(t *testing.T, fixture assessmentHTTPFixture) thirdparty.SendRequestOutcome {
	t.Helper()
	deadline := time.Now().UTC().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	body := []byte(`{"expected_version":` + jsonInt(fixture.assessment.Version) + `,"audience":"security@vendor.example","deadline":"` + deadline + `","invitation_ttl_minutes":60}`)
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+fixture.assessment.ID+"/send-request", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("initial request expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var outcome thirdparty.SendRequestOutcome
	if err := json.NewDecoder(response.Body).Decode(&outcome); err != nil {
		t.Fatal(err)
	}
	return outcome
}

func jsonInt(value int64) string {
	data, _ := json.Marshal(value)
	return string(data)
}
