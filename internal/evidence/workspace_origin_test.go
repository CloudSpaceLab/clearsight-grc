package evidence

import "testing"

func TestWorkspaceOriginRequiresExactlyOneHonestSource(t *testing.T) {
	cases := []struct {
		name, origin, principal, session, route, carried string
		assurance                                        AccessAssurance
		valid                                            bool
	}{
		{"external legacy", "", "", "session", "route", "", AssuranceEmailVerified, true},
		{"internal principal", "INTERNAL", "person", "", "", "", AssuranceInternalAuthenticated, true},
		{"carried internal", "", "", "", "", "distribution", AssuranceInternalAuthenticated, true},
		{"carried external", "", "", "", "", "distribution", AssuranceLinkPossession, true},
		{"internal missing principal", "INTERNAL", "", "", "", "", AssuranceInternalAuthenticated, false},
		{"internal blank principal", "INTERNAL", "   ", "", "", "", AssuranceInternalAuthenticated, false},
		{"internal claiming email", "INTERNAL", "person", "", "", "", AssuranceEmailVerified, false},
		{"external claiming internal", "", "", "session", "route", "", AssuranceInternalAuthenticated, false},
		{"mixed session and internal", "INTERNAL", "person", "session", "route", "", AssuranceInternalAuthenticated, false},
		{"mixed carry and external", "", "", "session", "route", "distribution", AssuranceLinkPossession, false},
		{"principal without origin", "", "person", "", "", "", AssuranceInternalAuthenticated, false},
		{"unknown origin", "UNKNOWN", "person", "", "", "", AssuranceInternalAuthenticated, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validWorkspaceEditOrigin(tc.origin, tc.principal, tc.session, tc.route, tc.carried, tc.assurance); got != tc.valid {
				t.Fatalf("validWorkspaceEditOrigin = %t, want %t", got, tc.valid)
			}
		})
	}
}

func TestInternalAssuranceCannotAuthorizeExternalSession(t *testing.T) {
	if validAccessAssurance(AssuranceInternalAuthenticated) {
		t.Fatal("internal identity assurance must not be accepted as external session assurance")
	}
	if !validWorkspaceAssurance(AssuranceInternalAuthenticated) {
		t.Fatal("workspace provenance must recognize internal authenticated assurance")
	}
}
