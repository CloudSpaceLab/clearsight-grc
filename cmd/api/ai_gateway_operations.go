package main

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
)

func buildAIGatewayOperations(cfg config.Config) (*aigateway.OperationsClient, error) {
	operations, err := config.LoadAIGatewayOperations(cfg.Environment)
	if err != nil {
		return nil, err
	}
	if operations.BaseURL == "" {
		return nil, nil
	}
	return aigateway.NewOperationsClient(operations.BaseURL, operations.Token, operations.Timeout)
}
