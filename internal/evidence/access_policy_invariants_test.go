package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAccessPolicyPublicProjectionsOmitInternalSelectorsAndRecipientIDs(t *testing.T) {
	now := time.Date(2026, 8, 27, 23, 0, 0, 0, time.UTC)
	engine := testAccessPolicyEngine(now, bytes.Repeat([]byte{0x55}, 32))
	route, issued, err := engine.IssueRoute(AccessRouteInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-secret",
		Policy: AccessDirectEmailOTP, RouteExpiresAt: now.Add(time.Hour), Deadline: now.Add(time.Hour), CreatedBy: "actor-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(route)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, route.RecipientID) || strings.Contains(text, issued.Selector) || strings.Contains(text, fmt.Sprintf("%x", route.SelectorHash)) {
		t.Fatalf("route JSON leaked protected selectors: %s", text)
	}
}

func TestMagicLinkExpiryMigrationAllowsIndependentDirectLinksAndKeepsOTPReplacementUnique(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000075_magic_link_expiry_semantics.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upSQL := string(up)
	for _, required := range []string{
		"DROP INDEX capture_access_routes_active_direct_uq",
		"CREATE UNIQUE INDEX capture_access_routes_active_direct_otp_uq",
		"access_policy='DIRECT_LINK_EMAIL_OTP'",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("magic-link expiry migration lacks %q", required)
		}
	}
	if strings.Contains(upSQL, "access_policy='DIRECT_MAGIC_LINK'") {
		t.Fatal("magic-link expiry migration retained a single-active-link constraint")
	}
}

func TestAccessPolicyRejectsUnconfiguredHMACKey(t *testing.T) {
	now := time.Date(2026, 8, 27, 23, 30, 0, 0, time.UTC)
	engine := NewAccessPolicyEngine([32]byte{})
	engine.now = func() time.Time { return now }
	_, _, err := engine.IssueRoute(AccessRouteInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a",
		Policy: AccessDirectMagicLink, RouteExpiresAt: now.Add(time.Hour), Deadline: now.Add(time.Hour), CreatedBy: "actor-a",
	})
	if !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("zero HMAC key enabled access routes: %v", err)
	}
}

func TestAccessPolicyRejectsMalformedDurableRoutes(t *testing.T) {
	now := time.Date(2026, 8, 28, 7, 0, 0, 0, time.UTC)

	invalidRoutes := []AccessRoute{
		{ID: "route-a", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a", Policy: AccessSharedEmailOTP, SelectorHash: make([]byte, 32), CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)},
		{ID: "route-a", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", Policy: AccessDirectEmailOTP, SelectorHash: make([]byte, 32), CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)},
		{ID: "route-a", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a", Policy: AccessDirectMagicLink, SelectorHash: make([]byte, 32), CreatedAt: now.Add(time.Minute), ExpiresAt: now.Add(time.Hour)},
	}
	for index, route := range invalidRoutes {
		if accessRouteOpen(route, now) {
			t.Fatalf("malformed durable route %d was accepted", index)
		}
	}
}
