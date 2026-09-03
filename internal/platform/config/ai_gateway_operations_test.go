package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAIGatewayOperationsIsOptional(t *testing.T) {
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL", "")
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN", "")
	cfg, err := LoadAIGatewayOperations("production")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "" || cfg.Token != "" || cfg.Timeout != 0 {
		t.Fatalf("unexpected optional config: %#v", cfg)
	}
}

func TestLoadAIGatewayOperationsRequiresPairedSecureConfiguration(t *testing.T) {
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL", "https://gateway.example")
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN", "")
	if _, err := LoadAIGatewayOperations("production"); err == nil || !strings.Contains(err.Error(), "configured together") {
		t.Fatalf("paired configuration error = %v", err)
	}

	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN", strings.Repeat("a", 32))
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL", "http://gateway.example")
	if _, err := LoadAIGatewayOperations("production"); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("production HTTPS error = %v", err)
	}
}

func TestLoadAIGatewayOperationsAllowsBoundedLocalDevelopmentBridge(t *testing.T) {
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL", "http://127.0.0.1:8090")
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN", strings.Repeat("a", 32))
	t.Setenv("CLEARSIGHT_AI_GATEWAY_OPERATIONS_TIMEOUT", "750ms")
	cfg, err := LoadAIGatewayOperations("development")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://127.0.0.1:8090" || cfg.Timeout != 750*time.Millisecond {
		t.Fatalf("config = %#v", cfg)
	}
}
