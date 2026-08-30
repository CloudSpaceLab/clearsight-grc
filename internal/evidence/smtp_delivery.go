package evidence

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type SMTPTLSMode string

const (
	SMTPTLSStartTLS SMTPTLSMode = "STARTTLS"
	SMTPTLSImplicit SMTPTLSMode = "TLS"
	SMTPTLSPlain    SMTPTLSMode = "PLAIN"
)

type SMTPDeliveryConfig struct {
	Host        string
	Port        int
	Username    string
	SecretRef   string
	FromAddress string
	TLSMode     SMTPTLSMode
	Environment string
	RootCAs     *x509.CertPool
}

type SMTPSecretResolver interface {
	ResolveSecret(context.Context, string) (string, error)
}

type SMTPSecretResolverFunc func(context.Context, string) (string, error)

func (fn SMTPSecretResolverFunc) ResolveSecret(ctx context.Context, reference string) (string, error) {
	return fn(ctx, reference)
}

type SMTPDelivery struct {
	config  SMTPDeliveryConfig
	secrets SMTPSecretResolver
	dialer  net.Dialer
	now     func() time.Time
	newID   func() (string, error)
}

func NewSMTPDelivery(config SMTPDeliveryConfig, secrets SMTPSecretResolver) (*SMTPDelivery, error) {
	config.Host = strings.TrimSpace(config.Host)
	config.Username = strings.TrimSpace(config.Username)
	config.SecretRef = strings.TrimSpace(config.SecretRef)
	config.FromAddress = strings.TrimSpace(config.FromAddress)
	config.Environment = strings.TrimSpace(config.Environment)
	config.TLSMode = SMTPTLSMode(strings.ToUpper(strings.TrimSpace(string(config.TLSMode))))
	if err := validateSMTPDeliveryConfig(config, secrets); err != nil {
		return nil, err
	}
	return &SMTPDelivery{
		config:  config,
		secrets: secrets,
		dialer:  net.Dialer{Timeout: 15 * time.Second},
		now:     time.Now,
		newID:   id.NewUUIDv7,
	}, nil
}

func validateSMTPDeliveryConfig(config SMTPDeliveryConfig, secrets SMTPSecretResolver) error {
	if config.Host == "" || strings.ContainsAny(config.Host, "/\r\n\t ") || config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("%w: invalid SMTP endpoint", ErrInvitationDeliveryRequestInvalid)
	}
	from, err := mail.ParseAddress(config.FromAddress)
	if err != nil || !strings.EqualFold(from.Address, config.FromAddress) {
		return fmt.Errorf("%w: invalid SMTP from address", ErrInvitationDeliveryRequestInvalid)
	}
	if config.Username != "" && (config.SecretRef == "" || secrets == nil) {
		return fmt.Errorf("%w: SMTP authentication requires a secret reference", ErrInvitationDeliveryRequestInvalid)
	}
	if config.Username == "" && config.SecretRef != "" {
		return fmt.Errorf("%w: SMTP secret reference requires a username", ErrInvitationDeliveryRequestInvalid)
	}
	switch config.TLSMode {
	case SMTPTLSStartTLS, SMTPTLSImplicit:
	case SMTPTLSPlain:
		if !strings.EqualFold(config.Environment, "development") {
			return fmt.Errorf("%w: plaintext SMTP is restricted to development", ErrInvitationDeliveryRequestInvalid)
		}
		if config.Username != "" {
			return fmt.Errorf("%w: SMTP authentication requires TLS", ErrInvitationDeliveryRequestInvalid)
		}
	default:
		return fmt.Errorf("%w: SMTP TLS mode must be STARTTLS, TLS or development-only PLAIN", ErrInvitationDeliveryRequestInvalid)
	}
	return nil
}

func (delivery *SMTPDelivery) Deliver(ctx context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
	if delivery == nil {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryUnavailable
	}
	recipient, err := parseSMTPMailbox(request.RecipientAddress)
	if err != nil {
		return invitationFailureReceipt("", InvitationFailureRecipientRejected), nil
	}
	subject := strings.TrimSpace(request.Subject)
	plainText := request.PlainText
	htmlBody := request.HTML
	if subject == "" {
		subject = "Secure form invitation"
	}
	if strings.ContainsAny(subject, "\r\n") || len(subject) > 998 {
		return invitationFailureReceipt("", InvitationFailurePermanent), nil
	}
	if strings.TrimSpace(plainText) == "" {
		plainText = strings.TrimSpace(request.InvitationLink)
	}
	if strings.TrimSpace(plainText) == "" && strings.TrimSpace(htmlBody) == "" {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryRequestInvalid
	}
	messageID, err := delivery.newID()
	if err != nil {
		return InvitationDeliveryReceipt{}, ErrInvitationDeliveryUnavailable
	}
	messageID = "<" + messageID + "@" + smtpMessageIDDomain(delivery.config.FromAddress) + ">"
	payload, err := buildSMTPMessage(delivery.config.FromAddress, recipient, subject, plainText, htmlBody, messageID, delivery.currentTime())
	if err != nil {
		return InvitationDeliveryReceipt{}, err
	}
	password := ""
	if delivery.config.Username != "" {
		password, err = delivery.secrets.ResolveSecret(ctx, delivery.config.SecretRef)
		if err != nil || strings.TrimSpace(password) == "" {
			return InvitationDeliveryReceipt{}, ErrInvitationDeliveryUnavailable
		}
	}
	if err := delivery.send(ctx, recipient, payload, password); err != nil {
		return smtpFailureReceipt(err), nil
	}
	deliveredAt := delivery.currentTime()
	return InvitationDeliveryReceipt{
		Status:            InvitationDelivered,
		ProviderMessageID: messageID,
		DeliveredAt:       &deliveredAt,
	}, nil
}

func (delivery *SMTPDelivery) send(ctx context.Context, recipient string, payload []byte, password string) error {
	address := net.JoinHostPort(delivery.config.Host, strconv.Itoa(delivery.config.Port))
	var conn net.Conn
	var err error
	tlsConfig := &tls.Config{ServerName: delivery.config.Host, MinVersion: tls.VersionTLS12, RootCAs: delivery.config.RootCAs}
	if delivery.config.TLSMode == SMTPTLSImplicit {
		tlsDialer := tls.Dialer{NetDialer: &delivery.dialer, Config: tlsConfig}
		conn, err = tlsDialer.DialContext(ctx, "tcp", address)
	} else {
		conn, err = delivery.dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	}

	client, err := smtp.NewClient(conn, delivery.config.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if delivery.config.TLSMode == SMTPTLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not advertise STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if delivery.config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", delivery.config.Username, password, delivery.config.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(delivery.config.FromAddress); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return &smtpRecipientError{err: err}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func buildSMTPMessage(from, to, subject, plainText, htmlBody, messageID string, sentAt time.Time) ([]byte, error) {
	if _, err := parseSMTPMailbox(from); err != nil {
		return nil, err
	}
	if _, err := parseSMTPMailbox(to); err != nil {
		return nil, err
	}
	if strings.ContainsAny(subject, "\r\n") || strings.ContainsAny(messageID, "\r\n") {
		return nil, ErrInvitationDeliveryRequestInvalid
	}
	var buffer bytes.Buffer
	writeSMTPHeader(&buffer, "From", from)
	writeSMTPHeader(&buffer, "To", to)
	writeSMTPHeader(&buffer, "Subject", mime.QEncoding.Encode("UTF-8", subject))
	writeSMTPHeader(&buffer, "Date", sentAt.UTC().Format(time.RFC1123Z))
	writeSMTPHeader(&buffer, "Message-ID", messageID)
	writeSMTPHeader(&buffer, "MIME-Version", "1.0")
	writeSMTPHeader(&buffer, "Auto-Submitted", "auto-generated")
	if strings.TrimSpace(htmlBody) == "" {
		writeSMTPHeader(&buffer, "Content-Type", `text/plain; charset="UTF-8"`)
		writeSMTPHeader(&buffer, "Content-Transfer-Encoding", "quoted-printable")
		buffer.WriteString("\r\n")
		if err := writeQuotedPrintable(&buffer, plainText); err != nil {
			return nil, err
		}
		return buffer.Bytes(), nil
	}
	boundary := "clearsight-" + strings.Trim(messageID, "<>")
	boundary = strings.NewReplacer("@", "-", ".", "-").Replace(boundary)
	writeSMTPHeader(&buffer, "Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
	buffer.WriteString("\r\n--" + boundary + "\r\n")
	writeSMTPHeader(&buffer, "Content-Type", `text/plain; charset="UTF-8"`)
	writeSMTPHeader(&buffer, "Content-Transfer-Encoding", "quoted-printable")
	buffer.WriteString("\r\n")
	if err := writeQuotedPrintable(&buffer, plainText); err != nil {
		return nil, err
	}
	buffer.WriteString("\r\n--" + boundary + "\r\n")
	writeSMTPHeader(&buffer, "Content-Type", `text/html; charset="UTF-8"`)
	writeSMTPHeader(&buffer, "Content-Transfer-Encoding", "quoted-printable")
	buffer.WriteString("\r\n")
	if err := writeQuotedPrintable(&buffer, htmlBody); err != nil {
		return nil, err
	}
	buffer.WriteString("\r\n--" + boundary + "--\r\n")
	return buffer.Bytes(), nil
}

func writeSMTPHeader(buffer *bytes.Buffer, name, value string) {
	buffer.WriteString(name)
	buffer.WriteString(": ")
	buffer.WriteString(value)
	buffer.WriteString("\r\n")
}

func writeQuotedPrintable(buffer *bytes.Buffer, value string) error {
	writer := quotedprintable.NewWriter(buffer)
	if _, err := writer.Write([]byte(value)); err != nil {
		return err
	}
	return writer.Close()
}

func parseSMTPMailbox(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address == "" || !strings.EqualFold(parsed.Address, value) || strings.ContainsAny(value, "\r\n") {
		return "", ErrInvitationDeliveryRequestInvalid
	}
	return parsed.Address, nil
}

func smtpMessageIDDomain(from string) string {
	if index := strings.LastIndex(from, "@"); index >= 0 && index+1 < len(from) {
		return strings.ToLower(from[index+1:])
	}
	return "localhost"
}

type smtpRecipientError struct{ err error }

func (err *smtpRecipientError) Error() string { return err.err.Error() }
func (err *smtpRecipientError) Unwrap() error { return err.err }

func smtpFailureReceipt(err error) InvitationDeliveryReceipt {
	code := InvitationFailureTemporary
	var recipientErr *smtpRecipientError
	if errors.As(err, &recipientErr) {
		code = InvitationFailureRecipientRejected
		if smtpTemporary(recipientErr.err) {
			code = InvitationFailureTemporary
		}
	} else if smtpPermanent(err) {
		code = InvitationFailurePermanent
	}
	return InvitationDeliveryReceipt{Status: InvitationDeliveryFailed, FailureCode: code}
}

func smtpTemporary(err error) bool {
	var protocolErr *textproto.Error
	return errors.As(err, &protocolErr) && protocolErr.Code >= 400 && protocolErr.Code < 500
}

func smtpPermanent(err error) bool {
	var protocolErr *textproto.Error
	return errors.As(err, &protocolErr) && protocolErr.Code >= 500 && protocolErr.Code < 600
}

func (delivery *SMTPDelivery) currentTime() time.Time {
	if delivery != nil && delivery.now != nil {
		return delivery.now().UTC()
	}
	return time.Now().UTC()
}

var _ InvitationDelivery = (*SMTPDelivery)(nil)
