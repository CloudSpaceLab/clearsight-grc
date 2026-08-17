//go:build postgres

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5/pgxpool"
)

func buildGateway(ctx context.Context, config aigateway.RuntimeConfig, logger *slog.Logger) (*aigateway.Gateway, func(), error) {
	if config.GovernanceMode != aigateway.GovernanceDatabase {
		gateway, err := aigateway.NewGateway(config, logger)
		return gateway, func() {}, err
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return nil, func() {}, fmt.Errorf("DATABASE_URL is required for DATABASE gateway governance")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open gateway governance database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, func() {}, fmt.Errorf("ping gateway governance database: %w", err)
	}
	catalog := sourceaccess.NewCatalogService(sourceaccess.NewPostgresCatalogRepository(pool), sourceaccess.EnvironmentSecretResolver{}, sourceaccess.DefaultCatalogAdapters())
	runtime := aigovernance.NewRuntimeProvider(aigovernance.NewPostgresRepository(pool), catalog)
	gateway, err := aigateway.NewGatewayWithGovernance(config, runtime, runtime, logger)
	if err != nil {
		pool.Close()
		return nil, func() {}, err
	}
	return gateway, pool.Close, nil
}
