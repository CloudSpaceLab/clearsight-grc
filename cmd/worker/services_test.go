//go:build !postgres

package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestWorkerRegistersAssessmentSetupWorkClass(t *testing.T) {
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
		if item.Name == thirdparty.AssessmentSetupWorkClass {
			return
		}
	}
	t.Fatalf("assessment setup work class is not registered: %#v", health)
}
