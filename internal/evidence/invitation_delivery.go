package evidence

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type InvitationDeliveryStatus string

const (
	InvitationDelivered               InvitationDeliveryStatus = "DELIVERED"
	InvitationLinkCreatedEmailNotSent InvitationDeliveryStatus = "LINK_CREATED_EMAIL_NOT_SENT"
	InvitationDeliveryFailed          InvitationDeliveryStatus = "FAILED"
)

type InvitationDeliveryFailureCode string

const (
	InvitationFailureRecipientRejected InvitationDeliveryFailureCode = "RECIPIENT_REJECTED"
	InvitationFailureTemporary         InvitationDeliveryFailureCode = "TEMPORARY_FAILURE"
	InvitationFailurePermanent         InvitationDeliveryFailureCode = "PERMANENT_FAILURE"
	InvitationFailureProviderError     InvitationDeliveryFailureCode = "DELIVERY_PROVIDER_ERROR"
	InvitationFailureInvalidReceipt    InvitationDeliveryFailureCode = "INVALID_RECEIPT"
)

var (
	ErrInvitationDeliveryUnavailable    = errors.New("invitation delivery is unavailable")
	ErrInvitationDeliveryFailed         = errors.New("invitation delivery failed")
	ErrInvitationDeliveryReceiptUnsafe  = errors.New("invitation delivery receipt is unsafe")
	ErrInvitationDeliveryRequestInvalid = errors.New("invitation delivery request is invalid")
)

// InvitationDeliveryRequest contains protected values only for the duration of
// one synchronous adapter call. Adapters MUST NOT persist or log this value.
// JSON and ordinary formatted output deliberately omit the protected fields.
type InvitationDeliveryRequest struct {
	RecipientAddress string `json:"-"`
	InvitationLink   string `json:"-"`
}

func (InvitationDeliveryRequest) String() string {
	return "InvitationDeliveryRequest{protected}"
}

func (InvitationDeliveryRequest) GoString() string {
	return "InvitationDeliveryRequest{protected}"
}

// InvitationDelivery is an optional protected adapter invoked synchronously
// after an invitation and its one-time link have been created.
type InvitationDelivery interface {
	Deliver(context.Context, InvitationDeliveryRequest) (InvitationDeliveryReceipt, error)
}

// InvitationDeliveryReceipt is safe to retain. It contains no provider payload,
// raw recipient address, invitation token or one-time link.
type InvitationDeliveryReceipt struct {
	Status        InvitationDeliveryStatus      `json:"status"`
	RecipientHint string                        `json:"recipient_hint"`
	DeliveredAt   *time.Time                    `json:"delivered_at,omitempty"`
	FailureCode   InvitationDeliveryFailureCode `json:"failure_code,omitempty"`
}

type InvitationDeliveryService struct {
	adapter InvitationDelivery
}

func NewInvitationDeliveryService(adapter InvitationDelivery) *InvitationDeliveryService {
	return &InvitationDeliveryService{adapter: adapter}
}

func (service *InvitationDeliveryService) Deliver(ctx context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
	address := strings.TrimSpace(request.RecipientAddress)
	link := strings.TrimSpace(request.InvitationLink)
	if address == "" || link == "" {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryRequestInvalid
	}
	request.RecipientAddress = address
	request.InvitationLink = link

	hint := audienceHint(normalizeAudience(address))
	if service == nil || service.adapter == nil {
		return invitationFallbackReceipt(hint), nil
	}

	providerReceipt, err := service.adapter.Deliver(ctx, request)
	if errors.Is(err, ErrInvitationDeliveryUnavailable) {
		return invitationFallbackReceipt(hint), nil
	}
	if err != nil {
		return invitationFailureReceipt(hint, InvitationFailureProviderError), ErrInvitationDeliveryFailed
	}

	receipt, ok := normalizeInvitationDeliveryReceipt(providerReceipt, request, hint)
	if !ok {
		return invitationFailureReceipt(hint, InvitationFailureInvalidReceipt), ErrInvitationDeliveryReceiptUnsafe
	}
	return receipt, nil
}

func invitationFallbackReceipt(hint string) InvitationDeliveryReceipt {
	return InvitationDeliveryReceipt{
		Status:        InvitationLinkCreatedEmailNotSent,
		RecipientHint: hint,
	}
}

func invitationFailureReceipt(hint string, code InvitationDeliveryFailureCode) InvitationDeliveryReceipt {
	return InvitationDeliveryReceipt{
		Status:        InvitationDeliveryFailed,
		RecipientHint: hint,
		FailureCode:   code,
	}
}

func normalizeInvitationDeliveryReceipt(provider InvitationDeliveryReceipt, request InvitationDeliveryRequest, hint string) (InvitationDeliveryReceipt, bool) {
	if invitationReceiptContainsProtectedFailure(provider, request) {
		return InvitationDeliveryReceipt{}, false
	}

	switch provider.Status {
	case InvitationDelivered:
		if provider.DeliveredAt == nil || provider.DeliveredAt.IsZero() || provider.FailureCode != "" {
			return InvitationDeliveryReceipt{}, false
		}
		deliveredAt := provider.DeliveredAt.UTC()
		return InvitationDeliveryReceipt{
			Status:        InvitationDelivered,
			RecipientHint: hint,
			DeliveredAt:   &deliveredAt,
		}, true
	case InvitationLinkCreatedEmailNotSent:
		if provider.DeliveredAt != nil || provider.FailureCode != "" {
			return InvitationDeliveryReceipt{}, false
		}
		return invitationFallbackReceipt(hint), true
	case InvitationDeliveryFailed:
		if provider.DeliveredAt != nil || !validInvitationFailureCode(provider.FailureCode) {
			return InvitationDeliveryReceipt{}, false
		}
		return invitationFailureReceipt(hint, provider.FailureCode), true
	default:
		return InvitationDeliveryReceipt{}, false
	}
}

func validInvitationFailureCode(code InvitationDeliveryFailureCode) bool {
	switch code {
	case InvitationFailureRecipientRejected,
		InvitationFailureTemporary,
		InvitationFailurePermanent,
		InvitationFailureProviderError,
		InvitationFailureInvalidReceipt:
		return len(code) <= 64
	default:
		return false
	}
}

func invitationReceiptContainsProtectedFailure(receipt InvitationDeliveryReceipt, request InvitationDeliveryRequest) bool {
	value := strings.ToLower(string(receipt.FailureCode))
	if value == "" {
		return false
	}
	for _, protected := range invitationDeliveryProtectedValues(request) {
		protected = strings.ToLower(strings.TrimSpace(protected))
		if protected != "" && strings.Contains(value, protected) {
			return true
		}
	}
	return false
}

func invitationDeliveryProtectedValues(request InvitationDeliveryRequest) []string {
	values := []string{request.RecipientAddress, request.InvitationLink}
	parsed, err := url.Parse(request.InvitationLink)
	if err != nil {
		return values
	}
	for _, queryValues := range parsed.Query() {
		values = append(values, queryValues...)
	}
	if parsed.Fragment != "" {
		values = append(values, parsed.Fragment)
	}
	return values
}

var _ fmt.Stringer = InvitationDeliveryRequest{}
