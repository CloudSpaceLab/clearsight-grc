package evidence

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAccessPolicyDirectMagicLinkRedeemsLinkPossessionAndClampsDeadline(t *testing.T) {
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	engine := testAccessPolicyEngine(now, bytes.Repeat([]byte{0x11}, 32))
	route, issued, err := engine.IssueRoute(AccessRouteInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a",
		Policy: AccessDirectMagicLink, AudienceHint: "o***@example.test", RouteExpiresAt: now.Add(24 * time.Hour),
		Deadline: now.Add(2 * time.Hour), CreatedBy: "actor-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !route.ExpiresAt.Equal(now.Add(2*time.Hour)) || !issued.ExpiresAt.Equal(route.ExpiresAt) {
		t.Fatalf("route was not clamped to deadline: %+v %+v", route, issued)
	}
	if issued.Selector == "" || strings.Contains(issued.String(), issued.Selector) || strings.Contains(fmt.Sprintf("%#v", route), route.RecipientID) {
		t.Fatal("route selector or bound recipient leaked through formatted output")
	}

	recipients := []DistributionRecipient{accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Owner")}
	start, err := engine.Start(route, issued.Selector, recipients, now)
	if err != nil {
		t.Fatal(err)
	}
	if start.Policy != AccessDirectMagicLink || len(start.Recipients) != 0 {
		t.Fatalf("direct magic link unexpectedly requested identity selection: %+v", start)
	}
	grant, err := engine.RedeemDirectRoute(&route, issued.Selector, recipients, now.Add(8*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if grant.Assurance != AssuranceLinkPossession || grant.RecipientID != "recipient-a" || !grant.ExpiresAt.Equal(route.ExpiresAt) {
		t.Fatalf("unexpected direct route grant: %+v", grant)
	}
	if !AccessGrantUsable(route, grant, now.Add(time.Minute)) {
		t.Fatal("bounded direct route session became unusable after consuming the one-time route")
	}
	wrongAssurance := grant
	wrongAssurance.Assurance = AssuranceEmailVerified
	if AccessGrantUsable(route, wrongAssurance, now.Add(time.Minute)) {
		t.Fatal("direct magic-link session accepted email-verified assurance")
	}
	if _, err := engine.RedeemDirectRoute(&route, issued.Selector, recipients, now.Add(time.Hour), now); !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("one-time route allowed a second redemption: %v", err)
	}
	if err := RevokeAccessRoute(&route, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if AccessGrantUsable(route, grant, now.Add(6*time.Minute)) {
		t.Fatal("route revocation did not invalidate its session grant")
	}
}

func TestAccessPolicyDirectEmailOTPExposesOnlyItsBoundMaskedRecipient(t *testing.T) {
	now := time.Date(2026, 8, 27, 21, 0, 0, 0, time.UTC)
	engine := testAccessPolicyEngine(now, bytes.Repeat([]byte{0x22}, 32))
	route, issued, err := engine.IssueRoute(AccessRouteInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a",
		Policy: AccessDirectEmailOTP, RouteExpiresAt: now.Add(time.Hour), Deadline: now.Add(2 * time.Hour), CreatedBy: "actor-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	recipients := []DistributionRecipient{
		accessRecipient("recipient-b", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Backup"),
		accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Primary"),
		accessRecipient("recipient-cc", RecipientCC, RecipientExternalAudience, DistributionRecipientPending, "Observer"),
		accessRecipient("recipient-internal", RecipientTo, RecipientInternalPrincipal, DistributionRecipientPending, "Employee"),
		accessRecipient("recipient-revoked", RecipientTo, RecipientExternalAudience, DistributionRecipientRevoked, "Revoked"),
	}
	start, err := engine.Start(route, issued.Selector, recipients, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(start.Recipients) != 1 || start.Recipients[0].Hint != "r***@example.test" || start.Recipients[0].ContactLabel != "Primary" {
		t.Fatalf("direct OTP route exposed the wrong recipient set: %+v", start.Recipients)
	}
	selector := start.Recipients[0].SelectorID
	if selector == "" || selector == "recipient-a" || strings.Contains(selector, "recipient-a") {
		t.Fatalf("recipient database ID leaked into public selector %q", selector)
	}
	resolved, err := engine.ResolveOTPRecipient(route, issued.Selector, selector, recipients, now)
	if err != nil || resolved.ID != "recipient-a" {
		t.Fatalf("bound selector resolved incorrectly: %+v %v", resolved, err)
	}
	for name, invalid := range map[string]string{"other-recipient": engine.recipientSelector(route.ID, "recipient-b"), "random": "not-a-selector"} {
		t.Run(name, func(t *testing.T) {
			_, err := engine.ResolveOTPRecipient(route, issued.Selector, invalid, recipients, now)
			if !errors.Is(err, ErrAccessVerificationFailed) || err.Error() != ErrAccessVerificationFailed.Error() {
				t.Fatalf("unknown recipient returned distinguishable error: %v", err)
			}
		})
	}
	if _, err := engine.RedeemVerifiedRoute(&route, OTPVerification{challengeID: "challenge-b", routeID: route.ID, distributionID: route.DistributionID, recipientID: "recipient-b"}, now.Add(30*time.Minute), now); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("direct OTP route accepted a different recipient: %v", err)
	}
	if _, err := engine.RedeemVerifiedRoute(&route, OTPVerification{routeID: route.ID, distributionID: route.DistributionID, recipientID: resolved.ID}, now.Add(30*time.Minute), now); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("direct OTP route accepted a fabricated verification without a challenge: %v", err)
	}
	grant, err := engine.RedeemVerifiedRoute(&route, OTPVerification{challengeID: "challenge-a", routeID: route.ID, distributionID: route.DistributionID, recipientID: resolved.ID}, now.Add(30*time.Minute), now)
	if err != nil || grant.Assurance != AssuranceEmailVerified {
		t.Fatalf("verified route did not produce email assurance: %+v %v", grant, err)
	}
}
