package httpapi

import (
	"encoding/json"
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

func TestRelationshipRetirementUsesExactCurrentOwnerAndRouteBoundLink(t *testing.T) {
	service, program := programWithLifecycleResources(t)
	ctx := continuity.WithTrustedSystemScope(t.Context())
	program, err := service.AddRequirement(ctx, continuity.AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "REQ", Title: "File the return", Statement: "The bank must file the return.", Status: continuity.RequirementApproved, EffectiveFrom: time.Now().UTC(), ActorID: "program-owner"})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.LinkRequirementControl(ctx, continuity.LinkRequirementControlInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: program.Requirements[0].ID, ImplementationID: program.ControlImplementations[0].ID, ActorID: "program-owner"})
	if err != nil {
		t.Fatal(err)
	}
	linkID := program.RequirementControlLinks[0].ID
	resolver := &lifecycleAuthorityCapture{}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "program-owner", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	operations := api.buildProgramOperations(identity.WithActor(t.Context(), actor), actor, program, time.Now().UTC()).Operations
	assertOperationCanAct(t, operations, "program.safeguard.unlink", linkID)
	handler := api.command("program.safeguard.unlink", commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 3}, api.retireProgramRequirementControlLink)
	body, _ := json.Marshal(map[string]any{"tenant_id": "bank", "expected_version": program.Program.Version, "actor_id": "forged-person", "rationale": "The safeguard was mapped to the wrong requirement."})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/control-links/"+linkID+"/retirement", strings.NewReader(string(body)))
	request.SetPathValue("id", program.Program.ID)
	request.SetPathValue("link_id", linkID)
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("retire Program link returned %d: %s", response.Code, response.Body.String())
	}
	current, err := service.GetProgram(ctx, "bank", program.Program.ID)
	if err != nil || len(current.RequirementControlLinks) != 0 {
		t.Fatalf("current Program links = %#v, err=%v", current.RequirementControlLinks, err)
	}
	assertAuthorityInput(t, resolver.inputs, "program.safeguard.unlink", "PROGRAM", program.Program.ID, authority.ResponsibilityOwner)
}

func TestMatterLinkRetirementUsesExactCurrentOwnerAndRemovesOperation(t *testing.T) {
	service, program := programWithLifecycleResources(t)
	ctx := continuity.WithTrustedSystemScope(t.Context())
	matter, err := service.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 3,
		Title: "Correct the issue link", Summary: "The linked Program is incorrect.", Scope: json.RawMessage(`{}`),
		ProgramID: program.Program.ID, OwnerPrincipalID: "program-owner", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	linkID := matter.Links[0].ID
	resolver := &lifecycleAuthorityCapture{}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service, Authority: resolver, CommandGuard: guard}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "program-owner", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	operations := api.buildMatterOperations(identity.WithActor(t.Context(), actor), actor, matter, time.Now().UTC()).Operations
	assertOperationCanAct(t, operations, "matter.unlink", linkID)
	handler := api.command("matter.unlink", commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}, api.retireMatterLink)
	body, _ := json.Marshal(map[string]any{"tenant_id": "bank", "expected_version": matter.Matter.Version, "actor_id": "forged-person", "rationale": "This issue does not affect the Program."})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/links/"+linkID+"/retirement", strings.NewReader(string(body)))
	request.SetPathValue("id", matter.Matter.ID)
	request.SetPathValue("link_id", linkID)
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("retire Matter link returned %d: %s", response.Code, response.Body.String())
	}
	current, err := service.GetMatter(ctx, "bank", matter.Matter.ID)
	if err != nil || len(current.Links) != 0 {
		t.Fatalf("current Matter links = %#v, err=%v", current.Links, err)
	}
	assertAuthorityInput(t, resolver.inputs, "matter.unlink", "MATTER", matter.Matter.ID, authority.ResponsibilityOwner)
}

func assertOperationCanAct(t *testing.T, operations []RecordOperation, command, subresourceID string) {
	t.Helper()
	for _, operation := range operations {
		if operation.Command == command && operation.SubresourceID == subresourceID {
			if !operation.CanAct {
				t.Fatalf("operation cannot act: %#v", operation)
			}
			return
		}
	}
	t.Fatalf("operation %s/%s not found: %#v", command, subresourceID, operations)
}

func assertAuthorityInput(t *testing.T, inputs []authority.ResolveInput, decisionType, objectType, objectID string, responsibility authority.Responsibility) {
	t.Helper()
	for _, input := range inputs {
		if input.DecisionType == decisionType && input.ObjectType == objectType && input.ObjectID == objectID && input.Responsibility == responsibility {
			return
		}
	}
	t.Fatalf("exact authority input not found: %#v", inputs)
}
