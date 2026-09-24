package reporting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

const (
	testTenantID     = "00000000-0000-7000-8000-000000000101"
	testEntityA      = "00000000-0000-7000-8000-000000000102"
	testEntityB      = "00000000-0000-7000-8000-000000000103"
	testMakerID      = "00000000-0000-7000-8000-000000000104"
	testReviewerID   = "00000000-0000-7000-8000-000000000105"
	testAuthorizerID = "00000000-0000-7000-8000-000000000106"
	testPerformerID  = "00000000-0000-7000-8000-000000000107"
	testOtherActorID = "00000000-0000-7000-8000-000000000108"
	testDefinitionID = "00000000-0000-7000-8000-000000000109"
	testOtherDefID   = "00000000-0000-7000-8000-000000000110"
)

var serviceTestNow = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

func TestProposeDefinitionIgnoresActorFieldsFromTheRequestBody(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	authorityChecker.expected[authority.ResponsibilityProposer] = testMakerID

	definition, err := service.Propose(reportActorContext(testMakerID), ProposeInput{
		Code:        "ROPA-EXCEPTIONS",
		Name:        "Open processing exceptions",
		Description: "Exceptions that still need evidence or ownership.",
		Dataset:     DatasetProcessingActivityExceptions,
		ScopeKind:   ScopeLegalEntity,
		Format:      FormatCSV,
		Filter:      emptyReportFilter(),
		MakerID:     testAuthorizerID, // A request-body actor must never be trusted.
	})
	if err != nil {
		t.Fatalf("propose definition: %v", err)
	}
	if definition.MakerID != testMakerID {
		t.Fatalf("maker = %q, want verified actor %q", definition.MakerID, testMakerID)
	}
	if definition.Status != DefinitionDraft || definition.StoredChecksum != definition.Checksum() {
		t.Fatalf("proposal is not a checksum-bound draft: %#v", definition)
	}
	if len(repository.history[definition.ID]) != 1 || repository.history[definition.ID][0].MakerID != testMakerID {
		t.Fatal("proposal did not create the maker-bound revision in the same repository command")
	}
}

func TestSubmitRequiresADraftDefinition(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionActive)
	repository.definitions[definition.ID] = definition
	authorityChecker.expected[authority.ResponsibilityProposer] = testMakerID

	_, err := service.Submit(reportActorContext(testMakerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("submit non-draft error = %v, want ErrInvalid", err)
	}
}

func TestSubmitRejectsAMissingAuthorityRoute(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionDraft)
	repository.definitions[definition.ID] = definition
	authorityChecker.fail[authority.ResponsibilityProposer] = errors.New("no current route")

	_, err := service.Submit(reportActorContext(testMakerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
	})
	if err == nil || !strings.Contains(err.Error(), "no current route") {
		t.Fatalf("submit with unavailable authority error = %v", err)
	}
	if len(repository.transitions) != 0 {
		t.Fatal("submit wrote a transition after authority resolution failed")
	}
}

func TestReviewRefusesWhenTheReviewerIsTheMaker(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionPendingReview)
	repository.definitions[definition.ID] = definition
	authorityChecker.expected[authority.ResponsibilityReviewer] = testMakerID

	_, err := service.Review(reportActorContext(testMakerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
		Note: "Reviewed by the maker.",
	})
	if !errors.Is(err, ErrClosureBlocked) {
		t.Fatalf("maker review error = %v, want governed separation failure", err)
	}
	if len(repository.transitions) != 0 {
		t.Fatal("maker review reached the repository")
	}
}

func TestReviewRefusesWhenTheDefinitionChangedSinceTheReviewerSawIt(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionPendingReview)
	repository.definitions[definition.ID] = definition
	authorityChecker.expected[authority.ResponsibilityReviewer] = testReviewerID

	_, err := service.Review(reportActorContext(testReviewerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: strings.Repeat("0", 64),
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale review checksum error = %v, want ErrConflict", err)
	}
	if len(repository.transitions) != 0 {
		t.Fatal("stale review checksum reached the repository")
	}
}

func TestReviewRecordsAReviewerDistinctFromBothMakerAndAuthorizer(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	authorityChecker.expected[authority.ResponsibilityProposer] = testMakerID
	authorityChecker.expected[authority.ResponsibilityReviewer] = testReviewerID
	authorityChecker.expected[authority.ResponsibilityAuthorizer] = testAuthorizerID

	definition := proposeThroughReview(t, service)
	activated, err := service.Activate(reportActorContext(testAuthorizerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
		EffectiveFrom: timePtr(serviceTestNow),
	})
	if err != nil {
		t.Fatalf("activate reviewed definition: %v", err)
	}
	if activated.MakerID == activated.ReviewerID || activated.ReviewerID == activated.CheckerID || activated.MakerID == activated.CheckerID {
		t.Fatalf("maker, reviewer, and authorizer were not distinct: %q, %q, %q", activated.MakerID, activated.ReviewerID, activated.CheckerID)
	}
	if len(repository.transitions) != 3 {
		t.Fatalf("recorded %d lifecycle decisions, want submit/review/activate", len(repository.transitions))
	}
	for index, decision := range repository.transitions {
		if decision.ActorID == "" || decision.ChecksumSeen != activated.StoredChecksum {
			t.Fatalf("decision %d is not bound to the verified actor and reviewed checksum: %#v", index, decision)
		}
	}
}

func TestActivateRefusesUnlessAReviewWasRecorded(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionPendingReview)
	definition.ReviewerID = ""
	definition.StoredChecksum = definition.Checksum()
	repository.definitions[definition.ID] = definition
	authorityChecker.expected[authority.ResponsibilityAuthorizer] = testAuthorizerID

	_, err := service.Activate(reportActorContext(testAuthorizerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
		EffectiveFrom: timePtr(serviceTestNow),
	})
	if !errors.Is(err, ErrClosureBlocked) {
		t.Fatalf("activation without review error = %v, want governed precondition failure", err)
	}
}

func TestActivateRefusesWhenTheAuthorizerIsTheMakerOrTheReviewer(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		actor    string
		reviewer string
	}{
		{name: "maker", actor: testMakerID, reviewer: testReviewerID},
		{name: "reviewer", actor: testReviewerID, reviewer: testReviewerID},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			service, repository, _, authorityChecker := newReportingServiceTest()
			definition := activeTestDefinition(DefinitionReviewed)
			definition.ReviewerID = testCase.reviewer
			definition.StoredChecksum = definition.Checksum()
			repository.definitions[definition.ID] = definition
			authorityChecker.expected[authority.ResponsibilityAuthorizer] = testCase.actor

			_, err := service.Activate(reportActorContext(testCase.actor), DefinitionTransitionInput{
				Scope: testScope(), DefinitionID: definition.ID,
				ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
				EffectiveFrom: timePtr(serviceTestNow),
			})
			if !errors.Is(err, ErrClosureBlocked) {
				t.Fatalf("activation error = %v, want governed separation failure", err)
			}
		})
	}
}

func TestActivateMarksAFutureEffectiveDateAsNotYetCurrent(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionReviewed)
	definition.ReviewerID = testReviewerID
	definition.StoredChecksum = definition.Checksum()
	repository.definitions[definition.ID] = definition
	authorityChecker.expected[authority.ResponsibilityAuthorizer] = testAuthorizerID
	future := serviceTestNow.Add(24 * time.Hour)

	activated, err := service.Activate(reportActorContext(testAuthorizerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
		EffectiveFrom: &future,
	})
	if err != nil {
		t.Fatalf("activate future-dated definition: %v", err)
	}
	if activated.Status != DefinitionActive || activated.Effective {
		t.Fatalf("future activation = status %q effective %v, want ACTIVE and not yet current", activated.Status, activated.Effective)
	}
}

func TestRetireIsReversibleByANewRevision(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionActive)
	definition.ReviewerID = testReviewerID
	definition.StoredChecksum = definition.Checksum()
	repository.definitions[definition.ID] = definition
	repository.history[definition.ID] = []ReportDefinitionRevision{{DefinitionID: definition.ID, Version: 1, Checksum: definition.StoredChecksum, MakerID: testMakerID}}
	authorityChecker.expected[authority.ResponsibilityAuthorizer] = testAuthorizerID
	authorityChecker.expected[authority.ResponsibilityProposer] = testMakerID

	retired, err := service.Retire(reportActorContext(testAuthorizerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
		Note: "Superseded by a narrower definition.",
	})
	if err != nil {
		t.Fatalf("retire definition: %v", err)
	}
	replacement, err := service.Propose(reportActorContext(testMakerID), ProposeInput{
		Code: definition.Code, Name: "Replacement exception report", Dataset: definition.Dataset,
		ScopeKind: definition.ScopeKind, ScopeRef: definition.ScopeRef, Format: definition.Format,
		Filter: emptyReportFilter(), MakerID: testAuthorizerID,
	})
	if err != nil {
		t.Fatalf("propose replacement after retirement: %v", err)
	}
	if replacement.ID == retired.ID {
		t.Fatal("replacement reused the retired aggregate instead of preserving it")
	}
	if repository.definitions[retired.ID].Status != DefinitionRetired || len(repository.history[retired.ID]) == 0 {
		t.Fatal("retired version was not preserved")
	}
}

func TestValidateTransitionForWriteIsTheOnlyTransitionGate(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("service.go"))
	if err != nil {
		t.Fatalf("read service.go: %v", err)
	}
	if count := strings.Count(string(body), "switch current.Status"); count != 1 {
		t.Fatalf("service.go contains %d switches over current.Status, want only ValidateTransitionForWrite", count)
	}
	for _, forbidden := range []string{"switch definition.Status", "switch next.Status", "switch fromStatus"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("service.go contains a second definition transition gate: %s", forbidden)
		}
	}
}

func TestCreateRunRejectsAnUnapprovedDefinition(t *testing.T) {
	for _, status := range []DefinitionStatus{DefinitionDraft, DefinitionPendingReview, DefinitionReviewed} {
		t.Run(string(status), func(t *testing.T) {
			service, repository, _, authorityChecker := newReportingServiceTest()
			definition := activeTestDefinition(status)
			repository.definitions[definition.ID] = definition
			authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
			_, err := service.CreateRun(reportActorContext(testPerformerID), CreateRunInput{
				Scope: testScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion,
			})
			if !errors.Is(err, ErrClosureBlocked) {
				t.Fatalf("run of %s error = %v, want governed precondition failure", status, err)
			}
		})
	}
}

func TestCreateRunRejectsARetiredDefinition(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := activeTestDefinition(DefinitionRetired)
	definition.RetiredAt = timePtr(serviceTestNow)
	definition.StoredChecksum = definition.Checksum()
	repository.definitions[definition.ID] = definition
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	_, err := service.CreateRun(reportActorContext(testPerformerID), CreateRunInput{
		Scope: testScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion,
	})
	if !errors.Is(err, ErrClosureBlocked) {
		t.Fatalf("retired run error = %v, want governed precondition failure", err)
	}
}

func TestCreateRunQueuesWithoutRenderingInline(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	repository.rows[definition.ID] = []ReportRow{{ID: "row-1", Values: map[string]any{"name": "Customer account opening"}}}

	run, err := service.CreateRun(reportActorContext(testPerformerID), CreateRunInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedDefinitionVersion: definition.CurrentVersion, RequestedByRef: testAuthorizerID,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if run.Status != RunQueued || repository.pageCalls != 0 || len(objects.puts) != 0 {
		t.Fatalf("CreateRun rendered inline: status=%q pages=%d puts=%d", run.Status, repository.pageCalls, len(objects.puts))
	}
	if run.RequestedByRef != testPerformerID {
		t.Fatalf("requested_by = %q, want verified actor %q", run.RequestedByRef, testPerformerID)
	}
	if run.DefinitionVersion != definition.CurrentVersion || run.DefinitionChecksum != definition.StoredChecksum {
		t.Fatal("run did not pin the reviewed definition version and checksum")
	}
}

func TestServiceCompleteRunRefusesWithoutArtefacts(t *testing.T) {
	service, repository, _, _ := newReportingServiceTest()
	now := serviceTestNow
	run := ReportRun{
		ID: "00000000-0000-7000-8000-000000000301", TenantID: testTenantID, LegalEntityID: testEntityA,
		DefinitionID: testDefinitionID, DefinitionVersion: 1, DefinitionCode: "ROPA-EXCEPTIONS",
		DefinitionChecksum: strings.Repeat("a", 64), ScopeKind: ScopeLegalEntity,
		RequestedByRef: testPerformerID, AsOf: now, Filter: emptyReportFilter(),
		Dataset: DatasetProcessingActivityExceptions, Format: FormatCSV, Status: RunReady,
		RowCount: 0, CompletedAt: &now, CreatedAt: now, ExpiresAt: now.Add(ReportRunRetention),
		SourceBoundary: testSourceBoundary(0),
	}

	if _, err := service.completeRun(context.Background(), testScope(), run); !errors.Is(err, ErrInvalid) {
		t.Fatalf("service completion without artefacts error = %v, want ErrInvalid", err)
	}
	if repository.completeCalls != 0 {
		t.Fatal("service completion without artefacts reached the repository")
	}
}

func TestCreateRunStopsAtTheRowBound(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	repository.rows[definition.ID] = make([]ReportRow, MaxReportRunRows+1)
	for index := range repository.rows[definition.ID] {
		repository.rows[definition.ID][index] = ReportRow{ID: strconv.Itoa(index), Values: map[string]any{"name": "row"}}
	}
	repository.boundary = testSourceBoundary(len(repository.rows[definition.ID]))
	run := createQueuedRun(t, service, definition)

	if _, err := service.ExecuteRun(context.Background(), run); !errors.Is(err, ErrReportTooLarge) {
		t.Fatalf("10,001-row execution error = %v, want ErrReportTooLarge", err)
	}
	failed := repository.runs[run.ID]
	if failed.Status != RunFailed || failed.FailureCode != FailureRowLimitExceeded || failed.RowCount != 0 {
		t.Fatalf("bounded run receipt = %#v", failed)
	}
	if len(objects.puts) != 0 || len(objects.objects) != 0 {
		t.Fatalf("row-bound failure left an artefact: puts=%v objects=%v", objects.puts, objectKeys(objects))
	}
}

func TestCreateRunStopsAtTheByteBound(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	repository.rows[definition.ID] = []ReportRow{{ID: "oversized", Values: map[string]any{"payload": strings.Repeat("a", int(MaxReportRunBytes))}}}
	repository.boundary = testSourceBoundary(1)
	run := createQueuedRun(t, service, definition)

	if _, err := service.ExecuteRun(context.Background(), run); !errors.Is(err, ErrReportTooLarge) {
		t.Fatalf("32-MiB execution error = %v, want ErrReportTooLarge", err)
	}
	failed := repository.runs[run.ID]
	if failed.FailureCode != FailureByteLimitExceeded || failed.RowCount != 0 {
		t.Fatalf("byte-bound failure receipt = %#v", failed)
	}
	if len(objects.puts) != 0 || len(objects.objects) != 0 {
		t.Fatalf("byte-bound failure left an artefact: puts=%v objects=%v", objects.puts, objectKeys(objects))
	}
}

func TestCreateRunRejectsACrossEntityDefinition(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	definition.LegalEntityID = testEntityB
	definition.StoredChecksum = definition.Checksum()
	repository.definitions[definition.ID] = definition
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	_, err := service.CreateRun(reportActorContext(testPerformerID), CreateRunInput{
		Scope: testScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity run error = %v, want ErrNotFound", err)
	}
}

func TestCreateRunPersistsTheSourceBoundaryBeforeGeneration(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	boundary := testSourceBoundary(17)
	boundary.SourceHighWater["processing_activities"] = serviceTestNow.Add(-time.Hour)
	repository.boundary = boundary
	run := createQueuedRun(t, service, definition)
	repository.pageErr = errors.New("source unavailable during generation")

	if _, err := service.ExecuteRun(context.Background(), run); err == nil {
		t.Fatal("generation unexpectedly succeeded without its source")
	}
	persisted := repository.runs[run.ID]
	if persisted.SourceBoundary.ProjectionVersion != boundary.ProjectionVersion || persisted.SourceBoundary.Population != 17 {
		t.Fatalf("source boundary was not retained on failed run: %#v", persisted.SourceBoundary)
	}
	if persisted.SourceBoundary.SourceHighWater["processing_activities"].IsZero() {
		t.Fatal("source high-water was not retained on failed run")
	}
}

func TestRunManifestRecordsTheSourceBoundaryAndPopulation(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	repository.rows[definition.ID] = []ReportRow{{ID: "row-1", Values: map[string]any{"name": "Customer account opening", "status": "OPEN"}}}
	repository.boundary = testSourceBoundary(1)
	run := createQueuedRun(t, service, definition)
	if _, err := service.ExecuteRun(context.Background(), run); err != nil {
		t.Fatalf("execute run: %v", err)
	}
	ready := repository.runs[run.ID]
	reader, err := objects.Open(context.Background(), ready.ManifestObjectKey)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var manifest Manifest
	if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.DefinitionVersion != definition.CurrentVersion || manifest.DefinitionChecksum != definition.StoredChecksum {
		t.Fatalf("manifest definition binding = version %d checksum %q", manifest.DefinitionVersion, manifest.DefinitionChecksum)
	}
	if !manifest.Source.PopulationComplete || manifest.Source.Population != 1 || manifest.Source.SourceHighWater["processing_activities"].IsZero() {
		t.Fatalf("manifest source boundary = %#v", manifest.Source)
	}
	if manifest.RowCount != 1 || !manifest.RetentionUntil.Equal(ready.ExpiresAt) || manifest.DataSHA256 != ready.DataSHA256 {
		t.Fatalf("manifest receipt = %#v", manifest)
	}
}

func TestRunArtefactKeyIsScopedAndOpaque(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	repository.rows[definition.ID] = []ReportRow{{ID: "row-1", Values: map[string]any{"name": "Sensitive Customer Account", "owner": "Avery Owner"}}}
	repository.boundary = testSourceBoundary(1)
	run := createQueuedRun(t, service, definition)
	if _, err := service.ExecuteRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	key := repository.runs[run.ID].DataObjectKey
	for _, required := range []string{testTenantID, testEntityA, run.ID} {
		if !strings.Contains(key, required) {
			t.Fatalf("artefact key %q is not scoped by %q", key, required)
		}
	}
	for _, forbidden := range []string{"Sensitive", "Customer", "Avery", "owner"} {
		if strings.Contains(strings.ToLower(key), strings.ToLower(forbidden)) {
			t.Fatalf("artefact key leaked row content %q: %q", forbidden, key)
		}
	}
	if _, err := objects.Open(context.Background(), key); err != nil {
		t.Fatalf("scoped artefact key was not stored: %v", err)
	}
}

func TestExecuteRunUsesTheDurableAttemptBudget(t *testing.T) {
	service, repository, _, authorityChecker := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	run := createQueuedRun(t, service, definition)
	repository.claimAttemptOverride = MaxReportRunTries + 1

	if _, err := service.ExecuteRun(context.Background(), run); !errors.Is(err, ErrConflict) {
		t.Fatalf("exhausted run error = %v, want conflict", err)
	}
	if got := repository.runs[run.ID].FailureCode; got != FailureRetryBudgetExhausted {
		t.Fatalf("failure code = %q, want %q", got, FailureRetryBudgetExhausted)
	}
}

func TestOpenRunAuthorisesOnEveryDownload(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	run := installReadyRun(t, repository, objects)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID

	first, reader, err := service.Open(reportActorContext(testPerformerID), testScope(), run.ID)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	_, _ = io.ReadAll(reader)
	_ = reader.Close()
	authorityChecker.fail[authority.ResponsibilityPerformer] = errors.New("authority route unavailable")
	_, _, err = service.Open(reportActorContext(testPerformerID), testScope(), run.ID)
	if err == nil {
		t.Fatal("second download succeeded after current authority became unavailable")
	}
	if first.ID != run.ID || repository.downloads != 1 {
		t.Fatalf("download receipt count = %d, want only the authorized first download", repository.downloads)
	}
}

func TestOpenRunRejectsAnExpiredRun(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	run := installReadyRun(t, repository, objects)
	run.ExpiresAt = serviceTestNow
	repository.runs[run.ID] = run
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	_, _, err := service.Open(reportActorContext(testPerformerID), testScope(), run.ID)
	if !errors.Is(err, ErrReportExpired) {
		t.Fatalf("expired open error = %v, want ErrReportExpired", err)
	}
	if repository.downloads != 0 {
		t.Fatal("expired run recorded a download")
	}
}

func TestOpenRunRejectsARunFromAnotherLegalEntity(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	run := installReadyRun(t, repository, objects)
	run.LegalEntityID = testEntityB
	repository.runs[run.ID] = run
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID

	_, _, err := service.Open(reportActorContext(testPerformerID), testScope(), run.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("same-tenant cross-entity open error = %v, want ErrNotFound", err)
	}
	if repository.downloads != 0 {
		t.Fatal("cross-entity run recorded a download")
	}
}

func TestOpenRunVerifiesThePersistedDigestBeforeReturningBytes(t *testing.T) {
	service, repository, objects, authorityChecker := newReportingServiceTest()
	run := installReadyRun(t, repository, objects)
	authorityChecker.expected[authority.ResponsibilityPerformer] = testPerformerID
	corrupt := repository.runs[run.ID]
	corrupt.DataSHA256 = strings.Repeat("0", 64)
	repository.runs[run.ID] = corrupt

	if _, _, err := service.Open(reportActorContext(testPerformerID), testScope(), run.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("digest mismatch error = %v, want ErrNotFound", err)
	}
	if repository.downloads != 0 {
		t.Fatal("corrupt artefact recorded a successful download")
	}
}

func TestOpenRunRejectsMissingVerifiedIdentityBeforeTheRepository(t *testing.T) {
	service, repository, objects, _ := newReportingServiceTest()
	run := installReadyRun(t, repository, objects)
	_, _, err := service.Open(context.Background(), testScope(), run.ID)
	if !errors.Is(err, identity.ErrMissingIdentity) {
		t.Fatalf("missing identity error = %v, want identity.ErrMissingIdentity", err)
	}
	if repository.getRunCalls != 0 {
		t.Fatal("unidentified caller reached the report repository")
	}
}

func proposeThroughReview(t *testing.T, service *Service) ReportDefinition {
	t.Helper()
	definition, err := service.Propose(reportActorContext(testMakerID), ProposeInput{
		Code: "ROPA-EXCEPTIONS", Name: "Open processing exceptions",
		Dataset: DatasetProcessingActivityExceptions, ScopeKind: ScopeLegalEntity,
		Format: FormatCSV, Filter: emptyReportFilter(), MakerID: testAuthorizerID,
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	definition, err = service.Submit(reportActorContext(testMakerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	definition, err = service.Review(reportActorContext(testReviewerID), DefinitionTransitionInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum,
		Note: "Scope, filter, and evidence need checked.",
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	return definition
}

func newReportingServiceTest() (*Service, *serviceTestRepository, *serviceTestObjectStore, *serviceTestAuthority) {
	repository := newServiceTestRepository()
	objects := newServiceTestObjectStore()
	authorityChecker := &serviceTestAuthority{
		expected: make(map[authority.Responsibility]string),
		fail:     make(map[authority.Responsibility]error),
	}
	service := NewService(repository, objects, authorityChecker)
	service.Now = func() time.Time { return serviceTestNow }
	service.WorkerID = "report-worker-1"
	return service, repository, objects, authorityChecker
}

func reportActorContext(principalID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{
		TenantID: testTenantID, LegalEntityID: testEntityA, PrincipalID: principalID,
		Kind: "PERSON", IssuedAt: serviceTestNow.Add(-time.Hour), ExpiresAt: serviceTestNow.Add(time.Hour),
	})
}

func testScope() ReportScope { return ReportScope{TenantID: testTenantID, LegalEntityID: testEntityA} }

func activeTestDefinition(status DefinitionStatus) ReportDefinition {
	effective := serviceTestNow.Add(-time.Hour)
	definition := ReportDefinition{
		ID: testDefinitionID, TenantID: testTenantID, LegalEntityID: testEntityA,
		Code: "ROPA-EXCEPTIONS", Name: "Open processing exceptions",
		Description: "Open exceptions requiring action.", Dataset: DatasetProcessingActivityExceptions,
		ScopeKind: ScopeLegalEntity, Format: FormatCSV, Filter: emptyReportFilter(),
		Status: status, CurrentVersion: 1, MakerID: testMakerID,
		ReviewerID: testReviewerID, CheckerID: testAuthorizerID,
		EffectiveFrom: &effective, ApprovedAt: &effective, CreatedAt: serviceTestNow.Add(-24 * time.Hour),
		UpdatedAt: serviceTestNow.Add(-time.Hour), Version: 1,
	}
	definition.StoredChecksum = definition.Checksum()
	return definition
}

func installActiveDefinition(repository *serviceTestRepository) ReportDefinition {
	definition := activeTestDefinition(DefinitionActive)
	definition.Effective = true
	repository.definitions[definition.ID] = definition
	repository.history[definition.ID] = []ReportDefinitionRevision{{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: 1, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind, Format: definition.Format,
		Filter: cloneFilter(definition.Filter), Checksum: definition.StoredChecksum, MakerID: definition.MakerID,
		Decision: "APPROVED", ReviewedBy: testReviewerID, ApprovedBy: testAuthorizerID,
	}}
	repository.boundary = testSourceBoundary(0)
	return definition
}

func createQueuedRun(t *testing.T, service *Service, definition ReportDefinition) ReportRun {
	t.Helper()
	run, err := service.CreateRun(reportActorContext(testPerformerID), CreateRunInput{
		Scope: testScope(), DefinitionID: definition.ID,
		ExpectedDefinitionVersion: definition.CurrentVersion, RequestedByRef: testAuthorizerID,
	})
	if err != nil {
		t.Fatalf("create queued run: %v", err)
	}
	return run
}

func testSourceBoundary(population int) SourceBoundary {
	highWater := serviceTestNow.Add(-time.Minute)
	return SourceBoundary{
		CapturedAt: serviceTestNow.Add(-time.Second), ProjectionVersion: "ropa-register.v1",
		SourceHighWater: map[string]time.Time{"processing_activities": highWater},
		Population:      population, PopulationComplete: true,
	}
}

func installReadyRun(t *testing.T, repository *serviceTestRepository, objects *serviceTestObjectStore) ReportRun {
	t.Helper()
	run := ReportRun{
		ID: "00000000-0000-7000-8000-000000000201", TenantID: testTenantID, LegalEntityID: testEntityA,
		DefinitionID: testDefinitionID, DefinitionVersion: 1, DefinitionCode: "ROPA-EXCEPTIONS",
		DefinitionChecksum: strings.Repeat("a", 64), ScopeKind: ScopeLegalEntity,
		RequestedByRef: testPerformerID, AsOf: serviceTestNow, Dataset: DatasetProcessingActivityExceptions,
		Format: FormatCSV, Status: RunReady, RowCount: 1, SourceBoundary: testSourceBoundary(1),
		CreatedAt: serviceTestNow, ExpiresAt: serviceTestNow.Add(ReportRunRetention),
	}
	data := []byte("id,name\nrow-1,Customer account opening\n")
	digest := sha256.Sum256(data)
	run.DataObjectKey = "reports/" + testTenantID + "/" + testEntityA + "/" + run.ID + "/report.csv"
	run.DataSHA256 = hex.EncodeToString(digest[:])
	run.ManifestObjectKey = "reports/" + testTenantID + "/" + testEntityA + "/" + run.ID + "/manifest.json"
	run.ManifestSHA256 = strings.Repeat("b", 64)
	completed := serviceTestNow
	run.CompletedAt = &completed
	repository.runs[run.ID] = run
	if _, err := objects.Put(context.Background(), run.DataObjectKey, bytes.NewReader(data), MaxReportRunBytes); err != nil {
		t.Fatal(err)
	}
	return run
}

func emptyReportFilter() *ReportFilterExpression {
	return &ReportFilterExpression{Kind: "group", Operator: "and"}
}

func timePtr(value time.Time) *time.Time { return &value }

type serviceTestAuthority struct {
	expected map[authority.Responsibility]string
	fail     map[authority.Responsibility]error
	inputs   []authority.ResolveInput
}

func (a *serviceTestAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	a.inputs = append(a.inputs, input)
	if err := a.fail[input.Responsibility]; err != nil {
		return authority.Resolution{}, err
	}
	principalID := a.expected[input.Responsibility]
	if principalID == "" {
		return authority.Resolution{}, fmt.Errorf("no current route for %s", input.Responsibility)
	}
	return authority.Resolution{Principal: authority.Principal{ID: principalID}, PolicyVersion: "authority.v1"}, nil
}

type serviceTestRepository struct {
	definitions          map[string]ReportDefinition
	history              map[string][]ReportDefinitionRevision
	runs                 map[string]ReportRun
	rows                 map[string][]ReportRow
	boundary             SourceBoundary
	transitions          []DecisionRecord
	downloads            int
	pageCalls            int
	getRunCalls          int
	pageErr              error
	failRunErr           error
	claimAttemptOverride int
	completeCalls        int
}

func newServiceTestRepository() *serviceTestRepository {
	return &serviceTestRepository{
		definitions: make(map[string]ReportDefinition),
		history:     make(map[string][]ReportDefinitionRevision),
		runs:        make(map[string]ReportRun),
		rows:        make(map[string][]ReportRow),
		boundary:    testSourceBoundary(0),
	}
}

func (r *serviceTestRepository) CreateDefinition(_ context.Context, scope ReportScope, definition ReportDefinition, revision ReportDefinitionRevision) (ReportDefinition, error) {
	if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return ReportDefinition{}, ErrNotFound
	}
	r.definitions[definition.ID] = definition
	r.history[definition.ID] = append(r.history[definition.ID], revision)
	return definition, nil
}

func (r *serviceTestRepository) GetDefinition(_ context.Context, scope ReportScope, id string) (ReportDefinition, error) {
	definition, ok := r.definitions[id]
	if !ok || definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return ReportDefinition{}, ErrNotFound
	}
	return definition, nil
}

func (r *serviceTestRepository) GetDefinitionByCode(_ context.Context, scope ReportScope, code string) (ReportDefinition, error) {
	ids := make([]string, 0, len(r.definitions))
	for id := range r.definitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		definition := r.definitions[id]
		if definition.TenantID == scope.TenantID && definition.LegalEntityID == scope.LegalEntityID && definition.Code == code {
			return definition, nil
		}
	}
	return ReportDefinition{}, ErrNotFound
}

func (r *serviceTestRepository) ListDefinitions(context.Context, ReportScope, bool) ([]ReportDefinition, error) {
	return nil, nil
}

func (r *serviceTestRepository) ListDefinitionHistory(_ context.Context, scope ReportScope, id string) ([]ReportDefinitionRevision, error) {
	if _, err := r.GetDefinition(context.Background(), scope, id); err != nil {
		return nil, err
	}
	return append([]ReportDefinitionRevision(nil), r.history[id]...), nil
}

func (r *serviceTestRepository) TransitionDefinition(_ context.Context, scope ReportScope, id string, expectedVersion int64, next DefinitionStatus, decision DecisionRecord) (ReportDefinition, error) {
	definition, err := r.GetDefinition(context.Background(), scope, id)
	if err != nil {
		return ReportDefinition{}, err
	}
	if definition.Version != expectedVersion {
		return ReportDefinition{}, ErrConflict
	}
	if err := ValidateTransitionForWrite(definition, next); err != nil {
		return ReportDefinition{}, err
	}
	now := decision.Timestamp
	definition.Status = next
	definition.Version++
	definition.UpdatedAt = now
	switch decision.Action {
	case DecisionSubmit:
		definition.SubmittedAt = &now
	case DecisionReview:
		definition.ReviewerID = decision.ActorID
		definition.ReviewerNote = decision.Note
	case DecisionActivate:
		definition.CheckerID = decision.ActorID
		definition.ApprovedAt = &now
		if decision.EffectiveFrom != nil {
			definition.EffectiveFrom = decision.EffectiveFrom
		}
	case DecisionReject, DecisionRetire:
		definition.RetiredAt = &now
	}
	definition.StoredChecksum = definition.Checksum()
	r.definitions[id] = definition
	r.transitions = append(r.transitions, decision)
	if revisions := r.history[id]; len(revisions) > 0 {
		index := len(revisions) - 1
		revisions[index].DecisionNote = decision.Note
		switch decision.Action {
		case DecisionReview:
			revisions[index].Decision = "REVIEWED"
			revisions[index].ReviewedBy = decision.ActorID
			revisions[index].ReviewedAt = &now
		case DecisionActivate:
			revisions[index].Decision = "APPROVED"
			revisions[index].ApprovedBy = decision.ActorID
			revisions[index].ApprovedAt = &now
		case DecisionReject:
			revisions[index].Decision = "REJECTED"
		case DecisionRetire:
			revisions[index].Decision = "RETIRED"
		}
	}
	return definition, nil
}

func (r *serviceTestRepository) CreateRun(_ context.Context, scope ReportScope, run ReportRun) (ReportRun, error) {
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	r.runs[run.ID] = run
	return run, nil
}

func (r *serviceTestRepository) GetRun(_ context.Context, scope ReportScope, id string) (ReportRun, error) {
	r.getRunCalls++
	run, ok := r.runs[id]
	if !ok || run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	return run, nil
}

func (r *serviceTestRepository) ListRuns(context.Context, ReportScope, string, int) ([]ReportRun, error) {
	return nil, nil
}

func (r *serviceTestRepository) ClaimQueuedRuns(_ context.Context, scope ReportScope, _ string, _ int) ([]ReportRun, error) {
	for id, run := range r.runs {
		if run.Status != RunQueued || run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
			continue
		}
		run.Status = RunRunning
		run.AttemptCount++
		if r.claimAttemptOverride > 0 {
			run.AttemptCount = r.claimAttemptOverride
		}
		r.runs[id] = run
		return []ReportRun{run}, nil
	}
	return nil, nil
}

func (r *serviceTestRepository) CompleteRun(_ context.Context, scope ReportScope, run ReportRun) (ReportRun, error) {
	r.completeCalls++
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	r.runs[run.ID] = run
	return run, nil
}

func (r *serviceTestRepository) FailRun(_ context.Context, scope ReportScope, id, failureCode string) (ReportRun, error) {
	if r.failRunErr != nil {
		return ReportRun{}, r.failRunErr
	}
	run, err := r.GetRun(context.Background(), scope, id)
	if err != nil {
		return ReportRun{}, err
	}
	run.Status = RunFailed
	run.FailureCode = failureCode
	run.RowCount = 0
	run.DataObjectKey = ""
	run.DataSHA256 = ""
	run.ManifestObjectKey = ""
	run.ManifestSHA256 = ""
	r.runs[id] = run
	return run, nil
}

func (r *serviceTestRepository) RecordRunDownload(_ context.Context, scope ReportScope, id, downloadedBy string) error {
	if downloadedBy == "" {
		return ErrInvalid
	}
	if _, err := r.GetRun(context.Background(), scope, id); err != nil {
		return err
	}
	r.downloads++
	return nil
}

func (r *serviceTestRepository) CaptureSourceBoundary(_ context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error) {
	if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return SourceBoundary{}, ErrNotFound
	}
	boundary := r.boundary
	boundary.SourceHighWater = cloneTimeMap(boundary.SourceHighWater)
	return boundary, nil
}

func (r *serviceTestRepository) ListReportRows(_ context.Context, scope ReportScope, run ReportRun, cursor string, limit int) (ReportPage, error) {
	r.pageCalls++
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportPage{}, ErrNotFound
	}
	if r.pageErr != nil {
		return ReportPage{}, r.pageErr
	}
	start := 0
	if cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil {
			return ReportPage{}, ErrInvalid
		}
		start = parsed
	}
	rows := r.rows[run.DefinitionID]
	if start > len(rows) {
		return ReportPage{}, ErrInvalid
	}
	end := start + limit
	if end > len(rows) {
		end = len(rows)
	}
	page := ReportPage{Rows: append([]ReportRow(nil), rows[start:end]...)}
	if end < len(rows) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

type serviceTestObjectStore struct {
	mu          sync.Mutex
	objects     map[string][]byte
	puts        []string
	deleteCalls []string
	failPutKey  string
}

func newServiceTestObjectStore() *serviceTestObjectStore {
	return &serviceTestObjectStore{objects: make(map[string][]byte)}
}

func (s *serviceTestObjectStore) Put(_ context.Context, key string, reader io.Reader, max int64) (evidence.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == s.failPutKey {
		return evidence.ObjectInfo{}, errors.New("object store write failed")
	}
	data, err := io.ReadAll(io.LimitReader(reader, max+1))
	if err != nil {
		return evidence.ObjectInfo{}, err
	}
	if int64(len(data)) > max {
		return evidence.ObjectInfo{}, evidence.ErrArtifactTooLarge
	}
	digest := sha256.Sum256(data)
	s.objects[key] = bytes.Clone(data)
	s.puts = append(s.puts, key)
	return evidence.ObjectInfo{Key: key, SizeBytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}, nil
}

func (s *serviceTestObjectStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(bytes.Clone(data))), nil
}

func (s *serviceTestObjectStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	s.deleteCalls = append(s.deleteCalls, key)
	return nil
}

func objectKeys(store *serviceTestObjectStore) []string {
	keys := make([]string, 0, len(store.objects))
	for key := range store.objects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneTimeMap(source map[string]time.Time) map[string]time.Time {
	if source == nil {
		return nil
	}
	result := make(map[string]time.Time, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
