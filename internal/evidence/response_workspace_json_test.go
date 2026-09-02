package evidence

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestResponseWorkspaceUsesCanonicalExternalJSONNames(t *testing.T) {
	now := time.Date(2026, time.September, 2, 9, 13, 52, 0, time.UTC)
	payload, err := json.Marshal(ResponseWorkspace{
		ID:             "workspace-1",
		TenantID:       "tenant-1",
		LegalEntityID:  "entity-1",
		DistributionID: "distribution-1",
		Status:         ResponseWorkspaceOpen,
		Version:        2,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("marshal response workspace: %v", err)
	}

	encoded := string(payload)
	for _, field := range []string{
		`"id":"workspace-1"`,
		`"tenant_id":"tenant-1"`,
		`"legal_entity_id":"entity-1"`,
		`"distribution_id":"distribution-1"`,
		`"status":"OPEN"`,
		`"version":2`,
		`"created_at":`,
		`"updated_at":`,
	} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("canonical workspace JSON is missing %s: %s", field, encoded)
		}
	}
	for _, legacy := range []string{`"ID":`, `"Status":`, `"Version":`} {
		if strings.Contains(encoded, legacy) {
			t.Fatalf("workspace JSON exposed Go field name %s: %s", legacy, encoded)
		}
	}
}
