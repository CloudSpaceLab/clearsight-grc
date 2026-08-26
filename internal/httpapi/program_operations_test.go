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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type capturingProgramAuthority struct {
	*assignmentAuthorityStub
	legalEntities []string
}

func (s *capturingProgramAuthority) Resolve(ctx context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	s.legalEntities = append(s.legalEntities, input.LegalEntityID)
	return s.assignmentAuthorityStub.Resolve(ctx, input)
}

func TestProgramSetupCandidatesAndCreationKeepOwnerAndApprovalAuthorityDistinct(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner: {
			Principal: authority.Principal{ID: "owner-1", DisplayName: "Data Protection Officer", Kind: "PERSON", Role: "DPO"},
		},
		authority.ResponsibilityAuthorizer: {
			Principal:           authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer", Kind: "PERSON", Role: "CRO"},
			CandidatePrincipals: []authority.Principal{{ID: "deputy-cro", DisplayName: "Deputy Chief Risk Officer", Kind: "PERSON", Role: "Deputy CRO"}},
		},
	}}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"),
		Continuity: service, Authority: resolver,
	})

	candidates := httptest.NewRecorder()
	handler.ServeHTTP(candidates, httptest.NewRequest(http.MethodGet, "/api/v1/programs/setup-candidates?tenant_id=bank", nil))
	if candidates.Code != http.StatusOK || !bytes.Contains(candidates.Body.Bytes(), []byte(`"owner_candidates"`)) || !bytes.Contains(candidates.Body.Bytes(), []byte(`"approval_authority_candidates"`)) {
		t.Fatalf("setup candidates returned %d: %s", candidates.Code, candidates.Body.String())
	}

	created := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"tenant_id":"bank","legal_entity_id":"forged-entity","code":"NDPA","name":"Data protection","type":"PRIVACY","owning_function":"Privacy","owner_candidate_id":"owner-1","approval_authority_candidate_id":"cro-1","owner_principal_id":"forged-owner","authority_principal_id":"forged-approver","scope":{},"effective_from":"2026-08-26T00:00:00Z"}`)
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/programs", body))
	if created.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", created.Code, created.Body.String())
	}
	var aggregate continuity.ProgramAggregate
	if err := json.NewDecoder(created.Body).Decode(&aggregate); err != nil {
		t.Fatal(err)
	}
	if aggregate.Program.LegalEntityID != "bank-ng" || aggregate.Program.OwnerPrincipalID != "owner-1" || aggregate.Program.AuthorityPrincipalID != "cro-1" {
		t.Fatalf("create trusted unverified scope or principals: %#v", aggregate.Program)
	}
	if aggregate.Program.OwnerPrincipalID == aggregate.Program.AuthorityPrincipalID {
		t.Fatal("Program owner and approval authority were collapsed")
	}
}

func TestProgramSetupCandidatesRequireAnExactEntityForWildcardActors(t *testing.T) {
	resolver := &capturingProgramAuthority{assignmentAuthorityStub: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program owner"}},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "cro-1", DisplayName: "Approval authority"}},
	}}}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "*"),
		Continuity: continuity.NewService(continuity.NewMemoryRepository()), Authority: resolver,
	})

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/programs/setup-candidates?tenant_id=bank", nil))
	if missing.Code != http.StatusForbidden {
		t.Fatalf("wildcard candidate scope without an exact entity returned %d: %s", missing.Code, missing.Body.String())
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/programs/setup-candidates?tenant_id=bank&scope_legal_entity_id=bank-ng", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("exact entity candidate scope returned %d: %s", response.Code, response.Body.String())
	}
	for _, entity := range resolver.legalEntities {
		if entity != "bank-ng" {
			t.Fatalf("authority resolved outside the selected entity: %#v", resolver.legalEntities)
		}
	}
}

func TestProgramCreationRejectsIneligibleOrConflictedApprovalAuthority(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program owner"}},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer"}},
	}}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"),
		Continuity: service, Authority: resolver,
	})

	for name, approvalID := range map[string]string{"unrouted": "forged-approver", "same as owner": "owner-1"} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			body := bytes.NewBufferString(`{"tenant_id":"bank","code":"TEST","name":"Test Program","type":"PRIVACY","owning_function":"Privacy","owner_candidate_id":"owner-1","approval_authority_candidate_id":"` + approvalID + `","scope":{},"effective_from":"2026-08-26T00:00:00Z"}`)
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs", body))
			if response.Code != http.StatusConflict && response.Code != http.StatusBadRequest && response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected rejected assignment, got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestProgramApprovalAuthorityMaintenanceUsesCurrentRouteAndIgnoresForgedAuthorityField(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "NDPA", Name: "Data protection", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro-1", Scope: json.RawMessage(`{}`), EffectiveFrom: time.Now().UTC(), ActorID: "maker-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityAuthorizer: {
			Principal:           authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer"},
			CandidatePrincipals: []authority.Principal{{ID: "deputy-cro", DisplayName: "Deputy Chief Risk Officer"}},
		},
	}}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "cro-1", "bank-ng"),
		Continuity: service, Authority: resolver,
	})
	body := bytes.NewBufferString(fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"candidate_id":"deputy-cro","authority_principal_id":"forged-approver","actor_id":"forged-actor","rationale":"The delegated CRO position now holds this approval."}`, program.Program.Version))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/approval-authority", body))
	if response.Code != http.StatusOK {
		t.Fatalf("approval authority change returned %d: %s", response.Code, response.Body.String())
	}
	current, err := service.GetProgram(continuity.WithTrustedSystemScope(t.Context()), "bank", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Program.AuthorityPrincipalID != "deputy-cro" || current.Program.OwnerPrincipalID != "owner-1" {
		t.Fatalf("approval authority was not changed independently: %#v", current.Program)
	}
}

func TestProgramOperationsExplainCurrentResponsibilitiesAcrossRoles(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Data Protection Office", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro",
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

func TestProgramOperationsBindWildcardViewerToRecordEntityAndStoredAuthority(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.ProgramAggregate{Program: continuity.Program{
		ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "NDPA", Name: "Data protection",
		Status: continuity.ProgramDraft, OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro-1", Version: 4,
		CreatedAt: now, UpdatedAt: now,
	}}
	resolver := &capturingProgramAuthority{assignmentAuthorityStub: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program owner"}},
		authority.ResponsibilityAuthorizer: {
			Principal:           authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer"},
			CandidatePrincipals: []authority.Principal{{ID: "deputy-cro", DisplayName: "Deputy Chief Risk Officer"}},
		},
	}}}
	api := &API{deps: Dependencies{Authority: resolver}}

	viewer := identity.Actor{TenantID: "bank", PrincipalID: "deputy-cro", LegalEntityID: "*", Kind: "PERSON"}
	payload := api.buildProgramOperations(t.Context(), viewer, aggregate, now)
	for _, entity := range resolver.legalEntities {
		if entity != "entity-a" {
			t.Fatalf("Program authority resolved outside the record entity: %#v", resolver.legalEntities)
		}
	}
	for _, operation := range payload.Operations {
		if operation.Command != "program.transition" && operation.Command != "program.applicability.decide" {
			continue
		}
		if operation.CanAct {
			t.Fatalf("eligible but unassigned authorizer can act: %#v", operation)
		}
		if operation.AssignedTo == nil || operation.AssignedTo.ID != "cro-1" || operation.AssignedTo.DisplayName != "Chief Risk Officer" {
			t.Fatalf("stored Program authority was not shown: %#v", operation)
		}
	}
}

func TestProgramOperationsAllowDelegateActingForStoredAuthority(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.ProgramAggregate{Program: continuity.Program{
		ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "NDPA", Name: "Data protection",
		Status: continuity.ProgramDraft, OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "cro-1", Version: 4,
		CreatedAt: now, UpdatedAt: now,
	}}
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal:           authority.Principal{ID: "cro-1", DisplayName: "Chief Risk Officer"},
		CandidatePrincipals: []authority.Principal{{ID: "delegate-1", DisplayName: "Acting Chief Risk Officer"}},
		EffectiveOrigins:    []authority.EffectiveOrigin{{PrincipalID: "delegate-1", OriginPrincipalID: "cro-1"}},
	}}
	api := &API{deps: Dependencies{Authority: resolver}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "delegate-1", LegalEntityID: "entity-a", Kind: "PERSON"}

	payload := api.buildProgramOperations(t.Context(), actor, aggregate, now)
	for _, operation := range payload.Operations {
		if operation.Command != "program.transition" && operation.Command != "program.applicability.decide" {
			continue
		}
		if !operation.CanAct || operation.AssignedTo == nil || operation.AssignedTo.ID != "cro-1" {
			t.Fatalf("delegated Program operation did not retain stored authority: %#v", operation)
		}
	}
}

func TestProgramOperationsKeepRetiredOwnerReadableWithoutCommandsOrPrincipalID(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.ProgramAggregate{Program: continuity.Program{
		ID: "program-retired", TenantID: "bank", LegalEntityID: "bank-ng", Code: "OLD-PRIVACY",
		Name: "Retired privacy controls", Status: continuity.ProgramRetired,
		OwnerPrincipalID: "stored-retired-owner", AuthorityPrincipalID: "stored-retired-authorizer", CreatedAt: now, UpdatedAt: now, Version: 9,
	}}
	api := &API{deps: Dependencies{Access: principalResolverStub{values: map[string]access.Resolution{
		"stored-retired-owner":      {TenantID: "bank", PrincipalID: "stored-retired-owner", LegalEntityID: "bank-ng", DisplayName: "Former Data Protection Officer", Kind: "PERSON"},
		"stored-retired-authorizer": {TenantID: "bank", PrincipalID: "stored-retired-authorizer", LegalEntityID: "bank-ng", DisplayName: "Chief Risk Officer", Kind: "PERSON"},
	}}}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "auditor", LegalEntityID: "bank-ng", Kind: "PERSON"}

	payload := api.buildProgramOperations(continuity.WithTrustedSystemScope(t.Context()), actor, aggregate, now)
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Operations         []RecordOperation `json:"operations"`
		ResponsibleParties []struct {
			Scope          string `json:"scope"`
			Responsibility string `json:"responsibility"`
			DisplayName    string `json:"display_name"`
		} `json:"responsible_parties"`
	}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Operations) != 0 {
		t.Fatalf("retired Program exposed commands: %#v", response.Operations)
	}
	got := map[string]string{}
	for _, party := range response.ResponsibleParties {
		got[party.Scope+":"+party.Responsibility] = party.DisplayName
	}
	want := map[string]string{
		"RECORD:ACCOUNTABLE_OWNER": "Former Data Protection Officer",
		"RECORD:AUTHORIZER":        "Chief Risk Officer",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("retired Program responsibilities = %#v, want %#v", got, want)
	}
	for _, principalID := range []string{"stored-retired-owner", "stored-retired-authorizer"} {
		if strings.Contains(string(encoded), principalID) {
			t.Fatalf("retired Program response exposed principal ID %q: %s", principalID, encoded)
		}
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

func TestProgramOperationsExposeMonitoringResponsibilitiesPerCheck(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.ProgramAggregate{Program: continuity.Program{
		ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "NDPA", Name: "Data protection", Status: continuity.ProgramDraft,
		OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "authorizer-1", Version: 4, CreatedAt: now, UpdatedAt: now,
	}}
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	draft, err := monitoringService.CreateCheck(t.Context(), monitoring.Actor{TenantID: "bank", PrincipalID: "owner-1"}, monitoring.CreateCheckInput{
		ProgramID: aggregate.Program.ID, Code: "DRAFT", Name: "Draft check", Claim: "The draft is reviewed.", InputKind: monitoring.InputSource,
		BindingID: "binding-1", BindingVersion: 1, SourceRules: []monitoring.SourceRule{{ID: "healthy", Field: "healthy", Operator: monitoring.OperatorEquals, Expected: "true", RiskPoints: 100}},
		FreshnessMinutes: 60, MinimumCoverage: 1, OwnerPrincipalID: "owner-1", ReviewerPrincipalID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err := monitoringService.CreateCheck(t.Context(), monitoring.Actor{TenantID: "bank", PrincipalID: "owner-1"}, monitoring.CreateCheckInput{
		ProgramID: aggregate.Program.ID, Code: "ACTIVE", Name: "Active check", Claim: "The active source is evaluated.", InputKind: monitoring.InputSource,
		BindingID: "binding-2", BindingVersion: 1, SourceRules: []monitoring.SourceRule{{ID: "healthy", Field: "healthy", Operator: monitoring.OperatorEquals, Expected: "true", RiskPoints: 100}},
		FreshnessMinutes: 60, MinimumCoverage: 1, OwnerPrincipalID: "owner-1", ReviewerPrincipalID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err = monitoringService.TransitionCheck(t.Context(), monitoring.Actor{TenantID: "bank", PrincipalID: "owner-1"}, monitoring.TransitionInput{ID: active.ID, ExpectedVersion: 1, To: monitoring.LifecyclePendingApproval})
	if err != nil {
		t.Fatal(err)
	}
	active, err = monitoringService.TransitionCheck(t.Context(), monitoring.Actor{TenantID: "bank", PrincipalID: "reviewer-1"}, monitoring.TransitionInput{ID: active.ID, ExpectedVersion: active.Version, To: monitoring.LifecycleActive})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:     {Principal: authority.Principal{ID: "owner-1", DisplayName: "Data Protection Officer"}},
		authority.ResponsibilityReviewer:  {Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Controls reviewer"}},
		authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "performer-1", DisplayName: "Monitoring analyst"}},
	}}
	api := &API{deps: Dependencies{Authority: resolver, Monitoring: monitoringService}}
	find := func(payload programOperationsResponse, command, subresource string) RecordOperation {
		t.Helper()
		for _, operation := range payload.Operations {
			if operation.Command == command && operation.SubresourceID == subresource {
				return operation
			}
		}
		t.Fatalf("operation %s/%s missing: %#v", command, subresource, payload.Operations)
		return RecordOperation{}
	}
	owner := api.buildProgramOperations(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "owner-1", LegalEntityID: "entity-a"}, aggregate, now)
	reviewer := api.buildProgramOperations(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "reviewer-1", LegalEntityID: "entity-a"}, aggregate, now)
	performer := api.buildProgramOperations(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "performer-1", LegalEntityID: "entity-a"}, aggregate, now)
	if operation := find(owner, "program.monitoring.define", ""); !operation.CanAct || operation.AssignedTo == nil || operation.AssignedTo.DisplayName != "Data Protection Officer" {
		t.Fatalf("monitoring definition responsibility = %#v", operation)
	}
	if operation := find(owner, "program.monitoring.transition", draft.ID); !operation.CanAct || operation.Responsibility != string(authority.ResponsibilityOwner) {
		t.Fatalf("draft transition responsibility = %#v", operation)
	}
	if operation := find(reviewer, "program.monitoring.transition", active.ID); !operation.CanAct || operation.AssignedTo == nil || operation.AssignedTo.DisplayName != "Controls reviewer" {
		t.Fatalf("active transition responsibility = %#v", operation)
	}
	if operation := find(performer, "program.monitoring.evaluate", active.ID); !operation.CanAct || operation.AssignedTo == nil || operation.AssignedTo.DisplayName != "Monitoring analyst" {
		t.Fatalf("source evaluation responsibility = %#v", operation)
	}
	if find(reviewer, "program.monitoring.define", "").CanAct || find(owner, "program.monitoring.evaluate", active.ID).CanAct {
		t.Fatal("monitoring operations were granted outside their current responsibility")
	}
}
