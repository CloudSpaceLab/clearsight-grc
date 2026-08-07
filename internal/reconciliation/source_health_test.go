package reconciliation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type sourceDependencyStub struct {
	programIDs []string
	current    map[string]bool
}

func (s sourceDependencyStub) ProgramIDsForEvidenceSource(context.Context, string, string) ([]string, error) {
	return append([]string(nil), s.programIDs...), nil
}
func (s sourceDependencyStub) EvidenceSourcesCurrentForProgram(_ context.Context, _, programID string) (bool, error) {
	return s.current[programID], nil
}

type triggerSinkStub struct {
	triggers    []continuity.Trigger
	failProgram string
}

func (s *triggerSinkStub) ApplyTrigger(_ context.Context, trigger continuity.Trigger) (continuity.ProgramAggregate, *continuity.Matter, bool, error) {
	if trigger.ProgramID == s.failProgram {
		return continuity.ProgramAggregate{}, nil, false, errors.New("trigger failed")
	}
	s.triggers = append(s.triggers, trigger)
	return continuity.ProgramAggregate{}, nil, true, nil
}

func TestSourceHealthReconciliationDrivesDriftAndProgramBoundaries(t *testing.T) {
	ctx := context.Background()
	inbox := workflowruntime.NewMemoryRepository()
	signals := autonomy.NewService(autonomy.NewMemoryRepository())
	programs := &triggerSinkStub{}
	consumer := &SourceHealthConsumer{
		Inbox:        inbox,
		Dependencies: sourceDependencyStub{programIDs: []string{"program-1", "program-2"}, current: map[string]bool{"program-1": true, "program-2": false}},
		Signals:      signals,
		Programs:     programs,
	}
	now := time.Date(2026, 8, 7, 1, 0, 0, 0, time.UTC)
	degraded := workflowruntime.OutboxEvent{
		ID: "event-degraded", TenantID: "bank-demo", AggregateType: "EVIDENCE_SOURCE", AggregateID: "source-1",
		EventType: "SourceHealthChanged", Payload: json.RawMessage(`{"from":"CURRENT","to":"DEGRADED","source_code":"CORE"}`), OccurredAt: now,
	}
	if err := consumer.Publish(ctx, degraded); err != nil {
		t.Fatal(err)
	}
	if len(programs.triggers) != 2 {
		t.Fatalf("expected one degradation trigger per dependent program, got %#v", programs.triggers)
	}
	if programs.triggers[0].DedupeKey == programs.triggers[1].DedupeKey {
		t.Fatal("program trigger dedupe keys must include the program id")
	}
	readiness, err := signals.Readiness(ctx, "bank-demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(readiness.ActiveDrifts) != 1 || readiness.ActiveDrifts[0].SubjectID != "source-1" {
		t.Fatalf("source drift was not created: %#v", readiness.ActiveDrifts)
	}
	if err := consumer.Publish(ctx, degraded); err != nil {
		t.Fatal(err)
	}
	if len(programs.triggers) != 2 {
		t.Fatalf("inbox dedupe replayed a delivered event: %#v", programs.triggers)
	}

	recovered := workflowruntime.OutboxEvent{
		ID: "event-recovered", TenantID: "bank-demo", AggregateType: "EVIDENCE_SOURCE", AggregateID: "source-1",
		EventType: "SourceHealthChanged", Payload: json.RawMessage(`{"from":"DEGRADED","to":"CURRENT","source_code":"CORE"}`), OccurredAt: now.Add(time.Minute),
	}
	if err := consumer.Publish(ctx, recovered); err != nil {
		t.Fatal(err)
	}
	if len(programs.triggers) != 3 || programs.triggers[2].ProgramID != "program-1" || programs.triggers[2].Type != "SOURCE_RECOVERED" {
		t.Fatalf("recovery should reach only programs whose active sources are all current: %#v", programs.triggers)
	}
	readiness, err = signals.Readiness(ctx, "bank-demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(readiness.ActiveDrifts) != 0 {
		t.Fatalf("source recovery did not resolve source drift: %#v", readiness.ActiveDrifts)
	}
}

func TestSourceHealthReconciliationDoesNotOpenNewEpisodeForUnhealthyToUnhealthyChange(t *testing.T) {
	consumer := &SourceHealthConsumer{
		Inbox:        workflowruntime.NewMemoryRepository(),
		Dependencies: sourceDependencyStub{programIDs: []string{"program-1"}, current: map[string]bool{}},
		Signals:      autonomy.NewService(autonomy.NewMemoryRepository()),
		Programs:     &triggerSinkStub{},
	}
	event := workflowruntime.OutboxEvent{
		ID: "event-worse", TenantID: "bank-demo", AggregateType: "EVIDENCE_SOURCE", AggregateID: "source-1",
		EventType: "SourceHealthChanged", Payload: json.RawMessage(`{"from":"DEGRADED","to":"UNAVAILABLE"}`), OccurredAt: time.Now().UTC(),
	}
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(consumer.Programs.(*triggerSinkStub).triggers) != 0 {
		t.Fatal("unhealthy-to-unhealthy transition created a duplicate source episode")
	}
}

func TestSourceHealthReconciliationRecordsInboxOnlyAfterAllProgramEffects(t *testing.T) {
	ctx := context.Background()
	inbox := workflowruntime.NewMemoryRepository()
	programs := &triggerSinkStub{failProgram: "program-2"}
	consumer := &SourceHealthConsumer{
		Inbox:        inbox,
		Dependencies: sourceDependencyStub{programIDs: []string{"program-1", "program-2"}, current: map[string]bool{}},
		Signals:      autonomy.NewService(autonomy.NewMemoryRepository()),
		Programs:     programs,
	}
	event := workflowruntime.OutboxEvent{
		ID: "event-fail", TenantID: "bank-demo", AggregateType: "EVIDENCE_SOURCE", AggregateID: "source-1",
		EventType: "SourceHealthChanged", Payload: json.RawMessage(`{"from":"CURRENT","to":"STALE"}`), OccurredAt: time.Now().UTC(),
	}
	if err := consumer.Publish(ctx, event); err == nil {
		t.Fatal("expected program effect failure")
	}
	processed, err := inbox.InboxProcessed(ctx, "bank-demo", sourceHealthConsumer, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if processed {
		t.Fatal("failed reconciliation was recorded as delivered")
	}
}
