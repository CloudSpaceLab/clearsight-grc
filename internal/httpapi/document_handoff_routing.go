package httpapi

import (
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) withDocumentHandoffRoutes(r *http.Request, actor identity.Actor, document documentimport.Document) documentimport.Document {
	if a.deps.Authority == nil {
		return document
	}
	resolver := documentimport.HandoffAuthorityResolver{Authority: a.deps.Authority}
	now := time.Now().UTC()
	for index := range document.Proposals {
		handoff := document.Proposals[index].Handoff
		if handoff == nil {
			continue
		}
		route, err := resolver.Resolve(r.Context(), document, *handoff, actor.PrincipalID, now)
		if err != nil {
			handoff.Route = &documentimport.HandoffRoute{Status: "UNAVAILABLE"}
			if a.deps.Logger != nil {
				a.deps.Logger.WarnContext(r.Context(), "document proposal route resolution failed", "document_id", document.ID, "proposal_id", document.Proposals[index].ID, "error", err)
			}
			continue
		}
		handoff.Route = route
	}
	return document
}
