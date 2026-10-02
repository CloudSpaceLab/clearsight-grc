package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type episodeContinuityStub struct {
	programs   map[string]continuity.ProgramAggregate
	matters    map[string]continuity.MatterAggregate
	byTrigger  map[string]string
	applyCount int
}

func newEpisodeContinuityStub(program continuity.Program) *episodeContinuityStub {
	return &episodeContinuityStub{
		programs:  map[string]continuity.ProgramAggregate{program.ID: {Program: program}},
		matters:   map[string]continuity.MatterAggregate{},
		byTrigger: map[string]string{},
	}
}

func (s *episodeContinuityStub) GetProgram(_ context.Context, tenant, id string) (continuity.ProgramAggregate, error) {
	value, ok := s.programs[id]
	if !ok || value.Program.TenantID != tenant {
		return continuity.ProgramAggregate{}, continuity.ErrNotFound
	}
	return value, nil
}

func (s *episodeContinuityStub) ApplyTrigger(_ context.Context, trigger continuity.Trigger) (continuity.ProgramAggregate, *continuity.Matter, bool, error) {
	program, ok := s.programs[trigger.ProgramID]
	if !ok || program.Program.TenantID != trigger.TenantID {
		return continuity.ProgramAggregate{}, nil, false, continuity.ErrNotFound
	}
	if matterID, ok := s.byTrigger[trigger.DedupeKey]; ok {
		value := s.matters[matterID]
		return program, &value.Matter, false, nil
	}
	s.applyCount++
	matterID := fmt.Sprintf("matter-%d", s.applyCount)
	matter := continuity.Matter{
		ID: matterID, TenantID: trigger.TenantID, LegalEntityID: program.Program.LegalEntityID,
		TriggerType: trigger.Type, TriggerID: trigger.SubjectID, TriggerKey: trigger.DedupeKey,
		Status: continuity.MatterInitialReview, Version: 1, CreatedAt: trigger.ObservedAt, UpdatedAt: trigger.ObservedAt,
	}
	s.byTrigger[trigger.DedupeKey] = matterID
	s.matters[matterID] = continuity.MatterAggregate{Matter: matter}
	return program, &matter, true, nil
}

func (s *episodeContinuityStub) GetMatter(_ context.Context, tenant, id string) (continuity.MatterAggregate, error) {
	value, ok := s.matters[id]
	if !ok || value.Matter.TenantID != tenant {
		return continuity.MatterAggregate{}, continuity.ErrNotFound
	}
	return value, nil
}

func TestAdverseEpisodeReusesMatterClosesOnClearAndReopensLater(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	program := continuity.Program{
		ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a",
		Code: "RESILIENCE", Name: "Resilience", Status: continuity.ProgramActive, Version: 1,
	}
	continuityStub := newEpisodeContinuityStub(program)
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	ids := []string{"episode-1", "episode-2"}
	coordinator := &AdverseEpisodeCoordinator{
		Repository: repo, Continuity: continuityStub, Now: func() time.Time { return now },
		NewID: func() (string, error) {
			if len(ids) == 0 {
				return "", fmt.Errorf("no episode IDs remain")
			}
			value := ids[0]
			ids = ids[1:]
			return value, nil
		},
	}
	check := MonitoringCheck{
		ID: "check-1", TenantID: "bank", ProgramID: program.ID, Code: "KRI-1", Name: "Critical failure rate",
		MinimumCoverage: 0.9, FailureAction: FailureRecommendMatter,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 3},
	}
	score := 65.0
	first := MonitoringResult{
		ID: "result-1", TenantID: "bank", ProgramID: program.ID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		Evaluation: Evaluation{Score: &score, Band: RiskHigh, Coverage: 1}, EvaluatedAt: now,
	}
	firstObservation := AdverseEpisodeObservation{TenantID: "bank", LegalEntityID: "entity-a", ProgramID: program.ID, Check: check, Result: first}

	opened, err := coordinator.Reconcile(ctx, firstObservation)
	if err != nil || opened.Episode == nil || opened.EpisodeChange != AdverseEpisodeOpened {
		t.Fatalf("open episode result=%#v err=%v", opened, err)
	}
	if opened.Episode.MatterID != "" || continuityStub.applyCount != 0 {
		t.Fatalf("episode reconciliation created work without user command: %#v", opened.Episode)
	}
	linked, err := coordinator.EnsureMatter(ctx, firstObservation, *opened.Episode, "risk-owner")
	if err != nil || linked.Matter == nil || !linked.MatterCreated {
		t.Fatalf("ensure Matter result=%#v err=%v", linked, err)
	}
	firstMatterID := linked.Matter.ID

	now = now.Add(time.Minute)
	score = 68
	second := first
	second.ID, second.Evaluation.Score, second.EvaluatedAt = "result-2", &score, now
	secondObservation := firstObservation
	secondObservation.Result = second
	updated, err := coordinator.Reconcile(ctx, secondObservation)
	if err != nil || updated.Episode == nil || updated.Episode.ID != opened.Episode.ID || updated.EpisodeChange != AdverseEpisodeUpdated {
		t.Fatalf("repeat adverse result=%#v err=%v", updated, err)
	}
	reused, err := coordinator.EnsureMatter(ctx, secondObservation, *updated.Episode, "risk-owner")
	if err != nil || reused.Matter == nil || reused.Matter.ID != firstMatterID || reused.MatterCreated || continuityStub.applyCount != 1 {
		t.Fatalf("persistent breach created duplicate Matter: result=%#v err=%v count=%d", reused, err, continuityStub.applyCount)
	}

	now = now.Add(time.Minute)
	score = 92
	worse := second
	worse.ID, worse.Evaluation.Score, worse.Evaluation.Band, worse.EvaluatedAt = "result-3", &score, RiskCritical, now
	worseObservation := firstObservation
	worseObservation.Result = worse
	worsened, err := coordinator.Reconcile(ctx, worseObservation)
	if err != nil || worsened.Episode == nil || worsened.Episode.ID != opened.Episode.ID || worsened.EpisodeChange != AdverseEpisodeWorsened {
		t.Fatalf("worsened episode=%#v err=%v", worsened, err)
	}

	now = now.Add(time.Minute)
	score = 10
	clear := worse
	clear.ID, clear.Evaluation.Score, clear.Evaluation.Band, clear.Evaluation.Coverage, clear.EvaluatedAt = "result-4", &score, RiskLow, 1, now
	clearObservation := firstObservation
	clearObservation.Result = clear
	cleared, err := coordinator.Reconcile(ctx, clearObservation)
	if err != nil || cleared.Episode == nil || cleared.Episode.State != AdverseEpisodeClosed || cleared.EpisodeChange != AdverseEpisodeCleared {
		t.Fatalf("clear episode=%#v err=%v", cleared, err)
	}
	if continuityStub.matters[firstMatterID].Matter.Status != continuity.MatterInitialReview {
		t.Fatalf("clear result changed governed Matter state: %#v", continuityStub.matters[firstMatterID].Matter)
	}
	if _, err := repo.OpenAdverseEpisode(ctx, "bank", "entity-a", check.ID); err != ErrNotFound {
		t.Fatalf("closed episode remains open: %v", err)
	}

	now = now.Add(time.Minute)
	score = 70
	later := clear
	later.ID, later.Evaluation.Score, later.Evaluation.Band, later.EvaluatedAt = "result-5", &score, RiskHigh, now
	laterObservation := firstObservation
	laterObservation.Result = later
	reopened, err := coordinator.Reconcile(ctx, laterObservation)
	if err != nil || reopened.Episode == nil || reopened.Episode.ID == opened.Episode.ID || reopened.EpisodeChange != AdverseEpisodeOpened {
		t.Fatalf("later breach did not start a new episode: %#v err=%v", reopened, err)
	}
	secondMatter, err := coordinator.EnsureMatter(ctx, laterObservation, *reopened.Episode, "risk-owner")
	if err != nil || secondMatter.Matter == nil || secondMatter.Matter.ID == firstMatterID || !secondMatter.MatterCreated || continuityStub.applyCount != 2 {
		t.Fatalf("new episode did not create distinct governed work: %#v err=%v", secondMatter, err)
	}
}

func TestAdverseEpisodeConsumerIsReplaySafeAndDoesNotAutoCreateMatter(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	inbox := workflowruntime.NewMemoryRepository()
	now := time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)
	program := continuity.Program{
		ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a",
		Code: "RESILIENCE", Name: "Resilience", Status: continuity.ProgramActive, Version: 1,
	}
	continuityStub := newEpisodeContinuityStub(program)
	check := MonitoringCheck{
		ID: "check-1", TenantID: "bank", ProgramID: program.ID, Code: "KRI-1", Name: "Critical failure rate",
		MinimumCoverage: 0.9, FailureAction: FailureRecommendMatter,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 1, CreatedAt: now, UpdatedAt: now},
	}
	if _, err := repo.CreateCheckRevision(ctx, check); err != nil {
		t.Fatal(err)
	}
	score := 80.0
	result := MonitoringResult{
		ID: "result-1", TenantID: "bank", ProgramID: program.ID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: InputSource, InputReferenceID: "receipt-1", InputReferenceVersion: 1,
		Evaluation: Evaluation{Score: &score, Band: RiskHigh, Coverage: 1},
		EvaluatedAt: now, EvaluatorVersion: "risk-v1", CreatedAt: now,
	}
	stored, err := repo.AppendResult(ctx, result)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(stored)
	event := workflowruntime.OutboxEvent{
		ID: "event-1", TenantID: "bank", AggregateType: AggregateMonitoringResult, AggregateID: stored.ID,
		EventType: EventMonitoringResultRecorded, Payload: payload, OccurredAt: now,
	}
	consumer := &AdverseEpisodeConsumer{
		Inbox: inbox, Monitoring: repo,
		Coordinator: &AdverseEpisodeCoordinator{
			Repository: repo, Continuity: continuityStub, Now: func() time.Time { return now },
			NewID: func() (string, error) { return "episode-1", nil },
		},
		Now: func() time.Time { return now },
	}
	if err := consumer.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	episode, err := repo.OpenAdverseEpisode(ctx, "bank", "entity-a", check.ID)
	if err != nil || episode.ID != "episode-1" {
		t.Fatalf("episode=%#v err=%v", episode, err)
	}
	if continuityStub.applyCount != 0 {
		t.Fatalf("background consumer auto-created Matter count=%d", continuityStub.applyCount)
	}
	before := len(repo.MonitoringEvents("bank", AggregateMonitoringAdverseEpisode, episode.ID))
	if err := consumer.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	after := len(repo.MonitoringEvents("bank", AggregateMonitoringAdverseEpisode, episode.ID))
	if before != after {
		t.Fatalf("replayed event duplicated episode history: before=%d after=%d", before, after)
	}
	processed, err := inbox.InboxProcessed(ctx, "bank", adverseEpisodeConsumer, event.ID)
	if err != nil || !processed {
		t.Fatalf("inbox processed=%v err=%v", processed, err)
	}
}
