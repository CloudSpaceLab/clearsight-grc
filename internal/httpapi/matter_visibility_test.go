package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestRestrictedMatterVisibility(t *testing.T) {
	matter := continuity.Matter{TenantID: "bank-a", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-1"]}`)}
	if canReadMatter(context.Background(), matter) {
		t.Fatal("restricted matter was visible without a verified actor")
	}
	unauthorized := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-2", LegalEntityID: "bank-ng"})
	if canReadMatter(unauthorized, matter) {
		t.Fatal("restricted matter was visible to an unlisted principal")
	}
	authorized := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-1", LegalEntityID: "bank-ng"})
	if !canReadMatter(authorized, matter) {
		t.Fatal("restricted matter was hidden from an allowed principal")
	}
	wildcard := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "oversight", LegalEntityID: "*"})
	if canReadMatter(wildcard, matter) {
		t.Fatal("legal-entity wildcard bypassed the explicit allow-list")
	}
	wrongTenant := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-b", PrincipalID: "person-1", LegalEntityID: "bank-ng"})
	if canReadMatter(wrongTenant, matter) {
		t.Fatal("matter was visible across tenants")
	}
}

func TestAddressVerificationAssignmentImmediatelyRevokesSupersededAssigneeRead(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank-a", LegalEntityID: "bank-ng", Type: continuity.MatterVendorReview, Priority: 4,
		Title: "Verify the registered address", Summary: "A staff member must verify the vendor address.", Scope: json.RawMessage(`{"access":"INTERNAL"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AddActionInput{
		TenantID: "bank-a", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Verify address", Description: "Visit the address and provide evidence.", OwnerPrincipalID: "staff-old", ActorID: "owner", OriginKey: "thirdparty-address-verification",
	})
	if err != nil {
		t.Fatal(err)
	}
	action := matter.Actions[0]
	matter, err = service.AssignAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AssignActionInput{
		TenantID: "bank-a", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: "staff-new", ActorID: "owner", Rationale: "Reassign the visit to the available verifier.",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service}}
	oldRequest := evidence.Request{TenantID: "bank-a", LegalEntityID: "bank-ng", SubjectType: "MATTER", SubjectID: matter.Matter.ID, CreatedBy: "owner", Recipient: evidence.Recipient{Type: evidence.RecipientInternalPrincipal, PrincipalID: "staff-old", State: evidence.RecipientStateAssigned}, Origin: evidence.RequestOrigin{Type: "THIRD_PARTY_ADDRESS_VERIFICATION", ID: action.ID, Version: action.Version}}
	oldActor := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank-a", LegalEntityID: "bank-ng", PrincipalID: "staff-old"})
	if api.canReadEvidenceRequest(oldActor, oldRequest) {
		t.Fatal("superseded address verifier retained access after Action reassignment")
	}
	currentRequest := oldRequest
	currentRequest.Recipient.PrincipalID = "staff-new"
	currentRequest.Origin.Version = matter.Actions[0].Version
	newActor := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank-a", LegalEntityID: "bank-ng", PrincipalID: "staff-new"})
	if !api.canReadEvidenceRequest(newActor, currentRequest) {
		t.Fatal("current address verifier could not read the exact Action-version request")
	}
}

func TestAddressVerificationAssignmentImmediatelyRevokesSupersededAssigneeUpload(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank-a", LegalEntityID: "bank-ng", Type: continuity.MatterVendorReview, Priority: 4,
		Title: "Verify the registered address", Summary: "A staff member must verify the vendor address.", Scope: json.RawMessage(`{"access":"INTERNAL"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AddActionInput{
		TenantID: "bank-a", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Verify address", Description: "Visit the address and provide evidence.", OwnerPrincipalID: "staff-old", ActorID: "owner", OriginKey: "thirdparty-address-verification",
	})
	if err != nil {
		t.Fatal(err)
	}
	action := matter.Actions[0]
	matter, err = service.AssignAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AssignActionInput{
		TenantID: "bank-a", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: "staff-new", ActorID: "owner", Rationale: "Reassign the visit to the available verifier.",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := evidence.Request{
		ID: "request-address", TenantID: "bank-a", LegalEntityID: "bank-ng", SubjectType: "MATTER", SubjectID: matter.Matter.ID,
		Title: "Verify address", Status: evidence.RequestReady, CreatedBy: "owner", Version: 1,
		Recipient: evidence.Recipient{Type: evidence.RecipientInternalPrincipal, PrincipalID: "staff-old", State: evidence.RecipientStateAssigned},
		Origin:    evidence.RequestOrigin{Type: "THIRD_PARTY_ADDRESS_VERIFICATION", ID: action.ID, Version: action.Version},
	}
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, []evidence.Request{request}), evidence.NewMemoryObjectStore())
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", "staff-old", "bank-ng"),
		Evidence: evidenceService, Continuity: service, MaxArtifactBytes: 1 << 20,
	})
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("tenant_id", "bank-a")
	_ = writer.WriteField("request_id", request.ID)
	file, createErr := writer.CreateFormFile("file", "address.jpg")
	if createErr != nil {
		t.Fatal(createErr)
	}
	_, _ = file.Write([]byte("stale verifier evidence"))
	_ = writer.Close()
	upload := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/artifacts", &body)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, upload)
	if response.Code != http.StatusNotFound {
		t.Fatalf("superseded assignee upload returned %d: %s", response.Code, response.Body.String())
	}
}

func TestMalformedRestrictionFailsClosed(t *testing.T) {
	actor := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-1", LegalEntityID: "bank-ng"})
	for _, scope := range []json.RawMessage{
		json.RawMessage(`{"access":`),
		json.RawMessage(`{"access":"RESTRICTED"}`),
		json.RawMessage(`{"access":"SECRET"}`),
	} {
		if canReadMatter(actor, continuity.Matter{TenantID: "bank-a", Scope: scope}) {
			t.Fatalf("malformed or unsupported access policy was visible: %s", scope)
		}
	}
}
