package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) formCommunicationRoutes() []routeSpec {
	configCommand := func(path, command string, handler http.HandlerFunc) routeSpec {
		return withPermission(material(path, command, handler, commandPolicy{
			ObjectType: "LEGAL_ENTITY", Responsibility: authority.ResponsibilityOwner,
			Materiality: 4, BindLegalEntity: true, ActorField: noActorField,
		}), identity.PermissionConfigWrite)
	}
	templateBase := "/api/v1/forms/communications/templates/{action}/{locale}/revisions/{version}"
	return []routeSpec{
		withPermission(read("/api/v1/forms/communications/profiles", a.listCommunicationProfiles), identity.PermissionConfigRead),
		configCommand("/api/v1/forms/communications/profiles", "forms.communication.profile.create", a.createCommunicationProfile),
		withPermission(read("/api/v1/forms/communications/profiles/{version}", a.getCommunicationProfile), identity.PermissionConfigRead),
		configCommand("/api/v1/forms/communications/profiles/{version}/transition", "forms.communication.profile.transition", a.transitionCommunicationProfile),
		configCommand("/api/v1/forms/communications/profiles/{version}/rollback", "forms.communication.profile.rollback", a.rollbackCommunicationProfile),

		withPermission(read("/api/v1/forms/communications/templates", a.listCommunicationTemplates), identity.PermissionConfigRead),
		configCommand("/api/v1/forms/communications/templates", "forms.communication.template.create", a.createCommunicationTemplate),
		withPermission(read(templateBase, a.getCommunicationTemplate), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, templateBase+"/preview", a.previewCommunicationTemplate, nil), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, templateBase+"/impact", a.impactCommunicationTemplate, nil), identity.PermissionConfigRead),
		configCommand(templateBase+"/transition", "forms.communication.template.transition", a.transitionCommunicationTemplate),
		configCommand(templateBase+"/rollback", "forms.communication.template.rollback", a.rollbackCommunicationTemplate),
		configCommand(templateBase+"/test-send", "forms.communication.template.test-send", a.testSendCommunicationTemplate),

		withPermission(read("/api/v1/forms/communications/brand-assets", a.listCommunicationBrandAssets), identity.PermissionConfigRead),
		configCommand("/api/v1/forms/communications/brand-assets", "forms.communication.brand.upload", a.uploadCommunicationBrandAsset),
	}
}
