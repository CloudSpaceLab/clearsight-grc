package autonomy

import (
	"context"
	"testing"
)

func TestRoutingGapChangesReadiness(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, inserted, err := service.Ingest(context.Background(), Signal{TenantID: "bank-demo", Type: SignalRoutingGap, SubjectType: "CONTROL", SubjectID: "c1", Source: "integrity"})
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
}
func TestSignalDedupe(t *testing.T) {
	service := NewService(NewMemoryRepository())
	signal := Signal{TenantID: "bank-demo", Type: SignalEvidenceExpired, SubjectType: "CLAIM", SubjectID: "x", Source: "scheduler", DedupeKey: "same"}
	_, first, _ := service.Ingest(context.Background(), signal)
	_, second, _ := service.Ingest(context.Background(), signal)
	if !first || second {
		t.Fatalf("expected first insert only")
	}
}
