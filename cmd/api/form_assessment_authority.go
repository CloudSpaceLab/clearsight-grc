package main

import (
	"context"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func formAssessmentAuthorizer(service authority.Service) func(context.Context, identity.Actor, evidence.CompletedResponseSummary, formcontract.Field) (string, error) {
	return func(ctx context.Context, actor identity.Actor, response evidence.CompletedResponseSummary, field formcontract.Field) (string, error) {
		if service == nil || actor.PrincipalID == "" || actor.TenantID != response.TenantID || (actor.LegalEntityID != response.LegalEntityID && actor.LegalEntityID != "*") || response.LegalEntityID == "" || response.LegalEntityID == "*" || field.Assessment == nil {
			return "", fmt.Errorf("assessment authority is unavailable")
		}
		route, err := service.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: response.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: response.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.assess", Materiality: 3})
		if err != nil {
			return "", err
		}
		if !route.AllowsPrincipal(actor.PrincipalID) || route.RuleID == "" || route.PolicyVersion == "" {
			return "", fmt.Errorf("current assessment reviewer route is required")
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
		if service == nil || actor.PrincipalID == "" || actor.TenantID != response.TenantID || (actor.LegalEntityID != response.LegalEntityID && actor.LegalEntityID != "*") || response.LegalEntityID == "" || response.LegalEntityID == "*" {
			return fmt.Errorf("response review authority is unavailable")
		}
		route, err := service.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: response.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: response.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.assess", Materiality: 3})
		if err != nil {
			return err
		}
		if !route.AllowsPrincipal(actor.PrincipalID) || route.RuleID == "" || route.PolicyVersion == "" {
			return fmt.Errorf("current response review route is required")
		}
		return nil
	}
}
