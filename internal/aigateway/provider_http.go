package aigateway

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func providerRequest(ctx context.Context, client *http.Client, timeout time.Duration, method, target string, payload []byte, headers map[string]string) (*http.Response, context.CancelFunc, error) {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	request, err := http.NewRequestWithContext(requestCtx, method, target, bytes.NewReader(payload))
	if err != nil {
		cancel()
		return nil, func() {}, withCause(ErrInternal, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.Do(request)
	if err != nil {
		cancel()
		if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return nil, func() {}, withCause(ErrTimeout, requestCtx.Err())
		}
		if errors.Is(requestCtx.Err(), context.Canceled) {
			return nil, func() {}, withCause(ErrCanceled, requestCtx.Err())
		}
		return nil, func() {}, withCause(ErrUnavailable, err)
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		closeBody(response.Body)
		cancel()
		return nil, func() {}, withCause(ErrProtocol, fmt.Errorf("provider redirect status %d", response.StatusCode))
	}
	return response, cancel, nil
}

func readProviderBody(response *http.Response, maxBytes int64) ([]byte, error) {
	defer closeBody(response.Body)
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, withCause(ErrUnavailable, err)
	}
	if int64(len(body)) > maxBytes {
		return nil, withCause(ErrProtocol, fmt.Errorf("provider body exceeds configured limit"))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, providerHTTPError(response.StatusCode, fmt.Errorf("provider status %d", response.StatusCode))
	}
	return body, nil
}

func requireEventStream(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		closeBody(response.Body)
		return providerHTTPError(response.StatusCode, fmt.Errorf("provider status %d", response.StatusCode))
	}
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/event-stream") {
		closeBody(response.Body)
		return withCause(ErrProtocol, fmt.Errorf("provider did not return an event stream"))
	}
	return nil
}
