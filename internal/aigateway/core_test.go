package aigateway

import (
	"context"
	"crypto/sha256"
	"errors"
	"math"
	"testing"
	"time"
)

func TestAuthenticatorMatchesConfiguredDigest(t *testing.T) {
	digest := sha256.Sum256([]byte("correct-workload-key"))
	auth := newAuthenticator([]ConfiguredWorkload{{Workload: Workload{ID: "w", TenantID: "t", AllowedModels: map[string]struct{}{"m": {}}}, KeyDigest: digest}})
	workload, err := auth.authenticate("Bearer correct-workload-key")
	if err != nil || workload.ID != "w" {
		t.Fatalf("authenticate = %#v, %v", workload, err)
	}
	if _, err := auth.authenticate("Bearer wrong-workload-key"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("wrong key error = %v", err)
	}
	delete(workload.AllowedModels, "m")
	again, err := auth.authenticate("Bearer correct-workload-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := again.AllowedModels["m"]; !exists {
		t.Fatal("authenticated workload exposed mutable configured model state")
	}
}

func TestBudgetReservationReleasesConcurrencyAcrossMinuteBoundary(t *testing.T) {
	manager := newBudgetManager()
	workload := Workload{ID: "w", RequestsPerMinute: 10, TokensPerMinute: 10000, CostMicroUSDPerMinute: 10000, MaxConcurrent: 1}
	request := Request{Messages: []Message{{Role: RoleUser, Text: "hello"}}, MaxOutputTokens: 100}
	start := time.Date(2026, 8, 15, 12, 0, 59, 0, time.UTC)
	reservation, err := manager.reserve(start, workload, request, TokenPrice{InputPerMillion: 1000, OutputPerMillion: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.reserve(start.Add(2*time.Minute), workload, request, TokenPrice{}); !errors.Is(err, ErrConcurrency) {
		t.Fatalf("concurrent reservation error = %v", err)
	}
	if _, err := reservation.finish(Usage{InputTokens: 5, OutputTokens: 10}, TokenPrice{InputPerMillion: 1000, OutputPerMillion: 1000}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.reserve(start.Add(2*time.Minute), workload, request, TokenPrice{}); err != nil {
		t.Fatalf("concurrency was not released: %v", err)
	}
}

func TestTokenCostRejectsOverflow(t *testing.T) {
	if _, err := tokenCost(1<<62, 1<<62, TokenPrice{InputPerMillion: 1 << 20, OutputPerMillion: 1 << 20}); err == nil {
		t.Fatal("expected token cost overflow")
	}
}

func TestTokenCostRoundsWithoutOverflow(t *testing.T) {
	got, err := tokenCost(999_999, 0, TokenPrice{InputPerMillion: 1_000_000, OutputPerMillion: 0})
	if err != nil || got != 999_999 {
		t.Fatalf("token cost = %d, %v", got, err)
	}
	if !exceedsLimit(math.MaxInt64-1, 2, math.MaxInt64) {
		t.Fatal("overflowing budget addition was accepted")
	}
}

func TestCircuitBreakerHalfOpenRecovery(t *testing.T) {
	breaker := newCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 2, OpenDurationMS: 1000})
	now := time.Now()
	if !breaker.allow(now) {
		t.Fatal("closed circuit denied request")
	}
	breaker.failure(now)
	if !breaker.allow(now) {
		t.Fatal("circuit opened before threshold")
	}
	breaker.failure(now)
	if breaker.allow(now.Add(500 * time.Millisecond)) {
		t.Fatal("open circuit allowed early request")
	}
	if !breaker.allow(now.Add(2 * time.Second)) {
		t.Fatal("half-open probe was not allowed")
	}
	if breaker.allow(now.Add(2 * time.Second)) {
		t.Fatal("second half-open probe was allowed")
	}
	breaker.success()
	if !breaker.allow(now.Add(2 * time.Second)) {
		t.Fatal("successful probe did not close circuit")
	}
}

func TestCircuitNeutralOutcomeDoesNotClaimProviderRecovery(t *testing.T) {
	breaker := newCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 2, OpenDurationMS: 1000})
	now := time.Now()
	breaker.failure(now)
	breaker.neutral()
	breaker.failure(now)
	if breaker.allow(now.Add(500 * time.Millisecond)) {
		t.Fatal("request-specific outcome erased prior provider-health failure")
	}
	if !breaker.allow(now.Add(2 * time.Second)) {
		t.Fatal("half-open probe was not released after neutral outcome")
	}
	breaker.neutral()
	if !breaker.allow(now.Add(2 * time.Second)) {
		t.Fatal("neutral half-open request did not release the single probe")
	}
}

func TestCompletedZeroUsageReleasesRequiredUsageReservation(t *testing.T) {
	provider := &fakeProvider{id: "primary", response: Response{FinishReason: "stop", Usage: Usage{}}}
	config := testRuntimeConfig()
	config.Models[0].Routes = config.Models[0].Routes[:1]
	request := validCanonicalRequest(false)
	request.ModelAlias = "governed-chat"
	estimate, err := estimatedTokens(request)
	if err != nil {
		t.Fatal(err)
	}
	config.Workloads[0].TokensPerMinute = estimate
	config.Workloads[0].RequestsPerMinute = 2
	gateway, err := newGatewayWithProviders(config, map[string]*providerRuntime{
		"primary": {provider: provider, config: ResolvedProviderConfig{RequireUsage: true}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		request.ID = "req_001122334455667" + string(rune('0'+attempt))
		if _, _, err := gateway.Complete(context.Background(), config.Workloads[0].Workload, request); err != nil {
			t.Fatalf("attempt %d failed after trustworthy zero usage: %v", attempt, err)
		}
	}
}

func TestRouterUsesWeightedInitialRouteAndAllFallbacks(t *testing.T) {
	config := RuntimeConfig{CircuitBreaker: CircuitBreakerConfig{FailureThreshold: 2, OpenDurationMS: 1000}, Models: []ModelConfig{{Alias: "m", Routes: []RouteConfig{
		{ID: "r1", ProviderID: "p1", Model: "a", Weight: 2}, {ID: "r2", ProviderID: "p2", Model: "b", Weight: 1},
	}}}}
	providers := map[string]*providerRuntime{"p1": {provider: &fakeProvider{id: "p1"}}, "p2": {provider: &fakeProvider{id: "p2"}}}
	router, err := newRouter(config, providers)
	if err != nil {
		t.Fatal(err)
	}
	workload := Workload{AllowedModels: map[string]struct{}{"m": {}}}
	first, _, _ := router.candidates(workload, "m")
	second, _, _ := router.candidates(workload, "m")
	third, _, _ := router.candidates(workload, "m")
	if first[0].ID != "r1" || second[0].ID != "r1" || third[0].ID != "r2" {
		t.Fatalf("weighted order = %s,%s,%s", first[0].ID, second[0].ID, third[0].ID)
	}
	for _, candidates := range [][]*routeRuntime{first, second, third} {
		if len(candidates) != 2 {
			t.Fatalf("fallback candidates = %d", len(candidates))
		}
	}
}

func TestOpenCircuitsDoNotConsumeTokenOrCostReservation(t *testing.T) {
	primary := &fakeProvider{id: "primary"}
	secondary := &fakeProvider{id: "secondary"}
	config := testRuntimeConfig()
	request := validCanonicalRequest(false)
	request.ModelAlias = "governed-chat"
	estimate, err := estimatedTokens(request)
	if err != nil {
		t.Fatal(err)
	}
	config.Workloads[0].TokensPerMinute = estimate
	config.Workloads[0].CostMicroUSDPerMinute = 1
	config.Workloads[0].RequestsPerMinute = 10
	gateway, err := newGatewayWithProviders(config, map[string]*providerRuntime{
		"primary": {provider: primary}, "secondary": {provider: secondary},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, route := range gateway.router.aliases["governed-chat"].routes {
		route.breaker.failureThreshold = 1
		route.breaker.failure(now)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, _, err := gateway.Complete(context.Background(), config.Workloads[0].Workload, request); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("attempt %d error=%v", attempt, err)
		}
	}
	primaryComplete, _ := primary.counts()
	secondaryComplete, _ := secondary.counts()
	if primaryComplete != 0 || secondaryComplete != 0 {
		t.Fatalf("open-circuit providers were called: primary=%d secondary=%d", primaryComplete, secondaryComplete)
	}
}

func TestValidateRequestPreservesPortableToolSemantics(t *testing.T) {
	base := Request{
		ID: "req_0011223344556677", Protocol: ProtocolChat, ModelAlias: "m",
		Messages: []Message{{Role: RoleUser, Text: "hello"}}, MaxOutputTokens: 64,
		Tools: []ToolDefinition{{Name: "lookup", Parameters: []byte(`{"type":"object"}`)}},
	}
	strict := base
	strict.Tools = append([]ToolDefinition(nil), base.Tools...)
	strict.Tools[0].Strict = true
	if err := ValidateRequest(strict); err == nil {
		t.Fatal("strict tool schema was silently accepted across incompatible providers")
	}
	temperature := 1.1
	tooHot := base
	tooHot.Temperature = &temperature
	if err := ValidateRequest(tooHot); err == nil {
		t.Fatal("cross-provider temperature mismatch was accepted")
	}
}

func TestValidateRequestRequiresExactToolCallResolution(t *testing.T) {
	request := Request{
		ID: "req_0011223344556677", Protocol: ProtocolChat, ModelAlias: "m", MaxOutputTokens: 64,
		Messages: []Message{
			{Role: RoleUser, Text: "look it up"},
			{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "call_1", Name: "lookup", Arguments: `{"id":1}`}}},
			{Role: RoleTool, ToolCallID: "call_1", Text: `{"name":"branch"}`},
			{Role: RoleUser, Text: "continue"},
		},
	}
	if err := ValidateRequest(request); err != nil {
		t.Fatalf("valid tool exchange rejected: %v", err)
	}

	dangling := request
	dangling.Messages = append([]Message(nil), request.Messages[:2]...)
	if err := ValidateRequest(dangling); err == nil {
		t.Fatal("dangling assistant function call was accepted")
	}

	unknownResult := request
	unknownResult.Messages = append([]Message(nil), request.Messages...)
	unknownResult.Messages[2].ToolCallID = "call_unknown"
	if err := ValidateRequest(unknownResult); err == nil {
		t.Fatal("tool result for an unknown call was accepted")
	}
}
