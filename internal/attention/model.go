package attention

import (
	"encoding/json"
	"fmt"
	"strings"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const (
	SourceAggregateType = "DOMAIN_METRIC_SNAPSHOT"
	SourceEventType     = "DomainMetricSnapshotProjected"

	EpisodeAggregateType = "ATTENTION_EPISODE"
	EventEpisodeOpened   = "AttentionEpisodeOpened"
	EventEpisodeWorsened = "AttentionEpisodeWorsened"
	EventEpisodeCleared  = "AttentionEpisodeCleared"

	ConsumerName = "attention-episode-v1"
)

type SourceEvent struct {
	SourceID           string `json:"source_id"`
	LegalEntityID      string `json:"legal_entity_id"`
	DefinitionRevision string `json:"definition_revision"`
	SourceRevision     string `json:"source_revision"`
}

type Intent struct {
	EpisodeID       string `json:"episode_id"`
	LegalEntityID   string `json:"legal_entity_id"`
	PrincipalID     string `json:"principal_id"`
	Condition       string `json:"condition"`
	ConditionState  string `json:"condition_state"`
	SubjectType     string `json:"subject_type"`
	SubjectID       string `json:"subject_id"`
	SourceID        string `json:"source_id"`
	NoticeSequence int    `json:"notice_sequence"`
}

func DecodeSourceEvent(event workflowruntime.OutboxEvent) (SourceEvent, bool, error) {
	if event.AggregateType != SourceAggregateType || event.EventType != SourceEventType {
		return SourceEvent{}, false, nil
	}
	var value SourceEvent
	if err := json.Unmarshal(event.Payload, &value); err != nil {
		return SourceEvent{}, true, fmt.Errorf("decode attention source event: %w", err)
	}
	value.SourceID = strings.TrimSpace(value.SourceID)
	value.LegalEntityID = strings.TrimSpace(value.LegalEntityID)
	value.DefinitionRevision = strings.TrimSpace(value.DefinitionRevision)
	value.SourceRevision = strings.TrimSpace(value.SourceRevision)
	if value.SourceID == "" || value.LegalEntityID == "" || value.DefinitionRevision == "" || value.SourceRevision == "" ||
		value.SourceID != strings.TrimSpace(event.AggregateID) {
		return SourceEvent{}, true, fmt.Errorf("invalid attention source event")
	}
	return value, true, nil
}

func DecodeIntent(event workflowruntime.OutboxEvent) (Intent, bool, error) {
	if event.AggregateType != EpisodeAggregateType {
		return Intent{}, false, nil
	}
	switch event.EventType {
	case EventEpisodeOpened, EventEpisodeWorsened, EventEpisodeCleared:
	default:
		return Intent{}, false, nil
	}
	var value Intent
	if err := json.Unmarshal(event.Payload, &value); err != nil {
		return Intent{}, true, fmt.Errorf("decode attention intent: %w", err)
	}
	value.EpisodeID = strings.TrimSpace(value.EpisodeID)
	value.LegalEntityID = strings.TrimSpace(value.LegalEntityID)
	value.PrincipalID = strings.TrimSpace(value.PrincipalID)
	value.Condition = strings.TrimSpace(value.Condition)
	value.ConditionState = strings.TrimSpace(value.ConditionState)
	value.SubjectType = strings.TrimSpace(value.SubjectType)
	value.SubjectID = strings.TrimSpace(value.SubjectID)
	value.SourceID = strings.TrimSpace(value.SourceID)
	if value.EpisodeID == "" || value.LegalEntityID == "" || value.PrincipalID == "" ||
		value.Condition == "" || value.ConditionState == "" || value.SubjectID == "" || value.SourceID == "" ||
		value.NoticeSequence < 1 || value.EpisodeID != strings.TrimSpace(event.AggregateID) ||
		(value.SubjectType != "RISK" && value.SubjectType != "LOSS") {
		return Intent{}, true, fmt.Errorf("invalid attention intent")
	}
	return value, true, nil
}
