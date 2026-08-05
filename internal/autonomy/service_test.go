package autonomy

import (
	"context"
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
