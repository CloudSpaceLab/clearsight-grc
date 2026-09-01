package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
)

type formPolicySubjectRegistry interface {
	SupportsSubjectType(string) bool
}

type formPolicyActivationAuthority struct {
	Automation *autonomy.Service
	Authority  authority.Service
	Subjects   formPolicySubjectRegistry
	Now        func() time.Time
}

func (validator formPolicyActivationAuthority) ValidatePolicyActivation(ctx context.Context, actor formpolicy.Actor, policy formpolicy.Policy) error {
	if validator.Automation == nil || validator.Authority == nil || validator.Subjects == nil {
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
	if automationPolicy.Status != autonomy.AutomationPolicyActive || automationPolicy.ActionClass != formpolicy.ActionClassCreateMatter || strings.TrimSpace(automationPolicy.Checksum) == "" || automationPolicy.EffectiveFrom != nil && automationPolicy.EffectiveFrom.After(now) || automationPolicy.EffectiveUntil != nil && !automationPolicy.EffectiveUntil.After(now) || !automationPolicyAllowsFormPolicy(automationPolicy, policy) {
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
		if !validator.Subjects.SupportsSubjectType(subjectType) {
			return formpolicy.ErrActivationAuthority
		}
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

func automationPolicyAllowsFormPolicy(approved autonomy.AutomationPolicy, policy formpolicy.Policy) bool {
	if !strings.EqualFold(strings.TrimSpace(approved.RolloutMode), string(policy.Rollout)) {
		return false
	}
	eligibility, err := json.Marshal(policy.Eligibility)
	if err != nil || !sameJSON(approved.Eligibility, eligibility) {
		return false
	}
	blastRadius, err := json.Marshal(policy.BlastRadius)
	if err != nil || !sameJSON(approved.BlastRadiusLimit, blastRadius) {
		return false
	}
	outcome, err := json.Marshal(policy.Outcome)
	return err == nil && sameJSON(approved.VerificationContract, outcome)
}

func sameJSON(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	leftCanonical, _ := json.Marshal(leftValue)
	rightCanonical, _ := json.Marshal(rightValue)
	return bytes.Equal(leftCanonical, rightCanonical)
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
