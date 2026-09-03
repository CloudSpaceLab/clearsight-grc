package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestMemoryCompletedResponsesIsolateFilterAndUseStableCursor(t *testing.T) {
	base := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, nil, []RecipientCandidate{{
		PrincipalID: "principal-a", TenantID: "tenant-a", Kind: "PERSON", Active: true,
		LegalEntityIDs:   []string{"entity-a"},
		ReadableSubjects: map[string]bool{"PROGRAM:program-visible": true},
	}})
	store := NewMemoryDistributionStore(repo, nil, nil)
	store.distributions = map[string]FormDistribution{
		"distribution-a":          {ID: "distribution-a", TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectType: "PROGRAM", SubjectID: "program-visible", Title: "Vendor certification refresh"},
		"distribution-restricted": {ID: "distribution-restricted", TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectType: "PROGRAM", SubjectID: "program-restricted", Title: "Restricted response"},
		"distribution-b":          {ID: "distribution-b", TenantID: "tenant-a", LegalEntityID: "entity-b", FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectType: "VENDOR", SubjectID: "vendor-b", Title: "Other entity response"},
	}
	store.responseRevisions = map[string][]ResponseRevision{
		"distribution-a": {
			{ID: "response-draft", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a", Revision: 4, State: "DRAFT", Current: true, CreatedAt: base.Add(4 * time.Minute)},
			completedRevision("response-3", "tenant-a", "entity-a", "distribution-a", base.Add(3*time.Minute), 88, formcontract.ConcernCritical),
			completedRevision("response-2", "tenant-a", "entity-a", "distribution-a", base.Add(2*time.Minute), 88, formcontract.ConcernCritical),
			completedRevision("response-1", "tenant-a", "entity-a", "distribution-a", base.Add(time.Minute), 72, formcontract.ConcernHigh),
			completedRevision("response-low", "tenant-a", "entity-a", "distribution-a", base, 20, formcontract.ConcernLow),
		},
		"distribution-restricted": {completedRevision("response-restricted", "tenant-a", "entity-a", "distribution-restricted", base.Add(5*time.Minute), 100, formcontract.ConcernCritical)},
		"distribution-b":          {completedRevision("response-other-entity", "tenant-a", "entity-b", "distribution-b", base.Add(4*time.Minute), 99, formcontract.ConcernCritical)},
	}

	query := CompletedResponseQuery{
		TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a", CurrentOnly: true,
		Bands: []formcontract.ConcernBand{formcontract.ConcernHigh, formcontract.ConcernCritical},
		Sort:  ResponseSortConcern, Limit: 2,
	}
	first, err := store.ListCompletedResponses(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].ID != "response-3" || first.Items[1].ID != "response-2" || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}
	query.Cursor = first.NextCursor
	second, err := store.ListCompletedResponses(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "response-1" || second.NextCursor != "" {
		t.Fatalf("second page = %#v", second)
	}
	for _, page := range []CompletedResponsePage{first, second} {
		for _, item := range page.Items {
			if item.LegalEntityID != "entity-a" || item.ID == "response-low" || item.ID == "response-other-entity" || item.ID == "response-restricted" {
				t.Fatalf("query leaked or ignored filters: %#v", item)
			}
		}
	}
	if _, _, err := store.GetCompletedResponse(context.Background(), "tenant-a", "entity-a", "principal-a", "response-restricted"); err != ErrNotFound {
		t.Fatalf("restricted response detail leaked: %v", err)
	}
	workerValue, err := store.GetCompletedResponseForExecution(context.Background(), "tenant-a", "response-restricted")
	if err != nil || workerValue.ID != "response-restricted" || workerValue.LegalEntityID != "entity-a" || workerValue.Score == nil {
		t.Fatalf("exact worker response = %#v err=%v", workerValue, err)
	}
	if _, err := store.GetCompletedResponseForExecution(context.Background(), "other-tenant", "response-restricted"); err != ErrNotFound {
		t.Fatalf("exact worker response crossed tenant scope: %v", err)
	}
	if _, err := store.GetCompletedResponseForExecution(context.Background(), "tenant-a", "response-draft"); err != ErrNotFound {
		t.Fatalf("exact worker response accepted an incomplete revision: %v", err)
	}
}

func TestCompletedResponseQueryRejectsInvalidScoreBoundsAndCursor(t *testing.T) {
	minimum, maximum := 80.0, 20.0
	store := NewMemoryDistributionStore(NewMemoryRepository(nil, nil), nil, nil)
	for _, query := range []CompletedResponseQuery{
		{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a", RawMinimum: &minimum, RawMaximum: &maximum, Limit: 25},
		{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a", Sort: "UNKNOWN", Limit: 25},
		{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a", Sort: ResponseSortConcern, Cursor: "not-a-cursor", Limit: 25},
		{TenantID: "tenant-a", LegalEntityID: "entity-a", Limit: 25},
	} {
		if _, err := store.ListCompletedResponses(context.Background(), query); err == nil {
			t.Fatalf("invalid query was accepted: %#v", query)
		}
	}
}

func TestCompletedResponseQueryNormalizesSubjectType(t *testing.T) {
	query := CompletedResponseQuery{
		TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a",
		SubjectType: " Vendor ", Limit: 25,
	}
	if _, err := normalizeCompletedResponseQuery(&query); err != nil {
		t.Fatal(err)
	}
	if query.SubjectType != "VENDOR" {
		t.Fatalf("subject type = %q, want VENDOR", query.SubjectType)
	}
}

func completedRevision(id, tenantID, entityID, distributionID string, completedAt time.Time, adverse float64, band formcontract.ConcernBand) ResponseRevision {
	raw := adverse
	return ResponseRevision{
		ID: id, TenantID: tenantID, LegalEntityID: entityID, DistributionID: distributionID,
		Revision: 1, State: ResponseRevisionFinal, Current: true, CreatedAt: completedAt,
		Score: &ResponseScoreResult{Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor, RawScore: &raw, AdverseScore: &adverse, Band: band, Coverage: 1, Final: true, State: ResponseScoreFinal, CalculatedAt: completedAt},
	}
}
