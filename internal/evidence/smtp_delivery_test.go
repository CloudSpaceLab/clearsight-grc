package evidence

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSMTPDeliveryRequiresTLSOutsideDevelopment(t *testing.T) {
	_, err := NewSMTPDelivery(SMTPDeliveryConfig{
		Host: "smtp.example.test", Port: 25, FromAddress: "forms@example.test",
		TLSMode: SMTPTLSPlain, Environment: "production",
	}, nil)
	if err == nil {
		t.Fatal("production plaintext SMTP was accepted")
	}
	_, err = NewSMTPDelivery(SMTPDeliveryConfig{
		Host: "smtp.example.test", Port: 587, FromAddress: "forms@example.test",
		TLSMode: SMTPTLSStartTLS, Environment: "production", Username: "mailer",
	}, nil)
	if err == nil {
		t.Fatal("SMTP authentication without a secret resolver was accepted")
	}
}

func TestSMTPDeliveryUsesSTARTTLSAndSendsRenderedAlternatives(t *testing.T) {
	certificate, roots := smtpTestCertificate(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	result := make(chan smtpTestServerResult, 1)
	go runSMTPStartTLSServer(listener, certificate, result)

	delivery, err := NewSMTPDelivery(SMTPDeliveryConfig{
		Host: "localhost", Port: port, FromAddress: "forms@example.test",
		TLSMode: SMTPTLSStartTLS, Environment: "production", RootCAs: roots,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 28, 7, 30, 0, 0, time.UTC)
	delivery.now = func() time.Time { return now }
	delivery.newID = func() (string, error) { return "0198f000-0000-7000-8000-000000000001", nil }

	receipt, err := delivery.Deliver(context.Background(), InvitationDeliveryRequest{
		RecipientAddress: "owner@example.test",
		InvitationLink:   "https://forms.example.test/s/selector",
		Subject:          "Example Bank: control review",
		PlainText:        "Open the secure form: https://forms.example.test/s/selector",
		HTML:             `<p>Open the secure form.</p><p><a href="https://forms.example.test/s/selector">Continue</a></p>`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != InvitationDelivered || receipt.DeliveredAt == nil || !receipt.DeliveredAt.Equal(now) || receipt.ProviderMessageID == "" {
		t.Fatalf("unexpected SMTP receipt: %+v", receipt)
	}

	select {
	case server := <-result:
		if server.err != nil {
			t.Fatal(server.err)
		}
		if !server.startTLS {
			t.Fatal("SMTP transport did not negotiate STARTTLS")
		}
		message := string(server.message)
		for _, expected := range []string{
			"Subject: Example Bank: control review",
			"Auto-Submitted: auto-generated",
			"Content-Type: multipart/alternative",
			"text/plain",
			"text/html",
			"https://forms.example.test/s/selector",
			"Message-ID: <0198f000-0000-7000-8000-000000000001@example.test>",
		} {
			if !strings.Contains(message, expected) {
				t.Fatalf("SMTP message missing %q:\n%s", expected, message)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SMTP test server did not complete")
	}
}

type smtpTestServerResult struct {
	startTLS bool
	message  []byte
	err      error
}

func runSMTPStartTLSServer(listener net.Listener, certificate tls.Certificate, result chan<- smtpTestServerResult) {
	serverResult := smtpTestServerResult{}
	conn, err := listener.Accept()
	if err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writeLine := func(value string) error {
		if _, err := writer.WriteString(value + "\r\n"); err != nil {
			return err
		}
		return writer.Flush()
	}
	readLine := func() (string, error) {
		value, err := reader.ReadString('\n')
		return strings.TrimRight(value, "\r\n"), err
	}
	if err := writeLine("220 localhost ESMTP"); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	if line, err := readLine(); err != nil || !strings.HasPrefix(line, "EHLO ") {
		serverResult.err = smtpTestProtocolError("expected EHLO", line, err)
		result <- serverResult
		return
	}
	if _, err := writer.WriteString("250-localhost\r\n250-STARTTLS\r\n250 HELP\r\n"); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	if err := writer.Flush(); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	line, err := readLine()
	if err != nil || line != "STARTTLS" {
		serverResult.err = smtpTestProtocolError("expected STARTTLS", line, err)
		result <- serverResult
		return
	}
	serverResult.startTLS = true
	if err := writeLine("220 Ready to start TLS"); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	if err := tlsConn.Handshake(); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	reader = bufio.NewReader(tlsConn)
	writer = bufio.NewWriter(tlsConn)
	if line, err = readLine(); err != nil || !strings.HasPrefix(line, "EHLO ") {
		serverResult.err = smtpTestProtocolError("expected EHLO after STARTTLS", line, err)
		result <- serverResult
		return
	}
	if err := writeLine("250 localhost"); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	for _, command := range []string{"MAIL FROM:", "RCPT TO:"} {
		line, err = readLine()
		if err != nil || !strings.HasPrefix(line, command) {
			serverResult.err = smtpTestProtocolError("expected "+command, line, err)
			result <- serverResult
			return
		}
		if err := writeLine("250 OK"); err != nil {
			serverResult.err = err
			result <- serverResult
			return
		}
	}
	line, err = readLine()
	if err != nil || line != "DATA" {
		serverResult.err = smtpTestProtocolError("expected DATA", line, err)
		result <- serverResult
		return
	}
	if err := writeLine("354 End data with <CR><LF>.<CR><LF>"); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	serverResult.message, err = readSMTPDotMessage(reader)
	if err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	if err := writeLine("250 queued"); err != nil {
		serverResult.err = err
		result <- serverResult
		return
	}
	line, err = readLine()
	if err != nil || line != "QUIT" {
		serverResult.err = smtpTestProtocolError("expected QUIT", line, err)
		result <- serverResult
		return
	}
	_ = writeLine("221 Bye")
	result <- serverResult
}

func readSMTPDotMessage(reader *bufio.Reader) ([]byte, error) {
	var message strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if line == ".\r\n" || line == ".\n" {
			return []byte(message.String()), nil
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		message.WriteString(line)
	}
}

func smtpTestCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certPEM)
	return certificate, roots
}

func smtpTestProtocolError(expectation, line string, err error) error {
	if err != nil && err != io.EOF {
		return err
	}
	return &smtpProtocolTestError{expectation: expectation, line: line}
}

type smtpProtocolTestError struct {
	expectation string
	line        string
}

func (err *smtpProtocolTestError) Error() string {
	return err.expectation + "; got " + err.line
}
