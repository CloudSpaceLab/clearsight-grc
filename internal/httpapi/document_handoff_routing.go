package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var (
	errDocumentHandoffRoutingUnresolved = errors.New("document proposal handoff routing is unresolved")
	errDocumentHandoffNotAssigned       = errors.New("document proposal handoff is assigned to another actor")
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

func (a *API) requireDocumentHandoffRoute(r *http.Request, actor identity.Actor, document documentimport.Document, handoff documentimport.ProposalHandoff) error {
	if a.deps.Authority == nil {
		return nil
	}
	resolver := documentimport.HandoffAuthorityResolver{Authority: a.deps.Authority}
	route, err := resolver.Resolve(r.Context(), document, handoff, actor.PrincipalID, time.Now().UTC())
	if err != nil {
		return err
	}
	if route == nil || route.Status != "DIRECT" {
		return errDocumentHandoffRoutingUnresolved
	}
	if !route.IsCurrentActor {
		return errDocumentHandoffNotAssigned
	}
	return nil
}
