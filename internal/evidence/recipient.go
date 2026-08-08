package evidence

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrRecipientRequired = errors.New("evidence request recipient is required")
	ErrRecipientInvalid  = errors.New("evidence request recipient is invalid")
	ErrRecipientMismatch = errors.New("evidence request is assigned to a different recipient")
)

type internalRecipientDirectory interface {
	InternalRecipientEligible(context.Context, string, string) (bool, error)
}

type recipientRequestLister interface {
	ListRecipientRequests(context.Context, string, string, int) ([]Request, error)
}

func buildRecipient(ctx context.Context, repo Repository, tenant, audienceType string, input RecipientInput) (Recipient, error) {
	switch audienceType {
	case "INTERNAL":
		if input.Type != RecipientInternalPrincipal || strings.TrimSpace(input.PrincipalID) == "" || strings.TrimSpace(input.Audience) != "" {
			return Recipient{}, ErrRecipientInvalid
		}
		if directory, ok := repo.(internalRecipientDirectory); ok {
			eligible, err := directory.InternalRecipientEligible(ctx, tenant, strings.TrimSpace(input.PrincipalID))
			if err != nil {
				return Recipient{}, err
			}
			if !eligible {
				return Recipient{}, ErrRecipientInvalid
			}
		}
		return Recipient{Type: RecipientInternalPrincipal, PrincipalID: strings.TrimSpace(input.PrincipalID)}, nil
	case "INVITED_EXTERNAL":
		audience := normalizeAudience(input.Audience)
		if input.Type != RecipientExternalAudience || strings.TrimSpace(input.PrincipalID) != "" || audience == "" {
			return Recipient{}, ErrRecipientInvalid
		}
		digest := sha256.Sum256([]byte(audience))
		return Recipient{Type: RecipientExternalAudience, AudienceHash: digest[:], AudienceHint: audienceHint(audience)}, nil
	default:
		return Recipient{}, fmt.Errorf("audience_type must be INTERNAL or INVITED_EXTERNAL")
	}
}

func RequestAssignedTo(request Request, principalID string) bool {
	return request.Recipient.Type == RecipientInternalPrincipal && strings.TrimSpace(principalID) != "" && request.Recipient.PrincipalID == strings.TrimSpace(principalID)
}

func externalAudienceMatches(request Request, audience string) bool {
	if request.Recipient.Type != RecipientExternalAudience || len(request.Recipient.AudienceHash) != sha256.Size {
		return false
	}
	audience = normalizeAudience(audience)
	if audience == "" {
		return false
	}
	digest := sha256.Sum256([]byte(audience))
	return subtle.ConstantTimeCompare(request.Recipient.AudienceHash, digest[:]) == 1
}

func internalSubmissionAllowed(request Request, submission Submission) bool {
	return request.AudienceType == "INTERNAL" && RequestAssignedTo(request, submission.SubmittedBy) && strings.EqualFold(strings.TrimSpace(submission.Channel), "INTERNAL")
}
