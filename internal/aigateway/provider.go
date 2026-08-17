package aigateway

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"
)

const (
	ProviderKindOpenAI    = "OPENAI"
	ProviderKindAnthropic = "ANTHROPIC"
)

// Provider is the small transport boundary used by the stateless gateway.
type Provider interface {
	ID() string
	Complete(context.Context, ProviderRequest) (Response, error)
	Stream(context.Context, ProviderRequest) (ProviderStream, error)
}

// ProviderStream returns validated semantic events. io.EOF is valid only after
// a StreamDone event has been returned.
type ProviderStream interface {
	Recv() (StreamEvent, error)
	Close() error
}

type ProviderRequest struct {
	Request
	ProviderModel string
}

type providerRuntime struct {
	provider Provider
	config   ResolvedProviderConfig
}

func newProviderHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 128
	transport.MaxIdleConnsPerHost = 32
	transport.IdleConnTimeout = 90 * time.Second
	transport.ResponseHeaderTimeout = timeout
	transport.ExpectContinueTimeout = time.Second
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	return &http.Client{
		Transport: transport,
		Timeout:   0, // request contexts and provider response-header deadlines own cancellation
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func closeBody(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}
