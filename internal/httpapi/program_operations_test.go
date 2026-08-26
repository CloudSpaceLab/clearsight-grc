package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestProgramOperationsExplainCurrentResponsibilitiesAcrossRoles(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Data Protection Office", OwnerPrincipalID: "owner-1",
		EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(continuity.WithTrustedSystemScope(t.Context()), continuity.AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "CAR-01", Title: "File the annual return", Statement: "The bank must file its annual compliance return.",
		SourceAnchor: "GAID 2025, section 7", EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	requirementID := program.Requirements[0].ID
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner: {
			Principal:           authority.Principal{ID: "owner-1", DisplayName: "Data Protection Officer"},
			CandidatePrincipals: []authority.Principal{{ID: "owner-2", DisplayName: "Deputy Data Protection Officer"}},
		},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "cro", DisplayName: "Chief Risk Officer"}},
		authority.ResponsibilityReviewer:   {Principal: authority.Principal{ID: "auditor", DisplayName: "Internal Audit reviewer"}},
		authority.ResponsibilityPerformer:  {Principal: authority.Principal{ID: "control-owner", DisplayName: "Privacy control owner"}},
	}}

	load := func(principal string) programOperationsResponse {
		t.Helper()
		handler := New(Dependencies{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", principal, "bank-ng"),
			Continuity: service, Authority: resolver,
		})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/programs/"+program.Program.ID+"/operations?tenant_id=bank", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("operations for %s returned %d: %s", principal, response.Code, response.Body.String())
		}
		var payload programOperationsResponse
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	find := func(payload programOperationsResponse, command, subresource string) RecordOperation {
		t.Helper()
		for _, operation := range payload.Operations {
			if operation.Command == command && operation.SubresourceID == subresource {
				return operation
			}
		}
		t.Fatalf("operation %s/%s not returned: %#v", command, subresource, payload.Operations)
		return RecordOperation{}
	}

	owner := load("owner-1")
	authorizer := load("cro")
	reviewer := load("auditor")
	if owner.ProgramID != program.Program.ID || owner.ProgramVersion != program.Program.Version || !owner.AuthorityAvailable {
		t.Fatalf("Program operation envelope is incorrect: %#v", owner)
	}
	details := find(owner, "program.details.update", "")
	assignment := find(owner, "program.assign", "")
	supersession := find(owner, "program.requirement.supersede", requirementID)
	transition := find(authorizer, "program.transition", "")
	assessment := find(reviewer, "program.evidence.assess", "")
	safeguard := find(owner, "program.safeguard.define", "")
	if !details.CanAct || details.AssignedTo == nil || details.AssignedTo.DisplayName != "Data Protection Officer" {
		t.Fatalf("owner responsibility is unexplained: %#v", details)
	}
	if !assignment.CanAct || len(assignment.Candidates) != 2 || assignment.Candidates[1].DisplayName != "Deputy Data Protection Officer" {
		t.Fatalf("ownership candidates are unexplained: %#v", assignment)
	}
	if !supersession.CanAct {
		t.Fatalf("requirement supersession is not available to the owner: %#v", supersession)
	}
	if !transition.CanAct || !reflect.DeepEqual(transition.AllowedTargets, []string{"ACTIVE", "RETIRED"}) {
		t.Fatalf("authorizer lifecycle operation is incorrect: %#v", transition)
	}
	if !assessment.CanAct || assessment.AssignedTo == nil || assessment.AssignedTo.DisplayName != "Internal Audit reviewer" {
		t.Fatalf("reviewer operation is unexplained: %#v", assessment)
	}
	if !safeguard.CanAct || len(safeguard.Candidates) != 1 || safeguard.Candidates[0].DisplayName != "Privacy control owner" {
		t.Fatalf("safeguard owner candidates are unexplained: %#v", safeguard)
	}
	if find(authorizer, "program.details.update", "").CanAct || find(owner, "program.applicability.decide", "").CanAct {
		t.Fatal("operations were granted outside the current responsibility route")
	}
}

func TestProgramOperationsFailClosedWhenAuthorityRouteIsUnavailable(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "AML", Name: "Financial crime", Type: "AML", OwningFunction: "Compliance",
		OwnerPrincipalID: "owner-1", EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"),
		Continuity: service, Authority: &assignmentAuthorityStub{err: authority.ErrNoRoute},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/programs/"+program.Program.ID+"/operations?tenant_id=bank", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("operations returned %d: %s", response.Code, response.Body.String())
	}
	var payload programOperationsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	for _, operation := range payload.Operations {
		if operation.CanAct || operation.Reason == "" {
			t.Fatalf("unavailable route invented an executable operation: %#v", operation)
		}
	}
}
