package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
)

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
		Status: autonomy.AutomationPolicyActive, Version: 2,
	}
	automation := autonomy.NewService(autonomy.NewMemoryRepository(automationPolicy))
	routes := &formPolicyAuthorityStub{performerKind: "SERVICE"}
	validator := formPolicyActivationAuthority{Automation: automation, Authority: routes, Now: func() time.Time { return now }}
	policy := formpolicy.Policy{
		ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2,
		Eligibility: formpolicy.Eligibility{SubjectTypes: []string{"VENDOR"}}, Action: formpolicy.MatterAction{Priority: 4},
	}
	actor := formpolicy.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	if err := validator.ValidatePolicyActivation(t.Context(), actor, policy); err != nil {
		t.Fatal(err)
	}
	if len(routes.inputs) != 4 {
		t.Fatalf("authority checks = %#v", routes.inputs)
	}
	if routes.inputs[0].ObjectType != "FORM_RESPONSE_POLICY" || routes.inputs[1].ObjectType != "MATTER" || routes.inputs[2].ObjectType != "VENDOR" || routes.inputs[3].Responsibility != authority.ResponsibilityPerformer {
		t.Fatalf("authority checks do not cover current authorizer, Matter, subject and service actor: %#v", routes.inputs)
	}
}

func TestFormPolicyActivationAuthorityFailsClosed(t *testing.T) {
	base := autonomy.AutomationPolicy{ID: "automation-a", TenantID: "bank", ActionClass: formpolicy.ActionClassCreateMatter, Status: autonomy.AutomationPolicyActive, Version: 2}
	policy := formpolicy.Policy{ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2, Eligibility: formpolicy.Eligibility{SubjectTypes: []string{"VENDOR"}}, Action: formpolicy.MatterAction{Priority: 4}}
	actor := formpolicy.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	for name, setup := range map[string]func() formPolicyActivationAuthority{
		"suspended automation policy": func() formPolicyActivationAuthority {
			value := base
			value.Status = autonomy.AutomationPolicySuspended
			return formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(value)), Authority: &formPolicyAuthorityStub{performerKind: "SERVICE"}}
		},
		"human execution principal": func() formPolicyActivationAuthority {
			return formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(base)), Authority: &formPolicyAuthorityStub{performerKind: "PERSON"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := setup().ValidatePolicyActivation(t.Context(), actor, policy); !errors.Is(err, formpolicy.ErrActivationAuthority) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	unavailable := formPolicyActivationAuthority{Automation: autonomy.NewService(autonomy.NewMemoryRepository(base)), Authority: &formPolicyAuthorityStub{err: errors.New("database unavailable")}}
	if err := unavailable.ValidatePolicyActivation(t.Context(), actor, policy); !errors.Is(err, formpolicy.ErrAuthorityUnavailable) {
		t.Fatalf("unavailable authority err = %v", err)
	}
}
