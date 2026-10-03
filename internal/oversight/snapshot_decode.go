package oversight

import (
	"encoding/json"
	"fmt"
)

func decodeStoredSnapshot(value *Snapshot, highWater, payload []byte) error {
	if value == nil {
		return ErrInvalid
	}
	if err := json.Unmarshal(highWater, &value.SourceHighWater); err != nil {
		return fmt.Errorf("decode oversight high-water marks: %w", err)
	}
	metadata := struct {
		Counts         Counts               `json:"counts"`
		Interventions  []Intervention       `json:"interventions"`
		Pressure       []CategoryPressure   `json:"pressure"`
		Aging          []AgingBucket        `json:"aging"`
		Performance    []Performance        `json:"performance"`
		Estimates      []ResolutionEstimate `json:"estimates"`
		HistoryQuality HistoryQuality       `json:"history_quality"`
	}{Counts: value.Counts}
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return fmt.Errorf("decode oversight snapshot: %w", err)
	}
	value.Counts = metadata.Counts
	value.Interventions = metadata.Interventions
	value.Pressure = metadata.Pressure
	value.Aging = metadata.Aging
	value.Performance = metadata.Performance
	value.Estimates = metadata.Estimates
	value.HistoryQuality = metadata.HistoryQuality
	return nil
}
