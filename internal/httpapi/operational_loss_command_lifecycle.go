package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
)

func (a *API) operationalLossLifecycleCommandPolicy(
	ctx context.Context,
	r *http.Request,
	tenant string,
	name string,
	payload map[string]any,
	policy commandPolicy,
) (commandPolicy, error) {
	if name == "loss.create" {
		return policy, nil
	}
	if a == nil || a.deps.OperationalLoss == nil {
		return policy, fmt.Errorf("%w: operational loss service is unavailable", commandauth.ErrGuardUnavailable)
	}
	actor, err := identity.Require(ctx)
	if err != nil {
		return policy, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
	}
	legalEntityID := strings.TrimSpace(actor.LegalEntityID)
	if legalEntityID == "" || legalEntityID == "*" {
		return policy, nil
	}
	lossID := ""
	if r != nil {
		lossID = strings.TrimSpace(r.PathValue("id"))
	}
	if lossID == "" {
		return policy, oploss.ErrNotFound
	}
	current, err := a.deps.OperationalLoss.Get(ctx, oploss.Scope{
		TenantID: tenant, LegalEntityID: legalEntityID,
	}, lossID)
	if err != nil {
		return policy, err
	}
	if err := validateRequestedRecordEntity(actor, stringValue(payload["legal_entity_id"]), current.Loss.LegalEntityID); err != nil {
		return policy, err
	}
	exactActor, err := a.exactRecordActor(ctx, actor, tenant, current.Loss.TenantID, current.Loss.LegalEntityID)
	if err != nil {
		return policy, err
	}
	ctx = identity.WithActor(ctx, exactActor)
	if r != nil {
		*r = *r.WithContext(ctx)
	}
	delete(payload, "legal_entity_id")
	payload["loss_id"] = current.Loss.ID

	if name == "loss.update" || name == "loss.recovery.record" {
		if err := a.validateStoredResponsibilityActor(
			ctx,
			tenant,
			current.Loss.LegalEntityID,
			"OPERATIONAL_LOSS",
			current.Loss.ID,
			name,
			policy.Materiality,
			authority.ResponsibilityOwner,
			current.Loss.OwnerPrincipalID,
		); err != nil {
			return policy, err
		}
	}
	return policy, nil
}
