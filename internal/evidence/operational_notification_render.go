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
	BankName        string
	RecipientName   string
	MatterTitle     string
	WorkTitle       string
	Responsibility  string
	DueAt           time.Time
	IssueURL        string
	UpdateRequested bool
	UpdateMessage   string
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
	context.UpdateMessage = strings.TrimSpace(context.UpdateMessage)
	values := []string{context.BankName, context.RecipientName, context.MatterTitle, context.WorkTitle, context.Responsibility, context.IssueURL}
	if context.UpdateRequested {
		values = append(values, context.UpdateMessage)
	}
	for _, value := range values {
		if value == "" || strings.ContainsAny(value, "\r\n\x00") {
			return RenderedMessage{}, errEmailPresentationInvalid
		}
	}

	responsibility := operationalResponsibilityLabel(context.Responsibility)
	intro := fmt.Sprintf("%s, you have been assigned issue work as the %s.", context.RecipientName, strings.ToLower(responsibility))
	bodyPlain := "Next action: " + context.WorkTitle + ".\n\nOpen the issue to review its current facts, evidence and permitted actions. Completing assigned work does not authorize, approve or sign off the issue."
	bodyHTML := `<p style="margin:0 0 12px;">Next action: <strong>` + html.EscapeString(context.WorkTitle) + `</strong>.</p>` +
		`<p style="margin:0;">Open the issue to review its current facts, evidence and permitted actions. Completing assigned work does not authorize, approve or sign off the issue.</p>`
	preheader, actionLabel := "Assigned issue work: "+context.WorkTitle, "Open assigned issue"
	if context.UpdateRequested {
		intro = fmt.Sprintf("%s, a status update is requested for %s.", context.RecipientName, context.WorkTitle)
		bodyPlain = "Status update requested.\n\n" + context.UpdateMessage + "\n\nOpen the issue and add your current status. This request does not complete the action or close the issue."
		bodyHTML = `<p style="margin:0 0 12px;"><strong>Status update requested.</strong></p><p style="margin:0 0 12px;">` + html.EscapeString(context.UpdateMessage) + `</p><p style="margin:0;">Open the issue and add your current status. This request does not complete the action or close the issue.</p>`
		preheader, actionLabel = "Status update requested: "+context.WorkTitle, "Open issue"
	}
	facts := []emailFact{{Label: "Responsibility", Value: responsibility}}
	if !context.DueAt.IsZero() {
		facts = append(facts, emailFact{Label: "Due", Value: context.DueAt.UTC().Format("2 Jan 2006, 15:04 UTC")})
	}
	presentation, err := renderEmailPresentation(emailPresentationInput{
		BrandName: context.BankName, Preheader: preheader,
		Heading: context.MatterTitle, Intro: intro, BodyPlain: bodyPlain, BodyHTML: bodyHTML,
		ActionLabel: actionLabel, ActionURL: context.IssueURL, Facts: facts,
	})
	if err != nil {
		return RenderedMessage{}, err
	}
	return RenderedMessage{
		Subject:   protectedString{value: map[bool]string{true: "Status update requested: ", false: "Assigned issue work: "}[context.UpdateRequested] + context.MatterTitle},
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
