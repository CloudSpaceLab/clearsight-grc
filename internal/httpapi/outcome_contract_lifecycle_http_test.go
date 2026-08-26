package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func seededOutcomeContract(t *testing.T) (*continuity.Service, *continuity.MemoryRepository, continuity.MatterAggregate) {
	t.Helper()
	repository := continuity.NewMemoryRepository()
	service := continuity.NewService(repository)
	ctx := continuity.WithTrustedSystemScope(t.Context())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap, Priority: 4,
		Title: "Annual return evidence is incomplete", Summary: "Two return sections need current evidence.",
		Scope: json.RawMessage(`{}`), OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		ExpectedOutcome: "All return sections have approved evidence.", Baseline: json.RawMessage(`{"description":"Two sections are incomplete."}`),
		Scope: json.RawMessage(`{"description":"All ten return sections."}`), Threshold: json.RawMessage(`{"success_condition":"Ten of ten sections pass."}`),
		ObservationPeriodMinutes: 1440, AuthorityPrincipalID: "reviewer-1", FailureResponse: "BLOCK_CLOSE", ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, matter
}

func outcomeLifecycleHandler(service *continuity.Service, principalID string, resolution authority.Resolution) http.Handler {
	return New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", principalID, "bank-ng"),
		Continuity: service, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityReviewer: resolution}},
	})
}

func TestOutcomeContractSupersedeRouteBindsExactContractAndVerifiedReviewer(t *testing.T) {
	service, repository, matter := seededOutcomeContract(t)
	contractID := matter.VerificationContracts[0].ID
	handler := outcomeLifecycleHandler(service, "reviewer-1", authority.Resolution{
		Principal:           authority.Principal{ID: "reviewer-1", DisplayName: "Current reviewer"},
		CandidatePrincipals: []authority.Principal{{ID: "reviewer-2", DisplayName: "Incoming reviewer"}},
	})
	body := []byte(`{"tenant_id":"bank","expected_version":2,"contract_id":"` + contractID + `","expected_outcome":"All return sections have current approved evidence.","baseline":{"description":"One section is incomplete."},"scope":{"description":"All ten return sections."},"threshold":{"success_condition":"Ten of ten sections pass."},"observation_period_minutes":2880,"reviewer_candidate_id":"reviewer-2","authority_principal_id":"forged-authority","actor_id":"forged-actor","failure_response":"REOPEN","rationale":"The evidence population and observation period changed."}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/verification-contracts/"+contractID+"/supersede", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("supersede returned %d: %s", response.Code, response.Body.String())
	}
	var updated continuity.MatterAggregate
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if len(updated.VerificationContracts) != 2 || updated.VerificationContracts[0].Status != continuity.VerificationRetired {
		t.Fatalf("unexpected contract history: %#v", updated.VerificationContracts)
	}
	replacement := updated.VerificationContracts[1]
	if replacement.SupersedesContractID != contractID || replacement.AuthorityPrincipalID != "reviewer-2" || replacement.Status != continuity.VerificationActive {
		t.Fatalf("replacement did not retain governed lineage: %#v", replacement)
	}
	events, err := repository.MatterEvents(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if events[len(events)-1].ActorID != "reviewer-1" {
		t.Fatalf("command trusted a body actor: %#v", events[len(events)-1])
	}
}

func TestOutcomeContractRoutesRejectConflictingContractIDAndIneligibleReviewer(t *testing.T) {
	service, _, matter := seededOutcomeContract(t)
	contractID := matter.VerificationContracts[0].ID
	handler := outcomeLifecycleHandler(service, "reviewer-1", authority.Resolution{Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Current reviewer"}})
	tests := []struct {
		name string
		body string
	}{
		{"conflicting contract", `{"tenant_id":"bank","expected_version":2,"contract_id":"another-contract","expected_outcome":"Current evidence is complete.","baseline":{},"scope":{},"threshold":{},"observation_period_minutes":0,"reviewer_candidate_id":"reviewer-1","failure_response":"BLOCK_CLOSE","rationale":"Correct the contract."}`},
		{"ineligible reviewer", `{"tenant_id":"bank","expected_version":2,"expected_outcome":"Current evidence is complete.","baseline":{},"scope":{},"threshold":{},"observation_period_minutes":0,"reviewer_candidate_id":"forged-reviewer","failure_response":"BLOCK_CLOSE","rationale":"Correct the contract."}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/verification-contracts/"+contractID+"/supersede", bytes.NewBufferString(test.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422, got %d: %s", response.Code, response.Body.String())
			}
			current, err := service.GetMatter(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID)
			if err != nil {
				t.Fatal(err)
			}
			if current.Matter.Version != matter.Matter.Version || len(current.VerificationContracts) != 1 {
				t.Fatalf("rejected command mutated the issue: %#v", current)
			}
		})
	}
}

func TestOutcomeContractRetireRouteUsesVerifiedAssignedReviewer(t *testing.T) {
	service, repository, matter := seededOutcomeContract(t)
	contractID := matter.VerificationContracts[0].ID
	handler := outcomeLifecycleHandler(service, "reviewer-1", authority.Resolution{Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Current reviewer"}})
	body := []byte(`{"tenant_id":"bank","expected_version":2,"contract_id":"` + contractID + `","actor_id":"forged-actor","rationale":"This outcome is no longer required because the linked action was cancelled."}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/verification-contracts/"+contractID+"/retire", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("retire returned %d: %s", response.Code, response.Body.String())
	}
	var updated continuity.MatterAggregate
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.VerificationContracts[0].Status != continuity.VerificationRetired {
		t.Fatalf("contract was not ended: %#v", updated.VerificationContracts[0])
	}
	events, err := repository.MatterEvents(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if events[len(events)-1].ActorID != "reviewer-1" {
		t.Fatalf("retirement trusted a body actor: %#v", events[len(events)-1])
	}
}

func TestOutcomeContractLifecycleRejectsReviewerWhoDoesNotHoldStoredAssignment(t *testing.T) {
	service, _, matter := seededOutcomeContract(t)
	contractID := matter.VerificationContracts[0].ID
	handler := outcomeLifecycleHandler(service, "reviewer-2", authority.Resolution{Principal: authority.Principal{ID: "reviewer-2", DisplayName: "Different reviewer"}})
	body := []byte(`{"tenant_id":"bank","expected_version":2,"rationale":"End this outcome check."}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/verification-contracts/"+contractID+"/retire", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected assigned-reviewer rejection, got %d: %s", response.Code, response.Body.String())
	}
	current, err := service.GetMatter(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Matter.Version != matter.Matter.Version || current.VerificationContracts[0].Status != continuity.VerificationActive {
		t.Fatalf("unauthorized retirement mutated the issue: %#v", current)
	}
}

func TestOutcomeContractOperationsShowAssignedReviewerAndLifecycleActions(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{
		Matter:                continuity.Matter{ID: "matter-1", TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap, Status: continuity.MatterVerification, Priority: 4, OwnerPrincipalID: "owner-1", Version: 2},
		VerificationContracts: []continuity.VerificationContract{{ID: "contract-1", MatterID: "matter-1", ExpectedOutcome: "All return sections pass.", AuthorityPrincipalID: "reviewer-1", Status: continuity.VerificationActive, Version: 1}},
	}
	api := &API{deps: Dependencies{Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:    {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program Owner"}},
		authority.ResponsibilityReviewer: {Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Internal Auditor"}, CandidatePrincipals: []authority.Principal{{ID: "reviewer-2", DisplayName: "Assurance Reviewer"}}},
	}}}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "reviewer-1", Kind: "PERSON"}
	value := api.buildMatterOperations(continuity.WithTrustedSystemScope(t.Context()), actor, aggregate, now)
	for _, command := range []string{"matter.outcome.supersede", "matter.outcome.retire"} {
		var found *RecordOperation
		for index := range value.Operations {
			operation := &value.Operations[index]
			if operation.Command == command && operation.SubresourceID == "contract-1" {
				found = operation
				break
			}
		}
		if found == nil || !found.CanAct || found.AssignedTo == nil || found.AssignedTo.DisplayName != "Internal Auditor" {
			t.Fatalf("%s operation = %#v", command, found)
		}
	}
}
