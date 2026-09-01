package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
)

type formPolicyExecutionAuthority struct {
	Automation *autonomy.Service
	Authority  authority.Service
	Subjects   evidence.SubjectScopeResolver
	Now        func() time.Time
}

func (validator formPolicyExecutionAuthority) ResolvePolicyExecution(ctx context.Context, policy formpolicy.Policy, response evidence.CompletedResponseSummary) (formpolicy.ExecutionRoute, error) {
	if validator.Automation == nil || validator.Authority == nil || validator.Subjects == nil {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrAuthorityUnavailable
	}
	automationPolicy, err := validator.Automation.GetAutomationPolicy(ctx, policy.TenantID, policy.AutomationPolicyID, policy.AutomationPolicyVersion)
	if errors.Is(err, autonomy.ErrAutomationPolicyNotFound) {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
	}
	if err != nil {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrAuthorityUnavailable
	}
	now := time.Now().UTC()
	if validator.Now != nil {
		now = validator.Now().UTC()
	}
	if !formpolicy.AutomationPolicyAllows(automationPolicy, policy, now) {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
	}
	scope, err := validator.Subjects.ResolveSubjectScope(ctx, response.TenantID, response.SubjectType, response.SubjectID)
	if err != nil {
		if errors.Is(err, evidence.ErrSubjectUnsupported) || errors.Is(err, evidence.ErrSubjectScopeMismatch) || errors.Is(err, evidence.ErrNotFound) {
			return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
		}
		return formpolicy.ExecutionRoute{}, formpolicy.ErrAuthorityUnavailable
	}
	if scope.TenantID != policy.TenantID || scope.LegalEntityID != policy.LegalEntityID || !strings.EqualFold(scope.SubjectType, response.SubjectType) || scope.SubjectID != response.SubjectID {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
	}
	subjectOwner, err := validator.resolve(ctx, authority.ResolveInput{TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: strings.ToUpper(scope.SubjectType), ObjectID: scope.SubjectID, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.response-policy.subject.resolve", Materiality: policy.Action.Priority})
	if err != nil {
		return formpolicy.ExecutionRoute{}, err
	}
	subjectOwnerID := strings.TrimSpace(subjectOwner.Principal.ID)
	if subjectOwnerID == "" {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
	}
	owner, err := validator.resolve(ctx, authority.ResolveInput{TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "MATTER", ObjectID: "*", Responsibility: authority.ResponsibilityOwner, DecisionType: "matter.create", Materiality: policy.Action.Priority})
	if err != nil {
		return formpolicy.ExecutionRoute{}, err
	}
	if strings.TrimSpace(owner.Principal.ID) == "" {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
	}
	reviewer, err := validator.resolve(ctx, authority.ResolveInput{TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "MATTER", ObjectID: "*", Responsibility: authority.ResponsibilityReviewer, DecisionType: "matter.verify", Materiality: policy.Action.Priority})
	if err != nil {
		return formpolicy.ExecutionRoute{}, err
	}
	reviewerID := strings.TrimSpace(reviewer.Principal.ID)
	if reviewerID == "" {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
	}
	performer, err := validator.resolve(ctx, authority.ResolveInput{TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID, Responsibility: authority.ResponsibilityPerformer, DecisionType: "forms.response-policy.execute", Materiality: policy.Action.Priority})
	if err != nil {
		return formpolicy.ExecutionRoute{}, err
	}
	serviceID := servicePrincipalID(performer)
	if serviceID == "" {
		return formpolicy.ExecutionRoute{}, formpolicy.ErrActivationAuthority
	}
	programID := ""
	if strings.EqualFold(scope.SubjectType, "PROGRAM") {
		programID = scope.SubjectID
	}
	return formpolicy.ExecutionRoute{TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, CanonicalSubjectType: strings.ToUpper(scope.SubjectType), CanonicalSubjectID: scope.SubjectID, ServicePrincipalID: serviceID, OwnerPrincipalID: subjectOwnerID, ReviewerPrincipalID: reviewerID, ProgramID: programID}, nil
}

func (validator formPolicyExecutionAuthority) ResolvePolicyExecutionException(ctx context.Context, policy formpolicy.Policy, response evidence.CompletedResponseSummary) (formpolicy.ExecutionExceptionRoute, error) {
	if validator.Authority == nil || policy.TenantID == "" || policy.LegalEntityID == "" || response.TenantID != policy.TenantID || response.LegalEntityID != policy.LegalEntityID {
		return formpolicy.ExecutionExceptionRoute{}, formpolicy.ErrAuthorityUnavailable
	}
	resolution, err := validator.resolve(ctx, authority.ResolveInput{
		TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, ObjectType: "FORM_RESPONSE_POLICY", ObjectID: policy.ID,
		Responsibility: authority.ResponsibilityEscalation, DecisionType: "forms.response-policy.recover", Materiality: policy.Action.Priority,
	})
	if err != nil {
		return formpolicy.ExecutionExceptionRoute{}, err
	}
	principalID := strings.TrimSpace(resolution.Principal.ID)
	if principalID == "" {
		return formpolicy.ExecutionExceptionRoute{}, formpolicy.ErrActivationAuthority
	}
	return formpolicy.ExecutionExceptionRoute{TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, PrincipalID: principalID}, nil
}

func (validator formPolicyExecutionAuthority) resolve(ctx context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	value, err := validator.Authority.Resolve(ctx, input)
	if err == nil {
		return value, nil
	}
	if errors.Is(err, authority.ErrNoRoute) || errors.Is(err, authority.ErrAmbiguousRoute) || errors.Is(err, authority.ErrInvalidInput) {
		return authority.Resolution{}, formpolicy.ErrActivationAuthority
	}
	return authority.Resolution{}, formpolicy.ErrAuthorityUnavailable
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

var _ formpolicy.ExecutionAuthority = formPolicyExecutionAuthority{}
