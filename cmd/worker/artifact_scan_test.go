package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestArtifactInspectionWorkerIsBoundedAndObservable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runtime := workflowruntime.NewService(workflowruntime.NewMemoryRepository(), nil, workflowruntime.LogPublisher{Logger: logger}, "test")
	repo := evidence.NewMemoryRepository(nil, nil)
	cfg := config.Config{WorkerID: "test", MaxArtifactBytes: 20 << 20, ArtifactScanner: "unavailable"}
	if err := configureArtifactScanWorker(runtime, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), repo, evidence.NewMemoryObjectStore()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	health, err := runtime.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range health {
		if class.Name == evidence.ArtifactScanWorkClass {
			if class.Options.Batch != 1 || class.Options.Timeout > 30*time.Second || class.Options.Lease <= class.Options.Timeout || class.Queue == nil {
				t.Fatalf("class=%+v", class)
			}
			return
		}
	}
	t.Fatal("inspection work class absent")
}
