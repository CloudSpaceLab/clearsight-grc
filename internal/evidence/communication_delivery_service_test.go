package evidence

import (
	"context"
	"testing"
	"time"
)

func TestGovernedDeliveryAllowsStatusOnlyCopyWithoutResponseLink(t *testing.T) {
	t.Parallel()

	deliveredAt := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	calls := 0
	service := NewInvitationDeliveryService(invitationDeliveryFunc(func(_ context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		calls++
		if request.InvitationLink != "" {
			t.Fatalf("status-only copy received response link %q", request.InvitationLink)
		}
		return InvitationDeliveryReceipt{Status: InvitationDelivered, DeliveredAt: &deliveredAt}, nil
	}))
	receipt, err := service.DeliverGoverned(context.Background(), InvitationDeliveryRequest{
		RecipientAddress: "observer@example.com",
		Subject:          "Form status changed",
		PlainText:        "The form status changed. No response access is included.",
		HTML:             "<p>The form status changed. No response access is included.</p>",
	})
	if err != nil || receipt.Status != InvitationDelivered || calls != 1 {
		t.Fatalf("governed delivery = (%#v, %v), calls=%d", receipt, err, calls)
	}
}

func TestLegacyInvitationDeliveryStillRequiresResponseLink(t *testing.T) {
	t.Parallel()

	service := NewInvitationDeliveryService(invitationDeliveryFunc(func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		t.Fatal("legacy adapter should not be called for an invalid invitation")
		return InvitationDeliveryReceipt{}, nil
	}))
	if _, err := service.Deliver(context.Background(), InvitationDeliveryRequest{RecipientAddress: "reviewer@example.com"}); err != ErrInvitationDeliveryRequestInvalid {
		t.Fatalf("error = %v, want invalid request", err)
	}
}
