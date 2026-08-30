package evidence

import (
	"context"
	"errors"
	"fmt"
	"html"
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
	RecipientAddress string                   `json:"-"`
	InvitationLink   string                   `json:"-"`
	Subject          string                   `json:"-"`
	PlainText        string                   `json:"-"`
	HTML             string                   `json:"-"`
	Message          InvitationMessageContext `json:"-"`
}

type InvitationMessageKind string

const (
	InvitationMessageGeneric              InvitationMessageKind = "FORM_REQUEST"
	InvitationMessageVendorRegistration   InvitationMessageKind = "VENDOR_REGISTRATION"
	InvitationMessageAddressVerification  InvitationMessageKind = "ADDRESS_VERIFICATION"
	InvitationMessageCertificationRefresh InvitationMessageKind = "CERTIFICATION_REFRESH"
)

type InvitationMessageContext struct {
	Kind           InvitationMessageKind
	BankName       string
	TaskTitle      string
	TaskSummary    string
	RecipientRole  string
	SupportContact string
	DueAt          time.Time
	ExpiresAt      time.Time
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

func (service *InvitationDeliveryService) Deliver(ctx context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
	address := strings.TrimSpace(request.RecipientAddress)
	link := strings.TrimSpace(request.InvitationLink)
	if address == "" || link == "" {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryRequestInvalid
	}
	request.RecipientAddress = address
	request.InvitationLink = link
	if strings.TrimSpace(request.Subject) == "" && strings.TrimSpace(request.PlainText) == "" && strings.TrimSpace(request.HTML) == "" {
		var err error
		request, err = renderDefaultInvitationMessage(request)
		if err != nil {
			return InvitationDeliveryReceipt{}, ErrInvitationDeliveryRequestInvalid
		}
	}

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

func renderDefaultInvitationMessage(request InvitationDeliveryRequest) (InvitationDeliveryRequest, error) {
	message := request.Message
	message.Kind = InvitationMessageKind(strings.ToUpper(strings.TrimSpace(string(message.Kind))))
	if message.Kind == "" {
		message.Kind = InvitationMessageGeneric
	}
	bankName := strings.TrimSpace(message.BankName)
	if bankName == "" {
		bankName = "ClearSight"
	}
	taskTitle := strings.TrimSpace(message.TaskTitle)
	if taskTitle == "" {
		taskTitle = "Secure form request"
	}
	heading, subject, action, preheader := taskTitle, "Action needed: "+taskTitle, "Open secure form", strings.TrimSpace(message.TaskSummary)
	switch message.Kind {
	case InvitationMessageGeneric:
	case InvitationMessageVendorRegistration:
		heading, subject, action = "Complete your vendor registration", "Complete your vendor registration for "+bankName, "Complete registration"
	case InvitationMessageAddressVerification:
		heading, subject, action = "Verify the vendor's registered address", "Verify the vendor's registered address", "Verify address"
	case InvitationMessageCertificationRefresh:
		heading, subject, action = "Submit current certification evidence", "Submit current certification evidence", "Submit certification evidence"
	default:
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}
	if preheader == "" {
		preheader = taskTitle
	}
	facts := make([]emailFact, 0, 3)
	if role := strings.TrimSpace(message.RecipientRole); role != "" {
		facts = append(facts, emailFact{Label: "Your role", Value: role})
	}
	if !message.DueAt.IsZero() {
		facts = append(facts, emailFact{Label: "Due", Value: message.DueAt.UTC().Format("2 Jan 2006, 15:04 UTC")})
	}
	if !message.ExpiresAt.IsZero() {
		facts = append(facts, emailFact{Label: "Link expires", Value: message.ExpiresAt.UTC().Format("2 Jan 2006, 15:04 UTC")})
	}
	intro := strings.TrimSpace(message.TaskSummary)
	if intro == "" {
		intro = "Use the secure link below to complete this request. Verify the invited email address when prompted."
	}
	presentation, err := renderEmailPresentation(emailPresentationInput{
		BrandName: bankName, Preheader: preheader, Heading: heading, Intro: intro,
		BodyPlain:   "Request: " + taskTitle,
		BodyHTML:    `<p style="margin:0 0 16px;">Request: <strong>` + html.EscapeString(taskTitle) + `</strong></p><p style="margin:0 0 16px;">Do not forward this message. Access is limited to this request and the invited email address.</p>`,
		ActionLabel: action, ActionURL: request.InvitationLink, Facts: facts, SupportContact: strings.TrimSpace(message.SupportContact),
	})
	if err != nil || len(subject) > 200 || strings.ContainsAny(subject, "\r\n") {
		return InvitationDeliveryRequest{}, ErrInvitationDeliveryRequestInvalid
	}
	request.Subject, request.PlainText, request.HTML, request.Message = subject, presentation.PlainText, presentation.HTML, message
	return request, nil
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
