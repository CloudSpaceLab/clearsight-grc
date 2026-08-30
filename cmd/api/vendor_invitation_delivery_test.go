package main

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestVendorInvitationDeliveryUsesConfiguredCommunicationDelivery(t *testing.T) {
	configured := evidence.NewInvitationDeliveryService(nil)
	services := serviceSet{FormCommunicationTestDelivery: configured}

	if got := vendorInvitationDelivery(services); got != configured {
		t.Fatal("vendor invitations did not use the configured communication delivery service")
	}
}

func TestVendorInvitationDeliveryKeepsLinkOnlyFallbackWithoutSMTP(t *testing.T) {
	if got := vendorInvitationDelivery(serviceSet{}); got == nil {
		t.Fatal("vendor invitations require a non-nil link-only fallback")
	}
}
