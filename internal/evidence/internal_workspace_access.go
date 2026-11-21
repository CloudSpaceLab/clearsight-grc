package evidence

import (
	"context"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// internalWorkspaceAccessAllowed is a transaction-boundary predicate, not a
// replacement for querying an exact recipient or checking live subject access.
// Callers must compute eligible from the current authoritative directory and
// subject-access policy and must re-evaluate it under the mutation lock.
func internalWorkspaceAccessAllowed(ctx context.Context, distribution FormDistribution, workspace ResponseWorkspace, recipient DistributionRecipient, request Request, eligible bool, now time.Time) bool {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.LegalEntityID == "*" || !eligible {
		return false
	}
	if distribution.ID == "" || distribution.TenantID != actor.TenantID || distribution.LegalEntityID != actor.LegalEntityID || !distributionAcceptsAccess(distribution) || !now.Before(distribution.Deadline) {
		return false
	}
	if workspace.ID == "" || workspace.TenantID != actor.TenantID || workspace.LegalEntityID != actor.LegalEntityID || workspace.DistributionID != distribution.ID || workspace.Status != ResponseWorkspaceOpen {
		return false
	}
	if recipient.ID == "" || recipient.TenantID != actor.TenantID || recipient.LegalEntityID != actor.LegalEntityID || recipient.DistributionID != distribution.ID || recipient.RequestID != request.ID || recipient.Role != RecipientTo || recipient.Type != RecipientInternalPrincipal || recipient.PrincipalID != actor.PrincipalID {
		return false
	}
	switch recipient.State {
	case DistributionRecipientPending, DistributionRecipientDelivered, DistributionRecipientVerified:
	default:
		return false
	}
	return request.ID != "" && request.TenantID == actor.TenantID && request.LegalEntityID == actor.LegalEntityID && requestOpenAt(request, now) &&
		request.Recipient.Type == RecipientInternalPrincipal && request.Recipient.PrincipalID == actor.PrincipalID && request.Recipient.State == RecipientStateAssigned &&
		request.SubjectType == distribution.SubjectType && request.SubjectID == distribution.SubjectID && request.SubjectID != "" &&
		request.FormTemplateID == distribution.FormTemplateID && request.FormTemplateID != "" && request.FormTemplateVersion == distribution.FormTemplateVersion && request.FormTemplateVersion > 0
}
