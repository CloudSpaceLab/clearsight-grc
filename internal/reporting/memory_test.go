package reporting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestMemoryRepositoryEnforcesTheSameRulesAsPostgres(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	scope := testScope()
	definition := activeTestDefinition(DefinitionDraft)
	definition.ReviewerID = ""
	definition.CheckerID = ""
	definition.ApprovedAt = nil
	definition.EffectiveFrom = nil
	definition.StoredChecksum = definition.Checksum()
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: definition.CurrentVersion, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind,
		ScopeRef: definition.ScopeRef, Format: definition.Format, Filter: cloneReportFilter(definition.Filter),
		Checksum: definition.StoredChecksum, MakerID: definition.MakerID, CreatedAt: serviceTestNow,
		Decision: "PROPOSED",
	}
	if _, err := repository.CreateDefinition(ctx, scope, definition, revision); err != nil {
		t.Fatalf("create definition: %v", err)
	}

	submitted, err := repository.TransitionDefinition(ctx, scope, definition.ID, definition.Version,
		DefinitionPendingReview, DecisionRecord{
			ActorID: testMakerID, Action: DecisionSubmit, ChecksumSeen: definition.StoredChecksum,
			Timestamp: serviceTestNow.Add(time.Second),
		})
	if err != nil {
		t.Fatalf("submit definition: %v", err)
	}
	if _, err := repository.TransitionDefinition(ctx, scope, definition.ID, definition.Version,
		DefinitionReviewed, DecisionRecord{
			ActorID: testReviewerID, Action: DecisionReview, ChecksumSeen: submitted.StoredChecksum,
			Timestamp: serviceTestNow.Add(2 * time.Second),
		}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale expected version error = %v, want ErrConflict", err)
	}
	if _, err := repository.TransitionDefinition(ctx, scope, definition.ID, submitted.Version,
		DefinitionReviewed, DecisionRecord{
			ActorID: testMakerID, Action: DecisionReview, ChecksumSeen: submitted.StoredChecksum,
			Timestamp: serviceTestNow.Add(2 * time.Second),
		}); !errors.Is(err, ErrClosureBlocked) {
		t.Fatalf("maker review error = %v, want governed separation failure", err)
	}

	reviewed, err := repository.TransitionDefinition(ctx, scope, definition.ID, submitted.Version,
		DefinitionReviewed, DecisionRecord{
			ActorID: testReviewerID, Action: DecisionReview, ChecksumSeen: submitted.StoredChecksum,
			Timestamp: serviceTestNow.Add(3 * time.Second),
		})
	if err != nil {
		t.Fatalf("independent review: %v", err)
	}
	for _, authorizer := range []string{testMakerID, testReviewerID} {
		if _, err := repository.TransitionDefinition(ctx, scope, definition.ID, reviewed.Version,
			DefinitionActive, DecisionRecord{
				ActorID: authorizer, Action: DecisionActivate, ChecksumSeen: reviewed.StoredChecksum,
				Timestamp: serviceTestNow.Add(4 * time.Second),
			}); !errors.Is(err, ErrClosureBlocked) {
			t.Fatalf("authorizer %q error = %v, want governed separation failure", authorizer, err)
		}
	}
}

func TestMemoryRepositoryRefusesReadyRunWithoutArtefacts(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	scope := testScope()
	definition := activeTestDefinition(DefinitionDraft)
	definition.ReviewerID = ""
	definition.CheckerID = ""
	effective := serviceTestNow.Add(-time.Hour)
	definition.EffectiveFrom = nil
	definition.ApprovedAt = nil
	definition.StoredChecksum = definition.Checksum()
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: 1, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind, Format: definition.Format,
		Filter: cloneReportFilter(definition.Filter), Checksum: definition.StoredChecksum, MakerID: definition.MakerID,
		CreatedAt: serviceTestNow, Decision: "PROPOSED",
	}
	if _, err := repository.CreateDefinition(ctx, scope, definition, revision); err != nil {
		t.Fatalf("create active definition: %v", err)
	}
	definition, err := repository.TransitionDefinition(ctx, scope, definition.ID, definition.Version,
		DefinitionPendingReview, DecisionRecord{ActorID: testMakerID, Action: DecisionSubmit, ChecksumSeen: definition.StoredChecksum, Timestamp: serviceTestNow.Add(time.Second)})
	if err != nil {
		t.Fatalf("submit active-definition fixture: %v", err)
	}
	definition, err = repository.TransitionDefinition(ctx, scope, definition.ID, definition.Version,
		DefinitionReviewed, DecisionRecord{ActorID: testReviewerID, Action: DecisionReview, ChecksumSeen: definition.StoredChecksum, Timestamp: serviceTestNow.Add(2 * time.Second)})
	if err != nil {
		t.Fatalf("review active-definition fixture: %v", err)
	}
	definition, err = repository.TransitionDefinition(ctx, scope, definition.ID, definition.Version,
		DefinitionActive, DecisionRecord{ActorID: testAuthorizerID, Action: DecisionActivate, ChecksumSeen: definition.StoredChecksum, Timestamp: serviceTestNow.Add(3 * time.Second), EffectiveFrom: &effective})
	if err != nil {
		t.Fatalf("activate active-definition fixture: %v", err)
	}
	run := ReportRun{
		ID: "00000000-0000-7000-8000-000000000401", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		DefinitionID: definition.ID, DefinitionVersion: definition.CurrentVersion, DefinitionCode: definition.Code,
		DefinitionChecksum: definition.StoredChecksum, ScopeKind: definition.ScopeKind, RequestedByRef: testPerformerID,
		AsOf: serviceTestNow, Filter: cloneReportFilter(definition.Filter), Dataset: definition.Dataset,
		Format: FormatCSV, Status: RunQueued, CreatedAt: serviceTestNow, ExpiresAt: serviceTestNow.Add(ReportRunRetention),
		SourceBoundary: testSourceBoundary(0),
	}
	queued, err := repository.CreateRun(ctx, scope, run)
	if err != nil {
		t.Fatalf("create queued run: %v", err)
	}
	claimed, err := repository.ClaimQueuedRuns(ctx, scope, "memory-worker", 1)
	if err != nil || len(claimed) != 1 || claimed[0].ID != queued.ID {
		t.Fatalf("claim queued run: runs=%#v err=%v", claimed, err)
	}
	withoutArtefacts := claimed[0]
	withoutArtefacts.Status = RunReady
	withoutArtefacts.CompletedAt = timePtr(serviceTestNow.Add(time.Second))
	if _, err := repository.CompleteRun(ctx, scope, withoutArtefacts); !errors.Is(err, ErrInvalid) {
		t.Fatalf("READY run without artefacts error = %v, want ErrInvalid", err)
	}
}

func TestMemoryRepositoryTreatsSameTenantCrossEntityIDsAsNotFound(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	scope := testScope()
	otherScope := ReportScope{TenantID: testTenantID, LegalEntityID: testEntityB}
	definition := activeTestDefinition(DefinitionDraft)
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
	if _, err := repository.CreateDefinition(ctx, scope, definition, revision); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.GetDefinition(ctx, otherScope, definition.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity definition read error = %v, want ErrNotFound", err)
	}
	if _, err := repository.ListDefinitionHistory(ctx, otherScope, definition.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity history error = %v, want ErrNotFound", err)
	}
	definitions, err := repository.ListDefinitions(ctx, otherScope, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 0 {
		t.Fatalf("cross-entity definition list = %#v", definitions)
	}
}

func TestInstallDemoProvidesFourGovernanceExamplesAndABoundStop(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	service := NewService(repository, evidence.NewMemoryObjectStore(), nil)
	if err := InstallDemo(ctx, service); err != nil {
		t.Fatalf("install reporting demo: %v", err)
	}
	if err := InstallDemo(ctx, service); err != nil {
		t.Fatalf("repeat reporting demo install: %v", err)
	}

	scope := ReportScope{TenantID: DemoTenant, LegalEntityID: DemoLegalEntity}
	definitions, err := repository.ListDefinitions(ctx, scope, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 4 {
		t.Fatalf("demo definitions = %d, want 4", len(definitions))
	}
	want := map[string]struct {
		dataset   ReportDataset
		scopeKind ReportScopeKind
		scopeRef  string
		status    DefinitionStatus
	}{
		"ROPA-OPEN-EXCEPTIONS":        {DatasetProcessingActivityExceptions, ScopeLegalEntity, "", DefinitionActive},
		"ROPA-CROSS-BORDER-TRANSFERS": {DatasetProcessingActivities, ScopeProgram, DemoProgramRef, DefinitionActive},
		"ISSUES-OVERDUE-OBLIGATIONS":  {DatasetMatterExceptions, ScopeMatter, DemoMatterRef, DefinitionPendingReview},
		"PROGRAM-HEALTH":              {DatasetPrograms, ScopeLegalEntity, "", DefinitionReviewed},
	}
	for _, definition := range definitions {
		expected, ok := want[definition.Code]
		if !ok {
			t.Errorf("unexpected demo definition %q", definition.Code)
			continue
		}
		if definition.Dataset != expected.dataset || definition.ScopeKind != expected.scopeKind ||
			definition.ScopeRef != expected.scopeRef || definition.Status != expected.status {
			t.Errorf("demo definition %q = dataset %q scope %q ref %q status %q", definition.Code,
				definition.Dataset, definition.ScopeKind, definition.ScopeRef, definition.Status)
		}
		if !strings.Contains(strings.ToLower(definition.Description), "sample data") {
			t.Errorf("demo definition %q is not clearly labelled as sample data", definition.Code)
		}
		delete(want, definition.Code)
	}
	if len(want) != 0 {
		t.Fatalf("missing demo definitions: %#v", want)
	}

	runs, err := repository.ListRuns(ctx, scope, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("demo report runs = %d, want 1", len(runs))
	}
	run := runs[0]
	if run.Status != RunFailed || run.FailureCode != FailureRowLimitExceeded || run.RowCount != 0 ||
		run.SourceBoundary.Population != MaxReportRunRows+1 ||
		run.DataObjectKey != "" || run.DataSHA256 != "" || run.ManifestObjectKey != "" || run.ManifestSHA256 != "" {
		t.Fatalf("bounded-stop demo run = %#v", run)
	}
}
