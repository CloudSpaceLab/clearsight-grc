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
// Subject, PlainText and HTML are optional so legacy invitation callers remain
// compatible; governed communication delivery supplies all three.
type InvitationDeliveryRequest struct {
	RecipientAddress string `json:"-"`
	InvitationLink   string `json:"-"`
	Subject          string `json:"-"`
	PlainText        string `json:"-"`
	HTML             string `json:"-"`
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
// raw recipient address, invitation token, message body or one-time link.
type InvitationDeliveryReceipt struct {
	Status            InvitationDeliveryStatus      `json:"status"`
	RecipientHint     string                        `json:"recipient_hint"`
	ProviderMessageID string                        `json:"provider_message_id,omitempty"`
	DeliveredAt       *time.Time                    `json:"delivered_at,omitempty"`
	FailureCode       InvitationDeliveryFailureCode `json:"failure_code,omitempty"`
}

type InvitationDeliveryService struct {
	adapter InvitationDelivery
}

func NewInvitationDeliveryService(adapter InvitationDelivery) *InvitationDeliveryService {
	return &InvitationDeliveryService{adapter: adapter}
}

// Deliver preserves the legacy invitation contract: both an address and a
// one-time invitation link are required.
func (service *InvitationDeliveryService) Deliver(ctx context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
	if strings.TrimSpace(request.InvitationLink) == "" {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryRequestInvalid
	}
	return service.deliver(ctx, request, false)
}

// DeliverGoverned sends fully rendered governed communication. A response link
// is optional so CC/status messages can never be forced to carry responder
// capability material.
func (service *InvitationDeliveryService) DeliverGoverned(ctx context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
	return service.deliver(ctx, request, true)
}

func (service *InvitationDeliveryService) deliver(ctx context.Context, request InvitationDeliveryRequest, governed bool) (InvitationDeliveryReceipt, error) {
	address := strings.TrimSpace(request.RecipientAddress)
	link := strings.TrimSpace(request.InvitationLink)
	request.Subject = strings.TrimSpace(request.Subject)
	request.PlainText = strings.TrimSpace(request.PlainText)
	request.HTML = strings.TrimSpace(request.HTML)
	if address == "" || (!governed && link == "") {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryRequestInvalid
	}
	if governed && (request.Subject == "" || (request.PlainText == "" && request.HTML == "")) {
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
	if invitationReceiptContainsProtectedValue(provider, request) {
		return InvitationDeliveryReceipt{}, false
	}
	messageID := strings.TrimSpace(provider.ProviderMessageID)
	if len(messageID) > 512 || strings.ContainsAny(messageID, "\r\n") {
		return InvitationDeliveryReceipt{}, false
	}

	switch provider.Status {
	case InvitationDelivered:
		if provider.DeliveredAt == nil || provider.DeliveredAt.IsZero() || provider.FailureCode != "" {
			return InvitationDeliveryReceipt{}, false
		}
		deliveredAt := provider.DeliveredAt.UTC()
		return InvitationDeliveryReceipt{
			Status:            InvitationDelivered,
			RecipientHint:     hint,
			ProviderMessageID: messageID,
			DeliveredAt:       &deliveredAt,
		}, true
	case InvitationLinkCreatedEmailNotSent:
		if provider.DeliveredAt != nil || provider.FailureCode != "" || messageID != "" {
			return InvitationDeliveryReceipt{}, false
		}
		return invitationFallbackReceipt(hint), true
	case InvitationDeliveryFailed:
		if provider.DeliveredAt != nil || !validInvitationFailureCode(provider.FailureCode) || messageID != "" {
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

func invitationReceiptContainsProtectedValue(receipt InvitationDeliveryReceipt, request InvitationDeliveryRequest) bool {
	for _, candidate := range []string{string(receipt.FailureCode), receipt.ProviderMessageID} {
		candidate = strings.ToLower(candidate)
		if candidate == "" {
			continue
		}
		for _, protected := range invitationDeliveryProtectedValues(request) {
			protected = strings.ToLower(strings.TrimSpace(protected))
			if protected != "" && strings.Contains(candidate, protected) {
				return true
			}
		}
	}
	return false
}

func invitationDeliveryProtectedValues(request InvitationDeliveryRequest) []string {
	values := []string{request.RecipientAddress, request.InvitationLink, request.Subject, request.PlainText, request.HTML}
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
