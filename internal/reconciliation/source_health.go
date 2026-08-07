package reconciliation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const sourceHealthConsumer = "source-health-reconciliation"

type Inbox interface {
	InboxProcessed(context.Context, string, string, string) (bool, error)
	RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
}

type SourceDependencies interface {
	ProgramIDsForEvidenceSource(context.Context, string, string) ([]string, error)
	EvidenceSourcesCurrentForProgram(context.Context, string, string) (bool, error)
}

type SignalSink interface {
	Ingest(context.Context, autonomy.Signal) (autonomy.Drift, bool, error)
	ResolveSourceHealth(context.Context, autonomy.Signal) (bool, error)
}

type ProgramTriggerSink interface {
	ApplyTrigger(context.Context, continuity.Trigger) (continuity.ProgramAggregate, *continuity.Matter, bool, error)
}

type SourceHealthConsumer struct {
	Inbox        Inbox
	Dependencies SourceDependencies
	Signals      SignalSink
	Programs     ProgramTriggerSink
}

type sourceHealthPayload struct {
	From       string `json:"from"`
	To         string `json:"to"`
	SourceCode string `json:"source_code"`
}

func (c *SourceHealthConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != "EVIDENCE_SOURCE" || event.EventType != "SourceHealthChanged" {
		return nil
	}
	if c == nil || c.Inbox == nil || c.Dependencies == nil || c.Signals == nil || c.Programs == nil {
		return fmt.Errorf("source health reconciliation is not configured")
	}
	processed, err := c.Inbox.InboxProcessed(ctx, event.TenantID, sourceHealthConsumer, event.ID)
	if err != nil {
		return fmt.Errorf("check source health inbox: %w", err)
	}
	if processed {
		return nil
	}

	change, err := decodeSourceHealth(event.Payload)
	if err != nil {
		return err
	}
	toHealthy := change.To == "CURRENT"
	fromHealthy := change.From == "CURRENT"
	toUnhealthy := unhealthySourceState(change.To)
	fromUnhealthy := unhealthySourceState(change.From)
	if !toHealthy && !toUnhealthy {
		return fmt.Errorf("unsupported source health target %q", change.To)
	}

	signalType := autonomy.SignalSourceDegraded
	if toHealthy {
		signalType = autonomy.SignalSourceRecovered
	}
	signal := autonomy.Signal{
		TenantID: event.TenantID, Type: signalType, SubjectType: "EVIDENCE_SOURCE", SubjectID: event.AggregateID,
		Source: sourceHealthConsumer, DedupeKey: "source-health:" + event.ID,
		ObservedAt: event.OccurredAt, EffectiveAt: event.OccurredAt,
		Payload: map[string]string{"from": change.From, "to": change.To, "source_code": change.SourceCode},
	}
	if toHealthy {
		_, err = c.Signals.ResolveSourceHealth(ctx, signal)
	} else {
		_, _, err = c.Signals.Ingest(ctx, signal)
	}
	if err != nil {
		return fmt.Errorf("apply source health signal: %w", err)
	}

	triggerType := ""
	switch {
	case toHealthy && (fromUnhealthy || change.From == "" || change.From == "UNKNOWN"):
		triggerType = "SOURCE_RECOVERED"
	case toUnhealthy && (fromHealthy || change.From == "" || change.From == "UNKNOWN"):
		triggerType = "SOURCE_DEGRADED"
	}
	if triggerType != "" {
		programIDs, err := c.Dependencies.ProgramIDsForEvidenceSource(ctx, event.TenantID, event.AggregateID)
		if err != nil {
			return fmt.Errorf("resolve source dependencies: %w", err)
		}
		payload, err := json.Marshal(map[string]string{
			"from": change.From, "to": change.To, "source_code": change.SourceCode, "source_event_id": event.ID,
		})
		if err != nil {
			return fmt.Errorf("encode source trigger: %w", err)
		}
		for _, programID := range programIDs {
			if triggerType == "SOURCE_RECOVERED" {
				current, err := c.Dependencies.EvidenceSourcesCurrentForProgram(ctx, event.TenantID, programID)
				if err != nil {
					return fmt.Errorf("check source recovery for program %s: %w", programID, err)
				}
				if !current {
					continue
				}
			}
			_, _, _, err := c.Programs.ApplyTrigger(ctx, continuity.Trigger{
				TenantID: event.TenantID, ProgramID: programID, Type: triggerType,
				SubjectType: "EVIDENCE_SOURCE", SubjectID: event.AggregateID,
				DedupeKey: "source-health:" + event.ID + ":" + programID,
				Payload:   payload, ObservedAt: event.OccurredAt, Source: sourceHealthConsumer,
			})
			if err != nil {
				return fmt.Errorf("apply source trigger to program %s: %w", programID, err)
			}
		}
	}

	if _, err := c.Inbox.RecordInbox(ctx, event.TenantID, sourceHealthConsumer, event.ID, time.Now().UTC()); err != nil {
		return fmt.Errorf("record source health inbox: %w", err)
	}
	return nil
}

func decodeSourceHealth(raw json.RawMessage) (sourceHealthPayload, error) {
	var value sourceHealthPayload
	if err := json.Unmarshal(raw, &value); err != nil {
		return sourceHealthPayload{}, fmt.Errorf("decode source health event: %w", err)
	}
	value.From = strings.ToUpper(strings.TrimSpace(value.From))
	value.To = strings.ToUpper(strings.TrimSpace(value.To))
	value.SourceCode = strings.TrimSpace(value.SourceCode)
	if value.To == "" {
		return sourceHealthPayload{}, fmt.Errorf("source health event is missing target state")
	}
	return value, nil
}

func unhealthySourceState(value string) bool {
	switch value {
	case "DEGRADED", "STALE", "UNAVAILABLE":
		return true
	default:
		return false
	}
}
