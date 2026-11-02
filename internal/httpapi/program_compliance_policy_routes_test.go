package httpapi

import (
	"net/http"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func TestProgramCalculationPolicyCommandsUseCurrentBusinessAuthority(t *testing.T) {
	expected := map[string]struct {
		command string
		role    authority.Responsibility
	}{
		"/api/v1/programs/{id}/calculation-policy/propose": {
			command: "program.compliance-policy.propose", role: authority.ResponsibilityOwner,
		},
		"/api/v1/programs/{id}/calculation-policy/approve": {
			command: "program.compliance-policy.approve", role: authority.ResponsibilityAuthorizer,
		},
		"/api/v1/programs/{id}/calculation-policy/reject": {
			command: "program.compliance-policy.reject", role: authority.ResponsibilityAuthorizer,
		},
	}
	for _, route := range (&API{}).routes() {
		want, exists := expected[route.Path]
		if !exists || route.Method != http.MethodPost {
			continue
		}
		if route.Command == nil || route.Command.Name != want.command ||
			route.Command.Policy.ObjectType != "PROGRAM" ||
			route.Command.Policy.Responsibility != want.role ||
			route.Command.Policy.ActorField != noActorField || route.Permission != "" {
			t.Fatalf("Program calculation policy command is not properly governed: %+v", route)
		}
		delete(expected, route.Path)
	}
	if len(expected) != 0 {
		t.Fatalf("missing Program calculation policy routes: %+v", expected)
	}
	if !programOwnerBoundCommand("program.compliance-policy.propose") {
		t.Fatal("proposing calculation rules bypassed current stored Program owner")
	}
	if programOwnerBoundCommand("program.compliance-policy.approve") ||
		programOwnerBoundCommand("program.compliance-policy.reject") {
		t.Fatal("an authorizer decision was treated as a Program owner action")
	}
}
