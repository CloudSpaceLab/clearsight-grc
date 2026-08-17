package aigateway

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func BenchmarkWarmRouting(b *testing.B) {
	config := testRuntimeConfig()
	providers := map[string]*providerRuntime{
		"primary": {provider: &fakeProvider{id: "primary"}}, "secondary": {provider: &fakeProvider{id: "secondary"}},
	}
	router, err := newRouter(config, providers)
	if err != nil {
		b.Fatal(err)
	}
	workload := config.Workloads[0].Workload
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := router.candidates(workload, "governed-chat"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuthenticatedHTTP(b *testing.B) {
	primary := &fakeProvider{id: "primary", response: Response{CreatedAt: time.Unix(1, 0), Text: "ok", FinishReason: "stop", Usage: Usage{InputTokens: 4, OutputTokens: 1}}}
	secondary := &fakeProvider{id: "secondary"}
	config := testRuntimeConfig()
	config.Workloads[0].RequestsPerMinute = 1_000_000_000
	config.Workloads[0].TokensPerMinute = 1_000_000_000_000
	config.Workloads[0].CostMicroUSDPerMinute = 1_000_000_000_000
	secondary.response = primary.response
	gateway, err := newGatewayWithProviders(config, map[string]*providerRuntime{
		"primary": {provider: primary}, "secondary": {provider: secondary},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		b.Fatal(err)
	}
	handler := NewHTTPHandler(gateway, nil)
	body := `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64}`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+testWorkloadKey)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Request-ID", "req_ffeeddccbbaa99887766554433221100")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("status=%d", recorder.Code)
		}
	}
}
