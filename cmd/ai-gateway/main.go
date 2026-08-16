package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func main() {
	level := slog.LevelInfo
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CLEARSIGHT_LOG_LEVEL")), "debug") {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	config, err := aigateway.LoadConfigFromEnvironment()
	if err != nil {
		logger.Error("ai gateway configuration failed", "error_code", "configuration_invalid", "error", err)
		os.Exit(1)
	}
	gateway, err := aigateway.NewGateway(config, logger)
	if err != nil {
		logger.Error("ai gateway initialization failed", "error_code", "initialization_failed", "error", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              config.ListenAddr,
		Handler:           aigateway.NewHTTPHandler(gateway, logger),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0, // provider streams are bounded by request contexts, not one HTTP write deadline
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("ai gateway listening", "address", config.ListenAddr, "environment", config.Environment)
		serverErrors <- server.ListenAndServe()
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case signal := <-stop:
		logger.Info("ai gateway shutdown requested", "signal", signal.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("ai gateway stopped unexpectedly", "error_code", "server_stopped", "error", err)
			os.Exit(1)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("ai gateway graceful shutdown failed", "error_code", "shutdown_failed", "error", err)
		_ = server.Close()
	}
}
