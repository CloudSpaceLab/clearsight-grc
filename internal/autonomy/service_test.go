package autonomy

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRoutingGapChangesReadinessWithoutFabricatedBaseline(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, inserted, err := service.Ingest(context.Background(), Signal{TenantID: "bank-demo", Type: SignalRoutingGap, SubjectType: "CONTROL", SubjectID: "c1", Source: "integrity", DedupeKey: "routing-gap-c1"})
	if err != nil || !inserted {
		t.Fatalf("ingest: %v inserted=%v", err, inserted)
	}
	readiness, err := service.Readiness(context.Background(), "bank-demo")
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Dimensions.BlockedRouting != 1 || readiness.Status != "AT_RISK" {
		t.Fatalf("unexpected readiness: %#v", readiness)
	}
	if readiness.BaselineKnown || readiness.Dimensions.Current != 0 {
		t.Fatalf("readiness must not fabricate a current baseline: %#v", readiness)
	}
}

func TestSignalDedupe(t *testing.T) {
	service := NewService(NewMemoryRepository())
	signal := Signal{TenantID: "bank-demo", Type: SignalEvidenceExpired, SubjectType: "CLAIM", SubjectID: "x", Source: "scheduler", DedupeKey: "same"}
	_, first, err := service.Ingest(context.Background(), signal)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := service.Ingest(context.Background(), signal)
	if err != nil {
		t.Fatal(err)
	}
	if !first || second {
		t.Fatalf("expected first insert only")
	}
}

func TestSignalRequiresSourceDedupeKey(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, _, err := service.Ingest(context.Background(), Signal{TenantID: "bank-demo", Type: SignalEvidenceExpired, SubjectType: "CLAIM", SubjectID: "x", Source: "scheduler"})
	if err == nil {
		t.Fatal("expected missing dedupe key to fail")
	}
}

func TestUnknownReadinessWhenNoBaselineOrDrift(t *testing.T) {
	service := NewService(NewMemoryRepository())
	readiness, err := service.Readiness(context.Background(), "bank-demo")
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Status != "UNKNOWN" || readiness.BaselineKnown || readiness.Dimensions.Current != 0 {
		t.Fatalf("unexpected readiness: %#v", readiness)
	}
}

func TestAutomationPoliciesAreTenantScopedLatestAndPreserveGuardrails(t *testing.T) {
	policy := AutomationPolicy{
		ID: "policy-2", TenantID: "bank-demo", Code: "EVIDENCE-REFRESH", Name: "Low-impact evidence refresh",
		ActionClass: "REVERSIBLE_WRITE", Status: "ACTIVE", Version: 2,
		Eligibility: json.RawMessage(`{"materiality_max":2}`),
		BlastRadiusLimit: json.RawMessage(`{"max_records":25}`),
		VerificationContract: json.RawMessage(`{"method":"source_recheck"}`),
	}
	service := NewService(NewMemoryRepository(
		AutomationPolicy{ID: "policy-1", TenantID: "bank-demo", Code: "EVIDENCE-REFRESH", Name: "Low-impact evidence refresh", Version: 1},
		policy,
		AutomationPolicy{ID: "other", TenantID: "other-bank", Code: "OTHER", Name: "Other", Version: 9},
	))

	values, err := service.ListAutomationPolicies(context.Background(), "bank-demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != policy.ID || values[0].Version != 2 {
		t.Fatalf("unexpected policies: %#v", values)
	}
	if string(values[0].Eligibility) != string(policy.Eligibility) || string(values[0].VerificationContract) != string(policy.VerificationContract) {
		t.Fatalf("guardrails were not preserved: %#v", values[0])
	}
	if _, err := service.ListAutomationPolicies(context.Background(), ""); err == nil {
		t.Fatal("expected tenant requirement")
	}
}
