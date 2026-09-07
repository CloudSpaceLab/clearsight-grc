package main

import "github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"

func buildAIGatewayPublic(cfg config.Config) (string, error) {
	public, err := config.LoadAIGatewayPublic(cfg.Environment)
	if err != nil {
		return "", err
	}
	return public.BaseURL, nil
}
