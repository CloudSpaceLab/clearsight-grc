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
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
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
	_, err = forms.CreateFormRevision(context.Background(), monitoring.FormTemplate{ID: "form-1", TenantID: "bank", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "VENDOR-CERTIFICATION-REFRESH", Name: "Certification refresh", Purpose: "Collect current ISO 27001 and PCI DSS evidence.", Sensitivity: "CONFIDENTIAL", Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic, AllowModeSwitch: true}, Sections: []formcontract.Section{{ID: "certification", Title: "Certifications"}}, Fields: []monitoring.TemplateField{{ID: "current", SectionID: "certification", Label: "Are the certifications current?", Type: formcontract.TypeYesNo, Required: true}}, Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	evidenceRepository := newScopedVendorEvidenceRepository("bank", "entity-a", "relationship-1")
	evidenceRepository.MemoryRepository = evidence.NewMemoryRepositoryWithRecipientCandidates(nil, nil, []evidence.RecipientCandidate{{PrincipalID: "verified-owner", TenantID: "bank", Active: true, Kind: "PERSON", ReadableSubjects: map[string]bool{"VENDOR_RELATIONSHIP:relationship-1": true}}})
	evidenceService := evidence.NewService(evidenceRepository, evidence.NewMemoryObjectStore())
	workService, err := thirdparty.NewVendorWorkService(thirdparty.NewMemoryVendorWorkRepository(), links, evidenceService, forms, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	var recipientKey, accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index], accessKey[index] = 0x31, 0x42
	}
	keyring, err := evidence.NewRecipientKeyring("handler-v1", map[string][32]byte{"handler-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	distributionStore := evidence.NewMemoryDistributionStore(evidenceRepository.MemoryRepository, vendorWorkHandlerFormReader{forms: forms}, keyring)
	distributions := evidence.NewDistributionService(distributionStore)
	otp := &summaryOTPDelivery{}
	access, err := evidence.NewDistributionAccessService(evidence.NewMemoryDistributionAccessStore(distributionStore), keyring, otp, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	workService.ConfigureDistributionDispatcher(evidence.NewWorkflowDistributionDispatcher(distributions, access))
	relationships := thirdparty.NewMemoryRepository()
	_, err = relationships.CreateRelationship(context.Background(), thirdparty.CreateRecord{Vendor: thirdparty.Vendor{ID: "vendor-1", TenantID: "bank", LegalName: "Northstar Hosting Limited", Status: thirdparty.VendorActive, Version: 1}, Relationship: thirdparty.Relationship{ID: "relationship-1", TenantID: "bank", LegalEntityID: "entity-a", VendorID: "vendor-1", ServiceName: "Managed transaction screening", BusinessOwnerPrincipalID: "verified-owner", Status: thirdparty.RelationshipActive, Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	workService.ConfigureRelationshipReader(relationships)
	distributions.ConfigureDocumentContexts(thirdparty.DocumentContextReader{Work: workService})
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory", Identity: identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"), ThirdPartyWork: workService, FormDistributions: distributions})
	due := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	body := `{"relationship_link_id":"` + link.ID + `","request_kind":"CERTIFICATION_REFRESH","purpose":"Collect current certification evidence for this Program.","instructions":"Provide the current ISO 27001 and PCI DSS evidence that applies to this service.","form_template_id":"form-1","form_template_version":1,"presentation":"WIZARD","vendor_audience":"security@vendor.example","due_at":"` + due + `"}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/relationship-1/work/prepare", bytes.NewBufferString(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("prepare status=%d body=%s", response.Code, response.Body.String())
	}
	var prepared thirdparty.VendorWorkRequest
	if err := json.NewDecoder(response.Body).Decode(&prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.RelationshipID != "relationship-1" || prepared.OwnerPrincipalID != "verified-owner" || prepared.RequestKind != thirdparty.VendorWorkCertificationRefresh || prepared.Presentation != formcontract.PresentationWizard {
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
	parsed, err := url.Parse(outcome.CaptureURL)
	if err != nil {
		t.Fatal(err)
	}
	fragment, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	selector := fragment.Get("form_access")
	start, err := access.StartDistributionAccess(context.Background(), selector)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := access.SendOTP(context.Background(), selector, start.Recipients[0].SelectorID)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := access.VerifyOTP(context.Background(), selector, challenge.ChallengeID, otp.code)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := access.GetResponseWorkspace(context.Background(), verified.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err = access.SaveResponseWorkspace(context.Background(), verified.SessionToken, evidence.SaveWorkspaceInput{ExpectedVersion: workspace.Workspace.Version, Edits: []evidence.FieldEdit{{FieldID: "current", Value: formcontract.TextAnswer("Yes"), BaseSequence: workspace.FieldSequences["current"]}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := access.SubmitResponseWorkspace(context.Background(), verified.SessionToken, evidence.SubmitWorkspaceInput{ExpectedVersion: workspace.Workspace.Version}); err != nil {
		t.Fatal(err)
	}
	target := &summaryTargetReader{allowed: true}
	workService.ConfigureTargetReader(target)
	var revisionID string
	for _, allowed := range []bool{true, false, true} {
		target.allowed = allowed
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses?limit=1", nil))
		var page evidence.CompletedResponsePage
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil {
			t.Fatalf("work summaries: %d %s", w.Code, w.Body.String())
		}
		if allowed {
			if len(page.Items) != 1 {
				t.Fatalf("authorized work response omitted: %s", w.Body.String())
			}
			revisionID = page.Items[0].ID
		} else if len(page.Items) != 0 || page.NextCursor != "" || strings.Contains(w.Body.String(), prepared.Purpose) {
			t.Fatalf("restricted work metadata leaked: %s", w.Body.String())
		}
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses/"+revisionID, nil))
		if allowed && w.Code != 200 || !allowed && w.Code != 404 {
			t.Fatalf("work exact response: %d %s", w.Code, w.Body.String())
		}
	}
}

type summaryOTPDelivery struct{ code string }

func (d *summaryOTPDelivery) DeliverDistributionOTP(_ context.Context, value evidence.DistributionOTPDelivery) error {
	d.code = value.Code
	return nil
}

type summaryTargetReader struct{ allowed bool }

func (r *summaryTargetReader) GetProgram(context.Context, string, string) (continuity.ProgramAggregate, error) {
	if !r.allowed {
		return continuity.ProgramAggregate{}, continuity.ErrNotFound
	}
	return continuity.ProgramAggregate{Program: continuity.Program{ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a"}}, nil
}
func (*summaryTargetReader) GetMatter(context.Context, string, string) (continuity.MatterAggregate, error) {
	return continuity.MatterAggregate{}, continuity.ErrNotFound
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

func TestVendorWorkRecipientMismatchExplainsHowToRecover(t *testing.T) {
	response := httptest.NewRecorder()
	writeVendorWorkError(response, thirdparty.ErrVendorWorkRecipientMismatch)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "vendor_work_recipient_mismatch" || body.Message != "Enter the contact used when this request was created. Create a new request if the vendor contact has changed." {
		t.Fatalf("error = %#v", body)
	}
}

func TestRetryVendorWorkRouteRequiresOwnerMaterialAuthority(t *testing.T) {
	routes := (&API{}).routes()
	for _, route := range routes {
		if route.Method == http.MethodPost && route.Path == "/api/v1/vendors/{id}/work/{request_id}/retry" {
			if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != "thirdparty.work.retry" || route.Command.Policy.ObjectType != "VENDOR_RELATIONSHIP" || route.Command.Policy.Responsibility != authority.ResponsibilityOwner {
				t.Fatalf("retry route = %#v", route)
			}
			return
		}
	}
	t.Fatal("vendor work retry route is missing")
}

func jsonNumber(value int64) string { raw, _ := json.Marshal(value); return string(raw) }

type vendorWorkHandlerFormReader struct{ forms *monitoring.MemoryRepository }

func (reader vendorWorkHandlerFormReader) GetDistributionFormRevision(ctx context.Context, tenantID, legalEntityID, formID string, version int64) (evidence.DistributionFormRevision, error) {
	form, err := reader.forms.ReusableFormRevision(ctx, tenantID, legalEntityID, formID, version)
	if err != nil {
		return evidence.DistributionFormRevision{}, err
	}
	return evidence.DistributionFormRevision{
		ID: form.ID, TenantID: form.TenantID, LegalEntityID: form.LegalEntityID, Version: form.Version,
		Sensitivity: form.Sensitivity, Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile,
		Sections: append([]formcontract.Section(nil), form.Sections...), Fields: append([]formcontract.Field(nil), form.Fields...),
		Active: form.Status == monitoring.LifecycleActive && form.IsCurrent,
	}, nil
}
