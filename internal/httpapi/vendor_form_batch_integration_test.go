package httpapi

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type vendorBatchRoute struct{ commandAuthorityStub }

func (s vendorBatchRoute) Resolve(_ context.Context, in authority.ResolveInput) (authority.Resolution, error) {
	principal := "verified-owner"
	if in.ObjectID == "restricted-relationship" {
		principal = "other-owner"
	}
	return authority.Resolution{Principal: authority.Principal{ID: principal}, RuleID: "vendor-owner-route", PolicyVersion: "v2"}, nil
}

func TestVendorBatchRecoversPreparedRequestAndUsesVerifiedOwner(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	relID := "00000000-0000-4000-8000-000000000101"
	forms := httpPolicyFormReader{form: evidence.DistributionFormRevision{ID: "form", TenantID: "bank", LegalEntityID: "entity", Version: 1, Active: true, Fields: []formcontract.Field{{ID: "current", Label: "Are certifications current?", Type: formcontract.TypeYesNo, Required: true}}}}
	var key [32]byte
	for i := range key {
		key[i] = 42
	}
	keyring, err := evidence.NewRecipientKeyring("test-key", map[string][32]byte{"test-key": key})
	if err != nil {
		t.Fatal(err)
	}
	store := evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), forms, keyring)
	distributions := evidence.NewDistributionService(store)
	relationships := thirdparty.NewMemoryRepository()
	_, err = relationships.CreateRelationship(ctx, thirdparty.CreateRecord{Vendor: thirdparty.Vendor{ID: "vendor", TenantID: "bank", LegalName: "Sample processing provider", Status: thirdparty.VendorActive, Version: 1}, Relationship: thirdparty.Relationship{ID: relID, TenantID: "bank", LegalEntityID: "entity", VendorID: "vendor", ServiceName: "Processing", BusinessOwnerPrincipalID: "verified-owner", Status: thirdparty.RelationshipActive, Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{FormDistributions: distributions, ThirdParty: thirdparty.NewService(relationships), Authority: vendorBatchRoute{}}}
	body := vendorFormBatchRequest{BatchID: "vendor-batch-security-2026", createFormDistributionRequest: createFormDistributionRequest{FormTemplateID: "form", FormTemplateVersion: 1, Title: "Certification review", Purpose: "Confirm certificates for processing", AccessPolicy: evidence.AccessDirectEmailOTP, EstimatedMinutes: 10, Deadline: now.Add(72 * time.Hour), RouteExpiresAt: now.Add(48 * time.Hour)}, Targets: []vendorFormRequestTarget{{RelationshipID: relID, Recipient: evidence.DistributionRecipientInput{Role: evidence.RecipientTo, Type: evidence.RecipientExternalAudience, Address: "vendor@example.test"}}}}
	actorEntity := "entity"
	send := func() []vendorFormRequestOutcome {
		t.Helper()
		raw, _ := json.Marshal(body)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/vendors/form-requests", strings.NewReader(string(raw)))
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: actorEntity, PrincipalID: "verified-owner", Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}))
		result := httptest.NewRecorder()
		api.requestVendorForms(result, request)
		if result.Code != 200 {
			t.Fatalf("status=%d body=%s", result.Code, result.Body.String())
		}
		var receipt struct {
			Items []vendorFormRequestOutcome `json:"items"`
		}
		if err = json.Unmarshal(result.Body.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(result.Body.String(), "vendor@example.test") || strings.Contains(result.Body.String(), "selector") {
			t.Fatal("recipient access leaked in batch receipt")
		}
		return receipt.Items
	}
	first := send()
	if len(first) != 1 || first[0].Status != "PREPARED" || first[0].DistributionID == "" {
		t.Fatalf("prepared=%+v", first)
	}
	access, err := evidence.NewDistributionAccessService(evidence.NewMemoryDistributionAccessStore(store), keyring, &summaryOTPDelivery{}, key, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	api.deps.FormDistributionAccess = access
	retry := send()
	if retry[0].Status != "CREATED" || retry[0].DistributionState != evidence.DistributionOpen || retry[0].DistributionID != first[0].DistributionID {
		t.Fatalf("retry=%+v", retry)
	}
	bundle, err := distributions.Get(ctx, "bank", "entity", retry[0].DistributionID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Distribution.CreatedBy != "verified-owner" || bundle.Recipients[0].AudienceHint != "Vendor contact" {
		t.Fatalf("verified actor or safe audience missing: %+v", bundle.Distribution)
	}
	final := send()
	if final[0].DistributionID != retry[0].DistributionID {
		t.Fatal("ambiguous retry duplicated request")
	}
	actorEntity = "*"
	body.LegalEntityID = "entity"
	wide := send()
	if wide[0].Status != "CREATED" || wide[0].DistributionID != retry[0].DistributionID {
		t.Fatalf("tenant-wide identity did not use the selected entity: %+v", wide)
	}
	body.BatchID = "vendor-batch-cross-entity-2026"
	body.LegalEntityID = "other-entity"
	wrongEntity := send()
	if wrongEntity[0].Status != "FAILED" || wrongEntity[0].DistributionID != "" {
		t.Fatalf("relationship escaped selected entity: %+v", wrongEntity)
	}
}
