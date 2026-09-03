package aigateway

import "testing"

func TestOrganizationInstructionsDoNotMutateShadowTraffic(t *testing.T) {
	request := Request{Messages: []Message{{Role: RoleUser, Text: "hello"}}}
	mutated, err := ApplyDecision(request, Decision{
		Action:      DecisionAllow,
		RolloutMode: RolloutShadow,
		Obligations: []Obligation{{Code: ObligationOrganizationInstruction, Detail: "organization instruction"}},
	})
	if err != nil {
		t.Fatalf("ApplyDecision() error = %v", err)
	}
	if len(mutated.Messages) != 1 || mutated.Messages[0].Text != "hello" {
		t.Fatalf("shadow policy mutated request: %#v", mutated.Messages)
	}
}
