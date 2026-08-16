package aigateway

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestParseConfigResolvesSecretsAndBounds(t *testing.T) {
	key := sha256.Sum256([]byte("workload-secret-value"))
	metrics := sha256.Sum256([]byte("metrics-secret-value"))
	payload := `{
		"environment":"test",
		"metrics_bearer_sha256":"` + hex.EncodeToString(metrics[:]) + `",
		"workloads":[{"id":"workload-a","tenant_id":"tenant-a","key_sha256":"` + hex.EncodeToString(key[:]) + `","allowed_models":["governed-chat"],"requests_per_minute":60,"tokens_per_minute":100000,"cost_microusd_per_minute":1000000,"max_concurrent":4}],
		"providers":[
			{"id":"openai-a","kind":"OPENAI","base_url":"http://127.0.0.1:10001","secret_env":"OPENAI_TEST_KEY"},
			{"id":"anthropic-a","kind":"ANTHROPIC","base_url":"http://localhost:10002","secret_env":"ANTHROPIC_TEST_KEY"}
		],
		"models":[{"alias":"governed-chat","routes":[
			{"id":"route-openai","provider_id":"openai-a","model":"model-a","weight":2,"input_microusd_per_million_tokens":1000,"output_microusd_per_million_tokens":2000},
			{"id":"route-anthropic","provider_id":"anthropic-a","model":"model-b","weight":1,"input_microusd_per_million_tokens":1200,"output_microusd_per_million_tokens":2200}
		]}]
	}`
	secrets := map[string]string{"OPENAI_TEST_KEY": "openai-secret", "ANTHROPIC_TEST_KEY": "anthropic-secret"}
	config, err := ParseConfig([]byte(payload), func(name string) (string, bool) { value, ok := secrets[name]; return value, ok })
	if err != nil {
		t.Fatal(err)
	}
	if config.ListenAddr != defaultGatewayAddr || len(config.Workloads) != 1 || len(config.Providers) != 2 || config.MetricsDigest == nil {
		t.Fatalf("unexpected config: %#v", config)
	}
	if config.Providers[0].Secret == "" || config.Providers[1].Secret == "" {
		t.Fatal("provider secret references were not resolved")
	}
}

func TestParseConfigRejectsProductionHTTPAndDuplicateCredentials(t *testing.T) {
	key := sha256.Sum256([]byte("shared-secret-value"))
	payload := `{
		"environment":"production",
		"workloads":[
			{"id":"a","tenant_id":"tenant-a","key_sha256":"` + hex.EncodeToString(key[:]) + `","allowed_models":["m"],"requests_per_minute":1,"tokens_per_minute":10,"cost_microusd_per_minute":10,"max_concurrent":1},
			{"id":"b","tenant_id":"tenant-b","key_sha256":"` + hex.EncodeToString(key[:]) + `","allowed_models":["m"],"requests_per_minute":1,"tokens_per_minute":10,"cost_microusd_per_minute":10,"max_concurrent":1}
		],
		"providers":[{"id":"p1","kind":"OPENAI","base_url":"http://127.0.0.1:1","secret_env":"A"},{"id":"p2","kind":"ANTHROPIC","base_url":"https://example.invalid","secret_env":"B"}],
		"models":[{"alias":"m","routes":[{"id":"r1","provider_id":"p1","model":"x","weight":1},{"id":"r2","provider_id":"p2","model":"y","weight":1}]}]
	}`
	_, err := ParseConfig([]byte(payload), func(string) (string, bool) { return "provider-secret", true })
	if err == nil || (!strings.Contains(err.Error(), "share one credential") && !strings.Contains(err.Error(), "HTTPS")) {
		t.Fatalf("expected fail-closed config error, got %v", err)
	}
}

func TestParseConfigRejectsTrailingJSONAndOverflowedTimeout(t *testing.T) {
	key := sha256.Sum256([]byte("workload-secret-value"))
	base := `{"environment":"test","request_timeout_ms":9223372036854775807,"workloads":[{"id":"w","tenant_id":"t","key_sha256":"` + hex.EncodeToString(key[:]) + `","allowed_models":["m"],"requests_per_minute":1,"tokens_per_minute":10,"cost_microusd_per_minute":10,"max_concurrent":1}],"providers":[{"id":"p","kind":"OPENAI","base_url":"http://127.0.0.1:1","secret_env":"P"}],"models":[{"alias":"m","routes":[{"id":"r","provider_id":"p","model":"x","weight":1}]}]}`
	if _, err := ParseConfig([]byte(base), func(string) (string, bool) { return "provider-secret", true }); err == nil || !strings.Contains(err.Error(), "timeouts") {
		t.Fatalf("overflowed timeout error = %v", err)
	}
	valid := strings.Replace(base, `"request_timeout_ms":9223372036854775807,`, "", 1)
	if _, err := ParseConfig([]byte(valid+` {}`), func(string) (string, bool) { return "provider-secret", true }); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("trailing JSON error = %v", err)
	}
}

func TestParseConfigRejectsOversizedAndUnsafeProviderConfiguration(t *testing.T) {
	if _, err := ParseConfig(make([]byte, maxConfigBytes+1), func(string) (string, bool) { return "provider-secret", true }); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("oversized config error = %v", err)
	}

	key := sha256.Sum256([]byte("workload-secret-value"))
	payload := `{"environment":"test","workloads":[{"id":"w","tenant_id":"t","key_sha256":"` + hex.EncodeToString(key[:]) + `","allowed_models":["m"],"requests_per_minute":1,"tokens_per_minute":10,"cost_microusd_per_minute":10,"max_concurrent":1}],"providers":[{"id":"p","kind":"ANTHROPIC","base_url":"http://127.0.0.1:1","secret_env":"P","api_version":"2023-06-01\ninvalid"}],"models":[{"alias":"m","routes":[{"id":"r","provider_id":"p","model":"x","weight":1}]}]}`
	if _, err := ParseConfig([]byte(payload), func(string) (string, bool) { return "provider-secret", true }); err == nil || !strings.Contains(err.Error(), "API version") {
		t.Fatalf("unsafe API version error = %v", err)
	}

	payload = strings.Replace(payload, `,"api_version":"2023-06-01\ninvalid"`, "", 1)
	if _, err := ParseConfig([]byte(payload), func(string) (string, bool) { return "provider\nsecret", true }); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("unsafe provider credential error = %v", err)
	}
}
