package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestMaterialHandlerReturnsReceiptWhenWriteCommittedBeforeResponseFailure(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank-demo", Type: continuity.MatterControlGap, Priority: 3,
		Title: "Commit truth", Summary: "Verify post-commit response handling", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	api := &API{deps: Dependencies{Continuity: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/actions", strings.NewReader(`{}`))
	request.SetPathValue("id", matter.Matter.ID)
	response := httptest.NewRecorder()
	payload := map[string]any{"tenant_id": "bank-demo", "expected_version": float64(matter.Matter.Version)}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner}

	api.executeMaterialHandler(response, request, policy, payload, func(w http.ResponseWriter, _ *http.Request) {
		if _, err := service.AddAction(t.Context(), continuity.AddActionInput{
			TenantID: "bank-demo", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
			Title: "Committed action", Description: "The authoritative write succeeds first.",
		}); err != nil {
			t.Fatal(err)
		}
		http.Error(w, "simulated response reconstruction failure", http.StatusInternalServerError)
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected committed receipt, got %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-ClearSight-Command-Outcome") != "committed-response-degraded" {
		t.Fatalf("missing committed outcome header: %#v", response.Header())
	}
	var receipt committedCommandReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "COMMITTED" || receipt.AggregateID != matter.Matter.ID || receipt.Version != matter.Matter.Version+1 || !receipt.ResponseDegraded {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
}

func TestMaterialHandlerPreservesFailureWhenNoWriteCommitted(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank-demo", Type: continuity.MatterControlGap, Priority: 3,
		Title: "No commit", Summary: "Verify genuine failure handling", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	api := &API{deps: Dependencies{Continuity: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/actions", strings.NewReader(`{}`))
	request.SetPathValue("id", matter.Matter.ID)
	response := httptest.NewRecorder()
	payload := map[string]any{"tenant_id": "bank-demo", "expected_version": float64(matter.Matter.Version)}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner}

	api.executeMaterialHandler(response, request, policy, payload, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "genuine failure", http.StatusInternalServerError)
	})

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected genuine failure, got %d: %s", response.Code, response.Body.String())
	}
}

func TestMaterialHandlerReturnsReceiptForCommittedVendorRelationship(t *testing.T) {
	repo := thirdparty.NewMemoryAssessmentRepository()
	service := thirdparty.NewService(repo)
	actor := thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	created, err := service.CreateRelationship(t.Context(), actor, thirdparty.CreateRelationshipInput{LegalName: "Acme Processing Limited", ServiceName: "Payment processing", Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor})
	if err != nil {
		t.Fatal(err)
	}

	api := &API{deps: Dependencies{ThirdParty: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+created.Relationship.ID, strings.NewReader(`{}`))
	request.SetPathValue("id", created.Relationship.ID)
	request = request.WithContext(verifiedCommandContext(request.Context(), actor))
	response := httptest.NewRecorder()
	policy := commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner}
	api.executeMaterialHandler(response, request, policy, map[string]any{"expected_version": float64(1)}, func(w http.ResponseWriter, _ *http.Request) {
		_, updateErr := service.UpdateRelationship(t.Context(), actor, created.Relationship.ID, thirdparty.UpdateRelationshipInput{ExpectedVersion: 1, ServiceName: "Payment processing and settlement", Criticality: thirdparty.CriticalityCritical, PrivacyRole: thirdparty.PrivacyProcessor})
		if updateErr != nil {
			t.Fatal(updateErr)
		}
		http.Error(w, "simulated response reconstruction failure", http.StatusInternalServerError)
	})
	assertCommittedReceipt(t, response, "VENDOR_RELATIONSHIP", created.Relationship.ID, 2)
}

func TestMaterialHandlerReturnsReceiptForCommittedVendorAssessment(t *testing.T) {
	repo := thirdparty.NewMemoryAssessmentRepository()
	service := thirdparty.NewAssessmentService(repo, nil)
	actor := thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	relationship, err := thirdparty.NewService(repo).CreateRelationship(t.Context(), actor, thirdparty.CreateRelationshipInput{LegalName: "Acme Processing Limited", ServiceName: "Payment processing", Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor})
	if err != nil {
		t.Fatal(err)
	}
	assessment := thirdparty.Assessment{ID: "assessment-1", TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RelationshipID: relationship.Relationship.ID, ReviewKind: thirdparty.AssessmentReviewOnboarding, StableEpisodeKey: "episode-1", Status: thirdparty.AssessmentSetupPending, FormTemplateID: "form-1", FormTemplateVersion: 1, ReviewDueAt: time.Now().UTC().Add(24 * time.Hour), StartedByPrincipalID: actor.PrincipalID, StartedAt: time.Now().UTC(), Version: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err := repo.CreateAssessment(t.Context(), thirdparty.CreateAssessmentRecord{Scope: thirdparty.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, RelationshipID: assessment.RelationshipID, RelationshipVersion: 1, Assessment: assessment}); err != nil {
		t.Fatal(err)
	}

	api := &API{deps: Dependencies{ThirdPartyAssessments: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vendor-assessments/"+assessment.ID+"/setup", strings.NewReader(`{}`))
	request.SetPathValue("id", assessment.ID)
	request = request.WithContext(verifiedCommandContext(request.Context(), actor))
	response := httptest.NewRecorder()
	policy := commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityOwner}
	api.executeMaterialHandler(response, request, policy, map[string]any{"expected_version": float64(1)}, func(w http.ResponseWriter, _ *http.Request) {
		_, reactionErr := service.RecordAssessmentSetupCompleted(t.Context(), thirdparty.AssessmentSetupCompletedInput{Scope: thirdparty.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, AssessmentID: assessment.ID, ExpectedVersion: 1, CausationID: "setup-event", SetupJobID: "setup-job", ReviewMatterID: "review-matter"})
		if reactionErr != nil {
			t.Fatal(reactionErr)
		}
		http.Error(w, "simulated response reconstruction failure", http.StatusInternalServerError)
	})
	assertCommittedReceipt(t, response, "THIRD_PARTY_ASSESSMENT", assessment.ID, 2)
}

func verifiedCommandContext(ctx context.Context, actor thirdparty.Actor) context.Context {
	now := time.Now().UTC()
	return identity.WithActor(ctx, identity.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)})
}

func assertCommittedReceipt(t *testing.T, response *httptest.ResponseRecorder, aggregateType, aggregateID string, version int64) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected committed receipt, got %d: %s", response.Code, response.Body.String())
	}
	var receipt committedCommandReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.AggregateType != aggregateType || receipt.AggregateID != aggregateID || receipt.Version != version || !receipt.ResponseDegraded {
		t.Fatalf("unexpected receipt %#v", receipt)
	}
}
