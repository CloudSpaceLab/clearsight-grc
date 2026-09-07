package config

import (
	"fmt"
	"net/url"
	"strings"
)

// AIGatewayPublicConfig contains only deployment metadata that is safe to
// project to authenticated administrators. Operational credentials and the
// internal operations URL remain in AIGatewayOperationsConfig.
type AIGatewayPublicConfig struct {
	BaseURL string
}

func LoadAIGatewayPublic(environment string) (AIGatewayPublicConfig, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(env("CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL", "")), "/")
	if baseURL == "" {
		return AIGatewayPublicConfig{}, nil
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return AIGatewayPublicConfig{}, fmt.Errorf("CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL must be an absolute URL without credentials, query or fragment")
	}
	local := strings.EqualFold(parsed.Hostname(), "localhost") || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme != "https" && !(strings.EqualFold(environment, "development") && parsed.Scheme == "http" && local) {
		return AIGatewayPublicConfig{}, fmt.Errorf("CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL must use HTTPS outside local development")
	}
	return AIGatewayPublicConfig{BaseURL: baseURL}, nil
}
