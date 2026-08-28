package evidence

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
)

var ErrOTPDeliveryFailed = errors.New("OTP delivery failed")

// EmailOTPDelivery adapts the governed mail boundary to distribution OTPs.
// The raw address and one-time code remain inside one synchronous delivery call
// and are never returned or formatted by this type.
type EmailOTPDelivery struct {
	delivery *InvitationDeliveryService
}

func NewEmailOTPDelivery(delivery *InvitationDeliveryService) *EmailOTPDelivery {
	if delivery == nil {
		return nil
	}
	return &EmailOTPDelivery{delivery: delivery}
}

func (adapter *EmailOTPDelivery) DeliverDistributionOTP(ctx context.Context, input DistributionOTPDelivery) error {
	if adapter == nil || adapter.delivery == nil || strings.TrimSpace(input.Address) == "" || strings.TrimSpace(input.Code) == "" || input.ExpiresAt.IsZero() {
		return ErrOTPDeliveryFailed
	}
	expiresAt := input.ExpiresAt.UTC().Format(time.RFC3339)
	plain := fmt.Sprintf("Your ClearSight verification code is %s. It expires at %s. If you did not request this code, ignore this message.", input.Code, expiresAt)
	markup := "<p>Your ClearSight verification code is <strong>" + html.EscapeString(input.Code) + "</strong>.</p><p>It expires at " + html.EscapeString(expiresAt) + ".</p><p>If you did not request this code, ignore this message.</p>"
	receipt, err := adapter.delivery.DeliverGoverned(ctx, InvitationDeliveryRequest{
		RecipientAddress: input.Address,
		Subject:          "Your ClearSight verification code",
		PlainText:        plain,
		HTML:             markup,
	})
	if err != nil || receipt.Status != InvitationDelivered {
		return ErrOTPDeliveryFailed
	}
	return nil
}

var _ OTPDelivery = (*EmailOTPDelivery)(nil)
