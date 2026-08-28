package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func (a *API) formDistributionRoutes() []routeSpec {
	owner := func(path, command string, handler http.HandlerFunc) routeSpec {
		return material(path, command, handler, commandPolicy{ObjectType: "FORM_DISTRIBUTION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner, Materiality: 3, ActorField: noActorField})
	}
	return []routeSpec{
		read("/api/v1/forms/distributions", a.listFilteredFormDistributions),
		material("/api/v1/forms/distributions", "forms.distribution.create", a.dispatchFormDistribution, commandPolicy{ObjectType: "LEGAL_ENTITY", Responsibility: authority.ResponsibilityOwner, Materiality: 3, BindLegalEntity: true, ActorField: noActorField}),
		read("/api/v1/forms/distributions/{id}", a.getFormDistribution),
		owner("/api/v1/forms/distributions/{id}/amend", "forms.distribution.amend", a.amendFormDistribution),
		owner("/api/v1/forms/distributions/{id}/access-routes/{route_id}/rotate", "forms.distribution.access.rotate", a.rotateFormDistributionAccessRoute),
		owner("/api/v1/forms/distributions/{id}/supersede", "forms.distribution.supersede", a.supersedeFormDistribution),
		owner("/api/v1/forms/distributions/{id}/lock", "forms.distribution.lock", a.lockFormDistribution),
		owner("/api/v1/forms/distributions/{id}/reopen", "forms.distribution.reopen", a.reopenFormDistribution),
		owner("/api/v1/forms/distributions/{id}/revoke", "forms.distribution.revoke", a.revokeFormDistribution),

		capability(http.MethodPost, "/api/v1/evidence/access/start", a.startFormAccess),
		capability(http.MethodPost, "/api/v1/evidence/access/otp/send", a.sendFormAccessOTP),
		capability(http.MethodPost, "/api/v1/evidence/access/otp/verify", a.verifyFormAccessOTP),
		capability(http.MethodPost, "/api/v1/evidence/access/redeem", a.redeemFormAccess),
		capability(http.MethodGet, "/api/v1/evidence/session/workspace", a.getFormResponseWorkspace),
		capability(http.MethodPatch, "/api/v1/evidence/session/workspace", a.saveFormResponseWorkspace),
		capability(http.MethodPost, "/api/v1/evidence/session/workspace/submissions", a.submitFormResponseWorkspace),
	}
}
