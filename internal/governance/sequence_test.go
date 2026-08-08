package governance

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestResolveLifecycleSequenceSelectsResponsibilityNotOutcome(t *testing.T) {
	at := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	policies := []RoutingPolicy{{
		Code: "DECISION-SEQ", Status: PolicyActive, CurrentVersion: 3,
		EffectiveFrom: ptrSequenceTime(at.Add(-time.Hour)),
		Definition: []byte(`{"rules":[
			{"id":"generic-review","legal_entity_id":"*","object_type":"MATTER","object_id":"*","responsibility":"REVIEWER","decision_type":"matter.decision.record","min_materiality":0,"priority":10,"lifecycle_type":"DECISION","lifecycle_state":"PROPOSED"},
			{"id":"entity-review","legal_entity_id":"BANK-NG","object_type":"MATTER","object_id":"*","responsibility":"REVIEWER","decision_type":"matter.decision.record","min_materiality":2,"priority":10,"lifecycle_type":"DECISION","lifecycle_subtype":"PRIVACY_POSITION","lifecycle_state":"PROPOSED"}
		]}`),
	}}

	resolution, err := resolveLifecycleSequence(policies, LifecycleSequenceInput{
		TenantID: "bank", LegalEntityCode: "BANK-NG", MatterID: "matter-1", MatterType: "REGULATORY_CHANGE",
		CommandName: "matter.decision.record", LifecycleType: "DECISION", LifecycleSubtype: "PRIVACY_POSITION", LifecycleState: "PROPOSED", Materiality: 4, At: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Responsibility != "REVIEWER" || resolution.RuleID != "DECISION-SEQ/entity-review" || resolution.PolicyVersion != "DECISION-SEQ:v3" {
		t.Fatalf("unexpected sequence resolution: %#v", resolution)
	}
}

func TestResolveLifecycleSequenceMatchesLegalEntityIDOrCode(t *testing.T) {
	at := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	const entityID = "98888888-8888-7888-8888-888888888882"
	policies := []RoutingPolicy{{
		Code: "ENTITY-ID", Status: PolicyActive, CurrentVersion: 1,
		Definition: []byte(`{"rules":[{"id":"review","legal_entity_id":"` + entityID + `","object_type":"MATTER","object_id":"*","responsibility":"REVIEWER","decision_type":"matter.decision.record","priority":10,"lifecycle_type":"DECISION","lifecycle_state":"PROPOSED"}]}`),
	}}
	resolution, err := resolveLifecycleSequence(policies, LifecycleSequenceInput{
		TenantID: "bank", LegalEntityID: entityID, LegalEntityCode: "BANK-NG", MatterID: "matter-1",
		CommandName: "matter.decision.record", LifecycleType: "DECISION", LifecycleState: "PROPOSED", Materiality: 3, At: at,
	})
	if err != nil || resolution.Responsibility != "REVIEWER" {
		t.Fatalf("legal entity UUID alias did not match: %#v err=%v", resolution, err)
	}
}

func TestResolveLifecycleSequenceFailsClosedOnEqualRankDifferentResponsibilities(t *testing.T) {
	at := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	policies := []RoutingPolicy{{
		Code: "AMBIG", Status: PolicyActive, CurrentVersion: 1,
		Definition: []byte(`{"rules":[
			{"id":"review","legal_entity_id":"*","object_type":"MATTER","object_id":"*","responsibility":"REVIEWER","priority":50,"lifecycle_type":"DECISION","lifecycle_state":"PROPOSED"},
			{"id":"challenge","legal_entity_id":"*","object_type":"MATTER","object_id":"*","responsibility":"INDEPENDENT_CHALLENGER","priority":50,"lifecycle_type":"DECISION","lifecycle_state":"PROPOSED"}
		]}`),
	}}
	_, err := resolveLifecycleSequence(policies, LifecycleSequenceInput{TenantID: "bank", MatterID: "matter-1", LifecycleType: "DECISION", LifecycleState: "PROPOSED", Materiality: 3, At: at})
	if !errors.Is(err, ErrAmbiguousLifecycleSequence) {
		t.Fatalf("expected ambiguous sequence, got %v", err)
	}
}

func TestResolveLifecycleSequenceIgnoresAuthorityOnlyRules(t *testing.T) {
	policies := []RoutingPolicy{{Code: "AUTH", Status: PolicyActive, CurrentVersion: 1, Definition: []byte(`{"rules":[{"id":"authorizer","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"CRO"}}]}`)}}
	_, err := resolveLifecycleSequence(policies, LifecycleSequenceInput{TenantID: "bank", MatterID: "matter-1", LifecycleType: "DECISION", LifecycleState: "PROPOSED", Materiality: 3, At: time.Now()})
	if !errors.Is(err, ErrNoLifecycleSequence) {
		t.Fatalf("authority-only routing rule became a sequence rule: %v", err)
	}
}

func TestLifecycleSequenceDeclarationIsRejectedDuringPolicyValidation(t *testing.T) {
	definition := []byte(`{"rules":[{"id":"bad","responsibility":"REVIEWER","lifecycle_type":"DECISION"}]}`)
	if err := validatePolicyDefinition(definition); err == nil {
		t.Fatal("partial lifecycle declaration passed routing-policy validation")
	}
}

func TestLifecycleSequenceRuleRejectsActorSelector(t *testing.T) {
	definition := []byte(`{"rules":[{"id":"bad","responsibility":"REVIEWER","selector":{"kind":"ROLE","ref":"REVIEWER"},"lifecycle_type":"DECISION","lifecycle_state":"PROPOSED"}]}`)
	if err := validatePolicyDefinition(definition); err == nil {
		t.Fatal("lifecycle sequence rule with actor selector passed policy validation")
	}
}

func TestAuthorityRuleStillRequiresSelector(t *testing.T) {
	definition := []byte(`{"rules":[{"id":"bad-authority","responsibility":"AUTHORIZER"}]}`)
	if err := validatePolicyDefinition(definition); err == nil {
		t.Fatal("ordinary authority rule without selector passed policy validation")
	}
}

func TestAuthorityOnlyPolicyDefinitionExcludesLifecycleSequenceRules(t *testing.T) {
	definition := []byte(`{"rules":[
		{"id":"sequence","responsibility":"REVIEWER","lifecycle_type":"DECISION","lifecycle_state":"PROPOSED"},
		{"id":"authority","responsibility":"REVIEWER","selector":{"kind":"ROLE","ref":"REVIEWER"}}
	]}`)
	filtered, err := authorityOnlyPolicyDefinition(definition)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Rules []struct {
			ID string `json:"id"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(filtered, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Rules) != 1 || result.Rules[0].ID != "authority" {
		t.Fatalf("selector conflict check still sees sequence rules: %s", filtered)
	}
}

func ptrSequenceTime(value time.Time) *time.Time { return &value }
