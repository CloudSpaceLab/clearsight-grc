package aigateway

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	defaultGatewayAddr          = ":8090"
	defaultRequestTimeout       = 2 * time.Minute
	defaultProviderTimeout      = 90 * time.Second
	defaultShutdownTimeout      = 10 * time.Second
	defaultMaxRequestBytes      = 2 << 20
	defaultMaxProviderBodyBytes = 8 << 20
	defaultMaxSSEEventBytes     = 256 << 10
	defaultGovernanceRefresh    = 5 * time.Second
	maxConfigBytes              = 8 << 20

	GovernanceStatic   = "STATIC"
	GovernanceDatabase = "DATABASE"
)

// FileConfig owns provider transport configuration. Workloads may be supplied
// statically only in development/test; production resolves governed workload
// and policy revisions from the ClearSight database.
type FileConfig struct {
	Environment          string               `json:"environment"`
	ListenAddr           string               `json:"listen_addr"`
	RequestTimeoutMS     int64                `json:"request_timeout_ms"`
	ShutdownTimeoutMS    int64                `json:"shutdown_timeout_ms"`
	MaxRequestBytes      int64                `json:"max_request_bytes"`
	MaxProviderBodyBytes int64                `json:"max_provider_body_bytes"`
	MaxSSEEventBytes     int64                `json:"max_sse_event_bytes"`
	MetricsBearerSHA256  string               `json:"metrics_bearer_sha256"`
	CircuitBreaker       CircuitBreakerConfig `json:"circuit_breaker"`
	GovernanceMode       string               `json:"governance_mode,omitempty"`
	GovernanceRefreshMS  int64                `json:"governance_refresh_ms,omitempty"`
	Workloads            []WorkloadConfig     `json:"workloads,omitempty"`
	Providers            []ProviderConfig     `json:"providers"`
	Models               []ModelConfig        `json:"models"`
}

type CircuitBreakerConfig struct {
	FailureThreshold int64 `json:"failure_threshold"`
	OpenDurationMS   int64 `json:"open_duration_ms"`
}

type WorkloadConfig struct {
	ID                    string   `json:"id"`
	TenantID              string   `json:"tenant_id"`
	KeySHA256             string   `json:"key_sha256"`
	AllowedModels         []string `json:"allowed_models"`
	RequestsPerMinute     int64    `json:"requests_per_minute"`
	TokensPerMinute       int64    `json:"tokens_per_minute"`
	CostMicroUSDPerMinute int64    `json:"cost_microusd_per_minute"`
	MaxConcurrent         int64    `json:"max_concurrent"`
}

type ProviderConfig struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	BaseURL      string `json:"base_url"`
	SecretEnv    string `json:"secret_env"`
	APIVersion   string `json:"api_version"`
	TimeoutMS    int64  `json:"timeout_ms"`
	RequireUsage *bool  `json:"require_usage,omitempty"`
}

type ModelConfig struct {
	Alias  string        `json:"alias"`
	Routes []RouteConfig `json:"routes"`
}

type RouteConfig struct {
	ID                             string `json:"id"`
	ProviderID                     string `json:"provider_id"`
	Model                          string `json:"model"`
	Weight                         int64  `json:"weight"`
	InputMicroUSDPerMillionTokens  int64  `json:"input_microusd_per_million_tokens"`
	OutputMicroUSDPerMillionTokens int64  `json:"output_microusd_per_million_tokens"`
}

// RuntimeConfig is the validated, secret-resolved configuration used in memory.
type RuntimeConfig struct {
	Environment          string
	ListenAddr           string
	RequestTimeout       time.Duration
	ShutdownTimeout      time.Duration
	MaxRequestBytes      int64
	MaxProviderBodyBytes int64
	MaxSSEEventBytes     int64
	MetricsDigest        *[sha256.Size]byte
	CircuitBreaker       CircuitBreakerConfig
	GovernanceMode       string
	GovernanceRefresh    time.Duration
	Workloads            []ConfiguredWorkload
	Providers            []ResolvedProviderConfig
	Models               []ModelConfig
}

type ConfiguredWorkload struct {
	Workload
	KeyDigest [sha256.Size]byte
}

type ResolvedProviderConfig struct {
	ProviderConfig
	Secret       string
	Timeout      time.Duration
	RequireUsage bool
}

func LoadConfigFromEnvironment() (RuntimeConfig, error) {
	path := strings.TrimSpace(os.Getenv("CLEARSIGHT_AI_GATEWAY_CONFIG_FILE"))
	inline := strings.TrimSpace(os.Getenv("CLEARSIGHT_AI_GATEWAY_CONFIG_JSON"))
	if (path == "") == (inline == "") {
		return RuntimeConfig{}, fmt.Errorf("set exactly one of CLEARSIGHT_AI_GATEWAY_CONFIG_FILE or CLEARSIGHT_AI_GATEWAY_CONFIG_JSON")
	}
	var payload []byte
	var err error
	if path != "" {
		file, openErr := os.Open(path)
		if openErr != nil {
			return RuntimeConfig{}, fmt.Errorf("read gateway config: %w", openErr)
		}
		payload, err = io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
		closeErr := file.Close()
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("read gateway config: %w", err)
		}
		if closeErr != nil {
			return RuntimeConfig{}, fmt.Errorf("close gateway config: %w", closeErr)
		}
	} else {
		payload = []byte(inline)
	}
	return ParseConfig(payload, os.LookupEnv)
}

func ParseConfig(payload []byte, lookupEnv func(string) (string, bool)) (RuntimeConfig, error) {
	if len(payload) == 0 || len(payload) > maxConfigBytes {
		return RuntimeConfig{}, fmt.Errorf("gateway config exceeds the supported byte limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var file FileConfig
	if err := decoder.Decode(&file); err != nil {
		return RuntimeConfig{}, fmt.Errorf("decode gateway config: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return RuntimeConfig{}, fmt.Errorf("gateway config contains trailing JSON values")
	}
	return file.resolve(lookupEnv)
}

func (file FileConfig) resolve(lookupEnv func(string) (string, bool)) (RuntimeConfig, error) {
	environment := strings.ToLower(strings.TrimSpace(file.Environment))
	if environment == "" {
		environment = "development"
	}
	if environment != "development" && environment != "test" && environment != "production" {
		return RuntimeConfig{}, fmt.Errorf("gateway environment must be development, test or production")
	}
	cfg := RuntimeConfig{
		Environment:          environment,
		ListenAddr:           strings.TrimSpace(file.ListenAddr),
		RequestTimeout:       durationMS(file.RequestTimeoutMS, defaultRequestTimeout),
		ShutdownTimeout:      durationMS(file.ShutdownTimeoutMS, defaultShutdownTimeout),
		MaxRequestBytes:      defaultInt64(file.MaxRequestBytes, defaultMaxRequestBytes),
		MaxProviderBodyBytes: defaultInt64(file.MaxProviderBodyBytes, defaultMaxProviderBodyBytes),
		MaxSSEEventBytes:     defaultInt64(file.MaxSSEEventBytes, defaultMaxSSEEventBytes),
		CircuitBreaker:       file.CircuitBreaker,
		GovernanceMode:       strings.ToUpper(strings.TrimSpace(file.GovernanceMode)),
		GovernanceRefresh:    durationMS(file.GovernanceRefreshMS, defaultGovernanceRefresh),
		Models:               append([]ModelConfig(nil), file.Models...),
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultGatewayAddr
	}
	if cfg.GovernanceMode == "" {
		if environment == "production" {
			cfg.GovernanceMode = GovernanceDatabase
		} else {
			cfg.GovernanceMode = GovernanceStatic
		}
	}
	if cfg.GovernanceMode != GovernanceStatic && cfg.GovernanceMode != GovernanceDatabase {
		return RuntimeConfig{}, fmt.Errorf("gateway governance_mode must be STATIC or DATABASE")
	}
	if cfg.GovernanceRefresh < time.Second || cfg.GovernanceRefresh > 5*time.Minute {
		return RuntimeConfig{}, fmt.Errorf("gateway governance refresh is outside supported bounds")
	}
	if environment == "production" && cfg.GovernanceMode != GovernanceDatabase {
		return RuntimeConfig{}, fmt.Errorf("production requires DATABASE governed workload state")
	}
	if cfg.RequestTimeout < time.Second || cfg.RequestTimeout > 10*time.Minute || cfg.ShutdownTimeout < time.Second || cfg.ShutdownTimeout > time.Minute {
		return RuntimeConfig{}, fmt.Errorf("gateway request/shutdown timeouts are outside supported bounds")
	}
	if cfg.MaxRequestBytes < 1024 || cfg.MaxRequestBytes > 16<<20 || cfg.MaxProviderBodyBytes < 1024 || cfg.MaxProviderBodyBytes > 64<<20 || cfg.MaxSSEEventBytes < 1024 || cfg.MaxSSEEventBytes > 2<<20 {
		return RuntimeConfig{}, fmt.Errorf("gateway request, provider-body or SSE event byte limits are outside supported bounds")
	}
	if cfg.CircuitBreaker.FailureThreshold == 0 {
		cfg.CircuitBreaker.FailureThreshold = 3
	}
	if cfg.CircuitBreaker.OpenDurationMS == 0 {
		cfg.CircuitBreaker.OpenDurationMS = 30000
	}
	if cfg.CircuitBreaker.FailureThreshold < 1 || cfg.CircuitBreaker.FailureThreshold > 100 || cfg.CircuitBreaker.OpenDurationMS < 100 || cfg.CircuitBreaker.OpenDurationMS > int64((10*time.Minute)/time.Millisecond) {
		return RuntimeConfig{}, fmt.Errorf("invalid circuit-breaker configuration")
	}
	if strings.TrimSpace(file.MetricsBearerSHA256) != "" {
		digest, err := parseDigest(file.MetricsBearerSHA256)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("metrics bearer digest: %w", err)
		}
		cfg.MetricsDigest = &digest
	}
	if cfg.GovernanceMode == GovernanceStatic {
		if len(file.Workloads) == 0 || len(file.Workloads) > 4096 {
			return RuntimeConfig{}, fmt.Errorf("STATIC governance requires 1-4096 configured workloads")
		}
		workloadIDs := make(map[string]struct{}, len(file.Workloads))
		keyDigests := make(map[[sha256.Size]byte]string, len(file.Workloads))
		for _, input := range file.Workloads {
			if !validIdentifier(input.ID) || !validIdentifier(input.TenantID) {
				return RuntimeConfig{}, fmt.Errorf("invalid workload or tenant identifier")
			}
			if _, exists := workloadIDs[input.ID]; exists {
				return RuntimeConfig{}, fmt.Errorf("duplicate workload id %q", input.ID)
			}
			workloadIDs[input.ID] = struct{}{}
			digest, err := parseDigest(input.KeySHA256)
			if err != nil {
				return RuntimeConfig{}, fmt.Errorf("workload %s key digest: %w", input.ID, err)
			}
			if prior, exists := keyDigests[digest]; exists {
				return RuntimeConfig{}, fmt.Errorf("workloads %s and %s share one credential digest", prior, input.ID)
			}
			keyDigests[digest] = input.ID
			if len(input.AllowedModels) == 0 || len(input.AllowedModels) > 256 || input.RequestsPerMinute < 1 || input.TokensPerMinute < 1 || input.CostMicroUSDPerMinute < 1 || input.MaxConcurrent < 1 {
				return RuntimeConfig{}, fmt.Errorf("workload %s has invalid model or budget limits", input.ID)
			}
			allowed := make(map[string]struct{}, len(input.AllowedModels))
			for _, alias := range input.AllowedModels {
				if !validIdentifier(alias) {
					return RuntimeConfig{}, fmt.Errorf("workload %s has invalid model alias", input.ID)
				}
				allowed[alias] = struct{}{}
			}
			cfg.Workloads = append(cfg.Workloads, ConfiguredWorkload{Workload: Workload{
				ID: input.ID, TenantID: input.TenantID, AllowedModels: allowed,
				RequestsPerMinute: input.RequestsPerMinute, TokensPerMinute: input.TokensPerMinute,
				CostMicroUSDPerMinute: input.CostMicroUSDPerMinute, MaxConcurrent: input.MaxConcurrent,
			}, KeyDigest: digest})
		}
	} else if len(file.Workloads) != 0 {
		return RuntimeConfig{}, fmt.Errorf("DATABASE governance does not accept static workload credentials")
	}
	if len(file.Providers) < 1 || len(file.Providers) > 64 {
		return RuntimeConfig{}, fmt.Errorf("gateway requires between 1 and 64 provider configurations")
	}
	providerIDs := make(map[string]struct{}, len(file.Providers))
	for _, provider := range file.Providers {
		provider.ID = strings.TrimSpace(provider.ID)
		provider.Kind = strings.ToUpper(strings.TrimSpace(provider.Kind))
		provider.BaseURL = strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
		provider.SecretEnv = strings.TrimSpace(provider.SecretEnv)
		provider.APIVersion = strings.TrimSpace(provider.APIVersion)
		if !validIdentifier(provider.ID) || (provider.Kind != ProviderKindOpenAI && provider.Kind != ProviderKindAnthropic) || !validEnvName(provider.SecretEnv) {
			return RuntimeConfig{}, fmt.Errorf("provider configuration is invalid")
		}
		if provider.APIVersion != "" && (len(provider.APIVersion) > 128 || !safeHeaderValue(provider.APIVersion)) {
			return RuntimeConfig{}, fmt.Errorf("provider %s API version is invalid", provider.ID)
		}
		if _, exists := providerIDs[provider.ID]; exists {
			return RuntimeConfig{}, fmt.Errorf("duplicate provider id %q", provider.ID)
		}
		providerIDs[provider.ID] = struct{}{}
		parsed, err := url.Parse(provider.BaseURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(environment != "production" && parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()))) || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return RuntimeConfig{}, fmt.Errorf("provider %s base URL must be a fixed HTTPS origin", provider.ID)
		}
		secret, ok := lookupEnv(provider.SecretEnv)
		secret = strings.TrimSpace(secret)
		if !ok || len(secret) < 8 || len(secret) > 4096 || !safeHeaderValue(secret) {
			return RuntimeConfig{}, fmt.Errorf("provider %s secret environment reference is unavailable", provider.ID)
		}
		timeout := durationMS(provider.TimeoutMS, defaultProviderTimeout)
		if timeout < time.Second || timeout > cfg.RequestTimeout {
			return RuntimeConfig{}, fmt.Errorf("provider %s timeout is outside supported bounds", provider.ID)
		}
		requireUsage := true
		if provider.RequireUsage != nil {
			requireUsage = *provider.RequireUsage
		}
		cfg.Providers = append(cfg.Providers, ResolvedProviderConfig{ProviderConfig: provider, Secret: secret, Timeout: timeout, RequireUsage: requireUsage})
	}
	if len(file.Models) == 0 || len(file.Models) > 256 {
		return RuntimeConfig{}, fmt.Errorf("gateway requires 1-256 model aliases")
	}
	aliases := make(map[string]struct{}, len(file.Models))
	for index := range cfg.Models {
		model := &cfg.Models[index]
		model.Alias = strings.TrimSpace(model.Alias)
		if !validIdentifier(model.Alias) || len(model.Routes) == 0 || len(model.Routes) > 32 {
			return RuntimeConfig{}, fmt.Errorf("model alias configuration is invalid")
		}
		if _, exists := aliases[model.Alias]; exists {
			return RuntimeConfig{}, fmt.Errorf("duplicate model alias %q", model.Alias)
		}
		aliases[model.Alias] = struct{}{}
		routeIDs := make(map[string]struct{}, len(model.Routes))
		for routeIndex := range model.Routes {
			route := &model.Routes[routeIndex]
			route.ID = strings.TrimSpace(route.ID)
			route.ProviderID = strings.TrimSpace(route.ProviderID)
			route.Model = strings.TrimSpace(route.Model)
			if !validIdentifier(route.ID) || !validIdentifier(route.ProviderID) || !validIdentifier(route.Model) || route.Weight < 1 || route.Weight > 100000 {
				return RuntimeConfig{}, fmt.Errorf("model %s has invalid route", model.Alias)
			}
			if _, exists := providerIDs[route.ProviderID]; !exists {
				return RuntimeConfig{}, fmt.Errorf("model %s references unknown provider %s", model.Alias, route.ProviderID)
			}
			if _, exists := routeIDs[route.ID]; exists {
				return RuntimeConfig{}, fmt.Errorf("model %s has duplicate route %s", model.Alias, route.ID)
			}
			routeIDs[route.ID] = struct{}{}
			if route.InputMicroUSDPerMillionTokens < 0 || route.OutputMicroUSDPerMillionTokens < 0 || route.InputMicroUSDPerMillionTokens > 1_000_000_000_000 || route.OutputMicroUSDPerMillionTokens > 1_000_000_000_000 {
				return RuntimeConfig{}, fmt.Errorf("model %s route %s has invalid token prices", model.Alias, route.ID)
			}
		}
	}
	for _, workload := range cfg.Workloads {
		for alias := range workload.AllowedModels {
			if _, exists := aliases[alias]; !exists {
				return RuntimeConfig{}, fmt.Errorf("workload %s references unknown model alias %s", workload.ID, alias)
			}
		}
	}
	sort.Slice(cfg.Workloads, func(i, j int) bool { return cfg.Workloads[i].ID < cfg.Workloads[j].ID })
	sort.Slice(cfg.Providers, func(i, j int) bool { return cfg.Providers[i].ID < cfg.Providers[j].ID })
	sort.Slice(cfg.Models, func(i, j int) bool { return cfg.Models[i].Alias < cfg.Models[j].Alias })
	return cfg, nil
}

func parseDigest(value string) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(decoded) != sha256.Size {
		return digest, fmt.Errorf("must be a 64-character SHA-256 hex digest")
	}
	copy(digest[:], decoded)
	return digest, nil
}

func durationMS(value int64, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	if value < 0 || value > math.MaxInt64/int64(time.Millisecond) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(value) * time.Millisecond
}

func defaultInt64(value, fallback int64) int64 {
	if value == 0 {
		return fallback
	}
	return value
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func validEnvName(value string) bool {
	if value == "" || len(value) > 256 {
		return false
	}
	for index, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (index > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func safeHeaderValue(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}
