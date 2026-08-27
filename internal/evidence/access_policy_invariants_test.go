package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func TestAccessPolicyRejectsDirectMultiUseAndMalformedDurableRoutes(t *testing.T) {
	now := time.Date(2026, 8, 28, 7, 0, 0, 0, time.UTC)
	engine := testAccessPolicyEngine(now, bytes.Repeat([]byte{0x66}, 64))
	if _, _, err := engine.IssueRoute(AccessRouteInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a",
		Policy: AccessDirectMagicLink, RouteExpiresAt: now.Add(time.Hour), Deadline: now.Add(time.Hour),
		MaxRedemptions: 2, CreatedBy: "actor-a",
	}); !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("direct route accepted multiple redemptions: %v", err)
	}

	invalidRoutes := []AccessRoute{
		{ID: "route-a", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a", Policy: AccessSharedEmailOTP, SelectorHash: make([]byte, 32), MaxRedemptions: 2, CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)},
		{ID: "route-a", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", Policy: AccessDirectEmailOTP, SelectorHash: make([]byte, 32), MaxRedemptions: 1, CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)},
		{ID: "route-a", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", RecipientID: "recipient-a", Policy: AccessDirectMagicLink, SelectorHash: make([]byte, 32), MaxRedemptions: 1, CreatedAt: now.Add(time.Minute), ExpiresAt: now.Add(time.Hour)},
	}
	for index, route := range invalidRoutes {
		if accessRouteOpen(route, now) {
			t.Fatalf("malformed durable route %d was accepted", index)
		}
	}
}
