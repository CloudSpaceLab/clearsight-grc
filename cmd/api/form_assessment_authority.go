package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func formAssessmentAuthorizer(service authority.Service) func(context.Context, identity.Actor, evidence.CompletedResponseSummary, formcontract.Field) (string, error) {
	return func(ctx context.Context, actor identity.Actor, response evidence.CompletedResponseSummary, field formcontract.Field) (string, error) {
		if field.Assessment == nil {
			return "", fmt.Errorf("assessment authority is unavailable")
		}
		route, err := resolveFormResponseReviewer(ctx, service, actor, response, "forms.response.assess")
		if err != nil {
			return "", err
		}
		roleAllowed := route.AllowsPrincipalWithRole(actor.PrincipalID, field.Assessment.ReviewerRole)
		if !roleAllowed {
			return "", fmt.Errorf("the field requires the configured bank reviewer role")
		}
		return route.PolicyVersion + ":" + route.RuleID, nil
	}
}

func formResponseDiscoveryAuthorizer(service authority.Service) evidence.ResponseDiscoveryAuthorizer {
	return func(ctx context.Context, actor identity.Actor, response evidence.CompletedResponseSummary) error {
		_, err := resolveFormResponseReviewer(ctx, service, actor, response, "forms.response.assess")
		return err
	}
}

func formResponseReviewAuthorizer(service authority.Service) evidence.ResponseReviewAuthorizer {
	return func(ctx context.Context, actor identity.Actor, response evidence.CompletedResponseSummary) (string, error) {
		route, err := resolveFormResponseReviewer(ctx, service, actor, response, "forms.response.review")
		if err != nil {
			return "", err
		}
		return route.PolicyVersion + ":" + route.RuleID, nil
	}
}

// Whole-response review and field assessment are separate decisions, but both
// use the same scope and current-reviewer checks.
func resolveFormResponseReviewer(ctx context.Context, service authority.Service, actor identity.Actor, response evidence.CompletedResponseSummary, decision string) (authority.Resolution, error) {
	if service == nil || strings.TrimSpace(actor.PrincipalID) == "" || strings.TrimSpace(actor.TenantID) == "" || actor.TenantID == "*" || actor.TenantID != response.TenantID || (actor.LegalEntityID != response.LegalEntityID && actor.LegalEntityID != "*") || strings.TrimSpace(response.LegalEntityID) == "" || response.LegalEntityID == "*" || strings.TrimSpace(response.ID) == "" || response.ID == "*" {
		return authority.Resolution{}, fmt.Errorf("response reviewer authority is unavailable")
	}
	route, err := service.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: response.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: response.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: decision, Materiality: 3})
	if err != nil {
		return authority.Resolution{}, err
	}
	if !route.AllowsPrincipal(actor.PrincipalID) || route.RuleID == "" || route.PolicyVersion == "" {
		return authority.Resolution{}, fmt.Errorf("current response reviewer route is required")
	}
	return route, nil
}

func internalFormResponseAuthorizer(service authority.Service) evidence.InternalWorkspaceAuthorizer {
	return func(ctx context.Context, actor identity.Actor, request evidence.Request, command string) (string, error) {
		if service == nil || actor.PrincipalID == "" || actor.Kind != "PERSON" || actor.TenantID != request.TenantID || actor.LegalEntityID == "" || actor.LegalEntityID == "*" || actor.LegalEntityID != request.LegalEntityID || request.ID == "" || (command != "forms.response.save" && command != "forms.response.submit" && command != "forms.response.upload") {
			return "", fmt.Errorf("response authority unavailable")
		}
		route, err := service.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "EVIDENCE_REQUEST", ObjectID: request.ID, Responsibility: authority.ResponsibilityPerformer, DecisionType: command, Materiality: 3})
		if err != nil {
			return "", err
		}
		if !route.AllowsPrincipal(actor.PrincipalID) || route.RuleID == "" || route.PolicyVersion == "" {
			return "", fmt.Errorf("current response performer route required")
		}
		return route.PolicyVersion + ":" + route.RuleID, nil
	}
}
