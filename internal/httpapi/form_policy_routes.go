package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) formPolicyRoutes() []routeSpec {
	command := func(path, name string, responsibility authority.Responsibility, materiality int, handler http.HandlerFunc) routeSpec {
		return withPermission(material(path, name, handler, commandPolicy{
			ObjectType: "FORM_RESPONSE_POLICY", ObjectIDPath: "id", Responsibility: responsibility,
			Materiality: materiality, BindLegalEntity: true, ActorField: noActorField,
		}), identity.PermissionConfigWrite)
	}
	base := "/api/v1/config/form-response-policies"
	return []routeSpec{
		withPermission(read(base, a.listFormPolicies), identity.PermissionConfigRead),
		command(base, "forms.response-policy.create", authority.ResponsibilityOwner, 4, a.createFormPolicy),
		withPermission(read(base+"/{id}", a.getFormPolicy), identity.PermissionConfigRead),
		command(base+"/{id}/simulate", "forms.response-policy.simulate", authority.ResponsibilityReviewer, 4, a.simulateFormPolicy),
		command(base+"/{id}/submit", "forms.response-policy.submit", authority.ResponsibilityProposer, 4, a.submitFormPolicy),
		command(base+"/{id}/approve", "forms.response-policy.approve", authority.ResponsibilityAuthorizer, 5, a.approveFormPolicy),
		command(base+"/{id}/activate", "forms.response-policy.activate", authority.ResponsibilityAuthorizer, 5, a.activateFormPolicy),
		command(base+"/{id}/suspend", "forms.response-policy.suspend", authority.ResponsibilityAuthorizer, 5, a.suspendFormPolicy),
		command(base+"/{id}/rollback", "forms.response-policy.rollback", authority.ResponsibilityAuthorizer, 5, a.rollbackFormPolicy),
		write(http.MethodPost, "/api/v1/config/form-templates/{id}/score-preview", a.previewFormScore, nil),
	}
}
