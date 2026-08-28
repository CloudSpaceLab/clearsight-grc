package evidence

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const communicationAccessFragmentKey = "form_access"

func communicationActionForOutboxEvent(event workflowruntime.OutboxEvent) (CommunicationAction, bool) {
	if event.AggregateType != "FORM_DISTRIBUTION" {
		return "", false
	}
	switch event.EventType {
	case "FORM_DISTRIBUTION_OPEN":
		return CommunicationInvitation, true
	case "FORM_DISTRIBUTION_AMENDED":
		return CommunicationAmendment, true
	case "FORM_DISTRIBUTION_EXPIRED":
		return CommunicationExpired, true
	case "FORM_DISTRIBUTION_COMPLETED":
		return CommunicationCompletion, true
	case "FORM_DISTRIBUTION_CHANGE_REQUESTED":
		return CommunicationChangeRequested, true
	case "FORM_COMMUNICATION_REMINDER_DUE":
		var payload struct {
			Action CommunicationAction `json:"action"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || (payload.Action != CommunicationReminder && payload.Action != CommunicationDueSoon) {
			return "", false
		}
		return payload.Action, true
	default:
		return "", false
	}
}

func communicationActionDeliverable(distribution FormDistribution, action CommunicationAction, now time.Time) bool {
	now = now.UTC()
	switch action {
	case CommunicationInvitation, CommunicationReminder, CommunicationDueSoon, CommunicationChangeRequested, CommunicationAmendment:
		return distribution.Status == DistributionOpen && distribution.Deadline.After(now) && distribution.RouteExpiresAt.After(now)
	case CommunicationExpired:
		return distribution.Status == DistributionExpired || !distribution.Deadline.After(now)
	case CommunicationCompletion:
		return distribution.Status == DistributionCompleted
	default:
		return false
	}
}

func communicationRecipientDeliverable(recipient communicationDeliveryRecipient, action CommunicationAction) bool {
	if recipient.Type != RecipientExternalAudience || recipient.State == DistributionRecipientRevoked {
		return false
	}
	if recipient.State == DistributionRecipientCompleted {
		switch action {
		case CommunicationInvitation, CommunicationReminder, CommunicationDueSoon:
			return false
		}
	}
	return recipient.Role == RecipientTo || recipient.Role == RecipientCC
}

func communicationRecipientName(recipient communicationDeliveryRecipient) string {
	if value := strings.TrimSpace(recipient.ContactLabel); value != "" {
		return value
	}
	if value := strings.TrimSpace(recipient.AudienceHint); value != "" {
		return value
	}
	return "Recipient"
}

func communicationAccessInstructions(policy AccessPolicy, role RecipientRole) string {
	if role == RecipientCC {
		return "This is a status copy. No response access is included."
	}
	switch policy {
	case AccessDirectMagicLink:
		return "Open the secure form using the link in this message."
	case AccessSharedEmailOTP:
		return "Open the secure form, select your masked email, then verify the one-time code sent to you."
	case AccessDirectEmailOTP:
		return "Open your personal secure link, then verify the one-time code sent to your email."
	default:
		return "Follow the secure access instructions in this message."
	}
}

func validateCommunicationCaptureBaseURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ErrCommunicationInvalid
	}
	return nil
}

func buildCommunicationAccessLink(baseURL, selector string) (string, error) {
	if validateCommunicationCaptureBaseURL(baseURL) != nil || strings.TrimSpace(selector) == "" || selector != strings.TrimSpace(selector) {
		return "", ErrCommunicationInvalid
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", ErrCommunicationInvalid
	}
	fragment := url.Values{}
	fragment.Set(communicationAccessFragmentKey, selector)
	parsed.Fragment = fragment.Encode()
	return parsed.String(), nil
}

func renderCommunicationWithoutResponseRoute(template CommunicationTemplate, context CommunicationContext) (RenderedMessage, error) {
	if err := ValidateCommunicationTemplate(template); err != nil {
		return RenderedMessage{}, err
	}
	values := communicationContextValues(context)
	values["secure_form_link"] = ""
	subjectTemplate := strings.ReplaceAll(template.SubjectTemplate, "{{secure_form_link}}", "")
	subject, err := expandCommunicationTemplate(subjectTemplate, values)
	if err != nil || strings.TrimSpace(subject) == "" || strings.ContainsAny(subject, "\r\n") {
		return RenderedMessage{}, ErrCommunicationInvalid
	}
	plainParts := make([]string, 0, len(template.Document))
	htmlParts := make([]string, 0, len(template.Document))
	for _, original := range template.Document {
		node, ok := communicationNodeWithoutResponseRoute(original)
		if !ok {
			continue
		}
		plain, markup, renderErr := renderCommunicationNode(node, values)
		if renderErr != nil {
			return RenderedMessage{}, renderErr
		}
		if plain != "" {
			plainParts = append(plainParts, plain)
		}
		if markup != "" {
			htmlParts = append(htmlParts, markup)
		}
	}
	if len(plainParts) == 0 && len(htmlParts) == 0 {
		return RenderedMessage{}, fmt.Errorf("%w: CC copy has no status content after response-route removal", ErrCommunicationInvalid)
	}
	return RenderedMessage{
		Subject:   protectedString{value: strings.TrimSpace(subject)},
		PlainText: protectedString{value: strings.Join(plainParts, "\n\n")},
		HTML:      protectedString{value: strings.Join(htmlParts, "\n")},
	}, nil
}

func communicationNodeWithoutResponseRoute(node CommunicationNode) (CommunicationNode, bool) {
	usesSecureLink, err := validateCommunicationNode(node)
	if err != nil {
		return CommunicationNode{}, false
	}
	if !usesSecureLink {
		return node, true
	}
	if strings.EqualFold(strings.TrimSpace(node.Type), "list") {
		filtered := node
		filtered.Items = filtered.Items[:0]
		for _, item := range node.Items {
			if !containsPlaceholder(item, "secure_form_link") {
				filtered.Items = append(filtered.Items, item)
			}
		}
		return filtered, len(filtered.Items) != 0
	}
	return CommunicationNode{}, false
}

func communicationPlainPreview(value protectedString) string { return html.UnescapeString(value.value) }
