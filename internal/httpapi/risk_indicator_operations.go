package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func (a *API) getRiskIndicatorOperations(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	aggregate, err := service.Get(r.Context(), scope, r.PathValue("id"))
	if err != nil {
		writeRiskError(w, err)
		return
	}
	actor, err = a.exactRecordActor(r.Context(), actor, actor.TenantID, aggregate.Risk.TenantID, aggregate.Risk.LegalEntityID)
	if err != nil {
		writeRiskError(w, risk.ErrNotFound)
		return
	}
	r = r.WithContext(identity.WithActor(r.Context(), actor))
	result := struct {
		RiskID             string `json:"risk_id"`
		RiskVersion        int64  `json:"risk_version"`
		CanAct             bool   `json:"can_act"`
		AuthorityAvailable bool   `json:"authority_available"`
	}{RiskID: aggregate.Risk.ID, RiskVersion: aggregate.Risk.Version}
	if a.deps.Authority != nil {
		resolution, err := a.deps.Authority.Resolve(r.Context(), authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "RISK", ObjectID: aggregate.Risk.ID, Responsibility: authority.ResponsibilityOwner, DecisionType: "risk.indicator.link", Materiality: 3, At: time.Now().UTC()})
		if err == nil {
			result.AuthorityAvailable = true
			result.CanAct = resolution.AllowsPrincipalFor(actor.PrincipalID, strings.TrimSpace(aggregate.Risk.OwnerPrincipalID))
		}
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
