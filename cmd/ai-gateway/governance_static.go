//go:build !postgres

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func buildGateway(_ context.Context, config aigateway.RuntimeConfig, logger *slog.Logger) (*aigateway.Gateway, func(), error) {
	if config.GovernanceMode == aigateway.GovernanceDatabase {
		return nil, func() {}, fmt.Errorf("DATABASE governance requires the postgres build")
	}
	gateway, err := aigateway.NewGateway(config, logger)
	return gateway, func() {}, err
}
