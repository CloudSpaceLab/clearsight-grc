package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logger.Info("worker scaffold ready", "capabilities", []string{"outbox", "timers", "routing-integrity", "signals", "drift", "readiness"})
	<-ctx.Done()
	logger.Info("worker stopped")
}
