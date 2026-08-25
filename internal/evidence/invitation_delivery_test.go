package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type invitationDeliveryFunc func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error)

func (function invitationDeliveryFunc) Deliver(ctx context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
	return function(ctx, request)
}

func TestInvitationDeliveryFallsBackWhenAdapterIsUnavailable(t *testing.T) {
	t.Parallel()

	request := InvitationDeliveryRequest{
		RecipientAddress: "security@vendor.example",
		InvitationLink:   "https://capture.example/respond?capture_invite=opaque-token",
	}

	for name, adapter := range map[string]InvitationDelivery{
		"not configured": nil,
		"unavailable": invitationDeliveryFunc(func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
			return InvitationDeliveryReceipt{}, ErrInvitationDeliveryUnavailable
		}),
	} {
		t.Run(name, func(t *testing.T) {
			receipt, err := NewInvitationDeliveryService(adapter).Deliver(context.Background(), request)
			if err != nil {
				t.Fatalf("fallback returned an error: %v", err)
			}
			if receipt.Status != InvitationLinkCreatedEmailNotSent {
				t.Fatalf("status = %q, want %q", receipt.Status, InvitationLinkCreatedEmailNotSent)
			}
			if receipt.RecipientHint != "s***@vendor.example" {
				t.Fatalf("recipient hint = %q", receipt.RecipientHint)
			}
			if receipt.DeliveredAt != nil || receipt.FailureCode != "" {
				t.Fatalf("fallback claimed delivery or failure: %#v", receipt)
			}
		})
	}

	assertInvitationDeliveryValueRedacted(t, request, "security@vendor.example", "opaque-token")
}

func TestInvitationDeliveryReturnsSuccessfulRedactedReceipt(t *testing.T) {
	t.Parallel()

	deliveredAt := time.Date(2026, 8, 26, 12, 30, 0, 0, time.UTC)
	request := InvitationDeliveryRequest{
		RecipientAddress: "security@vendor.example",
		InvitationLink:   "https://capture.example/respond?capture_invite=opaque-token",
	}
	adapter := invitationDeliveryFunc(func(_ context.Context, got InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		if got.RecipientAddress != request.RecipientAddress || got.InvitationLink != request.InvitationLink {
			t.Fatalf("adapter did not receive the request-scoped delivery values: %#v", got)
		}
		return InvitationDeliveryReceipt{
			Status:        InvitationDelivered,
			RecipientHint: "security@vendor.example",
			DeliveredAt:   &deliveredAt,
		}, nil
	})

	receipt, err := NewInvitationDeliveryService(adapter).Deliver(context.Background(), request)
	if err != nil {
		t.Fatalf("delivery failed: %v", err)
	}
	if receipt.Status != InvitationDelivered || receipt.RecipientHint != "s***@vendor.example" {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
	if receipt.DeliveredAt == nil || !receipt.DeliveredAt.Equal(deliveredAt) {
		t.Fatalf("delivery timestamp = %v, want %v", receipt.DeliveredAt, deliveredAt)
	}
	if receipt.FailureCode != "" {
		t.Fatalf("successful delivery retained a failure code: %#v", receipt)
	}
	assertInvitationDeliveryValueRedacted(t, receipt, request.RecipientAddress, "opaque-token")
}

func TestInvitationDeliveryReturnsBoundedFailureReceipt(t *testing.T) {
	t.Parallel()

	request := InvitationDeliveryRequest{
		RecipientAddress: "security@vendor.example",
		InvitationLink:   "https://capture.example/respond?capture_invite=opaque-token",
	}
	adapter := invitationDeliveryFunc(func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		return InvitationDeliveryReceipt{
			Status:        InvitationDeliveryFailed,
			RecipientHint: "provider-supplied-value",
			FailureCode:   InvitationFailureRecipientRejected,
		}, nil
	})

	receipt, err := NewInvitationDeliveryService(adapter).Deliver(context.Background(), request)
	if err != nil {
		t.Fatalf("bounded provider failure returned an execution error: %v", err)
	}
	if receipt.Status != InvitationDeliveryFailed || receipt.FailureCode != InvitationFailureRecipientRejected {
		t.Fatalf("unexpected failure receipt: %#v", receipt)
	}
	if receipt.RecipientHint != "s***@vendor.example" || receipt.DeliveredAt != nil {
		t.Fatalf("failure receipt was not normalized: %#v", receipt)
	}
}

func TestInvitationDeliverySanitizesAdapterErrors(t *testing.T) {
	t.Parallel()

	request := InvitationDeliveryRequest{
		RecipientAddress: "security@vendor.example",
		InvitationLink:   "https://capture.example/respond?capture_invite=opaque-token",
	}
	adapter := invitationDeliveryFunc(func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		return InvitationDeliveryReceipt{}, errors.New("could not send to security@vendor.example using opaque-token")
	})

	receipt, err := NewInvitationDeliveryService(adapter).Deliver(context.Background(), request)
	if !errors.Is(err, ErrInvitationDeliveryFailed) {
		t.Fatalf("error = %v, want fixed delivery failure", err)
	}
	if receipt.Status != InvitationDeliveryFailed || receipt.FailureCode != InvitationFailureProviderError {
		t.Fatalf("unexpected failure receipt: %#v", receipt)
	}
	assertInvitationDeliveryValueRedacted(t, receipt, request.RecipientAddress, "opaque-token")
	assertInvitationDeliveryValueRedacted(t, err, request.RecipientAddress, "opaque-token")
}

func TestInvitationDeliveryRejectsMaliciousReceipt(t *testing.T) {
	t.Parallel()

	request := InvitationDeliveryRequest{
		RecipientAddress: "security@vendor.example",
		InvitationLink:   "https://capture.example/respond?capture_invite=opaque-token",
	}
	tests := map[string]InvitationDeliveryReceipt{
		"address in failure code": {
			Status:      InvitationDeliveryFailed,
			FailureCode: InvitationDeliveryFailureCode("security@vendor.example"),
		},
		"token in failure code": {
			Status:      InvitationDeliveryFailed,
			FailureCode: InvitationDeliveryFailureCode("opaque-token"),
		},
		"unbounded failure code": {
			Status:      InvitationDeliveryFailed,
			FailureCode: InvitationDeliveryFailureCode(strings.Repeat("A", 65)),
		},
		"unrecognized failure code": {
			Status:      InvitationDeliveryFailed,
			FailureCode: InvitationDeliveryFailureCode("VENDOR_SPECIFIC_CODE"),
		},
		"unknown status": {
			Status: InvitationDeliveryStatus("DELIVERED:security@vendor.example"),
		},
	}

	for name, providerReceipt := range tests {
		t.Run(name, func(t *testing.T) {
			adapter := invitationDeliveryFunc(func(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
				return providerReceipt, nil
			})
			receipt, err := NewInvitationDeliveryService(adapter).Deliver(context.Background(), request)
			if !errors.Is(err, ErrInvitationDeliveryReceiptUnsafe) {
				t.Fatalf("error = %v, want unsafe receipt", err)
			}
			if receipt.Status != InvitationDeliveryFailed || receipt.FailureCode != InvitationFailureInvalidReceipt {
				t.Fatalf("unsafe receipt was not replaced: %#v", receipt)
			}
			assertInvitationDeliveryValueRedacted(t, receipt, request.RecipientAddress, "opaque-token")
			assertInvitationDeliveryValueRedacted(t, err, request.RecipientAddress, "opaque-token")
		})
	}
}

func assertInvitationDeliveryValueRedacted(t *testing.T, value any, secrets ...string) {
	t.Helper()

	values := []string{fmt.Sprintf("%v", value), fmt.Sprintf("%#v", value)}
	if encoded, err := json.Marshal(value); err == nil {
		values = append(values, string(encoded))
	}
	for _, rendered := range values {
		for _, secret := range secrets {
			if strings.Contains(rendered, secret) {
				t.Fatalf("protected delivery value exposed %q in %q", secret, rendered)
			}
		}
	}
}
