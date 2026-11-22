package reporting

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type countingReportOpenStore struct {
	evidence.ObjectStore
	opens int
}

type reportAvailabilityRepository struct {
	*serviceTestRepository
	definition ReportDefinition
	calls      int
}

func (r *reportAvailabilityRepository) ListDefinitions(context.Context, ReportScope, bool) ([]ReportDefinition, error) {
	r.calls++
	return []ReportDefinition{r.definition}, nil
}

func (s *countingReportOpenStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	s.opens++
	return s.ObjectStore.Open(ctx, key)
}

func TestProgramDerivedReportsCannotCreateOrOpenArtifacts(t *testing.T) {
	for _, dataset := range []ReportDataset{DatasetPrograms, DatasetMatterBoardBrief} {
		t.Run(string(dataset), func(t *testing.T) {
			service, repository, objects, _ := newReportingServiceTest()
			definition := installActiveDefinition(repository)
			definition.Dataset = dataset
			if dataset == DatasetMatterBoardBrief {
				definition.ScopeKind = ScopeMatter
				definition.ScopeRef = "00000000-0000-7000-8000-000000000301"
				definition.Format = FormatPDF
			}
			definition.StoredChecksum = definition.Checksum()
			repository.definitions[definition.ID] = definition
			if _, err := service.CreateRun(reportActorContext(testPerformerID), CreateRunInput{Scope: testScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion}); !errors.Is(err, ErrProgramReportUnavailable) || len(repository.runs) != 0 {
				t.Fatalf("unsafe run creation: %v", err)
			}
			ready := installReadyRun(t, repository, objects)
			ready.Dataset = dataset
			repository.runs[ready.ID] = ready
			counted := &countingReportOpenStore{ObjectStore: objects}
			service.objects = counted
			for _, principal := range []string{testPerformerID, testOtherActorID} {
				if _, reader, err := service.Open(reportActorContext(principal), testScope(), ready.ID); !errors.Is(err, ErrProgramReportUnavailable) || reader != nil || repository.downloads != 0 || counted.opens != 0 {
					t.Fatalf("unsafe existing artifact opened: %v", err)
				}
			}
			service.repo = privacyReceiptTestRepository{serviceTestRepository: repository, receipt: ReportRun{ID: ready.ID, Dataset: dataset}}
			if _, err := service.CreateRun(reportActorContext(testPerformerID), CreateRunInput{Scope: testScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion, IdempotencyKey: "00000000-0000-7000-8000-000000000555"}); !errors.Is(err, ErrProgramReportUnavailable) {
				t.Fatalf("unsafe existing receipt returned: %v", err)
			}
		})
	}
}

func TestProgramDerivedQueuedReportsStopBeforeReadingRows(t *testing.T) {
	for _, dataset := range []ReportDataset{DatasetPrograms, DatasetMatterBoardBrief} {
		t.Run(string(dataset), func(t *testing.T) {
			service, repository, objects, _ := newReportingServiceTest()
			run := ReportRun{ID: "00000000-0000-7000-8000-000000000201", TenantID: testTenantID, LegalEntityID: testEntityA, DefinitionID: testDefinitionID, DefinitionVersion: 1, DefinitionCode: "PROGRAM-REPORT", DefinitionChecksum: strings.Repeat("a", 64), ScopeKind: ScopeLegalEntity, RequestedByRef: testPerformerID, Dataset: dataset, Format: FormatCSV, Status: RunQueued, SourceBoundary: testSourceBoundary(0), CreatedAt: serviceTestNow, ExpiresAt: serviceTestNow.Add(ReportRunRetention)}
			if dataset == DatasetMatterBoardBrief {
				run.ScopeKind = ScopeMatter
				run.ScopeRef = "00000000-0000-7000-8000-000000000301"
				run.Format = FormatPDF
			}
			repository.runs[run.ID] = run
			if _, err := service.ExecuteRun(context.Background(), run); !errors.Is(err, ErrProgramReportUnavailable) || repository.pageCalls != 0 || len(objects.puts) != 0 || repository.runs[run.ID].Status != RunFailed || repository.runs[run.ID].FailureCode != FailureProgramScopeUnavailable {
				t.Fatalf("unsafe queued report read source: %v pages=%d", err, repository.pageCalls)
			}
		})
	}
}

func TestProgramReportOwnerOptionsRemainUnavailable(t *testing.T) {
	service, _, _, _ := newReportingServiceTest()
	if _, err := service.ListOwners(reportActorContext(testPerformerID), testScope(), DatasetPrograms, 10); !errors.Is(err, ErrProgramReportUnavailable) {
		t.Fatalf("Program owners exposed: %v", err)
	}
	if _, err := service.ListOwners(reportActorContext(testPerformerID), testScope(), DatasetMatters, 10); err != nil {
		t.Fatalf("unrelated Matter owner options blocked: %v", err)
	}
}

func TestMatterBoardBriefDoesNotAdvertiseUnavailableRun(t *testing.T) {
	service, repository, _, _ := newReportingServiceTest()
	definition := installActiveDefinition(repository)
	definition.Dataset = DatasetMatterBoardBrief
	definition.ScopeKind = ScopeMatter
	definition.ScopeRef = "00000000-0000-7000-8000-000000000301"
	definition.Format = FormatPDF
	definition.StoredChecksum = definition.Checksum()
	repository.definitions[definition.ID] = definition
	tracking := &reportAvailabilityRepository{serviceTestRepository: repository, definition: definition}
	service.repo = tracking
	availability, err := service.MatterBoardBriefAvailability(reportActorContext(testPerformerID), testScope(), definition.ScopeRef)
	if err != nil || availability.CanRun || availability.Definition != nil || availability.Reason == "" || tracking.calls != 0 {
		t.Fatalf("unavailable report advertised: %+v %v", availability, err)
	}
}
