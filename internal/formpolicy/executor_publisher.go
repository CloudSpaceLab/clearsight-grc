package formpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type ScoredResponseHandler interface {
	Handle(context.Context, ScoredResponseEvent) ([]ExecutionReceipt, error)
}

type ScoredResponsePublisher struct {
	Handler ScoredResponseHandler
}

func (publisher ScoredResponsePublisher) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.EventType != "FORM_RESPONSE_SCORED" {
		return nil
	}
	if publisher.Handler == nil || strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.TenantID) == "" || event.OccurredAt.IsZero() || len(event.Payload) == 0 || len(event.Payload) > 16*1024 {
		return fmt.Errorf("scored response event is invalid")
	}
	var payload struct {
		Version             int64                       `json:"version"`
		ResponseRevisionID  string                      `json:"response_revision_id"`
		FormTemplateID      string                      `json:"form_template_id"`
		FormTemplateVersion int64                       `json:"form_template_version"`
		ScoreState          evidence.ResponseScoreState `json:"score_state"`
	}
	decoder := json.NewDecoder(bytes.NewReader(event.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Version < 1 || payload.FormTemplateVersion < 1 || strings.TrimSpace(payload.ResponseRevisionID) == "" || len(payload.ResponseRevisionID) > 512 || strings.TrimSpace(payload.FormTemplateID) == "" || len(payload.FormTemplateID) > 512 || payload.ScoreState != evidence.ResponseScoreFinal && payload.ScoreState != evidence.ResponseScoreProvisional && payload.ScoreState != evidence.ResponseScoreFailed && payload.ScoreState != evidence.ResponseScoreNotConfigured {
		return fmt.Errorf("scored response event payload is invalid")
	}
	_, err := publisher.Handler.Handle(ctx, ScoredResponseEvent{ID: event.ID, TenantID: event.TenantID, ResponseRevisionID: strings.TrimSpace(payload.ResponseRevisionID), OccurredAt: event.OccurredAt.UTC()})
	return err
}

var _ workflowruntime.Publisher = ScoredResponsePublisher{}
