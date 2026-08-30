package evidence

import (
	"context"
	"errors"
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
	presentation, err := renderOTPEmail(input.Code, input.ExpiresAt)
	if err != nil {
		return ErrOTPDeliveryFailed
	}
	receipt, err := adapter.delivery.DeliverGoverned(ctx, InvitationDeliveryRequest{
		RecipientAddress: input.Address,
		Subject:          "Verify your email address",
		PlainText:        presentation.PlainText,
		HTML:             presentation.HTML,
	})
	if err != nil || receipt.Status != InvitationDelivered {
		return ErrOTPDeliveryFailed
	}
	return nil
}

func renderOTPEmail(code string, expiresAt time.Time) (renderedEmailPresentation, error) {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 20 || strings.ContainsAny(code, "\r\n\x00") || expiresAt.IsZero() {
		return renderedEmailPresentation{}, errEmailPresentationInvalid
	}
	expiry := expiresAt.UTC().Format("2 Jan 2006, 15:04 UTC")
	return renderEmailPresentation(emailPresentationInput{
		Preheader: "Use this one-time code to verify the invited email address.",
		Heading:   "Verify your email address",
		Intro:     "Enter this one-time code in the open ClearSight request. The code cannot approve or sign off bank work.",
		BodyPlain: "Verification code: " + code + "\n\nIf you did not request this code, ignore this message and do not share it.",
		BodyHTML:  `<p style="margin:18px 0;text-align:center;font-size:30px;line-height:1.2;font-weight:700;letter-spacing:.18em;color:#17212b;">` + html.EscapeString(code) + `</p><p style="margin:0 0 16px;">If you did not request this code, ignore this message and do not share it.</p>`,
		Facts:     []emailFact{{Label: "Code expires", Value: expiry}},
	})
}

var _ OTPDelivery = (*EmailOTPDelivery)(nil)
