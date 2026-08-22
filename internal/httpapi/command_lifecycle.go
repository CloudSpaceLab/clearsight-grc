package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
)

// lifecycleCommandPolicy resolves authority from the route-bound object and its
// current lifecycle state. Route identifiers are canonical: redundant body IDs
// may match them, but may never redirect authority evaluation to another object.
func (a *API) lifecycleCommandPolicy(ctx context.Context, r *http.Request, tenant, name string, payload map[string]any, policy commandPolicy) (commandPolicy, error) {
	if strings.HasPrefix(name, "document.proposal.") {
		return a.documentProposalCommandPolicy(ctx, r, tenant, name, payload, policy)
	}

	matterID, err := lifecycleMatterID(r, payload)
	if err != nil {
		return policy, err
	}

	var aggregate *continuity.MatterAggregate
	matterPriority := 0
	if existingMatterCommand(name) && matterID != "" {
		if a.deps.Continuity == nil {
			return policy, fmt.Errorf("continuity service is unavailable")
		}
		current, loadErr := a.deps.Continuity.GetMatter(ctx, tenant, matterID)
		if loadErr != nil {
			return policy, loadErr
		}
		aggregate = &current
		matterPriority = current.Matter.Priority
	}

	switch name {
	case "matter.transition":
		if aggregate == nil {
			return policy, nil
		}
		target := continuity.MatterStatus(strings.ToUpper(stringValue(payload["to"])))
		if target == continuity.MatterDecisionRequired || target == continuity.MatterClosed || target == continuity.MatterCancelled || aggregate.Matter.Status == continuity.MatterClosed {
			policy.Responsibility = authority.ResponsibilityAuthorizer
			policy.Materiality = max(4, matterPriority)
		} else {
			policy.Materiality = max(policy.Materiality, matterPriority)
		}
		return policy, nil

	case "matter.decision.record":
		if aggregate == nil {
			return policy, nil
		}
		decisionType := stringValue(payload["type"])
		target := continuity.DecisionStatus(strings.ToUpper(stringValue(payload["status"])))
		if err := continuity.ValidateDecisionLifecycle(aggregate.Decisions, decisionType, target); err != nil {
			return policy, err
		}
		lifecycle, err := continuity.DecisionLifecyclePolicy(target)
		if err != nil {
			return policy, err
		}
		policy.ActorField = "authority_principal_id"
		policy.Responsibility = authority.Responsibility(lifecycle.Responsibility)
		policy.Materiality = max(lifecycle.Materiality, matterPriority)
		return policy, nil

	case "matter.action.add":
		if aggregate != nil {
			policy.Materiality = max(policy.Materiality, matterPriority)
			ownerID := stringValue(payload["owner_principal_id"])
			if ownerID != "" && !continuity.MatterVisibleTo(aggregate.Matter, ownerID) {
				return policy, fmt.Errorf("%w: action owner is not permitted to view this issue", continuity.ErrInvalidState)
			}
		}
		return policy, nil

	case "matter.action.transition":
		if _, err := lifecycleSubresourceID(r, payload, "action_id"); err != nil {
			return policy, err
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		return policy, nil

	case "matter.response.add":
		policy.Responsibility = authority.ResponsibilityProposer
		policy.Materiality = max(2, matterPriority)
		return policy, nil

	case "matter.response.transition":
		if aggregate == nil {
			return policy, nil
		}
		responseID, err := lifecycleSubresourceID(r, payload, "response_id")
		if err != nil {
			return policy, err
		}
		if responseID == "" {
			return policy, nil
		}
		var current *continuity.ResponsePackage
		for index := range aggregate.ResponsePackages {
			if aggregate.ResponsePackages[index].ID == responseID {
				value := aggregate.ResponsePackages[index]
				current = &value
				break
			}
		}
		if current == nil {
			return policy, continuity.ErrNotFound
		}
		target := continuity.ResponseStatus(strings.ToUpper(stringValue(payload["to"])))
		lifecycle, err := continuity.ResponseLifecyclePolicy(current.Status, target)
		if err != nil {
			return policy, err
		}
		policy.Responsibility = authority.Responsibility(lifecycle.Responsibility)
		policy.Materiality = max(lifecycle.Materiality, matterPriority)
		return policy, nil

	default:
		if aggregate != nil {
			policy.Materiality = max(policy.Materiality, matterPriority)
		}
		return policy, nil
	}
}

func (a *API) documentProposalCommandPolicy(ctx context.Context, r *http.Request, tenant, name string, payload map[string]any, policy commandPolicy) (commandPolicy, error) {
	if a.deps.DocumentImports == nil {
		return policy, fmt.Errorf("document import service is unavailable")
	}
	documentID, err := boundLifecycleID(strings.TrimSpace(r.PathValue("id")), stringValue(payload["document_id"]), "document_id")
	if err != nil {
		return policy, err
	}
	proposalID, err := boundLifecycleID(strings.TrimSpace(r.PathValue("proposal_id")), stringValue(payload["proposal_id"]), "proposal_id")
	if err != nil {
		return policy, err
	}
	if documentID == "" || proposalID == "" {
		return policy, fmt.Errorf("%w: document and proposal identifiers are required", continuity.ErrInvalidState)
	}
	document, err := a.deps.DocumentImports.Get(ctx, tenant, documentID)
	if err != nil {
		return policy, err
	}
	if strings.TrimSpace(document.LegalEntityID) == "" {
		return policy, fmt.Errorf("%w: the imported document has no governed legal-entity scope", continuity.ErrInvalidState)
	}
	var proposal *documentimport.Proposal
	for index := range document.Proposals {
		if document.Proposals[index].ID == proposalID {
			proposal = &document.Proposals[index]
			break
		}
	}
	if proposal == nil || proposal.Handoff == nil || proposal.Status != documentimport.ProposalAccepted {
		return policy, fmt.Errorf("%w: accepted proposal handoff not found", continuity.ErrInvalidState)
	}
	switch name {
	case "document.proposal.review":
		if proposal.Handoff.Status != documentimport.HandoffAwaitingReview {
			return policy, fmt.Errorf("%w: proposal is not awaiting review", continuity.ErrInvalidState)
		}
		policy.Responsibility = authority.ResponsibilityReviewer
		policy.Materiality = max(policy.Materiality, 3)
	case "document.proposal.authorize":
		if proposal.Handoff.Status != documentimport.HandoffAwaitingAuthorization {
			return policy, fmt.Errorf("%w: proposal is not awaiting authorization", continuity.ErrInvalidState)
		}
		policy.Responsibility = authority.ResponsibilityAuthorizer
		policy.Materiality = max(policy.Materiality, 4)
	default:
		return policy, fmt.Errorf("%w: unsupported document proposal command", continuity.ErrInvalidState)
	}
	// The legal entity used for authority is derived from the current imported
	// document, never from caller input. This also gives bank-wide operators a
	// concrete governed scope instead of authorizing against "*".
	payload["document_id"] = documentID
	payload["proposal_id"] = proposalID
	payload["legal_entity_id"] = document.LegalEntityID
	return policy, nil
}

func existingMatterCommand(name string) bool {
	return strings.HasPrefix(name, "matter.") && name != "matter.create"
}

func lifecycleMatterID(r *http.Request, payload map[string]any) (string, error) {
	pathID := ""
	if r != nil {
		pathID = strings.TrimSpace(r.PathValue("id"))
	}
	return boundLifecycleID(pathID, stringValue(payload["matter_id"]), "matter_id")
}

func lifecycleSubresourceID(r *http.Request, payload map[string]any, field string) (string, error) {
	pathID := ""
	if r != nil {
		pathID = strings.TrimSpace(r.PathValue(field))
	}
	return boundLifecycleID(pathID, stringValue(payload[field]), field)
}

func boundLifecycleID(routeID, bodyID, field string) (string, error) {
	routeID = strings.TrimSpace(routeID)
	bodyID = strings.TrimSpace(bodyID)
	if routeID != "" && bodyID != "" && routeID != bodyID {
		return "", fmt.Errorf("%w: %s conflicts with the route identifier", continuity.ErrInvalidState, field)
	}
	if routeID != "" {
		return routeID, nil
	}
	return bodyID, nil
}
