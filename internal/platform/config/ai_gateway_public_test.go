package config

import "testing"

func TestLoadAIGatewayPublicIsOptional(t *testing.T) {
	t.Setenv("CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL", "")
	value, err := LoadAIGatewayPublic("production")
	if err != nil {
		t.Fatal(err)
	}
	if value.BaseURL != "" {
		t.Fatalf("base URL = %q", value.BaseURL)
	}
}

func TestLoadAIGatewayPublicNormalizesSecureBaseURL(t *testing.T) {
	t.Setenv("CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL", " https://ai.example.test/proxy/ ")
	value, err := LoadAIGatewayPublic("production")
	if err != nil {
		t.Fatal(err)
	}
	if value.BaseURL != "https://ai.example.test/proxy" {
		t.Fatalf("base URL = %q", value.BaseURL)
	}
}

func TestLoadAIGatewayPublicRejectsUnsafeProductionURLs(t *testing.T) {
	for name, value := range map[string]string{
		"http":        "http://ai.example.test",
		"credentials": "https://user:secret@ai.example.test",
		"query":       "https://ai.example.test?token=secret",
		"fragment":    "https://ai.example.test#gateway",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL", value)
			if _, err := LoadAIGatewayPublic("production"); err == nil {
				t.Fatalf("expected %q to be rejected", value)
			}
		})
	}
}

func TestLoadAIGatewayPublicAllowsLoopbackHTTPOnlyInDevelopment(t *testing.T) {
	t.Setenv("CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL", "http://localhost:8090")
	value, err := LoadAIGatewayPublic("development")
	if err != nil {
		t.Fatal(err)
	}
	if value.BaseURL != "http://localhost:8090" {
		t.Fatalf("base URL = %q", value.BaseURL)
	}
	if _, err := LoadAIGatewayPublic("production"); err == nil {
		t.Fatal("expected production loopback HTTP to be rejected")
	}
}
