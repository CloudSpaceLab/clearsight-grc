package identity

import "testing"

func TestHasDepartmentPermissionRequiresExactScope(t *testing.T) {
	actor := Actor{DepartmentGrants: []DepartmentGrant{{
		Path: []string{"bank", "operations", "payments"}, PermissionCodes: []string{"matter-read"},
	}}}
	if !HasDepartmentPermission(actor, []string{"BANK", "OPERATIONS", "PAYMENTS"}, "MATTER_READ") {
		t.Fatal("expected exact department grant")
	}
	if HasDepartmentPermission(actor, []string{"BANK", "OPERATIONS"}, "MATTER_READ") {
		t.Fatal("parent department must not inherit child capability")
	}
	if HasDepartmentPermission(actor, []string{"BANK", "OPERATIONS", "PAYMENTS", "CARDS"}, "MATTER_READ") {
		t.Fatal("child department must not inherit parent capability")
	}
}

func TestNormalizeDepartmentPathRejectsEmptySegment(t *testing.T) {
	if _, err := NormalizeDepartmentPath([]string{"BANK", " "}); err == nil {
		t.Fatal("expected empty department segment to fail")
	}
}
