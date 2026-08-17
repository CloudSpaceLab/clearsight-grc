package evidence

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEvidenceSourceEndpointIsInputOnly(t *testing.T) {
	value := Source{ID: "source-1", Endpoint: "https://example.invalid/source"}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "endpoint") || strings.Contains(string(encoded), value.Endpoint) {
		t.Fatalf("source JSON exposed endpoint configuration: %s", encoded)
	}
}

func TestCreateSourceRejectsUnsafeReferenceEndpoint(t *testing.T) {
	repository := NewMemoryRepository(nil, nil)
	service := NewService(repository, nil)
	service.now = func() time.Time { return time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC) }
	base := CreateSourceInput{
		TenantID:                 "bank-demo",
		Code:                     "CORE",
		Name:                     "Core source",
		Type:                     SourceSystem,
		AuthorityClass:           "SYSTEM_OF_RECORD",
		ExpectedFreshnessMinutes: 15,
	}
	for _, endpoint := range []string{
		strings.Repeat("x", (32<<10)+1),
		"https://example.invalid/source\nInjected: value",
		"https://example.invalid/source\x00tail",
	} {
		input := base
		input.Endpoint = endpoint
		if _, err := service.CreateSource(context.Background(), input); err == nil {
			t.Fatalf("unsafe endpoint %q was accepted", endpoint)
		}
	}
}
