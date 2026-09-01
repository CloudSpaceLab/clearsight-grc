package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
)

type formPolicySubjectRegistryStub struct{ supported map[string]bool }

func (stub formPolicySubjectRegistryStub) SupportsSubjectType(subjectType string) bool {
	return stub.supported[subjectType]
}

type formPolicyAuthorityStub struct {
	performerKind string
	err           error
	inputs        []authority.ResolveInput
}

func (stub *formPolicyAuthorityStub) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	stub.inputs = append(stub.inputs, input)
	if stub.err != nil {
		return authority.Resolution{}, stub.err
	}
	principal := authority.Principal{ID: "owner", Kind: "PERSON"}
	if input.Responsibility == authority.ResponsibilityAuthorizer {
		principal.ID = "checker"
	}
	if input.Responsibility == authority.ResponsibilityPerformer {
		principal = authority.Principal{ID: "automation-service", Kind: stub.performerKind}
	}
	return authority.Resolution{Principal: principal, RuleID: "rule", PolicyVersion: "v1"}, nil
}
func (*formPolicyAuthorityStub) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (*formPolicyAuthorityStub) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (*formPolicyAuthorityStub) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestFormPolicyActivationAuthorityRechecksAutomationAndCurrentRoutes(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	automationPolicy := autonomy.AutomationPolicy{
		ID: "automation-a", TenantID: "bank", ActionClass: formpolicy.ActionClassCreateMatter,
		Status: autonomy.AutomationPolicyActive, Version: 2, RolloutMode: string(formpolicy.RolloutShadow), Checksum: "approved-v2",
	}
	automation := autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy))
	routes := &formPolicyAuthorityStub{performerKind: "SERVICE"}
	policy := formpolicy.Policy{
		ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2,
		Eligibility: formpolicy.Eligibility{SubjectTypes: []string{"VENDOR_RELATIONSHIP"}}, Action: formpolicy.MatterAction{Priority: 4},
		BlastRadius: formpolicy.BlastRadius{PerRun: 10, PerDay: 25}, Outcome: formpolicy.OutcomeContract{ExpectedOutcome: "Current evidence is accepted.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"}, Rollout: formpolicy.RolloutShadow,
	}
	automationPolicy.Eligibility, _ = json.Marshal(policy.Eligibility)
	automationPolicy.BlastRadiusLimit, _ = json.Marshal(policy.BlastRadius)
	automationPolicy.VerificationContract, _ = json.Marshal(policy.Outcome)
	automation = autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy))
	validator := formPolicyActivationAuthority{Automation: automation, Authority: routes, Subjects: formPolicySubjectRegistryStub{supported: map[string]bool{"VENDOR_RELATIONSHIP": true}}, Now: func() time.Time { return now }}
	actor := formpolicy.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	if err := validator.ValidatePolicyActivation(t.Context(), actor, policy); err != nil {
		t.Fatal(err)
	}
	if len(routes.inputs) != 4 {
		t.Fatalf("authority checks = %#v", routes.inputs)
	}
	if routes.inputs[0].ObjectType != "FORM_RESPONSE_POLICY" || routes.inputs[1].ObjectType != "MATTER" || routes.inputs[2].ObjectType != "VENDOR_RELATIONSHIP" || routes.inputs[3].Responsibility != authority.ResponsibilityPerformer {
		t.Fatalf("authority checks do not cover current authorizer, Matter, subject and service actor: %#v", routes.inputs)
	}
}

func TestFormPolicyActivationAuthorityFailsClosed(t *testing.T) {
	policy := formpolicy.Policy{ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2, Eligibility: formpolicy.Eligibility{SubjectTypes: []string{"VENDOR_RELATIONSHIP"}}, Action: formpolicy.MatterAction{Priority: 4}, BlastRadius: formpolicy.BlastRadius{PerRun: 10, PerDay: 25}, Outcome: formpolicy.OutcomeContract{ExpectedOutcome: "Current evidence is accepted.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"}, Rollout: formpolicy.RolloutShadow}
	base := autonomy.AutomationPolicy{ID: "automation-a", TenantID: "bank", ActionClass: formpolicy.ActionClassCreateMatter, Status: autonomy.AutomationPolicyActive, Version: 2, RolloutMode: string(policy.Rollout), Checksum: "approved-v2"}
	base.Eligibility, _ = json.Marshal(policy.Eligibility)
	base.BlastRadiusLimit, _ = json.Marshal(policy.BlastRadius)
	base.VerificationContract, _ = json.Marshal(policy.Outcome)
	actor := formpolicy.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	subjects := formPolicySubjectRegistryStub{supported: map[string]bool{"VENDOR_RELATIONSHIP": true}}
	for name, setup := range map[string]func() formPolicyActivationAuthority{
		"suspended automation policy": func() formPolicyActivationAuthority {
			value := base
			value.Status = autonomy.AutomationPolicySuspended
			return formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(value)), Authority: &formPolicyAuthorityStub{performerKind: "SERVICE"}, Subjects: subjects}
		},
		"human execution principal": func() formPolicyActivationAuthority {
			return formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(base)), Authority: &formPolicyAuthorityStub{performerKind: "PERSON"}, Subjects: subjects}
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := setup().ValidatePolicyActivation(t.Context(), actor, policy); !errors.Is(err, formpolicy.ErrActivationAuthority) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	unavailable := formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(base)), Authority: &formPolicyAuthorityStub{err: errors.New("database unavailable")}, Subjects: subjects}
	if err := unavailable.ValidatePolicyActivation(t.Context(), actor, policy); !errors.Is(err, formpolicy.ErrAuthorityUnavailable) {
		t.Fatalf("unavailable authority err = %v", err)
	}
}

func TestFormPolicyActivationAuthorityRejectsBroaderTypedGuardrails(t *testing.T) {
	policy := formpolicy.Policy{ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2, Eligibility: formpolicy.Eligibility{SubjectTypes: []string{"VENDOR_RELATIONSHIP"}}, BlastRadius: formpolicy.BlastRadius{PerRun: 10, PerDay: 25}, Outcome: formpolicy.OutcomeContract{ExpectedOutcome: "Current evidence is accepted.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"}, Rollout: formpolicy.RolloutEnforce}
	automationPolicy := autonomy.AutomationPolicy{ID: "automation-a", TenantID: "bank", ActionClass: formpolicy.ActionClassCreateMatter, Status: autonomy.AutomationPolicyActive, Version: 2, RolloutMode: string(formpolicy.RolloutShadow), Checksum: "approved-v2"}
	automationPolicy.Eligibility, _ = json.Marshal(policy.Eligibility)
	automationPolicy.BlastRadiusLimit, _ = json.Marshal(formpolicy.BlastRadius{PerRun: 5, PerDay: 10})
	automationPolicy.VerificationContract, _ = json.Marshal(policy.Outcome)
	validator := formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy)), Authority: &formPolicyAuthorityStub{performerKind: "SERVICE"}, Subjects: formPolicySubjectRegistryStub{supported: map[string]bool{"VENDOR_RELATIONSHIP": true}}}
	if err := validator.ValidatePolicyActivation(t.Context(), formpolicy.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}, policy); !errors.Is(err, formpolicy.ErrActivationAuthority) {
		t.Fatalf("broader typed guardrails err = %v", err)
	}
}

func TestFormPolicyActivationAuthorityRejectsUnsupportedSubjectType(t *testing.T) {
	policy := formpolicy.Policy{ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2, Eligibility: formpolicy.Eligibility{SubjectTypes: []string{"UNREGISTERED_OBJECT"}}, BlastRadius: formpolicy.BlastRadius{PerRun: 1, PerDay: 1}, Outcome: formpolicy.OutcomeContract{ExpectedOutcome: "Current evidence is accepted.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"}, Rollout: formpolicy.RolloutShadow}
	automationPolicy := autonomy.AutomationPolicy{ID: "automation-a", TenantID: "bank", ActionClass: formpolicy.ActionClassCreateMatter, Status: autonomy.AutomationPolicyActive, Version: 2, RolloutMode: string(policy.Rollout), Checksum: "approved-v2"}
	automationPolicy.Eligibility, _ = json.Marshal(policy.Eligibility)
	automationPolicy.BlastRadiusLimit, _ = json.Marshal(policy.BlastRadius)
	automationPolicy.VerificationContract, _ = json.Marshal(policy.Outcome)
	validator := formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy)), Authority: &formPolicyAuthorityStub{performerKind: "SERVICE"}, Subjects: formPolicySubjectRegistryStub{supported: map[string]bool{"VENDOR_RELATIONSHIP": true}}}
	if err := validator.ValidatePolicyActivation(t.Context(), formpolicy.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}, policy); !errors.Is(err, formpolicy.ErrActivationAuthority) {
		t.Fatalf("unsupported subject type err = %v", err)
	}
}
