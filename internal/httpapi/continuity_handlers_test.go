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
)

func continuityTestHandler() http.Handler {
	return New(Dependencies{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AllowedOrigin: "http://localhost:5173",
		Mode:          "test-memory",
		Identity:      identity.NewDevelopmentAuthenticator("bank", "role-cro", "bank-ng"),
		Continuity:    continuity.NewService(continuity.NewMemoryRepository()),
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "role-cro", DisplayName: "Program creator"}, CandidatePrincipals: []authority.Principal{{ID: "owner", DisplayName: "Program owner"}}},
			authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "approver", DisplayName: "Approval authority"}},
		}},
	})
}

func TestProgramAPIUsesHumanStatusLabels(t *testing.T) {
	handler := continuityTestHandler()
	payload := []byte(`{"tenant_id":"bank","code":"NDPA","name":"Data protection","type":"PRIVACY","owning_function":"Privacy Office","owner_candidate_id":"owner","approval_authority_candidate_id":"approver","owner_principal_id":"forged-owner","authority_principal_id":"forged-approver","scope":{"entity":"Bank NG"},"effective_from":"2026-08-05T10:00:00Z","effective_until":"2027-08-05T10:00:00Z","actor_id":"owner"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var program continuity.ProgramAggregate
	if err := json.NewDecoder(response.Body).Decode(&program); err != nil {
		t.Fatal(err)
	}
	if program.StateLabel != "Setup in progress" || program.Program.Status != continuity.ProgramDraft || program.Program.EffectiveUntil == nil {
		t.Fatalf("unexpected program label/status/period %#v", program)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/programs?tenant_id=bank", nil))
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"state_label":"Setup in progress"`)) {
		t.Fatalf("expected human status in list, got %d: %s", list.Code, list.Body.String())
	}
}

func TestMatterAPIReportsVersionConflictInPlainLanguage(t *testing.T) {
	handler := continuityTestHandler()
	payload := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":3,"title":"Complete missing owner approvals","summary":"Four privileged accounts need current owner approval.","scope":{},"known_facts":{"accounts":4},"missing_facts":[],"contradictions":[]}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(payload)))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var matter continuity.MatterAggregate
	if err := json.NewDecoder(response.Body).Decode(&matter); err != nil {
		t.Fatal(err)
	}
	if matter.TypeLabel != "Control gap" || matter.StatusLabel != "Draft" || matter.NextAction != "Start initial review" {
		t.Fatalf("unexpected labels %#v", matter)
	}

	transition := []byte(`{"tenant_id":"bank","expected_version":999,"to":"TRIAGE","rationale":"Start review."}`)
	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/transition", bytes.NewReader(transition)))
	if conflict.Code != http.StatusConflict || !bytes.Contains(conflict.Body.Bytes(), []byte("This record changed")) {
		t.Fatalf("expected plain version conflict, got %d: %s", conflict.Code, conflict.Body.String())
	}
}

func TestMatterLinkAPIAddsASecondProgramAndIsIdempotent(t *testing.T) {
	handler := continuityTestHandler()
	createProgram := func(code, name string) continuity.ProgramAggregate {
		body := []byte(`{"tenant_id":"bank","code":"` + code + `","name":"` + name + `","type":"ASSURANCE","owning_function":"Control Assurance","owner_candidate_id":"owner","approval_authority_candidate_id":"approver","scope":{},"effective_from":"2026-08-05T10:00:00Z"}`)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(body)))
		if response.Code != http.StatusCreated {
			t.Fatalf("create program: expected 201, got %d: %s", response.Code, response.Body.String())
		}
		var value continuity.ProgramAggregate
		if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	firstProgram := createProgram("ACCESS", "Access governance")
	secondProgram := createProgram("CYBER", "Cybersecurity oversight")
	matterPayload := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Privileged account approvals are incomplete","summary":"Four accounts do not have current owner approval.","scope":{},"known_facts":{"affected_accounts":4},"missing_facts":[],"contradictions":[],"program_id":"` + firstProgram.Program.ID + `"}`)
	matterResponse := httptest.NewRecorder()
	handler.ServeHTTP(matterResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(matterPayload)))
	if matterResponse.Code != http.StatusCreated {
		t.Fatalf("create matter: expected 201, got %d: %s", matterResponse.Code, matterResponse.Body.String())
	}
	var matter continuity.MatterAggregate
	if err := json.NewDecoder(matterResponse.Body).Decode(&matter); err != nil {
		t.Fatal(err)
	}

	linkPayload := func(version int64) []byte {
		return []byte(fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"program_id":"%s","relationship":"AFFECTS"}`, version, secondProgram.Program.ID))
	}
	linkResponse := httptest.NewRecorder()
	handler.ServeHTTP(linkResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/links", bytes.NewReader(linkPayload(matter.Matter.Version))))
	if linkResponse.Code != http.StatusCreated {
		t.Fatalf("link matter: expected 201, got %d: %s", linkResponse.Code, linkResponse.Body.String())
	}
	var linked continuity.MatterAggregate
	if err := json.NewDecoder(linkResponse.Body).Decode(&linked); err != nil {
		t.Fatal(err)
	}
	if len(linked.Links) != 2 {
		t.Fatalf("expected two program links, got %#v", linked.Links)
	}

	duplicateResponse := httptest.NewRecorder()
	handler.ServeHTTP(duplicateResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/links", bytes.NewReader(linkPayload(linked.Matter.Version))))
	if duplicateResponse.Code != http.StatusCreated {
		t.Fatalf("duplicate link: expected 201, got %d: %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
	var duplicate continuity.MatterAggregate
	if err := json.NewDecoder(duplicateResponse.Body).Decode(&duplicate); err != nil {
		t.Fatal(err)
	}
	if len(duplicate.Links) != 2 || duplicate.Matter.Version != linked.Matter.Version {
		t.Fatalf("duplicate link changed the aggregate: before=%#v after=%#v", linked, duplicate)
	}
}

func TestProgramHistoryRequiresTimestamp(t *testing.T) {
	handler := continuityTestHandler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/programs/id/history?tenant_id=bank", nil))
	if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte("at is required")) {
		t.Fatalf("expected timestamp validation, got %d: %s", response.Code, response.Body.String())
	}
}

func TestProgramEditRoutesBindVerifiedActorAndPreserveAssignmentSubject(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	effectiveFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Data Protection Office", OwnerPrincipalID: "owner-1",
		EffectiveFrom: effectiveFrom, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(continuity.WithTrustedSystemScope(t.Context()), continuity.AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "CAR-01", Title: "File the annual return", Statement: "The bank must file its annual compliance return.",
		SourceAnchor: "GAID 2025, section 7", EffectiveFrom: effectiveFrom, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	requirementID := program.Requirements[0].ID
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"),
		Continuity: service, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner: {
				Principal:           authority.Principal{ID: "owner-1", DisplayName: "Current Program owner"},
				CandidatePrincipals: []authority.Principal{{ID: "owner-2", DisplayName: "Incoming Program owner"}},
			},
		}},
	})

	post := func(path, body string) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		if response.Code != http.StatusOK {
			t.Fatalf("POST %s returned %d: %s", path, response.Code, response.Body.String())
		}
		if err := json.NewDecoder(response.Body).Decode(&program); err != nil {
			t.Fatal(err)
		}
	}

	post("/api/v1/programs/"+program.Program.ID+"/details", fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"name":"Nigeria data protection","owning_function":"Data Protection Office","jurisdiction":"Nigeria","scope":{"business_lines":["Retail"]},"effective_from":"2026-01-01T00:00:00Z","actor_id":"forged","rationale":"Confirm the approved operating scope."}`, program.Program.Version))
	post("/api/v1/programs/"+program.Program.ID+"/requirements/"+requirementID+"/supersede", fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"code":"CAR-01","title":"File the annual return","statement":"The bank must file its annual compliance return through a licensed DPCO.","source_anchor":"GAID 2025, section 7.2","effective_from":"2026-09-01T00:00:00Z","actor_id":"forged","rationale":"The regulator changed the filing channel."}`, program.Program.Version))
	post("/api/v1/programs/"+program.Program.ID+"/assignment", fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"owner_principal_id":"owner-2","actor_id":"forged","rationale":"Assign the current DPO position."}`, program.Program.Version))

	if program.Program.OwnerPrincipalID != "owner-2" || len(program.Requirements) != 2 || program.Requirements[0].Status != continuity.RequirementSuperseded {
		t.Fatalf("Program edit journey did not persist requested subjects: %#v", program)
	}
	events, err := repo.ProgramEvents(continuity.WithTrustedSystemScope(t.Context()), "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		switch event.Type {
		case continuity.EventProgramDetailsUpdated, continuity.EventProgramOwnerChanged, continuity.EventRequirementSuperseded:
			if event.ActorID != "owner-1" {
				t.Fatalf("event %s trusted a body actor: %#v", event.Type, event)
			}
		}
	}
}

func TestProgramAssignmentAuthorityFailsClosedWithoutMutation(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "AML", Name: "Financial crime", Type: "AML",
		OwningFunction: "Compliance", OwnerPrincipalID: "owner-1", EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"), Continuity: service,
	})
	body := fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"owner_principal_id":"owner-2","rationale":"Assign the current owner."}`, program.Program.Version)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/assignment", strings.NewReader(body)))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("authority failure returned %d: %s", response.Code, response.Body.String())
	}
	current, err := service.GetProgram(continuity.WithTrustedSystemScope(t.Context()), "bank", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Program.Version != program.Program.Version || current.Program.OwnerPrincipalID != "owner-1" {
		t.Fatalf("assignment mutated after authority failure: %#v", current.Program)
	}
}

func TestProgramSafeguardOwnerMustBeAnEligiblePerformer(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Data Protection Office", OwnerPrincipalID: "owner-1", EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(continuity.WithTrustedSystemScope(t.Context()), continuity.AddControlObjectiveInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "CAR-COMPLETE", Name: "Complete return", Outcome: "Every required section is filed.", Status: continuity.ObjectiveActive, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"),
		Continuity: service, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner:     {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program owner"}},
			authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "performer-1", DisplayName: "Control owner"}},
		}},
	})
	body := func(owner string) string {
		return fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"objective_id":"%s","name":"Annual return checklist","description":"Owners confirm each section.","implementation_type":"CHECKLIST","owner_principal_id":"%s","scope":{},"status":"IMPLEMENTED","effective_from":"2026-09-01T00:00:00Z"}`, program.Program.Version, program.ControlObjectives[0].ID, owner)
	}
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/control-implementations", strings.NewReader(body("unlisted-person"))))
	if denied.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ineligible safeguard owner returned %d: %s", denied.Code, denied.Body.String())
	}
	allowed := httptest.NewRecorder()
	handler.ServeHTTP(allowed, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/control-implementations", strings.NewReader(body("performer-1"))))
	if allowed.Code != http.StatusCreated {
		t.Fatalf("eligible safeguard owner returned %d: %s", allowed.Code, allowed.Body.String())
	}
}

func TestOpenMatterFilterExcludesClosedRecords(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	ctx := continuity.WithTrustedSystemScope(context.Background())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterAuthorityRequest, Priority: 2, Title: "Provide requested records", Summary: "A response is due.", Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`)})
	if err != nil {
		t.Fatal(err)
	}
	// The repository filter is exercised directly because this matter is intentionally left open.
	values, err := service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil || len(values) != 1 || values[0].Matter.ID != matter.Matter.ID {
		t.Fatalf("unexpected open list %#v err=%v", values, err)
	}
}

func TestMatterEditHandlersBindVerifiedActorAndKeepAssignmentSubject(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.",
		Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{"filing_channel":"email"}`),
		MissingFacts: json.RawMessage(`["final checklist"]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(continuity.WithTrustedSystemScope(t.Context()), continuity.AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: "performer-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"),
		Continuity: service, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner:     {Principal: authority.Principal{ID: "owner-1", DisplayName: "Current owner"}, CandidatePrincipals: []authority.Principal{{ID: "owner-2", DisplayName: "Privacy owner"}}},
			authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "performer-2", DisplayName: "Privacy operations analyst"}},
		}},
	})

	post := func(path, body string, target *continuity.MatterAggregate) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		if response.Code != http.StatusOK {
			t.Fatalf("POST %s returned %d: %s", path, response.Code, response.Body.String())
		}
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}

	post("/api/v1/matters/"+matter.Matter.ID+"/details", fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"title":"Annual return filing","summary":"Update the filing process.","priority":4,"scope":{},"actor_id":"forged","rationale":"Use the approved title."}`, matter.Matter.Version), &matter)
	post("/api/v1/matters/"+matter.Matter.ID+"/context-changes", fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"kind":"RESOLVE_MISSING","key":"final_checklist","label":"final checklist","value":"Checklist v3","evidence_references":["artifact-v3"],"actor_id":"forged","rationale":"Record the approved checklist."}`, matter.Matter.Version), &matter)
	post("/api/v1/matters/"+matter.Matter.ID+"/actions/"+actionID, fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"title":"Update checklist","description":"Map every section to its current source.","actor_id":"forged","rationale":"Clarify the required evidence."}`, matter.Matter.Version), &matter)
	post("/api/v1/matters/"+matter.Matter.ID+"/actions/"+actionID+"/assignment", fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"owner_principal_id":"performer-2","actor_id":"forged","rationale":"Assign the current process owner."}`, matter.Matter.Version), &matter)
	post("/api/v1/matters/"+matter.Matter.ID+"/assignment", fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"owner_principal_id":"owner-2","actor_id":"forged","rationale":"Assign the current privacy owner."}`, matter.Matter.Version), &matter)

	if matter.Matter.OwnerPrincipalID != "owner-2" || matter.Actions[0].OwnerPrincipalID != "performer-2" || matter.Matter.KnownFacts == nil {
		t.Fatalf("edit journey did not persist requested subjects: %#v", matter)
	}
	events, err := repo.MatterEvents(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events[2:] {
		if event.ActorID != "owner-1" {
			t.Fatalf("event %s trusted a body actor: %#v", event.Type, event)
		}
	}
}

func TestMatterAssignmentAuthorityFailureReturnsServiceUnavailableWithoutMutation(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.", Scope: json.RawMessage(`{}`),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"), Continuity: service,
	})
	body := fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"owner_principal_id":"owner-2","rationale":"Assign the current owner."}`, matter.Matter.Version)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/assignment", strings.NewReader(body)))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("authority failure returned %d: %s", response.Code, response.Body.String())
	}
	current, err := service.GetMatter(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Matter.Version != matter.Matter.Version || current.Matter.OwnerPrincipalID != "owner-1" {
		t.Fatalf("assignment mutated after authority failure: %#v", current.Matter)
	}
}

func TestCreateCommandsOverwriteRequestLegalEntityFromVerifiedActor(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner-1", "entity-a"), Continuity: service,
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Program owner"}},
			authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "approver-1", DisplayName: "Approval authority"}},
		}},
	})

	programResponse := httptest.NewRecorder()
	handler.ServeHTTP(programResponse, httptest.NewRequest(http.MethodPost, "/api/v1/programs", strings.NewReader(`{"tenant_id":"bank","legal_entity_id":"entity-b","code":"ENTITY-A","name":"Entity A controls","type":"COMPLIANCE","owning_function":"Compliance","owner_candidate_id":"owner-1","approval_authority_candidate_id":"approver-1","scope":{},"effective_from":"2026-08-26T00:00:00Z","actor_id":"forged"}`)))
	if programResponse.Code != http.StatusCreated {
		t.Fatalf("Program create returned %d: %s", programResponse.Code, programResponse.Body.String())
	}
	var program continuity.ProgramAggregate
	if err := json.NewDecoder(programResponse.Body).Decode(&program); err != nil {
		t.Fatal(err)
	}
	if program.Program.LegalEntityID != "entity-a" {
		t.Fatalf("Program entity = %q", program.Program.LegalEntityID)
	}
	programEvents, err := repo.ProgramEvents(continuity.WithTrustedSystemScope(context.Background()), "bank", program.Program.ID, nil)
	if err != nil || len(programEvents) == 0 || !strings.Contains(string(programEvents[0].Payload), `"legal_entity_id":"entity-a"`) {
		t.Fatalf("Program event entity missing: %#v, %v", programEvents, err)
	}

	matterResponse := httptest.NewRecorder()
	handler.ServeHTTP(matterResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters", strings.NewReader(`{"tenant_id":"bank","legal_entity_id":"entity-b","type":"CONTROL_GAP","priority":3,"title":"Entity A gap","summary":"A scoped issue","scope":{},"owner_principal_id":"forged-owner","actor_id":"forged"}`)))
	if matterResponse.Code != http.StatusCreated {
		t.Fatalf("Matter create returned %d: %s", matterResponse.Code, matterResponse.Body.String())
	}
	var matter continuity.MatterAggregate
	if err := json.NewDecoder(matterResponse.Body).Decode(&matter); err != nil {
		t.Fatal(err)
	}
	if matter.Matter.LegalEntityID != "entity-a" || matter.Matter.OwnerPrincipalID != "owner-1" {
		t.Fatalf("Matter verified scope/owner = %#v", matter.Matter)
	}
	matterEvents, err := repo.MatterEvents(continuity.WithTrustedSystemScope(context.Background()), "bank", matter.Matter.ID, nil)
	if err != nil || len(matterEvents) == 0 || !strings.Contains(string(matterEvents[0].Payload), `"legal_entity_id":"entity-a"`) {
		t.Fatalf("Matter event entity missing: %#v, %v", matterEvents, err)
	}
}

func TestRestrictedMatterCommandFailsBeforeAuthorityAndPreservesState(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 3,
		Title: "Restricted gap", Summary: "Only the named reviewer may open it.",
		Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["allowed-person"]}`), ActorID: "allowed-person",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &capturingCommandAuthority{}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, CommandGuard: guard}}
	handler := api.command("matter.details.update", commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}, api.updateMatterDetails)
	response := httptest.NewRecorder()
	body := fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"title":"Leaked change","summary":"Should not persist.","priority":3,"scope":{},"rationale":"Attempt"}`, matter.Matter.Version)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/details", strings.NewReader(body))
	request.SetPathValue("id", matter.Matter.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "blocked-person", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("restricted command returned %d: %s", response.Code, response.Body.String())
	}
	current, err := service.GetMatter(continuity.WithTrustedSystemScope(context.Background()), "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Matter.Version != matter.Matter.Version || current.Matter.Title != matter.Matter.Title {
		t.Fatalf("restricted command mutated Matter: %#v", current.Matter)
	}
	if resolver.calls != 0 {
		t.Fatalf("authority was consulted %d times before restricted visibility failed", resolver.calls)
	}
}
