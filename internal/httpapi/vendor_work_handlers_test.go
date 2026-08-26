package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestVendorWorkHandlersUseVerifiedRelationshipAndReturnTruthfulDeliveryState(t *testing.T) {
	links := thirdparty.NewMemoryRelationshipLinkRepository()
	links.AllowRelationship("bank", "entity-a", "relationship-1")
	links.AllowTarget("bank", "entity-a", thirdparty.LinkTargetProgram, "program-1")
	link, err := thirdparty.NewRelationshipLinkService(links).Link(context.Background(), thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner"}, "relationship-1", thirdparty.LinkRelationshipInput{TargetType: thirdparty.LinkTargetProgram, TargetID: "program-1", PurposeCode: "EVIDENCE_PROVIDER", PurposeLabel: "Evidence provider"})
	if err != nil {
		t.Fatal(err)
	}
	forms := monitoring.NewMemoryRepository()
	_, err = forms.CreateFormRevision(context.Background(), monitoring.FormTemplate{ID: "form-1", TenantID: "bank", LegalEntityID: "entity-a", ProgramID: "program-1", Name: "Service confirmation", Purpose: "Confirm current service information.", Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic, AllowModeSwitch: true}, Sections: []formcontract.Section{{ID: "service", Title: "Service"}}, Fields: []monitoring.TemplateField{{ID: "current", SectionID: "service", Label: "Is this information current?", Type: formcontract.TypeYesNo, Required: true}}, Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	workService, err := thirdparty.NewVendorWorkService(thirdparty.NewMemoryVendorWorkRepository(), links, evidenceService, forms, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	relationships := thirdparty.NewMemoryRepository()
	_, err = relationships.CreateRelationship(context.Background(), thirdparty.CreateRecord{Vendor: thirdparty.Vendor{ID: "vendor-1", TenantID: "bank", LegalName: "Northstar Hosting Limited", Status: thirdparty.VendorActive, Version: 1}, Relationship: thirdparty.Relationship{ID: "relationship-1", TenantID: "bank", LegalEntityID: "entity-a", VendorID: "vendor-1", ServiceName: "Managed transaction screening", BusinessOwnerPrincipalID: "verified-owner", Status: thirdparty.RelationshipActive, Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	workService.ConfigureRelationshipReader(relationships)
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory", Identity: identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"), ThirdPartyWork: workService})
	due := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	body := `{"relationship_link_id":"` + link.ID + `","purpose":"Confirm the service information needed for this Program.","instructions":"Review the known details and correct anything that changed.","form_template_id":"form-1","form_template_version":1,"presentation":"WIZARD","vendor_audience":"security@vendor.example","due_at":"` + due + `"}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/relationship-1/work/prepare", bytes.NewBufferString(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("prepare status=%d body=%s", response.Code, response.Body.String())
	}
	var prepared thirdparty.VendorWorkRequest
	if err := json.NewDecoder(response.Body).Decode(&prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.RelationshipID != "relationship-1" || prepared.OwnerPrincipalID != "verified-owner" || prepared.Presentation != formcontract.PresentationWizard {
		t.Fatalf("prepared = %#v", prepared)
	}

	send := httptest.NewRecorder()
	sendBody := `{"expected_version":` + jsonNumber(prepared.Version) + `,"vendor_audience":"security@vendor.example","invitation_ttl_minutes":60}`
	handler.ServeHTTP(send, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/relationship-1/work/"+prepared.ID+"/send", bytes.NewBufferString(sendBody)))
	if send.Code != http.StatusOK {
		t.Fatalf("send status=%d body=%s", send.Code, send.Body.String())
	}
	var outcome thirdparty.VendorWorkSendOutcome
	if err := json.NewDecoder(send.Body).Decode(&outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.State != thirdparty.VendorWorkDeliveryLinkAvailable || outcome.CaptureURL == "" || outcome.Invitation == nil || outcome.Invitation.Token != "" {
		t.Fatalf("send outcome = %#v", outcome)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/vendor-work?relationship_id=relationship-1&limit=10", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
}

func TestVendorWorkAcceptanceBlockedReturnsActionableConflict(t *testing.T) {
	response := httptest.NewRecorder()
	writeVendorWorkError(response, thirdparty.ErrVendorWorkAcceptanceBlocked)
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "vendor_work_acceptance_blocked" || body.Message != "A submitted document is pending inspection, quarantined or unavailable. Wait for inspection or request a replacement before accepting this response." {
		t.Fatalf("error = %#v", body)
	}
}

func jsonNumber(value int64) string { raw, _ := json.Marshal(value); return string(raw) }
