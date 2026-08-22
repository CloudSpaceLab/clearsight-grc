package continuity

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCreateMatterRejectsInvalidAccessMetadata(t *testing.T) {
	tests := []struct {
		name  string
		scope json.RawMessage
	}{
		{name: "non-string access", scope: json.RawMessage(`{"access":{"unexpected":true}}`)},
		{name: "unknown access", scope: json.RawMessage(`{"access":"SECRET"}`)},
		{name: "restricted without allow list", scope: json.RawMessage(`{"access":"RESTRICTED"}`)},
		{name: "restricted empty allow list", scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":[]}`)},
		{name: "restricted mixed-type allow list", scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-1",42]}`)},
		{name: "restricted blank-only allow list", scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":[" "]}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := NewMemoryRepository()
			service := NewService(repo)
			_, err := service.CreateMatter(context.Background(), CreateMatterInput{
				TenantID: "bank", Type: MatterControlGap, Priority: 3,
				Title: "Access metadata check", Summary: "Invalid access metadata must not be persisted.",
				Scope: test.scope,
			})
			if err == nil {
				t.Fatal("invalid Matter access metadata was persisted")
			}
			matters, listErr := service.ListMatters(context.Background(), "bank", "", 10)
			if listErr != nil {
				t.Fatal(listErr)
			}
			if len(matters) != 0 {
				t.Fatalf("invalid Matter write created %d records", len(matters))
			}
		})
	}
}

func TestCreateMatterAcceptsCanonicalAccessMetadata(t *testing.T) {
	tests := []struct {
		name  string
		scope json.RawMessage
	}{
		{name: "implicit tenant internal", scope: json.RawMessage(`{"business_area":"Treasury"}`)},
		{name: "explicit internal", scope: json.RawMessage(`{"access":"INTERNAL"}`)},
		{name: "public", scope: json.RawMessage(`{"access":"PUBLIC"}`)},
		{name: "restricted", scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-1","person-1"," "]}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(NewMemoryRepository())
			created, err := service.CreateMatter(context.Background(), CreateMatterInput{
				TenantID: "bank", Type: MatterControlGap, Priority: 3,
				Title: "Access metadata check", Summary: "Canonical access metadata remains writable.",
				Scope: test.scope,
			})
			if err != nil {
				t.Fatalf("valid Matter access metadata was rejected: %v", err)
			}
			if _, valid := ParseMatterAccessPolicy(created.Matter.Scope); !valid {
				t.Fatalf("persisted scope is not readable by the canonical access parser: %s", created.Matter.Scope)
			}
		})
	}
}
