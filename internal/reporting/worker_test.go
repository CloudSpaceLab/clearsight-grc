package reporting

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestRunMaintainerProcessesEveryQueuedRunWithoutUsingRuntimeAttemptBudget(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	scope := testScope()
	definition := installMemoryWorkerDefinition(t, repository, scope)

	for index, id := range []string{
		"00000000-0000-7000-8000-000000000701",
		"00000000-0000-7000-8000-000000000702",
	} {
		run := ReportRun{
			ID: id, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
			DefinitionID: definition.ID, DefinitionVersion: definition.CurrentVersion,
			DefinitionCode: definition.Code, DefinitionChecksum: definition.StoredChecksum,
			ScopeKind: definition.ScopeKind, RequestedByRef: testPerformerID,
			AsOf: serviceTestNow.Add(time.Duration(index) * time.Second), Filter: cloneReportFilter(definition.Filter),
			Dataset: definition.Dataset, Format: FormatCSV, Status: RunQueued,
			CreatedAt: serviceTestNow.Add(time.Duration(index) * time.Second), ExpiresAt: serviceTestNow.Add(ReportRunRetention),
			SourceBoundary: testSourceBoundary(0),
		}
		if _, err := repository.CreateRun(ctx, scope, run); err != nil {
			t.Fatalf("create queued run %d: %v", index+1, err)
		}
	}

	service := NewService(repository, evidence.NewMemoryObjectStore(), nil)
	service.Now = func() time.Time { return serviceTestNow }
	service.WorkerID = "report-worker-test"
	maintainer := NewRunMaintainer(repository, service)
	processed, err := maintainer.Maintain(ctx, serviceTestNow, 5)
	if err != nil {
		t.Fatalf("maintain report runs: %v", err)
	}
	if processed != 2 {
		t.Fatalf("processed report runs = %d, want 2", processed)
	}
	runs, err := repository.ListRuns(ctx, scope, definition.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range runs {
		if run.Status != RunReady || run.AttemptCount != 1 || run.DataObjectKey == "" || run.ManifestObjectKey == "" {
			t.Fatalf("worker left report run incomplete or on a runtime-owned attempt budget: %#v", run)
		}
	}
}

func installMemoryWorkerDefinition(t *testing.T, repository *MemoryRepository, scope ReportScope) ReportDefinition {
	t.Helper()
	definition := activeTestDefinition(DefinitionDraft)
	definition.TenantID = scope.TenantID
	definition.LegalEntityID = scope.LegalEntityID
	definition.ReviewerID = ""
	definition.CheckerID = ""
	definition.ApprovedAt = nil
	definition.EffectiveFrom = nil
	definition.StoredChecksum = definition.Checksum()
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: 1, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind, Format: definition.Format,
		Filter: cloneReportFilter(definition.Filter), Checksum: definition.StoredChecksum, MakerID: definition.MakerID,
		CreatedAt: serviceTestNow, Decision: "PROPOSED",
	}
	definition, err := repository.CreateDefinition(context.Background(), scope, definition, revision)
	if err != nil {
		t.Fatal(err)
	}
	definition, err = repository.TransitionDefinition(context.Background(), scope, definition.ID, definition.Version,
		DefinitionPendingReview, DecisionRecord{ActorID: testMakerID, Action: DecisionSubmit, ChecksumSeen: definition.StoredChecksum, Timestamp: serviceTestNow.Add(time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	definition, err = repository.TransitionDefinition(context.Background(), scope, definition.ID, definition.Version,
		DefinitionReviewed, DecisionRecord{ActorID: testReviewerID, Action: DecisionReview, ChecksumSeen: definition.StoredChecksum, Timestamp: serviceTestNow.Add(2 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	effective := serviceTestNow.Add(-time.Hour)
	definition, err = repository.TransitionDefinition(context.Background(), scope, definition.ID, definition.Version,
		DefinitionActive, DecisionRecord{ActorID: testAuthorizerID, Action: DecisionActivate, ChecksumSeen: definition.StoredChecksum, Timestamp: serviceTestNow.Add(3 * time.Second), EffectiveFrom: &effective})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}
