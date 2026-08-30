package evidence

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const protectedRedaction = "[REDACTED]"

type protectedString struct{ value string }

func (protectedString) String() string   { return protectedRedaction }
func (protectedString) GoString() string { return protectedRedaction }

type RenderedMessage struct {
	Subject   protectedString
	PlainText protectedString
	HTML      protectedString
}

func (RenderedMessage) String() string   { return "RenderedMessage{protected}" }
func (RenderedMessage) GoString() string { return "RenderedMessage{protected}" }

type RenderedMessagePreview struct {
	Subject   string `json:"subject"`
	PlainText string `json:"plain_text"`
	HTML      string `json:"html"`
}

type CommunicationContext struct {
	RecipientName      string
	BankName           string
	FormTitle          string
	TaskSummary        string
	DueTime            string
	LinkExpiry         string
	AccessInstructions string
	SupportContact     string
	SecureFormLink     protectedString
}

var communicationPlaceholderPattern = regexp.MustCompile(`\{\{([a-z_]+)\}\}`)
var communicationDigestPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

var allowedCommunicationPlaceholders = map[string]struct{}{
	"recipient_name":      {},
	"bank_name":           {},
	"form_title":          {},
	"task_summary":        {},
	"due_time":            {},
	"link_expiry":         {},
	"access_instructions": {},
	"support_contact":     {},
	"secure_form_link":    {},
}

func ProtectCommunicationString(value string) protectedString { return protectedString{value: value} }

func ValidateCommunicationTemplate(template CommunicationTemplate) error {
	if !validCommunicationAction(template.Action) || normalizeCommunicationLocale(template.Locale) == "" {
		return ErrCommunicationInvalid
	}
	subject := strings.TrimSpace(template.SubjectTemplate)
	if subject == "" || len(subject) > 200 || strings.ContainsAny(subject, "\r\n") || containsRawHTML(subject) {
		return fmt.Errorf("%w: communication subject is invalid", ErrCommunicationInvalid)
	}
	if len(template.Document) == 0 || len(template.Document) > 100 {
		return fmt.Errorf("%w: communication document must contain 1-100 nodes", ErrCommunicationInvalid)
	}
	secureLinkReferenced := containsPlaceholder(subject, "secure_form_link")
	if err := validateCommunicationPlaceholders(subject); err != nil {
		return err
	}
	for _, node := range template.Document {
		usesSecureLink, err := validateCommunicationNode(node)
		if err != nil {
			return err
		}
		secureLinkReferenced = secureLinkReferenced || usesSecureLink
	}
	if communicationRequiresSecureLink(template.Action) && !secureLinkReferenced {
		return fmt.Errorf("%w: %s copy must include the secure_form_link placeholder", ErrCommunicationInvalid, template.Action)
	}
	return nil
}

func ValidateCommunicationLogo(input BrandAssetInput) error {
	artifactKey := strings.TrimSpace(input.ArtifactKey)
	altText := strings.TrimSpace(input.AltText)
	if artifactKey == "" || len(artifactKey) > 1024 || strings.Contains(artifactKey, "://") || strings.HasPrefix(artifactKey, "/") || strings.Contains(artifactKey, "..") {
		return fmt.Errorf("%w: logo artifact key is invalid", ErrCommunicationInvalid)
	}
	if !communicationDigestPattern.MatchString(strings.TrimSpace(input.DigestHex)) || input.MediaType != "image/png" || input.Width < 1 || input.Width > 4096 || input.Height < 1 || input.Height > 4096 || input.SizeBytes < 1 || input.SizeBytes > 524288 || altText == "" || len(altText) > 160 {
		return fmt.Errorf("%w: logo must be an inspected bounded PNG with alt text", ErrCommunicationInvalid)
	}
	return nil
}

func RenderCommunication(template CommunicationTemplate, context CommunicationContext) (RenderedMessage, error) {
	if err := ValidateCommunicationTemplate(template); err != nil {
		return RenderedMessage{}, err
	}
	values := communicationContextValues(context)
	if communicationRequiresSecureLink(template.Action) {
		link := strings.TrimSpace(values["secure_form_link"])
		if err := validateRenderedLink(link); err != nil {
			return RenderedMessage{}, fmt.Errorf("%w: secure form link is unavailable", ErrCommunicationInvalid)
		}
	}
	subject, err := expandCommunicationTemplate(template.SubjectTemplate, values)
	if err != nil || strings.ContainsAny(subject, "\r\n") {
		return RenderedMessage{}, ErrCommunicationInvalid
	}
	plainParts := make([]string, 0, len(template.Document))
	htmlParts := make([]string, 0, len(template.Document))
	actionLabel, actionURL := "", ""
	for _, node := range template.Document {
		plain, markup, err := renderCommunicationNode(node, values)
		if err != nil {
			return RenderedMessage{}, err
		}
		if strings.EqualFold(strings.TrimSpace(node.Type), "primary-action") {
			actionLabel, err = expandCommunicationTemplate(node.Text, values)
			if err != nil {
				return RenderedMessage{}, err
			}
			actionURL, err = expandCommunicationTemplate(node.Href, values)
			if err != nil {
				return RenderedMessage{}, err
			}
			continue
		}
		if plain != "" {
			plainParts = append(plainParts, plain)
		}
		if markup != "" {
			htmlParts = append(htmlParts, markup)
		}
	}
	facts := make([]emailFact, 0, 2)
	if strings.TrimSpace(context.DueTime) != "" {
		facts = append(facts, emailFact{Label: "Due", Value: context.DueTime})
	}
	if strings.TrimSpace(context.LinkExpiry) != "" {
		facts = append(facts, emailFact{Label: "Link expires", Value: context.LinkExpiry})
	}
	presentation, err := renderEmailPresentation(emailPresentationInput{
		BrandName: context.BankName, Preheader: context.TaskSummary, Heading: context.FormTitle,
		BodyPlain: strings.Join(plainParts, "\n\n"), BodyHTML: strings.Join(htmlParts, "\n"),
		ActionLabel: actionLabel, ActionURL: actionURL, Facts: facts, SupportContact: context.SupportContact,
	})
	if err != nil {
		return RenderedMessage{}, ErrCommunicationInvalid
	}
	return RenderedMessage{
		Subject:   protectedString{value: subject},
		PlainText: protectedString{value: presentation.PlainText},
		HTML:      protectedString{value: presentation.HTML},
	}, nil
}

func SampleCommunicationContext() CommunicationContext {
	return CommunicationContext{
		RecipientName: "[Sample recipient]", BankName: "[Sample bank]", FormTitle: "[Sample form]",
		TaskSummary: "[Sample task summary]", DueTime: "[Sample due time]", LinkExpiry: "[Sample link expiry]",
		AccessInstructions: "[Sample access instructions]", SupportContact: "[Sample support contact]",
		SecureFormLink: protectedString{value: "https://forms.example.invalid/sample-secure-link"},
	}
}

func revealPreview(message RenderedMessage) RenderedMessagePreview {
	return RenderedMessagePreview{Subject: message.Subject.value, PlainText: message.PlainText.value, HTML: message.HTML.value}
}

func communicationContextValues(context CommunicationContext) map[string]string {
	return map[string]string{
		"recipient_name":      context.RecipientName,
		"bank_name":           context.BankName,
		"form_title":          context.FormTitle,
		"task_summary":        context.TaskSummary,
		"due_time":            context.DueTime,
		"link_expiry":         context.LinkExpiry,
		"access_instructions": context.AccessInstructions,
		"support_contact":     context.SupportContact,
		"secure_form_link":    context.SecureFormLink.value,
	}
}

func validateCommunicationNode(node CommunicationNode) (bool, error) {
	nodeType := strings.TrimSpace(strings.ToLower(node.Type))
	text := strings.TrimSpace(node.Text)
	href := strings.TrimSpace(node.Href)
	usesSecureLink := false
	switch nodeType {
	case "paragraph", "strong", "emphasis", "callout":
		if text == "" || len(text) > 4000 || href != "" || node.Level != 0 || len(node.Items) != 0 || containsRawHTML(text) {
			return false, ErrCommunicationInvalid
		}
	case "heading":
		if text == "" || len(text) > 500 || href != "" || node.Level < 1 || node.Level > 3 || len(node.Items) != 0 || containsRawHTML(text) {
			return false, ErrCommunicationInvalid
		}
	case "link", "primary-action":
		if text == "" || len(text) > 500 || href == "" || node.Level != 0 || len(node.Items) != 0 || containsRawHTML(text) {
			return false, ErrCommunicationInvalid
		}
		if err := validateCommunicationHrefTemplate(href); err != nil {
			return false, err
		}
		usesSecureLink = containsPlaceholder(href, "secure_form_link")
	case "list":
		if text != "" || href != "" || node.Level != 0 || len(node.Items) == 0 || len(node.Items) > 20 {
			return false, ErrCommunicationInvalid
		}
		for _, item := range node.Items {
			if strings.TrimSpace(item) == "" || len(item) > 1000 || containsRawHTML(item) {
				return false, ErrCommunicationInvalid
			}
			if err := validateCommunicationPlaceholders(item); err != nil {
				return false, err
			}
			usesSecureLink = usesSecureLink || containsPlaceholder(item, "secure_form_link")
		}
	case "divider":
		if text != "" || href != "" || node.Level != 0 || len(node.Items) != 0 {
			return false, ErrCommunicationInvalid
		}
	default:
		return false, fmt.Errorf("%w: unsupported communication node %q", ErrCommunicationInvalid, node.Type)
	}
	if text != "" {
		if err := validateCommunicationPlaceholders(text); err != nil {
			return false, err
		}
		usesSecureLink = usesSecureLink || containsPlaceholder(text, "secure_form_link")
	}
	return usesSecureLink, nil
}

func validateCommunicationPlaceholders(value string) error {
	matches := communicationPlaceholderPattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if _, ok := allowedCommunicationPlaceholders[match[1]]; !ok {
			return fmt.Errorf("%w: unsupported placeholder %q", ErrCommunicationInvalid, match[1])
		}
	}
	withoutKnown := communicationPlaceholderPattern.ReplaceAllString(value, "")
	if strings.Contains(withoutKnown, "{{") || strings.Contains(withoutKnown, "}}") {
		return fmt.Errorf("%w: malformed communication placeholder", ErrCommunicationInvalid)
	}
	return nil
}

func validateCommunicationHrefTemplate(value string) error {
	if err := validateCommunicationPlaceholders(value); err != nil {
		return err
	}
	if containsPlaceholder(value, "secure_form_link") {
		if strings.TrimSpace(value) != "{{secure_form_link}}" {
			return fmt.Errorf("%w: secure link placeholder must occupy the complete href", ErrCommunicationInvalid)
		}
		return nil
	}
	return validateRenderedLink(value)
}

func validateRenderedLink(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return ErrCommunicationInvalid
	}
	return nil
}

func renderCommunicationNode(node CommunicationNode, values map[string]string) (string, string, error) {
	typ := strings.ToLower(strings.TrimSpace(node.Type))
	expand := func(value string) (string, error) { return expandCommunicationTemplate(value, values) }
	switch typ {
	case "divider":
		return "---", `<hr aria-hidden="true">`, nil
	case "list":
		plain := make([]string, 0, len(node.Items))
		markup := make([]string, 0, len(node.Items))
		for _, item := range node.Items {
			value, err := expand(item)
			if err != nil {
				return "", "", err
			}
			plain = append(plain, "- "+value)
			markup = append(markup, "<li>"+html.EscapeString(value)+"</li>")
		}
		return strings.Join(plain, "\n"), "<ul>" + strings.Join(markup, "") + "</ul>", nil
	case "link", "primary-action":
		text, err := expand(node.Text)
		if err != nil {
			return "", "", err
		}
		href, err := expand(node.Href)
		if err != nil || validateRenderedLink(href) != nil {
			return "", "", ErrCommunicationInvalid
		}
		class := ""
		if typ == "primary-action" {
			class = ` class="primary-action"`
		}
		return text + ": " + href, `<a` + class + ` href="` + html.EscapeString(href) + `">` + html.EscapeString(text) + `</a>`, nil
	default:
		text, err := expand(node.Text)
		if err != nil {
			return "", "", err
		}
		escaped := html.EscapeString(text)
		switch typ {
		case "paragraph":
			return text, "<p>" + escaped + "</p>", nil
		case "heading":
			tag := "h" + strconv.Itoa(node.Level)
			return text, "<" + tag + ">" + escaped + "</" + tag + ">", nil
		case "strong":
			return text, "<p><strong>" + escaped + "</strong></p>", nil
		case "emphasis":
			return text, "<p><em>" + escaped + "</em></p>", nil
		case "callout":
			return text, `<aside role="note">` + escaped + `</aside>`, nil
		default:
			return "", "", ErrCommunicationInvalid
		}
	}
}

func expandCommunicationTemplate(value string, values map[string]string) (string, error) {
	if err := validateCommunicationPlaceholders(value); err != nil {
		return "", err
	}
	var expansionErr error
	result := communicationPlaceholderPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := communicationPlaceholderPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			expansionErr = ErrCommunicationInvalid
			return ""
		}
		resolved, ok := values[parts[1]]
		if !ok {
			expansionErr = ErrCommunicationInvalid
			return ""
		}
		return resolved
	})
	return result, expansionErr
}

func containsPlaceholder(value, placeholder string) bool {
	return strings.Contains(value, "{{"+placeholder+"}}")
}

func containsRawHTML(value string) bool {
	return strings.Contains(value, "<") || strings.Contains(value, ">")
}

func communicationRequiresSecureLink(action CommunicationAction) bool {
	switch action {
	case CommunicationInvitation, CommunicationReminder, CommunicationDueSoon, CommunicationChangeRequested, CommunicationAmendment:
		return true
	default:
		return false
	}
}
