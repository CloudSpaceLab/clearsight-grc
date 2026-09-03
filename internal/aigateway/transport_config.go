package aigateway

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	TransportStatic   = "STATIC"
	TransportDatabase = "DATABASE"

	ProviderStateEnabled   = "ENABLED"
	ProviderStateSuspended = "SUSPENDED"
)

type TransportProviderConfig struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	BaseURL      string   `json:"base_url"`
	SecretRef    string   `json:"secret_ref"`
	APIVersion   string   `json:"api_version,omitempty"`
	TimeoutMS    int64    `json:"timeout_ms,omitempty"`
	RequireUsage *bool    `json:"require_usage,omitempty"`
	Regions      []string `json:"regions,omitempty"`
	State        string   `json:"state"`
}

type TransportDefinition struct {
	CircuitBreaker CircuitBreakerConfig      `json:"circuit_breaker"`
	Providers      []TransportProviderConfig `json:"providers"`
	Models         []ModelConfig             `json:"models"`
}

type TransportSnapshot struct {
	ID          string              `json:"id"`
	TenantID    string              `json:"tenant_id"`
	Environment string              `json:"environment"`
	Version     int64               `json:"version"`
	Checksum    string              `json:"checksum"`
	Definition  TransportDefinition `json:"definition"`
}

type TransportSnapshotSource interface {
	ActiveTransportSnapshot(context.Context, string, string) (TransportSnapshot, error)
	Ready() bool
}

type SecretResolver interface {
	ResolveSecret(context.Context, string) (string, error)
}

type RuntimeTransportConfig struct {
	CircuitBreaker CircuitBreakerConfig
	Providers      []ResolvedProviderConfig
	Models         []ModelConfig
}

type EnvironmentSecretResolver struct{}

func (EnvironmentSecretResolver) ResolveSecret(_ context.Context, ref string) (string, error) {
	name, ok := environmentSecretName(ref)
	if !ok {
		return "", fmt.Errorf("unsupported provider secret reference")
	}
	secret, found := os.LookupEnv(name)
	secret = strings.TrimSpace(secret)
	if !found || len(secret) < 8 || len(secret) > 4096 || !safeHeaderValue(secret) {
		return "", fmt.Errorf("provider secret reference is unavailable")
	}
	return secret, nil
}

func environmentSecretName(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "env:") {
		return "", false
	}
	name := strings.TrimSpace(strings.TrimPrefix(ref, "env:"))
	return name, validEnvName(name)
}

func ValidateTransportDefinition(environment string, requestTimeout time.Duration, input TransportDefinition) (TransportDefinition, error) {
	environment = strings.ToLower(strings.TrimSpace(environment))
	if environment != "development" && environment != "test" && environment != "production" {
		return TransportDefinition{}, fmt.Errorf("gateway transport environment must be development, test or production")
	}
	if requestTimeout <= 0 {
		requestTimeout = 10 * time.Minute
	}
	definition := TransportDefinition{
		CircuitBreaker: input.CircuitBreaker,
		Providers:      append([]TransportProviderConfig(nil), input.Providers...),
		Models:         cloneModelConfigs(input.Models),
	}
	if definition.CircuitBreaker.FailureThreshold == 0 {
		definition.CircuitBreaker.FailureThreshold = 3
	}
	if definition.CircuitBreaker.OpenDurationMS == 0 {
		definition.CircuitBreaker.OpenDurationMS = 30000
	}
	if definition.CircuitBreaker.FailureThreshold < 1 || definition.CircuitBreaker.FailureThreshold > 100 || definition.CircuitBreaker.OpenDurationMS < 100 || definition.CircuitBreaker.OpenDurationMS > int64((10*time.Minute)/time.Millisecond) {
		return TransportDefinition{}, fmt.Errorf("invalid circuit-breaker configuration")
	}
	if len(definition.Providers) < 1 || len(definition.Providers) > 64 {
		return TransportDefinition{}, fmt.Errorf("gateway requires between 1 and 64 provider configurations")
	}
	providerIDs := make(map[string]struct{}, len(definition.Providers))
	for index := range definition.Providers {
		provider := &definition.Providers[index]
		provider.ID = strings.TrimSpace(provider.ID)
		provider.Name = strings.TrimSpace(provider.Name)
		provider.Kind = strings.ToUpper(strings.TrimSpace(provider.Kind))
		provider.BaseURL = strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
		provider.SecretRef = strings.TrimSpace(provider.SecretRef)
		provider.APIVersion = strings.TrimSpace(provider.APIVersion)
		provider.State = strings.ToUpper(strings.TrimSpace(provider.State))
		if provider.State == "" {
			provider.State = ProviderStateEnabled
		}
		if !validIdentifier(provider.ID) || provider.Name == "" || len(provider.Name) > 160 || (provider.Kind != ProviderKindOpenAI && provider.Kind != ProviderKindAnthropic) {
			return TransportDefinition{}, fmt.Errorf("provider configuration is invalid")
		}
		if provider.State != ProviderStateEnabled && provider.State != ProviderStateSuspended {
			return TransportDefinition{}, fmt.Errorf("provider %s state is invalid", provider.ID)
		}
		if len(provider.SecretRef) > 512 || !safeHeaderValue(provider.SecretRef) {
			return TransportDefinition{}, fmt.Errorf("provider %s secret reference is invalid", provider.ID)
		}
		if _, ok := environmentSecretName(provider.SecretRef); !ok {
			return TransportDefinition{}, fmt.Errorf("provider %s secret reference must use env:NAME", provider.ID)
		}
		if provider.APIVersion != "" && (len(provider.APIVersion) > 128 || !safeHeaderValue(provider.APIVersion)) {
			return TransportDefinition{}, fmt.Errorf("provider %s API version is invalid", provider.ID)
		}
		if _, exists := providerIDs[provider.ID]; exists {
			return TransportDefinition{}, fmt.Errorf("duplicate provider id %q", provider.ID)
		}
		providerIDs[provider.ID] = struct{}{}
		parsed, err := url.Parse(provider.BaseURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(environment != "production" && parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()))) || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return TransportDefinition{}, fmt.Errorf("provider %s base URL must be a fixed HTTPS origin", provider.ID)
		}
		timeout := durationMS(provider.TimeoutMS, defaultProviderTimeout)
		if timeout < time.Second || timeout > requestTimeout {
			return TransportDefinition{}, fmt.Errorf("provider %s timeout is outside supported bounds", provider.ID)
		}
		regions := make([]string, 0, len(provider.Regions))
		seenRegions := make(map[string]struct{}, len(provider.Regions))
		for _, raw := range provider.Regions {
			region := strings.ToUpper(strings.TrimSpace(raw))
			if region == "" || len(region) > 64 || !validIdentifier(region) {
				return TransportDefinition{}, fmt.Errorf("provider %s region is invalid", provider.ID)
			}
			if _, exists := seenRegions[region]; exists {
				continue
			}
			seenRegions[region] = struct{}{}
			regions = append(regions, region)
		}
		sort.Strings(regions)
		provider.Regions = regions
	}
	if len(definition.Models) == 0 || len(definition.Models) > 256 {
		return TransportDefinition{}, fmt.Errorf("gateway requires 1-256 model aliases")
	}
	aliases := make(map[string]struct{}, len(definition.Models))
	for index := range definition.Models {
		model := &definition.Models[index]
		model.Alias = strings.TrimSpace(model.Alias)
		if !validIdentifier(model.Alias) || len(model.Routes) == 0 || len(model.Routes) > 32 {
			return TransportDefinition{}, fmt.Errorf("model alias configuration is invalid")
		}
		if _, exists := aliases[model.Alias]; exists {
			return TransportDefinition{}, fmt.Errorf("duplicate model alias %q", model.Alias)
		}
		aliases[model.Alias] = struct{}{}
		routeIDs := make(map[string]struct{}, len(model.Routes))
		for routeIndex := range model.Routes {
			route := &model.Routes[routeIndex]
			route.ID = strings.TrimSpace(route.ID)
			route.ProviderID = strings.TrimSpace(route.ProviderID)
			route.Model = strings.TrimSpace(route.Model)
			if !validIdentifier(route.ID) || !validIdentifier(route.ProviderID) || !validIdentifier(route.Model) || route.Weight < 1 || route.Weight > 100000 {
				return TransportDefinition{}, fmt.Errorf("model %s has invalid route", model.Alias)
			}
			if _, exists := providerIDs[route.ProviderID]; !exists {
				return TransportDefinition{}, fmt.Errorf("model %s references unknown provider %s", model.Alias, route.ProviderID)
			}
			if _, exists := routeIDs[route.ID]; exists {
				return TransportDefinition{}, fmt.Errorf("model %s has duplicate route %s", model.Alias, route.ID)
			}
			routeIDs[route.ID] = struct{}{}
			if route.InputMicroUSDPerMillionTokens < 0 || route.OutputMicroUSDPerMillionTokens < 0 || route.InputMicroUSDPerMillionTokens > 1_000_000_000_000 || route.OutputMicroUSDPerMillionTokens > 1_000_000_000_000 {
				return TransportDefinition{}, fmt.Errorf("model %s route %s has invalid token prices", model.Alias, route.ID)
			}
		}
	}
	sort.Slice(definition.Providers, func(i, j int) bool { return definition.Providers[i].ID < definition.Providers[j].ID })
	sort.Slice(definition.Models, func(i, j int) bool { return definition.Models[i].Alias < definition.Models[j].Alias })
	return definition, nil
}

func ResolveTransportDefinition(ctx context.Context, environment string, requestTimeout time.Duration, input TransportDefinition, resolver SecretResolver) (RuntimeTransportConfig, error) {
	if resolver == nil {
		return RuntimeTransportConfig{}, fmt.Errorf("provider secret resolver is unavailable")
	}
	definition, err := ValidateTransportDefinition(environment, requestTimeout, input)
	if err != nil {
		return RuntimeTransportConfig{}, err
	}
	result := RuntimeTransportConfig{CircuitBreaker: definition.CircuitBreaker, Models: cloneModelConfigs(definition.Models)}
	for _, provider := range definition.Providers {
		if provider.State != ProviderStateEnabled {
			continue
		}
		secret, err := resolver.ResolveSecret(ctx, provider.SecretRef)
		if err != nil {
			return RuntimeTransportConfig{}, fmt.Errorf("resolve provider %s secret reference: %w", provider.ID, err)
		}
		requireUsage := true
		if provider.RequireUsage != nil {
			requireUsage = *provider.RequireUsage
		}
		result.Providers = append(result.Providers, ResolvedProviderConfig{
			ProviderConfig: ProviderConfig{
				ID: provider.ID, Kind: provider.Kind, BaseURL: provider.BaseURL,
				APIVersion: provider.APIVersion, TimeoutMS: provider.TimeoutMS,
			},
			Secret: secret, Timeout: durationMS(provider.TimeoutMS, defaultProviderTimeout), RequireUsage: requireUsage,
		})
	}
	if len(result.Providers) == 0 {
		return RuntimeTransportConfig{}, fmt.Errorf("gateway transport has no enabled providers")
	}
	return result, nil
}

func cloneModelConfigs(values []ModelConfig) []ModelConfig {
	out := make([]ModelConfig, len(values))
	for index, model := range values {
		out[index] = model
		out[index].Routes = append([]RouteConfig(nil), model.Routes...)
	}
	return out
}
