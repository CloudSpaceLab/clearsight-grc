package aigateway

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is safe to return to clients. It deliberately contains no provider body,
// prompt, completion, tool arguments, secret, endpoint or source data.
type Error struct {
	Code          string
	Message       string
	Param         string
	Status        int
	Retriable     bool
	ProviderFault bool
	cause         error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && e.Code == other.Code
}

func withCause(err *Error, cause error) *Error {
	copyValue := *err
	copyValue.cause = cause
	return &copyValue
}

var (
	ErrInvalidRequest    = &Error{Code: "invalid_request", Message: "The request is not supported or is outside the configured limits.", Status: http.StatusBadRequest}
	ErrUnauthorized      = &Error{Code: "authentication_failed", Message: "The workload credential is missing or invalid.", Status: http.StatusUnauthorized}
	ErrModelNotFound     = &Error{Code: "model_not_found", Message: "The requested model alias is not available to this workload.", Param: "model", Status: http.StatusNotFound}
	ErrBudgetExceeded    = &Error{Code: "budget_exceeded", Message: "The workload budget does not permit this request in the current minute.", Status: http.StatusTooManyRequests}
	ErrConcurrency       = &Error{Code: "concurrency_exceeded", Message: "The workload has reached its concurrent request limit.", Status: http.StatusTooManyRequests}
	ErrUnavailable       = &Error{Code: "provider_unavailable", Message: "No configured provider route completed the request.", Status: http.StatusServiceUnavailable, Retriable: true, ProviderFault: true}
	ErrTimeout           = &Error{Code: "request_timeout", Message: "The request exceeded the configured gateway deadline.", Status: http.StatusGatewayTimeout, Retriable: true, ProviderFault: true}
	ErrProviderRate      = &Error{Code: "provider_rate_limited", Message: "The selected provider is temporarily rate limited.", Status: http.StatusServiceUnavailable, Retriable: true, ProviderFault: true}
	ErrProvider          = &Error{Code: "provider_error", Message: "The selected provider rejected or failed the request.", Status: http.StatusBadGateway, ProviderFault: true}
	ErrProtocol          = &Error{Code: "provider_protocol_error", Message: "The selected provider returned an invalid or incomplete protocol response.", Status: http.StatusBadGateway, Retriable: true, ProviderFault: true}
	ErrStream            = &Error{Code: "stream_failed", Message: "The provider stream ended before a valid completion was received.", Status: http.StatusBadGateway, Retriable: true, ProviderFault: true}
	ErrCanceled          = &Error{Code: "request_cancelled", Message: "The request was cancelled before completion.", Status: http.StatusRequestTimeout}
	ErrPolicyDenied      = &Error{Code: "policy_denied", Message: "The active AI governance policy does not permit this request.", Status: http.StatusForbidden}
	ErrApprovalRequired  = &Error{Code: "approval_required", Message: "The active AI governance policy requires an approved execution grant.", Status: http.StatusForbidden}
	ErrPolicyUnavailable = &Error{Code: "policy_unavailable", Message: "The governed workload or active policy is unavailable.", Status: http.StatusServiceUnavailable}
	ErrSourceFacts       = &Error{Code: "source_facts_unavailable", Message: "The required governed source facts could not be resolved.", Status: http.StatusServiceUnavailable}
	ErrInternal          = &Error{Code: "internal_error", Message: "The gateway could not complete the request.", Status: http.StatusInternalServerError}
)

func invalid(param, detail string) *Error {
	message := ErrInvalidRequest.Message
	if detail != "" {
		message = detail
	}
	return &Error{Code: ErrInvalidRequest.Code, Message: message, Param: param, Status: ErrInvalidRequest.Status}
}

func providerHTTPError(status int, cause error) *Error {
	switch {
	case status == http.StatusTooManyRequests:
		return withCause(ErrProviderRate, cause)
	case status >= 500:
		return withCause(ErrUnavailable, cause)
	case status == http.StatusRequestTimeout:
		return withCause(ErrTimeout, cause)
	case status >= 400:
		return withCause(ErrProvider, cause)
	default:
		return withCause(ErrProtocol, cause)
	}
}

func asGatewayError(err error) *Error {
	if err == nil {
		return nil
	}
	var gatewayErr *Error
	if errors.As(err, &gatewayErr) {
		return gatewayErr
	}
	return withCause(ErrInternal, fmt.Errorf("gateway operation: %w", err))
}
