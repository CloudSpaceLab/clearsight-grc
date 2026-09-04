package formpolicy

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type SubjectRegistry interface {
	SupportsSubjectType(string) bool
}

type ActivationAuthorityResolver struct {
	Automation *autonomy.Service
	Authority  authority.Service
	Subjects   SubjectRegistry
	Now        func() time.Time
}

func (validator ActivationAuthorityResolver) ValidatePolicyActivation(ctx context.Context, actor Actor, policy Policy) error {
	if validator.Automation == nil || validator.Authority == nil || validator.Subjects == nil {
		return ErrAuthorityUnavailable
	}
	automationPolicy, err := validator.Automation.GetAutomationPolicy(ctx, actor.TenantID, policy.AutomationPolicyID, policy.AutomationPolicyVersion)
	if errors.Is(err, autonomy.ErrAutomationPolicyNotFound) {
		return ErrActivationAuthority
	}
	if err != nil {
		return ErrAuthorityUnavailable
	}
	now := time.Now().UTC()
	if validator.Now != nil {
		now = validator.Now().UTC()
	}
	if !AutomationPolicyAllows(automationPolicy, policy, now) {
		return ErrActivationAuthority
	}
	authorizer, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID,
		Responsibility: authority.ResponsibilityAuthorizer, DecisionType: "forms.response-policy.activate", Materiality: 5,
	})
	if err != nil || !authorizer.AllowsPrincipal(actor.PrincipalID) {
		return errOrActivationAuthority(err)
	}
	for _, route := range []authority.ResolveInput{
		{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "MATTER", ObjectID: "*", Responsibility: authority.ResponsibilityOwner, DecisionType: "matter.create", Materiality: policy.Action.Priority},
		{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "MATTER", ObjectID: "*", Responsibility: authority.ResponsibilityReviewer, DecisionType: "matter.verify", Materiality: policy.Action.Priority},
	} {
		if _, err := resolveAuthority(ctx, validator.Authority, route); err != nil {
			return err
		}
	}
	for _, subjectType := range policy.Eligibility.SubjectTypes {
		if !validator.Subjects.SupportsSubjectType(subjectType) {
			return ErrActivationAuthority
		}
		if _, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
			TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: strings.ToUpper(subjectType), ObjectID: "*",
			Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.response-policy.subject.resolve", Materiality: policy.Action.Priority,
		}); err != nil {
			return err
		}
	}
	serviceRoute, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID,
		Responsibility: authority.ResponsibilityPerformer, DecisionType: "forms.response-policy.execute", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return err
	}
	if servicePrincipalID(serviceRoute) == "" {
		return ErrActivationAuthority
	}
	return nil
}

type ExecutionAuthorityResolver struct {
	Automation *autonomy.Service
	Authority  authority.Service
	Subjects   evidence.SubjectScopeResolver
	Now        func() time.Time
}

func (validator ExecutionAuthorityResolver) ResolvePolicyExecution(ctx context.Context, policy Policy, response evidence.CompletedResponseSummary) (ExecutionRoute, error) {
	if validator.Automation == nil || validator.Authority == nil || validator.Subjects == nil {
		return ExecutionRoute{}, ErrAuthorityUnavailable
	}
	automationPolicy, err := validator.Automation.GetAutomationPolicy(ctx, policy.TenantID, policy.AutomationPolicyID, policy.AutomationPolicyVersion)
	if errors.Is(err, autonomy.ErrAutomationPolicyNotFound) {
		return ExecutionRoute{}, ErrActivationAuthority
	}
	if err != nil {
		return ExecutionRoute{}, ErrAuthorityUnavailable
	}
	now := time.Now().UTC()
	if validator.Now != nil {
		now = validator.Now().UTC()
	}
	if !AutomationPolicyAllows(automationPolicy, policy, now) {
		return ExecutionRoute{}, ErrActivationAuthority
	}
	scope, err := validator.Subjects.ResolveSubjectScope(ctx, response.TenantID, response.SubjectType, response.SubjectID)
	if err != nil {
		if errors.Is(err, evidence.ErrSubjectUnsupported) || errors.Is(err, evidence.ErrSubjectScopeMismatch) || errors.Is(err, evidence.ErrNotFound) {
			return ExecutionRoute{}, ErrActivationAuthority
		}
		return ExecutionRoute{}, ErrAuthorityUnavailable
	}
	if scope.TenantID != policy.TenantID || scope.LegalEntityID != policy.LegalEntityID || !strings.EqualFold(scope.SubjectType, response.SubjectType) || scope.SubjectID != response.SubjectID {
		return ExecutionRoute{}, ErrActivationAuthority
	}
	subjectOwner, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
		TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: strings.ToUpper(scope.SubjectType), ObjectID: scope.SubjectID,
		Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.response-policy.subject.resolve", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return ExecutionRoute{}, err
	}
	subjectOwnerID := strings.TrimSpace(subjectOwner.Principal.ID)
	if subjectOwnerID == "" {
		return ExecutionRoute{}, ErrActivationAuthority
	}
	// The generic Matter-owner route proves that the legal entity has a current
	// governed creation route. The created Matter remains owned by the exact
	// canonical subject owner, matching the existing worker contract.
	matterOwner, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
		TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "MATTER", ObjectID: "*",
		Responsibility: authority.ResponsibilityOwner, DecisionType: "matter.create", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return ExecutionRoute{}, err
	}
	if strings.TrimSpace(matterOwner.Principal.ID) == "" {
		return ExecutionRoute{}, ErrActivationAuthority
	}
	reviewer, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
		TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "MATTER", ObjectID: "*",
		Responsibility: authority.ResponsibilityReviewer, DecisionType: "matter.verify", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return ExecutionRoute{}, err
	}
	reviewerID := strings.TrimSpace(reviewer.Principal.ID)
	if reviewerID == "" {
		return ExecutionRoute{}, ErrActivationAuthority
	}
	performer, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
		TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID,
		Responsibility: authority.ResponsibilityPerformer, DecisionType: "forms.response-policy.execute", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return ExecutionRoute{}, err
	}
	serviceID := servicePrincipalID(performer)
	if serviceID == "" {
		return ExecutionRoute{}, ErrActivationAuthority
	}
	programID := ""
	if strings.EqualFold(scope.SubjectType, "PROGRAM") {
		programID = scope.SubjectID
	}
	return ExecutionRoute{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		CanonicalSubjectType: strings.ToUpper(scope.SubjectType), CanonicalSubjectID: scope.SubjectID,
		ServicePrincipalID: serviceID, OwnerPrincipalID: subjectOwnerID, ReviewerPrincipalID: reviewerID, ProgramID: programID,
	}, nil
}

func (validator ExecutionAuthorityResolver) ResolvePolicyExecutionException(ctx context.Context, policy Policy, response evidence.CompletedResponseSummary) (ExecutionExceptionRoute, error) {
	if validator.Authority == nil || policy.TenantID == "" || policy.LegalEntityID == "" || response.TenantID != policy.TenantID || response.LegalEntityID != policy.LegalEntityID {
		return ExecutionExceptionRoute{}, ErrAuthorityUnavailable
	}
	resolution, err := resolveAuthority(ctx, validator.Authority, authority.ResolveInput{
		TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID,
		Responsibility: authority.ResponsibilityEscalation, DecisionType: "forms.response-policy.recover", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return ExecutionExceptionRoute{}, err
	}
	principalID := strings.TrimSpace(resolution.Principal.ID)
	if principalID == "" {
		return ExecutionExceptionRoute{}, ErrActivationAuthority
	}
	return ExecutionExceptionRoute{TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, PrincipalID: principalID}, nil
}

func resolveAuthority(ctx context.Context, service authority.Service, input authority.ResolveInput) (authority.Resolution, error) {
	value, err := service.Resolve(ctx, input)
	if err == nil {
		return value, nil
	}
	if errors.Is(err, authority.ErrNoRoute) || errors.Is(err, authority.ErrAmbiguousRoute) || errors.Is(err, authority.ErrInvalidInput) {
		return authority.Resolution{}, ErrActivationAuthority
	}
	return authority.Resolution{}, ErrAuthorityUnavailable
}

func errOrActivationAuthority(err error) error {
	if err != nil {
		return err
	}
	return ErrActivationAuthority
}

func servicePrincipalID(value authority.Resolution) string {
	if strings.EqualFold(value.Principal.Kind, "SERVICE") {
		return strings.TrimSpace(value.Principal.ID)
	}
	for _, candidate := range value.CandidatePrincipals {
		if strings.EqualFold(candidate.Kind, "SERVICE") {
			return strings.TrimSpace(candidate.ID)
		}
	}
	return ""
}

var _ ActivationAuthority = ActivationAuthorityResolver{}
var _ ExecutionAuthority = ExecutionAuthorityResolver{}
