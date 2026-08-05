package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	worker, err := buildWorker(ctx, cfg, logger)
	if err != nil {
		logger.Error("worker initialization failed", "error", err)
		os.Exit(1)
	}
	defer worker.Close()
	logger.Info("governance worker started", "poll", cfg.WorkerPoll.String(), "worker_id", cfg.WorkerID)
	if err := worker.Runtime.Run(ctx, cfg.WorkerPoll); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("governance worker stopped unexpectedly", "error", err)
		os.Exit(1)
	}
	logger.Info("governance worker stopped")
}
