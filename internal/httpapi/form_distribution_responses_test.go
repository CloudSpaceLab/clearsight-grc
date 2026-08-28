package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestResponseRevisionJSONOmitsInternalScopeAndSubmissionIdentifiers(t *testing.T) {
	score := 84.5
	value := evidence.ResponseRevision{
		ID: "revision-a", TenantID: "bank-secret", LegalEntityID: "entity-secret",
		DistributionID: "distribution-secret", WorkspaceID: "workspace-secret", SubmissionID: "submission-secret",
		Revision: 3, SupersedesRevisionID: "revision-b", AchievedAssurance: evidence.AssuranceEmailVerified,
		SignoffSummary: map[string]any{"attested": true}, ComplianceScore: &score, ScoredWeightCoverage: 100,
		State: evidence.ResponseRevisionFinal, CriticalFieldResults: []map[string]any{{"field_id": "critical"}},
		ScoringPolicyVersion: "policy-1", Current: true, CreatedAt: time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC),
	}

	encoded, err := json.Marshal(responseRevisionJSON(value))
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, secret := range []string{"bank-secret", "entity-secret", "distribution-secret", "workspace-secret", "submission-secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response projection leaked internal identifier %q: %s", secret, body)
		}
	}
	for _, required := range []string{`"id":"revision-a"`, `"revision":3`, `"current":true`, `"compliance_score":84.5`} {
		if !strings.Contains(body, required) {
			t.Fatalf("response projection missing %s: %s", required, body)
		}
	}
}
