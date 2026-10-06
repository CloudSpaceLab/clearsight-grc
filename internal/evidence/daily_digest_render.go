package evidence

import (
	"fmt"
	"net/url"
	"strings"
)

type DailyDigestContext struct {
	BrandName       string
	RecipientName   string
	HomeURL         string
	MaterialChanges int
	AssignedWork    int
	DueSoon         int
	Worsened        int
	Cleared         int
}

func BuildDailyDigestRequest(recipientAddress string, context DailyDigestContext) (InvitationDeliveryRequest, error) {
	recipientAddress = strings.TrimSpace(recipientAddress)
	context.BrandName = strings.TrimSpace(context.BrandName)
	context.RecipientName = strings.TrimSpace(context.RecipientName)
	context.HomeURL = strings.TrimSpace(context.HomeURL)
	if recipientAddress == "" ||
		invalidAttentionLabel(context.BrandName, 120) ||
		invalidAttentionLabel(context.RecipientName, 160) ||
		context.MaterialChanges < 0 || context.AssignedWork < 0 || context.DueSoon < 0 ||
		context.Worsened < 0 || context.Cleared < 0 {
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}
	parsed, err := url.Parse(context.HomeURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}

	facts := []emailFact{
		{Label: "Changes", Value: fmt.Sprintf("%d", context.MaterialChanges)},
		{Label: "Assigned work", Value: fmt.Sprintf("%d", context.AssignedWork)},
		{Label: "Due in 7 days", Value: fmt.Sprintf("%d", context.DueSoon)},
		{Label: "Worsened", Value: fmt.Sprintf("%d", context.Worsened)},
		{Label: "Cleared", Value: fmt.Sprintf("%d", context.Cleared)},
	}
	presentation, err := renderEmailPresentation(emailPresentationInput{
		BrandName:   context.BrandName,
		Preheader:   "Daily risk and compliance summary",
		Heading:     "Daily summary",
		Intro:       context.RecipientName + ", here is your current ClearSight summary.",
		BodyPlain:   "Open ClearSight for the authorized records and current work. This email contains counts only.",
		BodyHTML:    "<p style=\"margin:0;\">Open ClearSight for the authorized records and current work. This email contains counts only.</p>",
		ActionLabel: "Open ClearSight",
		ActionURL:   context.HomeURL,
		Facts:       facts,
	})
	if err != nil {
		return InvitationDeliveryRequest{}, err
	}
	return InvitationDeliveryRequest{
		RecipientAddress: recipientAddress,
		InvitationLink:   context.HomeURL,
		Subject:          "ClearSight daily summary",
		PlainText:        presentation.PlainText,
		HTML:             presentation.HTML,
	}, nil
}
