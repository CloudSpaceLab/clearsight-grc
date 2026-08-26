package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type exactFormAuthority struct {
	delegateFormID string
	inputs         []authority.ResolveInput
}

func (s *exactFormAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	s.inputs = append(s.inputs, input)
	principalID := "program-owner"
	candidates := []authority.Principal{{ID: principalID, DisplayName: "Program owner"}}
	origins := []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}}
	if input.Responsibility == authority.ResponsibilityReviewer {
		principalID = "program-reviewer"
		if input.ObjectType == "FORM_TEMPLATE" {
			principalID = "reviewer-" + input.ObjectID
		}
		candidates = []authority.Principal{{ID: principalID, DisplayName: principalID}}
		origins = []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}}
		if input.ObjectType == "FORM_TEMPLATE" && input.ObjectID == s.delegateFormID {
			candidates = append(candidates, authority.Principal{ID: "reviewer-delegate", DisplayName: "Acting form reviewer"})
			origins = append(origins, authority.EffectiveOrigin{PrincipalID: "reviewer-delegate", OriginPrincipalID: principalID})
		}
	}
	return authority.Resolution{
		Principal:           authority.Principal{ID: principalID, DisplayName: principalID},
		CandidatePrincipals: candidates,
		EffectiveOrigins:    origins,
	}, nil
}

func (s *exactFormAuthority) ResolveMany(ctx context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	outcomes := make([]authority.ResolveOutcome, len(inputs))
	for index, input := range inputs {
		resolution, err := s.Resolve(ctx, input)
		outcomes[index] = authority.ResolveOutcome{Resolution: resolution, Err: err}
	}
	return outcomes, nil
}

func (*exactFormAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (*exactFormAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (*exactFormAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func pendingMonitoringForm(t *testing.T, repository *monitoring.MemoryRepository, programID, id, submittedBy string, now time.Time) monitoring.FormTemplate {
	t.Helper()
	form, err := repository.CreateFormRevision(t.Context(), monitoring.FormTemplate{
		ID: id, TenantID: "bank", LegalEntityID: "entity-a", ProgramID: programID,
		Code: strings.ToUpper(id), Name: "Review " + id, Purpose: "Confirm that the control operated.",
		Fields: []monitoring.TemplateField{{ID: "operated", Label: "Did the control operate?", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle: monitoring.Lifecycle{
			Status: monitoring.LifecyclePendingApproval, IsCurrent: true, Version: 2,
			CreatedBy: submittedBy, SubmittedBy: submittedBy, CreatedAt: now, UpdatedAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return form
}

func monitoringFormAuthorityFixture(t *testing.T) (*continuity.Service, continuity.ProgramAggregate, *monitoring.MemoryRepository, *monitoring.Service, monitoring.FormTemplate, monitoring.FormTemplate, *exactFormAuthority) {
	t.Helper()
	now := time.Now().UTC()
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer", EffectiveFrom: now, ActorID: "program-owner", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	repository := monitoring.NewMemoryRepository()
	formA := pendingMonitoringForm(t, repository, program.Program.ID, "form-a", "maker-a", now)
	formB := pendingMonitoringForm(t, repository, program.Program.ID, "form-b", "maker-b", now)
	resolver := &exactFormAuthority{delegateFormID: formA.ID}
	return continuityService, program, repository, monitoring.NewService(repository, nil), formA, formB, resolver
}

func TestMonitoringFormTransitionUsesExactFormAuthorityAndVerifiedActor(t *testing.T) {
	continuityService, program, repository, monitoringService, formA, formB, resolver := monitoringFormAuthorityFixture(t)
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler := func(principalID string) http.Handler {
		return New(Dependencies{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", principalID, "entity-a"),
			Continuity: continuityService, Monitoring: monitoringService, Authority: resolver, CommandGuard: guard,
		})
	}
	transition := func(principalID string, form monitoring.FormTemplate, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		path := fmt.Sprintf("/api/v1/programs/%s/form-templates/%s/transition", program.Program.ID, form.ID)
		handler(principalID).ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body)))
		return response
	}

	delegateBody := `{"tenant_id":"bank","actor_id":"program-reviewer","expected_version":2,"to":"ACTIVE"}`
	if response := transition("reviewer-delegate", formA, delegateBody); response.Code != http.StatusOK {
		t.Fatalf("exact form reviewer delegate returned %d: %s", response.Code, response.Body.String())
	}
	if active, err := repository.FormRevision(t.Context(), "bank", "entity-a", program.Program.ID, formA.ID, 3); err != nil || active.ApprovedBy != "reviewer-delegate" || active.Status != monitoring.LifecycleActive {
		t.Fatalf("delegated form approval = %#v, err %v", active, err)
	}

	forgedTenantBody := `{"tenant_id":"another-bank","actor_id":"reviewer-form-b","expected_version":2,"to":"ACTIVE"}`
	if response := transition("program-reviewer", formB, forgedTenantBody); response.Code != http.StatusForbidden {
		t.Fatalf("forged tenant transition returned %d: %s", response.Code, response.Body.String())
	}
	broadReviewerBody := `{"tenant_id":"bank","actor_id":"reviewer-form-b","expected_version":2,"to":"ACTIVE"}`
	if response := transition("program-reviewer", formB, broadReviewerBody); response.Code != http.StatusForbidden {
		t.Fatalf("Program-wide reviewer substituted for exact form reviewer: %d %s", response.Code, response.Body.String())
	}
	if response := transition("reviewer-delegate", formB, broadReviewerBody); response.Code != http.StatusForbidden {
		t.Fatalf("delegate for form A acted on form B: %d %s", response.Code, response.Body.String())
	}
	if response := transition("reviewer-form-b", formB, broadReviewerBody); response.Code != http.StatusOK {
		t.Fatalf("exact form B reviewer returned %d: %s", response.Code, response.Body.String())
	}

	for _, input := range resolver.inputs {
		if input.DecisionType != "program.monitoring.form.transition" {
			continue
		}
		if input.ObjectType != "FORM_TEMPLATE" || input.ObjectID == "" || input.ObjectID == program.Program.ID || input.LegalEntityID != program.Program.LegalEntityID {
			t.Fatalf("form transition authority was not exact: %#v", input)
		}
	}
}

func TestProgramOperationsUseExactFormAuthority(t *testing.T) {
	_, program, _, monitoringService, formA, formB, resolver := monitoringFormAuthorityFixture(t)
	api := &API{deps: Dependencies{Authority: resolver, Monitoring: monitoringService}}
	find := func(operations []RecordOperation, formID string) RecordOperation {
		t.Helper()
		for _, operation := range operations {
			if operation.Command == "program.monitoring.form.transition" && operation.SubresourceID == formID {
				return operation
			}
		}
		t.Fatalf("form transition operation %s missing: %#v", formID, operations)
		return RecordOperation{}
	}
	load := func(principalID string) programOperationsResponse {
		actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: principalID, Kind: "PERSON"}
		return api.buildProgramOperations(identity.WithActor(t.Context(), actor), actor, program, time.Now().UTC())
	}

	delegate := load("reviewer-delegate")
	if operation := find(delegate.Operations, formA.ID); !operation.CanAct {
		t.Fatalf("exact form delegate operation = %#v", operation)
	}
	if operation := find(delegate.Operations, formB.ID); operation.CanAct {
		t.Fatalf("form A delegate received form B operation = %#v", operation)
	}
	broad := load("program-reviewer")
	for _, formID := range []string{formA.ID, formB.ID} {
		if operation := find(broad.Operations, formID); operation.CanAct {
			t.Fatalf("Program-wide reviewer received exact form operation %s: %#v", formID, operation)
		}
	}

	found := map[string]bool{}
	for _, input := range resolver.inputs {
		if input.DecisionType != "program.monitoring.form.transition" {
			continue
		}
		if input.ObjectType != "FORM_TEMPLATE" {
			t.Fatalf("form operation authority used %s/%s: %#v", input.ObjectType, input.ObjectID, input)
		}
		found[input.ObjectID] = true
	}
	for _, formID := range []string{formA.ID, formB.ID} {
		if !found[formID] {
			t.Fatalf("exact form authority input %s was not resolved: %#v", formID, resolver.inputs)
		}
	}
}
