package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

// riskLifecycleCommandPolicy binds existing Risk commands to the exact stored
// record before the generic command guard runs. Risk updates additionally
// require the stored owner (or a delegate whose authority originates from that
// owner), rather than any person who happens to be an eligible Risk owner.
func (a *API) riskLifecycleCommandPolicy(ctx context.Context, r *http.Request, tenant, name string, payload map[string]any, policy commandPolicy) (commandPolicy, error) {
	if name == "risk.create" {
		return policy, nil
	}
	if a == nil || a.deps.Risk == nil {
		return policy, fmt.Errorf("%w: risk service is unavailable", commandauth.ErrGuardUnavailable)
	}
	actor, err := identity.Require(ctx)
	if err != nil {
		return policy, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
	}
	legalEntityID := strings.TrimSpace(actor.LegalEntityID)
	if legalEntityID == "" || legalEntityID == "*" {
		// The Risk handler retains the user-facing exact-entity requirement.
		// Do not invent an entity from request data for tenant-wide identities.
		return policy, nil
	}
	riskID := ""
	if r != nil {
		riskID = strings.TrimSpace(r.PathValue("id"))
	}
	if riskID == "" {
		return policy, risk.ErrNotFound
	}
	current, err := a.deps.Risk.Get(ctx, risk.Scope{TenantID: tenant, LegalEntityID: legalEntityID}, riskID)
	if err != nil {
		return policy, err
	}
	if err := validateRequestedRecordEntity(actor, stringValue(payload["legal_entity_id"]), current.Risk.LegalEntityID); err != nil {
		return policy, err
	}
	exactActor, err := a.exactRecordActor(ctx, actor, tenant, current.Risk.TenantID, current.Risk.LegalEntityID)
	if err != nil {
		return policy, err
	}
	ctx = identity.WithActor(ctx, exactActor)
	if r != nil {
		*r = *r.WithContext(ctx)
	}
	delete(payload, "legal_entity_id")
	payload["risk_id"] = current.Risk.ID

	if name == "risk.update" || name == "risk.control.link" {
		if err := a.validateStoredResponsibilityActor(
			ctx,
			tenant,
			current.Risk.LegalEntityID,
			"RISK",
			current.Risk.ID,
			name,
			policy.Materiality,
			authority.ResponsibilityOwner,
			current.Risk.OwnerPrincipalID,
		); err != nil {
			return policy, err
		}
	}
	return policy, nil
}
