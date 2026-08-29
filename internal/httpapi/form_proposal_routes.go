package httpapi

import "net/http"

func (a *API) formProposalRoutes() []routeSpec {
	return []routeSpec{
		write(http.MethodPost, "/api/v1/document-imports/{id}/form-template-proposals", a.createDocumentFormProposal, nil),
		write(http.MethodPost, "/api/v1/forms/proposals/ai", a.createAIFormProposal, nil),
		write(http.MethodPost, "/api/v1/forms/templates/{id}/revisions/{version}/ai-proposals", a.createAIFormRevisionProposal, nil),
		read("/api/v1/forms/proposals/{id}", a.getFormProposal),
		write(http.MethodPost, "/api/v1/forms/proposals/{id}/accept", a.acceptFormProposal, nil),
		write(http.MethodPost, "/api/v1/forms/proposals/{id}/reject", a.rejectFormProposal, nil),
	}
}
