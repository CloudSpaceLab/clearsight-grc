package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const defaultAIGatewayOperationsTimeout = 2 * time.Second

type AIGatewayOperationsConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

func LoadAIGatewayOperations(environment string) (AIGatewayOperationsConfig, error) {
	baseURL := strings.TrimRight(env("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL", ""), "/")
	token := strings.TrimSpace(env("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN", ""))
	if (baseURL == "") != (token == "") {
		return AIGatewayOperationsConfig{}, fmt.Errorf("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL and CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN must be configured together")
	}
	if baseURL == "" {
		return AIGatewayOperationsConfig{}, nil
	}
	if len(token) < 32 || len(token) > 4096 || !safeGatewayOperationsToken(token) {
		return AIGatewayOperationsConfig{}, fmt.Errorf("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN must be 32-4096 printable characters")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return AIGatewayOperationsConfig{}, fmt.Errorf("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL must be a fixed origin")
	}
	local := localGatewayOperationsHost(parsed.Hostname())
	if parsed.Scheme != "https" && !(strings.EqualFold(environment, "development") && parsed.Scheme == "http" && local) {
		return AIGatewayOperationsConfig{}, fmt.Errorf("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL must use HTTPS outside local development")
	}
	timeout, err := duration("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TIMEOUT", defaultAIGatewayOperationsTimeout)
	if err != nil {
		return AIGatewayOperationsConfig{}, err
	}
	if timeout < 250*time.Millisecond || timeout > 10*time.Second {
		return AIGatewayOperationsConfig{}, fmt.Errorf("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TIMEOUT must be between 250ms and 10s")
	}
	return AIGatewayOperationsConfig{BaseURL: baseURL, Token: token, Timeout: timeout}, nil
}

func localGatewayOperationsHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func safeGatewayOperationsToken(value string) bool {
	for _, r := range value {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}
