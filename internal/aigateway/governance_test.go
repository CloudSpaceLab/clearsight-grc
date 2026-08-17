package aigateway

import (
	"errors"
	"testing"
	"time"
)

func TestEvaluatePolicyFailsClosedForRequiredSourceFactStates(t *testing.T) {
	policy := PolicySnapshot{
		ID: "policy-1", Code: "DATA_CLASSIFICATION", Version: 3, RolloutMode: RolloutEnforce,
		Definition: PolicyDefinition{
			Bindings:      []BindingRequirement{{FactKey: "classification", Required: true}},
			DefaultAction: DecisionAllow,
		},
	}
	request := Request{ID: "req-1", Protocol: ProtocolChat, ModelAlias: "general"}
	workload := Workload{ID: "workload-1", TenantID: "bank"}

	cases := []struct {
		name   string
		facts  []Fact
		reason string
	}{
		{"missing", nil, "SOURCE_FACT_UNKNOWN"},
		{"unknown", []Fact{{Key: "classification", State: FactUnknown}}, "SOURCE_FACT_UNKNOWN"},
		{"stale", []Fact{{Key: "classification", State: FactStale}}, "SOURCE_FACT_STALE"},
		{"unavailable", []Fact{{Key: "classification", State: FactUnavailable}}, "SOURCE_FACT_UNAVAILABLE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := EvaluatePolicy(policy, workload, request, tc.facts)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Action != DecisionDeny || len(decision.ReasonCodes) != 1 || decision.ReasonCodes[0] != tc.reason {
				t.Fatalf("unexpected fail-closed decision: %#v", decision)
			}
		})
	}
}

func TestEvaluatePolicyShadowPreservesProposedAction(t *testing.T) {
	policy := PolicySnapshot{
		ID: "policy-1", Code: "AI_CONTROL", Version: 7, RolloutMode: RolloutShadow,
		Definition: PolicyDefinition{Rules: []PolicyRule{{ID: "deny-prod", Priority: 10, FactKey: "purpose", Operator: "EQ", Value: "payments", Action: DecisionDeny, ReasonCode: "PAYMENTS_BLOCKED"}}, DefaultAction: DecisionAllow},
	}
	decision, err := EvaluatePolicy(policy, Workload{ID: "w1", Purpose: "payments"}, Request{ID: "r1", Protocol: ProtocolChat, ModelAlias: "general"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != DecisionShadow || decision.ProposedAction != DecisionDeny || decision.ReasonCodes[0] != "PAYMENTS_BLOCKED" {
		t.Fatalf("shadow decision lost proposed enforcement truth: %#v", decision)
	}
	if got := formatPolicyVersion(policy); got != "AI_CONTROL:7" {
		t.Fatalf("policy version = %q", got)
	}
}

func TestApplyDecisionModificationIsBoundedToConfiguredPromptRedactions(t *testing.T) {
	request := Request{ID: "r1", Protocol: ProtocolChat, ModelAlias: "general", Messages: []Message{{Role: RoleUser, Text: "account 1234 secret"}}}
	decision := Decision{Action: DecisionModify, Redactions: []Redaction{{Target: "prompt", Pattern: `\b\d{4}\b`}, {Target: "response", Pattern: "secret"}}}
	modified, err := ApplyDecision(request, decision)
	if err != nil {
		t.Fatal(err)
	}
	if got := modified.Messages[0].Text; got != "account [REDACTED] secret" {
		t.Fatalf("unexpected modified prompt: %q", got)
	}
}

func TestInspectResponseRedactsDeniesAndBlocksWholeResponseControlsForStreaming(t *testing.T) {
	control := ResponseControl{MaxBytes: 256, RedactPatterns: []string{`secret-[0-9]+`}, DenyPatterns: []string{`forbidden`}}
	response, err := InspectResponse(control, Response{Text: "token secret-123"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != "token [REDACTED]" {
		t.Fatalf("unexpected redaction: %q", response.Text)
	}
	if _, err := InspectResponse(control, Response{Text: "forbidden"}, false); !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("deny pattern must fail closed, got %v", err)
	}
	if _, err := InspectResponse(control, Response{}, true); err == nil {
		t.Fatal("streaming must be rejected when whole-response controls are configured")
	}
	if _, err := InspectResponse(ResponseControl{}, Response{}, true); err != nil {
		t.Fatalf("uncontrolled streaming should preserve T3 behavior: %v", err)
	}
}

func TestKnownServerFactOutranksWeakerFactState(t *testing.T) {
	policy := PolicySnapshot{
		ID: "p1", Code: "CLASSIFICATION", Version: 1, RolloutMode: RolloutEnforce,
		Definition: PolicyDefinition{
			Bindings:      []BindingRequirement{{FactKey: "classification", Required: true}},
			Rules:         []PolicyRule{{ID: "deny-secret", Priority: 1, FactKey: "classification", Operator: "EQ", Value: "SECRET", Action: DecisionDeny, ReasonCode: "SECRET_DATA"}},
			DefaultAction: DecisionAllow,
		},
	}
	facts := []Fact{
		{Key: "classification", State: FactUnavailable},
		{Key: "classification", State: FactKnown, Value: "SECRET", Source: "CONNECTED_SOURCE", ObservedAt: time.Now().UTC()},
	}
	decision, err := EvaluatePolicy(policy, Workload{}, Request{}, facts)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != DecisionDeny || decision.ReasonCodes[0] != "SECRET_DATA" {
		t.Fatalf("stronger known fact was not authoritative: %#v", decision)
	}
}
