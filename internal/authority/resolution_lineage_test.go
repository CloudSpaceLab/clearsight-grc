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
