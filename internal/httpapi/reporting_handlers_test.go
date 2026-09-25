package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
)

const (
	reportingTenantID     = "00000000-0000-7000-8000-000000000601"
	reportingEntityID     = "00000000-0000-7000-8000-000000000602"
	reportingMakerID      = "00000000-0000-7000-8000-000000000603"
	reportingReviewerID   = "00000000-0000-7000-8000-000000000604"
	reportingAuthorizerID = "00000000-0000-7000-8000-000000000605"
	reportingPerformerID  = "00000000-0000-7000-8000-000000000606"
)

type reportingHTTPAuthority struct {
	mu                 sync.Mutex
	expected           map[authority.Responsibility]string
	failResponsibility authority.Responsibility
	calls              map[authority.Responsibility]int
}

func (a *reportingHTTPAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.calls == nil {
		a.calls = make(map[authority.Responsibility]int)
	}
	a.calls[input.Responsibility]++
	if input.Responsibility == a.failResponsibility {
		return authority.Resolution{}, errors.New("current report authority is unavailable")
	}
	principal := a.expected[input.Responsibility]
	if principal == "" {
		return authority.Resolution{}, errors.New("no current report authority route")
	}
	return authority.Resolution{
		Principal: authority.Principal{ID: principal}, RuleID: "report-test-route", PolicyVersion: "report-test-v1",
	}, nil
}

func (a *reportingHTTPAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (a *reportingHTTPAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (a *reportingHTTPAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}
func (a *reportingHTTPAuthority) callCount(responsibility authority.Responsibility) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls[responsibility]
}

func reportingHTTPFixture(t *testing.T) (http.Handler, *reporting.Service, *reporting.MemoryRepository, evidence.ObjectStore, *reportingHTTPAuthority) {
	t.Helper()
	repository := reporting.NewMemoryRepository()
	objects := evidence.NewMemoryObjectStore()
	authorityChecker := &reportingHTTPAuthority{expected: map[authority.Responsibility]string{
		authority.ResponsibilityProposer:   reportingMakerID,
		authority.ResponsibilityReviewer:   reportingReviewerID,
		authority.ResponsibilityAuthorizer: reportingAuthorizerID,
		authority.ResponsibilityPerformer:  reportingPerformerID,
	}}
	service := reporting.NewService(repository, objects, authorityChecker)
	guard, err := commandauth.New(authorityChecker, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory",
		Identity:     identity.NewDevelopmentAuthenticator(reportingTenantID, reportingMakerID, reportingEntityID, "GRC_ADMIN"),
		CommandGuard: guard, Reporting: service,
	})
	return handler, service, repository, objects, authorityChecker
}

func reportingRequest(handler http.Handler, method, path, principal string, roles []string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("X-ClearSight-Demo-Principal", principal)
	if len(roles) > 0 {
		request.Header.Set("X-ClearSight-Demo-Roles", strings.Join(roles, ","))
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestFilterVocabularyEndpointIsPublished(t *testing.T) {
	handler, _, _, _, _ := reportingHTTPFixture(t)
	response := reportingRequest(handler, http.MethodGet, "/api/v1/reports/filter-fields", reportingMakerID, nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("filter vocabulary status = %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Fields []reporting.ReportFilterFieldDefinition `json:"fields"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Fields) != len(reporting.ReportFilterFieldVocabulary) {
		t.Fatalf("filter fields = %d, want %d", len(body.Fields), len(reporting.ReportFilterFieldVocabulary))
	}
	found := false
	for _, field := range body.Fields {
		if field.Field == reporting.ReportFieldCrossBorder && field.Dataset == reporting.DatasetProcessingActivities && len(field.Operators) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("published vocabulary omitted the processing-activity cross-border filter")
	}
}

func TestUnknownFilterFieldIsRejectedWithA400NamingTheField(t *testing.T) {
	handler, _, _, _, _ := reportingHTTPFixture(t)
	body := `{"tenant_id":"00000000-0000-7000-8000-000000000699","legal_entity_id":"00000000-0000-7000-8000-000000000698","maker_id":"forged-maker","code":"SAFE-ACTIVITY-REPORT","name":"Safe activity report","dataset":"PROCESSING_ACTIVITIES","scope_kind":"LEGAL_ENTITY","format":"CSV","filter":{"kind":"condition","field":"secret_column","operator":"is","value":"x"}}`
	response := reportingRequest(handler, http.MethodPost, "/api/v1/reports/definitions", reportingMakerID, nil, body)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown filter field status = %d: %s", response.Code, response.Body.String())
	}
	const want = "{\"error\":\"report_filter_field_unavailable\",\"message\":\"The report filter field \\\"secret_column\\\" is not available for this report. Available fields are: automated_decision_making, cross_border_transfer, lawful_basis, matter_id, missing_data_subjects, missing_lawful_basis, missing_owner, name, owner_principal_id, program_id, review_overdue, status. Choose an available field and save the report again.\"}\n"
	if response.Body.String() != want {
		t.Fatalf("unknown filter response = %q, want %q", response.Body.String(), want)
	}
}

func TestForgedScopeInTheRequestBodyIsOverwritten(t *testing.T) {
	handler, _, _, _, _ := reportingHTTPFixture(t)
	proposeBody := `{"tenant_id":"forged-tenant","legal_entity_id":"forged-entity","actor_id":"forged-actor","maker_id":"forged-maker","reviewer_id":"forged-reviewer","authorizer_id":"forged-authorizer","checker_id":"forged-checker","code":"ROPA-SCOPE-TEST","name":"Scope binding test","dataset":"PROCESSING_ACTIVITY_EXCEPTIONS","scope_kind":"LEGAL_ENTITY","format":"CSV","filter":{"kind":"group","operator":"and"}}`
	response := reportingRequest(handler, http.MethodPost, "/api/v1/reports/definitions", reportingMakerID, nil, proposeBody)
	if response.Code != http.StatusCreated {
		t.Fatalf("propose status = %d: %s", response.Code, response.Body.String())
	}
	var definition reporting.ReportDefinition
	if err := json.NewDecoder(response.Body).Decode(&definition); err != nil {
		t.Fatal(err)
	}
	if definition.TenantID != reportingTenantID || definition.LegalEntityID != reportingEntityID || definition.MakerID != reportingMakerID {
		t.Fatalf("propose trusted forged scope or actor: %#v", definition)
	}

	submit := func(path, principal, action string, expectedVersion int64) reporting.ReportDefinition {
		t.Helper()
		body := `{"tenant_id":"forged-transition-tenant","legal_entity_id":"forged-transition-entity","actor_id":"forged-transition-actor","reviewer_id":"forged-transition-reviewer","authorizer_id":"forged-transition-authorizer","checker_id":"forged-transition-checker","expected_version":` + jsonNumber(expectedVersion) + `,"checksum_seen":"` + definition.StoredChecksum + `","note":"Checked in the governed report workflow."}`
		response := reportingRequest(handler, http.MethodPost, path, principal, nil, body)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", action, response.Code, response.Body.String())
		}
		var next reporting.ReportDefinition
		if err := json.NewDecoder(response.Body).Decode(&next); err != nil {
			t.Fatal(err)
		}
		if next.TenantID != reportingTenantID || next.LegalEntityID != reportingEntityID {
			t.Fatalf("%s trusted forged scope: %#v", action, next)
		}
		definition = next
		return next
	}

	submit("/api/v1/reports/definitions/"+definition.ID+"/submit", reportingMakerID, "submit", definition.Version)
	submit("/api/v1/reports/definitions/"+definition.ID+"/review", reportingReviewerID, "review", definition.Version)
	activated := submit("/api/v1/reports/definitions/"+definition.ID+"/activate", reportingAuthorizerID, "activate", definition.Version)
	if activated.MakerID != reportingMakerID || activated.ReviewerID != reportingReviewerID || activated.CheckerID != reportingAuthorizerID {
		t.Fatalf("transition actors were not bound from verified identities: %#v", activated)
	}

	runBody := `{"tenant_id":"forged-run-tenant","legal_entity_id":"forged-run-entity","actor_id":"forged-run-actor","requested_by_ref":"forged-run-performer","definition_id":"` + definition.ID + `","expected_definition_version":` + jsonNumber(int64(definition.CurrentVersion)) + `}`
	runResponse := reportingRequest(handler, http.MethodPost, "/api/v1/reports/runs", reportingPerformerID, nil, runBody)
	if runResponse.Code != http.StatusCreated {
		t.Fatalf("run status = %d: %s", runResponse.Code, runResponse.Body.String())
	}
	var run reporting.ReportRun
	if err := json.NewDecoder(runResponse.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.TenantID != reportingTenantID || run.LegalEntityID != reportingEntityID || run.RequestedByRef != reportingPerformerID {
		t.Fatalf("run trusted forged scope or actor: %#v", run)
	}

	second := reportingRequest(handler, http.MethodPost, "/api/v1/reports/definitions", reportingMakerID, nil, `{"code":"ROPA-REJECT-SCOPE","name":"Reject scope test","dataset":"PROCESSING_ACTIVITIES","scope_kind":"LEGAL_ENTITY","format":"CSV","filter":{"kind":"group","operator":"and"}}`)
	if second.Code != http.StatusCreated {
		t.Fatalf("second propose = %d: %s", second.Code, second.Body.String())
	}
	var rejectDefinition reporting.ReportDefinition
	if err := json.NewDecoder(second.Body).Decode(&rejectDefinition); err != nil {
		t.Fatal(err)
	}
	transitionBody := func(version int64) string {
		return `{"tenant_id":"forged-reject-tenant","legal_entity_id":"forged-reject-entity","actor_id":"forged-reject-actor","expected_version":` + jsonNumber(version) + `,"checksum_seen":"` + rejectDefinition.StoredChecksum + `"}`
	}
	response = reportingRequest(handler, http.MethodPost, "/api/v1/reports/definitions/"+rejectDefinition.ID+"/submit", reportingMakerID, nil, transitionBody(rejectDefinition.Version))
	if response.Code != http.StatusOK {
		t.Fatalf("second submit = %d: %s", response.Code, response.Body.String())
	}
	if err := json.NewDecoder(response.Body).Decode(&rejectDefinition); err != nil {
		t.Fatal(err)
	}
	response = reportingRequest(handler, http.MethodPost, "/api/v1/reports/definitions/"+rejectDefinition.ID+"/reject", reportingReviewerID, nil, transitionBody(rejectDefinition.Version))
	if response.Code != http.StatusOK {
		t.Fatalf("reject = %d: %s", response.Code, response.Body.String())
	}
	if err := json.NewDecoder(response.Body).Decode(&rejectDefinition); err != nil {
		t.Fatal(err)
	}
	if rejectDefinition.TenantID != reportingTenantID || rejectDefinition.LegalEntityID != reportingEntityID {
		t.Fatalf("reject trusted forged scope: %#v", rejectDefinition)
	}
	historyResponse := reportingRequest(handler, http.MethodGet, "/api/v1/reports/definitions/"+rejectDefinition.ID+"/history", reportingReviewerID, nil, "")
	if historyResponse.Code != http.StatusOK {
		t.Fatalf("reject history = %d: %s", historyResponse.Code, historyResponse.Body.String())
	}
	var historyBody struct {
		Items []reporting.ReportDefinitionRevision `json:"items"`
	}
	if err := json.NewDecoder(historyResponse.Body).Decode(&historyBody); err != nil {
		t.Fatal(err)
	}
	if len(historyBody.Items) != 1 || historyBody.Items[0].ReviewedBy != reportingReviewerID {
		t.Fatalf("reject history did not bind the verified reviewer: %#v", historyBody.Items)
	}

	retireBody := `{"tenant_id":"forged-retire-tenant","legal_entity_id":"forged-retire-entity","actor_id":"forged-retire-actor","reviewer_id":"forged-retire-reviewer","authorizer_id":"forged-retire-authorizer","checker_id":"forged-retire-checker","expected_version":` + jsonNumber(activated.Version) + `,"checksum_seen":"` + activated.StoredChecksum + `"}`
	response = reportingRequest(handler, http.MethodPost, "/api/v1/reports/definitions/"+activated.ID+"/retire", reportingAuthorizerID, nil, retireBody)
	if response.Code != http.StatusOK {
		t.Fatalf("retire = %d: %s", response.Code, response.Body.String())
	}
	var retired reporting.ReportDefinition
	if err := json.NewDecoder(response.Body).Decode(&retired); err != nil {
		t.Fatal(err)
	}
	if retired.TenantID != reportingTenantID || retired.LegalEntityID != reportingEntityID || retired.CheckerID != reportingAuthorizerID {
		t.Fatalf("retire trusted forged scope or actor: %#v", retired)
	}
}

func TestRunDownloadReAuthorisesAndRequiresReportDownloadPermission(t *testing.T) {
	handler, service, repository, objects, authorityChecker := reportingHTTPFixture(t)
	run, _ := installReadyHTTPReport(t, service, repository, objects, time.Now().UTC())

	list := reportingRequest(handler, http.MethodGet, "/api/v1/reports/runs", reportingPerformerID, []string{"CRO"}, "")
	if list.Code != http.StatusOK {
		t.Fatalf("list without report-download permission = %d: %s", list.Code, list.Body.String())
	}
	forbidden := reportingRequest(handler, http.MethodGet, "/api/v1/reports/runs/"+run.ID+"/download", reportingPerformerID, []string{"CRO"}, "")
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("download without report permission = %d: %s", forbidden.Code, forbidden.Body.String())
	}

	allowed := reportingRequest(handler, http.MethodGet, "/api/v1/reports/runs/"+run.ID+"/download", reportingPerformerID, []string{"CCO"}, "")
	if allowed.Code != http.StatusOK {
		t.Fatalf("authorized download = %d: %s", allowed.Code, allowed.Body.String())
	}
	firstCalls := authorityChecker.callCount(authority.ResponsibilityPerformer)
	if firstCalls < 2 { // command-independent route permission plus service re-authorization
		t.Fatalf("first download authority calls = %d, want separate route and service checks", firstCalls)
	}

	authorityChecker.mu.Lock()
	authorityChecker.failResponsibility = authority.ResponsibilityPerformer
	authorityChecker.mu.Unlock()
	unavailable := reportingRequest(handler, http.MethodGet, "/api/v1/reports/runs/"+run.ID+"/download", reportingPerformerID, []string{"CCO"}, "")
	if unavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("download after authority outage = %d: %s", unavailable.Code, unavailable.Body.String())
	}
	if len(repository.Downloads()) != 1 {
		t.Fatalf("download receipts = %#v, want only the authorized download", repository.Downloads())
	}
}

func TestRunDownloadSendsNoStoreAndContentDisposition(t *testing.T) {
	handler, service, repository, objects, _ := reportingHTTPFixture(t)
	run, data := installReadyHTTPReport(t, service, repository, objects, time.Now().UTC())
	response := reportingRequest(handler, http.MethodGet, "/api/v1/reports/runs/"+run.ID+"/download", reportingPerformerID, []string{"CCO"}, "")
	if response.Code != http.StatusOK {
		t.Fatalf("download = %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Disposition"); got != `attachment; filename="ROPA-OPEN-EXCEPTIONS.csv"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/csv; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	digest := sha256.Sum256(data)
	if got, want := response.Header().Get("ETag"), `"`+hex.EncodeToString(digest[:])+`"`; got != want {
		t.Fatalf("ETag = %q, want %q", got, want)
	}
	if !bytes.Equal(response.Body.Bytes(), data) {
		t.Fatalf("download body = %q, want %q", response.Body.Bytes(), data)
	}
}

func TestRunDownloadRefusesAnExpiredRunWithAnExplanation(t *testing.T) {
	handler, service, repository, objects, authorityChecker := reportingHTTPFixture(t)
	run, _ := installReadyHTTPReport(t, service, repository, objects, time.Now().UTC().Add(-8*24*time.Hour))
	response := reportingRequest(handler, http.MethodGet, "/api/v1/reports/runs/"+run.ID+"/download", reportingPerformerID, []string{"CCO"}, "")
	if response.Code != http.StatusGone {
		t.Fatalf("expired download = %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Run the report again") {
		t.Fatalf("expired response does not explain recovery: %s", response.Body.String())
	}

	authorityChecker.mu.Lock()
	authorityChecker.failResponsibility = authority.ResponsibilityPerformer
	authorityChecker.mu.Unlock()
	unavailable := reportingRequest(handler, http.MethodGet, "/api/v1/reports/runs/"+run.ID+"/download", reportingPerformerID, []string{"CCO"}, "")
	if unavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("expired download disclosed run state during authority outage: %d %s", unavailable.Code, unavailable.Body.String())
	}
}

func installReadyHTTPReport(t *testing.T, service *reporting.Service, repository *reporting.MemoryRepository, objects evidence.ObjectStore, createdAt time.Time) (reporting.ReportRun, []byte) {
	t.Helper()
	now := createdAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	service.Now = func() time.Time { return now }
	actor := func(principal string) context.Context {
		return identity.WithActor(context.Background(), identity.Actor{
			TenantID: reportingTenantID, LegalEntityID: reportingEntityID, PrincipalID: principal,
			Kind: "PERSON", IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
		})
	}
	definition, err := service.Propose(actor(reportingMakerID), reporting.ProposeInput{
		Code: "ROPA-OPEN-EXCEPTIONS", Name: "Open processing exceptions", Dataset: reporting.DatasetProcessingActivityExceptions,
		ScopeKind: reporting.ScopeLegalEntity, Format: reporting.FormatCSV,
		Filter: &reporting.ReportFilterExpression{Kind: "group", Operator: "and"},
	})
	if err != nil {
		t.Fatal(err)
	}
	definition, err = service.Submit(actor(reportingMakerID), reporting.DefinitionTransitionInput{
		Scope: reportingScope(), DefinitionID: definition.ID, ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
	})
	if err != nil {
		t.Fatal(err)
	}
	definition, err = service.Review(actor(reportingReviewerID), reporting.DefinitionTransitionInput{
		Scope: reportingScope(), DefinitionID: definition.ID, ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
	})
	if err != nil {
		t.Fatal(err)
	}
	definition, err = service.Activate(actor(reportingAuthorizerID), reporting.DefinitionTransitionInput{
		Scope: reportingScope(), DefinitionID: definition.ID, ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.CreateRun(actor(reportingPerformerID), reporting.CreateRunInput{
		Scope: reportingScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := repository.ClaimQueuedRuns(context.Background(), reportingScope(), "report-http-test", 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim report run: runs=%#v err=%v", claimed, err)
	}
	dataKey := "reports/" + reportingTenantID + "/" + reportingEntityID + "/" + run.ID + "/report.csv"
	manifestKey := "reports/" + reportingTenantID + "/" + reportingEntityID + "/" + run.ID + "/manifest.json"
	data := []byte("id,name\nrow-1,Customer account opening\n")
	if objects == nil {
		t.Fatal("report object store is required")
	}
	if _, err := objects.Put(context.Background(), dataKey, bytes.NewReader(data), reporting.MaxReportRunBytes); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	completed := now.Add(time.Second)
	run = claimed[0]
	run.Status = reporting.RunReady
	run.RowCount = 1
	run.DataObjectKey = dataKey
	run.DataSHA256 = hex.EncodeToString(digest[:])
	run.ManifestObjectKey = manifestKey
	run.ManifestSHA256 = strings.Repeat("b", 64)
	run.CompletedAt = &completed
	run.ExpiresAt = now.Add(reporting.ReportRunRetention)
	run, err = repository.CompleteRun(context.Background(), reportingScope(), run)
	if err != nil {
		t.Fatal(err)
	}
	service.Now = time.Now
	return run, data
}

func reportingScope() reporting.ReportScope {
	return reporting.ReportScope{TenantID: reportingTenantID, LegalEntityID: reportingEntityID}
}
