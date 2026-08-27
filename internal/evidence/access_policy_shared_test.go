package evidence

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestAccessPolicySharedEmailOTPUsesRouteBoundOpaqueSelectors(t *testing.T) {
	now := time.Date(2026, 8, 27, 22, 0, 0, 0, time.UTC)
	random := append(bytes.Repeat([]byte{0x31}, 32), bytes.Repeat([]byte{0x42}, 32)...)
	engine := testAccessPolicyEngine(now, random)
	input := AccessRouteInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a",
		Policy: AccessSharedEmailOTP, RouteExpiresAt: now.Add(4 * time.Hour), Deadline: now.Add(3 * time.Hour), CreatedBy: "actor-a",
	}
	route, issued, err := engine.IssueRoute(input)
	if err != nil {
		t.Fatal(err)
	}
	recipients := []DistributionRecipient{
		accessRecipient("recipient-z", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Zeta"),
		accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Alpha"),
		accessRecipient("recipient-cc", RecipientCC, RecipientExternalAudience, DistributionRecipientPending, "Observer"),
	}
	start, err := engine.Start(route, issued.Selector, recipients, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(start.Recipients) != 2 || start.Recipients[0].ContactLabel != "Alpha" || start.Recipients[1].ContactLabel != "Zeta" {
		t.Fatalf("shared route did not return deterministic eligible TO recipients: %+v", start.Recipients)
	}
	if start.Recipients[0].SelectorID == start.Recipients[1].SelectorID {
		t.Fatal("shared route reused a recipient selector")
	}
	resolved, err := engine.ResolveOTPRecipient(route, issued.Selector, start.Recipients[1].SelectorID, recipients, now)
	if err != nil || resolved.ID != "recipient-z" {
		t.Fatalf("shared recipient selector resolved incorrectly: %+v %v", resolved, err)
	}

	next, nextIssued, err := engine.RotateRoute(&route, input, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if route.RevokedAt == nil || nextIssued.Selector == issued.Selector || next.ID == route.ID {
		t.Fatalf("route rotation did not revoke and replace the selector: old=%+v new=%+v", route, next)
	}
	if _, err := engine.Start(route, issued.Selector, recipients, now.Add(2*time.Minute)); !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("revoked route remained inspectable: %v", err)
	}
	nextStart, err := engine.Start(next, nextIssued.Selector, recipients, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if nextStart.Recipients[0].SelectorID == start.Recipients[0].SelectorID {
		t.Fatal("recipient selector survived route rotation")
	}
}
