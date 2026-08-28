package evidence

import (
	"context"
	"errors"
	"strings"
)

// DeliverGoverned accepts the same protected delivery envelope as legacy
// invitations, but permits status-only CC messages that intentionally contain
// no response route. The legacy Deliver method remains unchanged.
func (service *InvitationDeliveryService) DeliverGoverned(ctx context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
	address := strings.TrimSpace(request.RecipientAddress)
	link := strings.TrimSpace(request.InvitationLink)
	subject := strings.TrimSpace(request.Subject)
	plainText := strings.TrimSpace(request.PlainText)
	htmlBody := strings.TrimSpace(request.HTML)
	if address == "" || subject == "" || (plainText == "" && htmlBody == "") {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryRequestInvalid
	}
	request.RecipientAddress = address
	request.InvitationLink = link
	request.Subject = subject

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
