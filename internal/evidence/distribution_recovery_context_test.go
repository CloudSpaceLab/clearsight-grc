package evidence

import (
	"context"
	"errors"
	"testing"
)

func TestDistributionRecoveryContextUsesAuthorizedDistributionMetadata(t *testing.T) {
	fixture := newMemoryAccessFixture(t, AccessDirectMagicLink, []DistributionRecipientInput{{
		Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test", AudienceHint: "o***@example.test", ContactLabel: "Owner",
	}})
	issued, err := fixture.access.IssueDistributionAccessRoutes(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
	if err != nil || len(issued) != 1 {
		t.Fatalf("issue route: %+v %v", issued, err)
	}
	redeemed, err := fixture.access.RedeemDirectRoute(context.Background(), issued[0].Selector)
	if err != nil {
		t.Fatal(err)
	}
	session, _, err := fixture.access.SessionRequest(context.Background(), redeemed.SessionToken)
	if err != nil {
		t.Fatal(err)
	}

	value, err := fixture.access.ResponseRecoveryContext(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	if value.LegalEntityID != fixture.distribution.LegalEntityID || value.DistributionID != fixture.distribution.ID || value.SchemaVersion != fixture.distribution.FormTemplateVersion || !value.RouteExpiresAt.Equal(fixture.distribution.RouteExpiresAt) {
		t.Fatalf("unexpected recovery context: %+v distribution=%+v", value, fixture.distribution)
	}
}

func TestDistributionRecoveryContextFailsClosedForUnboundSession(t *testing.T) {
	fixture := newMemoryAccessFixture(t, AccessDirectMagicLink, []DistributionRecipientInput{{
		Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test", AudienceHint: "o***@example.test", ContactLabel: "Owner",
	}})

	_, err := fixture.access.ResponseRecoveryContext(context.Background(), DistributionAccessSession{
		TenantID: "tenant-a", LegalEntityID: "entity-b", DistributionID: fixture.distribution.ID,
	})
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("wrong-entity recovery context did not fail closed: %v", err)
	}
}

func TestDistributionRecoveryContextFailsClosedWithoutStore(t *testing.T) {
	service := &DistributionAccessService{}
	_, err := service.ResponseRecoveryContext(context.Background(), DistributionAccessSession{
		TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a",
	})
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("nil-store recovery context did not fail closed: %v", err)
	}
}
