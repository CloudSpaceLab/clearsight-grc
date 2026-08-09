package continuity

import (
	"encoding/json"
	"testing"
)

func TestMatterAccessPolicyFailsClosed(t *testing.T) {
	tests := []struct {
		name  string
		scope json.RawMessage
	}{
		{name: "missing scope", scope: nil},
		{name: "malformed json", scope: json.RawMessage(`{"access":`)},
		{name: "non-string access", scope: json.RawMessage(`{"access":42}`)},
		{name: "unknown access", scope: json.RawMessage(`{"access":"SECRET"}`)},
		{name: "restricted without allow list", scope: json.RawMessage(`{"access":"RESTRICTED"}`)},
		{name: "restricted empty allow list", scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":[]}`)},
		{name: "restricted mixed-type allow list", scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-1",42]}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, valid := ParseMatterAccessPolicy(test.scope); valid {
				t.Fatal("invalid access metadata was accepted")
			}
			if MatterVisibleTo(Matter{Scope: test.scope}, "person-1") {
				t.Fatal("invalid access metadata was visible")
			}
		})
	}
}

func TestMatterAccessPolicyDefaultsToTenantInternal(t *testing.T) {
	matter := Matter{Scope: json.RawMessage(`{"business_area":"Treasury"}`)}
	policy, valid := ParseMatterAccessPolicy(matter.Scope)
	if !valid || policy.Access != MatterAccessInternal {
		t.Fatalf("unexpected policy: %#v valid=%v", policy, valid)
	}
	if !MatterVisibleTo(matter, "person-1") {
		t.Fatal("tenant-internal matter should be visible after tenant verification")
	}
}

func TestRestrictedMatterRequiresExplicitPrincipal(t *testing.T) {
	matter := Matter{Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-1","person-1"," "]}`)}
	policy, valid := ParseMatterAccessPolicy(matter.Scope)
	if !valid || len(policy.AllowedPrincipalIDs) != 1 {
		t.Fatalf("allow-list was not normalized: %#v valid=%v", policy, valid)
	}
	if !MatterVisibleTo(matter, "person-1") {
		t.Fatal("allowed principal could not read restricted matter")
	}
	if MatterVisibleTo(matter, "person-2") || MatterVisibleTo(matter, "") {
		t.Fatal("restricted matter was visible without an explicit grant")
	}
}
