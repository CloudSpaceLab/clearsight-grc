package documentimport

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

type HandoffAuthorityResolver struct{ Authority authority.Service }

func (r HandoffAuthorityResolver) Resolve(ctx context.Context, document Document, handoff ProposalHandoff, actorID string, at time.Time) (*HandoffRoute, error) {
	if r.Authority == nil {
		return &HandoffRoute{Status: "UNAVAILABLE"}, nil
	}
	var responsibility authority.Responsibility
	var decisionType string
	excluded := []string{handoff.IntakePrincipalID}
	switch handoff.Status {
	case HandoffAwaitingReview:
		responsibility = authority.ResponsibilityReviewer
		decisionType = "document.proposal.review"
	case HandoffAwaitingAuthorization:
		responsibility = authority.ResponsibilityAuthorizer
		decisionType = "document.proposal.authorize"
		excluded = append(excluded, handoff.ReviewerPrincipalID)
	default:
		return nil, nil
	}
	route := &HandoffRoute{Responsibility: string(responsibility)}
	if strings.TrimSpace(document.LegalEntityID) == "" {
		route.Status = "NO_LEGAL_ENTITY"
		return route, nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	resolution, err := r.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: document.TenantID, LegalEntityID: document.LegalEntityID,
		ObjectType: "DOCUMENT_IMPORT", ObjectID: document.ID,
		Responsibility: responsibility, DecisionType: decisionType, Materiality: 3, At: at,
	})
	switch {
	case err == nil:
		route.RuleID = resolution.RuleID
		route.PolicyVersion = resolution.PolicyVersion
		route.Explanation = resolution.Explanation
		principals := independentHandoffPrincipals(resolution, excluded...)
		switch len(principals) {
		case 0:
			route.Status = "NO_INDEPENDENT_CANDIDATE"
		case 1:
			route.Status = "DIRECT"
			route.PrincipalID = principals[0].ID
			route.PrincipalName = principals[0].DisplayName
			route.IsCurrentActor = strings.TrimSpace(actorID) != "" && route.PrincipalID == strings.TrimSpace(actorID)
		default:
			route.Status = "CANDIDATE_SET"
		}
	case errors.Is(err, authority.ErrNoRoute):
		route.Status = "NO_ROUTE"
	case errors.Is(err, authority.ErrAmbiguousRoute):
		route.Status = "AMBIGUOUS_ROUTE"
	default:
		return nil, err
	}
	return route, nil
}

func independentHandoffPrincipals(resolution authority.Resolution, excluded ...string) []authority.Principal {
	excludedIDs := map[string]struct{}{}
	for _, value := range excluded {
		if value = strings.TrimSpace(value); value != "" {
			excludedIDs[value] = struct{}{}
		}
	}
	values := append([]authority.Principal(nil), resolution.CandidatePrincipals...)
	if len(values) == 0 && strings.TrimSpace(resolution.Principal.ID) != "" {
		values = append(values, resolution.Principal)
	}
	seen := map[string]struct{}{}
	result := make([]authority.Principal, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value.ID)
		if id == "" || value.Kind != "PERSON" {
			continue
		}
		if _, blocked := excludedIDs[id]; blocked {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
