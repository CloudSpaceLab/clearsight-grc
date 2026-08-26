package httpapi

import (
	"errors"
	"fmt"
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

func TestOutcomeEscalationBindsCurrentOwnerAndReviewerDelegateLineage(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(t.Context())
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterAuditFinding, Priority: 4,
		Title: "Resolve access exceptions", Summary: "Confirm that unsupported access has been removed.",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []continuity.MatterStatus{continuity.MatterInitialReview, continuity.MatterAssessment, continuity.MatterActionsInProgress} {
		matter, err = service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: "bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: target, Rationale: "Progress corrective work."})
		if err != nil {
			t.Fatal(err)
		}
	}
	matter, err = service.AddAction(ctx, continuity.AddActionInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: "Remove unsupported access", Description: "Remove the unsupported access entry.", OwnerPrincipalID: "performer"})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID
	for _, target := range []continuity.ActionStatus{continuity.ActionInProgress, continuity.ActionImplemented} {
		matter, err = service.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: "bank", MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version, To: target, Rationale: "Corrective work progressed."})
		if err != nil {
			t.Fatal(err)
		}
	}
	matter, err = service.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: actionID,
		ExpectedOutcome: "No unsupported access remains.", AuthorityPrincipalID: "reviewer-origin", FailureResponse: "ESCALATE",
	})
	if err != nil {
		t.Fatal(err)
	}
	contractID := matter.VerificationContracts[0].ID
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityReviewer: {
			Principal:           authority.Principal{ID: "reviewer-origin", DisplayName: "Internal Audit Lead"},
			CandidatePrincipals: []authority.Principal{{ID: "reviewer-origin"}, {ID: "reviewer-delegate"}, {ID: "unrelated-reviewer"}},
			EffectiveOrigins: []authority.EffectiveOrigin{
				{PrincipalID: "reviewer-origin", OriginPrincipalID: "reviewer-origin"},
				{PrincipalID: "reviewer-delegate", OriginPrincipalID: "reviewer-origin"},
				{PrincipalID: "unrelated-reviewer", OriginPrincipalID: "unrelated-reviewer"},
			},
		},
		authority.ResponsibilityEscalation: {Principal: authority.Principal{ID: "escalation-owner", DisplayName: "Operational Risk Director", Kind: "PERSON"}},
	}}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver}}
	payload := map[string]any{
		"contract_id": contractID, "result": "FAIL",
		"reviewer_authority_principal_id": "spoofed-reviewer", "escalation_principal_id": "spoofed-owner",
	}
	request := lifecycleRequest(matter.Matter.ID)
	delegate := identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-delegate", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	request = request.WithContext(delegate)

	if _, err := api.lifecycleCommandPolicy(delegate, request, "bank", "matter.outcome.record", payload, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 4}); err != nil {
		t.Fatalf("valid reviewer delegate was rejected: %v", err)
	}
	if payload["reviewer_authority_principal_id"] != "reviewer-origin" || payload["escalation_principal_id"] != "escalation-owner" {
		t.Fatalf("server did not overwrite protected responsibility fields: %#v", payload)
	}

	unrelated := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "unrelated-reviewer", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)})
	if _, err := api.lifecycleCommandPolicy(unrelated, lifecycleRequest(matter.Matter.ID), "bank", "matter.outcome.record", map[string]any{"contract_id": contractID, "result": "FAIL"}, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 4}); !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("unrelated reviewer candidate was not rejected: %v", err)
	}

	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api.deps.CommandGuard = guard
	handler := api.command("matter.outcome.record", commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 4, ActorField: "reviewer_principal_id"}, api.recordMatterVerificationResult)
	body := fmt.Sprintf(`{"expected_version":%d,"contract_id":%q,"result":"FAIL","observations":{"unsupported":1},"evidence_references":[],"rationale":"One unsupported entry remains.","reviewer_principal_id":"spoofed-reviewer","reviewer_authority_principal_id":"spoofed-reviewer","escalation_principal_id":"spoofed-owner"}`, matter.Matter.Version, contractID)
	commandRequest := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/verification-results", strings.NewReader(body))
	commandRequest.SetPathValue("id", matter.Matter.ID)
	commandRequest = commandRequest.WithContext(identity.WithActor(commandRequest.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-delegate", Kind: "PERSON", ExpiresAt: time.Now().UTC().Add(time.Hour)}))
	response := httptest.NewRecorder()
	handler(response, commandRequest)
	if response.Code != http.StatusCreated {
		t.Fatalf("delegated reviewer command failed: %d %s", response.Code, response.Body.String())
	}
	persisted, err := service.GetMatter(ctx, "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.Actions) != 2 || persisted.Actions[1].OwnerPrincipalID != "escalation-owner" || persisted.Actions[1].RequiredResponsibility != "ESCALATION_OWNER" {
		t.Fatalf("command did not persist an executable escalation Action: %#v", persisted.Actions)
	}
	if persisted.VerificationResults[0].ReviewerPrincipalID != "reviewer-delegate" || persisted.VerificationResults[0].ReviewerAuthorityPrincipalID != "reviewer-origin" {
		t.Fatalf("reviewer actor and delegated authority lineage were not preserved: %#v", persisted.VerificationResults[0])
	}
}
