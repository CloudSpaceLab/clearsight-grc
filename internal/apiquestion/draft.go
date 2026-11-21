// Package apiquestion owns protected form-question connection drafts.
// Drafts are not bindings, observations, or approval decisions.
package apiquestion

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
)

var ErrInvalid = errors.New("invalid API question draft")
var ErrUnavailable = errors.New("API question draft unavailable")
var ErrConflict = errors.New("API question draft changed")
var ErrNotFound = errors.New("API question draft not found")

type Receipt struct {
	ID      string `json:"id"`
	Version int64  `json:"version"`
}

type Draft struct {
	Version int64 `json:"version"`
	DraftInput
}

// Store is held behind an unregistered API dependency until authorization and
// activation semantics have been accepted.
type Store interface {
	ReadDraft(context.Context, string, int64, string) (Draft, error)
	WriteDraft(context.Context, string, int64, string, int64, DraftInput) (Receipt, error)
	ReadAdoptionManualDraft(context.Context, AdoptionManualDraft) (AdoptionQuestionDraft, error)
	WriteAdoptionManualDraft(context.Context, AdoptionManualDraft) (Receipt, error)
	ReadAdoptionQuestionCandidate(context.Context, AdoptionQuestionApproval) (AdoptionQuestionCandidate, error)
	ApproveAdoptionQuestion(context.Context, AdoptionQuestionApproval) (Receipt, error)
}

type GuardedStore interface {
	Store
	ConfigureAuthority(*commandauth.Guard)
}

type DraftInput struct {
	Mode       string `json:"mode"`
	Endpoint   string `json:"endpoint,omitempty"`
	ResultType string `json:"result_type,omitempty"`
	Key        string `json:"-"`
}

func validateDraft(input DraftInput, fieldType string) error {
	if input.Mode != "API" || len(input.Endpoint) > 2048 || strings.TrimSpace(input.Endpoint) != input.Endpoint {
		return ErrInvalid
	}
	if !CompatibleQuestionResult(fieldType, input.ResultType) {
		return ErrInvalid
	}
	u, err := url.Parse(input.Endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return ErrInvalid
	}
	return nil
}

// CompatibleQuestionResult keeps draft and format-test question types aligned.
func CompatibleQuestionResult(fieldType, resultType string) bool {
	switch resultType {
	case "BOOLEAN":
		return fieldType == "yes_no"
	case "NUMBER":
		return fieldType == "decimal" || fieldType == "percentage"
	default:
		return false
	}
}

func validIdempotencyKey(key string) bool {
	if len(key) < 16 || len(key) > 128 {
		return false
	}
	for _, char := range key {
		if !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '_') {
			return false
		}
	}
	return true
}
