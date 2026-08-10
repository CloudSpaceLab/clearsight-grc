package identity

import (
	"errors"
	"strings"
	"testing"
	"time"
)

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

func TestActorValidRejectsOversizedSessionMetadata(t *testing.T) {
	now := time.Now().UTC()
	actor := Actor{
		TenantID: "bank-demo", PrincipalID: "principal-1", LegalEntityID: "BANK-NG",
		AuthenticationMethod: "OIDC", AssuranceLevel: strings.Repeat("x", 513), SessionID: "ses_test",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if err := actor.Valid(now); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("expected oversized assurance to invalidate actor, got %v", err)
	}
}
