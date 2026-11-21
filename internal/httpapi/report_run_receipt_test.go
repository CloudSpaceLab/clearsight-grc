package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
)

func TestReportRunHTTPIdempotencyUsesVerifiedActor(t *testing.T) {
	handler, service, repository, objects, _ := reportingHTTPFixture(t)
	existing, _ := installReadyHTTPReport(t, service, repository, objects, time.Now().UTC())
	body := fmt.Sprintf(`{"definition_id":%q,"expected_definition_version":%d,"idempotency_key":"550e8400-e29b-41d4-a716-446655440000","requested_by_ref":"forged"}`, existing.DefinitionID, existing.DefinitionVersion)
	request := func(principal string) *reporting.ReportRun {
		t.Helper()
		response := reportingRequest(handler, http.MethodPost, "/api/v1/reports/runs", principal, nil, body)
		if response.Code != http.StatusCreated {
			t.Fatalf("status=%d %s", response.Code, response.Body.String())
		}
		var run reporting.ReportRun
		if err := json.Unmarshal(response.Body.Bytes(), &run); err != nil {
			t.Fatal(err)
		}
		return &run
	}
	first, second := request(reportingPerformerID), request(reportingPerformerID)
	if first.ID != second.ID || first.RequestedByRef != reportingPerformerID {
		t.Fatalf("receipts do not match: %+v %+v", first, second)
	}
	revokedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/reports/runs", strings.NewReader(body))
	revokedRequest.Header.Set("X-ClearSight-Demo-Principal", reportingPerformerID)
	revokedRequest.Header.Set("X-Test-Privacy-Revoked", "true")
	revoked := httptest.NewRecorder()
	handler.ServeHTTP(revoked, revokedRequest)
	if revoked.Code != http.StatusForbidden || !strings.Contains(revoked.Body.String(), "report_privacy_forbidden") {
		t.Fatalf("revoked receipt: %d %s", revoked.Code, revoked.Body.String())
	}
	denied := reportingRequest(handler, http.MethodPost, "/api/v1/reports/runs", reportingMakerID, nil, body)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unauthorized receipt: %d %s", denied.Code, denied.Body.String())
	}
}
