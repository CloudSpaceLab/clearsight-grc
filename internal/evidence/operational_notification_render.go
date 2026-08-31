package evidence

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"
)

// OperationalNotificationContext contains protected message-time values. It
// must not be persisted or logged as a formatted value.
type OperationalNotificationContext struct {
	BankName       string
	RecipientName  string
	MatterTitle    string
	WorkTitle      string
	Responsibility string
	DueAt          time.Time
	IssueURL       string
}

func (OperationalNotificationContext) String() string {
	return "OperationalNotificationContext{protected}"
}
func (OperationalNotificationContext) GoString() string {
	return "OperationalNotificationContext{protected}"
}

func RenderOperationalNotification(context OperationalNotificationContext) (RenderedMessage, error) {
	context.BankName = strings.TrimSpace(context.BankName)
	context.RecipientName = strings.TrimSpace(context.RecipientName)
	context.MatterTitle = strings.TrimSpace(context.MatterTitle)
	context.WorkTitle = strings.TrimSpace(context.WorkTitle)
	context.Responsibility = strings.ToUpper(strings.TrimSpace(context.Responsibility))
	context.IssueURL = strings.TrimSpace(context.IssueURL)
	parsed, err := url.Parse(context.IssueURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return RenderedMessage{}, errEmailPresentationInvalid
	}
	for _, value := range []string{context.BankName, context.RecipientName, context.MatterTitle, context.WorkTitle, context.Responsibility, context.IssueURL} {
		if value == "" || strings.ContainsAny(value, "\r\n\x00") {
			return RenderedMessage{}, errEmailPresentationInvalid
		}
	}

	responsibility := operationalResponsibilityLabel(context.Responsibility)
	intro := fmt.Sprintf("%s, you have been assigned issue work as the %s.", context.RecipientName, strings.ToLower(responsibility))
	bodyPlain := "What needs to happen next: " + context.WorkTitle + ".\n\nOpen the issue to review its current facts, evidence and permitted actions. Completing assigned work does not authorize, approve or sign off the issue."
	bodyHTML := `<p style="margin:0 0 12px;">What needs to happen next: <strong>` + html.EscapeString(context.WorkTitle) + `</strong>.</p>` +
		`<p style="margin:0;">Open the issue to review its current facts, evidence and permitted actions. Completing assigned work does not authorize, approve or sign off the issue.</p>`
	facts := []emailFact{{Label: "Responsibility", Value: responsibility}}
	if !context.DueAt.IsZero() {
		facts = append(facts, emailFact{Label: "Due", Value: context.DueAt.UTC().Format("2 Jan 2006, 15:04 UTC")})
	}
	presentation, err := renderEmailPresentation(emailPresentationInput{
		BrandName: context.BankName, Preheader: "Assigned issue work: " + context.WorkTitle,
		Heading: context.MatterTitle, Intro: intro, BodyPlain: bodyPlain, BodyHTML: bodyHTML,
		ActionLabel: "Open assigned issue", ActionURL: context.IssueURL, Facts: facts,
	})
	if err != nil {
		return RenderedMessage{}, err
	}
	return RenderedMessage{
		Subject:   protectedString{value: "Assigned issue work: " + context.MatterTitle},
		PlainText: protectedString{value: presentation.PlainText}, HTML: protectedString{value: presentation.HTML},
	}, nil
}

func BuildOperationalNotificationRequest(recipientAddress string, context OperationalNotificationContext) (InvitationDeliveryRequest, error) {
	recipientAddress = strings.TrimSpace(recipientAddress)
	if recipientAddress == "" {
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}
	message, err := RenderOperationalNotification(context)
	if err != nil {
		return InvitationDeliveryRequest{}, err
	}
	return InvitationDeliveryRequest{
		RecipientAddress: recipientAddress,
		InvitationLink:   context.IssueURL,
		Subject:          message.Subject.value,
		PlainText:        message.PlainText.value,
		HTML:             message.HTML.value,
	}, nil
}

func operationalResponsibilityLabel(value string) string {
	switch value {
	case "ACCOUNTABLE_OWNER":
		return "Accountable owner"
	case "PERFORMER":
		return "Assigned performer"
	default:
		return strings.ToLower(strings.ReplaceAll(value, "_", " "))
	}
}
