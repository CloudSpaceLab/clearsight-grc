package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMatterOutcomeDefinitionHTTPStoresOnlyTheSelectedCurrentReviewer(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap, Priority: 4,
		Title: "Restore posting", Summary: "Posting is unavailable.", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityReviewer: {
			Principal:           authority.Principal{ID: "reviewer-maker", DisplayName: "Review lead"},
			CandidatePrincipals: []authority.Principal{{ID: "reviewer-1", DisplayName: "Ada Okafor"}},
		},
	}}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "reviewer-maker", "bank-ng"),
		Continuity: service, Authority: resolver, CommandGuard: guard,
	})
	body := `{"expected_version":1,"expected_outcome":"Posting remains available.","baseline":{"description":"Posting is unavailable."},"scope":{"description":"Retail current accounts.","measurement_method":"Review the daily posting availability report."},"threshold":{"success_condition":"99.9% of postings succeed."},"observation_period_minutes":1440,"reviewer_candidate_id":"reviewer-1","authority_principal_id":"forged-reviewer","failure_response":"REOPEN"}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/verification-contracts", strings.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("outcome definition returned %d: %s", response.Code, response.Body.String())
	}
	var updated continuity.MatterAggregate
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if len(updated.VerificationContracts) != 1 || updated.VerificationContracts[0].AuthorityPrincipalID != "reviewer-1" {
		t.Fatalf("stored outcome reviewer was not selected from the current route: %#v", updated.VerificationContracts)
	}
}

func TestMatterOutcomeDefinitionBindsSelectedReviewerFromCurrentRoute(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap, Priority: 4,
		Title: "Restore posting", Summary: "Posting is unavailable.", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityReviewer: {
			Principal:           authority.Principal{ID: "reviewer-maker", DisplayName: "Review lead"},
			CandidatePrincipals: []authority.Principal{{ID: "reviewer-1", DisplayName: "Ada Okafor"}},
		},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	request := outcomeDefinitionRequest(t, matter.Matter.ID, "reviewer-maker")
	payload := map[string]any{"reviewer_candidate_id": "reviewer-1", "authority_principal_id": "forged-reviewer"}

	_, err = api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.outcome.define", payload, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3})
	if err != nil {
		t.Fatal(err)
	}
	if payload["authority_principal_id"] != "reviewer-1" {
		t.Fatalf("selected current reviewer was not server-bound: %#v", payload)
	}
}

func TestMatterOutcomeDefinitionRejectsReviewerOutsideCurrentRoute(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap, Priority: 4,
		Title: "Restore posting", Summary: "Posting is unavailable.", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityReviewer: {Principal: authority.Principal{ID: "reviewer-maker", DisplayName: "Review lead"}},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	request := outcomeDefinitionRequest(t, matter.Matter.ID, "reviewer-maker")
	payload := map[string]any{"reviewer_candidate_id": "forged-reviewer", "authority_principal_id": "forged-reviewer"}

	_, err = api.lifecycleCommandPolicy(request.Context(), request, "bank", "matter.outcome.define", payload, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3})
	if err == nil || !strings.Contains(err.Error(), "selected person is not eligible") {
		t.Fatalf("reviewer outside the current route was not rejected: %v", err)
	}
}

func outcomeDefinitionRequest(t *testing.T, matterID, actorID string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matterID+"/verification-contracts", nil)
	request.SetPathValue("id", matterID)
	return request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: actorID, LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour),
	}))
}
