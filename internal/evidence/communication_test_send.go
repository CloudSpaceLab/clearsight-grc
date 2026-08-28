package evidence

import (
	"context"
	"strings"
)

func (service *CommunicationService) TestSend(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64, address string, delivery *InvitationDeliveryService) (InvitationDeliveryReceipt, error) {
	if service == nil || delivery == nil || strings.TrimSpace(address) == "" {
		return InvitationDeliveryReceipt{}, ErrCommunicationUnavailable
	}
	template, err := service.GetTemplate(ctx, tenantID, legalEntityID, action, locale, version)
	if err != nil {
		return InvitationDeliveryReceipt{}, err
	}
	message, err := RenderCommunication(template, SampleCommunicationContext())
	if err != nil {
		return InvitationDeliveryReceipt{}, err
	}
	contextValues := communicationContextValues(SampleCommunicationContext())
	return delivery.DeliverGoverned(ctx, InvitationDeliveryRequest{
		RecipientAddress: strings.TrimSpace(address),
		InvitationLink:   contextValues["secure_form_link"],
		Subject:          message.Subject.value,
		PlainText:        message.PlainText.value,
		HTML:             message.HTML.value,
	})
}
