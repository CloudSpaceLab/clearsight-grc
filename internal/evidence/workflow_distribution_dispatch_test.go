package evidence

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWorkflowDistributionDispatcherIssuesRedeemableCanonicalRoute(t *testing.T) {
	now := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": testSecurityKey(0x31)})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMemoryRepository(nil, nil)
	store := NewMemoryDistributionStore(repo, stubDistributionFormReader{form: activeDistributionForm()}, keyring)
	store.now = func() time.Time { return now }
	distributions := NewDistributionService(store)
	distributions.now = func() time.Time { return now }
	accessStore := NewMemoryDistributionAccessStore(store)
	access, err := NewDistributionAccessService(accessStore, keyring, &recordingOTPDelivery{}, testSecurityKey(0x42), 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	access.now = func() time.Time { return now }
	dispatcher := NewWorkflowDistributionDispatcher(distributions, access)

	requestInput := CreateRequestInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "subject-a",
		Title: "Complete vendor registration", Purpose: "Collect registration evidence.", WhyYou: "You are the vendor contact.", Sensitivity: "CONFIDENTIAL",
		AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "vendor@example.test"}, EstimatedMinutes: 5,
		Deadline: now.Add(48 * time.Hour), KnownFacts: map[string]string{"vendor_legal_name": "Example Vendor Ltd"},
		Presentation: activeDistributionForm().Presentation, Sections: activeDistributionForm().Sections,
		Fields: requestFieldsFromContract(activeDistributionForm().Fields), FormTemplateID: "form-a", FormTemplateVersion: 3,
		Origin: RequestOrigin{Type: "THIRD_PARTY_ASSESSMENT", ID: "assessment-a", Version: 1}, CreatedBy: "actor-a",
	}
	result, err := dispatcher.Dispatch(WithRequestOriginAuthority(context.Background(), requestInput.Origin.Type), WorkflowDistributionDispatchInput{
		Request: requestInput, AccessPolicy: AccessDirectMagicLink, RouteExpiresAt: now.Add(24 * time.Hour), AudienceHint: "v***@example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Request.Origin != requestInput.Origin || result.Route.RouteID == "" || result.Route.Selector == "" || result.Distribution.Status != DistributionOpen {
		t.Fatalf("canonical dispatch = %#v", result)
	}
	start, err := access.StartDistributionAccess(context.Background(), result.Route.Selector)
	if err != nil || start.Policy != AccessDirectMagicLink {
		t.Fatalf("start canonical access = (%#v, %v)", start, err)
	}
	redeemed, err := access.RedeemDirectRoute(context.Background(), result.Route.Selector)
	if err != nil || redeemed.RequestID != result.Request.ID {
		t.Fatalf("redeem canonical access = (%#v, %v)", redeemed, err)
	}
	replacementExpiry := now.Add(36 * time.Hour)
	replacement, err := dispatcher.Resume(context.Background(), requestInput.TenantID, requestInput.LegalEntityID, result.Request.ID, requestInput.CreatedBy, replacementExpiry)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Route.RouteID == result.Route.RouteID || replacement.Route.Selector == "" || !replacement.Route.ExpiresAt.Equal(replacementExpiry) || !replacement.Distribution.RouteExpiresAt.Equal(replacementExpiry) {
		t.Fatalf("canonical replacement did not apply the requested future expiry: %#v", replacement)
	}
	secondReplacementExpiry := now.Add(40 * time.Hour)
	secondReplacement, err := dispatcher.Resume(context.Background(), requestInput.TenantID, requestInput.LegalEntityID, result.Request.ID, requestInput.CreatedBy, secondReplacementExpiry)
	if err != nil || secondReplacement.Route.RouteID == "" {
		t.Fatalf("second reissue failed: %#v %v", secondReplacement, err)
	}
	active, err := accessStore.ListActiveAccessRoutes(context.Background(), requestInput.TenantID, requestInput.LegalEntityID, result.Distribution.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 3 {
		t.Fatalf("each reissue must mint exactly one link; got %d active routes after two reissues", len(active))
	}
	if err := dispatcher.RevokeRequestCapabilities(context.Background(), requestInput.TenantID, result.Request.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := access.SessionRequest(context.Background(), redeemed.SessionToken); err == nil {
		t.Fatal("revoking request capabilities left the redeemed session usable")
	}
	if _, err := access.RedeemDirectRoute(context.Background(), replacement.Route.Selector); err == nil {
		t.Fatal("revoking request capabilities left the replacement route usable")
	}
}

func TestWorkflowDistributionDispatcherRejectsUnownedReservedOrigin(t *testing.T) {
	now := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": testSecurityKey(0x31)})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMemoryRepository(nil, nil)
	store := NewMemoryDistributionStore(repo, stubDistributionFormReader{form: activeDistributionForm()}, keyring)
	dispatcher := NewWorkflowDistributionDispatcher(NewDistributionService(store), &DistributionAccessService{store: NewMemoryDistributionAccessStore(store)})
	form := activeDistributionForm()
	_, err = dispatcher.Dispatch(context.Background(), WorkflowDistributionDispatchInput{
		Request: CreateRequestInput{
			TenantID: "tenant-a", LegalEntityID: "entity-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "subject-a",
			Title: "Complete vendor registration", Purpose: "Collect registration evidence.", WhyYou: "You are the vendor contact.", Sensitivity: "CONFIDENTIAL",
			AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "vendor@example.test"}, EstimatedMinutes: 5,
			Deadline: now.Add(48 * time.Hour), Presentation: form.Presentation, Sections: form.Sections,
			Fields: requestFieldsFromContract(form.Fields), FormTemplateID: "form-a", FormTemplateVersion: 3,
			Origin: RequestOrigin{Type: "THIRD_PARTY_WORK", ID: "work-a", Version: 1}, CreatedBy: "actor-a",
		},
		AccessPolicy: AccessDirectEmailOTP, RouteExpiresAt: now.Add(24 * time.Hour),
	})
	if err == nil || !strings.Contains(err.Error(), "request origin is reserved") {
		t.Fatalf("unowned reserved request origin error = %v", err)
	}
}
