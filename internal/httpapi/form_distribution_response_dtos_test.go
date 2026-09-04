package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestFormDistributionResponsesUseCanonicalJSONFields(t *testing.T) {
	now := time.Date(2026, time.September, 4, 8, 0, 0, 0, time.UTC)
	bundle := evidence.DistributionBundle{
		Distribution: evidence.FormDistribution{
			ID: "distribution-1", TenantID: "tenant-1", LegalEntityID: "entity-1",
			FormTemplateID: "form-1", FormTemplateVersion: 2, SubjectType: "VENDOR", SubjectID: "vendor-1",
			Title: "Annual evidence request", Purpose: "Confirm current controls", AccessPolicy: evidence.AccessDirectMagicLink,
			Status: evidence.DistributionOpen, Deadline: now.Add(24 * time.Hour), RouteExpiresAt: now.Add(12 * time.Hour),
			CreatedBy: "owner-1", Version: 3, CreatedAt: now, UpdatedAt: now,
		},
		Recipients: []evidence.DistributionRecipient{{
			ID: "recipient-1", DistributionID: "distribution-1", TenantID: "tenant-1", LegalEntityID: "entity-1",
			Role: evidence.RecipientTo, Type: evidence.RecipientInternalPrincipal, PrincipalID: "reviewer-1",
			RequestID: "request-1", State: evidence.DistributionRecipientPending, Version: 1, CreatedAt: now, UpdatedAt: now,
		}},
		Workspace: evidence.ResponseWorkspace{
			ID: "workspace-1", TenantID: "tenant-1", LegalEntityID: "entity-1", DistributionID: "distribution-1",
			Status: evidence.ResponseWorkspaceOpen, Version: 1, CreatedAt: now, UpdatedAt: now,
		},
	}

	assertCanonicalResponseJSON(t, distributionBundleJSON(bundle), []string{
		`"distribution"`, `"form_template_id":"form-1"`, `"recipients"`,
		`"distribution_id":"distribution-1"`, `"workspace"`, `"updated_at"`,
	})
	assertCanonicalResponseJSON(t, distributionPageDTO(evidence.DistributionPage{
		Items: []evidence.FormDistribution{bundle.Distribution}, NextCursor: "next-page",
	}), []string{`"items"`, `"next_cursor":"next-page"`, `"legal_entity_id":"entity-1"`})
}

func assertCanonicalResponseJSON(t *testing.T, value any, required []string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	body := string(encoded)
	for _, field := range required {
		if !strings.Contains(body, field) {
			t.Fatalf("response missing %s: %s", field, body)
		}
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertNoGoFieldNames(t, decoded)
}

func assertNoGoFieldNames(t *testing.T, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key != strings.ToLower(key) {
				t.Fatalf("response exposed non-canonical field %q", key)
			}
			assertNoGoFieldNames(t, child)
		}
	case []any:
		for _, child := range typed {
			assertNoGoFieldNames(t, child)
		}
	}
}
