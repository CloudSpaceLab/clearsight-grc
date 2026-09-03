package aigateway

import (
	"strings"
	"testing"
	"time"
)

func TestTransportDefinitionRejectsPlaintextLikeSecretReference(t *testing.T) {
	definition := TransportDefinition{
		Providers: []TransportProviderConfig{{
			ID: "provider-a", Name: "Provider A", Kind: ProviderKindOpenAI,
			BaseURL: "https://api.openai.com", SecretRef: "sk-this-must-never-be-stored", State: ProviderStateEnabled,
		}},
		Models: []ModelConfig{{Alias: "safe-chat", Routes: []RouteConfig{{ID: "route-a", ProviderID: "provider-a", Model: "gpt-5", Weight: 100}}}},
	}
	_, err := ValidateTransportDefinition("production", 2*time.Minute, definition)
	if err == nil || !strings.Contains(err.Error(), "env:NAME") {
		t.Fatalf("plaintext-like secret reference error = %v", err)
	}
}
