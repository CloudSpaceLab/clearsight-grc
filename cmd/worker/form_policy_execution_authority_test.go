package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
)

type workerFormPolicyAuthorityStub struct {
	inputs []authority.ResolveInput
	err    error
}

func (stub *workerFormPolicyAuthorityStub) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	stub.inputs = append(stub.inputs, input)
	if stub.err != nil {
		return authority.Resolution{}, stub.err
	}
	principal := authority.Principal{ID: "owner-current", Kind: "PERSON"}
	if input.Responsibility == authority.ResponsibilityReviewer {
		principal.ID = "reviewer-current"
	}
	if input.Responsibility == authority.ResponsibilityPerformer {
		principal = authority.Principal{ID: "automation-service", Kind: "SERVICE"}
	}
	return authority.Resolution{Principal: principal, RuleID: "current-route", PolicyVersion: "v3"}, nil
}
func (*workerFormPolicyAuthorityStub) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (*workerFormPolicyAuthorityStub) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (*workerFormPolicyAuthorityStub) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

type workerSubjectResolverStub struct {
	scope evidence.SubjectScope
	err   error
}

func (stub workerSubjectResolverStub) ResolveSubjectScope(context.Context, string, string, string) (evidence.SubjectScope, error) {
	return stub.scope, stub.err
}

func TestWorkerFormPolicyExecutionAuthorityResolvesCanonicalSubjectOwnerAndService(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	policy := workerExecutionPolicy()
	automationPolicy := workerAutomationPolicy(policy)
	routes := &workerFormPolicyAuthorityStub{}
	validator := formPolicyExecutionAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy)), Authority: routes, Subjects: workerSubjectResolverStub{scope: evidence.SubjectScope{TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "subject-a"}}, Now: func() time.Time { return now }}
	response := workerExecutionResponse()
	resolved, err := validator.ResolvePolicyExecution(t.Context(), policy, response)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ServicePrincipalID != "automation-service" || resolved.OwnerPrincipalID != "owner-current" || resolved.ReviewerPrincipalID != "reviewer-current" || resolved.CanonicalSubjectID != response.SubjectID || len(routes.inputs) != 4 {
		t.Fatalf("resolved route = %#v inputs=%#v", resolved, routes.inputs)
	}
	if routes.inputs[0].ObjectID != response.SubjectID || routes.inputs[1].ObjectType != "MATTER" || routes.inputs[2].Responsibility != authority.ResponsibilityReviewer || routes.inputs[3].Responsibility != authority.ResponsibilityPerformer {
		t.Fatalf("current execution routes = %#v", routes.inputs)
	}
}

func TestWorkerFormPolicyExecutionAuthorityFailsClosed(t *testing.T) {
	policy := workerExecutionPolicy()
	automationPolicy := workerAutomationPolicy(policy)
	response := workerExecutionResponse()
	wrongScope := formPolicyExecutionAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy)), Authority: &workerFormPolicyAuthorityStub{}, Subjects: workerSubjectResolverStub{scope: evidence.SubjectScope{TenantID: "bank", LegalEntityID: "other-entity", SubjectType: response.SubjectType, SubjectID: response.SubjectID}}}
	if _, err := wrongScope.ResolvePolicyExecution(t.Context(), policy, response); !errors.Is(err, formpolicy.ErrActivationAuthority) {
		t.Fatalf("wrong canonical scope err = %v", err)
	}
	broader := automationPolicy
	broader.RolloutMode = string(formpolicy.RolloutEnforce)
	wrongAutomation := formPolicyExecutionAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(broader)), Authority: &workerFormPolicyAuthorityStub{}, Subjects: workerSubjectResolverStub{scope: evidence.SubjectScope{TenantID: "bank", LegalEntityID: "entity", SubjectType: response.SubjectType, SubjectID: response.SubjectID}}}
	if _, err := wrongAutomation.ResolvePolicyExecution(t.Context(), policy, response); !errors.Is(err, formpolicy.ErrActivationAuthority) {
		t.Fatalf("broader automation err = %v", err)
	}
	unavailable := formPolicyExecutionAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy)), Authority: &workerFormPolicyAuthorityStub{err: errors.New("database unavailable")}, Subjects: workerSubjectResolverStub{scope: evidence.SubjectScope{TenantID: "bank", LegalEntityID: "entity", SubjectType: response.SubjectType, SubjectID: response.SubjectID}}}
	if _, err := unavailable.ResolvePolicyExecution(t.Context(), policy, response); !errors.Is(err, formpolicy.ErrAuthorityUnavailable) {
		t.Fatalf("authority outage err = %v", err)
	}
}

func TestWorkerFormPolicyExecutionAuthorityRoutesOperationalExceptionToCurrentEscalationOwner(t *testing.T) {
	routes := &workerFormPolicyAuthorityStub{}
	validator := formPolicyExecutionAuthority{Authority: routes}
	resolved, err := validator.ResolvePolicyExecutionException(t.Context(), workerExecutionPolicy(), workerExecutionResponse())
	if err != nil || resolved.PrincipalID != "owner-current" || len(routes.inputs) != 1 || routes.inputs[0].Responsibility != authority.ResponsibilityEscalation || routes.inputs[0].DecisionType != "forms.response-policy.recover" {
		t.Fatalf("resolved=%#v inputs=%#v err=%v", resolved, routes.inputs, err)
	}
}

func workerExecutionPolicy() formpolicy.Policy {
	return formpolicy.Policy{ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2, Eligibility: formpolicy.Eligibility{SubjectTypes: []string{"VENDOR_RELATIONSHIP"}}, Action: formpolicy.MatterAction{Priority: 4}, BlastRadius: formpolicy.BlastRadius{PerRun: 10, PerDay: 25}, Outcome: formpolicy.OutcomeContract{ExpectedOutcome: "Current evidence is accepted.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"}, Rollout: formpolicy.RolloutShadow}
}

func workerAutomationPolicy(policy formpolicy.Policy) autonomy.AutomationPolicy {
	value := autonomy.AutomationPolicy{ID: "automation-a", TenantID: "bank", ActionClass: formpolicy.ActionClassCreateMatter, Status: autonomy.AutomationPolicyActive, Version: 2, RolloutMode: string(policy.Rollout), Checksum: "approved-v2"}
	value.Eligibility, _ = json.Marshal(policy.Eligibility)
	value.BlastRadiusLimit, _ = json.Marshal(policy.BlastRadius)
	value.VerificationContract, _ = json.Marshal(policy.Outcome)
	return value
}

func workerExecutionResponse() evidence.CompletedResponseSummary {
	return evidence.CompletedResponseSummary{ID: "response-a", TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "subject-a"}
}
