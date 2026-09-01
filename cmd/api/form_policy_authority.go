package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
)

type formPolicyActivationAuthority struct {
	Automation *autonomy.Service
	Authority  authority.Service
	Now        func() time.Time
}

func (validator formPolicyActivationAuthority) ValidatePolicyActivation(ctx context.Context, actor formpolicy.Actor, policy formpolicy.Policy) error {
	if validator.Automation == nil || validator.Authority == nil {
		return formpolicy.ErrAuthorityUnavailable
	}
	automationPolicy, err := validator.Automation.GetAutomationPolicy(ctx, actor.TenantID, policy.AutomationPolicyID, policy.AutomationPolicyVersion)
	if errors.Is(err, autonomy.ErrAutomationPolicyNotFound) {
		return formpolicy.ErrActivationAuthority
	}
	if err != nil {
		return formpolicy.ErrAuthorityUnavailable
	}
	now := time.Now().UTC()
	if validator.Now != nil {
		now = validator.Now().UTC()
	}
	if automationPolicy.Status != autonomy.AutomationPolicyActive || automationPolicy.ActionClass != formpolicy.ActionClassCreateMatter || automationPolicy.EffectiveFrom != nil && automationPolicy.EffectiveFrom.After(now) || automationPolicy.EffectiveUntil != nil && !automationPolicy.EffectiveUntil.After(now) {
		return formpolicy.ErrActivationAuthority
	}
	authorizer, err := validator.resolve(ctx, authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID,
		Responsibility: authority.ResponsibilityAuthorizer, DecisionType: "forms.response-policy.activate", Materiality: 5,
	})
	if err != nil || !authorizer.AllowsPrincipal(actor.PrincipalID) {
		return errOrActivationAuthority(err)
	}
	if _, err := validator.resolve(ctx, authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "MATTER", ObjectID: "*",
		Responsibility: authority.ResponsibilityOwner, DecisionType: "matter.create", Materiality: policy.Action.Priority,
	}); err != nil {
		return err
	}
	for _, subjectType := range policy.Eligibility.SubjectTypes {
		if _, err := validator.resolve(ctx, authority.ResolveInput{
			TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: strings.ToUpper(subjectType), ObjectID: "*",
			Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.response-policy.subject.resolve", Materiality: policy.Action.Priority,
		}); err != nil {
			return err
		}
	}
	serviceRoute, err := validator.resolve(ctx, authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID,
		Responsibility: authority.ResponsibilityPerformer, DecisionType: "forms.response-policy.execute", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return err
	}
	if !resolutionHasServicePrincipal(serviceRoute) {
		return formpolicy.ErrActivationAuthority
	}
	return nil
}

func (validator formPolicyActivationAuthority) resolve(ctx context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	value, err := validator.Authority.Resolve(ctx, input)
	if err == nil {
		return value, nil
	}
	if errors.Is(err, authority.ErrNoRoute) || errors.Is(err, authority.ErrAmbiguousRoute) || errors.Is(err, authority.ErrInvalidInput) {
		return authority.Resolution{}, formpolicy.ErrActivationAuthority
	}
	return authority.Resolution{}, formpolicy.ErrAuthorityUnavailable
}

func errOrActivationAuthority(err error) error {
	if err != nil {
		return err
	}
	return formpolicy.ErrActivationAuthority
}

func resolutionHasServicePrincipal(value authority.Resolution) bool {
	if strings.EqualFold(value.Principal.Kind, "SERVICE") {
		return true
	}
	for _, candidate := range value.CandidatePrincipals {
		if strings.EqualFold(candidate.Kind, "SERVICE") {
			return true
		}
	}
	return false
}
