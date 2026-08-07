package autonomy

import (
	"context"
	"testing"
)

func TestSourceRecoveryResolvesOnlyMatchingActiveDrift(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	_, inserted, err := service.Ingest(ctx, Signal{
		TenantID: "bank-demo", Type: SignalSourceDegraded, SubjectType: "EVIDENCE_SOURCE", SubjectID: "source-1",
		Source: "source-health", DedupeKey: "source-1:degraded",
	})
	if err != nil || !inserted {
		t.Fatalf("degrade source: %v inserted=%v", err, inserted)
	}
	_, _, err = service.Ingest(ctx, Signal{
		TenantID: "bank-demo", Type: SignalSourceDegraded, SubjectType: "EVIDENCE_SOURCE", SubjectID: "source-2",
		Source: "source-health", DedupeKey: "source-2:degraded",
	})
	if err != nil {
		t.Fatal(err)
	}

	inserted, err = service.ResolveSourceHealth(ctx, Signal{
		TenantID: "bank-demo", Type: SignalSourceRecovered, SubjectType: "EVIDENCE_SOURCE", SubjectID: "source-1",
		Source: "source-health", DedupeKey: "source-1:recovered",
	})
	if err != nil || !inserted {
		t.Fatalf("recover source: %v inserted=%v", err, inserted)
	}
	readiness, err := service.Readiness(ctx, "bank-demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(readiness.ActiveDrifts) != 1 || readiness.ActiveDrifts[0].SubjectID != "source-2" {
		t.Fatalf("recovery resolved the wrong drift set: %#v", readiness.ActiveDrifts)
	}

	duplicate, err := service.ResolveSourceHealth(ctx, Signal{
		TenantID: "bank-demo", Type: SignalSourceRecovered, SubjectType: "EVIDENCE_SOURCE", SubjectID: "source-1",
		Source: "source-health", DedupeKey: "source-1:recovered",
	})
	if err != nil || duplicate {
		t.Fatalf("recovery dedupe failed: %v inserted=%v", err, duplicate)
	}
}

func TestGenericSignalIngestCannotResolveSourceDrift(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, _, err := service.Ingest(context.Background(), Signal{
		TenantID: "bank-demo", Type: SignalSourceRecovered, SubjectType: "EVIDENCE_SOURCE", SubjectID: "source-1",
		Source: "api", DedupeKey: "forged-recovery",
	})
	if err == nil {
		t.Fatal("generic signal ingestion must not resolve source-health drift")
	}
}
