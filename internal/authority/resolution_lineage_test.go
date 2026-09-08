package authority

import "testing"

func TestResolutionAllowsOnlyDelegatesActingForStoredAuthority(t *testing.T) {
	resolution := Resolution{
		Principal: Principal{ID: "stored-authority"},
		CandidatePrincipals: []Principal{
			{ID: "stored-authority"},
			{ID: "valid-delegate"},
			{ID: "other-route-candidate"},
		},
		EffectiveOrigins: []EffectiveOrigin{
			{PrincipalID: "stored-authority", OriginPrincipalID: "stored-authority"},
			{PrincipalID: "valid-delegate", OriginPrincipalID: "stored-authority"},
			{PrincipalID: "other-route-candidate", OriginPrincipalID: "other-route-candidate"},
		},
	}

	if !resolution.AllowsPrincipalFor("stored-authority", "stored-authority") {
		t.Fatal("stored authority was not allowed to act for itself")
	}
	if !resolution.AllowsPrincipalFor("valid-delegate", "stored-authority") {
		t.Fatal("active delegate was not allowed to act for the stored authority")
	}
	if resolution.AllowsPrincipalFor("other-route-candidate", "stored-authority") {
		t.Fatal("unrelated route candidate was allowed to act for the stored authority")
	}
}

func TestResolutionMatchesConfiguredRoleCodeThroughDelegation(t *testing.T) {
	r := Resolution{Principal: Principal{ID: "delegate"}, CandidatePrincipals: []Principal{{ID: "delegate"}, {ID: "reviewer", Role: "Chief information security officer", RoleCode: "CISO"}}, EffectiveOrigins: []EffectiveOrigin{{PrincipalID: "delegate", OriginPrincipalID: "reviewer", RoleCode: "CISO"}}}
	if !r.AllowsPrincipalWithRole("reviewer", "CISO") || !r.AllowsPrincipalWithRole("delegate", "CISO") {
		t.Fatal("configured role code lost")
	}
	if r.AllowsPrincipalWithRole("delegate", "OWNER") || r.AllowsPrincipalWithRole("stranger", "CISO") {
		t.Fatal("unrelated role or principal admitted")
	}
}
