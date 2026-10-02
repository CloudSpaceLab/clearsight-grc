package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const (
	AggregateMonitoringAdverseEpisode = "MONITORING_ADVERSE_EPISODE"

	EventMonitoringAdverseEpisodeOpened       = "MONITORING_ADVERSE_EPISODE_OPENED"
	EventMonitoringAdverseEpisodeUpdated      = "MONITORING_ADVERSE_EPISODE_UPDATED"
	EventMonitoringAdverseEpisodeWorsened     = "MONITORING_ADVERSE_EPISODE_WORSENED"
	EventMonitoringAdverseEpisodeCleared      = "MONITORING_ADVERSE_EPISODE_CLEARED"
	EventMonitoringAdverseEpisodeMatterLinked = "MONITORING_ADVERSE_EPISODE_MATTER_LINKED"

	adverseEpisodeConsumer = "monitoring-adverse-episode"
)

type AdverseEpisodeState string

const (
	AdverseEpisodeOpen   AdverseEpisodeState = "OPEN"
	AdverseEpisodeClosed AdverseEpisodeState = "CLOSED"
)

type AdverseEpisode struct {
	ID                     string              `json:"id"`
	TenantID               string              `json:"tenant_id"`
	LegalEntityID          string              `json:"legal_entity_id"`
	ProgramID              string              `json:"program_id"`
	MonitoringCheckID      string              `json:"monitoring_check_id"`
	State                  AdverseEpisodeState `json:"state"`
	MatterID               string              `json:"matter_id,omitempty"`
	FirstResultID          string              `json:"first_result_id"`
	LastResultID           string              `json:"last_result_id"`
	LastCheckVersion       int64               `json:"last_check_version"`
	LastBand               RiskBand            `json:"last_band"`
	LastScore              *float64            `json:"last_score,omitempty"`
	LastCoverage           float64             `json:"last_coverage"`
	OpenedAt               time.Time           `json:"opened_at"`
	ClosedAt               *time.Time          `json:"closed_at,omitempty"`
	UpdatedAt              time.Time           `json:"updated_at"`
	RecordVersion          int64               `json:"record_version"`
}

type AdverseEpisodeChange string

const (
	AdverseEpisodeNoChange AdverseEpisodeChange = "NO_CHANGE"
	AdverseEpisodeOpened   AdverseEpisodeChange = "OPENED"
	AdverseEpisodeUpdated  AdverseEpisodeChange = "UPDATED"
	AdverseEpisodeWorsened AdverseEpisodeChange = "WORSENED"
	AdverseEpisodeCleared  AdverseEpisodeChange = "CLEARED"
)

type AdverseEpisodeObservation struct {
	TenantID      string
	LegalEntityID string
	ProgramID     string
	Check         MonitoringCheck
	Result        MonitoringResult
}

type AdverseEpisodeRepository interface {
	OpenOrUpdateAdverseEpisode(context.Context, AdverseEpisodeObservation, string, time.Time) (AdverseEpisode, AdverseEpisodeChange, error)
	CloseAdverseEpisode(context.Context, AdverseEpisodeObservation, time.Time) (*AdverseEpisode, AdverseEpisodeChange, error)
	OpenAdverseEpisode(context.Context, string, string, string) (AdverseEpisode, error)
	AttachAdverseEpisodeMatter(context.Context, string, string, string, string, time.Time) (AdverseEpisode, error)
}

type adverseEpisodeInbox interface {
	InboxProcessed(context.Context, string, string, string) (bool, error)
	RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
}

type adverseEpisodeContinuity interface {
	GetProgram(context.Context, string, string) (continuity.ProgramAggregate, error)
	ApplyTrigger(context.Context, continuity.Trigger) (continuity.ProgramAggregate, *continuity.Matter, bool, error)
	GetMatter(context.Context, string, string) (continuity.MatterAggregate, error)
}

type AdverseEpisodeCoordinator struct {
	Repository AdverseEpisodeRepository
	Continuity adverseEpisodeContinuity
	Now        func() time.Time
	NewID      func() (string, error)
}

type AdverseEpisodeResult struct {
	Episode       *AdverseEpisode
	Matter        *continuity.Matter
	EpisodeChange AdverseEpisodeChange
	MatterCreated bool
}

func (c *AdverseEpisodeCoordinator) Reconcile(ctx context.Context, observation AdverseEpisodeObservation) (AdverseEpisodeResult, error) {
	if c == nil || c.Repository == nil {
		return AdverseEpisodeResult{}, fmt.Errorf("monitoring adverse episode coordinator is not configured")
	}
	if err := validateEpisodeObservation(observation); err != nil {
		return AdverseEpisodeResult{}, err
	}
	now := c.currentTime()
	if !AdverseResult(observation.Check, observation.Result) {
		closed, change, err := c.Repository.CloseAdverseEpisode(ctx, observation, now)
		if err != nil {
			return AdverseEpisodeResult{}, err
		}
		return AdverseEpisodeResult{Episode: closed, EpisodeChange: change}, nil
	}

	episodeID := ""
	current, lookupErr := c.Repository.OpenAdverseEpisode(ctx, observation.TenantID, observation.LegalEntityID, observation.Check.ID)
	switch {
	case lookupErr == nil:
		episodeID = current.ID
	case errors.Is(lookupErr, ErrNotFound):
		episodeID, err = c.nextID()
		if err != nil {
			return AdverseEpisodeResult{}, err
		}
	default:
		return AdverseEpisodeResult{}, lookupErr
	}

	episode, change, err := c.Repository.OpenOrUpdateAdverseEpisode(ctx, observation, episodeID, now)
	if errors.Is(err, ErrConflict) {
		// Another worker/API request may have opened the same stable episode
		// after this caller observed no row. Retry against that committed row.
		current, lookupErr = c.Repository.OpenAdverseEpisode(ctx, observation.TenantID, observation.LegalEntityID, observation.Check.ID)
		if lookupErr != nil {
			return AdverseEpisodeResult{}, err
		}
		episode, change, err = c.Repository.OpenOrUpdateAdverseEpisode(ctx, observation, current.ID, now)
	}
	if err != nil {
		return AdverseEpisodeResult{}, err
	}
	return AdverseEpisodeResult{Episode: &episode, EpisodeChange: change}, nil
}

func (c *AdverseEpisodeCoordinator) EnsureMatter(ctx context.Context, observation AdverseEpisodeObservation, episode AdverseEpisode, actorID string) (AdverseEpisodeResult, error) {
	if c == nil || c.Repository == nil || c.Continuity == nil {
		return AdverseEpisodeResult{}, fmt.Errorf("monitoring adverse episode Matter coordinator is not configured")
	}
	if err := validateEpisodeObservation(observation); err != nil {
		return AdverseEpisodeResult{}, err
	}
	if episode.ID == "" || episode.State != AdverseEpisodeOpen || episode.TenantID != observation.TenantID ||
		episode.LegalEntityID != observation.LegalEntityID || episode.ProgramID != observation.ProgramID ||
		episode.MonitoringCheckID != observation.Check.ID || !AdverseResult(observation.Check, observation.Result) ||
		observation.Check.FailureAction != FailureRecommendMatter {
		return AdverseEpisodeResult{}, ErrLinkedIssueIneligible
	}
	out := AdverseEpisodeResult{Episode: &episode}
	if episode.MatterID != "" {
		existing, err := c.Continuity.GetMatter(ctx, episode.TenantID, episode.MatterID)
		if err != nil {
			return AdverseEpisodeResult{}, fmt.Errorf("load episode Matter: %w", err)
		}
		if existing.Matter.LegalEntityID != episode.LegalEntityID {
			return AdverseEpisodeResult{}, ErrInvalid
		}
		out.Matter = &existing.Matter
		return out, nil
	}

	payload, err := json.Marshal(map[string]any{
		"monitoring_episode_id": episode.ID,
		"monitoring_result_id": observation.Result.ID,
		"monitoring_check_id": observation.Check.ID,
		"monitoring_check_version": observation.Check.Version,
		"monitoring_check_name": observation.Check.Name,
		"risk_band": observation.Result.Evaluation.Band,
		"score": observation.Result.Evaluation.Score,
		"coverage": observation.Result.Evaluation.Coverage,
		"evaluated_at": observation.Result.EvaluatedAt,
	})
	if err != nil {
		return AdverseEpisodeResult{}, err
	}
	triggerKey := "monitoring-adverse-episode:" + episode.ID
	_, matter, inserted, err := c.Continuity.ApplyTrigger(ctx, continuity.Trigger{
		TenantID: episode.TenantID, ProgramID: episode.ProgramID, Type: "MONITORING_RESULT_ADVERSE",
		SubjectType: AggregateMonitoringAdverseEpisode, SubjectID: episode.ID, DedupeKey: triggerKey,
		Payload: payload, ObservedAt: observation.Result.EvaluatedAt, Source: "monitoring-adverse-episode",
		ActorID: strings.TrimSpace(actorID),
	})
	if err != nil {
		return AdverseEpisodeResult{}, err
	}
	if matter == nil {
		return AdverseEpisodeResult{}, fmt.Errorf("monitoring adverse episode trigger returned no Matter")
	}
	updated, err := c.Repository.AttachAdverseEpisodeMatter(ctx, episode.TenantID, episode.LegalEntityID, episode.ID, matter.ID, c.currentTime())
	if err != nil {
		return AdverseEpisodeResult{}, err
	}
	out.Episode = &updated
	out.Matter = matter
	out.MatterCreated = inserted
	return out, nil
}

func AdverseResult(check MonitoringCheck, result MonitoringResult) bool {
	if result.MonitoringCheckID != check.ID || result.MonitoringCheckVersion != check.Version || result.ProgramID != check.ProgramID {
		return false
	}
	if result.Evaluation.Band == RiskHigh || result.Evaluation.Band == RiskCritical || result.Evaluation.Coverage < check.MinimumCoverage || len(result.Evaluation.CriticalFailures) > 0 {
		return true
	}
	for _, rule := range result.Evaluation.RuleResults {
		if rule.Critical && rule.Outcome != RulePassed {
			return true
		}
	}
	return false
}

func validateEpisodeObservation(value AdverseEpisodeObservation) error {
	if strings.TrimSpace(value.TenantID) == "" || strings.TrimSpace(value.LegalEntityID) == "" || value.LegalEntityID == "*" ||
		strings.TrimSpace(value.ProgramID) == "" || value.Check.ID == "" || value.Check.Version < 1 ||
		value.Result.ID == "" || value.Result.MonitoringCheckID != value.Check.ID ||
		value.Result.MonitoringCheckVersion != value.Check.Version || value.Result.ProgramID != value.ProgramID ||
		value.Check.ProgramID != value.ProgramID || value.Result.EvaluatedAt.IsZero() {
		return ErrInvalid
	}
	return nil
}

func (c *AdverseEpisodeCoordinator) currentTime() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c *AdverseEpisodeCoordinator) nextID() (string, error) {
	if c.NewID != nil {
		return c.NewID()
	}
	return id.NewUUIDv7()
}

type AdverseEpisodeConsumer struct {
	Inbox       adverseEpisodeInbox
	Monitoring  Repository
	Coordinator *AdverseEpisodeCoordinator
	Now         func() time.Time
}

func (c *AdverseEpisodeConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != AggregateMonitoringResult || event.EventType != EventMonitoringResultRecorded {
		return nil
	}
	if c == nil || c.Inbox == nil || c.Monitoring == nil || c.Coordinator == nil || c.Coordinator.Continuity == nil {
		return fmt.Errorf("monitoring adverse episode consumer is not configured")
	}
	processed, err := c.Inbox.InboxProcessed(ctx, event.TenantID, adverseEpisodeConsumer, event.ID)
	if err != nil {
		return fmt.Errorf("check monitoring adverse episode inbox: %w", err)
	}
	if processed {
		return nil
	}
	var result MonitoringResult
	if err := json.Unmarshal(event.Payload, &result); err != nil {
		return fmt.Errorf("decode monitoring result event: %w", err)
	}
	if result.ID != event.AggregateID || result.TenantID != event.TenantID {
		return fmt.Errorf("monitoring result event identity mismatch")
	}
	check, err := c.Monitoring.LatestCheckRevision(ctx, event.TenantID, result.MonitoringCheckID)
	if err != nil {
		return fmt.Errorf("load current monitoring check: %w", err)
	}
	if check.Version != result.MonitoringCheckVersion || !check.IsCurrent || check.Status != LifecycleActive {
		return c.recordInbox(ctx, event)
	}
	latest, err := c.Monitoring.LatestResultRevision(ctx, event.TenantID, check.ID, check.Version)
	if err != nil {
		return fmt.Errorf("load current monitoring result: %w", err)
	}
	if latest.ID != result.ID {
		return c.recordInbox(ctx, event)
	}
	program, err := c.Coordinator.Continuity.GetProgram(continuity.WithTrustedSystemScope(ctx), event.TenantID, check.ProgramID)
	if err != nil {
		return fmt.Errorf("load monitoring Program: %w", err)
	}
	if program.Program.ID != result.ProgramID || program.Program.LegalEntityID == "" {
		return fmt.Errorf("monitoring result does not match Program scope")
	}
	if _, err := c.Coordinator.Reconcile(ctx, AdverseEpisodeObservation{
		TenantID: event.TenantID, LegalEntityID: program.Program.LegalEntityID, ProgramID: program.Program.ID,
		Check: check, Result: result,
	}); err != nil {
		return fmt.Errorf("reconcile monitoring adverse episode: %w", err)
	}
	return c.recordInbox(ctx, event)
}

func (c *AdverseEpisodeConsumer) recordInbox(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if _, err := c.Inbox.RecordInbox(ctx, event.TenantID, adverseEpisodeConsumer, event.ID, c.currentTime()); err != nil {
		return fmt.Errorf("record monitoring adverse episode inbox: %w", err)
	}
	return nil
}

func (c *AdverseEpisodeConsumer) currentTime() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func episodeSeverity(value RiskBand) int {
	switch value {
	case RiskCritical:
		return 4
	case RiskHigh:
		return 3
	case RiskModerate:
		return 2
	case RiskLow:
		return 1
	default:
		return 0
	}
}

func episodeWorsened(before AdverseEpisode, observation AdverseEpisodeObservation) bool {
	if episodeSeverity(observation.Result.Evaluation.Band) > episodeSeverity(before.LastBand) {
		return true
	}
	minimum := observation.Check.MinimumCoverage
	return minimum > 0 && before.LastCoverage >= minimum && observation.Result.Evaluation.Coverage < minimum
}

func episodeEventPayload(value AdverseEpisode) map[string]any {
	return map[string]any{
		"episode_id": value.ID,
		"version": value.RecordVersion,
		"legal_entity_id": value.LegalEntityID,
		"program_id": value.ProgramID,
		"monitoring_check_id": value.MonitoringCheckID,
		"state": value.State,
		"matter_id": value.MatterID,
		"last_result_id": value.LastResultID,
		"last_check_version": value.LastCheckVersion,
		"last_band": value.LastBand,
		"last_score": value.LastScore,
		"last_coverage": value.LastCoverage,
		"record_version": value.RecordVersion,
		"opened_at": value.OpenedAt,
		"closed_at": value.ClosedAt,
		"updated_at": value.UpdatedAt,
	}
}

