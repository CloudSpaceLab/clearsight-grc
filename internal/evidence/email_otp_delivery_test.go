package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestEmailOTPDeliveryUsesGovernedProtectedEnvelope(t *testing.T) {
	t.Parallel()

	const address = "reviewer@example.com"
	const code = "482193"
	deliveredAt := time.Date(2026, 8, 28, 7, 30, 0, 0, time.UTC)
	adapter := invitationDeliveryFunc(func(_ context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		if request.RecipientAddress != address || request.InvitationLink != "" {
			t.Fatalf("unexpected OTP envelope routing: %#v", request)
		}
		if !strings.Contains(request.PlainText, code) || !strings.Contains(request.HTML, code) {
			t.Fatal("OTP code was not supplied to the protected delivery adapter")
		}
		if strings.Contains(fmt.Sprintf("%v", request), address) || strings.Contains(fmt.Sprintf("%#v", request), code) {
			t.Fatal("protected OTP envelope leaked through formatting")
		}
		return InvitationDeliveryReceipt{Status: InvitationDelivered, DeliveredAt: &deliveredAt}, nil
	})

	delivery := NewEmailOTPDelivery(NewInvitationDeliveryService(adapter))
	err := delivery.DeliverDistributionOTP(context.Background(), DistributionOTPDelivery{
		Address: address, Code: code, ChallengeID: "challenge", DistributionID: "distribution",
		ExpiresAt: deliveredAt.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("deliver OTP: %v", err)
	}
}

func TestEmailOTPDeliveryReturnsFixedFailureWithoutProtectedValues(t *testing.T) {
	t.Parallel()

	const address = "reviewer@example.com"
	const code = "482193"
	delivery := NewEmailOTPDelivery(NewInvitationDeliveryService(nil))
	err := delivery.DeliverDistributionOTP(context.Background(), DistributionOTPDelivery{
		Address: address, Code: code, ChallengeID: "challenge", DistributionID: "distribution",
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	})
	if !errors.Is(err, ErrOTPDeliveryFailed) {
		t.Fatalf("error = %v, want fixed OTP delivery failure", err)
	}
	message := fmt.Sprintf("%v", err)
	if strings.Contains(message, address) || strings.Contains(message, code) {
		t.Fatalf("OTP failure leaked protected values: %q", message)
	}
}
