package evidence

import (
	"fmt"
	"net/url"
	"strings"
)

type AttentionNotificationContext struct {
	BrandName       string
	RecipientName   string
	ConditionLabel  string
	TransitionLabel string
	RecordURL       string
}

func BuildAttentionNotificationRequest(recipientAddress string, context AttentionNotificationContext) (InvitationDeliveryRequest, error) {
	recipientAddress = strings.TrimSpace(recipientAddress)
	context.BrandName = strings.TrimSpace(context.BrandName)
	context.RecipientName = strings.TrimSpace(context.RecipientName)
	context.ConditionLabel = strings.TrimSpace(context.ConditionLabel)
	context.TransitionLabel = strings.TrimSpace(context.TransitionLabel)
	context.RecordURL = strings.TrimSpace(context.RecordURL)
	if recipientAddress == "" ||
		invalidAttentionLabel(context.BrandName, 120) ||
		invalidAttentionLabel(context.RecipientName, 160) ||
		invalidAttentionLabel(context.ConditionLabel, 80) ||
		invalidAttentionLabel(context.TransitionLabel, 40) {
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}
	parsed, err := url.Parse(context.RecordURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment == "" {
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}

	heading := "Critical " + context.ConditionLabel + " change"
	intro := fmt.Sprintf("%s, a critical governed condition was %s.", context.RecipientName, strings.ToLower(context.TransitionLabel))
	presentation, err := renderEmailPresentation(emailPresentationInput{
		BrandName:   context.BrandName,
		Preheader:   heading,
		Heading:     heading,
		Intro:       intro,
		BodyPlain:   "Open ClearSight to review the current authorized record and required work. This email does not grant access or change ownership.",
		BodyHTML:    "<p style=\"margin:0;\">Open ClearSight to review the current authorized record and required work. This email does not grant access or change ownership.</p>",
		ActionLabel: "Open record",
		ActionURL:   context.RecordURL,
		Facts: []emailFact{
			{Label: "Condition", Value: context.ConditionLabel},
			{Label: "Change", Value: context.TransitionLabel},
		},
	})
	if err != nil {
		return InvitationDeliveryRequest{}, err
	}
	subject := "Critical change: " + context.ConditionLabel
	if len(subject) > 200 || strings.ContainsAny(subject, "\r\n") {
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}
	return InvitationDeliveryRequest{
		RecipientAddress: recipientAddress,
		InvitationLink:   context.RecordURL,
		Subject:          subject,
		PlainText:        presentation.PlainText,
		HTML:             presentation.HTML,
	}, nil
}

func invalidAttentionLabel(value string, max int) bool {
	return value == "" || len(value) > max || strings.ContainsAny(value, "\r\n\x00")
}
