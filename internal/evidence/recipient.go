package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrRecipientRequired           = errors.New("evidence request recipient is required")
	ErrRecipientInvalid            = errors.New("evidence request recipient is invalid")
	ErrRecipientMismatch           = errors.New("evidence request is assigned to a different recipient")
	ErrRecipientReassignmentNeeded = errors.New("evidence request recipient requires reassignment")
	ErrRecipientManagerRequired    = errors.New("evidence request recipient change requires the request creator")
)

type internalRecipientDirectory interface {
	InternalRecipientEligible(context.Context, string, string) (bool, error)
}

type internalRecipientLabelDirectory interface {
	InternalRecipientDisplayName(context.Context, string, string) (string, error)
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
		recipient := Recipient{Type: RecipientInternalPrincipal, PrincipalID: strings.TrimSpace(input.PrincipalID), State: RecipientStateAssigned, Revision: 1}
		if directory, ok := repo.(internalRecipientLabelDirectory); ok {
			displayName, err := directory.InternalRecipientDisplayName(ctx, tenant, recipient.PrincipalID)
			if err != nil {
				return Recipient{}, err
			}
			recipient.DisplayName = displayName
		}
		return recipient, nil
	case "EXTERNAL", "CUSTOMER", "VENDOR", "AUTHORITY":
		audience := normalizeAudience(input.Audience)
		if input.Type != RecipientExternalAudience || strings.TrimSpace(input.PrincipalID) != "" || audience == "" {
			return Recipient{}, ErrRecipientInvalid
		}
		digest := sha256.Sum256([]byte(audience))
		return Recipient{Type: RecipientExternalAudience, AudienceHash: digest[:], AudienceHint: audienceHint(audience), State: RecipientStateAssigned, Revision: 1}, nil
	default:
		return Recipient{}, fmt.Errorf("audience_type is invalid")
	}
}

func recipientIsAssigned(recipient Recipient) bool {
	// Empty state is accepted only for pre-B2 in-memory fixtures. PostgreSQL B2
	// rows always carry an explicit state.
	return recipient.State == "" || recipient.State == RecipientStateAssigned
}

func sameRecipient(left, right Recipient) bool {
	if left.Type != right.Type {
		return false
	}
	switch left.Type {
	case RecipientInternalPrincipal:
		return left.PrincipalID == right.PrincipalID
	case RecipientExternalAudience:
		return bytes.Equal(left.AudienceHash, right.AudienceHash)
	default:
		return false
	}
}

func RequestAssignedTo(request Request, principalID string) bool {
	principalID = strings.TrimSpace(principalID)
	return recipientIsAssigned(request.Recipient) && request.Recipient.Type == RecipientInternalPrincipal && principalID != "" && request.Recipient.PrincipalID == principalID
}

// RequestManageableBy is intentionally narrower than subject visibility. A
// direct internal recipient may manage their request, while an external request
// remains manageable by the verified creator who must issue/revoke capability
// access and monitor completion. Legacy unassigned requests are never actor work
// but remain manageable by their trusted creator.
func RequestManageableBy(request Request, principalID string) bool {
	principalID = strings.TrimSpace(principalID)
	return principalID != "" && (RequestAssignedTo(request, principalID) || strings.TrimSpace(request.CreatedBy) == principalID)
}

func externalAudienceMatches(request Request, audience string) bool {
	if !recipientIsAssigned(request.Recipient) || request.Recipient.Type != RecipientExternalAudience || len(request.Recipient.AudienceHash) != sha256.Size {
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

func externalRecipientRequest(request Request) bool {
	if !recipientIsAssigned(request.Recipient) || request.Recipient.Type != RecipientExternalAudience {
		return false
	}
	switch request.AudienceType {
	case "EXTERNAL", "CUSTOMER", "VENDOR", "AUTHORITY":
		return true
	default:
		return false
	}
}
