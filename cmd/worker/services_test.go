//go:build !postgres

package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestWorkerRegistersAssessmentSetupWorkClass(t *testing.T) {
	worker, err := buildWorker(context.Background(), config.Config{WorkerID: "worker-test", WorkerPoll: time.Second, VendorBrandDiscoveryEnabled: true}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	health, err := worker.Runtime.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range health {
		if item.Name == thirdparty.AssessmentSetupWorkClass {
			return
		}
	}
	t.Fatalf("assessment setup work class is not registered: %#v", health)
}

func TestWorkerRegistersBoundedVendorBrandWorkClass(t *testing.T) {
	worker, err := buildWorker(context.Background(), config.Config{WorkerID: "worker-test", WorkerPoll: time.Second, VendorBrandDiscoveryEnabled: true}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	if err := worker.Runtime.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	health, err := worker.Runtime.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range health {
		if item.Name == workflowruntime.WorkClassThirdPartyVendorBrand {
			if item.Options.Batch > 10 || item.Options.Timeout > 20*time.Second || item.Options.Lease <= item.Options.Timeout {
				t.Fatalf("vendor brand worker options are not bounded: %#v", item.Options)
			}
			if item.Queue == nil {
				t.Fatalf("vendor brand queue health is unavailable: %#v", item)
			}
			return
		}
	}
	t.Fatalf("vendor brand work class is not registered: %#v", health)
}

func TestWorkerOmitsOutboundVendorBrandClassWhenDisabled(t *testing.T) {
	worker, err := buildWorker(context.Background(), config.Config{WorkerID: "worker-test", WorkerPoll: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	health, err := worker.Runtime.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range health {
		if item.Name == workflowruntime.WorkClassThirdPartyVendorBrand {
			t.Fatalf("disabled outbound vendor brand class was registered: %#v", item)
		}
	}
}

func TestWorkerRegistersBoundedVendorRefreshMaintenance(t *testing.T) {
	cfg := config.Config{
		WorkerID: "worker-test", WorkerPoll: time.Second,
		VendorRefreshBatchSize: 25, VendorRefreshCadence: 10 * time.Minute, VendorRefreshLease: time.Minute,
		VendorRefreshDocumentLead: 30 * 24 * time.Hour, VendorRefreshFactConfirmationInterval: 365 * 24 * time.Hour,
	}
	worker, err := buildWorker(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	if err := worker.Runtime.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	health, err := worker.Runtime.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range health {
		if item.Name == thirdparty.VendorRefreshMaintenanceWorkClass {
			if item.Options.Batch != 25 || item.Options.Poll != 10*time.Minute || item.Options.Lease != time.Minute {
				t.Fatalf("vendor refresh worker options = %#v", item.Options)
			}
			return
		}
	}
	t.Fatalf("vendor refresh work class is not registered: %#v", health)
}
