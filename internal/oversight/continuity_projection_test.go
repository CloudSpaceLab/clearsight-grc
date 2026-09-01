package oversight

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestMatterAggregateProjectionExcludesRestrictedAndUnknownScopes(t *testing.T) {
	now := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	aggregates := []continuity.MatterAggregate{
		{Matter: oversightMatter("visible", json.RawMessage(`{"access":"INTERNAL"}`), now)},
		{Matter: oversightMatter("restricted", json.RawMessage(`{"access":"RESTRICTED"}`), now)},
		{Matter: oversightMatter("unknown", json.RawMessage(`{"access":{"unexpected":true}}`), now)},
	}

	value := FromMatterAggregates("bank", "bank-ng", aggregates, now)
	if value.Coverage.Population != 3 || value.Coverage.Excluded == nil || *value.Coverage.Excluded != 1 || value.Coverage.Unknown == nil || *value.Coverage.Unknown != 1 {
		t.Fatalf("coverage=%#v", value.Coverage)
	}
	if value.Counts.CriticalHigh != 1 || len(value.Interventions) != 1 || value.Interventions[0].TargetID != "visible" {
		t.Fatalf("restricted records leaked into projection: counts=%#v interventions=%#v", value.Counts, value.Interventions)
	}
}

func oversightMatter(id string, scope json.RawMessage, now time.Time) continuity.Matter {
	return continuity.Matter{
		ID: id, TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap,
		Status: continuity.MatterInitialReview, Priority: 5, Title: id, Scope: scope,
		CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now,
	}
}
